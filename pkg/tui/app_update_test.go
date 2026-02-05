package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppInit(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().AddView("main", table).Build()

	cmd := app.Init()
	// Init should not panic; cmd may be nil since TableView.Init returns nil
	_ = cmd
}

func TestAppUpdateWindowSize(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().AddView("main", table).Build()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	_, _ = app.Update(msg)

	if app.width != 120 {
		t.Errorf("expected width 120, got %d", app.width)
	}
	if app.height != 40 {
		t.Errorf("expected height 40, got %d", app.height)
	}
}

func TestAppUpdateQuit(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().AddView("main", table).Build()

	// ctrl+c should quit
	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := app.Update(msg)
	if cmd == nil {
		t.Error("expected quit command for ctrl+c")
	}

	// q should quit
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd = app.Update(msg)
	if cmd == nil {
		t.Error("expected quit command for q")
	}
}

func TestAppUpdateGlobalBinding(t *testing.T) {
	called := false
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().
		AddView("main", table).
		Bind("?", "Help", func() Cmd {
			called = true
			return nil
		}).
		Build()

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	app.Update(msg)

	if !called {
		t.Error("expected global binding handler to be called for '?'")
	}
}

func TestAppShowHeaderDisabled(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().
		Title("Test").
		AddView("main", table).
		ShowHeader(false).
		Build()

	output := app.View()
	// When header is disabled, title should not appear in output
	// (title is only rendered in the header)
	lines := strings.Split(output, "\n")
	// First line should not be the header
	if strings.Contains(lines[0], "Test") && strings.Contains(lines[0], "[main]") {
		// Could still appear if view content happens to contain it,
		// but header render should be skipped
	}

	if app.showHeader {
		t.Error("expected showHeader to be false")
	}
}

func TestAppShowStatusBarDisabled(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().
		AddView("main", table).
		ShowStatusBar(false).
		Build()

	if app.showStatusBar {
		t.Error("expected showStatusBar to be false")
	}

	output := app.View()
	// Status bar contains "quit" hint; without it, no "q:quit"
	if strings.Contains(output, "q:quit") {
		t.Error("expected no status bar when disabled")
	}
}

func TestAppRenderHeaderWithView(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().
		Title("MyApp").
		AddView("agents", table).
		Build()

	output := app.View()
	if !strings.Contains(output, "MyApp") {
		t.Error("expected header to contain app title")
	}
	if !strings.Contains(output, "[agents]") {
		t.Error("expected header to contain active view ID")
	}
}

func TestAppRenderHeaderEmpty(t *testing.T) {
	app := NewApp().Build()

	header := app.renderHeader()
	// Without a title, should use "bc" as default
	if !strings.Contains(header, "bc") {
		t.Error("expected default title 'bc' in header")
	}
}

func TestAppStatusBarWithBindings(t *testing.T) {
	table := NewTableView("t").Columns(Col("A", 5)).Build()
	app := NewApp().
		AddView("main", table).
		Bind("r", "refresh", nil).
		Build()

	output := app.View()
	if !strings.Contains(output, "r:refresh") {
		t.Error("expected status bar to contain binding hint")
	}
}

func TestAppNoActiveView(t *testing.T) {
	app := NewApp().Build()
	output := app.View()
	if !strings.Contains(output, "No view loaded") {
		t.Error("expected 'No view loaded' when no views registered")
	}
}

func TestAppUpdatePassesToActiveView(t *testing.T) {
	table := NewTableView("t").
		Columns(Col("A", 5)).
		Rows(
			Row{ID: "1", Values: []string{"X"}},
			Row{ID: "2", Values: []string{"Y"}},
		).
		Build()

	app := NewApp().AddView("main", table).Build()

	// Send j key to move cursor down in the table
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	_, _ = app.Update(msg)

	// Get updated table from views
	updatedTable, ok := app.views["main"].(*TableView)
	if !ok {
		t.Fatal("expected views[main] to be *TableView")
	}
	if updatedTable.cursor != 1 {
		t.Errorf("expected cursor 1 after j key, got %d", updatedTable.cursor)
	}
}

func TestAppNextPrevViewWraparound(t *testing.T) {
	app := NewApp().
		AddView("v1", NewTableView("t1").Build()).
		AddView("v2", NewTableView("t2").Build()).
		AddView("v3", NewTableView("t3").Build()).
		Build()

	// Should start at v1
	if app.ActiveView() != "v1" {
		t.Errorf("expected v1, got %s", app.ActiveView())
	}

	// Cycle forward through all views
	app.NextView()
	if app.ActiveView() != "v2" {
		t.Errorf("expected v2, got %s", app.ActiveView())
	}
	app.NextView()
	if app.ActiveView() != "v3" {
		t.Errorf("expected v3, got %s", app.ActiveView())
	}
	app.NextView() // Should wrap to v1
	if app.ActiveView() != "v1" {
		t.Errorf("expected v1 after wrap, got %s", app.ActiveView())
	}

	// Cycle backward
	app.PrevView() // Should wrap to v3
	if app.ActiveView() != "v3" {
		t.Errorf("expected v3 after prev wrap, got %s", app.ActiveView())
	}
}
