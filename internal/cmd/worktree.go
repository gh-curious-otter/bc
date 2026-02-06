package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
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

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agent worktrees and their status",
	Long: `Scans all agents in the workspace and reports worktree status for each.

Status per agent:
  OK       — worktree directory exists and is properly linked
  MISSING  — agent has no worktree directory
  ORPHANED — worktree directory exists but agent is not registered

Also detects orphaned worktree directories that don't belong to any agent.`,
	RunE: runWorktreeList,
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
	worktreeCmd.AddCommand(worktreeCheckCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
}

// getCwd is the function used to get the current working directory.
// It defaults to os.Getwd and can be overridden in tests.
var getCwd = os.Getwd

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

	cwd, err := getCwd()
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

// WorktreeListEntry holds the status of one agent's worktree.
type WorktreeListEntry struct {
	Agent  string `json:"agent"`
	Path   string `json:"path"`
	Status string `json:"status"` // OK, MISSING, ORPHANED
}

func runWorktreeList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	worktreesDir := filepath.Join(ws.RootDir, ".bc", "worktrees")

	// Load registered agents
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err = mgr.LoadState(); err != nil {
		return fmt.Errorf("failed to load agent state: %w", err)
	}
	agents := mgr.ListAgents()

	agentNames := make(map[string]bool)
	var entries []WorktreeListEntry

	// Check each agent's worktree
	for _, a := range agents {
		agentNames[a.Name] = true
		wtDir := filepath.Join(worktreesDir, a.Name)

		status := "OK"
		if _, statErr := os.Stat(wtDir); os.IsNotExist(statErr) {
			status = "MISSING"
		}

		entries = append(entries, WorktreeListEntry{
			Agent:  a.Name,
			Path:   wtDir,
			Status: status,
		})
	}

	// Scan for orphaned worktrees (dirs in worktrees/ not matching any agent)
	dirEntries, err := os.ReadDir(worktreesDir)
	if err == nil {
		for _, d := range dirEntries {
			if !d.IsDir() {
				continue
			}
			if !agentNames[d.Name()] {
				entries = append(entries, WorktreeListEntry{
					Agent:  d.Name(),
					Path:   filepath.Join(worktreesDir, d.Name()),
					Status: "ORPHANED",
				})
			}
		}
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	// Table output
	fmt.Printf("%-20s %-10s %s\n", "AGENT", "STATUS", "PATH")
	for _, e := range entries {
		fmt.Printf("%-20s %-10s %s\n", e.Agent, e.Status, e.Path)
	}

	// Summary
	ok, missing, orphaned := 0, 0, 0
	for _, e := range entries {
		switch e.Status {
		case "OK":
			ok++
		case "MISSING":
			missing++
		case "ORPHANED":
			orphaned++
		}
	}
	fmt.Printf("\nTotal: %d  OK: %d  Missing: %d  Orphaned: %d\n", len(entries), ok, missing, orphaned)

	return nil
}
