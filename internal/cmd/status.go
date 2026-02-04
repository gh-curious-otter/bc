package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
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
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
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

	// Determine terminal width for dynamic task column
	termWidth := 80
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		termWidth = w
	}
	// Fixed columns: AGENT(15) + ROLE(12) + STATE(10) + UPTIME(20) = 57
	taskWidth := termWidth - 57
	if taskWidth < 20 {
		taskWidth = 20
	}

	// Print header
	fmt.Printf("%-15s %-12s %-10s %-20s %s\n", "AGENT", "ROLE", "STATE", "UPTIME", "TASK")
	fmt.Println(strings.Repeat("-", termWidth))

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
		if len(task) > taskWidth {
			task = task[:taskWidth-3] + "..."
		}

		stateStr := colorState(a.State)

		fmt.Printf("%-15s %-12s %s %-20s %s\n",
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

func colorState(s agent.State) string {
	const (
		reset  = "\033[0m"
		green  = "\033[32m"
		yellow = "\033[33m"
		red    = "\033[31m"
		cyan   = "\033[36m"
	)

	padded := fmt.Sprintf("%-10s", s)

	switch s {
	case agent.StateIdle:
		return cyan + padded + reset
	case agent.StateWorking:
		return green + padded + reset
	case agent.StateDone:
		return green + padded + reset
	case agent.StateStuck:
		return red + padded + reset
	case agent.StateError:
		return red + padded + reset
	case agent.StateStopped:
		return yellow + padded + reset
	default:
		return padded
	}
}
