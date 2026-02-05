package runtime

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewDriver(t *testing.T) {
	d := NewDriver()
	if d.title != "bc" {
		t.Errorf("expected default title 'bc', got '%s'", d.title)
	}
	if d.version != "dev" {
		t.Errorf("expected default version 'dev', got '%s'", d.version)
	}
	if d.renderer == nil {
		t.Error("expected renderer to be initialized")
	}
}

func TestDriverWithIO(t *testing.T) {
	d := NewDriver()
	var in bytes.Buffer
	var out bytes.Buffer

	d.WithIO(&in, &out)
	if d.input != &in {
		t.Error("expected input to be set")
	}
	if d.output != &out {
		t.Error("expected output to be set")
	}
}

func TestDriverWithTitle(t *testing.T) {
	d := NewDriver().WithTitle("myapp")
	if d.title != "myapp" {
		t.Errorf("expected title 'myapp', got '%s'", d.title)
	}
}

func TestDriverUpdateWindowSize(t *testing.T) {
	d := NewDriver()
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	_, _ = d.Update(msg)

	if d.width != 100 {
		t.Errorf("expected width 100, got %d", d.width)
	}
	if d.height != 50 {
		t.Errorf("expected height 50, got %d", d.height)
	}
}

func TestDriverHandleViewMessage(t *testing.T) {
	d := NewDriver()

	t.Run("table view", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{
			Type:    MsgView,
			View:    ViewTable,
			ID:      "agents",
			Title:   "Agents",
			Loading: true,
		})

		if d.currentView != ViewTable {
			t.Errorf("expected ViewTable, got %v", d.currentView)
		}
		if d.tableSpec == nil {
			t.Fatal("expected tableSpec to be set")
		}
		if d.tableSpec.ID != "agents" {
			t.Errorf("expected ID 'agents', got '%s'", d.tableSpec.ID)
		}
		if !d.tableSpec.Loading {
			t.Error("expected Loading to be true")
		}
		if d.detailSpec != nil {
			t.Error("expected detailSpec to be nil after switching to table")
		}
	})

	t.Run("detail view", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{
			Type:  MsgView,
			View:  ViewDetail,
			ID:    "agent-detail",
			Title: "worker-01",
		})

		if d.currentView != ViewDetail {
			t.Errorf("expected ViewDetail, got %v", d.currentView)
		}
		if d.detailSpec == nil {
			t.Fatal("expected detailSpec to be set")
		}
		if d.tableSpec != nil {
			t.Error("expected tableSpec to be nil after switching to detail")
		}
	})

	t.Run("modal view", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{
			Type:  MsgView,
			View:  ViewModal,
			ID:    "confirm",
			Title: "Are you sure?",
		})

		if d.currentView != ViewModal {
			t.Errorf("expected ViewModal, got %v", d.currentView)
		}
		if d.modalSpec == nil {
			t.Fatal("expected modalSpec to be set")
		}
	})

	t.Run("cursor resets on view change", func(t *testing.T) {
		d.cursor = 5
		d.handleViewMessage(&ViewMessage{
			View: ViewTable,
			ID:   "new",
		})
		if d.cursor != 0 {
			t.Errorf("expected cursor reset to 0, got %d", d.cursor)
		}
	})
}

func TestDriverHandleSetMessage(t *testing.T) {
	d := NewDriver()

	// Set up table view
	d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "t"})

	t.Run("set title", func(t *testing.T) {
		d.handleSetMessage(&SetMessage{Path: "title", Value: "New Title"})
		if d.tableSpec.Title != "New Title" {
			t.Errorf("expected title 'New Title', got '%s'", d.tableSpec.Title)
		}
	})

	t.Run("set loading", func(t *testing.T) {
		d.handleSetMessage(&SetMessage{Path: "loading", Value: true})
		if !d.tableSpec.Loading {
			t.Error("expected loading to be true")
		}
	})

	t.Run("set columns", func(t *testing.T) {
		cols := []ColumnSpec{{Name: "A", Width: 10}, {Name: "B", Width: 20}}
		d.handleSetMessage(&SetMessage{Path: "columns", Value: cols})
		if len(d.tableSpec.Columns) != 2 {
			t.Errorf("expected 2 columns, got %d", len(d.tableSpec.Columns))
		}
	})

	t.Run("set rows", func(t *testing.T) {
		rows := []RowSpec{{ID: "1", Values: []string{"x"}}}
		d.handleSetMessage(&SetMessage{Path: "rows", Value: rows})
		if len(d.tableSpec.Rows) != 1 {
			t.Errorf("expected 1 row, got %d", len(d.tableSpec.Rows))
		}
	})

	t.Run("set bindings", func(t *testing.T) {
		bindings := []BindingSpec{{Key: "r", Label: "refresh", Action: "refresh"}}
		d.handleSetMessage(&SetMessage{Path: "bindings", Value: bindings})
		if len(d.tableSpec.Bindings) != 1 {
			t.Errorf("expected 1 binding, got %d", len(d.tableSpec.Bindings))
		}
	})
}

func TestDriverHandleSetMessageDetail(t *testing.T) {
	d := NewDriver()
	d.handleViewMessage(&ViewMessage{View: ViewDetail, ID: "d"})

	t.Run("set detail title", func(t *testing.T) {
		d.handleSetMessage(&SetMessage{Path: "title", Value: "Detail Title"})
		if d.detailSpec.Title != "Detail Title" {
			t.Errorf("expected 'Detail Title', got '%s'", d.detailSpec.Title)
		}
	})

	t.Run("set detail loading", func(t *testing.T) {
		d.handleSetMessage(&SetMessage{Path: "loading", Value: true})
		if !d.detailSpec.Loading {
			t.Error("expected loading to be true")
		}
	})

	t.Run("set detail sections", func(t *testing.T) {
		sections := []SectionSpec{{Title: "Info", Fields: []FieldSpec{{Label: "K", Value: "V"}}}}
		d.handleSetMessage(&SetMessage{Path: "sections", Value: sections})
		if len(d.detailSpec.Sections) != 1 {
			t.Errorf("expected 1 section, got %d", len(d.detailSpec.Sections))
		}
	})
}

func TestDriverHandleAppendMessage(t *testing.T) {
	d := NewDriver()

	t.Run("append row to table", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "t"})
		d.handleAppendMessage(&AppendMessage{
			Path:  "rows",
			Value: RowSpec{ID: "1", Values: []string{"row1"}},
		})
		if len(d.tableSpec.Rows) != 1 {
			t.Errorf("expected 1 row, got %d", len(d.tableSpec.Rows))
		}

		d.handleAppendMessage(&AppendMessage{
			Path:  "rows",
			Value: RowSpec{ID: "2", Values: []string{"row2"}},
		})
		if len(d.tableSpec.Rows) != 2 {
			t.Errorf("expected 2 rows, got %d", len(d.tableSpec.Rows))
		}
	})

	t.Run("append field to detail", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{View: ViewDetail, ID: "d"})
		d.handleAppendMessage(&AppendMessage{
			Path:  "sections",
			Value: FieldSpec{Label: "Name", Value: "test"},
		})
		if len(d.detailSpec.Sections) != 1 {
			t.Errorf("expected 1 section auto-created, got %d", len(d.detailSpec.Sections))
		}
		if len(d.detailSpec.Sections[0].Fields) != 1 {
			t.Errorf("expected 1 field, got %d", len(d.detailSpec.Sections[0].Fields))
		}
	})

	t.Run("append to non-rows path ignored", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "t"})
		d.handleAppendMessage(&AppendMessage{
			Path:  "columns",
			Value: ColumnSpec{Name: "X"},
		})
		// Should not crash; columns isn't handled in append for table
	})
}

func TestDriverHandleIncoming(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out

	t.Run("view message", func(t *testing.T) {
		data, _ := json.Marshal(ViewMessage{Type: MsgView, View: ViewTable, ID: "t"})
		d.handleIncoming(incomingMessage{data: data})
		if d.currentView != ViewTable {
			t.Error("expected table view after incoming view message")
		}
	})

	t.Run("set message", func(t *testing.T) {
		data, _ := json.Marshal(SetMessage{Type: MsgSet, Path: "title", Value: "Hi"})
		d.handleIncoming(incomingMessage{data: data})
		if d.tableSpec.Title != "Hi" {
			t.Errorf("expected title 'Hi', got '%s'", d.tableSpec.Title)
		}
	})

	t.Run("append message", func(t *testing.T) {
		data, _ := json.Marshal(AppendMessage{Type: MsgAppend, Path: "rows", Value: RowSpec{ID: "1", Values: []string{"x"}}})
		d.handleIncoming(incomingMessage{data: data})
		if len(d.tableSpec.Rows) != 1 {
			t.Errorf("expected 1 row, got %d", len(d.tableSpec.Rows))
		}
	})

	t.Run("done message clears loading", func(t *testing.T) {
		d.tableSpec.Loading = true
		data, _ := json.Marshal(DoneMessage{Type: MsgDone})
		d.handleIncoming(incomingMessage{data: data})
		if d.tableSpec.Loading {
			t.Error("expected loading to be cleared after done")
		}
	})

	t.Run("error message", func(t *testing.T) {
		data, _ := json.Marshal(ErrorMessage{Type: MsgError, Message: "something failed"})
		d.handleIncoming(incomingMessage{data: data})
		// Should not crash
	})

	t.Run("invalid json", func(t *testing.T) {
		d.handleIncoming(incomingMessage{data: []byte("{invalid}")})
		// Should not crash
	})
}

func TestDriverHandleKeyTableNavigation(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out

	d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "t"})
	d.tableSpec.Rows = []RowSpec{
		{ID: "1", Values: []string{"A"}},
		{ID: "2", Values: []string{"B"}},
		{ID: "3", Values: []string{"C"}},
	}

	t.Run("j moves down", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		d.handleKey(msg)
		if d.cursor != 1 {
			t.Errorf("expected cursor 1, got %d", d.cursor)
		}
	})

	t.Run("k moves up", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		d.handleKey(msg)
		if d.cursor != 0 {
			t.Errorf("expected cursor 0, got %d", d.cursor)
		}
	})

	t.Run("G moves to bottom", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
		d.handleKey(msg)
		if d.cursor != 2 {
			t.Errorf("expected cursor 2, got %d", d.cursor)
		}
	})

	t.Run("g moves to top", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
		d.handleKey(msg)
		if d.cursor != 0 {
			t.Errorf("expected cursor 0, got %d", d.cursor)
		}
	})

	t.Run("down bound check", func(t *testing.T) {
		d.cursor = 2
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
		d.handleKey(msg)
		if d.cursor != 2 {
			t.Errorf("expected cursor 2 (at bottom), got %d", d.cursor)
		}
	})

	t.Run("up bound check", func(t *testing.T) {
		d.cursor = 0
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		d.handleKey(msg)
		if d.cursor != 0 {
			t.Errorf("expected cursor 0 (at top), got %d", d.cursor)
		}
	})
}

func TestDriverHandleKeyQuit(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out

	msg := tea.KeyMsg{Type: tea.KeyCtrlC}
	_, cmd := d.handleKey(msg)
	if cmd == nil {
		t.Error("expected quit command for ctrl+c")
	}
}

func TestDriverHandleKeySendsEvent(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out

	// Set up table view with a row
	d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "agents"})
	d.tableSpec.Rows = []RowSpec{{ID: "1", Values: []string{"w1"}}}

	// Press a non-navigation key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	_, cmd := d.handleKey(msg)

	// Execute the command to get the event sent
	if cmd != nil {
		cmd()
	}

	// Check output contains key event
	output := out.String()
	if !strings.Contains(output, `"type":"key"`) {
		t.Errorf("expected key event in output, got: %s", output)
	}
	if !strings.Contains(output, `"key":"p"`) {
		t.Errorf("expected key 'p' in event, got: %s", output)
	}
	if !strings.Contains(output, `"view":"agents"`) {
		t.Errorf("expected view 'agents' in event, got: %s", output)
	}
}

func TestDriverHandleKeySendsEventDetailView(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out

	d.handleViewMessage(&ViewMessage{View: ViewDetail, ID: "detail1"})

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	_, cmd := d.handleKey(msg)
	if cmd != nil {
		cmd()
	}

	output := out.String()
	if !strings.Contains(output, `"view":"detail1"`) {
		t.Errorf("expected view 'detail1' in event, got: %s", output)
	}
}

func TestDriverView(t *testing.T) {
	d := NewDriver()

	t.Run("no view", func(t *testing.T) {
		output := d.View()
		if !strings.Contains(output, "Waiting for AI") {
			t.Error("expected 'Waiting for AI' when no view is set")
		}
	})

	t.Run("table view", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "agents", Title: "Agents"})
		d.tableSpec.Columns = []ColumnSpec{{Name: "NAME", Width: 10}}
		d.tableSpec.Rows = []RowSpec{{ID: "1", Values: []string{"test"}}}

		output := d.View()
		if !strings.Contains(output, "bc") {
			t.Error("expected header with title")
		}
		if !strings.Contains(output, "test") {
			t.Error("expected table content")
		}
	})

	t.Run("detail view", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{View: ViewDetail, ID: "d", Title: "Detail"})
		d.detailSpec.Sections = []SectionSpec{
			{Fields: []FieldSpec{{Label: "K", Value: "V"}}},
		}

		output := d.View()
		if !strings.Contains(output, "Detail") {
			t.Error("expected detail title")
		}
	})

	t.Run("modal view", func(t *testing.T) {
		d.handleViewMessage(&ViewMessage{View: ViewModal, ID: "m", Title: "Confirm"})
		d.modalSpec.Type = "confirm"
		d.modalSpec.Message = "Sure?"

		output := d.View()
		if !strings.Contains(output, "Sure?") {
			t.Error("expected modal message")
		}
	})

	t.Run("table with no spec", func(t *testing.T) {
		d.currentView = ViewTable
		d.tableSpec = nil
		output := d.View()
		if !strings.Contains(output, "Loading") {
			t.Error("expected loading when table spec is nil")
		}
	})

	t.Run("detail with no spec", func(t *testing.T) {
		d.currentView = ViewDetail
		d.detailSpec = nil
		output := d.View()
		if !strings.Contains(output, "Loading") {
			t.Error("expected loading when detail spec is nil")
		}
	})
}

func TestDriverViewWithBindings(t *testing.T) {
	d := NewDriver()
	d.handleViewMessage(&ViewMessage{View: ViewTable, ID: "t"})
	d.tableSpec.Bindings = []BindingSpec{
		{Key: "r", Label: "refresh", Action: "refresh"},
	}

	output := d.View()
	if !strings.Contains(output, "r:refresh") {
		t.Error("expected binding hint in status bar")
	}
}

func TestDriverSendReady(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out
	d.width = 80
	d.height = 24

	result := d.sendReady()
	if result != nil {
		t.Error("expected nil from sendReady")
	}

	output := out.String()
	if !strings.Contains(output, `"type":"ready"`) {
		t.Errorf("expected ready event, got: %s", output)
	}
}

func TestDriverReadInput(t *testing.T) {
	d := NewDriver()
	input := bytes.NewBufferString(`{"type":"view","view":"table","id":"t"}` + "\n")
	d.input = input

	msg := d.readInput()
	if im, ok := msg.(incomingMessage); ok {
		if !strings.Contains(string(im.data), "table") {
			t.Error("expected message containing 'table'")
		}
	} else if _, isQuit := msg.(tea.QuitMsg); isQuit {
		// EOF after reading is acceptable
	} else {
		t.Errorf("expected incomingMessage or QuitMsg, got %T", msg)
	}
}

func TestDriverReadInputEOF(t *testing.T) {
	d := NewDriver()
	input := bytes.NewBufferString("") // Empty input = EOF
	d.input = input

	msg := d.readInput()
	// EOF should produce tea.Quit which is a func() tea.Msg, not tea.QuitMsg directly
	fn, ok := msg.(func() tea.Msg)
	if !ok {
		t.Fatalf("expected func() tea.Msg on EOF, got %T", msg)
	}
	// Calling the function should produce QuitMsg
	result := fn()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Errorf("expected QuitMsg from quit function, got %T", result)
	}
}

func TestDriverDoneMessageClearsLoadingBoth(t *testing.T) {
	d := NewDriver()
	var out bytes.Buffer
	d.output = &out

	// Set up both table and detail specs
	d.tableSpec = &TableSpec{Loading: true}
	d.detailSpec = &DetailSpec{Loading: true}

	data, _ := json.Marshal(DoneMessage{Type: MsgDone})
	d.handleIncoming(incomingMessage{data: data})

	if d.tableSpec.Loading {
		t.Error("expected table loading cleared")
	}
	if d.detailSpec.Loading {
		t.Error("expected detail loading cleared")
	}
}
