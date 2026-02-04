package tui

import (
	"strings"
	"testing"
)

func TestAppBuilder(t *testing.T) {
	table := NewTableView("agents").
		Columns(Col("Name", 10)).
		Build()

	app := NewApp().
		Title("Test App").
		AddView("agents", table).
		Build()

	if app.title != "Test App" {
		t.Errorf("expected title 'Test App', got '%s'", app.title)
	}

	if app.activeView != "agents" {
		t.Errorf("expected active view 'agents', got '%s'", app.activeView)
	}

	if len(app.views) != 1 {
		t.Errorf("expected 1 view, got %d", len(app.views))
	}
}

func TestAppMultipleViews(t *testing.T) {
	app := NewApp().
		AddView("view1", NewTableView("v1").Columns(Col("A", 5)).Build()).
		AddView("view2", NewTableView("v2").Columns(Col("B", 5)).Build()).
		DefaultView("view2").
		Build()

	if app.activeView != "view2" {
		t.Errorf("expected default view 'view2', got '%s'", app.activeView)
	}

	app.NextView()
	if app.activeView != "view1" {
		t.Errorf("expected 'view1' after NextView, got '%s'", app.activeView)
	}

	app.PrevView()
	if app.activeView != "view2" {
		t.Errorf("expected 'view2' after PrevView, got '%s'", app.activeView)
	}
}

func TestAppView(t *testing.T) {
	table := NewTableView("test").
		Columns(Col("Name", 10)).
		Rows(Row{ID: "1", Values: []string{"Hello"}}).
		Build()

	app := NewApp().
		Title("My App").
		AddView("main", table).
		Build()

	output := app.View()

	if !strings.Contains(output, "My App") {
		t.Error("expected view to contain app title")
	}
	if !strings.Contains(output, "Hello") {
		t.Error("expected view to contain table data")
	}
	if !strings.Contains(output, "quit") {
		t.Error("expected view to contain quit hint in status bar")
	}
}

func TestAppGlobalBindings(t *testing.T) {
	app := NewApp().
		AddView("main", NewTableView("t").Columns(Col("X", 5)).Build()).
		Bind("?", "Help", func() Cmd {
			return nil
		}).
		Build()

	if len(app.globalBindings) != 1 {
		t.Errorf("expected 1 global binding, got %d", len(app.globalBindings))
	}

	if app.globalBindings[0].Key != "?" {
		t.Errorf("expected key '?', got '%s'", app.globalBindings[0].Key)
	}
}

func TestAppSetView(t *testing.T) {
	app := NewApp().
		AddView("v1", NewTableView("t1").Build()).
		AddView("v2", NewTableView("t2").Build()).
		Build()

	app.SetView("v2")
	if app.ActiveView() != "v2" {
		t.Errorf("expected active view 'v2', got '%s'", app.ActiveView())
	}

	// Invalid view should be ignored
	app.SetView("nonexistent")
	if app.ActiveView() != "v2" {
		t.Errorf("expected active view to remain 'v2', got '%s'", app.ActiveView())
	}
}
