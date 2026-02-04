package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show workspace dashboard with stats",
	Long: `Show a dashboard with workspace stats including agent status,
queue metrics, and recent activity.

Example:
  bc dashboard          # Show dashboard
  bc dashboard --json   # Output as JSON`,
	RunE: runDashboard,
}

func init() {
	rootCmd.AddCommand(dashboardCmd)
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Load agents
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	mgr.LoadState()
	mgr.RefreshState()
	agents := mgr.ListAgents()

	// Load queue
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	q.Load()
	qs := q.Stats()

	// Load recent events
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	recentEvents, _ := log.ReadLast(10)

	// JSON output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return printJSONDashboard(ws.RootDir, ws.Config.Name, agents, qs, recentEvents)
	}

	// Header
	fmt.Printf("bc dashboard: %s\n", ws.Config.Name)
	fmt.Printf("  workspace: %s\n", ws.RootDir)
	fmt.Println()

	// Agents section
	printAgentSummary(agents)

	// Queue section
	printQueueStats(qs)

	// Recent activity
	printRecentActivity(recentEvents)

	// Hints
	fmt.Println("Commands:")
	fmt.Println("  bc status     # Detailed agent status")
	fmt.Println("  bc queue      # View work queue")
	fmt.Println("  bc home       # Open TUI dashboard")

	return nil
}

func printAgentSummary(agents []*agent.Agent) {
	fmt.Println("Agents")
	fmt.Println(strings.Repeat("-", 40))

	if len(agents) == 0 {
		fmt.Println("  No agents configured (run 'bc up')")
		fmt.Println()
		return
	}

	// Count by state
	stateCounts := make(map[agent.State]int)
	for _, a := range agents {
		stateCounts[a.State]++
	}

	total := len(agents)
	running := 0
	for _, a := range agents {
		if a.State != agent.StateStopped {
			running++
		}
	}

	fmt.Printf("  Total: %d  Running: %d  Stopped: %d\n", total, running, total-running)

	// Show states that have counts > 0 (excluding stopped, already shown)
	for _, s := range []agent.State{agent.StateIdle, agent.StateWorking, agent.StateStarting, agent.StateDone, agent.StateStuck, agent.StateError} {
		if c := stateCounts[s]; c > 0 {
			fmt.Printf("  %s %s: %d\n", stateIcon(s), s, c)
		}
	}

	fmt.Println()

	// Agent list
	for _, a := range agents {
		task := a.Task
		if task == "" {
			task = "-"
		}
		if len(task) > 30 {
			task = task[:27] + "..."
		}
		uptime := "-"
		if a.State != agent.StateStopped {
			uptime = formatDuration(time.Since(a.StartedAt))
		}
		fmt.Printf("  %-14s %-11s %s  %s  %s\n", a.Name, a.Role, colorState(a.State), uptime, task)
	}
	fmt.Println()
}

func printQueueStats(qs queue.Stats) {
	fmt.Println("Queue")
	fmt.Println(strings.Repeat("-", 40))

	if qs.Total == 0 {
		fmt.Println("  No work items (run 'bc queue add')")
		fmt.Println()
		return
	}

	fmt.Printf("  Total: %d\n", qs.Total)
	fmt.Printf("  Pending:  %-4d  Assigned: %d\n", qs.Pending, qs.Assigned)
	fmt.Printf("  Working:  %-4d  Done:     %d\n", qs.Working, qs.Done)
	if qs.Failed > 0 {
		fmt.Printf("  Failed:   %d\n", qs.Failed)
	}

	// Progress bar
	if qs.Total > 0 {
		completed := qs.Done
		pct := (completed * 100) / qs.Total
		barWidth := 20
		filled := (completed * barWidth) / qs.Total
		bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
		fmt.Printf("  Progress: [%s] %d%%\n", bar, pct)
	}
	fmt.Println()
}

func printRecentActivity(evts []events.Event) {
	fmt.Println("Recent Activity")
	fmt.Println(strings.Repeat("-", 40))

	if len(evts) == 0 {
		fmt.Println("  No events yet")
		fmt.Println()
		return
	}

	for _, ev := range evts {
		age := formatDuration(time.Since(ev.Timestamp))
		agentStr := ""
		if ev.Agent != "" {
			agentStr = fmt.Sprintf("[%s] ", ev.Agent)
		}
		msg := ev.Message
		if msg == "" {
			msg = string(ev.Type)
		}
		fmt.Printf("  %8s ago  %s%s\n", age, agentStr, msg)
	}
	fmt.Println()
}

func stateIcon(s agent.State) string {
	switch s {
	case agent.StateIdle:
		return "o"
	case agent.StateWorking:
		return ">"
	case agent.StateDone:
		return "+"
	case agent.StateStuck:
		return "!"
	case agent.StateError:
		return "x"
	case agent.StateStarting:
		return "~"
	case agent.StateStopped:
		return "-"
	default:
		return "?"
	}
}

// dashboardOutput is the JSON structure for dashboard output.
type dashboardOutput struct {
	Workspace string            `json:"workspace"`
	Name      string            `json:"name"`
	Agents    dashboardAgents   `json:"agents"`
	Queue     queue.Stats       `json:"queue"`
	Events    []dashboardEvent  `json:"recent_events"`
}

type dashboardAgents struct {
	Total   int            `json:"total"`
	Running int            `json:"running"`
	List    []agentOutput  `json:"list"`
}

type dashboardEvent struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Agent     string `json:"agent,omitempty"`
	Message   string `json:"message,omitempty"`
}

func printJSONDashboard(workspace, name string, agents []*agent.Agent, qs queue.Stats, evts []events.Event) error {
	running := 0
	for _, a := range agents {
		if a.State != agent.StateStopped {
			running++
		}
	}

	agentList := make([]agentOutput, 0, len(agents))
	for _, a := range agents {
		uptime := ""
		if a.State != agent.StateStopped {
			uptime = formatDuration(time.Since(a.StartedAt))
		}
		agentList = append(agentList, agentOutput{
			Name:      a.Name,
			Role:      string(a.Role),
			State:     string(a.State),
			Task:      a.Task,
			Uptime:    uptime,
			StartedAt: a.StartedAt.Format(time.RFC3339),
			Session:   a.Session,
		})
	}

	eventList := make([]dashboardEvent, 0, len(evts))
	for _, ev := range evts {
		eventList = append(eventList, dashboardEvent{
			Timestamp: ev.Timestamp.Format(time.RFC3339),
			Type:      string(ev.Type),
			Agent:     ev.Agent,
			Message:   ev.Message,
		})
	}

	out := dashboardOutput{
		Workspace: workspace,
		Name:      name,
		Agents: dashboardAgents{
			Total:   len(agents),
			Running: running,
			List:    agentList,
		},
		Queue:  qs,
		Events: eventList,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
