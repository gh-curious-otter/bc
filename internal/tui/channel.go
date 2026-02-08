package tui

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// AutocompleteType identifies the type of autocomplete being shown.
type AutocompleteType int

const (
	AutocompleteNone AutocompleteType = iota
	AutocompleteMention
	AutocompleteChannel
)

// ChannelModel shows the detail view for a single channel.
type ChannelModel struct {
	// Pointers first for better alignment
	channel *channel.Channel
	store   *channel.Store
	manager *agent.Manager

	styles style.Styles

	// String fields
	workspacePath      string
	input              string
	sendMsg            string // status message after send
	autocompletePrefix string // The text after @ or # being matched

	// Slice field
	autocompleteSuggestions []string

	// Int fields
	width                int
	height               int
	scroll               int // Scroll position (index of first visible message from end)
	cursor               int // Message selection cursor
	autocompleteSelected int

	// Small types last
	autocompleteType AutocompleteType
	sendMode         bool
	reactionMode     bool // Show reaction picker for selected message
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

	if m.reactionMode {
		return m.handleReactionKey(msg)
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
	case "e": // Open reaction picker
		if len(m.channel.History) > 0 {
			m.reactionMode = true
			m.sendMsg = ""
		}
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

// selectedMessageIndex returns the index of the selected message in the full history.
func (m *ChannelModel) selectedMessageIndex() int {
	n := len(m.channel.History)
	if n == 0 {
		return -1
	}
	start := 0
	if n > 20 {
		start = n - 20
	}
	if m.cursor < 0 || m.cursor >= n-start {
		return -1
	}
	return start + m.cursor
}

func (m *ChannelModel) handleReactionKey(msg tea.KeyMsg) Action {
	key := msg.String()

	switch key {
	case "esc":
		m.reactionMode = false
		return NoAction
	case "1":
		m.addReactionToSelected("👍")
		return NoAction
	case "2":
		m.addReactionToSelected("👎")
		return NoAction
	case "3":
		m.addReactionToSelected("❤️")
		return NoAction
	case "4":
		m.addReactionToSelected("🎉")
		return NoAction
	case "5":
		m.addReactionToSelected("👀")
		return NoAction
	case "6":
		m.addReactionToSelected("🚀")
		return NoAction
	}

	return NoAction
}

func (m *ChannelModel) addReactionToSelected(emoji string) {
	idx := m.selectedMessageIndex()
	if idx < 0 {
		return
	}

	user := os.Getenv("BC_AGENT_ID")
	if user == "" {
		user = "anonymous"
	}

	added, err := m.store.ToggleReaction(m.channel.Name, idx, emoji, user)
	if err != nil {
		m.sendMsg = "Error: " + err.Error()
	} else if added {
		m.sendMsg = fmt.Sprintf("Added %s reaction", emoji)
	} else {
		m.sendMsg = fmt.Sprintf("Removed %s reaction", emoji)
	}

	_ = m.store.Save()
	m.reloadChannel()
	m.reactionMode = false
}

func (m *ChannelModel) handleSendKey(msg tea.KeyMsg) Action {
	key := msg.String()

	// Handle autocomplete navigation first
	if m.autocompleteType != AutocompleteNone {
		switch key {
		case "esc":
			m.dismissAutocomplete()
			return NoAction
		case "up":
			if m.autocompleteSelected > 0 {
				m.autocompleteSelected--
			}
			return NoAction
		case "down":
			if m.autocompleteSelected < len(m.autocompleteSuggestions)-1 {
				m.autocompleteSelected++
			}
			return NoAction
		case "tab":
			m.selectAutocomplete()
			return NoAction
		}
		if isEnterKey(msg) {
			m.selectAutocomplete()
			return NoAction
		}
	}

	switch key {
	case "esc":
		m.sendMode = false
		m.input = ""
		m.dismissAutocomplete()
		return NoAction
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
			m.updateAutocomplete()
		}
		return NoAction
	case "ctrl+enter":
		// Ctrl+Enter sends the message
		if m.input != "" {
			m.sendMessage(m.input)
		}
		m.sendMode = false
		m.dismissAutocomplete()
		return NoAction
	}

	// Regular Enter adds a new line (multi-line support)
	if isEnterKey(msg) {
		m.input += "\n"
		m.dismissAutocomplete()
		return NoAction
	}

	// Append typed characters
	switch msg.Type {
	case tea.KeyRunes:
		m.input += string(msg.Runes)
		m.updateAutocomplete()
	case tea.KeySpace:
		m.input += " "
		m.dismissAutocomplete()
	}

	return NoAction
}

// updateAutocomplete checks input for @ or # triggers and updates suggestions.
func (m *ChannelModel) updateAutocomplete() {
	// Find the last @ or # in input that starts a word
	input := m.input
	atIdx := strings.LastIndex(input, "@")
	hashIdx := strings.LastIndex(input, "#")

	// Check if @ is at start or after whitespace/newline
	if atIdx >= 0 && (atIdx == 0 || input[atIdx-1] == ' ' || input[atIdx-1] == '\n') {
		prefix := input[atIdx+1:]
		// Only if no space after @
		if !strings.ContainsAny(prefix, " \n") {
			m.autocompleteType = AutocompleteMention
			m.autocompletePrefix = prefix
			m.autocompleteSuggestions = m.getMentionSuggestions(prefix)
			m.autocompleteSelected = 0
			return
		}
	}

	// Check if # is at start or after whitespace/newline
	if hashIdx >= 0 && (hashIdx == 0 || input[hashIdx-1] == ' ' || input[hashIdx-1] == '\n') {
		prefix := input[hashIdx+1:]
		// Only if no space after #
		if !strings.ContainsAny(prefix, " \n") {
			m.autocompleteType = AutocompleteChannel
			m.autocompletePrefix = prefix
			m.autocompleteSuggestions = m.getChannelSuggestions(prefix)
			m.autocompleteSelected = 0
			return
		}
	}

	m.dismissAutocomplete()
}

// getMentionSuggestions returns agent names matching the prefix.
func (m *ChannelModel) getMentionSuggestions(prefix string) []string {
	var suggestions []string
	prefix = strings.ToLower(prefix)

	// Add @all as first suggestion if it matches
	if strings.HasPrefix("all", prefix) {
		suggestions = append(suggestions, "all")
	}

	// Add channel members that match
	for _, member := range m.channel.Members {
		if prefix == "" || strings.HasPrefix(strings.ToLower(member), prefix) {
			suggestions = append(suggestions, member)
		}
	}

	// Also add agents from manager if available
	if m.manager != nil {
		for _, a := range m.manager.ListAgents() {
			name := a.Name
			if prefix == "" || strings.HasPrefix(strings.ToLower(name), prefix) {
				// Avoid duplicates
				found := false
				for _, s := range suggestions {
					if s == name {
						found = true
						break
					}
				}
				if !found {
					suggestions = append(suggestions, name)
				}
			}
		}
	}

	// Limit to 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// getChannelSuggestions returns channel names matching the prefix.
func (m *ChannelModel) getChannelSuggestions(prefix string) []string {
	var suggestions []string
	prefix = strings.ToLower(prefix)

	if m.store != nil {
		for _, ch := range m.store.List() {
			if prefix == "" || strings.HasPrefix(strings.ToLower(ch.Name), prefix) {
				suggestions = append(suggestions, ch.Name)
			}
		}
	}

	// Limit to 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

// selectAutocomplete inserts the selected suggestion.
func (m *ChannelModel) selectAutocomplete() {
	if m.autocompleteType == AutocompleteNone || len(m.autocompleteSuggestions) == 0 {
		return
	}

	selected := m.autocompleteSuggestions[m.autocompleteSelected]

	// Find the trigger position and replace
	var trigger string
	if m.autocompleteType == AutocompleteMention {
		trigger = "@"
	} else {
		trigger = "#"
	}

	// Find the last occurrence of the trigger
	idx := strings.LastIndex(m.input, trigger)
	if idx >= 0 {
		m.input = m.input[:idx] + trigger + selected + " "
	}

	m.dismissAutocomplete()
}

// dismissAutocomplete clears the autocomplete state.
func (m *ChannelModel) dismissAutocomplete() {
	m.autocompleteType = AutocompleteNone
	m.autocompleteSuggestions = nil
	m.autocompleteSelected = 0
	m.autocompletePrefix = ""
}

// renderAutocomplete renders the autocomplete popup.
func (m *ChannelModel) renderAutocomplete() string {
	var b strings.Builder

	// Header based on type
	var header string
	if m.autocompleteType == AutocompleteMention {
		header = "Mention"
	} else {
		header = "Channel"
	}
	b.WriteString(m.styles.Muted.Render("  ┌─ " + header + " "))
	b.WriteString(m.styles.Muted.Render(strings.Repeat("─", 20)))
	b.WriteString("\n")

	// Suggestions
	for i, suggestion := range m.autocompleteSuggestions {
		selected := i == m.autocompleteSelected
		var prefix string
		if m.autocompleteType == AutocompleteMention {
			prefix = "@"
		} else {
			prefix = "#"
		}

		line := "  │ " + prefix + suggestion
		if selected {
			b.WriteString(m.styles.Selected.Render(line))
		} else {
			b.WriteString(m.styles.Normal.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString(m.styles.Muted.Render("  └" + strings.Repeat("─", 25)))
	b.WriteString("\n")

	return b.String()
}

// renderInputArea renders the multi-line input area with send hint.
func (m *ChannelModel) renderInputArea() string {
	var b strings.Builder

	// Split input into lines
	lines := strings.Split(m.input, "\n")

	// Show up to 5 lines, scroll if more
	maxLines := 5
	startLine := 0
	if len(lines) > maxLines {
		startLine = len(lines) - maxLines
	}
	visibleLines := lines[startLine:]

	// Render each line with prompt on first line
	for i, line := range visibleLines {
		if i == 0 && startLine == 0 {
			// First line with prompt
			b.WriteString(m.styles.Info.Render("  > "))
		} else {
			// Continuation lines with indent
			b.WriteString(m.styles.Muted.Render("  │ "))
		}
		b.WriteString(m.styles.Normal.Render(line))

		// Cursor on the last line
		if i == len(visibleLines)-1 {
			b.WriteString(m.styles.Muted.Render("█"))
		}
		b.WriteString("\n")
	}

	// If no input yet, show placeholder
	if m.input == "" {
		b.WriteString(m.styles.Info.Render("  > "))
		b.WriteString(m.styles.Muted.Render("Type a message... (@mention, #channel)"))
		b.WriteString(m.styles.Muted.Render("█"))
		b.WriteString("\n")
	}

	// Show keyboard hints
	b.WriteString(m.styles.Muted.Render("  Ctrl+Enter to send • Enter for new line • Esc to cancel"))
	b.WriteString("\n")

	return b.String()
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

	// Record in history with consistent **[agent]** prefix (#292)
	sender := os.Getenv("BC_AGENT_ID")
	if sender == "" {
		sender = "tui"
	}
	formatted := channel.FormatAgentComment(sender, message)
	if err := m.store.AddHistory(m.channel.Name, sender, formatted); err != nil {
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

			// Infer message type for styling
			msgType := channel.InferMessageType(entry.Message)
			msgTypeStr := string(msgType)

			// Sender info
			sender := entry.Sender
			if sender == "" {
				sender = "system"
			}

			// Check if this is a continuation from the same sender (message grouping)
			isGrouped := i > 0 && visibleHistory[i-1].Sender == entry.Sender && isSameDay(entry.Time, visibleHistory[i-1].Time)

			if !isGrouped {
				// Add spacing between different senders
				if i > 0 {
					b.WriteString("\n")
				}

				// Sender and timestamp header
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
			}

			// Message content with subtle background — wrap long lines with highlighting
			msgStyle := m.styles.MessageTypeStyle(msgTypeStr)
			lines := wrapText(entry.Message, msgWidth-4)
			var content strings.Builder
			for j, line := range lines {
				if j > 0 {
					content.WriteString("\n")
				}
				highlightedLine := m.highlightMessage(line)
				content.WriteString(highlightedLine)
			}

			// Render message with subtle background bubble
			bubble := m.styles.MessageBubble.Width(msgWidth).Inherit(msgStyle).Render(content.String())
			for _, line := range strings.Split(bubble, "\n") {
				b.WriteString("  ")
				b.WriteString(line)
				b.WriteString("\n")
			}

			// Display reactions if any
			if reactionStr := m.formatReactions(entry.Reactions); reactionStr != "" {
				b.WriteString("    ")
				b.WriteString(reactionStr)
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
				highlightedMsg := m.highlightMessage(entry.Message)
				if entry.Sender != "" {
					// Use role-specific color for sender
					role := style.RoleFromAgentName(entry.Sender)
					senderStyle := m.styles.RoleStyle(role)
					sender := senderStyle.Render("[" + entry.Sender + "]")
					line = fmt.Sprintf("  %s%s  %s %s", icon, ts, sender, highlightedMsg)
				} else {
					line = fmt.Sprintf("  %s%s  %s", icon, ts, highlightedMsg)
				}
				b.WriteString(line)
			}
			// Display reactions inline if any
			if reactionStr := m.formatReactions(entry.Reactions); reactionStr != "" {
				b.WriteString("  ")
				b.WriteString(reactionStr)
			}
			b.WriteString("\n")
		}
	}

	// Send mode, reaction mode, or status
	if m.sendMode {
		// Render autocomplete popup if active
		if m.autocompleteType != AutocompleteNone && len(m.autocompleteSuggestions) > 0 {
			b.WriteString(m.renderAutocomplete())
		}

		// Render multi-line input area
		b.WriteString(m.renderInputArea())
	} else if m.reactionMode {
		b.WriteString(m.styles.Info.Render("  React: "))
		b.WriteString("[1]👍  [2]👎  [3]❤️  [4]🎉  [5]👀  [6]🚀  [esc]cancel")
		b.WriteString("\n")
	} else if m.sendMsg != "" {
		b.WriteString(m.styles.Success.Render("  ✓ " + m.sendMsg))
		b.WriteString("\n")
	} else {
		b.WriteString(m.styles.Muted.Render("  [s]end  [e]moji  [j/k]scroll  [r]efresh  [esc]back"))
		b.WriteString("\n")
	}

	return b.String()
}

// highlightMessage applies syntax highlighting to a message.
// It highlights @mentions, #channels, and GitHub issue/PR links.
func (m *ChannelModel) highlightMessage(message string) string {
	return channel.ApplyHighlights(message, func(text string, highlightType channel.HighlightType) string {
		var s lipgloss.Style
		switch highlightType {
		case channel.HighlightMention:
			s = m.styles.Mention
		case channel.HighlightChannel:
			s = m.styles.Channel
		case channel.HighlightGitHubLink:
			s = m.styles.Link
		default:
			s = m.styles.Normal
		}
		return s.Render(text)
	})
}

// formatReactions formats a reactions map for display.
func (m *ChannelModel) formatReactions(reactions map[string][]string) string {
	if len(reactions) == 0 {
		return ""
	}

	var parts []string
	// Sort emojis for consistent display
	for _, emoji := range channel.CommonReactions {
		if users, ok := reactions[emoji]; ok && len(users) > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", emoji, len(users)))
		}
	}
	// Add any other emojis not in CommonReactions
	for emoji, users := range reactions {
		if !slices.Contains(channel.CommonReactions, emoji) && len(users) > 0 {
			parts = append(parts, fmt.Sprintf("%s %d", emoji, len(users)))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return m.styles.Reaction.Render(strings.Join(parts, "  "))
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
