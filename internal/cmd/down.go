package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/client"
	"github.com/rpuneet/bc/pkg/log"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop bc agents",
	Long: `Stop all running bc agents.

This will gracefully stop all agent tmux sessions.

Examples:
  bc down          # Stop all agents
  bc down --force  # Force kill without cleanup`,
	RunE: runDown,
}

var downForce bool

func init() {
	downCmd.Flags().BoolVar(&downForce, "force", false, "Force kill without cleanup")
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, args []string) error {
	log.Debug("down command started", "force", downForce)

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	c := client.New("")
	if pingErr := c.Ping(cmd.Context()); pingErr != nil {
		return fmt.Errorf("bcd is not running — start it with 'bc up' first\n(%w)", pingErr)
	}

	fmt.Printf("Stopping bc agents in %s\n\n", ws.RootDir)

	stopped, stopErr := c.Workspaces.Down(cmd.Context())
	if stopErr != nil {
		return fmt.Errorf("failed to stop agents: %w", stopErr)
	}

	if stopped == 0 {
		fmt.Println("No agents running")
		return nil
	}

	fmt.Printf("Stopped %d agent(s)\n", stopped)
	return nil
}
