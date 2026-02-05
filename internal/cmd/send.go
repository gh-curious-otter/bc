package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
)

var sendCmd = &cobra.Command{
	Use:   "send <agent> <message>",
	Short: "Send a message to an agent",
	Long: `Send a message or command to an agent's tmux session.

The message is typed into the agent's session, then submitted using carriage
return which works for both Claude and Cursor agents.

Examples:
  bc send coordinator "build the auth module"
  bc send worker-01 "run the tests"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runSend,
}

func init() {
	rootCmd.AddCommand(sendCmd)
}

func runSend(cmd *cobra.Command, args []string) error {
	agentName := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Create workspace-scoped agent manager
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err := mgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}

	// Check agent exists
	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	if a.State == agent.StateStopped {
		return fmt.Errorf("agent '%s' is stopped", agentName)
	}

	// Send message with auto-submit (uses \r which works for Claude and Cursor agents)
	if err := mgr.SendToAgent(agentName, message); err != nil {
		return fmt.Errorf("failed to send to %s: %w", agentName, err)
	}

	// Log event
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	_ = log.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   agentName,
		Message: message,
	})

	fmt.Printf("Sent to %s: %s\n", agentName, message)
	return nil
}
