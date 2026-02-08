package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

// initGitRepo creates a bare-minimum git repo in a temp directory with one commit.
// Returns the repo path. Configures local user.name/email for CI environments.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", "-b", "main", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec,noctx // G204: test helper with variable args
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v (%s)", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

// createBranch creates a branch with an empty commit in the repo.
func createBranch(t *testing.T, repoDir, branch string) {
	t.Helper()
	cmds := [][]string{
		{"git", "-C", repoDir, "checkout", "-b", branch},
		{"git", "-C", repoDir, "commit", "--allow-empty", "-m", "work on " + branch},
		{"git", "-C", repoDir, "checkout", "main"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec,noctx // G204: test helper with variable args
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v (%s)", strings.Join(args, " "), err, out)
		}
	}
}

// --- gitBranchExists tests ---

func TestGitBranchExists_ValidBranch(t *testing.T) {
	repo := initGitRepo(t)
	// "main" exists after git init with a commit
	if err := gitBranchExists(repo, "main"); err != nil {
		t.Errorf("gitBranchExists(main) failed: %v", err)
	}
}

func TestGitBranchExists_InvalidBranch(t *testing.T) {
	repo := initGitRepo(t)
	err := gitBranchExists(repo, "nonexistent-branch")
	if err == nil {
		t.Error("expected error for nonexistent branch")
	}
}

func TestGitBranchExists_FeatureBranch(t *testing.T) {
	repo := initGitRepo(t)
	createBranch(t, repo, "feature/test-branch")
	if err := gitBranchExists(repo, "feature/test-branch"); err != nil {
		t.Errorf("gitBranchExists(feature/test-branch) failed: %v", err)
	}
}

// --- gitCurrentBranch tests ---

func TestGitCurrentBranch_Main(t *testing.T) {
	repo := initGitRepo(t)
	branch, err := gitCurrentBranch(repo)
	if err != nil {
		t.Fatalf("gitCurrentBranch failed: %v", err)
	}
	if branch != "main" {
		t.Errorf("expected main, got %q", branch)
	}
}

func TestGitCurrentBranch_FeatureBranch(t *testing.T) {
	repo := initGitRepo(t)
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/xyz") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	branch, err := gitCurrentBranch(repo)
	if err != nil {
		t.Fatalf("gitCurrentBranch failed: %v", err)
	}
	if branch != "feature/xyz" {
		t.Errorf("expected feature/xyz, got %q", branch)
	}
}

func TestGitCurrentBranch_DetachedHEAD(t *testing.T) {
	repo := initGitRepo(t)
	// Detach HEAD
	cmd := exec.Command("git", "-C", repo, "checkout", "--detach") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("detach failed: %v (%s)", err, out)
	}
	_, err := gitCurrentBranch(repo)
	if err == nil {
		t.Error("expected error for detached HEAD")
	}
	if !strings.Contains(err.Error(), "detached HEAD") {
		t.Errorf("expected 'detached HEAD' error, got: %v", err)
	}
}

func TestGitCurrentBranch_InvalidDir(t *testing.T) {
	_, err := gitCurrentBranch("/nonexistent/path")
	if err == nil {
		t.Error("expected error for invalid directory")
	}
}

// --- gitRevParse tests ---

func TestGitRevParse_Main(t *testing.T) {
	repo := initGitRepo(t)
	sha, err := gitRevParse(repo, "main")
	if err != nil {
		t.Fatalf("gitRevParse(main) failed: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA, got %q (len=%d)", sha, len(sha))
	}
}

func TestGitRevParse_InvalidRef(t *testing.T) {
	repo := initGitRepo(t)
	_, err := gitRevParse(repo, "nonexistent-ref")
	if err == nil {
		t.Error("expected error for nonexistent ref")
	}
}

// --- checkMergeConflicts tests ---

func TestCheckMergeConflicts_NoConflicts(t *testing.T) {
	repo := initGitRepo(t)
	createBranch(t, repo, "feature/clean")

	conflicts, err := checkMergeConflicts(repo, "feature/clean")
	if err != nil {
		t.Fatalf("checkMergeConflicts failed: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %v", conflicts)
	}
}

func TestCheckMergeConflicts_FastForward(t *testing.T) {
	repo := initGitRepo(t)
	// Create a branch that is ahead of main — main hasn't moved, so
	// merge-base == main HEAD → fast-forward path
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/ff") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "commit", "--allow-empty", "-m", "ff commit") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "checkout", "main") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout main failed: %v (%s)", err, out)
	}

	conflicts, err := checkMergeConflicts(repo, "feature/ff")
	if err != nil {
		t.Fatalf("checkMergeConflicts failed: %v", err)
	}
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts for fast-forward, got %v", conflicts)
	}
}

func TestCheckMergeConflicts_WithConflicts(t *testing.T) {
	repo := initGitRepo(t)

	// Create a file on main
	filePath := filepath.Join(repo, "conflict.txt")
	os.WriteFile(filePath, []byte("main content\n"), 0o600)                         //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "conflict.txt").Run()                    //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "commit", "-m", "main: add conflict.txt").Run() //nolint:errcheck,gosec,noctx

	// Create branch from parent of the above commit with different content
	exec.Command("git", "-C", repo, "checkout", "-b", "feature/conflict", "HEAD~1").Run() //nolint:errcheck,gosec,noctx
	os.WriteFile(filePath, []byte("branch content\n"), 0o600)                             //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "conflict.txt").Run()                          //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "commit", "-m", "branch: add conflict.txt").Run()     //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "checkout", "main").Run()                             //nolint:errcheck,gosec,noctx

	conflicts, err := checkMergeConflicts(repo, "feature/conflict")
	if err != nil {
		t.Fatalf("checkMergeConflicts failed: %v", err)
	}
	if len(conflicts) == 0 {
		t.Error("expected conflicts, got none")
	}
}

// --- resolveMergeTarget tests ---

func TestResolveMergeTarget_LiteralBranch(t *testing.T) {
	repo := initGitRepo(t)
	agentsDir := filepath.Join(repo, ".bc", "agents")
	os.MkdirAll(agentsDir, 0o750) //nolint:errcheck

	branch, worktreeDir, err := resolveMergeTarget(agentsDir, repo, "feature/my-branch")
	if err != nil {
		t.Fatalf("resolveMergeTarget failed: %v", err)
	}
	if branch != "feature/my-branch" {
		t.Errorf("expected literal branch name, got %q", branch)
	}
	if worktreeDir != "" {
		t.Errorf("expected empty worktreeDir for literal branch, got %q", worktreeDir)
	}
}

func TestResolveMergeTarget_AgentName(t *testing.T) {
	repo := initGitRepo(t)
	agentsDir := filepath.Join(repo, ".bc", "agents")
	os.MkdirAll(agentsDir, 0o750) //nolint:errcheck

	// Create a worktree for the agent
	worktreePath := filepath.Join(repo, ".bc", "worktrees", "eng-1")
	cmd := exec.Command("git", "-C", repo, "worktree", "add", "-b", "eng-1/work", worktreePath) //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("worktree add failed: %v (%s)", err, out)
	}
	t.Cleanup(func() {
		exec.Command("git", "-C", repo, "worktree", "remove", "--force", worktreePath).Run() //nolint:errcheck,gosec,noctx
	})

	// Seed agent state
	agents := map[string]*agent.Agent{
		"eng-1": {
			Name:        "eng-1",
			Role:        agent.RoleEngineer,
			State:       agent.StateIdle,
			WorktreeDir: worktreePath,
			Children:    []string{},
		},
	}
	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0o600) //nolint:errcheck

	branch, worktreeDir, err := resolveMergeTarget(agentsDir, repo, "eng-1")
	if err != nil {
		t.Fatalf("resolveMergeTarget failed: %v", err)
	}
	if branch != "eng-1/work" {
		t.Errorf("expected eng-1/work, got %q", branch)
	}
	if worktreeDir != worktreePath {
		t.Errorf("worktreeDir = %q, want %q", worktreeDir, worktreePath)
	}
}

// --- runValidation tests ---

func TestRunValidation_Success(t *testing.T) {
	// runValidation runs go build/test/vet in a directory.
	// Use a minimal Go module that will pass all checks.
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0o600)       //nolint:errcheck
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o600) //nolint:errcheck

	err := runValidation(dir)
	if err != nil {
		t.Errorf("runValidation failed on valid Go code: %v", err)
	}
}

func TestRunValidation_BuildFailure(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0o600)                    //nolint:errcheck
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { undefined() }\n"), 0o600) //nolint:errcheck

	err := runValidation(dir)
	if err == nil {
		t.Error("expected error for invalid Go code")
	}
	if !strings.Contains(err.Error(), "go build failed") {
		t.Errorf("expected 'go build failed', got: %v", err)
	}
}

// --- mergeBranch tests ---

func TestMergeBranch_FastForward(t *testing.T) {
	repo := initGitRepo(t)

	// Create a branch ahead of main
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/ff-merge") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	filePath := filepath.Join(repo, "new.txt")
	os.WriteFile(filePath, []byte("new content\n"), 0o600)               //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "new.txt").Run()              //nolint:errcheck,gosec,noctx
	cmd = exec.Command("git", "-C", repo, "commit", "-m", "add new.txt") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "checkout", "main") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout main failed: %v (%s)", err, out)
	}

	hash, err := mergeBranch(repo, "feature/ff-merge")
	if err != nil {
		t.Fatalf("mergeBranch failed: %v", err)
	}
	if len(hash) != 12 {
		t.Errorf("expected 12-char short hash, got %q (len=%d)", hash, len(hash))
	}

	// Verify main now has the file
	mainHead, _ := gitRevParse(repo, "main")               //nolint:errcheck
	branchHead, _ := gitRevParse(repo, "feature/ff-merge") //nolint:errcheck
	if mainHead != branchHead {
		t.Errorf("main (%s) should equal branch (%s) after fast-forward", mainHead, branchHead)
	}
}

func TestMergeBranch_NonFastForward(t *testing.T) {
	repo := initGitRepo(t)

	// Create diverged branches: main gets one commit, feature gets another
	// First create the feature branch from current main
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/merge-nff") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0o600) //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "feature.txt").Run()                  //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "commit", "-m", "feature commit").Run()      //nolint:errcheck,gosec,noctx

	// Go back to main and make a different commit
	exec.Command("git", "-C", repo, "checkout", "main").Run()              //nolint:errcheck,gosec,noctx
	os.WriteFile(filepath.Join(repo, "main.txt"), []byte("main\n"), 0o600) //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "main.txt").Run()               //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "commit", "-m", "main commit").Run()   //nolint:errcheck,gosec,noctx

	hash, err := mergeBranch(repo, "feature/merge-nff")
	if err != nil {
		t.Fatalf("mergeBranch failed: %v", err)
	}
	if len(hash) != 12 {
		t.Errorf("expected 12-char short hash, got %q (len=%d)", hash, len(hash))
	}
}

// --- rollbackMerge tests ---

func TestRollbackMerge_RestoresMainRef(t *testing.T) {
	repo := initGitRepo(t)

	// Get the original main HEAD
	originalHead, err := gitRevParse(repo, "main")
	if err != nil {
		t.Fatalf("gitRevParse failed: %v", err)
	}

	// Create and merge a branch to move main forward
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/to-rollback") //nolint:gosec,noctx // G204
	if out, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
		t.Fatalf("checkout failed: %v (%s)", cmdErr, out)
	}
	os.WriteFile(filepath.Join(repo, "rollback.txt"), []byte("will be rolled back\n"), 0o600) //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "rollback.txt").Run()                              //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "commit", "-m", "commit to rollback").Run()               //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "checkout", "main").Run()                                 //nolint:errcheck,gosec,noctx

	// Merge the branch (moves main forward)
	_, err = mergeBranch(repo, "feature/to-rollback")
	if err != nil {
		t.Fatalf("mergeBranch failed: %v", err)
	}

	// Verify main has moved
	newHead, _ := gitRevParse(repo, "main") //nolint:errcheck
	if newHead == originalHead {
		t.Fatal("main should have moved after merge")
	}

	// Now rollback
	err = rollbackMerge(repo, originalHead)
	if err != nil {
		t.Fatalf("rollbackMerge failed: %v", err)
	}

	// Verify main is back to original
	restoredHead, _ := gitRevParse(repo, "main") //nolint:errcheck
	if restoredHead != originalHead {
		t.Errorf("main should be restored to %s, got %s", originalHead, restoredHead)
	}
}

func TestRollbackMerge_InvalidRestorePoint(t *testing.T) {
	repo := initGitRepo(t)

	err := rollbackMerge(repo, "invalid-sha-that-does-not-exist")
	if err == nil {
		t.Error("expected error for invalid restore point")
	}
}

// --- Flag initialization tests ---

func TestMergeFlags_StatusExists(t *testing.T) {
	flag := mergeCmd.Flags().Lookup("status")
	if flag == nil {
		t.Fatal("expected --status flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --status default to be false, got %s", flag.DefValue)
	}
}

func TestMergeFlags_JSONAccessible(t *testing.T) {
	// The --json flag is a persistent flag on rootCmd, inherited by all subcommands
	// Verify it's accessible from mergeCmd
	_, err := mergeCmd.Flags().GetBool("json")
	if err != nil {
		t.Errorf("--json flag should be accessible from mergeCmd: %v", err)
	}
}

func TestMergeFlags_DryRunExists(t *testing.T) {
	flag := mergeCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("expected --dry-run flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --dry-run default to be false, got %s", flag.DefValue)
	}
}

func TestMergeFlags_YesExists(t *testing.T) {
	flag := mergeCmd.Flags().Lookup("yes")
	if flag == nil {
		t.Fatal("expected --yes flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --yes default to be false, got %s", flag.DefValue)
	}
}

func TestMergeFlags_SkipTestsExists(t *testing.T) {
	flag := mergeCmd.Flags().Lookup("skip-tests")
	if flag == nil {
		t.Fatal("expected --skip-tests flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --skip-tests default to be false, got %s", flag.DefValue)
	}
}

func TestMergeFlags_RebaseExists(t *testing.T) {
	flag := mergeCmd.Flags().Lookup("rebase")
	if flag == nil {
		t.Fatal("expected --rebase flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --rebase default to be false, got %s", flag.DefValue)
	}
}

func TestMergeFlags_NoRebaseExists(t *testing.T) {
	flag := mergeCmd.Flags().Lookup("no-rebase")
	if flag == nil {
		t.Fatal("expected --no-rebase flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --no-rebase default to be false, got %s", flag.DefValue)
	}
}

// --- isBranchStale tests ---

func TestIsBranchStale_UpToDate(t *testing.T) {
	repo := initGitRepo(t)
	// Create a branch from current main - should not be stale
	createBranch(t, repo, "feature/up-to-date")

	stale, count, err := isBranchStale(repo, "feature/up-to-date")
	if err != nil {
		t.Fatalf("isBranchStale failed: %v", err)
	}
	if stale {
		t.Errorf("expected branch to not be stale, but got stale with count=%d", count)
	}
}

func TestIsBranchStale_BehindMain(t *testing.T) {
	repo := initGitRepo(t)

	// Create a branch
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/stale") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "checkout", "main") //nolint:gosec,noctx // G204
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout main failed: %v (%s)", err, out)
	}

	// Add commits to main after branching
	os.WriteFile(filepath.Join(repo, "main-update.txt"), []byte("main update\n"), 0o600) //nolint:errcheck
	exec.Command("git", "-C", repo, "add", "main-update.txt").Run()                      //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", repo, "commit", "-m", "main update").Run()                 //nolint:errcheck,gosec,noctx

	stale, count, err := isBranchStale(repo, "feature/stale")
	if err != nil {
		t.Fatalf("isBranchStale failed: %v", err)
	}
	if !stale {
		t.Error("expected branch to be stale")
	}
	if count != 1 {
		t.Errorf("expected 1 commit behind, got %d", count)
	}
}

// --- extractAgentFromBranch tests ---

func TestExtractAgentFromBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "engineer branch",
			branch:   "engineer-01/issue-123/feature",
			expected: "engineer-01",
		},
		{
			name:     "tech-lead branch",
			branch:   "tech-lead-02/issue-456/fix",
			expected: "tech-lead-02",
		},
		{
			name:     "qa branch",
			branch:   "qa-01/issue-789/test",
			expected: "qa-01",
		},
		{
			name:     "coordinator branch",
			branch:   "coordinator/issue-100/update",
			expected: "coordinator",
		},
		{
			name:     "manager branch",
			branch:   "manager/issue-200/config",
			expected: "manager",
		},
		{
			name:     "feature branch without agent",
			branch:   "feature/add-login",
			expected: "",
		},
		{
			name:     "fix branch without agent",
			branch:   "fix/bug-123",
			expected: "",
		},
		{
			name:     "simple branch name",
			branch:   "main",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAgentFromBranch(tt.branch)
			if result != tt.expected {
				t.Errorf("extractAgentFromBranch(%q) = %q, want %q", tt.branch, result, tt.expected)
			}
		})
	}
}

// --- truncateSHA tests ---

func TestTruncateSHA(t *testing.T) {
	tests := []struct {
		name     string
		sha      string
		expected string
	}{
		{
			name:     "full SHA",
			sha:      "abc123def456789012345678901234567890",
			expected: "abc123def456",
		},
		{
			name:     "short SHA",
			sha:      "abc123",
			expected: "abc123",
		},
		{
			name:     "exactly 12 chars",
			sha:      "abc123def456",
			expected: "abc123def456",
		},
		{
			name:     "empty",
			sha:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateSHA(tt.sha)
			if result != tt.expected {
				t.Errorf("truncateSHA(%q) = %q, want %q", tt.sha, result, tt.expected)
			}
		})
	}
}

// --- notifyConflicts tests ---

func TestNotifyConflicts_CreatesChannelMessage(t *testing.T) {
	// Create a temp directory with workspace structure
	tmpDir := t.TempDir()
	bcDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		t.Fatalf("failed to create .bc dir: %v", err)
	}

	// Initialize a git repo for gitRevParse to work
	cmds := [][]string{
		{"git", "init", "-b", "main", tmpDir},
		{"git", "-C", tmpDir, "config", "user.email", "test@test.com"},
		{"git", "-C", tmpDir, "config", "user.name", "Test"},
		{"git", "-C", tmpDir, "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...) //nolint:gosec,noctx // G204: test helper
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s failed: %v (%s)", strings.Join(args, " "), err, out)
		}
	}

	// Create a feature branch
	exec.Command("git", "-C", tmpDir, "checkout", "-b", "engineer-04/issue-257/conflict-notify").Run() //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", tmpDir, "commit", "--allow-empty", "-m", "feature").Run()                //nolint:errcheck,gosec,noctx
	exec.Command("git", "-C", tmpDir, "checkout", "main").Run()                                        //nolint:errcheck,gosec,noctx

	// Call notifyConflicts
	conflicts := []string{"pkg/merge/merge.go", "internal/cmd/merge.go"}
	err := notifyConflicts(tmpDir, "engineer-04/issue-257/conflict-notify", conflicts)
	if err != nil {
		t.Fatalf("notifyConflicts failed: %v", err)
	}

	// Verify channel file was created with notification
	channelFile := filepath.Join(bcDir, "channels.json")
	data, err := os.ReadFile(channelFile) //nolint:gosec // G304: test file with controlled path
	if err != nil {
		t.Fatalf("failed to read channels file: %v", err)
	}

	// Check that the notification message is in the file
	if !strings.Contains(string(data), "Merge Conflict Detected") {
		t.Error("expected 'Merge Conflict Detected' in channel history")
	}
	if !strings.Contains(string(data), "pkg/merge/merge.go") {
		t.Error("expected conflicting file in channel history")
	}
	if !strings.Contains(string(data), "Resolution steps") {
		t.Error("expected resolution steps in channel history")
	}
}
