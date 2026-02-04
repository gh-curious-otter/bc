package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/spf13/cobra"
)

var sendSubmitKey string

var sendCmd = &cobra.Command{
	Use:   "send <agent> <message>",
	Short: "Send a message to an agent",
	Long: `Send a message or command to an agent's tmux session.

The message is typed into the agent's session, then a key is sent so the agent
receives it as submitted. When the workspace uses cursor-agent, we send Ctrl+Enter
(C-Enter) by default because Cursor Agent treats Enter as newline only. If
messages still do not submit, use --submit-key=none and attach (bc attach <agent>)
to press Send manually.

Examples:
  bc send coordinator "build the auth module"
  bc send worker-01 "run the tests"
  bc send --submit-key=none worker-01 "your task"   # paste only, submit manually`,
	Args: cobra.MinimumNArgs(2),
	RunE: runSend,
}

func init() {
	rootCmd.AddCommand(sendCmd)
	sendCmd.Flags().StringVar(&sendSubmitKey, "submit-key", "Enter", "Key after message (Enter; for cursor-agent we use C-Enter; use 'none' to paste only)")
}

func runSend(cmd *cobra.Command, args []string) error {
	agentName := args[0]
	message := strings.Join(args[1:], " ")

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Create workspace-scoped agent manager
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	mgr.LoadState()

	// Check agent exists
	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	if a.State == agent.StateStopped {
		return fmt.Errorf("agent '%s' is stopped", agentName)
	}

	submitKey := sendSubmitKey
	if submitKey == "none" {
		submitKey = ""
	}
	// Cursor Agent does not submit on Enter (only newline). Use Ctrl+Enter by default
	// when the workspace uses cursor-agent, so the message may actually submit.
	if submitKey == "Enter" {
		effectiveCmd := ws.Config.AgentCommand
		if effectiveCmd == "" {
			effectiveCmd = config.Agent.Command
		}
		if strings.Contains(effectiveCmd, "cursor-agent") {
			submitKey = "C-Enter"
		}
	}

	// Send message (and optional submit key)
	if err := mgr.SendToAgentWithSubmitKey(agentName, message, submitKey); err != nil {
		return fmt.Errorf("failed to send to %s: %w", agentName, err)
	}

	// Log event
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	log.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   agentName,
		Message: message,
	})

	fmt.Printf("Sent to %s: %s\n", agentName, message)
	return nil
}
