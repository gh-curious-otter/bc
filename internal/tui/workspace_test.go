package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/stats"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// newTestWorkspaceModel creates a WorkspaceModel with test data pre-loaded
// (no filesystem access needed).
func newTestWorkspaceModel() *WorkspaceModel {
	m := &WorkspaceModel{
		styles: style.DefaultStyles(),
		width:  120,
		height: 40,
		tab:    TabStats,
	}
	return m
}

func TestRenderStatsPanel_WorkItems(t *testing.T) {
	m := newTestWorkspaceModel()
	m.queueStats = queue.Stats{
		Total:    20,
		Pending:  3,
		Assigned: 2,
		Working:  5,
		Done:     8,
		Failed:   2,
	}

	output := m.renderStatsPanel()

	for _, want := range []string{
		"Work Items",
		"20",  // total
		"5",   // pending + assigned
		"8",   // done
		"40.0%", // completion = 8/20
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in work items section", want)
		}
	}
}

func TestRenderStatsPanel_Agents(t *testing.T) {
	m := newTestWorkspaceModel()
	m.agents = []*agent.Agent{
		{Name: "e-01", State: agent.StateWorking},
		{Name: "e-02", State: agent.StateIdle},
		{Name: "e-03", State: agent.StateStopped},
	}
	m.stats = WorkspaceStats{
		WorkingAgents: 1,
		IdleAgents:    1,
		StuckAgents:   0,
		StoppedAgents: 1,
	}

	output := m.renderStatsPanel()

	if !strings.Contains(output, "Agents") {
		t.Errorf("expected 'Agents' header")
	}
	if !strings.Contains(output, "3") { // total
		t.Errorf("expected total agents '3'")
	}
	if !strings.Contains(output, "50%") { // utilization = 1 working / 2 active
		t.Errorf("expected utilization '50%%'")
	}
}

func TestRenderStatsPanel_PerAgentStats(t *testing.T) {
	m := newTestWorkspaceModel()
	m.agents = []*agent.Agent{
		{Name: "engineer-01", State: agent.StateWorking, StartedAt: time.Now().Add(-2 * time.Hour)},
		{Name: "engineer-02", State: agent.StateIdle, StartedAt: time.Now().Add(-time.Hour)},
	}
	m.agentStats = map[string]stats.AgentStat{
		"engineer-01": {
			Name:           "engineer-01",
			State:          "working",
			TasksCompleted: 5,
			TasksFailed:    1,
			Uptime:         2 * time.Hour,
		},
		"engineer-02": {
			Name:           "engineer-02",
			State:          "idle",
			TasksCompleted: 3,
			TasksFailed:    0,
			Uptime:         time.Hour,
		},
	}

	output := m.renderStatsPanel()

	if !strings.Contains(output, "Per-Agent Stats") {
		t.Errorf("expected 'Per-Agent Stats' header")
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("expected agent name 'engineer-01'")
	}
	if !strings.Contains(output, "engineer-02") {
		t.Errorf("expected agent name 'engineer-02'")
	}
	// engineer-01: 5/(5+1) = 83%
	if !strings.Contains(output, "83%") {
		t.Errorf("expected completion rate '83%%' for engineer-01")
	}
	// engineer-02: 3/(3+0) = 100%
	if !strings.Contains(output, "100%") {
		t.Errorf("expected completion rate '100%%' for engineer-02")
	}
}

func TestRenderStatsPanel_NoAgentStats(t *testing.T) {
	m := newTestWorkspaceModel()
	m.agents = nil
	m.agentStats = nil

	output := m.renderStatsPanel()

	if !strings.Contains(output, "No agent stats available") {
		t.Errorf("expected 'No agent stats available' placeholder")
	}
}

func TestRenderStatsPanel_WorkItemTypes(t *testing.T) {
	m := newTestWorkspaceModel()
	m.pkgStats = &stats.Stats{
		WorkItems: stats.WorkItemMetrics{
			Epics: 3,
			Tasks: 10,
			Bugs:  5,
			Other: 2,
		},
	}

	output := m.renderStatsPanel()

	if !strings.Contains(output, "Work Item Types") {
		t.Errorf("expected 'Work Item Types' header")
	}
	for _, want := range []string{"3", "10", "5", "2"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected type count %q in output", want)
		}
	}
}

func TestRenderStatsPanel_AvgCompletionTime(t *testing.T) {
	m := newTestWorkspaceModel()
	m.pkgStats = &stats.Stats{
		WorkItems: stats.WorkItemMetrics{
			AvgTimeToComplete: 45 * time.Minute,
		},
	}

	output := m.renderStatsPanel()

	if !strings.Contains(output, "Avg Completion") {
		t.Errorf("expected 'Avg Completion' label")
	}
	if !strings.Contains(output, "45m") {
		t.Errorf("expected '45m' in avg completion time")
	}
}

func TestRenderStatsPanel_ZeroStats(t *testing.T) {
	m := newTestWorkspaceModel()
	m.queueStats = queue.Stats{}
	m.stats = WorkspaceStats{}
	m.agentStats = nil

	output := m.renderStatsPanel()

	// Should render without panics and show 0 values
	if !strings.Contains(output, "Work Items") {
		t.Errorf("expected 'Work Items' header even with zero stats")
	}
	if !strings.Contains(output, "0.0%") {
		t.Errorf("expected '0.0%%' for completion rate with zero items")
	}
}

func TestRenderStatsPanel_HighCompletion(t *testing.T) {
	m := newTestWorkspaceModel()
	m.queueStats = queue.Stats{
		Total:   10,
		Done:    9,
		Working: 1,
	}

	output := m.renderStatsPanel()

	if !strings.Contains(output, "90.0%") {
		t.Errorf("expected '90.0%%' completion rate")
	}
}

func TestRenderStatsPanel_HighFailure(t *testing.T) {
	m := newTestWorkspaceModel()
	m.queueStats = queue.Stats{
		Total:  10,
		Done:   3,
		Failed: 5,
	}

	output := m.renderStatsPanel()

	// 50% failure rate
	if !strings.Contains(output, "50.0%") {
		t.Errorf("expected '50.0%%' failure rate")
	}
}

func TestTabBarIncludesStats(t *testing.T) {
	m := newTestWorkspaceModel()
	m.tab = TabStats

	output := m.renderTabBar()

	if !strings.Contains(output, "Stats") {
		t.Errorf("expected 'Stats' tab in tab bar")
	}
	// Stats tab should not show a count
	if strings.Contains(output, "Stats (") {
		t.Errorf("Stats tab should not show a count in parentheses")
	}
}

func TestStatsTabRendersInView(t *testing.T) {
	m := newTestWorkspaceModel()
	m.tab = TabStats
	m.queueStats = queue.Stats{Total: 5, Done: 3}

	output := m.View()

	if !strings.Contains(output, "Work Items") {
		t.Errorf("expected stats panel content in View when TabStats active")
	}
}

func TestTabCount(t *testing.T) {
	// Ensure tabCount matches the number of defined tabs
	expected := 5
	if tabCount != expected {
		t.Errorf("expected tabCount=%d, got %d", expected, tabCount)
	}
	// Verify TabStats is the last tab
	if int(TabStats) != tabCount-1 {
		t.Errorf("expected TabStats=%d, got %d", tabCount-1, int(TabStats))
	}
}

func TestStatsTabCursorAlwaysZero(t *testing.T) {
	m := newTestWorkspaceModel()
	m.tab = TabStats

	max := m.maxCursor()
	if max != 0 {
		t.Errorf("expected maxCursor=0 for stats tab, got %d", max)
	}
}

func TestPerAgentStatsUptimeFallback(t *testing.T) {
	// When pkg/stats has no uptime, fall back to agent.StartedAt
	m := newTestWorkspaceModel()
	started := time.Now().Add(-3 * time.Hour)
	m.agents = []*agent.Agent{
		{Name: "e-01", State: agent.StateWorking, StartedAt: started},
	}
	m.agentStats = map[string]stats.AgentStat{
		"e-01": {
			Name:           "e-01",
			State:          "working",
			TasksCompleted: 2,
			TasksFailed:    0,
			Uptime:         0, // no uptime from stats — should fall back
		},
	}

	output := m.renderStatsPanel()

	// Should show "3h" from the fallback calculation
	if !strings.Contains(output, "3h") {
		t.Errorf("expected uptime fallback '3h' in output, got:\n%s", output)
	}
}

func TestPerAgentNoTasksShowsDash(t *testing.T) {
	m := newTestWorkspaceModel()
	m.agents = []*agent.Agent{
		{Name: "e-01", State: agent.StateIdle},
	}
	m.agentStats = map[string]stats.AgentStat{
		"e-01": {
			Name:           "e-01",
			State:          "idle",
			TasksCompleted: 0,
			TasksFailed:    0,
		},
	}

	output := m.renderStatsPanel()

	// Rate should be "-" when no tasks
	lines := strings.Split(output, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, "e-01") && strings.Contains(line, "-") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected '-' rate for agent with no tasks")
	}
}

// Verify that the format helper is consistent.
func TestFmtDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m 0s"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.d), func(t *testing.T) {
			got := fmtDuration(tt.d)
			if got != tt.want {
				t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
