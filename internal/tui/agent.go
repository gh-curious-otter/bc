package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/tui/style"
)

const agentMaxRecentEvents = 10

// AgentModel shows the detail view for a single agent.
type AgentModel struct {
	agent   *agent.Agent
	manager *agent.Manager
	styles  style.Styles
	width   int
	height  int
	wsPath  string

	// Stats
	tasksCompleted int
	tasksFailed    int
	taskItems      []queue.WorkItem // items assigned to this agent

	// Recent activity
	recentEvents []events.Event

	// Peek output
	peekOutput string
	peekActive bool

	// Send message mode
	sendMode bool
	input    string
	sendMsg  string
}

// NewAgentModel creates an agent detail view.
// wsPath is the workspace root path for loading queue and event data.
func NewAgentModel(a *agent.Agent, mgr *agent.Manager, wsPath string, s style.Styles) *AgentModel {
	m := &AgentModel{
		agent:   a,
		manager: mgr,
		styles:  s,
		wsPath:  wsPath,
	}
	m.loadStats()
	m.loadRecentActivity()
	return m
}

// HandleKey processes a key event and returns an action for the parent.
func (m *AgentModel) HandleKey(msg tea.KeyMsg) Action {
	if m.sendMode {
		return m.handleSendKey(msg)
	}

	key := msg.String()

	switch key {
	case "esc":
		if m.peekActive {
			m.peekActive = false
			return NoAction
		}
		return Action{Type: ActionBack}
	case "p":
		m.loadPeek()
		return NoAction
	case "a":
		return Action{Type: ActionAttach, Data: m.agent.Name}
	case "s":
		m.sendMode = true
		m.input = ""
		m.sendMsg = ""
		return NoAction
	case "r":
		m.refresh()
		return NoAction
	}

	return NoAction
}

func (m *AgentModel) handleSendKey(msg tea.KeyMsg) Action {
	key := msg.String()

	switch key {
	case "esc":
		m.sendMode = false
		m.input = ""
		return NoAction
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return NoAction
	}

	if isEnterKey(msg) {
		if m.input != "" {
			if err := m.manager.SendToAgent(m.agent.Name, m.input); err != nil {
				m.sendMsg = "Error: " + err.Error()
			} else {
				m.sendMsg = "Message sent to " + m.agent.Name
			}
		}
		m.sendMode = false
		m.input = ""
		return NoAction
	}

	// Append typed characters
	if msg.Type == tea.KeyRunes {
		m.input += string(msg.Runes)
	} else if msg.Type == tea.KeySpace {
		m.input += " "
	}

	return NoAction
}

// refresh reloads the agent's state data from the manager.
func (m *AgentModel) refresh() {
	m.manager.RefreshState()
	if a := m.manager.GetAgent(m.agent.Name); a != nil {
		m.agent = a
	}
	m.loadStats()
	m.loadRecentActivity()
}

func (m *AgentModel) loadPeek() {
	output, err := m.manager.CaptureOutput(m.agent.Name, 30)
	if err != nil {
		m.peekOutput = "Error: " + err.Error()
	} else {
		m.peekOutput = output
	}
	m.peekActive = true
}

func (m *AgentModel) loadStats() {
	if m.wsPath == "" {
		return
	}
	q := queue.New(filepath.Join(m.wsPath, ".bc", "queue.json"))
	if err := q.Load(); err != nil {
		return
	}
	items := q.ListByAgent(m.agent.Name)
	m.taskItems = items
	m.tasksCompleted = 0
	m.tasksFailed = 0
	for _, item := range items {
		switch item.Status {
		case queue.StatusDone:
			m.tasksCompleted++
		case queue.StatusFailed:
			m.tasksFailed++
		}
	}
}

func (m *AgentModel) loadRecentActivity() {
	if m.wsPath == "" {
		return
	}
	evtLog := events.NewLog(filepath.Join(m.wsPath, ".bc", "events.jsonl"))
	agentEvents, err := evtLog.ReadByAgent(m.agent.Name)
	if err != nil || len(agentEvents) == 0 {
		m.recentEvents = nil
		return
	}
	// Keep only last N events
	if len(agentEvents) > agentMaxRecentEvents {
		agentEvents = agentEvents[len(agentEvents)-agentMaxRecentEvents:]
	}
	m.recentEvents = agentEvents
}

// View renders the agent detail screen.
func (m *AgentModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.agent.Name))
	b.WriteString("\n\n")

	if m.peekActive {
		b.WriteString(m.renderPeek())
		return b.String()
	}

	// Info section
	b.WriteString(m.renderInfo())

	// Stats section
	b.WriteString(m.renderStats())

	// Task list section
	b.WriteString(m.renderTaskList())

	// Recent activity section
	b.WriteString(m.renderRecentActivity())

	return b.String()
}

func (m *AgentModel) renderInfo() string {
	var b strings.Builder

	b.WriteString(m.styles.Bold.Render("Agent Info"))
	b.WriteString("\n")

	type field struct {
		label string
		value string
		style string
	}

	sessionName := m.agent.Session
	if m.manager != nil {
		sessionName = m.manager.Tmux().SessionName(m.agent.Session)
	}

	fields := []field{
		{"Name", m.agent.Name, ""},
		{"Role", string(m.agent.Role), ""},
		{"State", string(m.agent.State), mapState(m.agent.State)},
		{"Session", sessionName, "code"},
		{"Workspace", m.agent.Workspace, ""},
		{"Started", m.agent.StartedAt.Format(time.RFC3339), ""},
	}

	if m.agent.State != agent.StateStopped && !m.agent.StartedAt.IsZero() {
		fields = append(fields, field{"Uptime", fmtDuration(time.Since(m.agent.StartedAt)), ""})
	}

	if m.agent.Task != "" {
		fields = append(fields, field{"Task", m.agent.Task, ""})
	}

	for _, f := range fields {
		label := m.styles.Muted.Width(15).Render(f.label + ":")
		valueStyle := m.styles.Normal
		switch f.style {
		case "code":
			valueStyle = m.styles.Code
		case "ok":
			valueStyle = m.styles.Success
		case "error":
			valueStyle = m.styles.Error
		case "warning":
			valueStyle = m.styles.Warning
		case "info":
			valueStyle = m.styles.Info
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", label, valueStyle.Render(f.value)))
	}

	b.WriteString("\n")
	return b.String()
}

func (m *AgentModel) renderStats() string {
	var b strings.Builder

	b.WriteString(m.styles.Bold.Render("Task Stats"))
	b.WriteString("\n")

	totalAssigned := len(m.taskItems)
	completedLabel := fmt.Sprintf("%d", m.tasksCompleted)
	failedLabel := fmt.Sprintf("%d", m.tasksFailed)

	type field struct {
		label string
		value string
		style string
	}

	fields := []field{
		{"Assigned", fmt.Sprintf("%d", totalAssigned), ""},
		{"Completed", completedLabel, "ok"},
		{"Failed", failedLabel, ""},
	}

	if m.tasksFailed > 0 {
		fields[2].style = "error"
	}

	for _, f := range fields {
		label := m.styles.Muted.Width(15).Render(f.label + ":")
		valueStyle := m.styles.Normal
		switch f.style {
		case "ok":
			valueStyle = m.styles.Success
		case "error":
			valueStyle = m.styles.Error
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", label, valueStyle.Render(f.value)))
	}

	b.WriteString("\n")
	return b.String()
}

func (m *AgentModel) renderTaskList() string {
	var b strings.Builder

	// Show active/recent tasks
	active := m.activeAndRecentTasks()
	if len(active) == 0 {
		return ""
	}

	b.WriteString(m.styles.Bold.Render("Tasks"))
	b.WriteString("\n")

	for _, item := range active {
		title := item.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		line := fmt.Sprintf("  %-10s %-10s %s", item.ID, string(item.Status), title)
		b.WriteString(m.styles.StatusStyle(mapQueueStatus(item.Status)).Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	return b.String()
}

// activeAndRecentTasks returns non-done tasks first, then last few done/failed.
func (m *AgentModel) activeAndRecentTasks() []queue.WorkItem {
	var active, finished []queue.WorkItem
	for _, item := range m.taskItems {
		switch item.Status {
		case queue.StatusDone, queue.StatusFailed:
			finished = append(finished, item)
		default:
			active = append(active, item)
		}
	}
	// Show up to 5 recent finished tasks
	if len(finished) > 5 {
		finished = finished[len(finished)-5:]
	}
	return append(active, finished...)
}

func (m *AgentModel) renderRecentActivity() string {
	var b strings.Builder

	b.WriteString(m.styles.Bold.Render("Recent Activity"))
	b.WriteString("\n")

	if len(m.recentEvents) == 0 {
		b.WriteString(m.styles.Muted.Render("  No recent activity."))
		b.WriteString("\n")
		return b.String()
	}

	for _, ev := range m.recentEvents {
		ts := ev.Timestamp.Format("15:04:05")
		msg := string(ev.Type)
		if ev.Message != "" {
			msg = ev.Message
			if len(msg) > 60 {
				msg = msg[:57] + "..."
			}
		}
		line := fmt.Sprintf("  %s %-18s %s", ts, ev.Type, msg)
		b.WriteString(m.styles.Normal.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

func (m *AgentModel) renderPeek() string {
	var b strings.Builder

	b.WriteString(m.styles.Bold.Render("Recent Output"))
	b.WriteString("\n\n")

	if m.peekOutput == "" {
		b.WriteString(m.styles.Muted.Render("  No output captured"))
	} else {
		// Show output in code style
		for _, line := range strings.Split(m.peekOutput, "\n") {
			b.WriteString(m.styles.Code.Render("  " + line))
			b.WriteString("\n")
		}
	}

	return b.String()
}
