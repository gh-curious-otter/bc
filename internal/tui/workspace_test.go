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
	"github.com/rpuneet/bc/pkg/queue"
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

	if !strings.Contains(output, "bd-002") {
		t.Errorf("expected bd-002 in recently closed, got: %s", output)
	}
	if !strings.Contains(output, "bd-003") {
		t.Errorf("expected bd-003 in recently closed, got: %s", output)
	}
	if !strings.Contains(output, "bd-004") {
		t.Errorf("expected bd-004 in recently closed, got: %s", output)
	}
	// Open issue should NOT appear in recently closed section
	if strings.Contains(output, "Open issue") {
		t.Errorf("open issue should not appear in recently closed section")
	}
}

func TestRenderDashboard_ActivityFeed(t *testing.T) {
	m := newTestModel()
	m.recentEvents = []events.Event{
		{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Type:      events.AgentSpawned,
			Agent:     "engineer-01",
			Message:   "spawned",
		},
		{
			Timestamp: time.Now().Add(-3 * time.Minute),
			Type:      events.WorkStarted,
			Agent:     "engineer-01",
			Message:   "started fixing auth",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Minute),
			Type:      events.AgentReport,
			Agent:     "engineer-01",
			Message:   "done with auth fix",
		},
	}
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "agent.spawned") {
		t.Errorf("expected event type in output, got: %s", output)
	}
	if !strings.Contains(output, "[engineer-01]") {
		t.Errorf("expected agent name in output, got: %s", output)
	}
	if !strings.Contains(output, "spawned") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "started fixing auth") {
		t.Errorf("expected work started message, got: %s", output)
	}
}

func TestRenderDashboard_ActivityFeedTruncatesLongMessages(t *testing.T) {
	m := newTestModel()
	longMsg := strings.Repeat("a", 100)
	m.recentEvents = []events.Event{
		{
			Timestamp: time.Now(),
			Type:      events.AgentReport,
			Agent:     "eng-01",
			Message:   longMsg,
		},
	}
	m.computeStats()

	output := m.renderDashboard()

	if strings.Contains(output, longMsg) {
		t.Error("long message should be truncated in activity feed")
	}
	if !strings.Contains(output, "...") {
		t.Errorf("truncated message should end with '...', got: %s", output)
	}
}

func TestRenderDashboard_ActivityFeedNoAgent(t *testing.T) {
	m := newTestModel()
	m.recentEvents = []events.Event{
		{
			Timestamp: time.Now(),
			Type:      events.QueueLoaded,
			Message:   "loaded 5 items",
		},
	}
	m.computeStats()

	output := m.renderDashboard()

	if !strings.Contains(output, "loaded 5 items") {
		t.Errorf("expected message in output, got: %s", output)
	}
	if !strings.Contains(output, "queue.loaded") {
		t.Errorf("expected event type in output, got: %s", output)
	}
}

// --- getRecentlyClosedIssues tests ---

func TestGetRecentlyClosedIssues_Empty(t *testing.T) {
	m := newTestModel()

	closed := m.getRecentlyClosedIssues()
	if len(closed) != 0 {
		t.Errorf("expected 0 closed issues, got %d", len(closed))
	}
}

func TestGetRecentlyClosedIssues_MaxLimit(t *testing.T) {
	m := newTestModel()
	// Create 10 closed issues
	for i := 0; i < 10; i++ {
		m.issues = append(m.issues, beads.Issue{
			ID:     fmt.Sprintf("bd-%03d", i+1),
			Title:  fmt.Sprintf("Issue %d", i+1),
			Status: "closed",
		})
	}

	closed := m.getRecentlyClosedIssues()
	if len(closed) != dashboardMaxClosedIssues {
		t.Errorf("expected %d closed issues (max), got %d", dashboardMaxClosedIssues, len(closed))
	}
}

func TestGetRecentlyClosedIssues_OnlyClosed(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Open", Status: "open"},
		{ID: "bd-002", Title: "In progress", Status: "in_progress"},
		{ID: "bd-003", Title: "Closed one", Status: "closed"},
		{ID: "bd-004", Title: "Done one", Status: "done"},
		{ID: "bd-005", Title: "Resolved one", Status: "resolved"},
		{ID: "bd-006", Title: "Pending", Status: "pending"},
	}

	closed := m.getRecentlyClosedIssues()
	if len(closed) != 3 {
		t.Errorf("expected 3 closed issues, got %d", len(closed))
	}
	for _, issue := range closed {
		if issue.Status != "closed" && issue.Status != "done" && issue.Status != "resolved" {
			t.Errorf("unexpected status %q in closed list", issue.Status)
		}
	}
}

// --- Tab rendering tests ---

func TestRenderTabBar_IncludesDashboard(t *testing.T) {
	m := newTestModel()
	m.computeStats()

	output := m.renderTabBar()

	if !strings.Contains(output, "Dashboard") {
		t.Errorf("tab bar should include 'Dashboard', got: %s", output)
	}
	if !strings.Contains(output, "Agents") {
		t.Errorf("tab bar should include 'Agents', got: %s", output)
	}
}

// --- Stats computation tests ---

func TestComputeStats_IssueBreakdown(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "1", Status: "open"},
		{ID: "2", Status: "pending"},
		{ID: "3", Status: "in_progress"},
		{ID: "4", Status: "closed"},
		{ID: "5", Status: "done"},
		{ID: "6", Status: "resolved"},
		{ID: "7", Status: "open", Type: "epic"},
	}
	m.computeStats()

	if m.stats.TotalIssues != 7 {
		t.Errorf("expected total 7, got %d", m.stats.TotalIssues)
	}
	if m.stats.OpenIssues != 4 { // open + pending + in_progress + epic(open)
		t.Errorf("expected 4 open, got %d", m.stats.OpenIssues)
	}
	if m.stats.ClosedIssues != 3 { // closed + done + resolved
		t.Errorf("expected 3 closed, got %d", m.stats.ClosedIssues)
	}
	if m.stats.EpicsCount != 1 {
		t.Errorf("expected 1 epic, got %d", m.stats.EpicsCount)
	}
}

// --- View integration test ---

func TestWorkspaceView_DashboardTab(t *testing.T) {
	m := newTestModel()
	m.tab = TabDashboard
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "Test issue", Status: "open"},
		{ID: "bd-002", Title: "Done issue", Status: "done"},
	}
	m.recentEvents = []events.Event{
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-01", Message: "testing"},
	}
	m.queueStats = queue.Stats{Total: 3, Pending: 1, Working: 1, Done: 1}
	m.computeStats()

	output := m.View()

	// Should show queue stats bar
	if !strings.Contains(output, "Queue:") {
		t.Errorf("expected queue stats, got: %s", output)
	}
	// Should show dashboard content
	if !strings.Contains(output, "Issue Overview") {
		t.Errorf("expected 'Issue Overview' in view, got: %s", output)
	}
	if !strings.Contains(output, "Activity Feed") {
		t.Errorf("expected 'Activity Feed' in view, got: %s", output)
	}
	if !strings.Contains(output, "Open: 1") {
		t.Errorf("expected 'Open: 1' in view, got: %s", output)
	}
}

func TestRenderDashboard_AllOpenZero(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "bd-001", Title: "All done", Status: "done"},
		{ID: "bd-002", Title: "Also done", Status: "closed"},
	}
	m.computeStats()

	output := m.renderDashboard()

	// With 0 open issues, the Open label should use success styling (green)
	if !strings.Contains(output, "Open: 0") {
		t.Errorf("expected 'Open: 0', got: %s", output)
	}
}

// --- Pagination / viewport tests ---

func TestVisibleRows(t *testing.T) {
	m := newTestModel()
	m.height = 40

	visible := m.visibleRows()
	expected := 40 - viewportOverhead
	if visible != expected {
		t.Errorf("visibleRows() = %d, want %d", visible, expected)
	}
}

func TestVisibleRows_SmallTerminal(t *testing.T) {
	m := newTestModel()
	m.height = 5 // smaller than overhead

	visible := m.visibleRows()
	if visible < 1 {
		t.Errorf("visibleRows() should be at least 1, got %d", visible)
	}
}

func TestEnsureCursorVisible_ScrollsDown(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 12 - 7 = 5
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 20)

	// Move cursor past the viewport
	m.cursor = 8
	m.scrollOffset = 0
	m.ensureCursorVisible()

	// scrollOffset should adjust so cursor is visible
	visible := m.visibleRows()
	if m.cursor < m.scrollOffset || m.cursor >= m.scrollOffset+visible {
		t.Errorf("cursor %d not visible in viewport [%d, %d)", m.cursor, m.scrollOffset, m.scrollOffset+visible)
	}
}

func TestEnsureCursorVisible_ScrollsUp(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 20)

	// Viewport is scrolled down, cursor moves up
	m.scrollOffset = 10
	m.cursor = 5
	m.ensureCursorVisible()

	if m.scrollOffset != 5 {
		t.Errorf("scrollOffset = %d, want 5", m.scrollOffset)
	}
}

func TestViewportRange(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.scrollOffset = 3

	start, end := m.viewportRange(20)
	if start != 3 {
		t.Errorf("start = %d, want 3", start)
	}
	if end != 8 {
		t.Errorf("end = %d, want 8", end)
	}
}

func TestViewportRange_ClampedToTotal(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.scrollOffset = 18

	start, end := m.viewportRange(20)
	if end != 20 {
		t.Errorf("end = %d, want 20 (clamped to total)", end)
	}
	if start != 18 {
		t.Errorf("start = %d, want 18", start)
	}
}

func TestViewportRange_Empty(t *testing.T) {
	m := newTestModel()
	start, end := m.viewportRange(0)
	if start != 0 || end != 0 {
		t.Errorf("viewportRange(0) = (%d, %d), want (0, 0)", start, end)
	}
}

func TestViewportRange_AllFit(t *testing.T) {
	m := newTestModel()
	m.height = 40 // visibleRows = 33
	m.scrollOffset = 0

	start, end := m.viewportRange(5)
	if start != 0 || end != 5 {
		t.Errorf("viewportRange(5) = (%d, %d), want (0, 5)", start, end)
	}
}

func TestPositionIndicator_NoIndicatorWhenAllFit(t *testing.T) {
	m := newTestModel()
	m.height = 40 // visibleRows = 33

	indicator := m.renderPositionIndicator(10) // 10 items < 33 visible
	if indicator != "" {
		t.Errorf("expected empty indicator when all items fit, got %q", indicator)
	}
}

func TestPositionIndicator_ShowsRange(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.scrollOffset = 10

	indicator := m.renderPositionIndicator(50)
	if !strings.Contains(indicator, "11-15 of 50") {
		t.Errorf("expected '11-15 of 50' in indicator, got %q", indicator)
	}
}

func TestRenderQueue_Paginated(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.tab = TabQueue
	m.scrollOffset = 0
	m.queueFilter = QueueFilterAll

	// Create 20 queue items
	for i := 0; i < 20; i++ {
		m.queueItems = append(m.queueItems, queue.WorkItem{
			ID:     fmt.Sprintf("work-%03d", i+1),
			Title:  fmt.Sprintf("Task %d", i+1),
			Status: queue.StatusDone,
		})
	}
	m.applyQueueFilter()

	output := m.renderQueue()

	// Should contain first 5 items (viewport)
	if !strings.Contains(output, "work-001") {
		t.Errorf("expected work-001 in viewport, got: %s", output)
	}
	if !strings.Contains(output, "work-005") {
		t.Errorf("expected work-005 in viewport, got: %s", output)
	}
	// Should NOT contain items outside viewport
	if strings.Contains(output, "work-006") {
		t.Errorf("work-006 should not be in viewport, got: %s", output)
	}
	// Should show position indicator
	if !strings.Contains(output, "1-5 of 20") {
		t.Errorf("expected position indicator '1-5 of 20', got: %s", output)
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
	m.tab = TabQueue
	m.issues = make([]beads.Issue, 20)
	m.queueItems = make([]queue.WorkItem, 20)

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

// --- Queue filter tests ---

func testQueueItems() []queue.WorkItem {
	return []queue.WorkItem{
		{ID: "work-001", Title: "Pending task", Status: queue.StatusPending},
		{ID: "work-002", Title: "Assigned task", Status: queue.StatusAssigned},
		{ID: "work-003", Title: "Working task", Status: queue.StatusWorking},
		{ID: "work-004", Title: "Done task", Status: queue.StatusDone},
		{ID: "work-005", Title: "Failed task", Status: queue.StatusFailed},
		{ID: "work-006", Title: "Another done", Status: queue.StatusDone},
	}
}

func TestQueueFilter_DefaultIsActive(t *testing.T) {
	m := newTestModel()
	m.queueItems = testQueueItems()
	m.applyQueueFilter()

	if m.queueFilter != QueueFilterActive {
		t.Errorf("default filter = %d, want QueueFilterActive", m.queueFilter)
	}
	// Active = everything except done (4 items)
	if len(m.filteredQueueItems) != 4 {
		t.Errorf("filtered count = %d, want 4 (active items)", len(m.filteredQueueItems))
	}
	for _, item := range m.filteredQueueItems {
		if item.Status == queue.StatusDone {
			t.Errorf("active filter should not include done items, found: %s", item.ID)
		}
	}
}

func TestQueueFilter_AllShowsEverything(t *testing.T) {
	m := newTestModel()
	m.queueItems = testQueueItems()
	m.queueFilter = QueueFilterAll
	m.applyQueueFilter()

	if len(m.filteredQueueItems) != len(m.queueItems) {
		t.Errorf("filtered count = %d, want %d", len(m.filteredQueueItems), len(m.queueItems))
	}
}

func TestQueueFilter_DoneShowsOnlyDone(t *testing.T) {
	m := newTestModel()
	m.queueItems = testQueueItems()
	m.queueFilter = QueueFilterDone
	m.applyQueueFilter()

	if len(m.filteredQueueItems) != 2 {
		t.Errorf("filtered count = %d, want 2 done items", len(m.filteredQueueItems))
	}
	for _, item := range m.filteredQueueItems {
		if item.Status != queue.StatusDone {
			t.Errorf("done filter should only include done items, found: %s (%s)", item.ID, item.Status)
		}
	}
}

func TestQueueFilter_CyclesCorrectly(t *testing.T) {
	f := QueueFilterActive
	f = f.next()
	if f != QueueFilterAll {
		t.Errorf("active.next() = %d, want QueueFilterAll", f)
	}
	f = f.next()
	if f != QueueFilterDone {
		t.Errorf("all.next() = %d, want QueueFilterDone", f)
	}
	f = f.next()
	if f != QueueFilterActive {
		t.Errorf("done.next() = %d, want QueueFilterActive", f)
	}
}

func TestQueueFilter_TabBarShowsFilterLabel(t *testing.T) {
	m := newTestModel()
	m.queueItems = testQueueItems()
	m.queueFilter = QueueFilterActive
	m.applyQueueFilter()
	m.computeStats()

	output := m.renderTabBar()
	if !strings.Contains(output, "active") {
		t.Errorf("tab bar should show 'active' filter label, got: %s", output)
	}

	// All filter should show plain count
	m.queueFilter = QueueFilterAll
	m.applyQueueFilter()
	output = m.renderTabBar()
	if strings.Contains(output, "active") || strings.Contains(output, "done") {
		t.Errorf("tab bar with All filter should not show filter label, got: %s", output)
	}
}

func TestQueueFilter_SelectCurrentUsesFilteredSlice(t *testing.T) {
	m := newTestModel()
	m.tab = TabQueue
	m.queueItems = testQueueItems()
	m.queueFilter = QueueFilterDone
	m.applyQueueFilter()
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionDrillQueue {
		t.Fatalf("expected ActionDrillQueue, got %d", action.Type)
	}

	item, ok := action.Data.(queue.WorkItem)
	if !ok {
		t.Fatal("expected WorkItem data")
	}
	if item.Status != queue.StatusDone {
		t.Errorf("selected item status = %s, want done (should select from filtered slice)", item.Status)
	}
}

func TestQueueFilter_MaxCursorUsesFilteredSlice(t *testing.T) {
	m := newTestModel()
	m.tab = TabQueue
	m.queueItems = testQueueItems()
	m.queueFilter = QueueFilterDone
	m.applyQueueFilter()

	max := m.maxCursor()
	if max != 1 { // 2 done items, max cursor = 1
		t.Errorf("maxCursor = %d, want 1 (2 done items)", max)
	}
}

func TestQueueFilter_EmptyFilterShowsMessage(t *testing.T) {
	m := newTestModel()
	m.tab = TabQueue
	// Items exist but all are done
	m.queueItems = []queue.WorkItem{
		{ID: "work-001", Title: "Done", Status: queue.StatusDone},
	}
	m.queueFilter = QueueFilterActive
	m.applyQueueFilter()

	output := m.renderQueue()
	if !strings.Contains(output, "Press 'f' to change filter") {
		t.Errorf("empty filtered queue should show filter hint, got: %s", output)
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
		{Name: "engineer-01", Role: agent.RoleEngineer, State: agent.StateWorking, Task: "fixing auth"},
		{Name: "qa-01", Role: agent.RoleQA, State: agent.StateIdle},
	}
	m.agentStats = map[string]stats.AgentStat{
		"engineer-01": {Name: "engineer-01", TasksCompleted: 3, TasksFailed: 1, Uptime: 2 * time.Hour},
		"qa-01":       {Name: "qa-01", TasksCompleted: 1, Uptime: 30 * time.Minute},
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
		{Name: "eng-01", Role: agent.RoleEngineer, State: agent.StateWorking},
		{Name: "eng-02", Role: agent.RoleEngineer, State: agent.StateIdle},
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
			Role:  agent.RoleEngineer,
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
		t.Errorf("expected error message in render, got: %s", output)
	}
}

func TestRenderIssues_Empty(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = nil
	m.issuesErr = nil

	output := m.renderIssues()
	if !strings.Contains(output, "No issues found") {
		t.Errorf("expected 'No issues found', got: %s", output)
	}
}

// --- renderChannels tests ---

func TestRenderChannels_NoChannels(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	m.channels = nil

	output := m.renderChannels()
	if !strings.Contains(output, "No channels") {
		t.Errorf("expected 'No channels' message, got: %s", output)
	}
}

func TestRenderChannels_WithChannels(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	m.channels = []*channel.Channel{
		{
			Name:    "standup",
			Members: []string{"coordinator", "eng-01", "eng-02"},
			History: []channel.HistoryEntry{
				{Sender: "eng-01", Message: "working on auth fix"},
			},
		},
		{
			Name:    "engineering",
			Members: []string{"manager", "eng-01"},
			History: nil,
		},
	}

	output := m.renderChannels()
	if !strings.Contains(output, "#standup") {
		t.Errorf("expected '#standup' in output, got: %s", output)
	}
	if !strings.Contains(output, "#engineering") {
		t.Errorf("expected '#engineering' in output, got: %s", output)
	}
	if !strings.Contains(output, "working on auth fix") {
		t.Errorf("expected last message in output, got: %s", output)
	}
	if !strings.Contains(output, "CHANNEL") {
		t.Errorf("expected header row, got: %s", output)
	}
}

func TestRenderChannels_LongLastMessage(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	longMsg := strings.Repeat("x", 100)
	m.channels = []*channel.Channel{
		{
			Name:    "test",
			Members: []string{"a"},
			History: []channel.HistoryEntry{
				{Sender: "a", Message: longMsg},
			},
		},
	}

	output := m.renderChannels()
	if strings.Contains(output, longMsg) {
		t.Error("long message should be truncated")
	}
	if !strings.Contains(output, "...") {
		t.Error("truncated message should contain '...'")
	}
}

func TestRenderChannels_Paginated(t *testing.T) {
	m := newTestModel()
	m.height = 12 // visibleRows = 5
	m.tab = TabChannels
	for i := 0; i < 15; i++ {
		m.channels = append(m.channels, &channel.Channel{
			Name:    fmt.Sprintf("chan-%02d", i+1),
			Members: []string{"a"},
		})
	}

	output := m.renderChannels()
	if !strings.Contains(output, "chan-01") {
		t.Errorf("expected chan-01 in viewport")
	}
	if strings.Contains(output, "chan-10") {
		t.Errorf("chan-10 should not be in viewport")
	}
	if !strings.Contains(output, "of 15") {
		t.Errorf("expected position indicator, got: %s", output)
	}
}

// --- queueStatusRank tests ---

func TestQueueStatusRank(t *testing.T) {
	tests := []struct {
		status queue.ItemStatus
		rank   int
	}{
		{queue.StatusWorking, 0},
		{queue.StatusAssigned, 1},
		{queue.StatusPending, 2},
		{queue.StatusFailed, 3},
		{queue.StatusDone, 4},
		{queue.ItemStatus("unknown"), 5},
	}

	for _, tt := range tests {
		got := queueStatusRank(tt.status)
		if got != tt.rank {
			t.Errorf("queueStatusRank(%q) = %d, want %d", tt.status, got, tt.rank)
		}
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

// --- mapQueueStatus tests ---

func TestMapQueueStatus(t *testing.T) {
	tests := []struct {
		status queue.ItemStatus
		want   string
	}{
		{queue.StatusPending, "pending"},
		{queue.StatusAssigned, "queued"},
		{queue.StatusWorking, "running"},
		{queue.StatusDone, "success"},
		{queue.StatusFailed, "failed"},
		{queue.ItemStatus("unknown"), ""},
	}

	for _, tt := range tests {
		got := mapQueueStatus(tt.status)
		if got != tt.want {
			t.Errorf("mapQueueStatus(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

// --- fmtDuration tests ---

func TestFmtDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{5 * time.Minute, "5m 0s"},
		{5*time.Minute + 30*time.Second, "5m 30s"},
		{2 * time.Hour, "2h 0m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}

	for _, tt := range tests {
		got := fmtDuration(tt.d)
		if got != tt.want {
			t.Errorf("fmtDuration(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// --- selectCurrent tests ---

func TestSelectCurrent_Agents(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	a := &agent.Agent{Name: "eng-01", Role: agent.RoleEngineer}
	m.agents = []*agent.Agent{a}
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionDrillAgent {
		t.Fatalf("expected ActionDrillAgent, got %d", action.Type)
	}
	if action.Data.(*agent.Agent).Name != "eng-01" {
		t.Error("wrong agent returned")
	}
}

func TestSelectCurrent_Issues(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = []beads.Issue{{ID: "bd-001", Title: "Test"}}
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionDrillIssue {
		t.Fatalf("expected ActionDrillIssue, got %d", action.Type)
	}
	issue := action.Data.(beads.Issue)
	if issue.ID != "bd-001" {
		t.Errorf("wrong issue: %s", issue.ID)
	}
}

func TestSelectCurrent_Channels(t *testing.T) {
	m := newTestModel()
	m.tab = TabChannels
	ch := &channel.Channel{Name: "standup"}
	m.channels = []*channel.Channel{ch}
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionDrillChannel {
		t.Fatalf("expected ActionDrillChannel, got %d", action.Type)
	}
}

func TestSelectCurrent_OutOfBounds(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = nil
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionNone {
		t.Errorf("expected NoAction for empty list, got %d", action.Type)
	}
}

func TestSelectCurrent_Dashboard(t *testing.T) {
	m := newTestModel()
	m.tab = TabDashboard
	m.cursor = 0

	action := m.selectCurrent()
	if action.Type != ActionNone {
		t.Errorf("dashboard selectCurrent should return NoAction, got %d", action.Type)
	}
}

// --- maxCursor tests for all tabs ---

func TestMaxCursor_AllTabs(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{{}, {}, {}}
	m.issues = []beads.Issue{{}, {}}
	m.channels = []*channel.Channel{{}, {}, {}, {}}
	m.queueItems = testQueueItems()
	m.queueFilter = QueueFilterAll
	m.applyQueueFilter()

	tests := []struct {
		tab  Tab
		want int
	}{
		{TabAgents, 2},
		{TabIssues, 1},
		{TabChannels, 3},
		{TabQueue, 5},
		{TabDashboard, 0},
		{TabStats, 0},
	}

	for _, tt := range tests {
		m.tab = tt.tab
		got := m.maxCursor()
		if got != tt.want {
			t.Errorf("maxCursor(tab=%d) = %d, want %d", tt.tab, got, tt.want)
		}
	}
}

// --- HandleKey tests ---

func TestHandleKey_TabCycle(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents

	// Tab forward through all tabs
	for i := 0; i < tabCount; i++ {
		msg := tea.KeyMsg{Type: tea.KeyTab}
		m.HandleKey(msg)
	}
	// Should be back at TabAgents after full cycle
	if m.tab != TabAgents {
		t.Errorf("after full tab cycle, tab = %d, want %d", m.tab, TabAgents)
	}
}

func TestHandleKey_ShiftTab(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents

	msg := tea.KeyMsg{Type: tea.KeyShiftTab}
	m.HandleKey(msg)
	if m.tab != TabStats {
		t.Errorf("shift+tab from Agents should go to Stats, got %d", m.tab)
	}
}

func TestHandleKey_HomeEnd(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = make([]beads.Issue, 20)
	m.cursor = 5

	// Home key
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.cursor != 0 {
		t.Errorf("g key should go to cursor 0, got %d", m.cursor)
	}

	// End key
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.cursor != 19 {
		t.Errorf("G key should go to cursor 19, got %d", m.cursor)
	}
}

func TestHandleKey_FKeyOnQueueTab(t *testing.T) {
	m := newTestModel()
	m.tab = TabQueue
	m.queueItems = testQueueItems()
	m.applyQueueFilter()
	m.cursor = 2

	// Press f to cycle filter
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if m.queueFilter != QueueFilterAll {
		t.Errorf("expected QueueFilterAll after f, got %d", m.queueFilter)
	}
	if m.cursor != 0 {
		t.Errorf("cursor should reset to 0 after filter change, got %d", m.cursor)
	}
}

func TestHandleKey_FKeyOnNonQueueTab(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	initialFilter := m.queueFilter

	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if m.queueFilter != initialFilter {
		t.Errorf("f key on non-queue tab should not change filter")
	}
}

func TestHandleKey_EnterDrillsDown(t *testing.T) {
	m := newTestModel()
	m.tab = TabAgents
	m.agents = []*agent.Agent{{Name: "eng-01"}}
	m.cursor = 0

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if action.Type != ActionDrillAgent {
		t.Errorf("enter on agents tab should drill down, got %d", action.Type)
	}
}

// --- renderStatsBar tests ---

func TestRenderStatsBar(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{{}, {}}
	m.stats = WorkspaceStats{
		TotalIssues:   10,
		OpenIssues:    7,
		ClosedIssues:  3,
		WorkingAgents: 1,
		IdleAgents:    1,
	}
	m.queueStats = queue.Stats{Total: 5, Pending: 2, Working: 1, Done: 2}

	output := m.renderStatsBar()
	if !strings.Contains(output, "Issues: 10") {
		t.Errorf("expected issue stats, got: %s", output)
	}
	if !strings.Contains(output, "Queue:") {
		t.Errorf("expected queue stats, got: %s", output)
	}
}

// --- renderStatsPanel tests ---

func TestRenderStatsPanel(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "eng-01", Role: agent.RoleEngineer, State: agent.StateWorking},
	}
	m.agentStats = map[string]stats.AgentStat{
		"eng-01": {Name: "eng-01", State: "working", TasksCompleted: 5, TasksFailed: 1, Uptime: time.Hour},
	}
	m.stats = WorkspaceStats{WorkingAgents: 1}
	m.queueStats = queue.Stats{Total: 10, Pending: 2, Working: 3, Done: 4, Failed: 1}

	output := m.renderStatsPanel()
	if !strings.Contains(output, "Work Items") {
		t.Errorf("expected 'Work Items' section, got: %s", output)
	}
	if !strings.Contains(output, "Agents") {
		t.Errorf("expected 'Agents' section, got: %s", output)
	}
	if !strings.Contains(output, "Per-Agent Stats") {
		t.Errorf("expected 'Per-Agent Stats' section, got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("expected agent name in per-agent stats")
	}
}

func TestRenderStatsPanel_NoAgentStats(t *testing.T) {
	m := newTestModel()
	m.agentStats = map[string]stats.AgentStat{}
	m.queueStats = queue.Stats{}
	m.stats = WorkspaceStats{}

	output := m.renderStatsPanel()
	if !strings.Contains(output, "No agent stats") {
		t.Errorf("expected 'No agent stats' message, got: %s", output)
	}
}

// --- renderStats tests ---

func TestRenderStats_NoPkgStats(t *testing.T) {
	m := newTestModel()
	m.pkgStats = nil

	output := m.renderStats()
	if !strings.Contains(output, "No stats available") {
		t.Errorf("expected 'No stats available', got: %s", output)
	}
}

func TestRenderStats_WithPkgStats(t *testing.T) {
	m := newTestModel()
	m.issues = []beads.Issue{
		{ID: "1", Status: "open", Assignee: "eng-01"},
		{ID: "2", Status: "closed"},
	}
	m.agents = []*agent.Agent{
		{Name: "eng-01", State: agent.StateWorking},
	}
	m.computeStats()
	m.pkgStats = &stats.Stats{
		WorkItems: stats.WorkItemMetrics{
			Total:          10,
			Pending:        2,
			Assigned:       1,
			Working:        3,
			Done:           3,
			Failed:         1,
			CompletionRate: 0.3,
		},
		Agents: stats.AgentMetrics{
			TotalAgents:  5,
			ActiveAgents: 3,
			Idle:         1,
			Stuck:        0,
			Stopped:      1,
			AgentStats: []stats.AgentStat{
				{Name: "eng-01", Role: "engineer", State: "working", TasksCompleted: 3, TasksFailed: 0, Uptime: time.Hour},
			},
		},
	}

	output := m.renderStats()
	if !strings.Contains(output, "Workspace Overview") {
		t.Errorf("expected 'Workspace Overview', got: %s", output)
	}
	if !strings.Contains(output, "Per-Agent Metrics") {
		t.Errorf("expected 'Per-Agent Metrics', got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("expected agent in per-agent metrics, got: %s", output)
	}
	if !strings.Contains(output, "Completion Rate") {
		t.Errorf("expected 'Completion Rate', got: %s", output)
	}
}

func TestRenderStats_NoAgentData(t *testing.T) {
	m := newTestModel()
	m.computeStats()
	m.pkgStats = &stats.Stats{
		WorkItems: stats.WorkItemMetrics{Total: 1},
		Agents:    stats.AgentMetrics{AgentStats: nil},
	}

	output := m.renderStats()
	if !strings.Contains(output, "No agent data") {
		t.Errorf("expected 'No agent data', got: %s", output)
	}
}

// --- renderQueue tests ---

func TestRenderQueue_Empty(t *testing.T) {
	m := newTestModel()
	m.tab = TabQueue
	m.queueItems = nil
	m.queueFilter = QueueFilterAll
	m.applyQueueFilter()

	output := m.renderQueue()
	if !strings.Contains(output, "No queue items") {
		t.Errorf("expected 'No queue items', got: %s", output)
	}
}

func TestRenderQueue_WithItems(t *testing.T) {
	m := newTestModel()
	m.tab = TabQueue
	m.queueItems = []queue.WorkItem{
		{ID: "work-001", Title: "Fix auth", Status: queue.StatusWorking, AssignedTo: "eng-01", BeadsID: "bd-001"},
		{ID: "work-002", Title: "Add tests", Status: queue.StatusPending},
	}
	m.queueFilter = QueueFilterAll
	m.applyQueueFilter()

	output := m.renderQueue()
	if !strings.Contains(output, "work-001") {
		t.Errorf("expected work-001, got: %s", output)
	}
	if !strings.Contains(output, "Fix auth") {
		t.Errorf("expected title, got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("expected assignee, got: %s", output)
	}
}

// --- QueueFilter label tests ---

func TestQueueFilterLabel(t *testing.T) {
	tests := []struct {
		f    QueueFilter
		want string
	}{
		{QueueFilterActive, "active"},
		{QueueFilterAll, "all"},
		{QueueFilterDone, "done"},
		{QueueFilter(99), ""},
	}
	for _, tt := range tests {
		got := tt.f.label()
		if got != tt.want {
			t.Errorf("QueueFilter(%d).label() = %q, want %q", tt.f, got, tt.want)
		}
	}
}

// --- issueSource tests ---

func TestIssueSource(t *testing.T) {
	issue := beads.Issue{Assignee: "eng-01"}
	if got := issueSource(issue); got != "bd/eng-01" {
		t.Errorf("issueSource with assignee = %q, want 'bd/eng-01'", got)
	}
	issue.Assignee = ""
	if got := issueSource(issue); got != "bd" {
		t.Errorf("issueSource without assignee = %q, want 'bd'", got)
	}
}

// --- View integration ---

func TestView_AllTabs(t *testing.T) {
	tabs := []Tab{TabAgents, TabIssues, TabQueue, TabChannels, TabDashboard, TabStats}
	for _, tab := range tabs {
		m := newTestModel()
		m.tab = tab
		m.computeStats()
		m.agentStats = map[string]stats.AgentStat{}
		m.queueFilter = QueueFilterAll
		m.applyQueueFilter()

		output := m.View()
		if output == "" {
			t.Errorf("View() for tab %d returned empty string", tab)
		}
	}
}

// --- clampCursor test ---

func TestClampCursor(t *testing.T) {
	m := newTestModel()
	m.tab = TabIssues
	m.issues = []beads.Issue{{ID: "1"}, {ID: "2"}}
	m.cursor = 10

	m.clampCursor()
	if m.cursor != 1 {
		t.Errorf("cursor should be clamped to 1, got %d", m.cursor)
	}
}

// --- computeStats agent state tests ---

func TestComputeStats_AgentStates(t *testing.T) {
	m := newTestModel()
	m.agents = []*agent.Agent{
		{Name: "a", State: agent.StateIdle},
		{Name: "b", State: agent.StateWorking},
		{Name: "c", State: agent.StateStuck},
		{Name: "d", State: agent.StateStopped},
		{Name: "e", State: agent.StateWorking},
	}
	m.computeStats()

	if m.stats.IdleAgents != 1 {
		t.Errorf("IdleAgents = %d, want 1", m.stats.IdleAgents)
	}
	if m.stats.WorkingAgents != 2 {
		t.Errorf("WorkingAgents = %d, want 2", m.stats.WorkingAgents)
	}
	if m.stats.StuckAgents != 1 {
		t.Errorf("StuckAgents = %d, want 1", m.stats.StuckAgents)
	}
	if m.stats.StoppedAgents != 1 {
		t.Errorf("StoppedAgents = %d, want 1", m.stats.StoppedAgents)
	}
}
