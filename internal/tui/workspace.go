package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/memory"
	"github.com/rpuneet/bc/pkg/stats"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// Tab identifies which tab is active in the workspace view.
type Tab int

const (
	TabAgents Tab = iota
	TabIssues
	TabChannels
	TabQueue
	TabDashboard
	TabStats

	tabCount = 6
)

// QueueFilter defines the filter mode for the queue view.
type QueueFilter int

const (
	QueueFilterAll QueueFilter = iota
	QueueFilterReady
	QueueFilterInProgress
	QueueFilterByAgent
)

// QueueItem represents an item in the work or merge queue.
type QueueItem struct {
	ID       string
	Title    string
	Status   string
	Assignee string
	Type     string // "work" or "merge"
	Agent    string // Associated agent name
	Branch   string // Branch name for merge queue
}

// WorkspaceStats holds aggregated statistics for the workspace.
type WorkspaceStats struct {
	// Issue stats
	TotalIssues  int
	OpenIssues   int
	ClosedIssues int
	EpicsCount   int

	// Queue stats
	ReadyIssues      int // Issues unblocked and ready for work
	InProgressIssues int // Issues currently being worked on
	AssignedIssues   int // Issues assigned to agents

	// Agent stats by state
	IdleAgents    int
	WorkingAgents int
	StuckAgents   int
	StoppedAgents int
}

// WorkspaceModel shows the detail view for a single workspace.
type WorkspaceModel struct {
	// Data
	agents        []*agent.Agent
	channels      []*channel.Channel
	issues        []beads.Issue
	recentEvents  []events.Event
	queueItems    []QueueItem
	filteredQueue []QueueItem

	// Per-agent stats from pkg/stats
	agentStats map[string]stats.AgentStat

	// Per-agent issue counts
	issuesByAgent map[string]int

	// Per-agent memory experience counts
	experiencesByAgent map[string]int

	manager   *agent.Manager
	pkgStats  *stats.Stats
	issuesErr error

	styles style.Styles
	info   WorkspaceInfo
	stats  WorkspaceStats

	width        int
	height       int
	cursor       int
	scrollOffset int // first visible item index for current tab
	tab          Tab
	queueFilter  QueueFilter

	// Loaded flags (lazy-load per tab / on focus; see ensureTabDataLoaded)
	agentsLoaded       bool
	issuesLoaded       bool
	channelsLoaded     bool
	queueLoaded        bool
	agentStatsLoaded   bool
	recentEventsLoaded bool
	pkgStatsLoaded     bool
}

// NewWorkspaceModel creates a workspace detail view.
// Heavy data (issues, channels, queue, events, stats) is deferred until the user
// focuses the relevant tab; see ensureTabDataLoaded (epic #322 / #324).
func NewWorkspaceModel(info WorkspaceInfo, s style.Styles) *WorkspaceModel {
	m := &WorkspaceModel{
		info:   info,
		styles: s,
		tab:    TabAgents,
	}

	// Minimal load for first paint (Agents tab): manager + agents + agent stats only
	m.manager = agent.NewWorkspaceManager(
		info.Entry.Path+"/.bc/agents",
		info.Entry.Path,
	)
	_ = m.manager.LoadState()
	_ = m.manager.RefreshState()
	m.agents = m.manager.ListAgents()
	m.agentsLoaded = true
	m.issuesByAgent = make(map[string]int)
	m.loadAgentStats()
	m.computeStatsFromAgentsOnly()
	// Issues, channels, queue, events, memory, pkg stats loaded on tab focus via ensureTabDataLoaded

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
		m.ensureTabDataLoaded(m.tab)
		return NoAction
	case "shift+tab":
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.cursor = 0
		m.scrollOffset = 0
		m.ensureTabDataLoaded(m.tab)
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
		// Cycle queue filter when on queue tab
		if m.tab == TabQueue {
			m.cycleQueueFilter()
			return NoAction
		}
	}
	if isEnterKey(msg) {
		return m.selectCurrent()
	}

	return NoAction
}

// refresh performs a full reload: agent state, issues, channels, queue, events, stats.
// Used on explicit 'r' key. See refreshLight for the lightweight tick path.
func (m *WorkspaceModel) refresh() {
	_ = m.manager.RefreshState()
	m.agents = m.manager.ListAgents()
	m.issues, m.issuesErr = beads.ListIssues(m.info.Entry.Path)
	m.issuesLoaded = true
	m.loadChannels()
	m.loadQueue()
	m.queueLoaded = true
	m.loadMemoryInfo()
	m.loadRecentEvents()
	m.recentEventsLoaded = true
	m.computeStats()
	m.loadAgentStats()
	m.agentStatsLoaded = true
	m.loadPkgStats()
	m.pkgStatsLoaded = true
	m.clampCursor()
	m.ensureCursorVisible()
}

// refreshLight updates only agent state and derived stats; no file I/O.
// Used on tick so the TUI stays responsive. Full data remains until user presses 'r'.
func (m *WorkspaceModel) refreshLight() {
	_ = m.manager.RefreshState()
	m.agents = m.manager.ListAgents()
	m.computeStats()
	m.clampCursor()
	m.ensureCursorVisible()
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

func (m *WorkspaceModel) loadMemoryInfo() {
	m.experiencesByAgent = make(map[string]int)
	for _, a := range m.agents {
		store := memory.NewStore(m.info.Entry.Path, a.Name)
		if !store.Exists() {
			continue
		}
		experiences, err := store.GetExperiences()
		if err != nil {
			continue
		}
		m.experiencesByAgent[a.Name] = len(experiences)
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
		if len(m.filteredQueue) > 0 {
			return len(m.filteredQueue) - 1
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

// ensureTabDataLoaded loads heavy data for the given tab on first focus (lazy-load per #324).
func (m *WorkspaceModel) ensureTabDataLoaded(tab Tab) {
	switch tab {
	case TabAgents:
		if !m.agentStatsLoaded {
			m.loadAgentStats()
			m.agentStatsLoaded = true
		}
		m.loadMemoryInfo()
	case TabIssues:
		if !m.issuesLoaded {
			m.issues, m.issuesErr = beads.ListIssues(m.info.Entry.Path)
			m.issuesLoaded = true
			m.computeStats()
		}
	case TabChannels:
		if !m.channelsLoaded {
			m.loadChannels()
		}
	case TabQueue:
		if !m.queueLoaded {
			m.loadQueue()
			m.queueLoaded = true
		}
	case TabDashboard:
		if !m.issuesLoaded {
			m.issues, m.issuesErr = beads.ListIssues(m.info.Entry.Path)
			m.issuesLoaded = true
			m.computeStats()
		}
		if !m.channelsLoaded {
			m.loadChannels()
			m.channelsLoaded = true
		}
		if !m.queueLoaded {
			m.loadQueue()
			m.queueLoaded = true
		}
		if !m.recentEventsLoaded {
			m.loadRecentEvents()
			m.recentEventsLoaded = true
		}
		m.loadMemoryInfo()
		if !m.agentStatsLoaded {
			m.loadAgentStats()
			m.agentStatsLoaded = true
		}
		m.computeStats()
		if !m.pkgStatsLoaded {
			m.loadPkgStats()
			m.pkgStatsLoaded = true
		}
	case TabStats:
		if !m.agentStatsLoaded {
			m.loadAgentStats()
			m.agentStatsLoaded = true
		}
		if !m.pkgStatsLoaded {
			m.loadPkgStats()
			m.pkgStatsLoaded = true
		}
	}
}

// View renders the workspace detail screen.
func (m *WorkspaceModel) View() string {
	// Lazy-load heavy data for the active tab on first focus (#324).
	m.ensureTabDataLoaded(m.tab)

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
		{"Channels", TabChannels, len(m.channels)},
		{"Queue", TabQueue, len(m.filteredQueue)},
		{"Dashboard", TabDashboard, m.stats.OpenIssues},
		{"Stats", TabStats, -1},
	}

	var parts []string
	for _, t := range tabs {
		var label string
		if t.count >= 0 {
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

	// Header with ISSUES and MEM columns for queue/memory info
	header := fmt.Sprintf("  %-15s %-12s %-10s %-8s %-6s %-12s %s",
		"NAME", "ROLE", "STATE", "ISSUES", "MEM", "UPTIME", "TASK")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	// Fixed columns: 2(indent) + NAME(15) + ROLE(12) + STATE(10) + ISSUES(8) + MEM(6) + UPTIME(12) = 65
	// Task gets the rest of the terminal width
	taskWidth := m.width - 65
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

		// Get issues count for this agent
		issues := m.issuesByAgent[a.Name]
		issuesStr := "-"
		if issues > 0 {
			issuesStr = fmt.Sprintf("%d", issues)
		}

		// Get experience count for this agent (memory info)
		experiences := m.experiencesByAgent[a.Name]
		memStr := "-"
		if experiences > 0 {
			memStr = fmt.Sprintf("%d", experiences)
		}

		task := a.Task
		if task == "" {
			task = "-"
		}
		if len(task) > taskWidth {
			task = task[:taskWidth-3] + "..."
		}

		line := fmt.Sprintf("  %-15s %-12s %-10s %-8s %-6s %-12s %s",
			a.Name, a.Role, a.State, issuesStr, memStr, uptime, task,
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
	store, err := channel.OpenStore(m.info.Entry.Path)
	if err != nil {
		store = channel.NewStore(m.info.Entry.Path)
	}
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

// loadQueue loads work queue and merge queue items.
func (m *WorkspaceModel) loadQueue() {
	m.queueItems = nil

	// Load work queue items from ready issues
	readyIssues := beads.ReadyIssues(m.info.Entry.Path)
	for _, issue := range readyIssues {
		m.queueItems = append(m.queueItems, QueueItem{
			ID:       issue.ID,
			Title:    issue.Title,
			Status:   issue.Status,
			Assignee: issue.Assignee,
			Type:     "work",
		})
	}

	// Load merge queue items from agents with active branches
	for _, a := range m.agents {
		if a.State == agent.StateWorking || a.State == agent.StateDone {
			branch := ""
			if a.WorktreeDir != "" {
				branch = m.getAgentBranch(a)
			}
			if branch != "" && branch != "main" && branch != "master" {
				m.queueItems = append(m.queueItems, QueueItem{
					ID:     a.Name,
					Title:  a.Task,
					Status: string(a.State),
					Agent:  a.Name,
					Branch: branch,
					Type:   "merge",
				})
			}
		}
	}

	m.applyQueueFilter()
}

// getAgentBranch returns the current branch for an agent's worktree.
func (m *WorkspaceModel) getAgentBranch(a *agent.Agent) string {
	if a.WorktreeDir == "" {
		return ""
	}
	cmd := exec.CommandContext(context.Background(), "git", "-C", a.WorktreeDir, "rev-parse", "--abbrev-ref", "HEAD") //nolint:gosec // WorktreeDir is from trusted agent data
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// cycleQueueFilter cycles through the available queue filters.
func (m *WorkspaceModel) cycleQueueFilter() {
	m.queueFilter = (m.queueFilter + 1) % 4
	m.applyQueueFilter()
	m.cursor = 0
	m.scrollOffset = 0
}

// applyQueueFilter applies the current filter to the queue items.
func (m *WorkspaceModel) applyQueueFilter() {
	switch m.queueFilter {
	case QueueFilterAll:
		m.filteredQueue = m.queueItems
	case QueueFilterReady:
		m.filteredQueue = nil
		for _, item := range m.queueItems {
			if item.Type == "work" {
				m.filteredQueue = append(m.filteredQueue, item)
			}
		}
	case QueueFilterInProgress:
		m.filteredQueue = nil
		for _, item := range m.queueItems {
			if item.Type == "merge" {
				m.filteredQueue = append(m.filteredQueue, item)
			}
		}
	case QueueFilterByAgent:
		m.filteredQueue = nil
		for _, item := range m.queueItems {
			if item.Assignee != "" || item.Agent != "" {
				m.filteredQueue = append(m.filteredQueue, item)
			}
		}
	}
}

// queueFilterLabel returns a display label for the current queue filter.
func (m *WorkspaceModel) queueFilterLabel() string {
	switch m.queueFilter {
	case QueueFilterAll:
		return "All"
	case QueueFilterReady:
		return "Work Queue"
	case QueueFilterInProgress:
		return "Merge Queue"
	case QueueFilterByAgent:
		return "Assigned"
	default:
		return "All"
	}
}

func (m *WorkspaceModel) renderQueue() string {
	var b strings.Builder

	// Filter indicator
	filterLabel := m.queueFilterLabel()
	b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  Filter: %s (press 'f' to cycle)", filterLabel)))
	b.WriteString("\n\n")

	if len(m.filteredQueue) == 0 {
		b.WriteString(m.styles.Muted.Render("  No items in queue."))
		b.WriteString("\n")
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-12s %-8s %-12s %-15s %s", "ID", "TYPE", "STATUS", "AGENT", "TITLE/BRANCH")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	start, end := m.viewportRange(len(m.filteredQueue))
	for i := start; i < end; i++ {
		item := m.filteredQueue[i]
		selected := i == m.cursor

		// Format title/branch
		titleOrBranch := item.Title
		if item.Type == "merge" && item.Branch != "" {
			titleOrBranch = item.Branch
		}
		if len(titleOrBranch) > 35 {
			titleOrBranch = titleOrBranch[:32] + "..."
		}

		agentName := item.Assignee
		if agentName == "" {
			agentName = item.Agent
		}
		if agentName == "" {
			agentName = "-"
		}

		line := fmt.Sprintf("  %-12s %-8s %-12s %-15s %s",
			item.ID, item.Type, item.Status, agentName, titleOrBranch,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else if item.Type == "merge" {
			b.WriteString(m.styles.Success.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.renderPositionIndicator(len(m.filteredQueue)))
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
	parts := make([]string, 0, 3)

	parts = append(parts, fmt.Sprintf("Issues: %d (%d open, %d closed)",
		m.stats.TotalIssues, m.stats.OpenIssues, m.stats.ClosedIssues))
	parts = append(parts, fmt.Sprintf("Epics: %d", m.stats.EpicsCount))
	activeCount := m.stats.WorkingAgents + m.stats.IdleAgents + m.stats.StuckAgents
	parts = append(parts, fmt.Sprintf("Agents: %d (%d active, %d idle, %d stuck)",
		len(m.agents), activeCount, m.stats.IdleAgents, m.stats.StuckAgents))

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
		style string
		count int
	}{
		{"Working", "ok", working},
		{"Idle", "info", idle},
		{"Stuck", "warning", stuck},
		{"Stopped", "stopped", stopped},
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

	// --- Queue Progress section ---
	b.WriteString(m.styles.Bold.Render("  QUEUE PROGRESS"))
	b.WriteString("\n")

	ready := m.stats.ReadyIssues
	inProgress := m.stats.InProgressIssues
	assigned := m.stats.AssignedIssues
	openIssues := m.stats.OpenIssues

	// Progress through open issues
	if openIssues > 0 {
		// Progress bar for in-progress vs total open
		progressFilled := 0
		if openIssues > 0 {
			progressFilled = int(float64(barWidth) * float64(inProgress) / float64(openIssues))
		}
		progressBar := strings.Repeat("█", progressFilled) + strings.Repeat("░", barWidth-progressFilled)
		progressPct := float64(inProgress) / float64(openIssues) * 100
		b.WriteString(fmt.Sprintf("  In Progress: %s %5.1f%% (%d/%d)\n", m.styles.Info.Render(progressBar), progressPct, inProgress, openIssues))
	}

	queueItems := []struct {
		label string
		style string
		count int
	}{
		{"Ready (unblocked)", "ok", ready},
		{"In Progress", "info", inProgress},
		{"Assigned", "", assigned},
		{"Total Open", "", openIssues},
	}

	for _, q := range queueItems {
		countBar := ""
		if openIssues > 0 && q.count > 0 {
			w := int(float64(barWidth) * float64(q.count) / float64(openIssues))
			if w == 0 {
				w = 1
			}
			countBar = strings.Repeat("█", w)
		}
		line := fmt.Sprintf("  %-18s %3d  %s", q.label, q.count, countBar)
		if q.style != "" {
			b.WriteString(m.styles.StatusStyle(q.style).Render(line))
		} else {
			b.WriteString(m.styles.Muted.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// --- Per-agent health table ---
	b.WriteString(m.styles.Bold.Render("  AGENT HEALTH"))
	b.WriteString("\n")

	if len(m.agents) == 0 {
		b.WriteString(m.styles.Muted.Render("  No agents running."))
		b.WriteString("\n")
	} else {
		header := fmt.Sprintf("  %-15s %-12s %-10s %-12s",
			"NAME", "ROLE", "STATE", "UPTIME")
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

			line := fmt.Sprintf("  %-15s %-12s %-10s %-12s",
				a.Name, a.Role, a.State, uptime)
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

// computeStatsFromAgentsOnly sets only agent state counts; used for fast first paint before issues/queue are loaded.
func (m *WorkspaceModel) computeStatsFromAgentsOnly() {
	m.issuesByAgent = make(map[string]int)
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

func (m *WorkspaceModel) computeStats() {
	m.stats = WorkspaceStats{}
	m.issuesByAgent = make(map[string]int)
	for _, issue := range m.issues {
		m.stats.TotalIssues++
		if issue.Type == "epic" {
			m.stats.EpicsCount++
		}
		switch issue.Status {
		case "open", "pending":
			m.stats.OpenIssues++
		case "in_progress":
			m.stats.OpenIssues++
			m.stats.InProgressIssues++
		case "closed", "done", "resolved":
			m.stats.ClosedIssues++
		}
		if issue.Assignee != "" {
			m.stats.AssignedIssues++
			m.issuesByAgent[issue.Assignee]++
		}
	}

	// Count ready issues (unblocked and available for work)
	readyIssues := beads.ReadyIssues(m.info.Entry.Path)
	m.stats.ReadyIssues = len(readyIssues)

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
	b.WriteString("\n\n")

	// --- Per-Agent Metrics ---
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
	header := fmt.Sprintf("  %-15s %-14s %-10s %8s %12s",
		"NAME", "ROLE", "STATE", "ISSUES", "UPTIME")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	for _, as := range am.AgentStats {
		uptime := "-"
		if as.Uptime > 0 {
			uptime = fmtDuration(as.Uptime)
		}

		issues := issuesByAgent[as.Name]

		line := fmt.Sprintf("  %-15s %-14s %-10s %8d %12s",
			as.Name, as.Role, as.State, issues, uptime)

		b.WriteString(m.styles.StatusStyle(mapState(agent.State(as.State))).Render(line))
		b.WriteString("\n")
	}

	return b.String()
}
