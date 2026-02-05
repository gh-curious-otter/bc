package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
)

func TestStateIcon_AllStates(t *testing.T) {
	tests := []struct {
		state agent.State
		want  string
	}{
		{agent.StateIdle, "o"},
		{agent.StateWorking, ">"},
		{agent.StateDone, "+"},
		{agent.StateStuck, "!"},
		{agent.StateError, "x"},
		{agent.StateStarting, "~"},
		{agent.StateStopped, "-"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := stateIcon(tt.state)
			if got != tt.want {
				t.Errorf("stateIcon(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestStateIcon_Unknown(t *testing.T) {
	got := stateIcon(agent.State("nonexistent"))
	if got != "?" {
		t.Errorf("stateIcon(unknown) = %q, want %q", got, "?")
	}
}

func TestPrintAgentSummary_NoAgents(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printAgentSummary(nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "No agents configured") {
		t.Errorf("expected 'No agents configured', got: %s", output)
	}
}

func TestPrintAgentSummary_WithAgents_Dashboard(t *testing.T) {
	agents := []*agent.Agent{
		{
			Name:      "engineer-01",
			Role:      agent.RoleEngineer,
			State:     agent.StateWorking,
			Task:      "fixing auth",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			Name:  "qa-01",
			Role:  agent.RoleQA,
			State: agent.StateStopped,
		},
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printAgentSummary(agents)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "Total: 2") {
		t.Errorf("expected 'Total: 2', got: %s", output)
	}
	if !strings.Contains(output, "Running: 1") {
		t.Errorf("expected 'Running: 1', got: %s", output)
	}
	if !strings.Contains(output, "Stopped: 1") {
		t.Errorf("expected 'Stopped: 1', got: %s", output)
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("expected agent name 'engineer-01' in output")
	}
}

func TestPrintAgentSummary_TaskTruncation(t *testing.T) {
	agents := []*agent.Agent{
		{
			Name:      "worker-01",
			Role:      agent.RoleWorker,
			State:     agent.StateWorking,
			Task:      "This is a very long task description that should be truncated at 30 chars",
			StartedAt: time.Now(),
		},
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printAgentSummary(agents)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "...") {
		t.Errorf("expected truncated task with '...', got: %s", output)
	}
}

func TestPrintQueueStats_Empty_Dashboard(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printQueueStats(queue.Stats{})

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "No work items") {
		t.Errorf("expected 'No work items', got: %s", output)
	}
}

func TestPrintQueueStats_WithItems(t *testing.T) {
	qs := queue.Stats{
		Total:    10,
		Pending:  3,
		Assigned: 2,
		Working:  1,
		Done:     4,
		Failed:   0,
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printQueueStats(qs)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "Total: 10") {
		t.Errorf("expected 'Total: 10', got: %s", output)
	}
	if !strings.Contains(output, "40%") {
		t.Errorf("expected '40%%' progress, got: %s", output)
	}
}

func TestPrintQueueStats_WithFailures(t *testing.T) {
	qs := queue.Stats{
		Total:  5,
		Done:   3,
		Failed: 2,
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printQueueStats(qs)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "Failed") {
		t.Errorf("expected 'Failed' in output, got: %s", output)
	}
}

func TestPrintRecentActivity_Empty_Dashboard(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecentActivity(nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "No events yet") {
		t.Errorf("expected 'No events yet', got: %s", output)
	}
}

func TestPrintRecentActivity_WithEvents_Dashboard(t *testing.T) {
	evts := []events.Event{
		{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Type:      events.AgentSpawned,
			Agent:     "worker-01",
			Message:   "spawned",
		},
		{
			Timestamp: time.Now().Add(-2 * time.Minute),
			Type:      events.WorkStarted,
			Agent:     "worker-01",
			Message:   "started fixing auth",
		},
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecentActivity(evts)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "[worker-01]") {
		t.Errorf("expected agent name in output, got: %s", output)
	}
	if !strings.Contains(output, "spawned") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestPrintRecentActivity_NoAgent(t *testing.T) {
	evts := []events.Event{
		{
			Timestamp: time.Now(),
			Type:      events.QueueLoaded,
			Message:   "loaded 5 items",
		},
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecentActivity(evts)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "loaded 5 items") {
		t.Errorf("expected message without agent prefix, got: %s", output)
	}
	// Should NOT contain brackets since no agent
	if strings.Contains(output, "[") && !strings.Contains(output, "Recent") {
		t.Errorf("expected no agent brackets in output, got: %s", output)
	}
}

func TestPrintRecentActivity_EventTypeAsFallback(t *testing.T) {
	evts := []events.Event{
		{
			Timestamp: time.Now(),
			Type:      events.AgentStopped,
		},
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printRecentActivity(evts)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "agent.stopped") {
		t.Errorf("expected event type as fallback message, got: %s", output)
	}
}

func TestPrintJSONDashboard_Dashboard(t *testing.T) {
	agents := []*agent.Agent{
		{
			Name:      "coordinator",
			Role:      agent.RoleCoordinator,
			State:     agent.StateWorking,
			Task:      "coordinating",
			StartedAt: time.Now().Add(-30 * time.Minute),
			Session:   "coord-session",
		},
		{
			Name:    "worker-01",
			Role:    agent.RoleWorker,
			State:   agent.StateStopped,
			Session: "worker-session",
		},
	}

	qs := queue.Stats{
		Total:   5,
		Pending: 2,
		Done:    3,
	}

	evts := []events.Event{
		{
			Timestamp: time.Now(),
			Type:      events.AgentSpawned,
			Agent:     "coordinator",
			Message:   "spawned",
		},
	}

	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSONDashboard("/test/workspace", "test-project", agents, qs, evts)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	if err != nil {
		t.Fatalf("printJSONDashboard returned error: %v", err)
	}

	// Verify it's valid JSON
	var result dashboardOutput
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	if result.Workspace != "/test/workspace" {
		t.Errorf("workspace = %q, want %q", result.Workspace, "/test/workspace")
	}
	if result.Name != "test-project" {
		t.Errorf("name = %q, want %q", result.Name, "test-project")
	}
	if result.Agents.Total != 2 {
		t.Errorf("agents.total = %d, want 2", result.Agents.Total)
	}
	if result.Agents.Running != 1 {
		t.Errorf("agents.running = %d, want 1", result.Agents.Running)
	}
	if result.Queue.Total != 5 {
		t.Errorf("queue.total = %d, want 5", result.Queue.Total)
	}
	if len(result.Events) != 1 {
		t.Errorf("events length = %d, want 1", len(result.Events))
	}
}

func TestPrintJSONDashboard_EmptyData(t *testing.T) {
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := printJSONDashboard("/test", "empty", nil, queue.Stats{}, nil)

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	if err != nil {
		t.Fatalf("printJSONDashboard returned error: %v", err)
	}

	var result dashboardOutput
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if result.Agents.Total != 0 {
		t.Errorf("agents.total = %d, want 0", result.Agents.Total)
	}
	if result.Agents.Running != 0 {
		t.Errorf("agents.running = %d, want 0", result.Agents.Running)
	}
}
