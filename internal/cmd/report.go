package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
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

	// Update agent state
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	mgr.LoadState()
	if err := mgr.UpdateAgentState(agentID, state, message); err != nil {
		return fmt.Errorf("failed to update agent state: %w", err)
	}

	// Update queue items based on state transition
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	q.Load()

	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Find work items assigned to this agent
	agentItems := q.ListByAgent(agentID)
	for _, item := range agentItems {
		switch state {
		case agent.StateWorking:
			if item.Status == queue.StatusAssigned {
				q.UpdateStatus(item.ID, queue.StatusWorking)
				log.Append(events.Event{
					Type:    events.WorkStarted,
					Agent:   agentID,
					Message: message,
					Data:    map[string]any{"work_id": item.ID},
				})
			}
		case agent.StateDone:
			if item.Status == queue.StatusWorking || item.Status == queue.StatusAssigned {
				q.UpdateStatus(item.ID, queue.StatusDone)
				log.Append(events.Event{
					Type:    events.WorkCompleted,
					Agent:   agentID,
					Message: message,
					Data:    map[string]any{"work_id": item.ID},
				})
			}
		}
	}
	q.Save()

	// Log the report event
	log.Append(events.Event{
		Type:    events.AgentReport,
		Agent:   agentID,
		Message: fmt.Sprintf("%s: %s", state, message),
	})

	fmt.Printf("Reported: %s %s\n", state, message)
	return nil
}
