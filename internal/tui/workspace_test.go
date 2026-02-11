package tui

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/stats"
	"github.com/rpuneet/bc/pkg/tui/style"
	"github.com/rpuneet/bc/pkg/workspace"
)

// newTestModel creates a WorkspaceModel with pre-populated data for testing,
// without hitting disk. It skips agent manager initialization.
func newTestModel() *WorkspaceModel {
	return &WorkspaceModel{
		info: WorkspaceInfo{
			Entry: workspace.RegistryEntry{
				Name: "test-project",
				Path: "/tmp/test-project",
			},
		},
		styles: style.DefaultStyles(),
		width:  120,
		height: 40,
		tab:    TabDashboard,
	}
}

// --- Lazy-load / computeStatsFromAgentsOnly (#324) ---

func TestComputeStatsFromAgentsOnly(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "a", State: agent.StateWorking},
		{Name: "b", State: agent.StateIdle},
		{Name: "c", State: agent.StateStopped},
	}
	m.computeStatsFromAgentsOnly()
	if m.stats.WorkingAgents != 1 || m.stats.IdleAgents != 1 || m.stats.StoppedAgents != 1 {
		t.Errorf("agent counts: working=1 idle=1 stopped=1, got working=%d idle=%d stopped=%d",
			m.stats.WorkingAgents, m.stats.IdleAgents, m.stats.StoppedAgents)
	}
	if m.stats.TotalIssues != 0 || m.stats.OpenIssues != 0 {
		t.Errorf("issue stats should be zero, got total=%d open=%d", m.stats.TotalIssues, m.stats.OpenIssues)
	}
}

func TestEnsureTabDataLoaded_NoPanic(t *testing.T) {
	m := newTestModel()
	// ensureTabDataLoaded should not panic for any tab (may load or no-op)
	for tab := TabAgents; tab < tabCount; tab++ {
		m.ensureTabDataLoaded(tab)
	}
}

// --- Dashboard rendering tests ---

func TestRenderDashboard_NoData(t *testing.T) {
	m := newTestModel()
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "Issue Overview") {
		t.Error("expected 'Issue Overview' section header")
	}
	if !strings.Contains(output, "No issues tracked") {
		t.Errorf("expected 'No issues tracked' for empty issues, got: %s", output)
	}
	if !strings.Contains(output, "Recently Closed") {
		t.Error("expected 'Recently Closed' section header")
	}
	if !strings.Contains(output, "No recently closed issues") {
		t.Errorf("expected 'No recently closed issues', got: %s", output)
	}
	if !strings.Contains(output, "Activity Feed") {
		t.Error("expected 'Activity Feed' section header")
	}
	if !strings.Contains(output, "No recent activity") {
		t.Errorf("expected 'No recent activity', got: %s", output)
	}
}

func TestRenderDashboard_WithIssues(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Fix login bug", Status: "open"},
		{ID: "bd-002", Title: "Add dark mode", Status: "in_progress"},
		{ID: "bd-003", Title: "Update README", Status: "closed"},
		{ID: "bd-004", Title: "Refactor auth", Status: "done"},
		{ID: "bd-005", Title: "Add tests", Status: "pending"},
	}
	m.computeStats()

	output := m.renderDashboard()

	// Should show open count (open + pending + in_progress = 3)
	if !strings.Contains(output, "Open: 3") {
		t.Errorf("expected 'Open: 3', got: %s", output)
	}
	// Should show closed count (closed + done = 2)
	if !strings.Contains(output, "Closed: 2") {
		t.Errorf("expected 'Closed: 2', got: %s", output)
	}
	// Should show total
	if !strings.Contains(output, "Total: 5") {
		t.Errorf("expected 'Total: 5', got: %s", output)
	}
}

func TestRenderDashboard_RecentlyClosed(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Open issue", Status: "open"},
		{ID: "bd-002", Title: "First closed", Status: "closed"},
		{ID: "bd-003", Title: "Second done", Status: "done"},
		{ID: "bd-004", Title: "Third resolved", Status: "resolved"},
	}
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "First closed") {
		t.Errorf("expected 'First closed' in recently closed, got: %s", output)
	}
	if !strings.Contains(output, "Second done") {
		t.Errorf("expected 'Second done' in recently closed, got: %s", output)
	}
}

func TestRenderDashboard_ActivityFeed(t *testing.T) {
	m := newTestModel()
	m.recentEvents = []events.Event{
		{
			Timestamp: time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC),
			Type:      events.WorkCompleted,
			Agent:     "engineer-01",
			Message:   "Completed work-108",
		},
		{
			Timestamp: time.Date(2025, 1, 15, 14, 35, 0, 0, time.UTC),
			Type:      events.AgentReport,
			Agent:     "",
			Message:   "System status: healthy",
		},
	}
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "Completed work-108") {
		t.Errorf("expected event message in activity feed, got: %s", output)
	}
	if !strings.Contains(output, "[engineer-01]") {
		t.Errorf("expected agent name in activity feed, got: %s", output)
	}
	if !strings.Contains(output, "14:30:00") {
		t.Errorf("expected timestamp in activity feed, got: %s", output)
	}
}

func TestRenderDashboard_AgentUtilization(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateWorking},
		{Name: "eng-03", State: agent.StateIdle},
		{Name: "eng-04", State: agent.StateStopped},
	}
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "AGENT UTILIZATION") {
		t.Error("expected 'AGENT UTILIZATION' section header")
	}
	if !strings.Contains(output, "STATE BREAKDOWN") {
		t.Error("expected 'STATE BREAKDOWN' section header")
	}
}

func TestRenderDashboard_AgentHealth(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "eng-01", Role: agent.Role("engineer"), State: agent.StateWorking},
		{Name: "qa-01", Role: agent.Role("qa"), State: agent.StateIdle},
	}
	m.agentStats = map[string]stats.AgentStat{
		"eng-01": {Name: "eng-01", State: "working", Uptime: time.Hour},
		"qa-01":  {Name: "qa-01", State: "idle", Uptime: 30 * time.Minute},
	}
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "AGENT HEALTH") {
		t.Error("expected 'AGENT HEALTH' section header")
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("expected eng-01 in agent health, got: %s", output)
	}
	if !strings.Contains(output, "qa-01") {
		t.Errorf("expected qa-01 in agent health, got: %s", output)
	}
}

// --- renderStatsBar tests ---

func TestRenderStatsBar_Empty(t *testing.T) {
	m := newTestModel()
	m.computeStats()

	output := m.renderStatsBar()

	if !strings.Contains(output, "Issues: 0") {
		t.Errorf("expected 'Issues: 0', got: %s", output)
	}
	if !strings.Contains(output, "Epics: 0") {
		t.Errorf("expected 'Epics: 0', got: %s", output)
	}
}

func TestRenderStatsBar_WithData(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Type: "epic", Status: "open"},
		{ID: "bd-002", Status: "open"},
		{ID: "bd-003", Status: "closed"},
	}
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateIdle},
	}
	m.computeStats()

	output := m.renderStatsBar()

	if !strings.Contains(output, "Issues: 3") {
		t.Errorf("expected 'Issues: 3', got: %s", output)
	}
	if !strings.Contains(output, "Epics: 1") {
		t.Errorf("expected 'Epics: 1', got: %s", output)
	}
}

// --- renderChannels tests ---

func TestRenderChannels_Empty(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	m.channels = nil

	output := m.renderChannels()
	if !strings.Contains(output, "No channels") {
		t.Errorf("expected 'No channels', got: %s", output)
	}
}

func TestRenderChannels_WithData(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	m.channels = []*channel.Channel{
		{
			Name:    "all",
			Members: []string{"eng-01", "eng-02", "manager"},
			History: []channel.HistoryEntry{
				{Sender: "manager", Message: "Ship faster!"},
			},
		},
		{
			Name:    "eng",
			Members: []string{"eng-01", "eng-02"},
		},
	}

	output := m.renderChannels()
	if !strings.Contains(output, "#all") {
		t.Errorf("expected '#all' channel, got: %s", output)
	}
	if !strings.Contains(output, "#eng") {
		t.Errorf("expected '#eng' channel, got: %s", output)
	}
	if !strings.Contains(output, "Ship faster!") {
		t.Errorf("expected last message, got: %s", output)
	}
}

// --- Pagination and scroll tests ---

func TestVisibleRows(t *testing.T) {
	m := newTestModel()
	m.height = 20 // 20 - 7 overhead = 13 visible

	visible := m.visibleRows()
	if visible != 13 {
		t.Errorf("visibleRows = %d, want 13", visible)
	}
}

func TestVisibleRowsMinimum(t *testing.T) {
	m := newTestModel()
	m.height = 5 // Too small, should still return at least 1

	visible := m.visibleRows()
	if visible < 1 {
		t.Errorf("visibleRows should be at least 1, got %d", visible)
	}
}

func TestViewportRange(t *testing.T) {
	m := newTestModel()
	m.height = 14 // visibleRows = 7
	m.scrollOffset = 5

	start, end := m.viewportRange(20)
	if start != 5 {
		t.Errorf("start = %d, want 5", start)
	}
	if end != 12 {
		t.Errorf("end = %d, want 12", end)
	}
}

func TestViewportRangeAtEnd(t *testing.T) {
	m := newTestModel()
	m.height = 14 // visibleRows = 7
	m.scrollOffset = 17

	start, end := m.viewportRange(20)
	if start != 17 {
		t.Errorf("start = %d, want 17", start)
	}
	if end != 20 {
		t.Errorf("end = %d, want 20 (clamped to total)", end)
	}
}

func TestViewportRangeEmpty(t *testing.T) {
	m := newTestModel()
	start, end := m.viewportRange(0)
	if start != 0 || end != 0 {
		t.Errorf("empty range should be (0, 0), got (%d, %d)", start, end)
	}
}

func TestPositionIndicator(t *testing.T) {
	m := newTestModel()
	m.height = 14 // visibleRows = 7
	m.scrollOffset = 10

	indicator := m.renderPositionIndicator(50)
	if !strings.Contains(indicator, "11-17 of 50") {
		t.Errorf("expected '11-17 of 50' in indicator, got %q", indicator)
	}
}

func TestPositionIndicatorFitsAll(t *testing.T) {
	m := newTestModel()
	m.height = 20 // visibleRows = 13

	indicator := m.renderPositionIndicator(5)
	if indicator != "" {
		t.Errorf("indicator should be empty when all items fit, got %q", indicator)
	}
}

func TestRenderIssues_Paginated(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.tab = TabIssues
	m.scrollOffset = 2

	for i := 0; i < 15; i++ {
		m.issues = append(m.issues, beads.Issue{
			ID:     fmt.Sprintf("bd-%03d", i+1),
			Title:  fmt.Sprintf("Issue %d", i+1),
			Status: "open",
		})
	}

	output := m.renderIssues()

	// With scrollOffset=2, should show items 3-7
	if !strings.Contains(output, "bd-003") {
		t.Errorf("expected bd-003 in viewport, got: %s", output)
	}
	if !strings.Contains(output, "bd-007") {
		t.Errorf("expected bd-007 in viewport, got: %s", output)
	}
	// Should NOT contain items before viewport
	if strings.Contains(output, "bd-001") {
		t.Errorf("bd-001 should not be in viewport")
	}
	// Should NOT contain items after viewport
	if strings.Contains(output, "bd-008") {
		t.Errorf("bd-008 should not be in viewport")
	}
	if !strings.Contains(output, "3-7 of 15") {
		t.Errorf("expected position indicator '3-7 of 15', got: %s", output)
	}
}

func TestTabSwitch_ResetsCursorAndScroll(t *testing.T) {
	m := newTestModel()
	m.cursor = 10
	m.scrollOffset = 5
	m.tab = TabAgents
	m.issues = make([]beads.Issue, 20)

	// Simulate pressing tab
	msg := tea.KeyMsg{Type: tea.KeyTab}
	m.HandleKey(msg)

	if m.cursor != 0 {
		t.Errorf("cursor should be 0 after tab switch, got %d", m.cursor)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset should be 0 after tab switch, got %d", m.scrollOffset)
	}
}

func TestCursorMovement_AdjustsViewport(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 20)
	m.cursor = 0
	m.scrollOffset = 0

	// Move cursor down past viewport
	for i := 0; i < 7; i++ {
		msg := tea.KeyMsg{Type: tea.KeyDown}
		m.HandleKey(msg)
	}

	// Cursor should be at 7, viewport should have scrolled
	if m.cursor != 7 {
		t.Errorf("cursor = %d, want 7", m.cursor)
	}
	visible := m.visibleRows()
	if m.cursor >= m.scrollOffset+visible {
		t.Errorf("cursor %d not visible in viewport [%d, %d)", m.cursor, m.scrollOffset, m.scrollOffset+visible)
	}
}

// --- renderAgents tests ---

func TestRenderAgents_NoAgents(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = nil

	output := m.renderAgents()
	if !strings.Contains(output, "No agents") {
		t.Errorf("expected 'No agents' message, got: %s", output)
	}
}

func TestRenderAgents_WithAgents(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = []*agent.Agent{
		{Name: "engineer-01", Role: agent.Role("engineer"), State: agent.StateWorking, Task: "fixing auth"},
		{Name: "qa-01", Role: agent.Role("qa"), State: agent.StateIdle},
	}
	m.agentStats = map[string]stats.AgentStat{
		"engineer-01": {Name: "engineer-01", Uptime: 2 * time.Hour},
		"qa-01":       {Name: "qa-01", Uptime: 30 * time.Minute},
	}

	output := m.renderAgents()
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("expected engineer-01 in output, got: %s", output)
	}
	if !strings.Contains(output, "qa-01") {
		t.Errorf("expected qa-01 in output, got: %s", output)
	}
	if !strings.Contains(output, "fixing auth") {
		t.Errorf("expected task in output, got: %s", output)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("expected header row, got: %s", output)
	}
}

func TestRenderAgents_SelectedHighlight(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = []*agent.Agent{
		{Name: "eng-01", Role: agent.Role("engineer"), State: agent.StateWorking},
		{Name: "eng-02", Role: agent.Role("engineer"), State: agent.StateIdle},
	}
	m.agentStats = map[string]stats.AgentStat{}
	m.cursor = 1 // select second agent

	output := m.renderAgents()
	// Both agents should appear
	if !strings.Contains(output, "eng-01") || !strings.Contains(output, "eng-02") {
		t.Errorf("expected both agents in output, got: %s", output)
	}
}

func TestRenderAgents_Paginated(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.tab = TabAgents
	m.agentStats = map[string]stats.AgentStat{}

	for i := 0; i < 20; i++ {
		m.agents = append(m.agents, &agent.Agent{
			Name:  fmt.Sprintf("agent-%02d", i+1),
			Role:  agent.Role("engineer"),
			State: agent.StateIdle,
		})
	}
	m.scrollOffset = 0

	output := m.renderAgents()
	if !strings.Contains(output, "agent-01") {
		t.Errorf("expected agent-01 in viewport")
	}
	if strings.Contains(output, "agent-10") {
		t.Errorf("agent-10 should not be in viewport")
	}
	if !strings.Contains(output, "1-5 of 20") {
		t.Errorf("expected position indicator '1-5 of 20', got: %s", output)
	}
}

// --- issuesErrorMessage tests ---

func TestIssuesErrorMessage_NoBeadsDir(t *testing.T) {
	m := newTestModel()
	m.issuesErr = beads.ErrNoBeadsDir
	msg := m.issuesErrorMessage()
	if !strings.Contains(msg, "No issue tracker configured") {
		t.Errorf("expected 'No issue tracker configured', got: %s", msg)
	}
}

func TestIssuesErrorMessage_ExecError(t *testing.T) {
	m := newTestModel()
	m.issuesErr = &exec.Error{Name: "bd", Err: fmt.Errorf("not found")}
	msg := m.issuesErrorMessage()
	if !strings.Contains(msg, "No issue tracker configured") {
		t.Errorf("expected 'No issue tracker configured' for exec error, got: %s", msg)
	}
}

func TestIssuesErrorMessage_GenericError(t *testing.T) {
	m := newTestModel()
	m.issuesErr = fmt.Errorf("connection timeout")
	msg := m.issuesErrorMessage()
	if !strings.Contains(msg, "Failed to load issues") {
		t.Errorf("expected 'Failed to load issues', got: %s", msg)
	}
	if !strings.Contains(msg, "connection timeout") {
		t.Errorf("expected error message in output, got: %s", msg)
	}
}

func TestRenderIssues_WithError(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issuesErr = beads.ErrNoBeadsDir

	output := m.renderIssues()
	if !strings.Contains(output, "No issue tracker configured") {
		t.Errorf("expected error message in output, got: %s", output)
	}
}

// --- renderIssues tests ---

func TestRenderIssues_Empty(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = nil

	output := m.renderIssues()
	if !strings.Contains(output, "No issues found") {
		t.Errorf("expected 'No issues found', got: %s", output)
	}
}

func TestRenderIssues_WithData(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Fix auth", Status: "open", Assignee: "eng-01"},
		{ID: "bd-002", Title: "Add tests", Status: "closed"},
	}

	output := m.renderIssues()
	if !strings.Contains(output, "bd-001") {
		t.Errorf("expected bd-001 in output, got: %s", output)
	}
	if !strings.Contains(output, "Fix auth") {
		t.Errorf("expected title in output, got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("expected assignee in output, got: %s", output)
	}
}

// --- issueSource tests ---

func TestIssueSource_WithAssignee(t *testing.T) {
	issue := beads.Issue{Assignee: "eng-01"}
	source := issueSource(issue)
	if source != "bd/eng-01" {
		t.Errorf("issueSource = %q, want 'bd/eng-01'", source)
	}
}

func TestIssueSource_NoAssignee(t *testing.T) {
	issue := beads.Issue{}
	source := issueSource(issue)
	if source != "bd" {
		t.Errorf("issueSource = %q, want 'bd'", source)
	}
}

// --- mapState tests ---

func TestMapState(t *testing.T) {
	tests := []struct {
		state agent.State
		want  string
	}{
		{agent.StateIdle, "info"},
		{agent.StateWorking, "ok"},
		{agent.StateDone, "ok"},
		{agent.StateStuck, "warning"},
		{agent.StateError, "error"},
		{agent.StateStopped, "stopped"},
		{agent.State("unknown"), ""},
	}

	for _, tt := range tests {
		got := mapState(tt.state)
		if got != tt.want {
			t.Errorf("mapState(%q) = %q, want %q", tt.state, got, tt.want)
		}
	}
}

// --- fmtDuration tests ---

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		want string
		d    time.Duration
	}{
		{"0s", 0},
		{"30s", 30 * time.Second},
		{"1m 30s", 90 * time.Second},
		{"1h 0m", time.Hour},
		{"1h 30m", 90 * time.Minute},
		{"2h 45m", 2*time.Hour + 45*time.Minute},
	}

	for _, tt := range tests {
		got := fmtDuration(tt.d)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// --- Tab navigation tests ---

func TestHandleKey_Tab(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents

	msg := tea.KeyMsg{Type: tea.KeyTab}
	m.HandleKey(msg)

	if m.tab != TabIssues {
		t.Errorf("tab = %d, want TabIssues", m.tab)
	}
}

func TestHandleKey_ShiftTab(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.HandleKey(msg)

	if m.tab != TabAgents {
		t.Errorf("tab = %d, want TabAgents", m.tab)
	}
}

func TestHandleKey_CursorDown(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 10)
	m.cursor = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	m.HandleKey(msg)

	if m.cursor != 1 {
		t.Errorf("cursor = %d, want 1", m.cursor)
	}
}

func TestHandleKey_CursorUp(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 10)
	m.cursor = 5

	msg := tea.KeyMsg{Type: tea.KeyUp}
	m.HandleKey(msg)

	if m.cursor != 4 {
		t.Errorf("cursor = %d, want 4", m.cursor)
	}
}

func TestHandleKey_Home(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 10)
	m.cursor = 5
	m.scrollOffset = 3

	msg := tea.KeyMsg{Type: tea.KeyHome}
	m.HandleKey(msg)

	if m.cursor != 0 {
		t.Errorf("cursor = %d, want 0", m.cursor)
	}
	if m.scrollOffset != 0 {
		t.Errorf("scrollOffset = %d, want 0", m.scrollOffset)
	}
}

func TestHandleKey_End(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 10)
	m.cursor = 0

	msg := tea.KeyMsg{Type: tea.KeyEnd}
	m.HandleKey(msg)

	if m.cursor != 9 {
		t.Errorf("cursor = %d, want 9", m.cursor)
	}
}

// --- selectCurrent tests ---

func TestSelectCurrent_Agents(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = []*agent.Agent{
		{Name: "eng-01"},
		{Name: "eng-02"},
	}
	m.cursor = 1

	action := m.selectCurrent()
	if action.Type != ActionDrillAgent {
		t.Errorf("action type = %d, want ActionDrillAgent", action.Type)
	}
	a, ok := action.Data.(*agent.Agent)
	if !ok {
		t.Fatal("expected *agent.Agent data")
	}
	if a.Name != "eng-02" {
		t.Errorf("selected agent = %q, want 'eng-02'", a.Name)
	}
}

func TestSelectCurrent_Issues(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = []beads.Issue{
		{ID: "bd-001"},
		{ID: "bd-002"},
	}
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionDrillIssue {
		t.Errorf("action type = %d, want ActionDrillIssue", action.Type)
	}
	issue, ok := action.Data.(beads.Issue)
	if !ok {
		t.Fatal("expected beads.Issue data")
	}
	if issue.ID != "bd-001" {
		t.Errorf("selected issue = %q, want 'bd-001'", issue.ID)
	}
}

func TestSelectCurrent_Channels(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	m.channels = []*channel.Channel{
		{Name: "all"},
		{Name: "eng"},
	}
	m.cursor = 1

	action := m.selectCurrent()
	if action.Type != ActionDrillChannel {
		t.Errorf("action type = %d, want ActionDrillChannel", action.Type)
	}
	ch, ok := action.Data.(*channel.Channel)
	if !ok {
		t.Fatal("expected *channel.Channel data")
	}
	if ch.Name != "eng" {
		t.Errorf("selected channel = %q, want 'eng'", ch.Name)
	}
}

func TestSelectCurrent_EmptyList(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = nil
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionNone {
		t.Errorf("action type = %d, want ActionNone for empty list", action.Type)
	}
}

// --- maxCursor tests ---

func TestMaxCursor_Agents(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = make([]*agent.Agent, 5)

	max := m.maxCursor()
	if max != 4 {
		t.Errorf("maxCursor = %d, want 4", max)
	}
}

func TestMaxCursor_Issues(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 10)

	max := m.maxCursor()
	if max != 9 {
		t.Errorf("maxCursor = %d, want 9", max)
	}
}

func TestMaxCursor_Empty(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = nil

	max := m.maxCursor()
	if max != 0 {
		t.Errorf("maxCursor = %d, want 0 for empty list", max)
	}
}

// --- clampCursor tests ---

func TestClampCursor(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 5)
	m.cursor = 10

	m.clampCursor()

	if m.cursor != 4 {
		t.Errorf("cursor = %d, want 4 after clamp", m.cursor)
	}
}

// --- renderTabBar tests ---

func TestRenderTabBar_Shows5Tabs(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateIdle},
		{Name: "eng-03", State: agent.StateStopped},
	}
	m.issues = make([]beads.Issue, 5)
	m.channels = make([]*channel.Channel, 2)
	m.computeStats()

	output := m.renderTabBar()

	tabs := []string{"Agents", "Issues", "Channels", "Dashboard", "Stats"}
	for _, tab := range tabs {
		if !strings.Contains(output, tab) {
			t.Errorf("expected '%s' tab in tab bar, got: %s", tab, output)
		}
	}
}

// --- computeStats tests ---

func TestComputeStats_IssueTypes(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Type: "epic", Status: "open"},
		{ID: "bd-002", Type: "task", Status: "open"},
		{ID: "bd-003", Type: "epic", Status: "closed"},
		{ID: "bd-004", Status: "pending"},
	}

	m.computeStats()

	if m.stats.TotalIssues != 4 {
		t.Errorf("TotalIssues = %d, want 4", m.stats.TotalIssues)
	}
	if m.stats.EpicsCount != 2 {
		t.Errorf("EpicsCount = %d, want 2", m.stats.EpicsCount)
	}
	if m.stats.OpenIssues != 3 { // open + pending
		t.Errorf("OpenIssues = %d, want 3", m.stats.OpenIssues)
	}
	if m.stats.ClosedIssues != 1 {
		t.Errorf("ClosedIssues = %d, want 1", m.stats.ClosedIssues)
	}
}

func TestComputeStats_AgentStates(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
		{Name: "eng-02", State: agent.StateWorking},
		{Name: "eng-03", State: agent.StateIdle},
		{Name: "eng-04", State: agent.StateStuck},
		{Name: "eng-05", State: agent.StateStopped},
	}

	m.computeStats()

	if m.stats.WorkingAgents != 2 {
		t.Errorf("WorkingAgents = %d, want 2", m.stats.WorkingAgents)
	}
	if m.stats.IdleAgents != 1 {
		t.Errorf("IdleAgents = %d, want 1", m.stats.IdleAgents)
	}
	if m.stats.StuckAgents != 1 {
		t.Errorf("StuckAgents = %d, want 1", m.stats.StuckAgents)
	}
	if m.stats.StoppedAgents != 1 {
		t.Errorf("StoppedAgents = %d, want 1", m.stats.StoppedAgents)
	}
}

// --- getRecentlyClosedIssues tests ---

func TestGetRecentlyClosedIssues_LimitTo5(t *testing.T) {
	m := newTestModel()
	for i := 0; i < 10; i++ {
		m.issues = append(m.issues, beads.Issue{
			ID:     fmt.Sprintf("bd-%03d", i+1),
			Status: "closed",
		})
	}

	closed := m.getRecentlyClosedIssues()
	if len(closed) != 5 {
		t.Errorf("len(closed) = %d, want 5", len(closed))
	}
}

func TestGetRecentlyClosedIssues_ReturnsNewest(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Status: "open"},
		{ID: "bd-002", Status: "closed"},
		{ID: "bd-003", Status: "closed"},
		{ID: "bd-004", Status: "closed"},
	}

	closed := m.getRecentlyClosedIssues()

	// Should be in reverse order (newest first)
	if closed[0].ID != "bd-004" {
		t.Errorf("expected bd-004 first, got %s", closed[0].ID)
	}
	if closed[len(closed)-1].ID != "bd-002" {
		t.Errorf("expected bd-002 last, got %s", closed[len(closed)-1].ID)
	}
}

// --- renderStats tests ---

func TestRenderStats_NoStats(t *testing.T) {
	m := newTestModel()
	m.tab = TabStats
	m.pkgStats = nil

	output := m.renderStats()
	if !strings.Contains(output, "No stats available") {
		t.Errorf("expected 'No stats available', got: %s", output)
	}
}

func TestRenderStats_WithData(t *testing.T) {
	m := newTestModel()
	m.tab = TabStats
	m.issues = []beads.Issue{
		{ID: "bd-001", Status: "open"},
		{ID: "bd-002", Status: "closed"},
	}
	m.pkgStats = &stats.Stats{
		Agents: stats.AgentMetrics{
			TotalAgents:  3,
			ActiveAgents: 2,
			Idle:         1,
			Working:      1,
		},
	}
	m.computeStats()

	output := m.renderStats()
	if !strings.Contains(output, "Workspace Overview") {
		t.Errorf("expected 'Workspace Overview', got: %s", output)
	}
	if !strings.Contains(output, "Issues:") {
		t.Errorf("expected 'Issues:' row, got: %s", output)
	}
	if !strings.Contains(output, "Agents:") {
		t.Errorf("expected 'Agents:' row, got: %s", output)
	}
}

// --- View tests ---

func TestView_ShowsTabBar(t *testing.T) {
	m := newTestModel()
	m.computeStats()

	output := m.View()
	if !strings.Contains(output, "Agents") {
		t.Errorf("expected tab bar with 'Agents', got: %s", output)
	}
}

func TestView_ShowsStatsBar(t *testing.T) {
	m := newTestModel()
	m.computeStats()

	output := m.View()
	if !strings.Contains(output, "Issues:") {
		t.Errorf("expected stats bar with 'Issues:', got: %s", output)
	}
}

// --- Queue Progress tests ---

func TestRenderDashboard_QueueProgress(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Ready task", Status: "open"},
		{ID: "bd-002", Title: "In progress task", Status: "in_progress"},
		{ID: "bd-003", Title: "Assigned task", Status: "open", Assignee: "eng-01"},
		{ID: "bd-004", Title: "Closed task", Status: "closed"},
	}
	m.computeStats()

	output := m.renderDashboard()

	// Should show queue progress section
	if !strings.Contains(output, "QUEUE PROGRESS") {
		t.Errorf("expected 'QUEUE PROGRESS' section header, got: %s", output)
	}
	// Should show in progress count (1 in_progress)
	if !strings.Contains(output, "In Progress") {
		t.Errorf("expected 'In Progress' in queue stats, got: %s", output)
	}
	// Should show assigned count
	if !strings.Contains(output, "Assigned") {
		t.Errorf("expected 'Assigned' in queue stats, got: %s", output)
	}
	// Should show total open count (open + pending + in_progress = 3)
	if !strings.Contains(output, "Total Open") {
		t.Errorf("expected 'Total Open' in queue stats, got: %s", output)
	}
}

func TestComputeStats_QueueStats(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Open task", Status: "open"},
		{ID: "bd-002", Title: "In progress task", Status: "in_progress"},
		{ID: "bd-003", Title: "Pending task", Status: "pending"},
		{ID: "bd-004", Title: "Assigned open", Status: "open", Assignee: "eng-01"},
		{ID: "bd-005", Title: "Assigned in progress", Status: "in_progress", Assignee: "eng-02"},
		{ID: "bd-006", Title: "Closed", Status: "closed"},
	}
	m.computeStats()

	// In progress count should be 2 (both in_progress issues)
	if m.stats.InProgressIssues != 2 {
		t.Errorf("InProgressIssues = %d, want 2", m.stats.InProgressIssues)
	}
	// Assigned count should be 2 (issues with assignee set)
	if m.stats.AssignedIssues != 2 {
		t.Errorf("AssignedIssues = %d, want 2", m.stats.AssignedIssues)
	}
	// Open count should be 5 (open + pending + in_progress)
	if m.stats.OpenIssues != 5 {
		t.Errorf("OpenIssues = %d, want 5", m.stats.OpenIssues)
	}
	// Closed count should be 1
	if m.stats.ClosedIssues != 1 {
		t.Errorf("ClosedIssues = %d, want 1", m.stats.ClosedIssues)
	}
}

func TestRenderDashboard_QueueProgressBar(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Open task", Status: "open"},
		{ID: "bd-002", Title: "In progress", Status: "in_progress"},
	}
	m.computeStats()

	output := m.renderDashboard()

	// Should show progress bar with percentage
	if !strings.Contains(output, "In Progress:") {
		t.Errorf("expected 'In Progress:' progress bar, got: %s", output)
	}
	// Should contain progress percentage (50% = 1 in progress / 2 open)
	if !strings.Contains(output, "50.0%") {
		t.Errorf("expected '50.0%%' in progress bar, got: %s", output)
	}
}
