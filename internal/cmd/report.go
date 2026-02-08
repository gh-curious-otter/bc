package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	bclog "github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/memory"
	"github.com/rpuneet/bc/pkg/workspace"
)

var reportCmd = &cobra.Command{
	Use:   "report <state> [message]",
	Short: "Report agent state (called by agents)",
	Long: `Report the calling agent's current state. Uses BC_AGENT_ID env var.

Valid states: idle, working, done, stuck, error

Example:
  bc report working "fixing auth bug"
  bc report done "auth bug fixed"
  bc report stuck "need database credentials"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)
}

func runReport(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("BC_AGENT_ID not set (this command is meant to be called by agents)")
	}

	stateStr := args[0]
	message := ""
	if len(args) > 1 {
		message = strings.Join(args[1:], " ")
	}

	// Validate state
	state := agent.State(stateStr)
	switch state {
	case agent.StateIdle, agent.StateWorking, agent.StateDone, agent.StateStuck, agent.StateError:
		// valid
	default:
		return fmt.Errorf("invalid state: %s (valid: idle, working, done, stuck, error)", stateStr)
	}

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Worktree validation: warn if agent is outside its assigned worktree
	checkWorktreeWarning(agentID, ws)

	// Update agent state
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err := mgr.LoadState(); err != nil {
		bclog.Warn("failed to load agent state", "error", err)
	}
	if err := mgr.UpdateAgentState(agentID, state, message); err != nil {
		return fmt.Errorf("failed to update agent state: %w", err)
	}

	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Log the report event
	if err := log.Append(events.Event{
		Type:    events.AgentReport,
		Agent:   agentID,
		Message: fmt.Sprintf("%s: %s", state, message),
	}); err != nil {
		bclog.Warn("failed to append agent report event", "error", err)
	}

	// Auto-record experience when agent reports done
	if state == agent.StateDone && message != "" {
		if err := recordExperience(ws.RootDir, agentID, message, "success"); err != nil {
			bclog.Warn("failed to auto-record experience", "error", err)
		}
	}

	fmt.Printf("Reported: %s %s\n", state, message)
	return nil
}

// recordExperience records a task completion experience to the agent's memory.
// Deduplicates by checking if the exact description already exists in recent experiences.
func recordExperience(rootDir, agentID, description, outcome string) error {
	store := memory.NewStore(rootDir, agentID)

	// Initialize memory if it doesn't exist
	if !store.Exists() {
		if err := store.Init(); err != nil {
			return fmt.Errorf("failed to initialize memory: %w", err)
		}
	}

	// Check for duplicates in recent experiences
	experiences, err := store.GetExperiences()
	if err != nil {
		return fmt.Errorf("failed to get experiences: %w", err)
	}

	// Check last 5 experiences for duplicates
	checkCount := 5
	if len(experiences) < checkCount {
		checkCount = len(experiences)
	}
	for i := len(experiences) - checkCount; i < len(experiences); i++ {
		if experiences[i].Description == description {
			bclog.Debug("skipping duplicate experience", "description", description)
			return nil // Already recorded
		}
	}

	// Record the experience
	exp := memory.Experience{
		Description: description,
		Outcome:     outcome,
		TaskType:    "task", // Default task type for auto-recorded experiences
	}

	if err := store.RecordExperience(exp); err != nil {
		return fmt.Errorf("failed to record experience: %w", err)
	}

	bclog.Debug("auto-recorded experience", "agent", agentID, "description", description)
	return nil
}

// checkWorktreeWarning warns (to stderr + event log) if the agent is outside its worktree.
// Never blocks — the report always proceeds.
func checkWorktreeWarning(agentID string, ws *workspace.Workspace) {
	worktree := os.Getenv("BC_AGENT_WORKTREE")
	if worktree == "" {
		return // Not set (pre-Phase A agent, or test environment)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return // Can't determine cwd, skip check
	}

	// Resolve symlinks for accurate comparison
	worktreeAbs, err := filepath.EvalSymlinks(worktree)
	if err != nil {
		return // Worktree doesn't exist, skip (will be caught by bc worktree check)
	}
	cwdAbs, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return
	}

	if isWithinDir(cwdAbs, worktreeAbs) {
		return // All good
	}

	// Agent is outside its worktree — warn but don't block
	fmt.Fprintf(os.Stderr, "WARNING: %s is reporting from outside its worktree (cwd: %s, expected: %s)\n",
		agentID, cwdAbs, worktreeAbs)

	// Log to events
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := log.Append(events.Event{
		Type:    events.AgentReport,
		Agent:   agentID,
		Message: fmt.Sprintf("worktree violation: cwd=%s expected=%s", cwdAbs, worktreeAbs),
		Data:    map[string]any{"violation": "worktree_mismatch", "cwd": cwdAbs, "worktree": worktreeAbs},
	}); err != nil {
		bclog.Warn("failed to log worktree violation", "error", err)
	}
}
