package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/events"
	bclog "github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/workspace"
	"github.com/spf13/cobra"
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

	// Update queue items based on state transition
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	if err := q.Load(); err != nil {
		bclog.Warn("failed to load queue", "error", err)
	}

	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Find work items assigned to this agent
	agentItems := q.ListByAgent(agentID)
itemLoop:
	for _, item := range agentItems {
		switch state {
		case agent.StateWorking:
			if item.Status == queue.StatusAssigned {
				_ = q.UpdateStatus(item.ID, queue.StatusWorking)
				if err := log.Append(events.Event{
					Type:    events.WorkStarted,
					Agent:   agentID,
					Message: message,
					Data:    map[string]any{"work_id": item.ID},
				}); err != nil {
					bclog.Warn("failed to append work started event", "error", err)
				}
			}
		case agent.StateDone:
			if item.Status == queue.StatusWorking {
				_ = q.UpdateStatus(item.ID, queue.StatusDone)
				if err := log.Append(events.Event{
					Type:    events.WorkCompleted,
					Agent:   agentID,
					Message: message,
					Data:    map[string]any{"work_id": item.ID},
				}); err != nil {
					bclog.Warn("failed to append work completed event", "error", err)
				}
				// Close linked beads issue if present
				if item.BeadsID != "" {
					if err := beads.CloseIssue(ws.RootDir, item.BeadsID); err != nil {
						// Log but don't fail - beads sync is best-effort
						if appendErr := log.Append(events.Event{
							Type:    events.AgentReport,
							Agent:   agentID,
							Message: fmt.Sprintf("warning: failed to close beads issue %s: %v", item.BeadsID, err),
						}); appendErr != nil {
							bclog.Warn("failed to append beads close warning event", "error", appendErr)
						}
					}
				}
				break itemLoop // Only complete the first working item
			}
		}
	}
	if err := q.Save(); err != nil {
		bclog.Warn("failed to save queue", "error", err)
	}

	// Log the report event
	if err := log.Append(events.Event{
		Type:    events.AgentReport,
		Agent:   agentID,
		Message: fmt.Sprintf("%s: %s", state, message),
	}); err != nil {
		bclog.Warn("failed to append agent report event", "error", err)
	}

	fmt.Printf("Reported: %s %s\n", state, message)
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
