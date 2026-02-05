package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/tui/style"
)

func newTestQueueItemModel(item queue.WorkItem) *QueueItemModel {
	return &QueueItemModel{
		item:   item,
		styles: style.DefaultStyles(),
		width:  120,
		height: 40,
	}
}

func TestQueueItemView_BasicFields(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:         "work-042",
		Title:      "Fix authentication bug",
		Status:     queue.StatusWorking,
		AssignedTo: "engineer-01",
		BeadsID:    "bd-007",
		CreatedAt:  time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2025, 1, 15, 14, 30, 0, 0, time.UTC),
	})

	output := m.View()

	for _, want := range []string{
		"work-042",
		"Fix authentication bug",
		"working",
		"engineer-01",
		"bd-007",
		"Queue Item Info",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}

func TestQueueItemView_MergeInfo(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:          "work-042",
		Title:       "Fix auth",
		Status:      queue.StatusDone,
		Merge:       queue.MergeMerged,
		Branch:      "engineer-01/work-042/fix-auth",
		MergeCommit: "abc123def456",
		MergedAt:    time.Date(2025, 1, 15, 16, 0, 0, 0, time.UTC),
	})

	output := m.View()

	for _, want := range []string{
		"Merge Info",
		"merged",
		"engineer-01/work-042/fix-auth",
		"abc123def456",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in merge info section", want)
		}
	}
}

func TestQueueItemView_NoMergeInfo(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "Pending task",
		Status: queue.StatusPending,
	})

	output := m.View()

	if strings.Contains(output, "Merge Info") {
		t.Errorf("should not show Merge Info for items without merge status")
	}
}

func TestQueueItemView_MergeConflict(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-042",
		Title:  "Conflicting task",
		Status: queue.StatusDone,
		Merge:  queue.MergeConflict,
		Branch: "feature/broken",
	})

	output := m.View()

	if !strings.Contains(output, "conflict") {
		t.Errorf("expected 'conflict' merge status")
	}
	if !strings.Contains(output, "feature/broken") {
		t.Errorf("expected branch name")
	}
}

func TestQueueItemView_MergeUnmerged(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-042",
		Title:  "Done but not merged",
		Status: queue.StatusDone,
		Merge:  queue.MergeUnmerged,
	})

	output := m.View()

	if !strings.Contains(output, "unmerged") {
		t.Errorf("expected 'unmerged' merge status")
	}
	// Should not show commit or merged time
	if strings.Contains(output, "Commit:") {
		t.Errorf("should not show commit for unmerged item")
	}
}

func TestQueueItemHandleKey_Esc(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "Test",
		Status: queue.StatusPending,
	})

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if action.Type != ActionBack {
		t.Errorf("expected ActionBack for esc, got %v", action.Type)
	}
}

func TestQueueItemHandleKey_Unknown(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "Test",
		Status: queue.StatusPending,
	})

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	if action.Type != ActionNone {
		t.Errorf("expected ActionNone for unknown key, got %v", action.Type)
	}
}

func TestMapMergeStatusStyle(t *testing.T) {
	tests := []struct {
		status queue.MergeStatus
		want   string
	}{
		{queue.MergeMerged, "ok"},
		{queue.MergeUnmerged, "warning"},
		{queue.MergeMerging, "info"},
		{queue.MergeConflict, "error"},
		{queue.MergeNone, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := mapMergeStatusStyle(tt.status)
			if got != tt.want {
				t.Errorf("mapMergeStatusStyle(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestMapQueueStatusStyle(t *testing.T) {
	tests := []struct {
		status queue.ItemStatus
		want   string
	}{
		{queue.StatusPending, "warning"},
		{queue.StatusAssigned, "warning"},
		{queue.StatusWorking, "ok"},
		{queue.StatusDone, "ok"},
		{queue.StatusFailed, "error"},
		{queue.ItemStatus("unknown"), "info"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := mapQueueStatusStyle(tt.status)
			if got != tt.want {
				t.Errorf("mapQueueStatusStyle(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestFindBranch_EmptyWorkspace(t *testing.T) {
	m := &QueueItemModel{
		item:          queue.WorkItem{ID: "work-001"},
		workspacePath: "",
	}
	if m.findBranch() != "" {
		t.Error("findBranch with empty workspace should return empty")
	}
}

func TestQueueItemView_NoAssignee(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "Unassigned task",
		Status: queue.StatusPending,
	})

	output := m.View()
	if strings.Contains(output, "Assigned To") {
		t.Error("should not show Assigned To when empty")
	}
}

func TestQueueItemView_NoBeadsID(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "Non-beads task",
		Status: queue.StatusPending,
	})

	output := m.View()
	if strings.Contains(output, "Bead ID") {
		t.Error("should not show Bead ID when empty")
	}
}

func TestQueueItemView_Description(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:          "work-001",
		Title:       "Task with desc",
		Status:      queue.StatusWorking,
		Description: "Line one\nLine two",
	})

	output := m.View()
	if !strings.Contains(output, "Description") {
		t.Error("expected Description section")
	}
	if !strings.Contains(output, "Line one") {
		t.Error("expected description content")
	}
}

func TestQueueItemView_NoDescription(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "No desc",
		Status: queue.StatusPending,
	})

	output := m.View()
	if strings.Contains(output, "Description") {
		t.Error("should not show Description section when empty")
	}
}

func TestQueueItemView_WithBranch(t *testing.T) {
	m := newTestQueueItemModel(queue.WorkItem{
		ID:     "work-001",
		Title:  "Task with branch",
		Status: queue.StatusWorking,
	})
	m.branch = "engineer-01/fix-bug"

	output := m.View()
	if !strings.Contains(output, "Branch") {
		t.Error("expected Branch field")
	}
	if !strings.Contains(output, "engineer-01/fix-bug") {
		t.Error("expected branch name")
	}
}
