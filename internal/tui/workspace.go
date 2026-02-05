package tui

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/stats"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// Tab identifies which tab is active in the workspace view.
type Tab int

const (
	TabAgents Tab = iota
	TabIssues
	TabQueue
	TabChannels
	TabDashboard
	TabStats

	tabCount = 6
)

// QueueFilter controls which queue items are displayed.
type QueueFilter int

const (
	// QueueFilterActive shows pending, assigned, working, and failed items.
	QueueFilterActive QueueFilter = iota
	// QueueFilterAll shows all items including done.
	QueueFilterAll
	// QueueFilterDone shows only done items.
	QueueFilterDone
)

func (f QueueFilter) label() string {
	switch f {
	case QueueFilterActive:
		return "active"
	case QueueFilterAll:
		return "all"
	case QueueFilterDone:
		return "done"
	default:
		return ""
	}
}

func (f QueueFilter) next() QueueFilter {
	return (f + 1) % 3
}

// WorkspaceStats holds aggregated statistics for the workspace.
type WorkspaceStats struct {
	// Issue stats
	TotalIssues  int
	OpenIssues   int
	ClosedIssues int
	EpicsCount   int

	// Agent stats by state
	IdleAgents    int
	WorkingAgents int
	StuckAgents   int
	StoppedAgents int
}

// WorkspaceModel shows the detail view for a single workspace.
type WorkspaceModel struct {
	info         WorkspaceInfo
	styles       style.Styles
	width        int
	height       int
	tab          Tab
	cursor       int
	scrollOffset int // first visible item index for current tab
	manager      *agent.Manager

	// Data
	agents             []*agent.Agent
	issues             []beads.Issue
	issuesErr          error
	channels           []*channel.Channel
	queueItems         []queue.WorkItem
	filteredQueueItems []queue.WorkItem
	queueFilter        QueueFilter

	// Queue stats
	queueStats queue.Stats

	// Per-agent stats from pkg/stats
	agentStats map[string]stats.AgentStat
	// Dashboard stats
	stats        WorkspaceStats
	pkgStats     *stats.Stats
	recentEvents []events.Event

	// Loaded flags
	agentsLoaded   bool
	issuesLoaded   bool
	channelsLoaded bool
	queueLoaded    bool
}

// NewWorkspaceModel creates a workspace detail view.
func NewWorkspaceModel(info WorkspaceInfo, s style.Styles) *WorkspaceModel {
	m := &WorkspaceModel{
		info:        info,
		styles:      s,
		tab:         TabAgents,
		queueFilter: QueueFilterActive,
	}

	// Load agent data
	m.manager = agent.NewWorkspaceManager(
		info.Entry.Path+"/.bc/agents",
		info.Entry.Path,
	)
	m.manager.LoadState()
	m.manager.RefreshState()
	m.agents = m.manager.ListAgents()
	m.agentsLoaded = true

	// Load beads issues
	m.issues, m.issuesErr = beads.ListIssues(info.Entry.Path)
	m.issuesLoaded = true

	// Load channels
	m.loadChannels()

	// Load queue stats and items
	m.loadQueueStats()
	m.loadQueueItems()

	// Load per-agent stats from pkg/stats
	m.loadAgentStats()

	// Load recent events for activity feed
	m.loadRecentEvents()

	// Compute dashboard stats
	m.computeStats()
	m.loadPkgStats()

	return m
}

// HandleKey processes a key event and returns an action for the parent.
func (m *WorkspaceModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()

	switch key {
	case "tab":
		m.tab = (m.tab + 1) % tabCount
		m.cursor = 0
		m.scrollOffset = 0
		return NoAction
	case "shift+tab":
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.cursor = 0
		m.scrollOffset = 0
		return NoAction
	case "j", "down":
		m.cursor++
		m.clampCursor()
		m.ensureCursorVisible()
		return NoAction
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		m.ensureCursorVisible()
		return NoAction
	case "g", "home":
		m.cursor = 0
		m.scrollOffset = 0
		return NoAction
	case "G", "end":
		m.cursor = m.maxCursor()
		m.ensureCursorVisible()
		return NoAction
	case "r":
		m.refresh()
		return NoAction
	case "f":
		if m.tab == TabQueue {
			m.queueFilter = m.queueFilter.next()
			m.applyQueueFilter()
			m.cursor = 0
			return NoAction
		}
	}
	if isEnterKey(msg) {
		return m.selectCurrent()
	}

	return NoAction
}

func (m *WorkspaceModel) refresh() {
	m.manager.RefreshState()
	m.agents = m.manager.ListAgents()
	m.issues, m.issuesErr = beads.ListIssues(m.info.Entry.Path)
	m.loadChannels()
	m.loadQueueStats()
	m.loadQueueItems()
	m.loadRecentEvents()
	m.computeStats()
	m.loadPkgStats()
	m.clampCursor()
	m.ensureCursorVisible()
}

func (m *WorkspaceModel) loadQueueStats() {
	q := queue.New(filepath.Join(m.info.Entry.Path, ".bc", "queue.json"))
	if err := q.Load(); err == nil {
		m.queueStats = q.Stats()
	}
}

func (m *WorkspaceModel) loadQueueItems() {
	q := queue.New(filepath.Join(m.info.Entry.Path, ".bc", "queue.json"))
	if err := q.Load(); err == nil {
		m.queueItems = q.ListAll()
	}
	// Sort in place so display order matches selectCurrent() indexing.
	sort.SliceStable(m.queueItems, func(i, j int) bool {
		ri, rj := queueStatusRank(m.queueItems[i].Status), queueStatusRank(m.queueItems[j].Status)
		if ri != rj {
			return ri < rj
		}
		if m.queueItems[i].Status == queue.StatusDone {
			return m.queueItems[i].ID > m.queueItems[j].ID
		}
		return m.queueItems[i].ID < m.queueItems[j].ID
	})
	m.queueLoaded = true
	m.applyQueueFilter()
}

func (m *WorkspaceModel) applyQueueFilter() {
	switch m.queueFilter {
	case QueueFilterAll:
		m.filteredQueueItems = m.queueItems
	case QueueFilterDone:
		filtered := make([]queue.WorkItem, 0)
		for _, item := range m.queueItems {
			if item.Status == queue.StatusDone {
				filtered = append(filtered, item)
			}
		}
		m.filteredQueueItems = filtered
	default: // QueueFilterActive
		filtered := make([]queue.WorkItem, 0)
		for _, item := range m.queueItems {
			if item.Status != queue.StatusDone {
				filtered = append(filtered, item)
			}
		}
		m.filteredQueueItems = filtered
	}
}

func (m *WorkspaceModel) loadAgentStats() {
	m.agentStats = make(map[string]stats.AgentStat)
	stateDir := filepath.Join(m.info.Entry.Path, ".bc")
	s, err := stats.Load(stateDir)
	if err != nil {
		return
	}
	for _, as := range s.Agents.AgentStats {
		m.agentStats[as.Name] = as
	}
}

func (m *WorkspaceModel) selectCurrent() Action {
	switch m.tab {
	case TabAgents:
		if m.cursor < len(m.agents) {
			return Action{Type: ActionDrillAgent, Data: m.agents[m.cursor]}
		}
	case TabIssues:
		if m.cursor < len(m.issues) {
			return Action{Type: ActionDrillIssue, Data: m.issues[m.cursor]}
		}
	case TabChannels:
		if m.cursor < len(m.channels) {
			return Action{Type: ActionDrillChannel, Data: m.channels[m.cursor]}
		}
	case TabQueue:
		if m.cursor < len(m.filteredQueueItems) {
			return Action{Type: ActionDrillQueue, Data: m.filteredQueueItems[m.cursor]}
		}
	}
	return NoAction
}

func (m *WorkspaceModel) maxCursor() int {
	switch m.tab {
	case TabAgents:
		if len(m.agents) > 0 {
			return len(m.agents) - 1
		}
	case TabIssues:
		if len(m.issues) > 0 {
			return len(m.issues) - 1
		}
	case TabChannels:
		if len(m.channels) > 0 {
			return len(m.channels) - 1
		}
	case TabQueue:
		if len(m.filteredQueueItems) > 0 {
			return len(m.filteredQueueItems) - 1
		}
	case TabDashboard:
		return 0
	case TabStats:
		return 0
	}
	return 0
}

func (m *WorkspaceModel) clampCursor() {
	max := m.maxCursor()
	if m.cursor > max {
		m.cursor = max
	}
}

// visibleRows returns the number of data rows that fit in the viewport.
// Accounts for overhead: stats bar (1) + gap (1) + tab bar (1) + gap (2) + header (1) + position indicator (1) = 7.
const viewportOverhead = 7

func (m *WorkspaceModel) visibleRows() int {
	rows := m.height - viewportOverhead
	if rows < 1 {
		rows = 1
	}
	return rows
}

// ensureCursorVisible adjusts scrollOffset so the cursor is within the viewport.
func (m *WorkspaceModel) ensureCursorVisible() {
	visible := m.visibleRows()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

// viewportRange returns the [start, end) range of items to render for the current viewport.
func (m *WorkspaceModel) viewportRange(total int) (int, int) {
	if total == 0 {
		return 0, 0
	}
	visible := m.visibleRows()
	start := m.scrollOffset
	if start > total {
		start = total
	}
	end := start + visible
	if end > total {
		end = total
	}
	return start, end
}

// renderPositionIndicator returns a "start-end of total" indicator string.
func (m *WorkspaceModel) renderPositionIndicator(total int) string {
	if total == 0 {
		return ""
	}
	visible := m.visibleRows()
	if total <= visible {
		return "" // everything fits, no indicator needed
	}
	start, end := m.viewportRange(total)
	return m.styles.Muted.Render(fmt.Sprintf("  %d-%d of %d", start+1, end, total)) + "\n"
}

// View renders the workspace detail screen.
func (m *WorkspaceModel) View() string {
	var b strings.Builder

	// Workspace summary stats bar (above tab bar)
	b.WriteString(m.renderStatsBar())
	b.WriteString("\n")

	// Tab bar
	b.WriteString(m.renderTabBar())
	b.WriteString("\n\n")

	// Content
	switch m.tab {
	case TabAgents:
		b.WriteString(m.renderAgents())
	case TabIssues:
		b.WriteString(m.renderIssues())
	case TabChannels:
		b.WriteString(m.renderChannels())
	case TabQueue:
		b.WriteString(m.renderQueue())
	case TabDashboard:
		b.WriteString(m.renderDashboard())
	case TabStats:
		b.WriteString(m.renderStats())
	}

	return b.String()
}

func (m *WorkspaceModel) renderTabBar() string {
	tabs := []struct {
		label string
		tab   Tab
		count int
	}{
		{"Agents", TabAgents, len(m.agents)},
		{"Issues", TabIssues, len(m.issues)},
		{"Queue", TabQueue, len(m.filteredQueueItems)},
		{"Channels", TabChannels, len(m.channels)},
		{"Dashboard", TabDashboard, m.stats.OpenIssues},
		{"Stats", TabStats, -1},
	}

	var parts []string
	for _, t := range tabs {
		var label string
		if t.tab == TabQueue && m.queueFilter != QueueFilterAll {
			label = fmt.Sprintf(" %s (%d %s) ", t.label, t.count, m.queueFilter.label())
		} else if t.count >= 0 {
			label = fmt.Sprintf(" %s (%d) ", t.label, t.count)
		} else {
			label = fmt.Sprintf(" %s ", t.label)
		}
		if t.tab == m.tab {
			parts = append(parts, m.styles.Header.Render(label))
		} else {
			parts = append(parts, m.styles.Muted.Render(label))
		}
	}

	return strings.Join(parts, " ")
}

func (m *WorkspaceModel) renderAgents() string {
	var b strings.Builder

	if len(m.agents) == 0 {
		b.WriteString(m.styles.Muted.Render("  No agents. Run 'bc up' to start agents."))
		b.WriteString("\n")
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-15s %-12s %-10s %-12s %-5s %-5s %s",
		"NAME", "ROLE", "STATE", "UPTIME", "DONE", "FAIL", "TASK")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	// Fixed columns: 2(indent) + NAME(15) + ROLE(12) + STATE(10) + UPTIME(12) + DONE(5) + FAIL(5) = 61
	// Task gets the rest of the terminal width
	taskWidth := m.width - 61
	if taskWidth < 20 {
		taskWidth = 20
	}

	// Render only the visible viewport.
	start, end := m.viewportRange(len(m.agents))
	for i := start; i < end; i++ {
		a := m.agents[i]
		selected := i == m.cursor

		as := m.agentStats[a.Name]
		uptime := "-"
		if as.Uptime > 0 {
			uptime = fmtDuration(as.Uptime)
		} else if a.State != agent.StateStopped && !a.StartedAt.IsZero() {
			uptime = fmtDuration(time.Since(a.StartedAt))
		}
		done := as.TasksCompleted
		failed := as.TasksFailed

		task := a.Task
		if task == "" {
			task = "-"
		}
		if len(task) > taskWidth {
			task = task[:taskWidth-3] + "..."
		}

		line := fmt.Sprintf("  %-15s %-12s %-10s %-12s %-5d %-5d %s",
			a.Name, a.Role, a.State, uptime, done, failed, task,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.StatusStyle(mapState(a.State)).Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.renderPositionIndicator(len(m.agents)))
	return b.String()
}

func (m *WorkspaceModel) renderIssues() string {
	var b strings.Builder

	if m.issuesErr != nil {
		msg := m.issuesErrorMessage()
		b.WriteString(m.styles.Muted.Render("  " + msg))
		b.WriteString("\n")
		return b.String()
	}

	if len(m.issues) == 0 {
		b.WriteString(m.styles.Muted.Render("  No issues found."))
		b.WriteString("\n")
		return b.String()
	}

	header := fmt.Sprintf("  %-12s %-10s %-15s %-15s %s", "ID", "STATUS", "ASSIGNED", "SOURCE", "TITLE")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	start, end := m.viewportRange(len(m.issues))
	for i := start; i < end; i++ {
		issue := m.issues[i]
		selected := i == m.cursor

		title := issue.Title
		if len(title) > 38 {
			title = title[:35] + "..."
		}

		source := issueSource(issue)

		assignee := issue.Assignee
		if assignee == "" {
			assignee = "-"
		}

		line := fmt.Sprintf("  %-12s %-10s %-15s %-15s %s",
			issue.ID, issue.Status, assignee, source, title,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.renderPositionIndicator(len(m.issues)))
	return b.String()
}

// issuesErrorMessage returns a user-facing message for the issues loading error.
func (m *WorkspaceModel) issuesErrorMessage() string {
	if errors.Is(m.issuesErr, beads.ErrNoBeadsDir) {
		return "No issue tracker configured. Run bd init to set up beads."
	}
	var execErr *exec.Error
	if errors.As(m.issuesErr, &execErr) {
		return "No issue tracker configured. Run bd init to set up beads."
	}
	return "Failed to load issues: " + m.issuesErr.Error()
}

// issueSource returns a human-readable source attribution for an issue.
// For beads issues: "bd/<assignee>" or "bd" if unassigned.
func issueSource(issue beads.Issue) string {
	if issue.Assignee != "" {
		return "bd/" + issue.Assignee
	}
	return "bd"
}

func mapState(s agent.State) string {
	switch s {
	case agent.StateIdle:
		return "info"
	case agent.StateWorking:
		return "ok"
	case agent.StateDone:
		return "ok"
	case agent.StateStuck:
		return "warning"
	case agent.StateError:
		return "error"
	case agent.StateStopped:
		return "stopped"
	default:
		return ""
	}
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func (m *WorkspaceModel) loadChannels() {
	store := channel.NewStore(m.info.Entry.Path)
	if err := store.Load(); err != nil {
		m.channels = nil
		return
	}
	m.channels = store.List()
	m.channelsLoaded = true
}

func (m *WorkspaceModel) renderChannels() string {
	var b strings.Builder

	if len(m.channels) == 0 {
		b.WriteString(m.styles.Muted.Render("  No channels. Run 'bc channel create <name>' to create one."))
		b.WriteString("\n")
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-20s %-8s %-8s %s", "CHANNEL", "MEMBERS", "MSGS", "LAST MESSAGE")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	start, end := m.viewportRange(len(m.channels))
	for i := start; i < end; i++ {
		ch := m.channels[i]
		selected := i == m.cursor

		msgCount := len(ch.History)
		lastMsg := "-"
		if msgCount > 0 {
			last := ch.History[msgCount-1]
			lastMsg = last.Message
			if len(lastMsg) > 40 {
				lastMsg = lastMsg[:37] + "..."
			}
		}

		line := fmt.Sprintf("  %-20s %-8d %-8d %s",
			"#"+ch.Name, len(ch.Members), msgCount, lastMsg,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.renderPositionIndicator(len(m.channels)))
	return b.String()
}

// queueStatusRank returns a sort rank for queue item status.
// Lower rank = higher in the list.
func queueStatusRank(s queue.ItemStatus) int {
	switch s {
	case queue.StatusWorking:
		return 0
	case queue.StatusAssigned:
		return 1
	case queue.StatusPending:
		return 2
	case queue.StatusFailed:
		return 3
	case queue.StatusDone:
		return 4
	default:
		return 5
	}
}

func (m *WorkspaceModel) renderQueue() string {
	var b strings.Builder

	if len(m.filteredQueueItems) == 0 {
		if m.queueFilter != QueueFilterAll && len(m.queueItems) > 0 {
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  No %s queue items. Press 'f' to change filter.", m.queueFilter.label())))
		} else {
			b.WriteString(m.styles.Muted.Render("  No queue items. Run 'bc queue add <title>' to create one."))
		}
		b.WriteString("\n")
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-10s %-12s %-10s %-10s %-15s %s", "ID", "BEAD", "STATUS", "MERGE", "ASSIGNED", "TITLE")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	start, end := m.viewportRange(len(m.filteredQueueItems))
	for i := start; i < end; i++ {
		item := m.filteredQueueItems[i]
		selected := i == m.cursor

		title := item.Title
		if len(title) > 36 {
			title = title[:33] + "..."
		}

		assignedTo := item.AssignedTo
		if assignedTo == "" {
			assignedTo = "-"
		}

		beadsID := item.BeadsID
		if beadsID == "" {
			beadsID = "-"
		}

		mergeStr := "-"
		if item.Merge != "" {
			mergeStr = string(item.Merge)
		}

		line := fmt.Sprintf("  %-10s %-12s %-10s %-10s %-15s %s",
			item.ID, beadsID, string(item.Status), mergeStr, assignedTo, title,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.StatusStyle(mapQueueStatus(item.Status)).Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.renderPositionIndicator(len(m.filteredQueueItems)))
	return b.String()
}

func (m *WorkspaceModel) loadPkgStats() {
	stateDir := filepath.Join(m.info.Entry.Path, ".bc")
	s, err := stats.Load(stateDir)
	if err != nil {
		return
	}
	m.pkgStats = s
}

func (m *WorkspaceModel) renderStatsBar() string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Issues: %d (%d open, %d closed)",
		m.stats.TotalIssues, m.stats.OpenIssues, m.stats.ClosedIssues))
	parts = append(parts, fmt.Sprintf("Epics: %d", m.stats.EpicsCount))
	activeCount := m.stats.WorkingAgents + m.stats.IdleAgents + m.stats.StuckAgents
	parts = append(parts, fmt.Sprintf("Agents: %d (%d active, %d idle, %d stuck)",
		len(m.agents), activeCount, m.stats.IdleAgents, m.stats.StuckAgents))
	parts = append(parts, fmt.Sprintf("Queue: %d pending, %d working, %d done",
		m.queueStats.Pending+m.queueStats.Assigned,
		m.queueStats.Working,
		m.queueStats.Done))
	if m.pkgStats != nil && m.pkgStats.WorkItems.Total > 0 {
		parts = append(parts, fmt.Sprintf("Completion: %.1f%%",
			m.pkgStats.WorkItems.CompletionRate*100))
	}

	return m.styles.Muted.Render(strings.Join(parts, "  |  "))
}

func (m *WorkspaceModel) loadRecentEvents() {
	evtLog := events.NewLog(filepath.Join(m.info.Entry.Path, ".bc", "events.jsonl"))
	evts, err := evtLog.ReadLast(dashboardMaxEvents)
	if err != nil {
		m.recentEvents = nil
		return
	}
	m.recentEvents = evts
}

func (m *WorkspaceModel) renderDashboard() string {
	var b strings.Builder

	// --- Utilization section ---
	b.WriteString(m.styles.Bold.Render("  AGENT UTILIZATION"))
	b.WriteString("\n")

	totalAgents := len(m.agents)
	working := m.stats.WorkingAgents
	idle := m.stats.IdleAgents
	stuck := m.stats.StuckAgents
	stopped := m.stats.StoppedAgents

	// Utilization = working / (total - stopped), i.e. working / active
	active := totalAgents - stopped
	var utilPct float64
	if active > 0 {
		utilPct = float64(working) / float64(active) * 100
	}

	// Render utilization bar
	barWidth := 30
	if m.width > 80 {
		barWidth = 40
	}
	filled := 0
	if active > 0 {
		filled = int(float64(barWidth) * float64(working) / float64(active))
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	b.WriteString(fmt.Sprintf("  Utilization: %s %5.1f%%\n", m.styles.Success.Render(bar), utilPct))
	b.WriteString("\n")

	// --- State breakdown ---
	b.WriteString(m.styles.Bold.Render("  STATE BREAKDOWN"))
	b.WriteString("\n")

	states := []struct {
		label string
		count int
		style string
	}{
		{"Working", working, "ok"},
		{"Idle", idle, "info"},
		{"Stuck", stuck, "warning"},
		{"Stopped", stopped, "stopped"},
	}

	for _, s := range states {
		countBar := ""
		if totalAgents > 0 {
			w := int(float64(barWidth) * float64(s.count) / float64(totalAgents))
			if s.count > 0 && w == 0 {
				w = 1
			}
			countBar = strings.Repeat("█", w)
		}
		line := fmt.Sprintf("  %-10s %3d  %s", s.label, s.count, countBar)
		b.WriteString(m.styles.StatusStyle(s.style).Render(line))
		b.WriteString("\n")
	}

	totalLine := fmt.Sprintf("  %-10s %3d", "Total", totalAgents)
	b.WriteString(m.styles.Muted.Render(totalLine))
	b.WriteString("\n\n")

	// --- Per-agent health table ---
	b.WriteString(m.styles.Bold.Render("  AGENT HEALTH"))
	b.WriteString("\n")

	if len(m.agents) == 0 {
		b.WriteString(m.styles.Muted.Render("  No agents running."))
		b.WriteString("\n")
	} else {
		header := fmt.Sprintf("  %-15s %-12s %-10s %-12s %-6s %-6s",
			"NAME", "ROLE", "STATE", "UPTIME", "DONE", "FAIL")
		b.WriteString(m.styles.Bold.Render(header))
		b.WriteString("\n")

		for _, a := range m.agents {
			as := m.agentStats[a.Name]
			uptime := "-"
			if as.Uptime > 0 {
				uptime = fmtDuration(as.Uptime)
			} else if a.State != agent.StateStopped && !a.StartedAt.IsZero() {
				uptime = fmtDuration(time.Since(a.StartedAt))
			}

			line := fmt.Sprintf("  %-15s %-12s %-10s %-12s %-6d %-6d",
				a.Name, a.Role, a.State, uptime, as.TasksCompleted, as.TasksFailed)
			b.WriteString(m.styles.StatusStyle(mapState(a.State)).Render(line))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// --- Issue Overview ---
	b.WriteString(m.styles.Bold.Render("  Issue Overview"))
	b.WriteString("\n")

	if m.stats.TotalIssues == 0 {
		b.WriteString(m.styles.Muted.Render("    No issues tracked."))
		b.WriteString("\n")
	} else {
		openLabel := fmt.Sprintf("    Open: %d", m.stats.OpenIssues)
		closedLabel := fmt.Sprintf("  Closed: %d", m.stats.ClosedIssues)
		totalLabel := fmt.Sprintf("  Total: %d", m.stats.TotalIssues)

		if m.stats.OpenIssues > 0 {
			b.WriteString(m.styles.Warning.Render(openLabel))
		} else {
			b.WriteString(m.styles.Success.Render(openLabel))
		}
		b.WriteString(m.styles.Success.Render(closedLabel))
		b.WriteString(m.styles.Muted.Render(totalLabel))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(m.styles.Bold.Render("  Recently Closed"))
	b.WriteString("\n")

	closedIssues := m.getRecentlyClosedIssues()
	if len(closedIssues) == 0 {
		b.WriteString(m.styles.Muted.Render("    No recently closed issues."))
		b.WriteString("\n")
	} else {
		for _, issue := range closedIssues {
			title := issue.Title
			if len(title) > 50 {
				title = title[:47] + "..."
			}
			line := fmt.Sprintf("    %-12s %-10s %s", issue.ID, issue.Status, title)
			b.WriteString(m.styles.Success.Render(line))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	b.WriteString(m.styles.Bold.Render("  Activity Feed"))
	b.WriteString("\n")

	if len(m.recentEvents) == 0 {
		b.WriteString(m.styles.Muted.Render("    No recent activity."))
		b.WriteString("\n")
	} else {
		for _, ev := range m.recentEvents {
			ts := ev.Timestamp.Format("15:04:05")
			agentStr := ""
			if ev.Agent != "" {
				agentStr = fmt.Sprintf(" [%s]", ev.Agent)
			}
			msg := string(ev.Type)
			if ev.Message != "" {
				msg = ev.Message
				if len(msg) > 60 {
					msg = msg[:57] + "..."
				}
			}
			line := fmt.Sprintf("    %s %-18s%s %s", ts, ev.Type, agentStr, msg)
			b.WriteString(m.styles.Normal.Render(line))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func mapQueueStatus(s queue.ItemStatus) string {
	switch s {
	case queue.StatusPending:
		return "pending"
	case queue.StatusAssigned:
		return "queued"
	case queue.StatusWorking:
		return "running"
	case queue.StatusDone:
		return "success"
	case queue.StatusFailed:
		return "failed"
	default:
		return ""
	}
}

const (
	dashboardMaxEvents       = 10
	dashboardMaxClosedIssues = 5
)

func (m *WorkspaceModel) getRecentlyClosedIssues() []beads.Issue {
	var closed []beads.Issue
	for i := len(m.issues) - 1; i >= 0; i-- {
		issue := m.issues[i]
		switch issue.Status {
		case "closed", "done", "resolved":
			closed = append(closed, issue)
			if len(closed) >= dashboardMaxClosedIssues {
				return closed
			}
		}
	}
	return closed
}

func (m *WorkspaceModel) renderStatsPanel() string {
	var b strings.Builder

	// Section 1: Work Items Overview
	b.WriteString(m.styles.Bold.Render("Work Items"))
	b.WriteString("\n")

	wi := m.queueStats
	total := wi.Total
	completionPct := 0.0
	failurePct := 0.0
	if total > 0 {
		completionPct = float64(wi.Done) / float64(total) * 100
		failurePct = float64(wi.Failed) / float64(total) * 100
	}

	workFields := []struct {
		label string
		value string
		style string
	}{
		{"Total", fmt.Sprintf("%d", total), ""},
		{"Pending", fmt.Sprintf("%d", wi.Pending+wi.Assigned), "warning"},
		{"Working", fmt.Sprintf("%d", wi.Working), "ok"},
		{"Done", fmt.Sprintf("%d", wi.Done), "ok"},
		{"Failed", fmt.Sprintf("%d", wi.Failed), ""},
		{"Completion", fmt.Sprintf("%.1f%%", completionPct), ""},
		{"Failure", fmt.Sprintf("%.1f%%", failurePct), ""},
	}

	if wi.Failed > 0 {
		workFields[4].style = "error"
	}
	if completionPct >= 80 {
		workFields[5].style = "ok"
	}
	if failurePct > 10 {
		workFields[6].style = "error"
	}

	for _, f := range workFields {
		label := m.styles.Muted.Width(15).Render(f.label + ":")
		valueStyle := m.styles.Normal
		switch f.style {
		case "ok":
			valueStyle = m.styles.Success
		case "warning":
			valueStyle = m.styles.Warning
		case "error":
			valueStyle = m.styles.Error
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", label, valueStyle.Render(f.value)))
	}
	b.WriteString("\n")

	// Section 2: Agent Overview
	b.WriteString(m.styles.Bold.Render("Agents"))
	b.WriteString("\n")

	activeCount := m.stats.WorkingAgents + m.stats.IdleAgents + m.stats.StuckAgents
	utilPct := 0.0
	if activeCount > 0 {
		utilPct = float64(m.stats.WorkingAgents) / float64(activeCount) * 100
	}

	agentFields := []struct {
		label string
		value string
		style string
	}{
		{"Total", fmt.Sprintf("%d", len(m.agents)), ""},
		{"Active", fmt.Sprintf("%d", activeCount), "ok"},
		{"Working", fmt.Sprintf("%d", m.stats.WorkingAgents), "ok"},
		{"Idle", fmt.Sprintf("%d", m.stats.IdleAgents), "info"},
		{"Stuck", fmt.Sprintf("%d", m.stats.StuckAgents), ""},
		{"Stopped", fmt.Sprintf("%d", m.stats.StoppedAgents), ""},
		{"Utilization", fmt.Sprintf("%.0f%%", utilPct), ""},
	}

	if m.stats.StuckAgents > 0 {
		agentFields[4].style = "error"
	}

	for _, f := range agentFields {
		label := m.styles.Muted.Width(15).Render(f.label + ":")
		valueStyle := m.styles.Normal
		switch f.style {
		case "ok":
			valueStyle = m.styles.Success
		case "warning":
			valueStyle = m.styles.Warning
		case "error":
			valueStyle = m.styles.Error
		case "info":
			valueStyle = m.styles.Info
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", label, valueStyle.Render(f.value)))
	}
	b.WriteString("\n")

	// Section 3: Per-Agent Completion Rates
	b.WriteString(m.styles.Bold.Render("Per-Agent Stats"))
	b.WriteString("\n")

	if len(m.agentStats) == 0 {
		b.WriteString(m.styles.Muted.Render("  No agent stats available."))
		b.WriteString("\n")
	} else {
		header := fmt.Sprintf("  %-15s %-10s %-8s %-8s %-10s %s",
			"NAME", "STATE", "DONE", "FAIL", "RATE", "UPTIME")
		b.WriteString(m.styles.Bold.Render(header))
		b.WriteString("\n")

		for _, a := range m.agents {
			as, ok := m.agentStats[a.Name]
			if !ok {
				continue
			}

			totalTasks := as.TasksCompleted + as.TasksFailed
			ratePct := 0.0
			if totalTasks > 0 {
				ratePct = float64(as.TasksCompleted) / float64(totalTasks) * 100
			}

			uptime := "-"
			if as.Uptime > 0 {
				uptime = fmtDuration(as.Uptime)
			} else if a.State != agent.StateStopped && !a.StartedAt.IsZero() {
				uptime = fmtDuration(time.Since(a.StartedAt))
			}

			rateStr := "-"
			if totalTasks > 0 {
				rateStr = fmt.Sprintf("%.0f%%", ratePct)
			}

			line := fmt.Sprintf("  %-15s %-10s %-8d %-8d %-10s %s",
				as.Name, as.State, as.TasksCompleted, as.TasksFailed, rateStr, uptime)
			b.WriteString(m.styles.StatusStyle(mapState(a.State)).Render(line))
			b.WriteString("\n")
		}
	}
	b.WriteString("\n")

	// Section 4: Extended metrics from pkg/stats if available
	if m.pkgStats != nil {
		b.WriteString(m.styles.Bold.Render("Work Item Types"))
		b.WriteString("\n")

		typeFields := []struct {
			label string
			value string
		}{
			{"Epics", fmt.Sprintf("%d", m.pkgStats.WorkItems.Epics)},
			{"Tasks", fmt.Sprintf("%d", m.pkgStats.WorkItems.Tasks)},
			{"Bugs", fmt.Sprintf("%d", m.pkgStats.WorkItems.Bugs)},
			{"Other", fmt.Sprintf("%d", m.pkgStats.WorkItems.Other)},
		}

		if m.pkgStats.WorkItems.AvgTimeToComplete > 0 {
			typeFields = append(typeFields, struct {
				label string
				value string
			}{"Avg Completion", fmtDuration(m.pkgStats.WorkItems.AvgTimeToComplete)})
		}

		for _, f := range typeFields {
			label := m.styles.Muted.Width(15).Render(f.label + ":")
			b.WriteString(fmt.Sprintf("  %s %s\n", label, m.styles.Normal.Render(f.value)))
		}
	}

	return b.String()
}

func (m *WorkspaceModel) computeStats() {
	m.stats = WorkspaceStats{}
	for _, issue := range m.issues {
		m.stats.TotalIssues++
		if issue.Type == "epic" {
			m.stats.EpicsCount++
		}
		switch issue.Status {
		case "open", "pending", "in_progress":
			m.stats.OpenIssues++
		case "closed", "done", "resolved":
			m.stats.ClosedIssues++
		}
	}
	for _, a := range m.agents {
		switch a.State {
		case agent.StateIdle:
			m.stats.IdleAgents++
		case agent.StateWorking:
			m.stats.WorkingAgents++
		case agent.StateStuck:
			m.stats.StuckAgents++
		case agent.StateStopped:
			m.stats.StoppedAgents++
		}
	}
}

func (m *WorkspaceModel) renderStats() string {
	var b strings.Builder

	if m.pkgStats == nil {
		b.WriteString(m.styles.Muted.Render("  No stats available."))
		b.WriteString("\n")
		return b.String()
	}

	wi := m.pkgStats.WorkItems
	am := m.pkgStats.Agents

	// --- Aggregate Overview ---
	b.WriteString(m.styles.Bold.Render("  Workspace Overview"))
	b.WriteString("\n\n")

	// Issues row
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  %-20s", "Issues:")))
	b.WriteString(m.styles.Normal.Render(fmt.Sprintf("%d total  ", m.stats.TotalIssues)))
	b.WriteString(m.styles.Success.Render(fmt.Sprintf("%d open  ", m.stats.OpenIssues)))
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("%d closed", m.stats.ClosedIssues)))
	b.WriteString("\n")

	// Epics row
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  %-20s", "Epics:")))
	b.WriteString(m.styles.Normal.Render(fmt.Sprintf("%d", m.stats.EpicsCount)))
	b.WriteString("\n")

	// Agents row
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  %-20s", "Agents:")))
	b.WriteString(m.styles.Normal.Render(fmt.Sprintf("%d total  ", am.TotalAgents)))
	b.WriteString(m.styles.Success.Render(fmt.Sprintf("%d active  ", am.ActiveAgents)))
	b.WriteString(m.styles.Info.Render(fmt.Sprintf("%d idle  ", am.Idle)))
	if am.Stuck > 0 {
		b.WriteString(m.styles.Warning.Render(fmt.Sprintf("%d stuck  ", am.Stuck)))
	}
	if am.Stopped > 0 {
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf("%d stopped", am.Stopped)))
	}
	b.WriteString("\n")

	// Queue row
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  %-20s", "Queue:")))
	b.WriteString(m.styles.Normal.Render(fmt.Sprintf("%d total  ", wi.Total)))
	b.WriteString(m.styles.Warning.Render(fmt.Sprintf("%d pending  ", wi.Pending+wi.Assigned)))
	b.WriteString(m.styles.Success.Render(fmt.Sprintf("%d working  ", wi.Working)))
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("%d done", wi.Done)))
	if wi.Failed > 0 {
		b.WriteString(m.styles.Error.Render(fmt.Sprintf("  %d failed", wi.Failed)))
	}
	b.WriteString("\n")

	// Completion rate row
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  %-20s", "Completion Rate:")))
	rate := wi.CompletionRate * 100
	rateStr := fmt.Sprintf("%.1f%%", rate)
	if rate >= 75 {
		b.WriteString(m.styles.Success.Render(rateStr))
	} else if rate >= 50 {
		b.WriteString(m.styles.Warning.Render(rateStr))
	} else {
		b.WriteString(m.styles.Normal.Render(rateStr))
	}
	b.WriteString("\n")

	// --- Per-Agent Metrics ---
	b.WriteString("\n")
	b.WriteString(m.styles.Bold.Render("  Per-Agent Metrics"))
	b.WriteString("\n\n")

	if len(am.AgentStats) == 0 {
		b.WriteString(m.styles.Muted.Render("  No agent data."))
		b.WriteString("\n")
		return b.String()
	}

	// Count issues per agent (from beads assignee)
	issuesByAgent := make(map[string]int)
	for _, issue := range m.issues {
		if issue.Assignee != "" {
			issuesByAgent[issue.Assignee]++
		}
	}

	// Header
	header := fmt.Sprintf("  %-15s %-14s %-10s %6s %6s %6s %12s",
		"NAME", "ROLE", "STATE", "DONE", "FAIL", "ISSUES", "UPTIME")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	for _, as := range am.AgentStats {
		uptime := "-"
		if as.Uptime > 0 {
			uptime = fmtDuration(as.Uptime)
		}

		issues := issuesByAgent[as.Name]

		line := fmt.Sprintf("  %-15s %-14s %-10s %6d %6d %6d %12s",
			as.Name, as.Role, as.State, as.TasksCompleted, as.TasksFailed, issues, uptime)

		b.WriteString(m.styles.StatusStyle(mapState(agent.State(as.State))).Render(line))
		b.WriteString("\n")
	}

	return b.String()
}
