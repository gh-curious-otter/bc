package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/github"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// Tab identifies which tab is active in the workspace view.
type Tab int

const (
	TabAgents Tab = iota
	TabIssues
	TabPRs
	TabChannels
	TabQueue

	tabCount = 5
)

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
	info    WorkspaceInfo
	styles  style.Styles
	width   int
	height  int
	tab     Tab
	cursor  int
	manager *agent.Manager

	// Data
	agents     []*agent.Agent
	issues     []beads.Issue
	prs        []github.PR
	channels   []*channel.Channel
	queueItems []queue.WorkItem

	// Queue stats
	queueStats queue.Stats

	// Dashboard stats
	stats WorkspaceStats

	// Loaded flags
	agentsLoaded   bool
	issuesLoaded   bool
	prsLoaded      bool
	channelsLoaded bool
	queueLoaded    bool
}

// NewWorkspaceModel creates a workspace detail view.
func NewWorkspaceModel(info WorkspaceInfo, s style.Styles) *WorkspaceModel {
	m := &WorkspaceModel{
		info:   info,
		styles: s,
		tab:    TabAgents,
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
	m.issues = beads.ListIssues(info.Entry.Path)
	m.issuesLoaded = true

	// Load GitHub PRs and issues
	m.prs = github.ListPRs(info.Entry.Path)
	m.prsLoaded = true

	// Load channels
	m.loadChannels()

	// Load queue stats and items
	m.loadQueueStats()
	m.loadQueueItems()

	// Compute dashboard stats
	m.computeStats()

	return m
}

// HandleKey processes a key event and returns an action for the parent.
func (m *WorkspaceModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()

	switch key {
	case "tab":
		m.tab = (m.tab + 1) % tabCount
		m.cursor = 0
		return NoAction
	case "shift+tab":
		m.tab = (m.tab + tabCount - 1) % tabCount
		m.cursor = 0
		return NoAction
	case "j", "down":
		m.cursor++
		m.clampCursor()
		return NoAction
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return NoAction
	case "g", "home":
		m.cursor = 0
		return NoAction
	case "G", "end":
		m.cursor = m.maxCursor()
		return NoAction
	case "r":
		m.refresh()
		return NoAction
	}
	if isEnterKey(msg) {
		return m.selectCurrent()
	}

	return NoAction
}

func (m *WorkspaceModel) refresh() {
	m.manager.RefreshState()
	m.agents = m.manager.ListAgents()
	m.issues = beads.ListIssues(m.info.Entry.Path)
	m.prs = github.ListPRs(m.info.Entry.Path)
	m.loadChannels()
	m.loadQueueStats()
	m.loadQueueItems()
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
	m.queueLoaded = true
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
	case TabPRs:
		if m.cursor < len(m.prs) {
			return Action{Type: ActionDrillPR, Data: m.prs[m.cursor]}
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
	case TabPRs:
		if len(m.prs) > 0 {
			return len(m.prs) - 1
		}
	case TabChannels:
		if len(m.channels) > 0 {
			return len(m.channels) - 1
		}
	case TabQueue:
		if len(m.queueItems) > 0 {
			return len(m.queueItems) - 1
		}
	}
	return 0
}

func (m *WorkspaceModel) clampCursor() {
	max := m.maxCursor()
	if m.cursor > max {
		m.cursor = max
	}
}

// View renders the workspace detail screen.
func (m *WorkspaceModel) View() string {
	var b strings.Builder

	// Queue stats (above tab bar)
	if m.queueStats.Total > 0 {
		stats := fmt.Sprintf("Queue: %d pending / %d working / %d done / %d total",
			m.queueStats.Pending+m.queueStats.Assigned,
			m.queueStats.Working,
			m.queueStats.Done,
			m.queueStats.Total,
		)
		b.WriteString(m.styles.Muted.Render(stats))
		b.WriteString("\n\n")
	}

	// Tab bar
	b.WriteString(m.renderTabBar())
	b.WriteString("\n\n")

	// Content
	switch m.tab {
	case TabAgents:
		b.WriteString(m.renderAgents())
	case TabIssues:
		b.WriteString(m.renderIssues())
	case TabPRs:
		b.WriteString(m.renderPRs())
	case TabChannels:
		b.WriteString(m.renderChannels())
	case TabQueue:
		b.WriteString(m.renderQueue())
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
		{"PRs", TabPRs, len(m.prs)},
		{"Channels", TabChannels, len(m.channels)},
		{"Queue", TabQueue, len(m.queueItems)},
	}

	var parts []string
	for _, t := range tabs {
		label := fmt.Sprintf(" %s (%d) ", t.label, t.count)
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
	header := fmt.Sprintf("  %-15s %-12s %-10s %-15s %s", "NAME", "ROLE", "STATE", "UPTIME", "TASK")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	// Fixed columns: 2(indent) + NAME(15) + ROLE(12) + STATE(10) + UPTIME(15) = 54
	// Task gets the rest of the terminal width
	taskWidth := m.width - 54
	if taskWidth < 20 {
		taskWidth = 20
	}

	runningCount := 0
	for i, a := range m.agents {
		selected := i == m.cursor

		uptime := "-"
		if a.State != agent.StateStopped {
			runningCount++
			uptime = fmtDuration(time.Since(a.StartedAt))
		}

		task := a.Task
		if task == "" {
			task = "-"
		}
		if len(task) > taskWidth {
			task = task[:taskWidth-3] + "..."
		}

		line := fmt.Sprintf("  %-15s %-12s %-10s %-15s %s",
			a.Name, a.Role, a.State, uptime, task,
		)

		overLimit := m.info.MaxWorkers > 0 && a.State != agent.StateStopped && runningCount > m.info.MaxWorkers
		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else if overLimit {
			b.WriteString(m.styles.Error.Render(line))
		} else {
			b.WriteString(m.styles.StatusStyle(mapState(a.State)).Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *WorkspaceModel) renderIssues() string {
	var b strings.Builder

	if len(m.issues) == 0 {
		b.WriteString(m.styles.Muted.Render("  No issues found."))
		b.WriteString("\n")
		return b.String()
	}

	header := fmt.Sprintf("  %-12s %-10s %-40s %s", "ID", "STATUS", "TITLE", "SOURCE")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	for i, issue := range m.issues {
		selected := i == m.cursor

		title := issue.Title
		if len(title) > 38 {
			title = title[:35] + "..."
		}

		line := fmt.Sprintf("  %-12s %-10s %-40s %s",
			issue.ID, issue.Status, title, issue.Source,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *WorkspaceModel) renderPRs() string {
	var b strings.Builder

	if len(m.prs) == 0 {
		b.WriteString(m.styles.Muted.Render("  No pull requests found."))
		b.WriteString("\n")
		return b.String()
	}

	header := fmt.Sprintf("  %-8s %-12s %-45s %s", "NUMBER", "STATE", "TITLE", "REVIEW")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	for i, pr := range m.prs {
		selected := i == m.cursor

		title := pr.Title
		if len(title) > 43 {
			title = title[:40] + "..."
		}

		line := fmt.Sprintf("  %-8s %-12s %-45s %s",
			fmt.Sprintf("#%d", pr.Number), pr.State, title, pr.ReviewDecision,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	return b.String()
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

	for i, ch := range m.channels {
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

	return b.String()
}

func (m *WorkspaceModel) renderQueue() string {
	var b strings.Builder

	if len(m.queueItems) == 0 {
		b.WriteString(m.styles.Muted.Render("  No queue items. Run 'bc queue add <title>' to create one."))
		b.WriteString("\n")
		return b.String()
	}

	// Header
	header := fmt.Sprintf("  %-10s %-10s %-15s %s", "ID", "STATUS", "ASSIGNED", "TITLE")
	b.WriteString(m.styles.Bold.Render(header))
	b.WriteString("\n")

	for i, item := range m.queueItems {
		selected := i == m.cursor

		title := item.Title
		if len(title) > 45 {
			title = title[:42] + "..."
		}

		assignedTo := item.AssignedTo
		if assignedTo == "" {
			assignedTo = "-"
		}

		line := fmt.Sprintf("  %-10s %-10s %-15s %s",
			item.ID, string(item.Status), assignedTo, title,
		)

		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.StatusStyle(mapQueueStatus(item.Status)).Render(line))
		}
		b.WriteString("\n")
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
