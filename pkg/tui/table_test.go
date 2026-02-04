package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTableBuilder(t *testing.T) {
	table := NewTableView("test").
		Title("Test Table").
		Columns(
			Col("Name", 20),
			Col("Status", 10),
		).
		Rows(
			Row{ID: "1", Values: []string{"Alice", "active"}},
			Row{ID: "2", Values: []string{"Bob", "idle"}},
		).
		Build()

	if table.ID() != "test" {
		t.Errorf("expected ID 'test', got '%s'", table.ID())
	}

	if table.RowCount() != 2 {
		t.Errorf("expected 2 rows, got %d", table.RowCount())
	}

	if table.title != "Test Table" {
		t.Errorf("expected title 'Test Table', got '%s'", table.title)
	}
}

func TestTableNavigation(t *testing.T) {
	table := NewTableView("nav").
		Columns(Col("Item", 10)).
		Rows(
			Row{ID: "1", Values: []string{"A"}},
			Row{ID: "2", Values: []string{"B"}},
			Row{ID: "3", Values: []string{"C"}},
		).
		Build()

	// Initial position
	if table.cursor != 0 {
		t.Errorf("expected initial cursor 0, got %d", table.cursor)
	}

	// Move down
	table.MoveDown()
	if table.cursor != 1 {
		t.Errorf("expected cursor 1 after MoveDown, got %d", table.cursor)
	}

	// Move up
	table.MoveUp()
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 after MoveUp, got %d", table.cursor)
	}

	// Move to bottom
	table.MoveToBottom()
	if table.cursor != 2 {
		t.Errorf("expected cursor 2 after MoveToBottom, got %d", table.cursor)
	}

	// Move to top
	table.MoveToTop()
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 after MoveToTop, got %d", table.cursor)
	}
}

func TestTableKeyHandling(t *testing.T) {
	table := NewTableView("keys").
		Columns(Col("Item", 10)).
		Rows(
			Row{ID: "1", Values: []string{"A"}},
			Row{ID: "2", Values: []string{"B"}},
		).
		Build()

	// Test 'j' key moves down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	table.Update(msg)
	if table.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j' key, got %d", table.cursor)
	}

	// Test 'k' key moves up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	table.Update(msg)
	if table.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k' key, got %d", table.cursor)
	}
}

func TestTableView(t *testing.T) {
	table := NewTableView("view").
		Title("Agents").
		Columns(
			Col("Name", 10),
			Col("Status", 8),
		).
		Rows(
			Row{ID: "1", Values: []string{"Alice", "active"}},
		).
		Build()

	output := table.View()

	if !strings.Contains(output, "Agents") {
		t.Error("expected view to contain title 'Agents'")
	}
	if !strings.Contains(output, "Name") {
		t.Error("expected view to contain header 'Name'")
	}
	if !strings.Contains(output, "Alice") {
		t.Error("expected view to contain row value 'Alice'")
	}
}

func TestTableSelectedRow(t *testing.T) {
	table := NewTableView("select").
		Columns(Col("Name", 10)).
		Rows(
			Row{ID: "1", Values: []string{"First"}, Data: "data1"},
			Row{ID: "2", Values: []string{"Second"}, Data: "data2"},
		).
		Build()

	row := table.SelectedRow()
	if row == nil {
		t.Fatal("expected selected row, got nil")
	}
	if row.ID != "1" {
		t.Errorf("expected selected row ID '1', got '%s'", row.ID)
	}

	table.MoveDown()
	row = table.SelectedRow()
	if row.ID != "2" {
		t.Errorf("expected selected row ID '2', got '%s'", row.ID)
	}
}

func TestTableEmptyState(t *testing.T) {
	table := NewTableView("empty").
		Title("Empty Table").
		Columns(Col("Name", 10)).
		Build()

	output := table.View()
	if !strings.Contains(output, "No data") {
		t.Error("expected empty table to show 'No data'")
	}

	// Should not panic on navigation
	table.MoveDown()
	table.MoveUp()
	table.MoveToBottom()
	table.MoveToTop()

	if table.SelectedRow() != nil {
		t.Error("expected nil selected row in empty table")
	}
}

func TestTableSetRows(t *testing.T) {
	table := NewTableView("dynamic").
		Columns(Col("Name", 10)).
		Build()

	if table.RowCount() != 0 {
		t.Errorf("expected 0 rows initially, got %d", table.RowCount())
	}

	table.SetRows([]Row{
		{ID: "1", Values: []string{"New"}},
		{ID: "2", Values: []string{"Data"}},
	})

	if table.RowCount() != 2 {
		t.Errorf("expected 2 rows after SetRows, got %d", table.RowCount())
	}
}
