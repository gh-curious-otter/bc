package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/github"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// PRModel shows the detail view for a single GitHub pull request.
type PRModel struct {
	pr     github.PR
	styles style.Styles
	width  int
	height int
}

// NewPRModel creates a PR detail view.
func NewPRModel(pr github.PR, s style.Styles) *PRModel {
	return &PRModel{
		pr:     pr,
		styles: s,
	}
}

// HandleKey processes a key event and returns an action for the parent.
func (m *PRModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()
	switch key {
	case "esc":
		return Action{Type: ActionBack}
	}
	return NoAction
}

// View renders the PR detail screen.
func (m *PRModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.pr.Title))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Bold.Render("Pull Request Info"))
	b.WriteString("\n")

	stateStyle := "info"
	switch m.pr.State {
	case "open":
		stateStyle = "ok"
	case "merged":
		stateStyle = "info"
	case "closed":
		stateStyle = "warning"
	case "draft":
		stateStyle = "warning"
	}

	stateValue := m.pr.State
	if m.pr.IsDraft {
		stateValue = "draft"
		stateStyle = "warning"
	}

	fields := []struct {
		label string
		value string
		style string
	}{
		{"Number", fmt.Sprintf("#%d", m.pr.Number), "code"},
		{"State", stateValue, stateStyle},
	}

	if m.pr.ReviewDecision != "" {
		reviewStyle := "info"
		switch m.pr.ReviewDecision {
		case "APPROVED":
			reviewStyle = "ok"
		case "CHANGES_REQUESTED":
			reviewStyle = "error"
		}
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Review", m.pr.ReviewDecision, reviewStyle})
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

	return b.String()
}
