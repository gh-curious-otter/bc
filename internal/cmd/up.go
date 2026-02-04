package cmd

import (
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system.

This will:
1. Start the coordinator agent
2. Start worker agents (based on max-workers setting)
3. All agents run Claude with full permissions in tmux sessions

Example:
  bc up              # Start with default settings
  bc up --workers 5  # Start with 5 workers`,
	RunE: runUp,
}

var upWorkers int

func init() {
	upCmd.Flags().IntVar(&upWorkers, "workers", 0, "Number of workers (0 = use config default)")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w\nRun 'bc init' first", err)
	}
	
	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)
	
	// Create agent manager (agents run with full permissions by default)
	mgr := agent.NewManager(ws.AgentsDir())
	
	// Use custom agent command if configured
	if ws.Config.AgentCommand != "" {
		mgr.SetAgentCommand(ws.Config.AgentCommand)
	}
	
	// Determine worker count
	numWorkers := ws.Config.MaxWorkers
	if upWorkers > 0 {
		numWorkers = upWorkers
	}
	
	// Start coordinator
	fmt.Print("Starting coordinator... ")
	coord, err := mgr.SpawnAgent("coordinator", agent.RoleCoordinator, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start coordinator: %w", err)
	}
	fmt.Printf("✓ (session: bc-%s)\n", coord.Session)
	
	// Give coordinator time to initialize
	time.Sleep(500 * time.Millisecond)
	
	// Start workers
	for i := 1; i <= numWorkers; i++ {
		name := fmt.Sprintf("worker-%02d", i)
		fmt.Printf("Starting %s... ", name)
		
		worker, err := mgr.SpawnAgent(name, agent.RoleWorker, ws.RootDir)
		if err != nil {
			fmt.Println("✗")
			fmt.Printf("  Warning: failed to start %s: %v\n", name, err)
			continue
		}
		fmt.Printf("✓ (session: bc-%s)\n", worker.Session)
		
		// Small delay between spawns
		time.Sleep(300 * time.Millisecond)
	}
	
	fmt.Println()
	fmt.Println("All agents started!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc status          # View agent status")
	fmt.Println("  bc attach <agent>  # Attach to an agent's session")
	fmt.Println("  bc down            # Stop all agents")
	
	return nil
}
