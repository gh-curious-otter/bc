package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// IssueModel shows the detail view for a single beads issue.
type IssueModel struct {
	// Scroll state
	contentLines []string

	issue  beads.Issue
	styles style.Styles

	width        int
	height       int
	scrollOffset int
}

// NewIssueModel creates an issue detail view.
func NewIssueModel(issue beads.Issue, s style.Styles) *IssueModel {
	m := &IssueModel{
		issue:  issue,
		styles: s,
	}
	m.buildContent()
	return m
}

// HandleKey processes a key event and returns an action for the parent.
func (m *IssueModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()
	switch key {
	case "esc":
		return Action{Type: ActionBack}
	case "j", "down":
		m.scrollDown()
		return NoAction
	case "k", "up":
		m.scrollUp()
		return NoAction
	case "g", "home":
		m.scrollOffset = 0
		return NoAction
	case "G", "end":
		m.scrollToEnd()
		return NoAction
	}
	return NoAction
}

func (m *IssueModel) scrollDown() {
	maxOffset := m.maxScrollOffset()
	if m.scrollOffset < maxOffset {
		m.scrollOffset++
	}
}

func (m *IssueModel) scrollUp() {
	if m.scrollOffset > 0 {
		m.scrollOffset--
	}
}

func (m *IssueModel) scrollToEnd() {
	m.scrollOffset = m.maxScrollOffset()
}

func (m *IssueModel) maxScrollOffset() int {
	viewHeight := m.viewableHeight()
	if len(m.contentLines) <= viewHeight {
		return 0
	}
	return len(m.contentLines) - viewHeight
}

func (m *IssueModel) viewableHeight() int {
	h := m.height - 4 // Reserve space for header and scroll indicator
	if h < 5 {
		h = 5
	}
	return h
}

// issueStatusStyle returns the style key for an issue status.
func issueStatusStyle(status string) string {
	switch status {
	case "open", "ready":
		return "ok"
	case "in_progress", "pending":
		return "warning"
	case "closed", "done", "resolved":
		return "info"
	default:
		return ""
	}
}

// buildContent renders the full issue detail into lines for scrolling.
func (m *IssueModel) buildContent() {
	var lines []string

	// Info section header
	lines = append(lines, m.styles.Bold.Render("Issue Info"))
	lines = append(lines, "")

	type field struct {
		label string
		value string
		style string
	}

	statusSt := issueStatusStyle(m.issue.Status)

	fields := []field{
		{"ID", m.issue.ID, "code"},
		{"Status", m.issue.Status, statusSt},
		{"Source", m.issue.Source, ""},
	}

	if m.issue.Type != "" {
		fields = append(fields, field{"Type", m.issue.Type, ""})
	}

	if m.issue.Assignee != "" {
		fields = append(fields, field{"Assignee", m.issue.Assignee, ""})
	}

	if m.issue.Priority != nil {
		fields = append(fields, field{"Priority", fmt.Sprintf("%v", m.issue.Priority), "warning"})
	}

	if len(m.issue.Dependencies) > 0 {
		fields = append(fields, field{"Dependencies", strings.Join(m.issue.Dependencies, ", "), "code"})
	}

	for _, f := range fields {
		label := m.styles.Muted.Width(15).Render(f.label + ":")
		valueStyle := m.styles.Normal
		switch f.style {
		case "code":
			valueStyle = m.styles.Code
		case "ok":
			valueStyle = m.styles.Success
		case "warning":
			valueStyle = m.styles.Warning
		case "error":
			valueStyle = m.styles.Error
		case "info":
			valueStyle = m.styles.Info
		}
		lines = append(lines, fmt.Sprintf("  %s %s", label, valueStyle.Render(f.value)))
	}

	// Description section
	if m.issue.Description != "" {
		lines = append(lines, "")
		lines = append(lines, m.styles.Bold.Render("Description"))
		lines = append(lines, "")
		for _, line := range strings.Split(m.issue.Description, "\n") {
			lines = append(lines, "  "+m.styles.Normal.Render(line))
		}
	}

	m.contentLines = lines
}

// View renders the issue detail screen with scroll support.
func (m *IssueModel) View() string {
	var b strings.Builder

	// Title (always visible, outside scroll area)
	b.WriteString(m.styles.Title.Render(m.issue.Title))
	b.WriteString("\n\n")

	// Scrollable content
	viewHeight := m.viewableHeight()
	end := m.scrollOffset + viewHeight
	if end > len(m.contentLines) {
		end = len(m.contentLines)
	}

	visible := m.contentLines[m.scrollOffset:end]
	for _, line := range visible {
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.contentLines) > viewHeight {
		pos := ""
		if m.scrollOffset == 0 {
			pos = "top"
		} else if m.scrollOffset >= m.maxScrollOffset() {
			pos = "end"
		} else {
			pct := float64(m.scrollOffset) / float64(m.maxScrollOffset()) * 100
			pos = fmt.Sprintf("%d%%", int(pct))
		}
		indicator := fmt.Sprintf("  -- %s (%d/%d lines) --", pos, m.scrollOffset+viewHeight, len(m.contentLines))
		b.WriteString("\n")
		b.WriteString(m.styles.Muted.Render(indicator))
	}

	return b.String()
}
