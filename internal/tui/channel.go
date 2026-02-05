package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// ChannelModel shows the detail view for a single channel.
type ChannelModel struct {
	channel       *channel.Channel
	store         *channel.Store
	manager       *agent.Manager
	workspacePath string
	styles        style.Styles
	width         int
	height        int

	// Message selection cursor
	cursor int

	// Send message mode
	sendMode bool
	input    string
	sendMsg  string // status message after send
}

// NewChannelModel creates a channel detail view.
func NewChannelModel(ch *channel.Channel, store *channel.Store, mgr *agent.Manager, wsPath string, s style.Styles) *ChannelModel {
	return &ChannelModel{
		channel:       ch,
		store:         store,
		manager:       mgr,
		workspacePath: wsPath,
		styles:        s,
	}
}

// HandleKey processes a key event and returns an action for the parent.
func (m *ChannelModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()

	if m.sendMode {
		return m.handleSendKey(msg)
	}

	switch key {
	case "esc":
		return Action{Type: ActionBack}
	case "j", "down":
		if m.cursor < m.visibleCount()-1 {
			m.cursor++
		}
		return NoAction
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return NoAction
	case "s":
		m.sendMode = true
		m.input = ""
		m.sendMsg = ""
		return NoAction
	case "r":
		m.reloadChannel()
		return NoAction
	case "i":
		if entry, ok := m.selectedMessage(); ok {
			return Action{Type: ActionCreateIssue, Data: entry}
		}
		return NoAction
	}

	return NoAction
}

// visibleCount returns the number of messages currently displayed.
func (m *ChannelModel) visibleCount() int {
	n := len(m.channel.History)
	if n > 20 {
		return 20
	}
	return n
}

// selectedMessage returns the currently selected history entry.
func (m *ChannelModel) selectedMessage() (channel.HistoryEntry, bool) {
	n := len(m.channel.History)
	if n == 0 {
		return channel.HistoryEntry{}, false
	}
	start := 0
	if n > 20 {
		start = n - 20
	}
	visible := m.channel.History[start:]
	if m.cursor < 0 || m.cursor >= len(visible) {
		return channel.HistoryEntry{}, false
	}
	return visible[m.cursor], true
}

func (m *ChannelModel) handleSendKey(msg tea.KeyMsg) Action {
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
			m.sendMessage(m.input)
		}
		m.sendMode = false
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

func (m *ChannelModel) sendMessage(message string) {
	members, err := m.store.GetMembers(m.channel.Name)
	if err != nil {
		m.sendMsg = "Error: " + err.Error()
		return
	}

	if len(members) == 0 {
		m.sendMsg = "No members in channel"
		return
	}

	// Record in history
	sender := os.Getenv("BC_AGENT_ID")
	if err := m.store.AddHistory(m.channel.Name, sender, message); err != nil {
		m.sendMsg = "Error recording history: " + err.Error()
		return
	}
	_ = m.store.Save()

	// Send to all members
	sent := 0
	for _, member := range members {
		a := m.manager.GetAgent(member)
		if a == nil || a.State == agent.StateStopped {
			continue
		}
		if err := m.manager.SendToAgent(member, fmt.Sprintf("[#%s] %s", m.channel.Name, message)); err != nil {
			continue
		}
		sent++
	}

	m.sendMsg = fmt.Sprintf("Sent to %d/%d members", sent, len(members))
	m.reloadChannel()
}

func (m *ChannelModel) reloadChannel() {
	if err := m.store.Load(); err != nil {
		return
	}
	if ch, ok := m.store.Get(m.channel.Name); ok {
		m.channel = ch
	}
}

// View renders the channel detail screen.
func (m *ChannelModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render("#" + m.channel.Name))
	b.WriteString("\n\n")

	// Members section
	b.WriteString(m.styles.Bold.Render("Members"))
	b.WriteString("\n")
	if len(m.channel.Members) == 0 {
		b.WriteString(m.styles.Muted.Render("  No members"))
		b.WriteString("\n")
	} else {
		for _, member := range m.channel.Members {
			b.WriteString(m.styles.Normal.Render("  " + member))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// History section
	b.WriteString(m.styles.Bold.Render("Recent Messages"))
	b.WriteString("\n")
	if len(m.channel.History) == 0 {
		b.WriteString(m.styles.Muted.Render("  No messages"))
		b.WriteString("\n")
	} else {
		// Show last 20 messages
		start := 0
		if len(m.channel.History) > 20 {
			start = len(m.channel.History) - 20
		}
		for i, entry := range m.channel.History[start:] {
			selected := i == m.cursor
			var line string
			if entry.Sender != "" {
				line = fmt.Sprintf("  %s  [%s] %s", entry.Time.Format("15:04:05"), entry.Sender, entry.Message)
			} else {
				line = fmt.Sprintf("  %s  %s", entry.Time.Format("15:04:05"), entry.Message)
			}
			if selected {
				b.WriteString(m.styles.Selected.Render(line))
			} else {
				ts := m.styles.Muted.Render(entry.Time.Format("15:04:05"))
				msg := m.styles.Normal.Render(entry.Message)
				if entry.Sender != "" {
					sender := m.styles.Info.Render("[" + entry.Sender + "]")
					line = fmt.Sprintf("  %s  %s %s", ts, sender, msg)
				} else {
					line = fmt.Sprintf("  %s  %s", ts, msg)
				}
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	// Send mode or status
	b.WriteString("\n")
	if m.sendMode {
		prompt := m.styles.Info.Render("  > ")
		b.WriteString(prompt)
		b.WriteString(m.styles.Normal.Render(m.input))
		b.WriteString(m.styles.Muted.Render("_"))
		b.WriteString("\n")
	} else if m.sendMsg != "" {
		b.WriteString(m.styles.Success.Render("  " + m.sendMsg))
		b.WriteString("\n")
	}

	return b.String()
}
