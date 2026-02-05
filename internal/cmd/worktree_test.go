package cmd

import (
	"os"
	"path/filepath"
	"testing"
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

func TestWorktreeCheckNoEnvVar(t *testing.T) {
	// Unset BC_AGENT_WORKTREE to test graceful error
	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Unsetenv("BC_AGENT_WORKTREE")
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		}
	}()

	err := runWorktreeCheck(nil, nil)
	if err == nil {
		t.Error("expected error when BC_AGENT_WORKTREE is not set")
	}
}

func TestWorktreeCheckMatch(t *testing.T) {
	// Create a temp dir to use as worktree
	dir := t.TempDir()

	os.Setenv("BC_AGENT_WORKTREE", dir)
	os.Setenv("BC_AGENT_ID", "test-agent")
	defer os.Unsetenv("BC_AGENT_WORKTREE")
	defer os.Unsetenv("BC_AGENT_ID")

	// cd into the dir
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	err := runWorktreeCheck(nil, nil)
	if err != nil {
		t.Errorf("expected no error when in correct worktree, got: %v", err)
	}
}

func TestWorktreeCheckMismatch(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	os.Setenv("BC_AGENT_WORKTREE", dir1)
	os.Setenv("BC_AGENT_ID", "test-agent")
	defer os.Unsetenv("BC_AGENT_WORKTREE")
	defer os.Unsetenv("BC_AGENT_ID")

	// cd into a different dir
	origDir, _ := os.Getwd()
	os.Chdir(dir2)
	defer os.Chdir(origDir)

	err := runWorktreeCheck(nil, nil)
	if err == nil {
		t.Error("expected error when in wrong directory")
	}
}

func TestWorktreeCheckSubdirectory(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)

	os.Setenv("BC_AGENT_WORKTREE", dir)
	os.Setenv("BC_AGENT_ID", "test-agent")
	defer os.Unsetenv("BC_AGENT_WORKTREE")
	defer os.Unsetenv("BC_AGENT_ID")

	// cd into a subdirectory of the worktree
	origDir, _ := os.Getwd()
	os.Chdir(subdir)
	defer os.Chdir(origDir)

	err := runWorktreeCheck(nil, nil)
	if err != nil {
		t.Errorf("expected no error when in worktree subdirectory, got: %v", err)
	}
}

func TestWorktreeCheckMissingDir(t *testing.T) {
	os.Setenv("BC_AGENT_WORKTREE", "/nonexistent/worktree/path")
	os.Setenv("BC_AGENT_ID", "test-agent")
	defer os.Unsetenv("BC_AGENT_WORKTREE")
	defer os.Unsetenv("BC_AGENT_ID")

	err := runWorktreeCheck(nil, nil)
	if err == nil {
		t.Error("expected error when worktree directory does not exist")
	}
}

func TestCheckWorktreeWarningNoEnv(t *testing.T) {
	// When BC_AGENT_WORKTREE is not set, should silently return
	orig := os.Getenv("BC_AGENT_WORKTREE")
	os.Unsetenv("BC_AGENT_WORKTREE")
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_WORKTREE", orig)
		}
	}()

	// Should not panic
	checkWorktreeWarning("test-agent", nil)
}
