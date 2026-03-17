package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	bclog "github.com/rpuneet/bc/pkg/log"
)

// Flags for report command (enhanced for stuck reports - #675)
var (
	reportReason       string
	reportReproduction string
	reportSeverity     string
)

var agentReportCmd = &cobra.Command{
	Use:   "report <state> [message]",
	Short: "Report agent state (called by agents)",
	Long: `Report the calling agent's current state. This command must be run from within an agent session.

Valid states: idle, working, done, stuck, error

For stuck state, use --reason to provide detailed context:
  bc agent report stuck --reason "database connection timeout"
  bc agent report stuck --reason "auth fails" --reproduction "login with test user" --severity critical

Examples:
  bc agent report working "fixing auth bug"
  bc agent report done "auth bug fixed"
  bc agent report stuck "need database credentials"
  bc agent report stuck --reason "TUI freezes on channel select" --severity high`,
	Args: cobra.MinimumNArgs(1),
	RunE: runReport,
}

func init() {
	agentCmd.AddCommand(agentReportCmd)

	// Enhanced flags for stuck reports (#675)
	agentReportCmd.Flags().StringVar(&reportReason, "reason", "", "Detailed reason for stuck state")
	agentReportCmd.Flags().StringVar(&reportReproduction, "reproduction", "", "Steps to reproduce the issue")
	agentReportCmd.Flags().StringVar(&reportSeverity, "severity", "medium", "Issue severity (critical, high, medium, low)")
}

func runReport(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return errorAgentNotRunning(fmt.Sprintf("bc agent report %s", args[0]))
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
		return errNotInWorkspace(err)
	}

	// Update agent state
	mgr := newAgentManager(ws)
	if err := mgr.LoadState(); err != nil {
		bclog.Warn("failed to load agent state", "error", err)
	}
	if err := mgr.UpdateAgentState(agentID, state, message); err != nil {
		return fmt.Errorf("failed to update agent state: %w", err)
	}

	// Build event data
	eventData := make(map[string]any)
	eventMsg := fmt.Sprintf("%s: %s", state, message)

	// Enhanced stuck reporting (#675)
	if state == agent.StateStuck {
		if reportReason != "" {
			eventData["reason"] = reportReason
			eventMsg = fmt.Sprintf("%s: %s", state, reportReason)
		}
		if reportReproduction != "" {
			eventData["reproduction"] = reportReproduction
		}
		if reportSeverity != "" {
			eventData["severity"] = reportSeverity
		}
		eventData["stuck"] = true
	}

	// Log the report event
	event := events.Event{
		Type:    events.AgentReport,
		Agent:   agentID,
		Message: eventMsg,
	}
	if len(eventData) > 0 {
		event.Data = eventData
	}
	logEvent(ws, event)

	// Output message
	if state == agent.StateStuck && reportReason != "" {
		fmt.Printf("Reported: %s [%s]\n", state, reportSeverity)
		fmt.Printf("  Reason: %s\n", reportReason)
		if reportReproduction != "" {
			fmt.Printf("  Reproduction: %s\n", reportReproduction)
		}
	} else {
		fmt.Printf("Reported: %s %s\n", state, message)
	}
	return nil
}
