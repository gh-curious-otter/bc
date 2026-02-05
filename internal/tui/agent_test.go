package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/tui/style"
)

func newTestAgentModel(a *agent.Agent) *AgentModel {
	m := &AgentModel{
		agent:  a,
		styles: style.DefaultStyles(),
		width:  120,
		height: 40,
	}
	return m
}

func TestAgentView_BasicInfo(t *testing.T) {
	a := &agent.Agent{
		Name:      "engineer-01",
		Role:      agent.RoleEngineer,
		State:     agent.StateIdle,
		Workspace: "/tmp/ws",
		StartedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
	}
	m := newTestAgentModel(a)

	output := m.View()

	for _, want := range []string{
		"engineer-01",
		"engineer",
		"idle",
		"/tmp/ws",
		"Agent Info",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}

func TestAgentView_TaskField(t *testing.T) {
	a := &agent.Agent{
		Name:      "engineer-01",
		Role:      agent.RoleEngineer,
		State:     agent.StateWorking,
		StartedAt: time.Now().Add(-time.Hour),
		Task:      "work-108",
	}
	m := newTestAgentModel(a)

	output := m.View()

	if !strings.Contains(output, "work-108") {
		t.Errorf("expected task 'work-108' in output")
	}
	if !strings.Contains(output, "Task:") {
		t.Errorf("expected 'Task:' label in output")
	}
}

func TestAgentView_NoTaskWhenEmpty(t *testing.T) {
	a := &agent.Agent{
		Name:      "engineer-01",
		Role:      agent.RoleEngineer,
		State:     agent.StateIdle,
		StartedAt: time.Now(),
	}
	m := newTestAgentModel(a)

	output := m.View()

	// "Task:" should not appear when no task is assigned
	if strings.Contains(output, "Task:") {
		t.Errorf("should not show Task field when empty")
	}
}

func TestAgentView_Uptime(t *testing.T) {
	a := &agent.Agent{
		Name:      "engineer-01",
		Role:      agent.RoleEngineer,
		State:     agent.StateIdle,
		StartedAt: time.Now().Add(-2 * time.Hour),
	}
	m := newTestAgentModel(a)

	output := m.View()

	if !strings.Contains(output, "Uptime:") {
		t.Errorf("expected 'Uptime:' label in output")
	}
}

func TestAgentView_NoUptimeWhenStopped(t *testing.T) {
	a := &agent.Agent{
		Name:      "engineer-01",
		Role:      agent.RoleEngineer,
		State:     agent.StateStopped,
		StartedAt: time.Now().Add(-time.Hour),
	}
	m := newTestAgentModel(a)

	output := m.View()

	if strings.Contains(output, "Uptime:") {
		t.Errorf("should not show Uptime when agent is stopped")
	}
}

func TestAgentView_StatsSection(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateIdle,
	}
	m := newTestAgentModel(a)
	m.taskItems = []queue.WorkItem{
		{ID: "w-1", Status: queue.StatusDone, AssignedTo: "engineer-01", Title: "Task 1"},
		{ID: "w-2", Status: queue.StatusDone, AssignedTo: "engineer-01", Title: "Task 2"},
		{ID: "w-3", Status: queue.StatusFailed, AssignedTo: "engineer-01", Title: "Task 3"},
		{ID: "w-4", Status: queue.StatusWorking, AssignedTo: "engineer-01", Title: "Task 4"},
	}
	m.tasksCompleted = 2
	m.tasksFailed = 1

	output := m.View()

	if !strings.Contains(output, "Task Stats") {
		t.Errorf("expected 'Task Stats' header in output")
	}
	if !strings.Contains(output, "4") {
		t.Errorf("expected assigned count '4' in output")
	}
	if !strings.Contains(output, "2") {
		t.Errorf("expected completed count '2' in output")
	}
}

func TestAgentView_TaskListSection(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateWorking,
	}
	m := newTestAgentModel(a)
	m.taskItems = []queue.WorkItem{
		{ID: "w-10", Status: queue.StatusWorking, Title: "Implement login"},
		{ID: "w-11", Status: queue.StatusDone, Title: "Add tests"},
	}

	output := m.View()

	if !strings.Contains(output, "Tasks") {
		t.Errorf("expected 'Tasks' header in output")
	}
	if !strings.Contains(output, "w-10") {
		t.Errorf("expected task ID 'w-10' in output")
	}
	if !strings.Contains(output, "Implement login") {
		t.Errorf("expected task title 'Implement login' in output")
	}
}

func TestAgentView_TaskListEmpty(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateIdle,
	}
	m := newTestAgentModel(a)
	m.taskItems = nil

	output := m.View()

	// When no tasks, the Tasks section should not appear
	if strings.Contains(output, "\nTasks\n") {
		t.Errorf("should not show Tasks section when no tasks assigned")
	}
}

func TestAgentView_TaskTitleTruncation(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateWorking,
	}
	m := newTestAgentModel(a)
	m.taskItems = []queue.WorkItem{
		{ID: "w-1", Status: queue.StatusWorking, Title: "This is a very long task title that should be truncated at fifty characters"},
	}

	output := m.View()

	if !strings.Contains(output, "...") {
		t.Errorf("expected truncation '...' for long title")
	}
	// Original full title should NOT appear
	if strings.Contains(output, "fifty characters") {
		t.Errorf("long title should be truncated")
	}
}

func TestAgentView_RecentActivity(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateIdle,
	}
	m := newTestAgentModel(a)
	m.recentEvents = []events.Event{
		{
			Timestamp: time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC),
			Type:      events.WorkCompleted,
			Agent:     "engineer-01",
			Message:   "Completed work-108",
		},
		{
			Timestamp: time.Date(2025, 1, 15, 14, 35, 0, 0, time.UTC),
			Type:      events.WorkAssigned,
			Agent:     "engineer-01",
			Message:   "Assigned work-125",
		},
	}

	output := m.View()

	if !strings.Contains(output, "Recent Activity") {
		t.Errorf("expected 'Recent Activity' header")
	}
	if !strings.Contains(output, "Completed work-108") {
		t.Errorf("expected event message in output")
	}
	if !strings.Contains(output, "14:30:00") {
		t.Errorf("expected timestamp in output")
	}
}

func TestAgentView_NoRecentActivity(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateIdle,
	}
	m := newTestAgentModel(a)
	m.recentEvents = nil

	output := m.View()

	if !strings.Contains(output, "Recent Activity") {
		t.Errorf("expected 'Recent Activity' header even when empty")
	}
	if !strings.Contains(output, "No recent activity") {
		t.Errorf("expected 'No recent activity' placeholder")
	}
}

func TestAgentView_EventMessageTruncation(t *testing.T) {
	a := &agent.Agent{
		Name:  "engineer-01",
		Role:  agent.RoleEngineer,
		State: agent.StateIdle,
	}
	m := newTestAgentModel(a)
	m.recentEvents = []events.Event{
		{
			Timestamp: time.Now(),
			Type:      events.AgentReport,
			Agent:     "engineer-01",
			Message:   "This is a very long event message that should be truncated because it exceeds the sixty character limit set in the code",
		},
	}

	output := m.View()

	if !strings.Contains(output, "...") {
		t.Errorf("expected truncation for long event message")
	}
	if strings.Contains(output, "sixty character limit") {
		t.Errorf("long message should be truncated")
	}
}

func TestAgentHandleKey_Esc(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if action.Type != ActionBack {
		t.Errorf("expected ActionBack, got %v", action.Type)
	}
}

func TestAgentHandleKey_EscFromPeek(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)
	m.peekActive = true

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if action.Type != ActionNone {
		t.Errorf("expected ActionNone when dismissing peek, got %v", action.Type)
	}
	if m.peekActive {
		t.Errorf("peekActive should be false after Esc")
	}
}

func TestAgentHandleKey_Attach(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if action.Type != ActionAttach {
		t.Errorf("expected ActionAttach, got %v", action.Type)
	}
	if action.Data != "engineer-01" {
		t.Errorf("expected agent name as data, got %v", action.Data)
	}
}

func TestAgentHandleKey_Unknown(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if action.Type != ActionNone {
		t.Errorf("expected ActionNone for unknown key, got %v", action.Type)
	}
}

func TestAgentView_PeekOutput(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)
	m.peekActive = true
	m.peekOutput = "line 1\nline 2\nline 3"

	output := m.View()

	if !strings.Contains(output, "Recent Output") {
		t.Errorf("expected 'Recent Output' header in peek view")
	}
	if !strings.Contains(output, "line 1") {
		t.Errorf("expected peek output content")
	}
	if !strings.Contains(output, "line 3") {
		t.Errorf("expected all peek output lines")
	}
	// Should NOT show normal info sections
	if strings.Contains(output, "Agent Info") {
		t.Errorf("should not show Agent Info in peek mode")
	}
}

func TestAgentView_PeekEmpty(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)
	m.peekActive = true
	m.peekOutput = ""

	output := m.View()

	if !strings.Contains(output, "No output captured") {
		t.Errorf("expected 'No output captured' placeholder in empty peek")
	}
}

func TestActiveAndRecentTasks(t *testing.T) {
	a := &agent.Agent{Name: "engineer-01", State: agent.StateIdle}
	m := newTestAgentModel(a)
	m.taskItems = []queue.WorkItem{
		{ID: "w-1", Status: queue.StatusDone, Title: "Done 1"},
		{ID: "w-2", Status: queue.StatusDone, Title: "Done 2"},
		{ID: "w-3", Status: queue.StatusDone, Title: "Done 3"},
		{ID: "w-4", Status: queue.StatusDone, Title: "Done 4"},
		{ID: "w-5", Status: queue.StatusDone, Title: "Done 5"},
		{ID: "w-6", Status: queue.StatusDone, Title: "Done 6"},
		{ID: "w-7", Status: queue.StatusDone, Title: "Done 7"},
		{ID: "w-8", Status: queue.StatusWorking, Title: "Active 1"},
		{ID: "w-9", Status: queue.StatusPending, Title: "Pending 1"},
	}

	result := m.activeAndRecentTasks()

	// Active items come first
	if result[0].ID != "w-8" {
		t.Errorf("expected active task first, got %s", result[0].ID)
	}
	if result[1].ID != "w-9" {
		t.Errorf("expected pending task second, got %s", result[1].ID)
	}

	// Only 5 most recent finished tasks should appear (w-3 through w-7)
	finishedCount := 0
	for _, item := range result {
		if item.Status == queue.StatusDone {
			finishedCount++
		}
	}
	if finishedCount != 5 {
		t.Errorf("expected 5 finished tasks, got %d", finishedCount)
	}

	// Total: 2 active + 5 finished = 7
	if len(result) != 7 {
		t.Errorf("expected 7 total tasks, got %d", len(result))
	}
}
