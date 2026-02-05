// Package git provides worktree-aware git command wrappers for bc agents.
// All write operations validate that the working directory is within the
// agent's assigned worktree (BC_AGENT_WORKTREE) before executing.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrOutsideWorktree is returned when a git write operation is attempted
// outside the agent's assigned worktree.
var ErrOutsideWorktree = fmt.Errorf("git operation outside assigned worktree")

// Run executes a git command in the given directory. Write operations
// (add, commit, push, checkout, reset, etc.) are rejected if dir is
// outside BC_AGENT_WORKTREE. Read operations (status, log, diff) are
// always allowed.
func Run(dir string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("no git command specified")
	}

	if isWriteOp(args[0]) {
		if err := validateWorktree(dir); err != nil {
			return "", err
		}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s failed: %w (%s)", args[0], err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// Status runs git status in the given directory.
func Status(dir string) (string, error) {
	return Run(dir, "status", "--short")
}

// Add stages files. Validates worktree before executing.
func Add(dir string, files ...string) error {
	args := append([]string{"add"}, files...)
	_, err := Run(dir, args...)
	return err
}

// Commit creates a commit with the given message. Validates worktree.
func Commit(dir, message string) error {
	_, err := Run(dir, "commit", "-m", message)
	return err
}

// Push pushes the current branch. Validates worktree.
func Push(dir string, args ...string) error {
	cmdArgs := append([]string{"push"}, args...)
	_, err := Run(dir, cmdArgs...)
	return err
}

// CheckoutBranch creates and switches to a new branch. Validates worktree.
func CheckoutBranch(dir, branch string) error {
	_, err := Run(dir, "checkout", "-b", branch)
	return err
}

// Diff runs git diff (read-only, no worktree check).
func Diff(dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"diff"}, args...)
	return Run(dir, cmdArgs...)
}

// Log runs git log (read-only, no worktree check).
func Log(dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"log"}, args...)
	return Run(dir, cmdArgs...)
}

// isWriteOp returns true if the git subcommand modifies the repo.
func isWriteOp(subcmd string) bool {
	switch subcmd {
	case "add", "commit", "push", "checkout", "reset", "clean",
		"merge", "rebase", "stash", "rm", "mv", "init", "pull",
		"cherry-pick", "revert", "tag", "branch":
		return true
	}
	return false
}

// validateWorktree checks that dir is within the agent's assigned worktree.
// Returns nil if BC_AGENT_WORKTREE is unset (non-agent context).
func validateWorktree(dir string) error {
	worktree := os.Getenv("BC_AGENT_WORKTREE")
	if worktree == "" {
		return nil // Not running as an agent; no restriction
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve path %q: %w", dir, err)
	}

	absWorktree, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return fmt.Errorf("worktree path %q does not exist: %w", worktree, err)
	}

	absDir, _ = filepath.EvalSymlinks(absDir)

	rel, err := filepath.Rel(absWorktree, absDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("%w: %s is outside %s", ErrOutsideWorktree, absDir, absWorktree)
	}
	return nil
}
