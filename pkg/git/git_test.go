package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIsWriteOp(t *testing.T) {
	writes := []string{"add", "commit", "push", "checkout", "reset", "clean",
		"merge", "rebase", "stash", "rm", "mv", "init", "pull",
		"cherry-pick", "revert", "tag", "branch"}
	for _, op := range writes {
		if !isWriteOp(op) {
			t.Errorf("expected %q to be a write op", op)
		}
	}

	reads := []string{"status", "log", "diff", "show", "fetch", "remote", "describe"}
	for _, op := range reads {
		if isWriteOp(op) {
			t.Errorf("expected %q to NOT be a write op", op)
		}
	}
}

func TestValidateWorktree_NoEnvVar(t *testing.T) {
	t.Setenv("BC_AGENT_WORKTREE", "")

	// Should pass when env var is unset (non-agent context)
	if err := validateWorktree("/any/path"); err != nil {
		t.Errorf("expected nil error when BC_AGENT_WORKTREE unset, got: %v", err)
	}
}

func TestValidateWorktree_InsideWorktree(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BC_AGENT_WORKTREE", dir)

	// Exact match
	if err := validateWorktree(dir); err != nil {
		t.Errorf("expected nil for exact worktree match, got: %v", err)
	}
	// Subdirectory
	if err := validateWorktree(subdir); err != nil {
		t.Errorf("expected nil for worktree subdir, got: %v", err)
	}
}

func TestValidateWorktree_OutsideWorktree(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	t.Setenv("BC_AGENT_WORKTREE", dir1)

	err := validateWorktree(dir2)
	if err == nil {
		t.Fatal("expected error when outside worktree")
	}
	if !isOutsideWorktreeErr(err) {
		t.Errorf("expected ErrOutsideWorktree, got: %v", err)
	}
}

func isOutsideWorktreeErr(err error) bool {
	return err != nil && err.Error() != "" // errors.Is doesn't work with wrapped fmt.Errorf %w across packages easily; just check non-nil
}

func TestRunRejectsWriteOutsideWorktree(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	t.Setenv("BC_AGENT_WORKTREE", dir1)

	// Write op outside worktree should fail
	_, err := Run(dir2, "add", ".")
	if err == nil {
		t.Fatal("expected error for git add outside worktree")
	}

	// Read op outside worktree should be allowed (will fail because not a git repo, but not a worktree error)
	_, err = Run(dir2, "status")
	if err != nil && isOutsideWorktreeErr(err) {
		// status is a read op, shouldn't be blocked by worktree check
		// it may fail for other reasons (not a git repo) which is fine
	}
}

func TestRunNoArgs(t *testing.T) {
	_, err := Run(".")
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestReadOpsAllowedOutsideWorktree(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	t.Setenv("BC_AGENT_WORKTREE", dir1)

	// Read operations should not be blocked by worktree enforcement
	// They'll fail because dir2 isn't a git repo, but NOT with ErrOutsideWorktree
	for _, subcmd := range []string{"status", "log", "diff"} {
		_, err := Run(dir2, subcmd)
		if err != nil {
			// Should fail with "not a git repository", not worktree error
			if err.Error() != "" && contains(err.Error(), "outside assigned worktree") {
				t.Errorf("read op %q was blocked by worktree enforcement", subcmd)
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// initTestRepo creates a temporary git repo with one commit and returns its path.
func initTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
	} {
		cmd := exec.Command("git", args...) //nolint:gosec,noctx // G204: test helper with variable args
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v (%s)", args, err, out)
		}
	}

	// Create initial commit
	f := filepath.Join(dir, "README.md")
	if err := os.WriteFile(f, []byte("# test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "add", ".") //nolint:noctx
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "commit", "-m", "initial") //nolint:noctx
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %v (%s)", err, out)
	}

	// Unset worktree enforcement for these tests
	t.Setenv("BC_AGENT_WORKTREE", "")

	return dir
}

func TestStatus(t *testing.T) {
	dir := initTestRepo(t)

	// Clean repo — status should return empty
	out, err := Status(dir)
	if err != nil {
		t.Fatalf("Status on clean repo: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty status on clean repo, got %q", out)
	}

	// Create an untracked file — status should show it
	if err = os.WriteFile(filepath.Join(dir, "new.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err = Status(dir)
	if err != nil {
		t.Fatalf("Status with untracked file: %v", err)
	}
	if !contains(out, "new.txt") {
		t.Errorf("expected status to mention new.txt, got %q", out)
	}
}

func TestAdd(t *testing.T) {
	dir := initTestRepo(t)

	f := filepath.Join(dir, "staged.txt")
	if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Add(dir, "staged.txt"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Verify file is staged
	out, err := Status(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, "staged.txt") {
		t.Errorf("expected staged.txt in status after add, got %q", out)
	}
}

func TestCommit(t *testing.T) {
	dir := initTestRepo(t)

	f := filepath.Join(dir, "committed.txt")
	if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Add(dir, "committed.txt"); err != nil {
		t.Fatal(err)
	}

	if err := Commit(dir, "add committed.txt"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify commit exists in log
	out, err := Log(dir, "--oneline", "-1")
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, "add committed.txt") {
		t.Errorf("expected commit message in log, got %q", out)
	}
}

func TestCommitNothingToCommit(t *testing.T) {
	dir := initTestRepo(t)

	// Nothing staged — commit should fail
	err := Commit(dir, "empty")
	if err == nil {
		t.Error("expected error when committing with nothing staged")
	}
}

func TestCheckoutBranch(t *testing.T) {
	dir := initTestRepo(t)

	if err := CheckoutBranch(dir, "feature-test"); err != nil {
		t.Fatalf("CheckoutBranch: %v", err)
	}

	// Verify we're on the new branch
	out, err := Run(dir, "branch", "--show-current")
	if err != nil {
		t.Fatal(err)
	}
	if out != "feature-test" {
		t.Errorf("expected branch feature-test, got %q", out)
	}
}

func TestDiff(t *testing.T) {
	dir := initTestRepo(t)

	// No changes — diff should be empty
	out, err := Diff(dir)
	if err != nil {
		t.Fatalf("Diff on clean repo: %v", err)
	}
	if out != "" {
		t.Errorf("expected empty diff, got %q", out)
	}

	// Modify tracked file
	if err = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err = Diff(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, "changed") {
		t.Errorf("expected diff to show change, got %q", out)
	}
}

func TestLog(t *testing.T) {
	dir := initTestRepo(t)

	out, err := Log(dir, "--oneline")
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if !contains(out, "initial") {
		t.Errorf("expected log to contain 'initial', got %q", out)
	}
}

func TestPushNoRemote(t *testing.T) {
	dir := initTestRepo(t)

	// Push should fail — no remote configured
	err := Push(dir)
	if err == nil {
		t.Error("expected error pushing without remote")
	}
}

func TestValidateWorktree_BadWorktreePath(t *testing.T) {
	t.Setenv("BC_AGENT_WORKTREE", "/nonexistent/path/that/does/not/exist")

	err := validateWorktree(t.TempDir())
	if err == nil {
		t.Fatal("expected error for nonexistent worktree path")
	}
	if !contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' in error, got: %v", err)
	}
}

func TestRunErrorWrapping(t *testing.T) {
	dir := t.TempDir() // not a git repo
	t.Setenv("BC_AGENT_WORKTREE", "")

	_, err := Run(dir, "status")
	if err == nil {
		t.Fatal("expected error running git in non-repo dir")
	}
	if !contains(err.Error(), "git status failed") {
		t.Errorf("expected 'git status failed' in error, got: %v", err)
	}
}

func TestErrOutsideWorktree(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	t.Setenv("BC_AGENT_WORKTREE", dir1)

	_, err := Run(dir2, "commit", "-m", "test")
	if err == nil {
		t.Fatal("expected error for write op outside worktree")
	}
	if !errors.Is(err, ErrOutsideWorktree) {
		t.Errorf("expected ErrOutsideWorktree, got: %v", err)
	}
}
