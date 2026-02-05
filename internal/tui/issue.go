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
	issue  beads.Issue
	styles style.Styles
	width  int
	height int
}

// NewIssueModel creates an issue detail view.
func NewIssueModel(issue beads.Issue, s style.Styles) *IssueModel {
	return &IssueModel{
		issue:  issue,
		styles: s,
	}
}

// HandleKey processes a key event and returns an action for the parent.
func (m *IssueModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()
	switch key {
	case "esc":
		return Action{Type: ActionBack}
	}
	return NoAction
}

// View renders the issue detail screen.
func (m *IssueModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.issue.Title))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Bold.Render("Issue Info"))
	b.WriteString("\n")

	statusStyle := "info"
	switch m.issue.Status {
	case "open", "ready":
		statusStyle = "ok"
	case "closed", "done":
		statusStyle = "warning"
	}

	fields := []struct {
		label string
		value string
		style string
	}{
		{"ID", m.issue.ID, "code"},
		{"Status", m.issue.Status, statusStyle},
		{"Source", m.issue.Source, ""},
	}

	if m.issue.Type != "" {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Type", m.issue.Type, ""})
	}

	if m.issue.Assignee != "" {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Assignee", m.issue.Assignee, ""})
	}

	if m.issue.Priority != nil {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Priority", fmt.Sprintf("%v", m.issue.Priority), ""})
	}

	if len(m.issue.Dependencies) > 0 {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Dependencies", strings.Join(m.issue.Dependencies, ", "), "code"})
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
		b.WriteString(fmt.Sprintf("  %s %s\n", label, valueStyle.Render(f.value)))
	}

	if m.issue.Description != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Bold.Render("Description"))
		b.WriteString("\n")
		for _, line := range strings.Split(m.issue.Description, "\n") {
			b.WriteString("  " + m.styles.Normal.Render(line) + "\n")
		}
	}

	return b.String()
}
