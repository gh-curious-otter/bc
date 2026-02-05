package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

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
