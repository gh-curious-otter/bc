package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/tui/runtime"
	"github.com/spf13/cobra"
)

var homeCmd = &cobra.Command{
	Use:   "home",
	Short: "Open the bc home screen TUI",
	Long: `Open the interactive home screen showing agent status.

The TUI updates in real-time as agents start, stop, and report progress.
You can attach to agents, send commands, and monitor the system.

Navigation:
  j/k      Move up/down
  Enter    Attach to selected agent
  p        Peek at agent output
  r        Refresh status
  q        Quit`,
	RunE: runHome,
}

func init() {
	rootCmd.AddCommand(homeCmd)
}

func runHome(cmd *cobra.Command, args []string) error {
	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w\nRun 'bc init' first", err)
	}

	// Create agent manager
	mgr := agent.NewManager(ws.AgentsDir())
	mgr.LoadState()

	// Create pipes for TUI communication
	aiToTUI, tuiInput, _ := os.Pipe()
	tuiOutput, tuiToAI, _ := os.Pipe()

	// Start the driver with our pipes
	driver := runtime.NewDriver().
		WithTitle("bc - " + ws.Config.Name).
		WithIO(aiToTUI, tuiOutput)

	// Start home controller in background
	go runHomeController(tuiInput, tuiToAI, mgr, ws.RootDir)

	return driver.Run()
}

func runHomeController(toTUI *os.File, fromTUI *os.File, mgr *agent.Manager, rootDir string) {
	send := func(v any) {
		data, _ := json.Marshal(v)
		fmt.Fprintln(toTUI, string(data))
	}

	// Wait for ready
	scanner := bufio.NewScanner(fromTUI)
	for scanner.Scan() {
		var msg runtime.Message
		if json.Unmarshal(scanner.Bytes(), &msg) == nil {
			if msg.Type == runtime.MsgReady {
				break
			}
		}
	}

	// Initial delay
	time.Sleep(100 * time.Millisecond)

	// Show home view
	refreshAgentTable(send, mgr)

	// Handle events
	for scanner.Scan() {
		var keyEvent runtime.KeyEvent
		if err := json.Unmarshal(scanner.Bytes(), &keyEvent); err != nil {
			continue
		}

		if keyEvent.Type != runtime.MsgKey {
			continue
		}

		switch keyEvent.Key {
		case "r":
			// Refresh
			mgr.RefreshState()
			refreshAgentTable(send, mgr)

		case "enter":
			// Would attach to agent - but that requires exiting TUI
			// For now, show detail view
			if keyEvent.Selected != nil && len(keyEvent.Selected.Values) > 0 {
				showAgentDetailView(send, mgr, keyEvent.Selected.Values[0])
			}

		case "esc":
			// Back to main view
			refreshAgentTable(send, mgr)

		case "p":
			// Peek at agent output
			if keyEvent.Selected != nil && len(keyEvent.Selected.Values) > 0 {
				showAgentPeek(send, mgr, keyEvent.Selected.Values[0])
			}
		}
	}
}

func refreshAgentTable(send func(any), mgr *agent.Manager) {
	// Create table view
	send(runtime.ViewMessage{
		Type:    runtime.MsgView,
		View:    runtime.ViewTable,
		ID:      "agents",
		Title:   "Agents",
		Loading: true,
	})

	// Set columns
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "columns",
		Value: []runtime.ColumnSpec{
			{Name: "AGENT", Width: 15},
			{Name: "ROLE", Width: 12},
			{Name: "STATE", Width: 10},
			{Name: "UPTIME", Width: 12},
			{Name: "TASK", Width: 30},
		},
	})

	// Refresh state
	mgr.RefreshState()
	agents := mgr.ListAgents()

	// Stream rows
	for _, a := range agents {
		uptime := "-"
		if a.State != agent.StateStopped {
			uptime = formatDuration(time.Since(a.StartedAt))
		}

		task := a.Task
		if task == "" {
			task = "-"
		}

		status := mapAgentState(a.State)

		send(runtime.AppendMessage{
			Type: runtime.MsgAppend,
			Path: "rows",
			Value: runtime.RowSpec{
				ID:     a.ID,
				Values: []string{a.Name, string(a.Role), string(a.State), uptime, task},
				Status: status,
			},
		})
		time.Sleep(50 * time.Millisecond)
	}

	// If no agents, show empty state
	if len(agents) == 0 {
		send(runtime.SetMessage{
			Type:  runtime.MsgSet,
			Path:  "empty",
			Value: "No agents running. Use 'bc up' to start agents.",
		})
	}

	// Set bindings
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "bindings",
		Value: []runtime.BindingSpec{
			{Key: "enter", Label: "Details", Action: "select"},
			{Key: "p", Label: "Peek", Action: "peek"},
			{Key: "r", Label: "Refresh", Action: "refresh"},
		},
	})

	send(runtime.DoneMessage{Type: runtime.MsgDone})
}

func showAgentDetailView(send func(any), mgr *agent.Manager, name string) {
	a := mgr.GetAgent(name)
	if a == nil {
		return
	}

	send(runtime.ViewMessage{
		Type:    runtime.MsgView,
		View:    runtime.ViewDetail,
		ID:      "agent-detail",
		Title:   a.Name,
		Loading: true,
	})

	// Set sections
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "sections",
		Value: []runtime.SectionSpec{
			{Title: "Agent Info"},
		},
	})

	// Stream fields
	fields := []runtime.FieldSpec{
		{Label: "Name", Value: a.Name},
		{Label: "Role", Value: string(a.Role)},
		{Label: "State", Value: string(a.State), Style: mapAgentState(a.State)},
		{Label: "Session", Value: "bc-" + a.Session, Style: "code"},
		{Label: "Workspace", Value: a.Workspace},
		{Label: "Started", Value: a.StartedAt.Format(time.RFC3339)},
	}

	if a.Task != "" {
		fields = append(fields, runtime.FieldSpec{Label: "Task", Value: a.Task})
	}

	for _, f := range fields {
		send(runtime.AppendMessage{
			Type:  runtime.MsgAppend,
			Path:  "sections[0].fields",
			Value: f,
		})
		time.Sleep(50 * time.Millisecond)
	}

	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "bindings",
		Value: []runtime.BindingSpec{
			{Key: "esc", Label: "Back", Action: "back"},
			{Key: "p", Label: "Peek", Action: "peek"},
		},
	})

	send(runtime.DoneMessage{Type: runtime.MsgDone})
}

func showAgentPeek(send func(any), mgr *agent.Manager, name string) {
	output, err := mgr.CaptureOutput(name, 30)
	if err != nil {
		output = "Error: " + err.Error()
	}

	send(runtime.ViewMessage{
		Type:  runtime.MsgView,
		View:  runtime.ViewDetail,
		ID:    "agent-peek",
		Title: name + " - Output",
	})

	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "sections",
		Value: []runtime.SectionSpec{
			{
				Title: "Recent Output",
				Fields: []runtime.FieldSpec{
					{Label: "", Value: output, Style: "code"},
				},
			},
		},
	})

	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "bindings",
		Value: []runtime.BindingSpec{
			{Key: "esc", Label: "Back", Action: "back"},
			{Key: "r", Label: "Refresh", Action: "refresh"},
		},
	})

	send(runtime.DoneMessage{Type: runtime.MsgDone})
}

func mapAgentState(s agent.State) string {
	switch s {
	case agent.StateIdle:
		return "info"
	case agent.StateWorking:
		return "ok"
	case agent.StateDone:
		return "ok"
	case agent.StateStuck:
		return "warning"
	case agent.StateError:
		return "error"
	case agent.StateStopped:
		return "muted"
	default:
		return ""
	}
}
