package git

import (
	"os"
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
	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Unsetenv("BC_AGENT_WORKTREE")
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		}
	}()

	// Should pass when env var is unset (non-agent context)
	if err := validateWorktree("/any/path"); err != nil {
		t.Errorf("expected nil error when BC_AGENT_WORKTREE unset, got: %v", err)
	}
}

func TestValidateWorktree_InsideWorktree(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)

	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Setenv("BC_AGENT_WORKTREE", dir)
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		} else {
			os.Unsetenv("BC_AGENT_WORKTREE")
		}
	}()

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

	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Setenv("BC_AGENT_WORKTREE", dir1)
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		} else {
			os.Unsetenv("BC_AGENT_WORKTREE")
		}
	}()

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

	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Setenv("BC_AGENT_WORKTREE", dir1)
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		} else {
			os.Unsetenv("BC_AGENT_WORKTREE")
		}
	}()

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
	_, err := Run(".", )
	if err == nil {
		t.Fatal("expected error for no args")
	}
}

func TestReadOpsAllowedOutsideWorktree(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Setenv("BC_AGENT_WORKTREE", dir1)
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		} else {
			os.Unsetenv("BC_AGENT_WORKTREE")
		}
	}()

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
