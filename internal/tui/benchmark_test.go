package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/tui/style"
	"github.com/rpuneet/bc/pkg/workspace"
)

// BenchmarkWorkspaceRefresh measures refresh() cost for regression (#296).
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
