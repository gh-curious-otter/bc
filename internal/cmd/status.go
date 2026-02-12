package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/log"
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
	log.Debug("status command started")

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}
	log.Debug("workspace found", "root", ws.RootDir)

	// Create agent manager and load state
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err = mgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}

	// Refresh state from tmux
	if err = mgr.RefreshState(); err != nil {
		log.Warn("failed to refresh agent state", "error", err)
	}

	agents := mgr.ListAgents()
	log.Debug("agents loaded", "count", len(agents))

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agents)
	}

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
	fmt.Println("Symbols (in TASK column):")
	fmt.Println("  ✻ ✳ ✽ ·  Thinking (agent is processing)")
	fmt.Println("  ⏺        Tool call (agent is running a tool)")
	fmt.Println("  ❯        Prompt (agent is waiting for input)")
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
