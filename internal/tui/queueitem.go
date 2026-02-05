package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/tui/style"
)

// QueueItemModel shows the detail view for a single work queue item.
type QueueItemModel struct {
	item   queue.WorkItem
	styles style.Styles
	width  int
	height int
}

// NewQueueItemModel creates a queue item detail view.
func NewQueueItemModel(item queue.WorkItem, s style.Styles) *QueueItemModel {
	return &QueueItemModel{
		item:   item,
		styles: s,
	}
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

// View renders the queue item detail screen.
func (m *QueueItemModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Title.Render(m.item.Title))
	b.WriteString("\n\n")

	b.WriteString(m.styles.Bold.Render("Queue Item Info"))
	b.WriteString("\n")

	statusStyle := mapQueueStatusStyle(m.item.Status)

	fields := []struct {
		label string
		value string
		style string
	}{
		{"ID", m.item.ID, "code"},
		{"Status", string(m.item.Status), statusStyle},
	}

	if m.item.AssignedTo != "" {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Assigned To", m.item.AssignedTo, ""})
	}

	if m.item.BeadsID != "" {
		fields = append(fields, struct {
			label string
			value string
			style string
		}{"Bead ID", m.item.BeadsID, "code"})
	}

	fields = append(fields, struct {
		label string
		value string
		style string
	}{"Created", m.item.CreatedAt.Format("2006-01-02 15:04:05"), ""})

	fields = append(fields, struct {
		label string
		value string
		style string
	}{"Updated", m.item.UpdatedAt.Format("2006-01-02 15:04:05"), ""})

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
