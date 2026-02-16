package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Worktree management commands",
	Long: `Manage git worktrees for bc agents.

Each agent operates in its own git worktree, providing isolated working
directories while sharing the same repository. This enables parallel
development without branch conflicts.

Worktree locations: .bc/worktrees/<agent-name>/

Examples:
  bc worktree list              # List all worktrees and their status
  bc worktree list --orphaned   # Show only orphaned worktrees
  bc worktree prune             # Dry-run: show what would be cleaned
  bc worktree prune --force     # Remove orphaned worktrees
  bc worktree check             # Verify current agent's worktree`,
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

var (
	worktreePruneForce bool
	worktreeListOrphan bool
)

var worktreePruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Clean up orphaned worktrees",
	Long: `Identifies and removes orphaned worktrees from previous agent sessions.

A worktree is considered orphaned if:
  - Not associated with a running agent (no active tmux session)
  - OR its agent is not registered in the workspace

By default, shows what would be pruned (dry-run). Use --force to actually remove.

Examples:
  bc worktree prune           # Dry-run: show what would be pruned
  bc worktree prune --force   # Actually prune orphaned worktrees`,
	RunE: runWorktreePrune,
}

func init() {
	rootCmd.AddCommand(worktreeCmd)
	worktreeCmd.AddCommand(worktreeCheckCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreePruneCmd)
	worktreeListCmd.Flags().BoolVar(&worktreeListOrphan, "orphaned", false, "Show only orphaned worktrees")
	worktreePruneCmd.Flags().BoolVarP(&worktreePruneForce, "force", "f", false, "Actually remove orphaned worktrees (default is dry-run)")
}

// getCwd is the function used to get the current working directory.
// It defaults to os.Getwd and can be overridden in tests.
var getCwd = os.Getwd

// WorktreeStatus holds the result of a worktree check.
type WorktreeStatus struct {
	Expected string `json:"expected_worktree"`
	Actual   string `json:"actual_cwd"`
	AgentID  string `json:"agent_id"`
	Exists   bool   `json:"worktree_exists"`
	Match    bool   `json:"match"`
}

func runWorktreeCheck(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	worktree := os.Getenv("BC_AGENT_WORKTREE")

	if worktree == "" {
		return errorWorktreeNotSet()
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
		return errNotInWorkspace(err)
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

	// Filter to orphaned only if requested
	if worktreeListOrphan {
		var filtered []WorktreeListEntry
		for _, e := range entries {
			if e.Status == "ORPHANED" {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
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

	if len(entries) == 0 {
		if worktreeListOrphan {
			fmt.Println("No orphaned worktrees found.")
		} else {
			fmt.Println("No worktrees found.")
		}
		return nil
	}

	// Table output
	fmt.Printf("%-20s %-10s %s\n", "AGENT", "STATUS", "PATH")
	for _, e := range entries {
		fmt.Printf("%-20s %-10s %s\n", e.Agent, e.Status, e.Path)
	}

	// Summary (only when not filtering)
	if !worktreeListOrphan {
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
	} else {
		fmt.Printf("\nOrphaned: %d\n", len(entries))
	}

	return nil
}

// OrphanedWorktree holds information about an orphaned worktree.
type OrphanedWorktree struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// PruneResult holds the result of a prune operation.
type PruneResult struct {
	Orphaned []OrphanedWorktree `json:"orphaned"`
	Pruned   []string           `json:"pruned"`
	Errors   []string           `json:"errors,omitempty"`
	DryRun   bool               `json:"dry_run"`
}

func runWorktreePrune(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	worktreesDir := filepath.Join(ws.RootDir, ".bc", "worktrees")

	// Load registered agents
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err = mgr.LoadState(); err != nil {
		// Warn but continue - missing state file is fine
		fmt.Fprintf(os.Stderr, "Warning: failed to load agent state: %v\n", err)
	}

	// Build maps of registered and active agents from workspace state
	// This is more reliable than tmux session matching which can fail due to
	// workspace hash mismatches in worktree environments
	agents := mgr.ListAgents()
	registeredAgents := make(map[string]bool)
	activeAgents := make(map[string]bool)
	for _, a := range agents {
		registeredAgents[a.Name] = true
		// Consider agent "active" if state is idle, starting, or working
		// These states indicate the agent session should not be pruned
		switch a.State {
		case agent.StateIdle, agent.StateStarting, agent.StateWorking:
			activeAgents[a.Name] = true
		}
	}

	result := PruneResult{
		DryRun: !worktreePruneForce,
	}

	// Scan worktrees directory for orphaned worktrees
	dirEntries, err := os.ReadDir(worktreesDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No worktrees directory - nothing to prune
			fmt.Println("No worktrees directory found. Nothing to prune.")
			return nil
		}
		return fmt.Errorf("failed to read worktrees directory: %w", err)
	}

	for _, d := range dirEntries {
		if !d.IsDir() {
			continue
		}

		name := d.Name()
		wtPath := filepath.Join(worktreesDir, name)

		// Determine if this worktree is orphaned and why
		reason := ""

		if !registeredAgents[name] {
			reason = "not registered as an agent"
		} else if !activeAgents[name] {
			// Agent is registered but not active (running/busy)
			if a := mgr.GetAgent(name); a != nil && a.State == agent.StateStopped {
				reason = "agent is stopped"
			}

			// Only check for empty/detached HEAD when agent is NOT active
			// Active agents may have detached HEAD as normal git operation
			if reason == "" {
				if isEmpty, _ := isEmptyDir(wtPath); isEmpty {
					reason = "worktree directory is empty"
				} else if isDetached, _ := isDetachedHead(wtPath); isDetached {
					reason = "worktree has detached HEAD with no changes"
				}
			}
		}

		if reason != "" {
			result.Orphaned = append(result.Orphaned, OrphanedWorktree{
				Name:   name,
				Path:   wtPath,
				Reason: reason,
			})
		}
	}

	// Handle output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// In dry-run mode, just show orphaned worktrees
		if !worktreePruneForce {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}
	}

	if len(result.Orphaned) == 0 {
		if !jsonOutput {
			fmt.Println("No orphaned worktrees found. Nothing to prune.")
		}
		// Run git worktree prune anyway to clean up stale entries
		_ = runGitWorktreePrune(ws.RootDir)
		return nil
	}

	// Show what would be/will be pruned
	if !jsonOutput {
		if worktreePruneForce {
			fmt.Println("Pruning orphaned worktrees:")
		} else {
			fmt.Println("Orphaned worktrees (dry-run, use --force to remove):")
		}
		fmt.Println()
		fmt.Printf("%-20s %-40s %s\n", "NAME", "PATH", "REASON")
		for _, o := range result.Orphaned {
			fmt.Printf("%-20s %-40s %s\n", o.Name, o.Path, o.Reason)
		}
		fmt.Println()
	}

	// Actually prune if --force is set
	if worktreePruneForce {
		for _, o := range result.Orphaned {
			if !jsonOutput {
				fmt.Printf("Removing %s... ", o.Name)
			}

			// Try git worktree remove first
			if err := removeWorktreeGit(ws.RootDir, o.Path); err != nil {
				// Fall back to removing directory directly
				if rmErr := os.RemoveAll(o.Path); rmErr != nil {
					errMsg := fmt.Sprintf("failed to remove %s: %v", o.Name, rmErr)
					result.Errors = append(result.Errors, errMsg)
					if !jsonOutput {
						fmt.Println("FAILED")
						fmt.Printf("  Error: %v\n", rmErr)
					}
					continue
				}
			}

			result.Pruned = append(result.Pruned, o.Name)
			if !jsonOutput {
				fmt.Println("OK")
			}
		}

		// Run git worktree prune to clean up any stale worktree entries
		if !jsonOutput {
			fmt.Print("\nRunning git worktree prune... ")
		}
		if err := runGitWorktreePrune(ws.RootDir); err != nil {
			if !jsonOutput {
				fmt.Println("WARNING")
				fmt.Printf("  Warning: git worktree prune failed: %v\n", err)
			}
		} else if !jsonOutput {
			fmt.Println("OK")
		}

		if !jsonOutput {
			fmt.Printf("\nPruned %d worktree(s)\n", len(result.Pruned))
		}
	} else if !jsonOutput {
		fmt.Printf("Found %d orphaned worktree(s). Use --force to remove.\n", len(result.Orphaned))
	}

	if jsonOutput && worktreePruneForce {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	return nil
}

// isEmptyDir checks if a directory is empty.
func isEmptyDir(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// isDetachedHead checks if a git worktree has a detached HEAD with no uncommitted changes.
func isDetachedHead(path string) (bool, error) {
	// Check if HEAD is detached
	cmd := exec.CommandContext(context.Background(), "git", "-C", path, "symbolic-ref", "-q", "HEAD")
	if err := cmd.Run(); err == nil {
		// symbolic-ref succeeded, so HEAD is not detached
		return false, nil
	}

	// HEAD is detached - check if there are uncommitted changes
	cmd = exec.CommandContext(context.Background(), "git", "-C", path, "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// If there are changes, don't consider it for pruning
	if len(output) > 0 {
		return false, nil
	}

	return true, nil
}

// removeWorktreeGit removes a worktree using git worktree remove.
func removeWorktreeGit(workspace, worktreePath string) error {
	cmd := exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "remove", "--force", worktreePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}

// runGitWorktreePrune runs git worktree prune to clean up stale worktree entries.
func runGitWorktreePrune(workspace string) error {
	cmd := exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "prune")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, string(output))
	}
	return nil
}
