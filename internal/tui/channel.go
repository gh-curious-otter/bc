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

	// Scroll position (index of first visible message from end)
	scroll int
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
		if m.scroll > 0 {
			m.scroll--
		}
		if m.cursor < m.visibleCount()-1 {
			m.cursor++
		}
		return NoAction
	case "k", "up":
		maxScroll := len(m.channel.History) - m.visibleMsgCount()
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.scroll < maxScroll {
			m.scroll++
		}
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
	case "g", "home":
		maxScroll := len(m.channel.History) - m.visibleMsgCount()
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.scroll = maxScroll
		return NoAction
	case "G", "end":
		m.scroll = 0
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

// visibleMsgCount returns how many messages fit in the visible area.
func (m *ChannelModel) visibleMsgCount() int {
	// Reserve lines: title(1) + blank(1) + members(1) + member list + blank(1)
	// + divider(1) + blank(1) + input/status(2) + scroll hint(1)
	overhead := 8 + len(m.channel.Members)
	// Each message takes ~3 lines (sender+time, content, blank separator)
	available := m.height - overhead
	if available < 3 {
		available = 3
	}
	return available / 3
}

// View renders the channel detail screen.
func (m *ChannelModel) View() string {
	var b strings.Builder

	// Channel header
	b.WriteString(m.styles.Title.Render("#" + m.channel.Name))

	// Inline members
	if len(m.channel.Members) > 0 {
		memberList := strings.Join(m.channel.Members, ", ")
		b.WriteString("  ")
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf("(%d members: %s)", len(m.channel.Members), memberList)))
	}
	b.WriteString("\n")

	// Divider
	divWidth := m.width - 2
	if divWidth < 20 {
		divWidth = 20
	}
	b.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", divWidth)))
	b.WriteString("\n")

	// Message history
	if len(m.channel.History) == 0 {
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render("  No messages yet. Press 's' to send a message."))
		b.WriteString("\n")
	} else {
		visible := m.visibleMsgCount()
		total := len(m.channel.History)

		// Calculate visible window based on scroll offset from end
		end := total - m.scroll
		start := end - visible
		if start < 0 {
			start = 0
		}
		if end < 0 {
			end = 0
		}

		// Scroll indicator (top)
		if start > 0 {
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ▲ %d older messages", start)))
			b.WriteString("\n")
		}

		msgWidth := m.width - 6
		if msgWidth < 30 {
			msgWidth = 30
		}

		for i, entry := range m.channel.History[start:end] {
			if i > 0 {
				b.WriteString("\n")
			}

			// Sender and timestamp line
			sender := entry.Sender
			if sender == "" {
				sender = "system"
			}
			ts := entry.Time.Format("15:04:05")
			b.WriteString("  ")
			b.WriteString(m.styles.Info.Render(sender))
			b.WriteString("  ")
			b.WriteString(m.styles.Muted.Render(ts))
			b.WriteString("\n")

			// Message content — wrap long lines
			lines := wrapText(entry.Message, msgWidth)
			for _, line := range lines {
				b.WriteString("  ")
				b.WriteString(m.styles.Normal.Render("  " + line))
				b.WriteString("\n")
			}
		}

		// Scroll indicator (bottom)
		if m.scroll > 0 {
			b.WriteString("\n")
			b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ▼ %d newer messages", m.scroll)))
			b.WriteString("\n")
		}
	}

	// Bottom divider
	b.WriteString(m.styles.Muted.Render("  " + strings.Repeat("─", divWidth)))
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
	if m.sendMode {
		prompt := m.styles.Info.Render("  > ")
		b.WriteString(prompt)
		b.WriteString(m.styles.Normal.Render(m.input))
		b.WriteString(m.styles.Muted.Render("_"))
		b.WriteString("\n")
	} else if m.sendMsg != "" {
		b.WriteString(m.styles.Success.Render("  " + m.sendMsg))
		b.WriteString("\n")
	} else {
		b.WriteString(m.styles.Muted.Render("  [s]end  [j/k]scroll  [r]efresh  [esc]back"))
		b.WriteString("\n")
	}

	return b.String()
}

// wrapText splits text into lines of at most width characters, breaking at spaces.
func wrapText(text string, width int) []string {
	if width <= 0 || len(text) <= width {
		return []string{text}
	}
	var lines []string
	for len(text) > width {
		// Find last space within width
		i := strings.LastIndex(text[:width], " ")
		if i <= 0 {
			i = width
		}
		lines = append(lines, text[:i])
		text = strings.TrimLeft(text[i:], " ")
	}
	if text != "" {
		lines = append(lines, text)
	}
	return lines
}
