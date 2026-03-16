package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

var statusActivity bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent status",
	Long: `Show the status of all bc agents.

Examples:
  bc status                   # Show all agents
  bc status --json            # Output as JSON
  bc status --activity        # Show recent channel activity

Output:
  AGENT     ROLE      STATE    UPTIME    TASK
  eng-01    engineer  working  2h 15m    Implementing feature X
  eng-02    engineer  idle     1h 30m    -

Agent States:
  working   Agent is actively processing
  idle      Agent is waiting for input
  done      Agent has completed task
  error     Agent encountered an error
  stopped   Agent is not running

See Also:
  bc agent list   List agents with more detail
  bc logs         View agent event logs
  bc home         Open TUI dashboard`,
	RunE: runStatus,
}

func init() {
	statusCmd.Flags().BoolVar(&statusActivity, "activity", false, "Show recent channel activity")
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	log.Debug("status command started")

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}
	log.Debug("workspace found", "root", ws.RootDir)

	// Create agent manager and load state
	mgr := newAgentManager(ws)
	if err = mgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}

	// Refresh state from tmux
	if err = mgr.RefreshState(); err != nil {
		log.Warn("failed to refresh agent state", "error", err)
	}

	agents := mgr.ListAgents()
	log.Debug("agents loaded", "count", len(agents))

	// Count active agents
	activeCount := 0
	workingCount := 0
	for _, a := range agents {
		if a.State != agent.StateStopped && a.State != agent.StateError {
			activeCount++
		}
		if a.State == agent.StateWorking {
			workingCount++
		}
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		// Enhanced JSON output with summary
		output := struct { //nolint:govet // fieldalignment: inline struct for JSON, alignment not critical
			Workspace string         `json:"workspace"`
			Total     int            `json:"total"`
			Active    int            `json:"active"`
			Working   int            `json:"working"`
			Agents    []*agent.Agent `json:"agents"`
		}{
			Workspace: filepath.Base(ws.RootDir),
			Agents:    agents,
			Total:     len(agents),
			Active:    activeCount,
			Working:   workingCount,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Summary header
	wsName := filepath.Base(ws.RootDir)
	fmt.Printf("Workspace: %s | Agents: %d | Active: %d | Working: %d\n", wsName, len(agents), activeCount, workingCount)
	fmt.Println(strings.Repeat("─", 60))
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
	fixedWidth := 57
	taskWidth := termWidth - fixedWidth
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

		task := normalizeTask(a.Task)
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

	// Show recent channel activity if requested
	if statusActivity {
		fmt.Println()
		fmt.Println(strings.Repeat("─", 60))
		fmt.Println("Recent Activity:")
		fmt.Println()

		store, storeErr := channel.OpenStore(ws.RootDir)
		if storeErr == nil {
			defer func() { _ = store.Close() }()
			if loadErr := store.Load(); loadErr == nil {
				channels := store.List()
				messageCount := 0
				for _, ch := range channels {
					history, histErr := store.GetHistory(ch.Name)
					if histErr != nil {
						continue
					}
					// Show last 3 messages per channel
					start := 0
					if len(history) > 3 {
						start = len(history) - 3
					}
					for _, entry := range history[start:] {
						age := time.Since(entry.Time)
						ageStr := formatDuration(age) + " ago"
						msgPreview := truncateActivityMsg(entry.Message, 50)
						fmt.Printf("  [#%s] %s: %s (%s)\n", ch.Name, entry.Sender, msgPreview, ageStr)
						messageCount++
					}
				}
				if messageCount == 0 {
					fmt.Println("  No recent messages")
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc attach <agent>  # Attach to agent's session")
	fmt.Println("  bc agent health    # Check agent health status")
	fmt.Println("  bc down            # Stop all agents")

	return nil
}

// truncateActivityMsg truncates a message to maxLen, removing newlines
func truncateActivityMsg(s string, maxLen int) string {
	// Replace newlines with spaces
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	// Collapse multiple spaces
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
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

// normalizeTask transforms cooking metaphors in Claude Code's status line
// to clearer terminology. Issue #970.
func normalizeTask(task string) string {
	// Map cooking metaphors to clear terms
	replacements := []struct {
		old, new string
	}{
		{"Sautéed", "Working"},
		{"Sauteed", "Working"}, // ASCII fallback
		{"Cooked", "Processed"},
		{"Cogitated", "Thinking"},
		{"Marinated", "Idle"},
		{"Frolicking", "Active"},
	}
	for _, r := range replacements {
		if strings.Contains(task, r.old) {
			return strings.Replace(task, r.old, r.new, 1)
		}
	}
	return task
}

func colorState(s agent.State) string {
	padded := fmt.Sprintf("%-10s", s)

	switch s {
	case agent.StateIdle:
		return ui.CyanText(padded)
	case agent.StateWorking:
		return ui.GreenText(padded)
	case agent.StateDone:
		return ui.GreenText(padded)
	case agent.StateStuck:
		return ui.RedText(padded)
	case agent.StateError:
		return ui.RedText(padded)
	case agent.StateStopped:
		return ui.YellowText(padded)
	default:
		return padded
	}
}
