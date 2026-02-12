package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/workspace"
)

var (
	mergeSkipTests bool
	mergeDryRun    bool
	mergeYes       bool
	mergeRebase    bool
	mergeNoRebase  bool
	mergeStatus    bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge [agent-name|branch]",
	Short: "Merge an agent branch into main after validation",
	Long: `Merge an agent's work branch into main after running build, test, and vet checks.

The merge command:
  1. Checks for conflicts with main
  2. Optionally rebases branch onto main (--rebase)
  3. Runs go build, go test, go vet in the agent worktree
  4. Merges the branch into main (fast-forward or merge commit)

Use --status to view pending merges and their state.

Flags:
  --dry-run     Check for conflicts without merging
  --yes         Proceed without confirmation (for automation)
  --skip-tests  Skip build/test/vet validation
  --rebase      Rebase branch onto main before merging
  --no-rebase   Skip auto-rebase even if branch is stale
  --status      Show merge queue status

Examples:
  bc merge engineer-01
  bc merge engineer-01 --dry-run
  bc merge engineer-01 --rebase
  bc merge fix/enter-submit-reliability --skip-tests
  bc merge engineer-02 --yes
  bc merge --status              # Show pending merges
  bc merge --status --json       # JSON output`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMerge,
}

func init() {
	mergeCmd.Flags().BoolVar(&mergeSkipTests, "skip-tests", false, "Skip build/test/vet validation")
	mergeCmd.Flags().BoolVar(&mergeDryRun, "dry-run", false, "Check for conflicts without merging")
	mergeCmd.Flags().BoolVar(&mergeYes, "yes", false, "Proceed without confirmation (non-interactive)")
	mergeCmd.Flags().BoolVar(&mergeRebase, "rebase", false, "Rebase branch onto main before merging")
	mergeCmd.Flags().BoolVar(&mergeNoRebase, "no-rebase", false, "Skip auto-rebase even if branch is stale")
	mergeCmd.Flags().BoolVar(&mergeStatus, "status", false, "Show merge queue status")
	mergeCmd.Flags().Bool("json", false, "Output status as JSON (with --status)")
	rootCmd.AddCommand(mergeCmd)
}

func runMerge(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Handle --status flag
	if mergeStatus {
		return runMergeStatus(cmd, ws)
	}

	// Require target for actual merge
	if len(args) == 0 {
		return fmt.Errorf("requires agent-name or branch argument (use --status to view queue)")
	}

	target := args[0]
	rootDir := ws.RootDir

	// Determine the branch to merge. If target matches an agent name,
	// discover the branch from the agent's worktree. Otherwise treat it
	// as a literal branch name.
	branch, worktreeDir, err := resolveMergeTarget(ws.AgentsDir(), rootDir, target)
	if err != nil {
		return err
	}

	if mergeDryRun {
		fmt.Printf("Checking branch %s for conflicts with main...\n", branch)
	} else {
		fmt.Printf("Merging branch %s into main...\n", branch)
	}

	// Step 1: Check that the branch exists
	if err = gitBranchExists(rootDir, branch); err != nil {
		return fmt.Errorf("branch %s not found: %w", branch, err)
	}

	// Step 2: Auto-rebase if requested and branch is stale
	if mergeRebase && !mergeNoRebase && worktreeDir != "" {
		stale, behindCount, staleErr := isBranchStale(rootDir, branch)
		if staleErr != nil {
			return fmt.Errorf("failed to check if branch is stale: %w", staleErr)
		}
		if stale {
			fmt.Printf("  Branch is %d commit(s) behind main, rebasing...\n", behindCount)
			if rebaseErr := rebaseBranchOntoMain(worktreeDir); rebaseErr != nil {
				return fmt.Errorf("rebase failed: %w\n\nTo resolve:\n  1. cd %s\n  2. git rebase --abort (if needed)\n  3. git fetch origin main\n  4. git rebase origin/main\n  5. Resolve conflicts and continue", rebaseErr, worktreeDir)
			}
			fmt.Println("  Rebase successful")
		} else {
			fmt.Println("  Branch is up to date with main")
		}
	} else if mergeRebase && worktreeDir == "" {
		fmt.Println("  Skipping rebase (no worktree directory for literal branch)")
	}

	// Step 3: Check for conflicts with main (after potential rebase)
	conflicts, err := checkMergeConflicts(rootDir, branch)
	if err != nil {
		return fmt.Errorf("failed to check conflicts: %w", err)
	}
	if len(conflicts) > 0 {
		fmt.Println("Conflicting files:")
		for _, f := range conflicts {
			fmt.Printf("  - %s\n", f)
		}

		// Notify the responsible agent about the conflicts
		if notifyErr := notifyConflicts(rootDir, branch, conflicts); notifyErr != nil {
			log.Warn("failed to send conflict notification", "error", notifyErr)
		}

		if mergeDryRun {
			return fmt.Errorf("dry-run: branch %s has %d conflicting file(s) with main", branch, len(conflicts))
		}
		if !mergeYes {
			fmt.Printf("\nBranch %s has conflicts with main. Resolve conflicts before merging.\n", branch)
			return fmt.Errorf("branch %s has conflicts with main — resolve before merging", branch)
		}
		// With --yes flag, user explicitly wants to proceed despite conflicts
		// This is unusual but allowed for automation scenarios
		fmt.Println("  Proceeding despite conflicts (--yes flag)")
	} else {
		fmt.Println("  No conflicts with main")
	}

	// If dry-run mode, exit after conflict check
	if mergeDryRun {
		fmt.Printf("Dry-run complete: branch %s can be cleanly merged into main\n", branch)
		return nil
	}

	// Step 4: Run validation (build, test, vet) in the source directory
	if !mergeSkipTests {
		validateDir := worktreeDir
		if validateDir == "" {
			validateDir = rootDir
		}
		if err = runValidation(validateDir); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	} else {
		fmt.Println("  Skipping validation (--skip-tests)")
	}

	// Step 5: Save restore point and perform atomic merge
	restorePoint, err := gitRevParse(rootDir, "main")
	if err != nil {
		return fmt.Errorf("failed to get main HEAD for restore point: %w", err)
	}
	fmt.Printf("  Restore point: %s\n", restorePoint[:12])

	commitHash, err := mergeBranch(rootDir, branch)
	if err != nil {
		// Rollback: restore main to pre-merge state
		if rollbackErr := rollbackMerge(rootDir, restorePoint); rollbackErr != nil {
			return fmt.Errorf("merge failed and rollback also failed: merge error: %w, rollback error: %v", err, rollbackErr)
		}
		fmt.Printf("  ⚠️  Merge failed — rolled back main to %s\n", restorePoint[:12])
		return fmt.Errorf("merge failed (rolled back): %w", err)
	}
	fmt.Printf("  Merged at %s\n", commitHash)

	// Step 6: Log event
	evLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	_ = evLog.Append(events.Event{
		Type:    events.WorkCompleted,
		Message: fmt.Sprintf("merged %s into main at %s", branch, commitHash),
		Data: map[string]any{
			"branch": branch,
			"commit": commitHash,
		},
	})

	fmt.Printf("Successfully merged %s into main\n", branch)
	return nil
}

// resolveMergeTarget resolves a target (agent name or branch) to a branch name
// and optionally an agent worktree directory for validation.
func resolveMergeTarget(agentsDir, rootDir, target string) (branch string, worktreeDir string, err error) {
	mgr := agent.NewWorkspaceManager(agentsDir, rootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(target)
	if a != nil {
		// Target is an agent name — discover branch from worktree
		wDir := a.WorktreeDir
		if wDir == "" {
			wDir = filepath.Join(rootDir, ".bc", "worktrees", target)
		}

		branchName, brErr := gitCurrentBranch(wDir)
		if brErr != nil {
			return "", "", fmt.Errorf("agent %s has no active branch: %w", target, brErr)
		}
		return branchName, wDir, nil
	}

	// Target is a literal branch name
	return target, "", nil
}

// gitBranchExists checks if a branch exists.
func gitBranchExists(repoDir, branch string) error {
	cmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "rev-parse", "--verify", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// gitCurrentBranch returns the current branch of a directory.
func gitCurrentBranch(dir string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse failed: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	if branch == "HEAD" {
		return "", fmt.Errorf("worktree is in detached HEAD state")
	}
	return branch, nil
}

// checkMergeConflicts does a trial merge to detect conflicts without
// changing the working tree. Returns the list of conflicting files.
func checkMergeConflicts(repoDir, branch string) ([]string, error) {
	// Get merge base
	baseCmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "merge-base", "main", branch)
	baseOut, err := baseCmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("merge-base: %s", strings.TrimSpace(string(baseOut)))
	}
	mergeBase := strings.TrimSpace(string(baseOut))

	// Check if branch is already up to date (fast-forward possible)
	mainHead, err := gitRevParse(repoDir, "main")
	if err != nil {
		return nil, err
	}
	if mergeBase == mainHead {
		return nil, nil // Fast-forward, no conflicts possible
	}

	// Use merge-tree --write-tree to detect conflicts without touching worktree.
	// Exit code 0 = clean merge, exit code 1 = conflicts detected.
	// The first line of output is the resulting tree hash.
	// With --name-only, conflicting file paths follow after a blank line.
	treeCmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "merge-tree", "--write-tree", "--name-only", "main", branch)
	out, err := treeCmd.CombinedOutput()
	if err == nil {
		return nil, nil // Clean merge possible
	}

	// Parse conflict info: first line is tree hash, then blank line, then file paths
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var conflicts []string
	pastBlank := false
	for _, line := range lines[1:] { // skip tree hash on first line
		if line == "" {
			pastBlank = true
			continue
		}
		if pastBlank {
			conflicts = append(conflicts, line)
		}
	}
	if len(conflicts) > 0 {
		return conflicts, nil
	}
	return nil, fmt.Errorf("merge-tree failed: %s", strings.TrimSpace(string(out)))
}

// gitRevParse returns the SHA of a ref.
func gitRevParse(repoDir, ref string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "rev-parse", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rev-parse %s: %s", ref, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// runValidation runs go build, go test, and go vet in the given directory.
func runValidation(dir string) error {
	checks := []struct {
		name string
		args []string
	}{
		{"build", []string{"go", "build", "./..."}},
		{"test", []string{"go", "test", "./..."}},
		{"vet", []string{"go", "vet", "./..."}},
	}

	for _, check := range checks {
		fmt.Printf("  Running go %s...\n", check.name)
		cmd := exec.CommandContext(context.Background(), check.args[0], check.args[1:]...) //nolint:gosec // G204: args from hardcoded list
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("go %s failed:\n%s", check.name, string(out))
		}
	}

	fmt.Println("  All checks passed")
	return nil
}

// mergeBranch merges the given branch into main and returns the resulting commit hash.
func mergeBranch(repoDir, branch string) (string, error) {
	// Check if fast-forward is possible
	mainHead, err := gitRevParse(repoDir, "main")
	if err != nil {
		return "", err
	}

	baseCmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "merge-base", "main", branch)
	baseOut, err := baseCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("merge-base: %s", strings.TrimSpace(string(baseOut)))
	}
	mergeBase := strings.TrimSpace(string(baseOut))

	if mergeBase == mainHead {
		// Fast-forward: move main to branch HEAD using update-ref
		// (works even when main is checked out in another worktree)
		branchHead, branchErr := gitRevParse(repoDir, branch)
		if branchErr != nil {
			return "", branchErr
		}
		cmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "update-ref", "refs/heads/main", branchHead) //nolint:gosec // G204: git command with validated repo dir and branch head
		if out, cmdErr := cmd.CombinedOutput(); cmdErr != nil {
			return "", fmt.Errorf("fast-forward failed: %s", strings.TrimSpace(string(out)))
		}
		return branchHead[:12], nil
	}

	// Non-fast-forward: create a merge commit
	// We need to be on main to merge. Use a temporary worktree if main
	// is checked out elsewhere.
	mergeMsg := fmt.Sprintf("Merge branch '%s' into main", branch)

	// Try direct merge first (works if we can update main ref)
	cmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "merge-tree", "--write-tree", "main", branch)
	treeOut, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("merge-tree failed: %s", strings.TrimSpace(string(treeOut)))
	}
	treeHash := strings.TrimSpace(string(treeOut))

	// Create merge commit using commit-tree
	branchHead, err := gitRevParse(repoDir, branch)
	if err != nil {
		return "", err
	}
	commitCmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "commit-tree", treeHash, //nolint:gosec // G204: git command with validated git objects
		"-p", mainHead, "-p", branchHead, "-m", mergeMsg)
	commitOut, err := commitCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("commit-tree failed: %s", strings.TrimSpace(string(commitOut)))
	}
	mergeCommit := strings.TrimSpace(string(commitOut))

	// Update main ref to point to the merge commit
	updateCmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "update-ref", "refs/heads/main", mergeCommit) //nolint:gosec // G204: git command with validated commit hash
	if out, err := updateCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("update-ref failed: %s", strings.TrimSpace(string(out)))
	}

	return mergeCommit[:12], nil
}

// rollbackMerge restores main to the given commit hash.
// This is used when a merge operation fails partway through.
func rollbackMerge(repoDir, restorePoint string) error {
	cmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "update-ref", "refs/heads/main", restorePoint) //nolint:gosec // G204: git command with validated restore point
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("update-ref failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// isBranchStale checks if a branch is behind main.
// Returns true if the branch needs rebasing, along with the number of commits behind.
func isBranchStale(repoDir, branch string) (bool, int, error) {
	// Count commits that main has but branch doesn't
	cmd := exec.CommandContext(context.Background(), "git", "-C", repoDir, "rev-list", "--count", branch+"..main") //nolint:gosec // G204: git command with validated branch name
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false, 0, fmt.Errorf("rev-list failed: %s", strings.TrimSpace(string(out)))
	}

	countStr := strings.TrimSpace(string(out))
	var count int
	if _, parseErr := fmt.Sscanf(countStr, "%d", &count); parseErr != nil {
		return false, 0, fmt.Errorf("failed to parse commit count: %s", countStr)
	}

	return count > 0, count, nil
}

// rebaseBranchOntoMain rebases the current branch in a worktree onto main.
// Uses --autostash to safely handle uncommitted changes.
func rebaseBranchOntoMain(worktreeDir string) error {
	// Fetch latest main first
	fetchCmd := exec.CommandContext(context.Background(), "git", "-C", worktreeDir, "fetch", "origin", "main")
	if out, err := fetchCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("fetch failed: %s", strings.TrimSpace(string(out)))
	}

	// Rebase with autostash for safety
	rebaseCmd := exec.CommandContext(context.Background(), "git", "-C", worktreeDir, "rebase", "--autostash", "origin/main")
	if out, err := rebaseCmd.CombinedOutput(); err != nil {
		// Abort the rebase to leave worktree in clean state
		abortCmd := exec.CommandContext(context.Background(), "git", "-C", worktreeDir, "rebase", "--abort")
		_ = abortCmd.Run() // Best effort abort
		return fmt.Errorf("rebase conflicts detected:\n%s", strings.TrimSpace(string(out)))
	}

	return nil
}

// notifyConflicts sends a notification to the agent responsible for the conflicts.
// It identifies the agent from the branch name and sends a channel notification
// with conflict details and resolution steps.
func notifyConflicts(rootDir, branch string, conflicts []string) error {
	// Get the branch head commit for context
	branchHead, err := gitRevParse(rootDir, branch)
	if err != nil {
		branchHead = "unknown"
	}

	mainHead, err := gitRevParse(rootDir, "main")
	if err != nil {
		mainHead = "unknown"
	}

	// Identify the agent from the branch name (e.g., engineer-01/issue-123/feature)
	responsibleAgent := extractAgentFromBranch(branch)

	// Build notification message
	var sb strings.Builder
	sb.WriteString("⚠️ **Merge Conflict Detected**\n\n")
	sb.WriteString(fmt.Sprintf("Branch `%s` has conflicts with `main`.\n\n", branch))
	sb.WriteString("**Conflicting files:**\n")
	for _, f := range conflicts {
		sb.WriteString(fmt.Sprintf("  - `%s`\n", f))
	}
	sb.WriteString("\n**Commit details:**\n")
	sb.WriteString(fmt.Sprintf("  - Branch HEAD: `%s`\n", truncateSHA(branchHead)))
	sb.WriteString(fmt.Sprintf("  - Main HEAD: `%s`\n", truncateSHA(mainHead)))
	sb.WriteString("\n**Resolution steps:**\n")
	sb.WriteString("1. `git fetch origin main`\n")
	sb.WriteString("2. `git rebase origin/main`\n")
	sb.WriteString("3. Resolve conflicts in listed files\n")
	sb.WriteString("4. `git add .`\n")
	sb.WriteString("5. `git rebase --continue`\n")
	sb.WriteString("6. `git push --force-with-lease`\n")

	message := sb.String()

	// Load channel store and send notification (use OpenStore to match bc up / CLI)
	store, err := channel.OpenStore(rootDir)
	if err != nil {
		store = channel.NewStore(rootDir)
	}
	defer func() { _ = store.Close() }()
	if err := store.Load(); err != nil {
		return fmt.Errorf("failed to load channel store: %w", err)
	}

	// Determine which channel to notify
	// Priority: engineering channel if available, otherwise all channel
	notifyChannel := "engineering"
	if _, exists := store.Get(notifyChannel); !exists {
		notifyChannel = "all"
		if _, exists := store.Get(notifyChannel); !exists {
			// Create all channel if it doesn't exist
			if _, err := store.Create("all"); err != nil {
				return fmt.Errorf("failed to create all channel: %w", err)
			}
		}
	}

	// Add message to channel history
	sender := "merge-bot"
	if envSender := os.Getenv("BC_AGENT_ID"); envSender != "" {
		sender = envSender
	}

	if err := store.AddHistory(notifyChannel, sender, message); err != nil {
		return fmt.Errorf("failed to add conflict notification to history: %w", err)
	}

	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save channel store: %w", err)
	}

	fmt.Printf("  Conflict notification sent to #%s", notifyChannel)
	if responsibleAgent != "" {
		fmt.Printf(" (responsible: @%s)", responsibleAgent)
	}
	fmt.Println()

	return nil
}

// extractAgentFromBranch extracts the agent name from a branch name.
// Branch naming convention: agent-name/issue-XXX/description
func extractAgentFromBranch(branch string) string {
	parts := strings.SplitN(branch, "/", 2)
	if len(parts) > 0 {
		// Check if the first part looks like an agent name
		agentName := parts[0]
		if strings.HasPrefix(agentName, "engineer-") ||
			strings.HasPrefix(agentName, "tech-lead-") ||
			strings.HasPrefix(agentName, "qa-") ||
			agentName == "coordinator" ||
			agentName == "manager" {
			return agentName
		}
	}
	return ""
}

// truncateSHA returns the first 12 characters of a SHA, or the full string if shorter.
func truncateSHA(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// MergeQueueItem represents a pending or in-progress merge.
type MergeQueueItem struct {
	StartedAt   time.Time `json:"started_at,omitempty"`
	Agent       string    `json:"agent"`
	Branch      string    `json:"branch"`
	Target      string    `json:"target"`
	State       string    `json:"state"` // pending, in-progress, blocked
	HasConflict bool      `json:"has_conflict,omitempty"`
}

// runMergeStatus displays the merge queue status.
func runMergeStatus(cmd *cobra.Command, ws *workspace.Workspace) error {
	rootDir := ws.RootDir
	agentsDir := ws.AgentsDir()

	// Load agents
	mgr := agent.NewWorkspaceManager(agentsDir, rootDir)
	if err := mgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}

	agents := mgr.ListAgents()
	var items []MergeQueueItem

	// Check each agent for mergeable branches
	for _, a := range agents {
		if a.State == agent.StateStopped || a.State == agent.StateError {
			continue
		}

		// Get agent's worktree directory
		wDir := a.WorktreeDir
		if wDir == "" {
			wDir = filepath.Join(rootDir, ".bc", "worktrees", a.Name)
		}

		// Check if worktree exists
		if _, err := os.Stat(wDir); os.IsNotExist(err) {
			continue
		}

		// Get current branch
		branch, err := gitCurrentBranch(wDir)
		if err != nil {
			continue
		}

		// Skip if on main
		if branch == "main" {
			continue
		}

		// Check for conflicts
		conflicts, err := checkMergeConflicts(rootDir, branch)
		hasConflict := err == nil && len(conflicts) > 0

		state := "pending"
		if hasConflict {
			state = "blocked"
		}
		if a.State == agent.StateWorking {
			state = "in-progress"
		}

		items = append(items, MergeQueueItem{
			Agent:       a.Name,
			Branch:      branch,
			Target:      "main",
			State:       state,
			HasConflict: hasConflict,
		})
	}

	// Check for JSON output
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}

	// Display table format
	if len(items) == 0 {
		fmt.Println("No pending merges")
		return nil
	}

	fmt.Printf("%-15s %-40s %-10s %s\n", "AGENT", "BRANCH", "TARGET", "STATE")
	fmt.Println(strings.Repeat("-", 75))

	for _, item := range items {
		branch := item.Branch
		if len(branch) > 38 {
			branch = branch[:35] + "..."
		}
		stateDisplay := item.State
		if item.HasConflict {
			stateDisplay = "blocked (conflicts)"
		}
		fmt.Printf("%-15s %-40s %-10s %s\n", item.Agent, branch, item.Target, stateDisplay)
	}

	return nil
}
