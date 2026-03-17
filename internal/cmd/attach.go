package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

var attachCmd = &cobra.Command{
	Use:        "attach <agent>",
	Short:      "Attach to an agent's tmux session (deprecated: use 'bc agent attach')",
	Deprecated: "use 'bc agent attach' instead",
	Long: `Attach to an agent's tmux session to interact with it directly.

This opens the tmux session where the agent (Claude) is running.
Use Ctrl+b d to detach and return to your shell.

Examples:
  bc attach coordinator   # Attach to coordinator
  bc attach worker-01     # Attach to worker 1`,
	Args: cobra.ExactArgs(1),
	RunE: runAttach,
}

func init() {
	rootCmd.AddCommand(attachCmd)
}

func runAttach(cmd *cobra.Command, args []string) error {
	agentName := args[0]
	log.Debug("attach command started", "agent", agentName)

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Create agent manager
	ctx := cmd.Context()
	mgr := newAgentManager(ws)

	// Check if session exists
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}
	if !mgr.RuntimeForAgent(agentName).HasSession(ctx, agentName) {
		log.Debug("agent session not found", "agent", agentName)
		return fmt.Errorf("agent %q not running (session bc-%s not found)", agentName, agentName)
	}

	log.Debug("attaching to agent session", "agent", agentName)
	fmt.Printf("Attaching to %s (use Ctrl+b d to detach)...\n", agentName)

	return mgr.AttachToAgent(agentName)
}
