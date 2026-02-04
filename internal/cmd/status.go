package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent status",
	Long: `Show the status of all bc agents.

Example:
  bc status          # Show all agents
  bc status --json   # Output as JSON`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}
	
	// Create agent manager and load state
	mgr := agent.NewManager(ws.AgentsDir())
	mgr.LoadState()
	
	// Refresh state from tmux
	mgr.RefreshState()
	
	agents := mgr.ListAgents()
	
	fmt.Printf("bc workspace: %s\n", ws.RootDir)
	fmt.Println()
	
	if len(agents) == 0 {
		fmt.Println("No agents configured")
		fmt.Println()
		fmt.Println("Run 'bc up' to start agents")
		return nil
	}
	
	// Print header
	fmt.Printf("%-15s %-12s %-10s %-20s %s\n", "AGENT", "ROLE", "STATE", "UPTIME", "TASK")
	fmt.Println(strings.Repeat("-", 75))
	
	// Print agents
	for _, a := range agents {
		uptime := "-"
		if a.State != agent.StateStopped {
			uptime = formatDuration(time.Since(a.StartedAt))
		}
		
		task := a.Task
		if task == "" {
			task = "-"
		}
		if len(task) > 30 {
			task = task[:27] + "..."
		}
		
		stateStr := formatState(a.State)
		
		fmt.Printf("%-15s %-12s %-10s %-20s %s\n",
			a.Name,
			a.Role,
			stateStr,
			uptime,
			task,
		)
	}
	
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc attach <agent>  # Attach to agent's session")
	fmt.Println("  bc down            # Stop all agents")
	
	return nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatState(s agent.State) string {
	switch s {
	case agent.StateIdle:
		return "idle"
	case agent.StateWorking:
		return "working"
	case agent.StateDone:
		return "done"
	case agent.StateStuck:
		return "STUCK"
	case agent.StateError:
		return "ERROR"
	case agent.StateStopped:
		return "stopped"
	default:
		return string(s)
	}
}
