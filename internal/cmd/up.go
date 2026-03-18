package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system via the bcd daemon.

Starts the root agent through the running bcd daemon.

Examples:
  bc up                      # Start root agent
  bc up --agent cursor       # Use Cursor AI for agents
  bc up --runtime docker     # Use Docker runtime`,
	RunE: runUp,
}

var (
	upAgent   string
	upRuntime string
)

func init() {
	upCmd.Flags().StringVar(&upAgent, "agent", "", "Agent type from config (e.g. claude, cursor, cursor-agent, codex)")
	upCmd.Flags().StringVar(&upRuntime, "runtime", "", "Runtime backend override: tmux or docker")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	log.Debug("up command started", "agent", upAgent, "runtime", upRuntime)

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	ws, wsErr := getWorkspace()
	if wsErr != nil {
		return errNotInWorkspace(wsErr)
	}

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)
	fmt.Print("Starting root... ")

	result, upErr := c.Workspaces.Up(cmd.Context(), upAgent, upRuntime)
	if upErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start: %w", upErr)
	}

	status, _ := result["status"].(string)
	if status == "already_running" {
		fmt.Println("already running")
		fmt.Println()
		fmt.Println("Root agent is already running.")
		fmt.Println("Use 'bc agent attach root' to attach or 'bc down' first to restart.")
		return nil
	}

	fmt.Println("✓")
	fmt.Println()
	fmt.Println("Root agent started!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc status             # View agent status")
	fmt.Println("  bc agent attach root  # Attach to root session")
	fmt.Println("  bc down               # Stop all agents")

	return nil
}
