package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// AgentModel shows the detail view for a single agent.
type AgentModel struct {
	agent   *agent.Agent
	manager *agent.Manager
	styles  style.Styles
	width   int
	height  int

	// Peek output
	peekOutput string
	peekActive bool
}

// NewAgentModel creates an agent detail view.
func NewAgentModel(a *agent.Agent, mgr *agent.Manager, s style.Styles) *AgentModel {
	return &AgentModel{
		agent:   a,
		manager: mgr,
		styles:  s,
	}
}

// HandleKey processes a key event and returns an action for the parent.
func (m *AgentModel) HandleKey(msg tea.KeyMsg) Action {
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
	case "r":
		m.refresh()
		m.loadPeek()
		return NoAction
	}

	return NoAction
}

// refresh reloads the agent's state data from the manager.
func (m *AgentModel) refresh() {
	m.manager.RefreshState()
	if a := m.manager.GetAgent(m.agent.Name); a != nil {
		m.agent = a
	}
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
	b.WriteString(m.styles.Bold.Render("Agent Info"))
	b.WriteString("\n")

	fields := []struct {
		label string
		value string
		style string
	}{
		{"Name", m.agent.Name, ""},
		{"Role", string(m.agent.Role), ""},
		{"State", string(m.agent.State), mapState(m.agent.State)},
		{"Session", m.manager.Tmux().SessionName(m.agent.Session), "code"},
		{"Workspace", m.agent.Workspace, ""},
		{"Started", m.agent.StartedAt.Format(time.RFC3339), ""},
	}

	if m.agent.State != agent.StateStopped {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Uptime", fmtDuration(time.Since(m.agent.StartedAt)), ""})
	}

	if m.agent.Task != "" {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Task", m.agent.Task, ""})
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
