package tui

import (
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// queueField is a label/value pair for display in the queue item detail view.
type queueField struct {
	label string
	value string
	style string
}

// QueueItemModel shows the detail view for a single work queue item.
type QueueItemModel struct {
	item          queue.WorkItem
	styles        style.Styles
	width         int
	height        int
	workspacePath string
	branch        string
}

// NewQueueItemModel creates a queue item detail view.
func NewQueueItemModel(item queue.WorkItem, workspacePath string, s style.Styles) *QueueItemModel {
	m := &QueueItemModel{
		item:          item,
		styles:        s,
		workspacePath: workspacePath,
	}
	m.branch = m.findBranch()
	return m
}

// HandleKey processes a key event and returns an action for the parent.
func (m *QueueItemModel) HandleKey(msg tea.KeyMsg) Action {
	key := msg.String()
	switch key {
	case "esc":
		return Action{Type: ActionBack}
	}
	return NoAction
}

// findBranch looks up git branches matching this work item's ID or beads ID.
func (m *QueueItemModel) findBranch() string {
	if m.workspacePath == "" {
		return ""
	}

	// Search patterns: try item ID first, then beads ID
	patterns := []string{"*" + m.item.ID + "*"}
	if m.item.BeadsID != "" {
		patterns = append(patterns, "*"+m.item.BeadsID+"*")
	}

	for _, pattern := range patterns {
		cmd := exec.Command("git", "-C", m.workspacePath, "branch", "-a", "--list", pattern)
		out, err := cmd.Output()
		if err != nil || len(out) == 0 {
			continue
		}
		// Return first matching branch, trimmed
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			branch := strings.TrimSpace(line)
			branch = strings.TrimPrefix(branch, "* ")
			branch = strings.TrimPrefix(branch, "remotes/origin/")
			if branch != "" {
				return branch
			}
		}
	}
	return ""
}

// View renders the queue item detail screen.
func (m *QueueItemModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.item.Title))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Bold.Render("Queue Item Info"))
	b.WriteString("\n")

	statusStyle := mapQueueStatusStyle(m.item.Status)

	fields := []queueField{
		{"ID", m.item.ID, "code"},
		{"Status", string(m.item.Status), statusStyle},
	}

	if m.item.AssignedTo != "" {
		fields = append(fields, queueField{"Assigned To", m.item.AssignedTo, ""})
	}

	if m.item.BeadsID != "" {
		fields = append(fields, queueField{"Bead ID", m.item.BeadsID, "code"})
	}

	if m.branch != "" {
		fields = append(fields, queueField{"Branch", m.branch, "code"})
	}

	fields = append(fields, queueField{"Created", m.item.CreatedAt.Format("2006-01-02 15:04:05"), ""})
	fields = append(fields, queueField{"Updated", m.item.UpdatedAt.Format("2006-01-02 15:04:05"), ""})

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

	// Merge info section
	if m.item.Merge != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Bold.Render("Merge Info"))
		b.WriteString("\n")

		mergeStyle := mapMergeStatusStyle(m.item.Merge)
		mergeFields := []struct {
			label string
			value string
			style string
		}{
			{"Merge Status", string(m.item.Merge), mergeStyle},
		}

		if m.item.Branch != "" {
			mergeFields = append(mergeFields, struct {
				label string
				value string
				style string
			}{"Branch", m.item.Branch, "code"})
		}

		if m.item.MergeCommit != "" {
			mergeFields = append(mergeFields, struct {
				label string
				value string
				style string
			}{"Commit", m.item.MergeCommit, "code"})
		}

		if !m.item.MergedAt.IsZero() {
			mergeFields = append(mergeFields, struct {
				label string
				value string
				style string
			}{"Merged At", m.item.MergedAt.Format("2006-01-02 15:04:05"), ""})
		}

		for _, f := range mergeFields {
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
	}

	if m.item.Description != "" {
		b.WriteString("\n")
		b.WriteString(m.styles.Bold.Render("Description"))
		b.WriteString("\n")
		for _, line := range strings.Split(m.item.Description, "\n") {
			b.WriteString("  " + m.styles.Normal.Render(line) + "\n")
		}
	}

	return b.String()
}

func mapQueueStatusStyle(s queue.ItemStatus) string {
	switch s {
	case queue.StatusPending:
		return "warning"
	case queue.StatusAssigned:
		return "warning"
	case queue.StatusWorking:
		return "ok"
	case queue.StatusDone:
		return "ok"
	case queue.StatusFailed:
		return "error"
	default:
		return "info"
	}
}

func mapMergeStatusStyle(s queue.MergeStatus) string {
	switch s {
	case queue.MergeMerged:
		return "ok"
	case queue.MergeUnmerged:
		return "warning"
	case queue.MergeMerging:
		return "info"
	case queue.MergeConflict:
		return "error"
	default:
		return ""
	}
}
