package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Worktree management commands",
}

var worktreeCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if the agent is running in its assigned worktree",
	Long: `Validates the current working directory against the agent's assigned worktree.

Checks:
  1. BC_AGENT_WORKTREE env var is set
  2. The worktree directory exists
  3. The current working directory is within the worktree

Exit code 0 if all checks pass, 1 otherwise.
Useful for QA diagnostics and agent self-checks.`,
	RunE: runWorktreeCheck,
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
	worktreeCmd.AddCommand(worktreeCheckCmd)
}

// WorktreeStatus holds the result of a worktree check.
type WorktreeStatus struct {
	Expected string `json:"expected_worktree"`
	Actual   string `json:"actual_cwd"`
	Exists   bool   `json:"worktree_exists"`
	Match    bool   `json:"match"`
	AgentID  string `json:"agent_id"`
}

func runWorktreeCheck(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	worktree := os.Getenv("BC_AGENT_WORKTREE")

	if worktree == "" {
		fmt.Fprintln(os.Stderr, "BC_AGENT_WORKTREE not set (not running as a bc agent, or Phase A env var not applied)")
		return fmt.Errorf("BC_AGENT_WORKTREE not set")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Resolve symlinks for accurate comparison
	worktreeAbs, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		// Worktree path doesn't exist
		fmt.Fprintf(os.Stderr, "WARNING: worktree directory does not exist: %s\n", worktree)
		fmt.Printf("Agent:    %s\n", agentID)
		fmt.Printf("Expected: %s\n", worktree)
		fmt.Printf("Actual:   %s\n", cwd)
		fmt.Printf("Exists:   no\n")
		fmt.Printf("Match:    no\n")
		return fmt.Errorf("worktree directory does not exist")
	}

	cwdAbs, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return fmt.Errorf("failed to resolve cwd: %w", err)
	}

	// Check if cwd is within the worktree (exact match or subdirectory)
	match := isWithinDir(cwdAbs, worktreeAbs)

	// Check if worktree dir exists
	exists := true
	if _, err := os.Stat(worktreeAbs); err != nil {
		exists = false
	}

	status := "OK"
	if !match {
		status = "MISMATCH"
		fmt.Fprintf(os.Stderr, "WARNING: current directory is outside agent worktree\n")
	}

	fmt.Printf("Agent:    %s\n", agentID)
	fmt.Printf("Expected: %s\n", worktreeAbs)
	fmt.Printf("Actual:   %s\n", cwdAbs)
	fmt.Printf("Exists:   %v\n", exists)
	fmt.Printf("Status:   %s\n", status)

	if !match {
		return fmt.Errorf("working directory mismatch")
	}
	return nil
}

// isWithinDir checks if child is equal to or a subdirectory of parent.
func isWithinDir(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	// rel should not start with ".." if child is within parent
	return rel == "." || (len(rel) > 0 && rel[0] != '.')
}
