package cmd

import (
	"encoding/json"
	"os"
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
		"eng-01": {Name: "eng-01", Role: agent.RoleEngineer, State: agent.StateIdle},
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
		"eng-02": {Name: "eng-02", Role: agent.RoleEngineer, State: agent.StateIdle},
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
