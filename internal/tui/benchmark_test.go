package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/tui/style"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Benchmarks for TUI/dashboard performance regression (#312).

// BenchmarkWorkspaceRefresh measures refresh() for regression (#312, #296).
func BenchmarkWorkspaceRefresh(b *testing.B) {
	dir := b.TempDir()
	bcDir := filepath.Join(dir, ".bc", "agents")
	if err := os.MkdirAll(bcDir, 0750); err != nil {
		b.Fatal(err)
	}
	info := WorkspaceInfo{Entry: workspace.RegistryEntry{Path: dir}}
	m := &WorkspaceModel{
		info:   info,
		styles: style.DefaultStyles(),
		tab:    TabAgents,
	}
	m.manager = agent.NewWorkspaceManager(bcDir, dir)
	_ = m.manager.LoadState()
	m.agents = m.manager.ListAgents()
	m.agentsLoaded = true
	m.issuesLoaded = true
	m.channelsLoaded = true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.refresh()
	}
}

// BenchmarkRenderDashboard measures dashboard rendering (#312).
func BenchmarkRenderDashboard(b *testing.B) {
	m := newTestModel()
	m.tab = TabDashboard
	m.stats = WorkspaceStats{
		TotalIssues: 50, OpenIssues: 30, ClosedIssues: 20, EpicsCount: 3,
		ReadyIssues: 5, InProgressIssues: 8, AssignedIssues: 10,
		IdleAgents: 2, WorkingAgents: 3, StuckAgents: 0, StoppedAgents: 1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderDashboard()
	}
}

// BenchmarkRenderQueue measures queue tab rendering (#312).
func BenchmarkRenderQueue(b *testing.B) {
	m := newTestModel()
	m.tab = TabQueue
	m.filteredQueue = make([]QueueItem, 30)
	for i := range m.filteredQueue {
		m.filteredQueue[i] = QueueItem{
			ID: "bc-1", Title: "Task title here", Type: "work", Status: "ready",
			Assignee: "engineer-01", Agent: "", Branch: "",
		}
	}
	m.width = 120
	m.height = 40

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = m.renderQueue()
	}
}
