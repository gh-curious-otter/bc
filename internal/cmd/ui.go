package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rpuneet/bc/pkg/tui/runtime"
	"github.com/spf13/cobra"
)

// uiCmd runs the streaming TUI runtime.
var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Run the streaming TUI runtime",
	Long: `Runs the TUI in runtime mode where it receives UI specs from stdin
and sends user events to stdout.

This allows an AI to dynamically control the interface:

  AI sends:    {"type": "view", "view": "table", "id": "agents", "title": "Agents"}
  AI sends:    {"type": "append", "path": "rows", "value": {"id": "1", "values": ["worker"]}}
  User sees:   Table updates in real-time
  TUI sends:   {"type": "key", "key": "enter", "view": "agents", "selected": {...}}

Use --demo to see a simulation of streaming updates.`,
	RunE: runUI,
}

var uiDemo bool

func init() {
	uiCmd.Flags().BoolVar(&uiDemo, "demo", false, "Run demo mode with simulated streaming")
	rootCmd.AddCommand(uiCmd)
}

func runUI(cmd *cobra.Command, args []string) error {
	if uiDemo {
		return runUIDemo()
	}

	// Run the streaming runtime
	driver := runtime.NewDriver().
		WithTitle("bc")

	return driver.Run()
}

// runUIDemo simulates an AI sending streaming updates.
func runUIDemo() error {
	// Create pipes for communication
	aiToTUI, tuiInput, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	tuiOutput, tuiToAI, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	// Start the driver with our pipes
	driver := runtime.NewDriver().
		WithTitle("bc - Demo").
		WithIO(aiToTUI, tuiOutput)

	// Start AI simulator in background
	go simulateAI(tuiInput, tuiToAI)

	return driver.Run()
}

func simulateAI(toTUI *os.File, fromTUI *os.File) {
	// Helper to send messages
	send := func(v any) {
		data, err := json.Marshal(v)
		if err != nil {
			return
		}
		_, _ = fmt.Fprintln(toTUI, string(data))
	}

	// Wait for ready event
	scanner := bufio.NewScanner(fromTUI)
	for scanner.Scan() {
		var msg runtime.Message
		if json.Unmarshal(scanner.Bytes(), &msg) == nil {
			if msg.Type == runtime.MsgReady {
				break
			}
		}
	}

	// Small delay to let UI initialize
	time.Sleep(100 * time.Millisecond)

	// === Simulate streaming table view ===

	// 1. Create table view (loading state)
	send(runtime.ViewMessage{
		Type:    runtime.MsgView,
		View:    runtime.ViewTable,
		ID:      "agents",
		Title:   "Active Agents",
		Loading: true,
	})
	time.Sleep(200 * time.Millisecond)

	// 2. Set columns
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "columns",
		Value: []runtime.ColumnSpec{
			{Name: "NAME", Width: 15},
			{Name: "STATUS", Width: 10},
			{Name: "RIG", Width: 12},
			{Name: "TASK", Width: 30},
		},
	})
	time.Sleep(100 * time.Millisecond)

	// 3. Stream rows one by one (simulating AI generating them)
	rows := []runtime.RowSpec{
		{ID: "1", Values: []string{"coordinator", "running", "bc", "Managing queue"}, Status: "ok"},
		{ID: "2", Values: []string{"worker-01", "working", "bc", "Building TUI"}, Status: "ok"},
		{ID: "3", Values: []string{"worker-02", "idle", "api", "-"}, Status: "info"},
		{ID: "4", Values: []string{"worker-03", "stuck", "api", "Waiting for review"}, Status: "warning"},
		{ID: "5", Values: []string{"worker-04", "error", "web", "Build failed"}, Status: "error"},
	}

	for _, row := range rows {
		send(runtime.AppendMessage{
			Type:  runtime.MsgAppend,
			Path:  "rows",
			Value: row,
		})
		time.Sleep(300 * time.Millisecond) // Simulate AI thinking time
	}

	// 4. Set key bindings
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "bindings",
		Value: []runtime.BindingSpec{
			{Key: "enter", Label: "Select", Action: "select"},
			{Key: "p", Label: "Peek", Action: "peek"},
			{Key: "n", Label: "Nudge", Action: "nudge"},
			{Key: "r", Label: "Refresh", Action: "refresh"},
		},
	})

	// 5. Done loading
	send(runtime.DoneMessage{Type: runtime.MsgDone})

	// === Handle user events ===
	for scanner.Scan() {
		var keyEvent runtime.KeyEvent
		if err := json.Unmarshal(scanner.Bytes(), &keyEvent); err != nil {
			continue
		}

		if keyEvent.Type != runtime.MsgKey {
			continue
		}

		switch keyEvent.Key {
		case "enter":
			// Show detail view for selected agent
			if keyEvent.Selected != nil {
				showAgentDetail(toTUI, send, keyEvent.Selected)
			}

		case "r":
			// Refresh - re-stream the data
			send(runtime.SetMessage{
				Type:  runtime.MsgSet,
				Path:  "loading",
				Value: true,
			})
			time.Sleep(500 * time.Millisecond)
			send(runtime.SetMessage{
				Type:  runtime.MsgSet,
				Path:  "loading",
				Value: false,
			})

		case "esc":
			// Go back to table view
			send(runtime.ViewMessage{
				Type:  runtime.MsgView,
				View:  runtime.ViewTable,
				ID:    "agents",
				Title: "Active Agents",
			})
			// Re-send columns and rows...
			send(runtime.SetMessage{
				Type: runtime.MsgSet,
				Path: "columns",
				Value: []runtime.ColumnSpec{
					{Name: "NAME", Width: 15},
					{Name: "STATUS", Width: 10},
					{Name: "RIG", Width: 12},
					{Name: "TASK", Width: 30},
				},
			})
			for _, row := range rows {
				send(runtime.AppendMessage{
					Type:  runtime.MsgAppend,
					Path:  "rows",
					Value: row,
				})
			}
			send(runtime.DoneMessage{Type: runtime.MsgDone})
		}
	}
}

func showAgentDetail(toTUI *os.File, send func(any), selected *runtime.RowRef) {
	name := "unknown"
	if len(selected.Values) > 0 {
		name = selected.Values[0]
	}

	// 1. Switch to detail view (loading)
	send(runtime.ViewMessage{
		Type:    runtime.MsgView,
		View:    runtime.ViewDetail,
		ID:      "agent-detail",
		Title:   name,
		Loading: true,
	})
	time.Sleep(200 * time.Millisecond)

	// 2. Stream fields one by one
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "sections",
		Value: []runtime.SectionSpec{
			{Title: "Agent Info"},
		},
	})
	time.Sleep(100 * time.Millisecond)

	fields := []runtime.FieldSpec{
		{Label: "Name", Value: name},
	}
	if len(selected.Values) > 1 {
		fields = append(fields, runtime.FieldSpec{
			Label: "Status",
			Value: selected.Values[1],
			Style: selected.Values[1], // Use status as style
		})
	}
	if len(selected.Values) > 2 {
		fields = append(fields, runtime.FieldSpec{Label: "Rig", Value: selected.Values[2]})
	}
	if len(selected.Values) > 3 {
		fields = append(fields, runtime.FieldSpec{Label: "Task", Value: selected.Values[3]})
	}

	// Add more fields with delays
	fields = append(fields,
		runtime.FieldSpec{Label: "Session", Value: "gt-bc-" + name, Style: "code"},
		runtime.FieldSpec{Label: "Uptime", Value: "2h 34m"},
		runtime.FieldSpec{Label: "Cost", Value: "$1.23", Style: "muted"},
	)

	for _, field := range fields {
		send(runtime.AppendMessage{
			Type:  runtime.MsgAppend,
			Path:  "sections[0].fields",
			Value: field,
		})
		time.Sleep(150 * time.Millisecond)
	}

	// 3. Set bindings
	send(runtime.SetMessage{
		Type: runtime.MsgSet,
		Path: "bindings",
		Value: []runtime.BindingSpec{
			{Key: "esc", Label: "Back", Action: "back"},
			{Key: "p", Label: "Peek", Action: "peek"},
			{Key: "n", Label: "Nudge", Action: "nudge"},
		},
	})

	// 4. Done
	send(runtime.DoneMessage{Type: runtime.MsgDone})
}
