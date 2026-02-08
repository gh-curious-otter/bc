package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// ChannelModel shows the detail view for a single channel.
type ChannelModel struct {
	channel *channel.Channel
	store   *channel.Store
	manager *agent.Manager

	styles        style.Styles
	workspacePath string
	input         string
	sendMsg       string // status message after send

	width  int
	height int
	// Scroll position (index of first visible message from end)
	scroll int
	// Message selection cursor
	cursor int

	sendMode bool
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
	switch msg.Type {
	case tea.KeyRunes:
		m.input += string(msg.Runes)
	case tea.KeySpace:
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

	// Calculate widths
	divWidth := m.width - 2
	if divWidth < 20 {
		divWidth = 20
	}

	// ═══════════════════════════════════════════════════════════════════════
	// CHANNEL HEADER
	// ═══════════════════════════════════════════════════════════════════════

	// Channel name prominently displayed
	b.WriteString(m.styles.Title.Render("  # " + m.channel.Name))
	b.WriteString("\n")

	// Description/topic (if set)
	if m.channel.Description != "" {
		b.WriteString(m.styles.Muted.Render("  " + m.channel.Description))
		b.WriteString("\n")
	}

	// Member count with online/active indicators
	totalMembers := len(m.channel.Members)
	activeCount := 0
	if m.manager != nil {
		for _, member := range m.channel.Members {
			if a := m.manager.GetAgent(member); a != nil && a.State != agent.StateStopped {
				activeCount++
			}
		}
	}

	// Member stats line
	b.WriteString("  ")
	if activeCount > 0 {
		b.WriteString(m.styles.Success.Render(fmt.Sprintf("● %d online", activeCount)))
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf(" / %d members", totalMembers)))
	} else {
		b.WriteString(m.styles.Muted.Render(fmt.Sprintf("○ %d members", totalMembers)))
	}

	// Quick actions hint
	b.WriteString(m.styles.Muted.Render("  │  "))
	b.WriteString(m.styles.Info.Render("[s]"))
	b.WriteString(m.styles.Muted.Render("end  "))
	b.WriteString(m.styles.Info.Render("[r]"))
	b.WriteString(m.styles.Muted.Render("efresh  "))
	b.WriteString(m.styles.Info.Render("[i]"))
	b.WriteString(m.styles.Muted.Render("ssue  "))
	b.WriteString(m.styles.Info.Render("[esc]"))
	b.WriteString(m.styles.Muted.Render("back"))
	b.WriteString("\n")

	// Header divider
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

		var lastDate time.Time
		visibleHistory := m.channel.History[start:end]
		for i, entry := range visibleHistory {
			// Add date separator if this is a new day
			if i == 0 || !isSameDay(entry.Time, lastDate) {
				if i > 0 {
					b.WriteString("\n")
				}
				dateSep := formatDateSeparator(entry.Time)
				sepLine := fmt.Sprintf("── %s ──", dateSep)
				b.WriteString("  ")
				b.WriteString(m.styles.Muted.Render(sepLine))
				b.WriteString("\n")
			}
			lastDate = entry.Time

			if i > 0 {
				b.WriteString("\n")
			}

			// Infer message type for styling
			msgType := channel.InferMessageType(entry.Message)
			msgTypeStr := string(msgType)

			// Sender and timestamp line with relative time
			sender := entry.Sender
			if sender == "" {
				sender = "system"
			}
			relTime := formatRelativeTime(entry.Time)
			b.WriteString("  ")

			// Add message type icon if not a regular text message
			icon := m.styles.MessageTypeIcon(msgTypeStr)
			if icon != "" {
				b.WriteString(icon)
			}

			// Render sender with role-specific color
			role := style.RoleFromAgentName(sender)
			senderStyle := m.styles.RoleStyle(role)
			b.WriteString(senderStyle.Render(sender))

			// Add role badge
			if role != sender && role != "" {
				b.WriteString(" ")
				b.WriteString(m.styles.RoleBadge(role).Render(role))
			}

			b.WriteString("  ")
			b.WriteString(m.styles.Muted.Render(relTime))
			b.WriteString("\n")

			// Message content — wrap long lines with type-specific styling
			msgStyle := m.styles.MessageTypeStyle(msgTypeStr)
			lines := wrapText(entry.Message, msgWidth)
			for _, line := range lines {
				b.WriteString("  ")
				b.WriteString(msgStyle.Render("  " + line))
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
		var lastDate time.Time
		recentHistory := m.channel.History[start:]
		for i, entry := range recentHistory {
			// Add date separator if this is a new day
			if i == 0 || !isSameDay(entry.Time, lastDate) {
				dateSep := formatDateSeparator(entry.Time)
				b.WriteString(m.styles.Muted.Render(fmt.Sprintf("  ─── %s ───", dateSep)))
				b.WriteString("\n")
			}
			lastDate = entry.Time

			selected := i == m.cursor

			// Infer message type for styling
			msgType := channel.InferMessageType(entry.Message)
			msgTypeStr := string(msgType)
			icon := m.styles.MessageTypeIcon(msgTypeStr)
			relTime := formatRelativeTime(entry.Time)

			var line string
			if entry.Sender != "" {
				line = fmt.Sprintf("  %s%s  [%s] %s", icon, relTime, entry.Sender, entry.Message)
			} else {
				line = fmt.Sprintf("  %s%s  %s", icon, relTime, entry.Message)
			}
			if selected {
				b.WriteString(m.styles.Selected.Render(line))
			} else {
				ts := m.styles.Muted.Render(relTime)
				msgStyle := m.styles.MessageTypeStyle(msgTypeStr)
				msg := msgStyle.Render(entry.Message)
				if entry.Sender != "" {
					// Use role-specific color for sender
					role := style.RoleFromAgentName(entry.Sender)
					senderStyle := m.styles.RoleStyle(role)
					sender := senderStyle.Render("[" + entry.Sender + "]")
					line = fmt.Sprintf("  %s%s  %s %s", icon, ts, sender, msg)
				} else {
					line = fmt.Sprintf("  %s%s  %s", icon, ts, msg)
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
		b.WriteString(m.styles.Muted.Render("█"))
		b.WriteString("  ")
		b.WriteString(m.styles.Muted.Render("Enter to send • Esc to cancel"))
		b.WriteString("\n")
	} else if m.sendMsg != "" {
		b.WriteString(m.styles.Success.Render("  ✓ " + m.sendMsg))
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

// formatRelativeTime returns a human-readable relative time string.
// For times within the last hour, shows minutes (e.g., "5m ago").
// For times within the last day, shows hours (e.g., "3h ago").
// For older times, shows the absolute time (e.g., "15:04").
func formatRelativeTime(t time.Time) string {
	return formatRelativeTimeFrom(t, time.Now())
}

// formatRelativeTimeFrom returns a relative time string compared to a reference time.
// This is useful for testing with a fixed reference time.
func formatRelativeTimeFrom(t, now time.Time) string {
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < 48*time.Hour:
		return "yesterday " + t.Format("15:04")
	default:
		return t.Format("Jan 2 15:04")
	}
}

// isSameDay returns true if two times are on the same calendar day.
func isSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// formatDateSeparator returns a formatted date separator string.
func formatDateSeparator(t time.Time) string {
	return formatDateSeparatorFrom(t, time.Now())
}

// formatDateSeparatorFrom returns a formatted date separator compared to a reference time.
func formatDateSeparatorFrom(t, now time.Time) string {
	if isSameDay(t, now) {
		return "Today"
	}
	if isSameDay(t, now.AddDate(0, 0, -1)) {
		return "Yesterday"
	}
	return t.Format("Monday, Jan 2, 2006")
}
