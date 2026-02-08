// Package tui: benchmarks and regression tests for bc home TUI/dashboard.
// Run benchmarks: go test -bench=. -benchmem ./internal/tui/...
// Or: make bench
// Regression tests run with the rest of the suite (go test ./... / make test).
package tui

import (
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
)

// --- Benchmarks: critical TUI/dashboard paths ---

func BenchmarkHomeView_Empty(b *testing.B) {
	m := NewHomeModel(nil, 0)
	m.width = 120
	m.height = 40
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkHomeView_WithWorkspaces(b *testing.B) {
	m := newTestHomeModel()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_Agents(b *testing.B) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateIdle},
		{Name: "eng-03", State: agent.StateStopped},
	}
	m.agentsLoaded = true
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_Issues(b *testing.B) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Fix login", Status: "open"},
		{ID: "bd-002", Title: "Add tests", Status: "in_progress"},
		{ID: "bd-003", Title: "Docs", Status: "closed"},
	}
	m.issuesLoaded = true
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_Channels(b *testing.B) {
	m := newTestModel()
	m.tab = TabChannels
	m.channels = []*channel.Channel{
		{Name: "standup"},
		{Name: "reviews"},
	}
	m.channelsLoaded = true
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_Queue(b *testing.B) {
	m := newTestModel()
	m.tab = TabQueue
	m.queueItems = []QueueItem{
		{ID: "bd-1", Title: "Task one", Status: "ready", Assignee: "", Type: "work"},
		{ID: "bd-2", Title: "Task two", Status: "in_progress", Assignee: "eng-01", Type: "work"},
	}
	m.filteredQueue = m.queueItems
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_Dashboard(b *testing.B) {
	m := newTestModel()
	m.tab = TabDashboard
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Open issue", Status: "open"},
		{ID: "bd-002", Title: "In progress", Status: "in_progress"},
		{ID: "bd-003", Title: "Done", Status: "closed"},
	}
	m.recentEvents = []events.Event{
		{Type: events.WorkCompleted, Agent: "engineer-01", Message: "Done"},
	}
	m.issuesLoaded = true
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_Stats(b *testing.B) {
	m := newTestModel()
	m.tab = TabStats
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateIdle},
	}
	m.agentsLoaded = true
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

// --- Regression tests: catch perf/behavior regressions ---

// TestHomeView_Regression_NoPanic ensures HomeModel.View() does not panic for common states.
func TestHomeView_Regression_NoPanic(t *testing.T) {
	// Empty workspaces
	m := NewHomeModel(nil, 0)
	m.width = 80
	m.height = 24
	_ = m.View()

	// With workspaces (home screen)
	m2 := newTestHomeModel()
	_ = m2.View()

	// With help active
	m2.helpActive = true
	_ = m2.View()
}

// TestHomeView_Regression_ExpectedSections ensures home view output contains expected structure.
func TestHomeView_Regression_ExpectedSections(t *testing.T) {
	m := newTestHomeModel()
	out := m.View()
	for _, want := range []string{"bc", "Workspaces", "project-a", "project-b", "NAME", "PATH", "AGENTS"} {
		if !strings.Contains(out, want) {
			t.Errorf("home view missing expected section or label %q", want)
		}
	}
}

// TestWorkspaceView_Regression_NoPanic ensures WorkspaceModel.View() does not panic for all tabs.
func TestWorkspaceView_Regression_NoPanic(t *testing.T) {
	tabs := []Tab{TabAgents, TabIssues, TabChannels, TabQueue, TabDashboard, TabStats}
	for _, tab := range tabs {
		m := newTestModel()
		m.tab = tab
		m.computeStats()
		_ = m.View()
	}
}

// TestWorkspaceView_Regression_ExpectedSections ensures workspace view has expected structure per tab.
func TestWorkspaceView_Regression_ExpectedSections(t *testing.T) {
	m := newTestModel()
	m.computeStats()
	out := m.View()
	// Tab bar and stats bar should always be present
	if !strings.Contains(out, "Dashboard") {
		t.Error("workspace view missing tab label Dashboard")
	}
	if !strings.Contains(out, "Agents") {
		t.Error("workspace view missing tab label Agents")
	}
	// Dashboard tab content (newTestModel defaults to TabDashboard)
	if !strings.Contains(out, "Issue Overview") {
		t.Error("workspace view missing Issue Overview section")
	}
}

// TestHomeView_Regression_AllTabsRender ensures full home flow (home + workspace) renders without panic.
func TestHomeView_Regression_AllTabsRender(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenWorkspace
	m.wsModel = newTestModel()
	m.wsModel.computeStats()
	out := m.View()
	if out == "" {
		t.Error("home view with workspace screen produced empty output")
	}
	if !strings.Contains(out, "test-project") {
		t.Error("workspace name should appear in view")
	}
}
