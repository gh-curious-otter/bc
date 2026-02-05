package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/tui/style"
)

func newTestIssueModel(issue beads.Issue) *IssueModel {
	m := NewIssueModel(issue, style.DefaultStyles())
	m.width = 120
	m.height = 40
	return m
}

func TestIssueView_BasicFields(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:     "bd-001",
		Title:  "Fix login bug",
		Status: "open",
		Source: "beads",
	})

	output := m.View()

	if !strings.Contains(output, "Fix login bug") {
		t.Errorf("expected title, got: %s", output)
	}
	if !strings.Contains(output, "bd-001") {
		t.Errorf("expected ID, got: %s", output)
	}
	if !strings.Contains(output, "open") {
		t.Errorf("expected status, got: %s", output)
	}
	if !strings.Contains(output, "beads") {
		t.Errorf("expected source, got: %s", output)
	}
	if !strings.Contains(output, "Issue Info") {
		t.Errorf("expected 'Issue Info' header, got: %s", output)
	}
}

func TestIssueView_AllFields(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:           "bd-042",
		Title:        "Add dark mode",
		Status:       "in_progress",
		Source:       "beads",
		Type:         "task",
		Assignee:     "engineer-01",
		Priority:     "P1",
		Dependencies: []string{"bd-001", "bd-002"},
		Description:  "Implement dark mode toggle in the settings panel.",
	})

	output := m.View()

	for _, want := range []string{
		"bd-042", "Add dark mode", "in_progress", "task",
		"engineer-01", "P1", "bd-001", "bd-002",
		"Description", "dark mode toggle",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got: %s", want, output)
		}
	}
}

func TestIssueView_Description(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:          "bd-001",
		Title:       "Test issue",
		Status:      "open",
		Description: "Line one\nLine two\nLine three",
	})

	output := m.View()

	if !strings.Contains(output, "Description") {
		t.Errorf("expected Description header, got: %s", output)
	}
	if !strings.Contains(output, "Line one") {
		t.Errorf("expected 'Line one', got: %s", output)
	}
	if !strings.Contains(output, "Line three") {
		t.Errorf("expected 'Line three', got: %s", output)
	}
}

func TestIssueView_NoDescription(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:     "bd-001",
		Title:  "No desc",
		Status: "open",
	})

	output := m.View()

	if strings.Contains(output, "Description") {
		t.Errorf("should not show Description section when empty, got: %s", output)
	}
}

// --- Scroll tests ---

func TestIssueScroll_Down(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:          "bd-001",
		Title:       "Scrollable issue",
		Status:      "open",
		Description: strings.Repeat("Line\n", 100), // Many lines
	})
	m.height = 15 // Small viewport to trigger scrolling

	if m.scrollOffset != 0 {
		t.Errorf("initial scroll offset should be 0, got %d", m.scrollOffset)
	}

	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.scrollOffset != 1 {
		t.Errorf("after j, scroll offset should be 1, got %d", m.scrollOffset)
	}

	m.HandleKey(tea.KeyMsg{Type: tea.KeyDown})
	if m.scrollOffset != 2 {
		t.Errorf("after down, scroll offset should be 2, got %d", m.scrollOffset)
	}
}

func TestIssueScroll_Up(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:          "bd-001",
		Title:       "Scrollable",
		Status:      "open",
		Description: strings.Repeat("Line\n", 100),
	})
	m.height = 15
	m.scrollOffset = 5

	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.scrollOffset != 4 {
		t.Errorf("after k, scroll offset should be 4, got %d", m.scrollOffset)
	}

	m.HandleKey(tea.KeyMsg{Type: tea.KeyUp})
	if m.scrollOffset != 3 {
		t.Errorf("after up, scroll offset should be 3, got %d", m.scrollOffset)
	}
}

func TestIssueScroll_UpAtTop(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:     "bd-001",
		Title:  "Short",
		Status: "open",
	})

	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.scrollOffset != 0 {
		t.Errorf("scrolling up at top should stay at 0, got %d", m.scrollOffset)
	}
}

func TestIssueScroll_HomeAndEnd(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:          "bd-001",
		Title:       "Scrollable",
		Status:      "open",
		Description: strings.Repeat("Line\n", 100),
	})
	m.height = 15

	// Go to end
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.scrollOffset == 0 {
		t.Error("G should scroll to end, but offset is still 0")
	}
	maxOff := m.maxScrollOffset()
	if m.scrollOffset != maxOff {
		t.Errorf("G should set offset to max %d, got %d", maxOff, m.scrollOffset)
	}

	// Go to beginning
	m.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.scrollOffset != 0 {
		t.Errorf("g should reset to 0, got %d", m.scrollOffset)
	}
}

func TestIssueScroll_IndicatorShown(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:          "bd-001",
		Title:       "Scrollable",
		Status:      "open",
		Description: strings.Repeat("Line\n", 100),
	})
	m.height = 15

	output := m.View()

	if !strings.Contains(output, "top") {
		t.Errorf("expected scroll indicator with 'top', got: %s", output)
	}
	if !strings.Contains(output, "lines") {
		t.Errorf("expected 'lines' in scroll indicator, got: %s", output)
	}
}

func TestIssueScroll_NoIndicatorWhenFits(t *testing.T) {
	m := newTestIssueModel(beads.Issue{
		ID:     "bd-001",
		Title:  "Short issue",
		Status: "open",
	})
	m.height = 40 // Tall viewport

	output := m.View()

	if strings.Contains(output, "lines") {
		t.Errorf("should not show scroll indicator when content fits, got: %s", output)
	}
}

func TestIssueEsc_ReturnsBack(t *testing.T) {
	m := newTestIssueModel(beads.Issue{ID: "bd-001", Status: "open"})

	action := m.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if action.Type != ActionBack {
		t.Errorf("esc should return ActionBack, got %v", action.Type)
	}
}

// --- Status styling tests ---

func TestIssueStatusStyle(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"open", "ok"},
		{"ready", "ok"},
		{"in_progress", "warning"},
		{"pending", "warning"},
		{"closed", "info"},
		{"done", "info"},
		{"resolved", "info"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := issueStatusStyle(tt.status)
			if got != tt.want {
				t.Errorf("issueStatusStyle(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}
