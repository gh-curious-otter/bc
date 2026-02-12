package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/log"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop bc agents",
	Long: `Stop all running bc agents.

This will gracefully stop all agent tmux sessions.

Example:
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

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	fmt.Printf("Stopping bc agents in %s\n\n", ws.RootDir)

	// Create agent manager and load state
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err := mgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}

	agents := mgr.ListAgents()
	log.Debug("agents to stop", "count", len(agents))
	if len(agents) == 0 {
		fmt.Println("No agents running")
		return nil
	}

	// Stop all agents
	for _, a := range agents {
		log.Debug("stopping agent", "name", a.Name, "state", a.State)
		fmt.Printf("Stopping %s... ", a.Name)
		if err := mgr.StopAgent(a.Name); err != nil {
			log.Debug("agent stop failed", "name", a.Name, "error", err)
			fmt.Println("✗")
			fmt.Printf("  Warning: %v\n", err)
		} else {
			log.Debug("agent stopped", "name", a.Name)
			fmt.Println("✓")
		}
	}

	fmt.Println()
	fmt.Println("All agents stopped")

	return nil
}
