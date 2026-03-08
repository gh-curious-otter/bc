package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	bclog "github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/memory"
)

// Flags for report command (enhanced for stuck reports - #675)
var (
	reportReason       string
	reportReproduction string
	reportSeverity     string
)

var reportCmd = &cobra.Command{
	Use:   "report <state> [message]",
	Short: "Report agent state (called by agents)",
	Long: `Report the calling agent's current state. This command must be run from within an agent session.

Valid states: idle, working, done, stuck, error

For stuck state, use --reason to provide detailed context:
  bc report stuck --reason "database connection timeout"
  bc report stuck --reason "auth fails" --reproduction "login with test user" --severity critical

Examples:
  bc report working "fixing auth bug"
  bc report done "auth bug fixed"
  bc report stuck "need database credentials"
  bc report stuck --reason "TUI freezes on channel select" --severity high`,
	Args: cobra.MinimumNArgs(1),
	RunE: runReport,
}

func init() {
	rootCmd.AddCommand(reportCmd)

	// Enhanced flags for stuck reports (#675)
	reportCmd.Flags().StringVar(&reportReason, "reason", "", "Detailed reason for stuck state")
	reportCmd.Flags().StringVar(&reportReproduction, "reproduction", "", "Steps to reproduce the issue")
	reportCmd.Flags().StringVar(&reportSeverity, "severity", "medium", "Issue severity (critical, high, medium, low)")
}

func runReport(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return errorAgentNotRunning(fmt.Sprintf("bc report %s", args[0]))
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

	// Auto-record experience when agent reports done
	if state == agent.StateDone && message != "" {
		if err := recordExperience(ws.RootDir, agentID, message, "success"); err != nil {
			bclog.Warn("failed to auto-record experience", "error", err)
		}
	}

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

