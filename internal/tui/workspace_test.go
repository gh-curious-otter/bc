package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
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
