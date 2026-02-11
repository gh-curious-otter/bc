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

func TestIsWithinDir(t *testing.T) {
	tests := []struct {
		name   string
		child  string
		parent string
		want   bool
	}{
		{"exact match", "/a/b/c", "/a/b/c", true},
		{"subdirectory", "/a/b/c/d", "/a/b/c", true},
		{"deep subdirectory", "/a/b/c/d/e/f", "/a/b/c", true},
		{"parent directory", "/a/b", "/a/b/c", false},
		{"sibling directory", "/a/b/d", "/a/b/c", false},
		{"unrelated path", "/x/y/z", "/a/b/c", false},
		{"root vs subdir", "/a", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWithinDir(tt.child, tt.parent)
			if got != tt.want {
				t.Errorf("isWithinDir(%q, %q) = %v, want %v", tt.child, tt.parent, got, tt.want)
			}
		})
	}
}

// mockCwd sets getCwd to return the given directory and restores it on cleanup.
func mockCwd(t *testing.T, dir string) {
	t.Helper()
	orig := getCwd
	getCwd = func() (string, error) { return dir, nil }
	t.Cleanup(func() { getCwd = orig })
}

func TestWorktreeCheckNoEnvVar(t *testing.T) {
	// Unset BC_AGENT_WORKTREE to test graceful error
	t.Setenv("BC_AGENT_WORKTREE", "")
	os.Unsetenv("BC_AGENT_WORKTREE") //nolint:errcheck

	err := runWorktreeCheck(nil, nil)
	if err == nil {
		t.Error("expected error when BC_AGENT_WORKTREE is not set")
	}
}

func TestWorktreeCheckMatch(t *testing.T) {
	dir := t.TempDir()

	t.Setenv("BC_AGENT_WORKTREE", dir)
	t.Setenv("BC_AGENT_ID", "test-agent")

	// Mock getCwd to return the worktree directory
	mockCwd(t, dir)

	err := runWorktreeCheck(nil, nil)
	if err != nil {
		t.Errorf("expected no error when in correct worktree, got: %v", err)
	}
}

func TestWorktreeCheckMismatch(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	t.Setenv("BC_AGENT_WORKTREE", dir1)
	t.Setenv("BC_AGENT_ID", "test-agent")

	// Mock getCwd to return a different directory
	mockCwd(t, dir2)

	err := runWorktreeCheck(nil, nil)
	if err == nil {
		t.Error("expected error when in wrong directory")
	}
}

func TestWorktreeCheckSubdirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subdir, 0o750); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BC_AGENT_WORKTREE", dir)
	t.Setenv("BC_AGENT_ID", "test-agent")

	// Mock getCwd to return a subdirectory of the worktree
	mockCwd(t, subdir)

	err := runWorktreeCheck(nil, nil)
	if err != nil {
		t.Errorf("expected no error when in worktree subdirectory, got: %v", err)
	}
}

func TestWorktreeCheckMissingDir(t *testing.T) {
	t.Setenv("BC_AGENT_WORKTREE", "/nonexistent/worktree/path")
	t.Setenv("BC_AGENT_ID", "test-agent")

	err := runWorktreeCheck(nil, nil)
	if err == nil {
		t.Error("expected error when worktree directory does not exist")
	}
}

func TestCheckWorktreeWarningNoEnv(t *testing.T) {
	// When BC_AGENT_WORKTREE is not set, should silently return
	t.Setenv("BC_AGENT_WORKTREE", "")
	os.Unsetenv("BC_AGENT_WORKTREE") //nolint:errcheck

	// Should not panic
	checkWorktreeWarning("test-agent", nil)
}

func TestWorktreeListOKAndOrphaned(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir and a worktree for an agent
	worktreesDir := filepath.Join(wsDir, ".bc", "worktrees")
	if err := os.MkdirAll(filepath.Join(worktreesDir, "eng-01"), 0o750); err != nil {
		t.Fatal(err)
	}
	// Create an orphaned worktree (no matching agent)
	if err := os.MkdirAll(filepath.Join(worktreesDir, "ghost-agent"), 0o750); err != nil {
		t.Fatal(err)
	}

	// Register eng-01 as an agent
	agentsDir := filepath.Join(wsDir, ".bc", "agents")
	agents := map[string]*agent.Agent{
		"eng-01": {Name: "eng-01", Role: agent.Role("engineer"), State: agent.StateIdle},
	}
	data, err := json.Marshal(agents)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, err := executeIntegrationCmd("worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v", err)
	}

	if !strings.Contains(stdout, "eng-01") {
		t.Error("expected eng-01 in output")
	}
	if !strings.Contains(stdout, "OK") {
		t.Error("expected OK status for eng-01")
	}
	if !strings.Contains(stdout, "ghost-agent") {
		t.Error("expected ghost-agent in output")
	}
	if !strings.Contains(stdout, "ORPHANED") {
		t.Error("expected ORPHANED status for ghost-agent")
	}
}

func TestWorktreeListMissing(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Register an agent but don't create its worktree directory
	agentsDir := filepath.Join(wsDir, ".bc", "agents")
	agents := map[string]*agent.Agent{
		"eng-02": {Name: "eng-02", Role: agent.Role("engineer"), State: agent.StateIdle},
	}
	data, err := json.Marshal(agents)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Ensure worktrees dir exists but no eng-02 subdir
	err = os.MkdirAll(filepath.Join(wsDir, ".bc", "worktrees"), 0o750)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, err := executeIntegrationCmd("worktree", "list")
	if err != nil {
		t.Fatalf("worktree list failed: %v", err)
	}

	if !strings.Contains(stdout, "eng-02") {
		t.Error("expected eng-02 in output")
	}
	if !strings.Contains(stdout, "MISSING") {
		t.Error("expected MISSING status for eng-02")
	}
}

func TestWorktreePruneNoOrphans(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir but no orphaned worktrees
	err := os.MkdirAll(filepath.Join(wsDir, ".bc", "worktrees"), 0o750)
	if err != nil {
		t.Fatal(err)
	}

	stdout, _, err := executeIntegrationCmd("worktree", "prune")
	if err != nil {
		t.Fatalf("worktree prune failed: %v", err)
	}

	if !strings.Contains(stdout, "No orphaned worktrees found") {
		t.Errorf("expected 'No orphaned worktrees found', got: %s", stdout)
	}
}

func TestWorktreePruneNoWorktreesDir(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Don't create worktrees dir
	stdout, _, err := executeIntegrationCmd("worktree", "prune")
	if err != nil {
		t.Fatalf("worktree prune failed: %v", err)
	}

	if !strings.Contains(stdout, "No worktrees directory found") {
		t.Errorf("expected 'No worktrees directory found', got: %s", stdout)
	}
}

func TestWorktreePruneDryRun(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir and an orphaned worktree (no matching agent)
	worktreesDir := filepath.Join(wsDir, ".bc", "worktrees")
	orphanedDir := filepath.Join(worktreesDir, "ghost-agent")
	if err := os.MkdirAll(orphanedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Create a dummy file so it's not empty
	if err := os.WriteFile(filepath.Join(orphanedDir, "dummy.txt"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Reset flag to ensure dry-run mode
	worktreePruneForce = false

	stdout, _, err := executeIntegrationCmd("worktree", "prune")
	if err != nil {
		t.Fatalf("worktree prune failed: %v", err)
	}

	if !strings.Contains(stdout, "ghost-agent") {
		t.Errorf("expected ghost-agent in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "dry-run") {
		t.Errorf("expected 'dry-run' message, got: %s", stdout)
	}
	if !strings.Contains(stdout, "not registered") {
		t.Errorf("expected 'not registered' reason, got: %s", stdout)
	}

	// Verify worktree was NOT removed (dry-run)
	if _, err := os.Stat(orphanedDir); os.IsNotExist(err) {
		t.Error("orphaned worktree should NOT be removed in dry-run mode")
	}
}

func TestWorktreePruneForce(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir and an orphaned worktree (no matching agent)
	worktreesDir := filepath.Join(wsDir, ".bc", "worktrees")
	orphanedDir := filepath.Join(worktreesDir, "orphan-agent")
	if err := os.MkdirAll(orphanedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	// Create a dummy file
	if err := os.WriteFile(filepath.Join(orphanedDir, "dummy.txt"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Set force flag
	worktreePruneForce = true
	defer func() { worktreePruneForce = false }()

	stdout, _, err := executeIntegrationCmd("worktree", "prune", "--force")
	if err != nil {
		t.Fatalf("worktree prune --force failed: %v", err)
	}

	if !strings.Contains(stdout, "orphan-agent") {
		t.Errorf("expected orphan-agent in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Pruning") {
		t.Errorf("expected 'Pruning' message, got: %s", stdout)
	}
	if !strings.Contains(stdout, "OK") {
		t.Errorf("expected 'OK' status, got: %s", stdout)
	}

	// Verify worktree was removed
	if _, err := os.Stat(orphanedDir); !os.IsNotExist(err) {
		t.Error("orphaned worktree should be removed with --force")
	}
}

func TestWorktreePruneStoppedAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir and a worktree for a stopped agent
	worktreesDir := filepath.Join(wsDir, ".bc", "worktrees")
	stoppedDir := filepath.Join(worktreesDir, "stopped-eng")
	if err := os.MkdirAll(stoppedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stoppedDir, "dummy.txt"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Register the agent as stopped
	agentsDir := filepath.Join(wsDir, ".bc", "agents")
	agents := map[string]*agent.Agent{
		"stopped-eng": {Name: "stopped-eng", Role: agent.Role("engineer"), State: agent.StateStopped},
	}
	data, err := json.Marshal(agents)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	// Reset flag for dry-run
	worktreePruneForce = false

	stdout, _, err := executeIntegrationCmd("worktree", "prune")
	if err != nil {
		t.Fatalf("worktree prune failed: %v", err)
	}

	if !strings.Contains(stdout, "stopped-eng") {
		t.Errorf("expected stopped-eng in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "stopped") {
		t.Errorf("expected 'stopped' reason, got: %s", stdout)
	}
}

func TestWorktreePruneJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir and an orphaned worktree
	worktreesDir := filepath.Join(wsDir, ".bc", "worktrees")
	orphanedDir := filepath.Join(worktreesDir, "json-orphan")
	if err := os.MkdirAll(orphanedDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(orphanedDir, "dummy.txt"), []byte("test"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Reset flag for dry-run
	worktreePruneForce = false

	stdout, _, err := executeIntegrationCmd("worktree", "prune", "--json")
	if err != nil {
		t.Fatalf("worktree prune --json failed: %v", err)
	}

	var result PruneResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}

	if !result.DryRun {
		t.Error("expected DryRun to be true")
	}
	if len(result.Orphaned) != 1 {
		t.Errorf("expected 1 orphaned worktree, got %d", len(result.Orphaned))
	}
	if result.Orphaned[0].Name != "json-orphan" {
		t.Errorf("expected orphan name 'json-orphan', got %s", result.Orphaned[0].Name)
	}
}

func TestIsEmptyDir(t *testing.T) {
	// Create empty directory
	emptyDir := t.TempDir()

	isEmpty, err := isEmptyDir(emptyDir)
	if err != nil {
		t.Fatalf("isEmptyDir failed: %v", err)
	}
	if !isEmpty {
		t.Error("expected empty directory to return true")
	}

	// Create non-empty directory
	nonEmptyDir := t.TempDir()
	if writeErr := os.WriteFile(filepath.Join(nonEmptyDir, "file.txt"), []byte("data"), 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}

	isEmpty, err = isEmptyDir(nonEmptyDir)
	if err != nil {
		t.Fatalf("isEmptyDir failed: %v", err)
	}
	if isEmpty {
		t.Error("expected non-empty directory to return false")
	}
}

func TestWorktreePruneMultipleOrphans(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create worktrees dir with multiple orphaned worktrees
	worktreesDir := filepath.Join(wsDir, ".bc", "worktrees")
	for _, name := range []string{"orphan-a", "orphan-b", "orphan-c"} {
		dir := filepath.Join(worktreesDir, name)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "dummy.txt"), []byte("test"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	// Set force flag
	worktreePruneForce = true
	defer func() { worktreePruneForce = false }()

	stdout, _, err := executeIntegrationCmd("worktree", "prune", "--force")
	if err != nil {
		t.Fatalf("worktree prune --force failed: %v", err)
	}

	// Check all orphans are mentioned
	for _, name := range []string{"orphan-a", "orphan-b", "orphan-c"} {
		if !strings.Contains(stdout, name) {
			t.Errorf("expected %s in output, got: %s", name, stdout)
		}
	}

	// Check count message
	if !strings.Contains(stdout, "Pruned 3 worktree(s)") {
		t.Errorf("expected 'Pruned 3 worktree(s)' message, got: %s", stdout)
	}

	// Verify all worktrees were removed
	for _, name := range []string{"orphan-a", "orphan-b", "orphan-c"} {
		dir := filepath.Join(worktreesDir, name)
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("worktree %s should be removed", name)
		}
	}
}

// --- isDetachedHead tests ---

func TestIsDetachedHead_NotDetached(t *testing.T) {
	// Create a git repo with a normal branch
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil { //nolint:gosec,noctx // G204: test helper
			t.Fatalf("%s failed: %v (%s)", strings.Join(args, " "), err, out)
		}
	}

	isDetached, err := isDetachedHead(dir)
	if err != nil {
		t.Fatalf("isDetachedHead failed: %v", err)
	}
	if isDetached {
		t.Error("expected not detached, got detached")
	}
}

func TestIsDetachedHead_DetachedClean(t *testing.T) {
	// Create a git repo and detach HEAD
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "commit", "--allow-empty", "-m", "init"},
		{"git", "-C", dir, "checkout", "--detach"},
	}
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil { //nolint:gosec,noctx // G204: test helper
			t.Fatalf("%s failed: %v (%s)", strings.Join(args, " "), err, out)
		}
	}

	isDetached, err := isDetachedHead(dir)
	if err != nil {
		t.Fatalf("isDetachedHead failed: %v", err)
	}
	if !isDetached {
		t.Error("expected detached, got not detached")
	}
}

func TestIsDetachedHead_DetachedWithChanges(t *testing.T) {
	// Create a git repo, detach HEAD, and add uncommitted changes
	dir := t.TempDir()
	cmds := [][]string{
		{"git", "init", "-b", "main", dir},
		{"git", "-C", dir, "config", "user.email", "test@test.com"},
		{"git", "-C", dir, "config", "user.name", "Test"},
		{"git", "-C", dir, "commit", "--allow-empty", "-m", "init"},
		{"git", "-C", dir, "checkout", "--detach"},
	}
	for _, args := range cmds {
		if out, err := exec.Command(args[0], args[1:]...).CombinedOutput(); err != nil { //nolint:gosec,noctx // G204: test helper
			t.Fatalf("%s failed: %v (%s)", strings.Join(args, " "), err, out)
		}
	}

	// Add an uncommitted file
	if err := os.WriteFile(filepath.Join(dir, "uncommitted.txt"), []byte("changes"), 0o600); err != nil {
		t.Fatal(err)
	}

	isDetached, err := isDetachedHead(dir)
	if err != nil {
		t.Fatalf("isDetachedHead failed: %v", err)
	}
	if isDetached {
		t.Error("expected false (has uncommitted changes), got true")
	}
}
