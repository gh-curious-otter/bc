// Package tui: benchmarks and regression tests for bc home TUI/dashboard (#311, #312).
//
// Run benchmarks:
//
//	go test -bench=. -benchmem ./internal/tui/...
//	make bench
//
// Benchmarks cover: home (empty, loading, with workspaces, each screen), workspace
// (all tabs, first-paint lazy, many agents), and regression tests for no-panic and structure.
package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/workspace"
)

// --- Benchmarks: critical TUI/dashboard paths (#312) ---

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

// BenchmarkHomeView_AsyncLoadFirstFrame measures time to first frame when using
// async workspace load (#323): TUI shows "Loading workspaces..." immediately.
func BenchmarkHomeView_AsyncLoadFirstFrame(b *testing.B) {
	m := NewHomeModel(nil, 5, true)
	m.width = 120
	m.height = 40
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
	m.agentStatsLoaded = true
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
	m.queueLoaded = true
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
	m.channelsLoaded = true
	m.queueLoaded = true
	m.agentStatsLoaded = true
	m.pkgStatsLoaded = true
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
	m.agentStatsLoaded = true
	m.pkgStatsLoaded = true
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

// --- Home view: drill-down screens and stress (#312) ---

func BenchmarkHomeView_WorkspaceScreen(b *testing.B) {
	m := newTestHomeModel()
	m.screen = ScreenWorkspace
	m.wsModel = newTestModel()
	m.wsModel.tab = TabAgents
	m.wsModel.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateIdle},
	}
	m.wsModel.agentsLoaded = true
	m.wsModel.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkHomeView_AgentScreen(b *testing.B) {
	m := newTestHomeModel()
	m.screen = ScreenAgent
	m.wsModel = newTestModel()
	m.agentModel = &AgentModel{
		agent:  &agent.Agent{Name: "engineer-01", State: agent.StateWorking, Task: "Implement login"},
		styles: m.styles,
		width:  120,
		height: 40,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkHomeView_ChannelScreen(b *testing.B) {
	m := newTestHomeModel()
	m.screen = ScreenChannel
	m.wsModel = newTestModel()
	m.channelModel = &ChannelModel{
		channel: &channel.Channel{Name: "standup", History: []channel.HistoryEntry{
			{Sender: "eng-01", Message: "Starting work", Time: time.Now()},
			{Sender: "eng-02", Message: "Tests passing", Time: time.Now()},
		}},
		styles: newTestModel().styles,
		width:  120,
		height: 40,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkHomeView_HelpActive(b *testing.B) {
	m := newTestHomeModel()
	m.helpActive = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkHomeView_ManyWorkspaces(b *testing.B) {
	workspaces := make([]WorkspaceInfo, 0, 25)
	for i := 0; i < 25; i++ {
		workspaces = append(workspaces, WorkspaceInfo{
			Entry:      workspace.RegistryEntry{Name: fmt.Sprintf("project-%d", i), Path: fmt.Sprintf("/path/%d", i)},
			Running:    i % 4,
			Total:      5,
			MaxWorkers: 10,
			Issues:     i * 2,
			HasBeads:   i%2 == 0,
		})
	}
	m := NewHomeModel(workspaces, 10)
	m.width = 120
	m.height = 40
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

// --- Workspace: minimal agents (fast path) and stress (#312) ---

func BenchmarkWorkspaceView_FirstPaintLazy(b *testing.B) {
	// Minimal agent list and stats (no issues/channels); fast first-paint path.
	m := newTestModel()
	m.tab = TabAgents
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateIdle},
		{Name: "eng-03", State: agent.StateStopped},
	}
	m.agentsLoaded = true
	m.issues = nil
	m.computeStats()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.View()
	}
}

func BenchmarkWorkspaceView_AgentsMany(b *testing.B) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = make([]*agent.Agent, 0, 30)
	for i := 0; i < 30; i++ {
		m.agents = append(m.agents, &agent.Agent{
			Name:  fmt.Sprintf("engineer-%02d", i),
			State: agent.StateWorking,
			Task:  "Implementing feature X",
		})
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

// TestHomeView_Regression_WorkspaceLoadingNoPanic ensures View() does not panic when workspace is loading (#311/#325).
func TestHomeView_Regression_WorkspaceLoadingNoPanic(t *testing.T) {
	m := newTestHomeModel()
	m.screen = ScreenWorkspace
	m.workspaceLoading = true
	m.pendingWorkspaceName = "test-project"
	m.wsModel = nil
	out := m.View()
	if out == "" {
		t.Error("workspace loading view produced empty output")
	}
	if !strings.Contains(out, "Loading") {
		t.Error("workspace loading view should contain Loading")
	}
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

// TestHomeView_Regression_AllScreensNoPanic ensures View() does not panic for every home screen (matches bench coverage).
func TestHomeView_Regression_AllScreensNoPanic(t *testing.T) {
	screens := []struct {
		init func() *HomeModel
		name string
	}{
		{func() *HomeModel {
			m := newTestHomeModel()
			m.screen = ScreenWorkspace
			m.wsModel = newTestModel()
			m.wsModel.computeStats()
			return m
		}, "WorkspaceScreen"},
		{func() *HomeModel {
			m := newTestHomeModel()
			m.screen = ScreenAgent
			m.wsModel = newTestModel()
			m.agentModel = &AgentModel{agent: &agent.Agent{Name: "eng-01"}, styles: m.styles}
			return m
		}, "AgentScreen"},
		{func() *HomeModel {
			m := newTestHomeModel()
			m.screen = ScreenChannel
			m.wsModel = newTestModel()
			m.channelModel = &ChannelModel{
				channel: &channel.Channel{Name: "standup"},
				styles:  m.styles,
			}
			return m
		}, "ChannelScreen"},
		{func() *HomeModel {
			m := newTestHomeModel()
			m.helpActive = true
			return m
		}, "HelpActive"},
	}
	for _, sc := range screens {
		t.Run(sc.name, func(t *testing.T) {
			m := sc.init()
			out := m.View()
			if out == "" {
				t.Error("view produced empty output")
			}
		})
	}
}
