package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTableOnSelect(t *testing.T) {
	var selectedID string
	table := NewTableView("sel").
		Columns(Col("Name", 10)).
		Rows(
			Row{ID: "a", Values: []string{"Alpha"}},
			Row{ID: "b", Values: []string{"Beta"}},
		).
		OnSelect(func(row Row) Cmd {
			selectedID = row.ID
			return nil
		}).
		Build()

	// Press enter on first row
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	table.Update(msg)
	if selectedID != "a" {
		t.Errorf("expected selected ID 'a', got '%s'", selectedID)
	}

	// Move to second row and select
	table.MoveDown()
	table.Update(msg)
	if selectedID != "b" {
		t.Errorf("expected selected ID 'b', got '%s'", selectedID)
	}
}

func TestTableOnRender(t *testing.T) {
	table := NewTableView("custom").
		Columns(Col("Name", 10)).
		Rows(
			Row{ID: "1", Values: []string{"Test"}},
		).
		OnRender(func(row Row, index int, selected bool) string {
			prefix := "  "
			if selected {
				prefix = "> "
			}
			return prefix + row.Values[0]
		}).
		Build()

	output := table.View()
	if !strings.Contains(output, "> Test") {
		t.Errorf("expected custom rendered output with '> Test', got:\n%s", output)
	}
}

func TestTableCustomBindings(t *testing.T) {
	called := false
	table := NewTableView("bind").
		Columns(Col("Name", 10)).
		Rows(Row{ID: "1", Values: []string{"X"}}).
		Bind("r", "refresh", func() Cmd {
			called = true
			return nil
		}).
		Build()

	// Press 'r' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	table.Update(msg)

	if !called {
		t.Error("expected custom binding handler to be called for 'r'")
	}
}

func TestTableBindingsMethod(t *testing.T) {
	b1Called := false
	b2Called := false
	table := NewTableView("multi").
		Columns(Col("Name", 10)).
		Bindings(
			KeyBinding{Key: "a", Label: "action-a", Handler: func() Cmd { b1Called = true; return nil }},
			KeyBinding{Key: "b", Label: "action-b", Handler: func() Cmd { b2Called = true; return nil }},
		).
		Build()

	if len(table.bindings) != 2 {
		t.Errorf("expected 2 bindings, got %d", len(table.bindings))
	}

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	table.Update(msg)
	if !b1Called {
		t.Error("expected binding 'a' handler to be called")
	}

	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}}
	table.Update(msg)
	if !b2Called {
		t.Error("expected binding 'b' handler to be called")
	}
}

func TestTableWindowResize(t *testing.T) {
	table := NewTableView("resize").
		Columns(Col("Name", 10)).
		Build()

	msg := tea.WindowSizeMsg{Width: 100, Height: 30}
	table.Update(msg)

	if table.width != 100 {
		t.Errorf("expected width 100, got %d", table.width)
	}
	if table.height != 30 {
		t.Errorf("expected height 30, got %d", table.height)
	}
}

func TestTableStatusRowStyling(t *testing.T) {
	table := NewTableView("status").
		Columns(Col("Name", 10), Col("Status", 10)).
		Rows(
			Row{ID: "1", Values: []string{"Server", "running"}, Status: "ok"},
			Row{ID: "2", Values: []string{"DB", "error"}, Status: "error"},
			Row{ID: "3", Values: []string{"Cache", "pending"}, Status: "warning"},
		).
		Build()

	// Move cursor to row 2 so row 1 gets status styling
	table.MoveDown()
	output := table.View()
	if !strings.Contains(output, "Server") {
		t.Error("expected output to contain 'Server'")
	}
	if !strings.Contains(output, "DB") {
		t.Error("expected output to contain 'DB'")
	}
}

func TestTableColumnAlignment(t *testing.T) {
	table := NewTableView("align").
		Columns(
			Col("Left", 10),
			ColRight("Right", 10),
			ColCenter("Center", 10),
		).
		Rows(
			Row{ID: "1", Values: []string{"A", "B", "C"}},
		).
		Build()

	output := table.View()
	if !strings.Contains(output, "A") {
		t.Error("expected output to contain value 'A'")
	}
	if !strings.Contains(output, "B") {
		t.Error("expected output to contain value 'B'")
	}
}

func TestTableScrolling(t *testing.T) {
	rows := make([]Row, 20)
	for i := range rows {
		rows[i] = Row{ID: string(rune('a' + i)), Values: []string{string(rune('A' + i))}}
	}

	table := NewTableView("scroll").
		Columns(Col("Letter", 10)).
		Rows(rows...).
		Build()

	// Set a small height to force scrolling
	table.height = 5
	table.MoveToBottom()

	if table.cursor != 19 {
		t.Errorf("expected cursor at 19, got %d", table.cursor)
	}
	if table.offset <= 0 {
		t.Errorf("expected positive offset after scrolling to bottom, got %d", table.offset)
	}

	// Scroll back to top
	table.MoveToTop()
	if table.offset != 0 {
		t.Errorf("expected offset 0 after MoveToTop, got %d", table.offset)
	}
}

func TestTableFocusBlur(t *testing.T) {
	table := NewTableView("focus").
		Columns(Col("A", 5)).
		Build()

	if !table.focused {
		t.Error("expected table to be focused by default")
	}

	table.Blur()
	if table.focused {
		t.Error("expected table to be blurred after Blur()")
	}

	table.Focus()
	if !table.focused {
		t.Error("expected table to be focused after Focus()")
	}
}

func TestTableSetRowsClamsCursor(t *testing.T) {
	table := NewTableView("clamp").
		Columns(Col("A", 5)).
		Rows(
			Row{ID: "1", Values: []string{"X"}},
			Row{ID: "2", Values: []string{"Y"}},
			Row{ID: "3", Values: []string{"Z"}},
		).
		Build()

	// Move to last row
	table.MoveToBottom()
	if table.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", table.cursor)
	}

	// Replace with fewer rows - cursor should clamp
	table.SetRows([]Row{
		{ID: "a", Values: []string{"A"}},
	})
	if table.cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", table.cursor)
	}
}

func TestTableSetRowsEmpty(t *testing.T) {
	table := NewTableView("empty").
		Columns(Col("A", 5)).
		Rows(Row{ID: "1", Values: []string{"X"}}).
		Build()

	table.SetRows([]Row{})
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 for empty rows, got %d", table.cursor)
	}
	if table.SelectedRow() != nil {
		t.Error("expected nil SelectedRow for empty table")
	}
}

func TestTableKeyG(t *testing.T) {
	table := NewTableView("gkey").
		Columns(Col("A", 5)).
		Rows(
			Row{ID: "1", Values: []string{"X"}},
			Row{ID: "2", Values: []string{"Y"}},
			Row{ID: "3", Values: []string{"Z"}},
		).
		Build()

	// Move down first
	table.MoveDown()
	table.MoveDown()

	// 'g' should go to top
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	table.Update(msg)
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 after 'g', got %d", table.cursor)
	}

	// 'G' should go to bottom
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	table.Update(msg)
	if table.cursor != 2 {
		t.Errorf("expected cursor 2 after 'G', got %d", table.cursor)
	}
}

func TestTableMoveDownBounds(t *testing.T) {
	table := NewTableView("bounds").
		Columns(Col("A", 5)).
		Rows(
			Row{ID: "1", Values: []string{"X"}},
		).
		Build()

	table.MoveDown() // At last row, should not move
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 (can't move past last), got %d", table.cursor)
	}
}

func TestTableMoveUpBounds(t *testing.T) {
	table := NewTableView("bounds").
		Columns(Col("A", 5)).
		Rows(
			Row{ID: "1", Values: []string{"X"}},
		).
		Build()

	table.MoveUp() // At first row, should not move
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 (can't move before first), got %d", table.cursor)
	}
}

func TestTableRenderRowMissingValues(t *testing.T) {
	// Row has fewer values than columns
	table := NewTableView("missing").
		Columns(Col("A", 5), Col("B", 5), Col("C", 5)).
		Rows(
			Row{ID: "1", Values: []string{"X"}}, // Only 1 value for 3 columns
		).
		Build()

	output := table.View()
	if !strings.Contains(output, "X") {
		t.Error("expected output to contain 'X'")
	}
}

func TestTableVisibleRowCount(t *testing.T) {
	table := NewTableView("vis").
		Title("Test").
		Columns(Col("A", 5)).
		Build()

	// With height 0, should return default 10
	table.height = 0
	count := table.visibleRowCount()
	if count != 10 {
		t.Errorf("expected default visible rows 10, got %d", count)
	}

	// With title, reserved is 3 (title + header + border)
	table.height = 20
	count = table.visibleRowCount()
	if count != 17 {
		t.Errorf("expected 17 visible rows (20 - 3 reserved), got %d", count)
	}
}

func TestTableNoTitle(t *testing.T) {
	table := NewTableView("notitle").
		Columns(Col("A", 10)).
		Rows(Row{ID: "1", Values: []string{"Data"}}).
		Build()

	output := table.View()
	if !strings.Contains(output, "Data") {
		t.Error("expected output to contain row data")
	}
}

func TestTableEnterWithNoCallback(t *testing.T) {
	table := NewTableView("noselect").
		Columns(Col("A", 5)).
		Rows(Row{ID: "1", Values: []string{"X"}}).
		Build()

	// Enter without OnSelect should not panic
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	_, cmd := table.Update(msg)
	if cmd != nil {
		t.Error("expected nil cmd when no OnSelect callback")
	}
}
