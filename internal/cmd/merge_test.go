package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/queue"
)

// initGitRepo creates a bare-minimum git repo in a temp directory with one commit.
// Returns the repo path. Configures local user.name/email for CI environments.
func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmds := [][]string{
		{"git", "init", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
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
		cmd := exec.Command(args[0], args[1:]...)
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
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/xyz")
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
	cmd := exec.Command("git", "-C", repo, "checkout", "--detach")
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
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/ff")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "commit", "--allow-empty", "-m", "ff commit")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "checkout", "main")
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
	os.WriteFile(filePath, []byte("main content\n"), 0644)
	exec.Command("git", "-C", repo, "add", "conflict.txt").Run()
	exec.Command("git", "-C", repo, "commit", "-m", "main: add conflict.txt").Run()

	// Create branch from parent of the above commit with different content
	exec.Command("git", "-C", repo, "checkout", "-b", "feature/conflict", "HEAD~1").Run()
	os.WriteFile(filePath, []byte("branch content\n"), 0644)
	exec.Command("git", "-C", repo, "add", "conflict.txt").Run()
	exec.Command("git", "-C", repo, "commit", "-m", "branch: add conflict.txt").Run()
	exec.Command("git", "-C", repo, "checkout", "main").Run()

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
	os.MkdirAll(agentsDir, 0755)

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
	os.MkdirAll(agentsDir, 0755)

	// Create a worktree for the agent
	worktreePath := filepath.Join(repo, ".bc", "worktrees", "eng-1")
	cmd := exec.Command("git", "-C", repo, "worktree", "add", "-b", "eng-1/work", worktreePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("worktree add failed: %v (%s)", err, out)
	}
	t.Cleanup(func() {
		exec.Command("git", "-C", repo, "worktree", "remove", "--force", worktreePath).Run()
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
	data, _ := json.MarshalIndent(agents, "", "  ")
	os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0644)

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
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)

	err := runValidation(dir)
	if err != nil {
		t.Errorf("runValidation failed on valid Go code: %v", err)
	}
}

func TestRunValidation_BuildFailure(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n\ngo 1.25\n"), 0644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() { undefined() }\n"), 0644)

	err := runValidation(dir)
	if err == nil {
		t.Error("expected error for invalid Go code")
	}
	if !strings.Contains(err.Error(), "go build failed") {
		t.Errorf("expected 'go build failed', got: %v", err)
	}
}

// --- markQueueDone tests ---

func TestMarkQueueDone_Success(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".bc")
	os.MkdirAll(stateDir, 0755)

	// Create a queue with an item
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	q.Add("Test item", "", "")
	items := q.ListAll()
	if len(items) == 0 {
		t.Fatal("expected at least one item")
	}
	workID := items[0].ID
	q.Save()

	err := markQueueDone(stateDir, dir, workID)
	if err != nil {
		t.Fatalf("markQueueDone failed: %v", err)
	}

	// Verify item is now done
	q2 := queue.New(filepath.Join(stateDir, "queue.json"))
	q2.Load()
	item := q2.Get(workID)
	if item == nil {
		t.Fatal("item not found after markQueueDone")
	}
	if item.Status != queue.StatusDone {
		t.Errorf("status = %s, want done", item.Status)
	}
}

func TestMarkQueueDone_NotFound(t *testing.T) {
	dir := t.TempDir()
	stateDir := filepath.Join(dir, ".bc")
	os.MkdirAll(stateDir, 0755)

	// Create empty queue
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	q.Save()

	err := markQueueDone(stateDir, dir, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent work item")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- mergeBranch tests ---

func TestMergeBranch_FastForward(t *testing.T) {
	repo := initGitRepo(t)

	// Create a branch ahead of main
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/ff-merge")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	filePath := filepath.Join(repo, "new.txt")
	os.WriteFile(filePath, []byte("new content\n"), 0644)
	exec.Command("git", "-C", repo, "add", "new.txt").Run()
	cmd = exec.Command("git", "-C", repo, "commit", "-m", "add new.txt")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit failed: %v (%s)", err, out)
	}
	cmd = exec.Command("git", "-C", repo, "checkout", "main")
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
	mainHead, _ := gitRevParse(repo, "main")
	branchHead, _ := gitRevParse(repo, "feature/ff-merge")
	if mainHead != branchHead {
		t.Errorf("main (%s) should equal branch (%s) after fast-forward", mainHead, branchHead)
	}
}

func TestMergeBranch_NonFastForward(t *testing.T) {
	repo := initGitRepo(t)

	// Create diverged branches: main gets one commit, feature gets another
	// First create the feature branch from current main
	cmd := exec.Command("git", "-C", repo, "checkout", "-b", "feature/merge-nff")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("checkout failed: %v (%s)", err, out)
	}
	os.WriteFile(filepath.Join(repo, "feature.txt"), []byte("feature\n"), 0644)
	exec.Command("git", "-C", repo, "add", "feature.txt").Run()
	exec.Command("git", "-C", repo, "commit", "-m", "feature commit").Run()

	// Go back to main and make a different commit
	exec.Command("git", "-C", repo, "checkout", "main").Run()
	os.WriteFile(filepath.Join(repo, "main.txt"), []byte("main\n"), 0644)
	exec.Command("git", "-C", repo, "add", "main.txt").Run()
	exec.Command("git", "-C", repo, "commit", "-m", "main commit").Run()

	hash, err := mergeBranch(repo, "feature/merge-nff")
	if err != nil {
		t.Fatalf("mergeBranch failed: %v", err)
	}
	if len(hash) != 12 {
		t.Errorf("expected 12-char short hash, got %q (len=%d)", hash, len(hash))
	}
}
