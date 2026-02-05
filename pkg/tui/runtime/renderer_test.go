package runtime

import (
	"strings"
	"testing"
)

func TestRendererRenderTable(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Title: "Agents",
		Columns: []ColumnSpec{
			{Name: "NAME", Width: 15},
			{Name: "STATUS", Width: 10},
		},
		Rows: []RowSpec{
			{ID: "1", Values: []string{"worker-01", "running"}, Status: "ok"},
			{ID: "2", Values: []string{"worker-02", "idle"}},
		},
	}

	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "Agents") {
		t.Error("expected output to contain title 'Agents'")
	}
	if !strings.Contains(output, "NAME") {
		t.Error("expected output to contain header 'NAME'")
	}
	if !strings.Contains(output, "worker-01") {
		t.Error("expected output to contain 'worker-01'")
	}
	if !strings.Contains(output, "worker-02") {
		t.Error("expected output to contain 'worker-02'")
	}
}

func TestRendererRenderTableEmpty(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Title:   "Empty",
		Columns: []ColumnSpec{{Name: "A", Width: 10}},
	}

	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "No data") {
		t.Error("expected 'No data' for empty table")
	}
}

func TestRendererRenderTableCustomEmpty(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Title:   "Custom",
		Columns: []ColumnSpec{{Name: "A", Width: 10}},
		Empty:   "Nothing here",
	}

	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "Nothing here") {
		t.Error("expected custom empty message")
	}
}

func TestRendererRenderTableLoading(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Title:   "Loading",
		Loading: true,
		Columns: []ColumnSpec{{Name: "A", Width: 10}},
		Rows:    []RowSpec{{ID: "1", Values: []string{"partial"}}},
	}

	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "loading") {
		t.Error("expected loading indicator in title")
	}
	if !strings.Contains(output, "Loading more") {
		t.Error("expected 'Loading more...' indicator")
	}
}

func TestRendererRenderTableAlignment(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Columns: []ColumnSpec{
			{Name: "LEFT", Width: 10},
			{Name: "RIGHT", Width: 10, Align: "right"},
			{Name: "CENTER", Width: 10, Align: "center"},
		},
		Rows: []RowSpec{
			{ID: "1", Values: []string{"L", "R", "C"}},
		},
	}

	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "L") {
		t.Error("expected output to contain 'L'")
	}
}

func TestRendererRenderTableAutoWidth(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Columns: []ColumnSpec{
			{Name: "AUTO", Width: 0}, // Width 0 should default to 15
		},
		Rows: []RowSpec{
			{ID: "1", Values: []string{"test"}},
		},
	}

	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "test") {
		t.Error("expected output with auto-width column")
	}
}

func TestRendererRenderTableRowMissingValues(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Columns: []ColumnSpec{
			{Name: "A", Width: 15},
			{Name: "B", Width: 15},
		},
		Rows: []RowSpec{
			{ID: "1", Values: []string{"only-one"}}, // Fewer values than columns
		},
	}

	// Should not panic and render the available value
	output := r.RenderTable(spec, 0)
	if !strings.Contains(output, "only-one") {
		t.Error("expected output to contain value")
	}
}

func TestRendererRenderDetail(t *testing.T) {
	r := NewRenderer()
	spec := &DetailSpec{
		Title: "Agent Detail",
		Sections: []SectionSpec{
			{
				Title: "Info",
				Fields: []FieldSpec{
					{Label: "Name", Value: "worker-01"},
					{Label: "Status", Value: "running", Style: "success"},
					{Label: "Error", Value: "none", Style: "error"},
					{Label: "Hint", Value: "note", Style: "warning"},
					{Label: "Path", Value: "/usr/bin", Style: "code"},
					{Label: "Old", Value: "legacy", Style: "muted"},
				},
			},
			{
				Title: "Resources",
				Fields: []FieldSpec{
					{Label: "CPU", Value: "12%"},
				},
			},
		},
	}

	output := r.RenderDetail(spec)
	if !strings.Contains(output, "Agent Detail") {
		t.Error("expected title 'Agent Detail'")
	}
	if !strings.Contains(output, "Info") {
		t.Error("expected section title 'Info'")
	}
	if !strings.Contains(output, "worker-01") {
		t.Error("expected field value 'worker-01'")
	}
	if !strings.Contains(output, "running") {
		t.Error("expected field value 'running'")
	}
	if !strings.Contains(output, "Resources") {
		t.Error("expected section title 'Resources'")
	}
}

func TestRendererRenderDetailLoading(t *testing.T) {
	r := NewRenderer()
	spec := &DetailSpec{
		Title:   "Loading Detail",
		Loading: true,
	}

	output := r.RenderDetail(spec)
	if !strings.Contains(output, "loading") {
		t.Error("expected loading indicator in title")
	}
	if !strings.Contains(output, "Loading...") {
		t.Error("expected loading placeholder")
	}
}

func TestRendererRenderDetailNoTitle(t *testing.T) {
	r := NewRenderer()
	spec := &DetailSpec{
		Sections: []SectionSpec{
			{Fields: []FieldSpec{{Label: "K", Value: "V"}}},
		},
	}

	output := r.RenderDetail(spec)
	if !strings.Contains(output, "V") {
		t.Error("expected field value")
	}
}

func TestRendererRenderModal(t *testing.T) {
	r := NewRenderer()

	t.Run("confirm", func(t *testing.T) {
		spec := &ModalSpec{
			Type:    "confirm",
			Title:   "Delete?",
			Message: "Are you sure?",
		}
		output := r.RenderModal(spec)
		if !strings.Contains(output, "Delete?") {
			t.Error("expected modal title")
		}
		if !strings.Contains(output, "Are you sure?") {
			t.Error("expected modal message")
		}
		if !strings.Contains(output, "Yes") {
			t.Error("expected confirm options")
		}
	})

	t.Run("input", func(t *testing.T) {
		spec := &ModalSpec{
			Type:  "input",
			Title: "Enter Name",
			Inputs: []InputSpec{
				{ID: "name", Label: "Name", Placeholder: "type here"},
			},
		}
		output := r.RenderModal(spec)
		if !strings.Contains(output, "Name") {
			t.Error("expected input label")
		}
		if !strings.Contains(output, "type here") {
			t.Error("expected input placeholder")
		}
	})

	t.Run("input with value", func(t *testing.T) {
		spec := &ModalSpec{
			Type:  "input",
			Title: "Enter Name",
			Inputs: []InputSpec{
				{ID: "name", Label: "Name", Value: "Alice"},
			},
		}
		output := r.RenderModal(spec)
		if !strings.Contains(output, "Alice") {
			t.Error("expected input value")
		}
	})

	t.Run("select", func(t *testing.T) {
		spec := &ModalSpec{
			Type:  "select",
			Title: "Choose",
			Options: []OptionSpec{
				{ID: "a", Label: "Option A"},
				{ID: "b", Label: "Option B", Selected: true},
			},
		}
		output := r.RenderModal(spec)
		if !strings.Contains(output, "Option A") {
			t.Error("expected option A")
		}
		if !strings.Contains(output, "> Option B") {
			t.Error("expected selected option B with '>' prefix")
		}
	})

	t.Run("no message", func(t *testing.T) {
		spec := &ModalSpec{
			Type:  "confirm",
			Title: "Quick",
		}
		output := r.RenderModal(spec)
		if !strings.Contains(output, "Quick") {
			t.Error("expected title")
		}
	})
}

func TestRendererRenderBindings(t *testing.T) {
	r := NewRenderer()

	bindings := []BindingSpec{
		{Key: "enter", Label: "select", Action: "select"},
		{Key: "r", Label: "refresh", Action: "refresh"},
		{Key: "h", Label: "hidden", Action: "hidden", Hidden: true},
	}

	output := r.RenderBindings(bindings)
	if !strings.Contains(output, "enter:select") {
		t.Error("expected 'enter:select' in bindings")
	}
	if !strings.Contains(output, "r:refresh") {
		t.Error("expected 'r:refresh' in bindings")
	}
	if strings.Contains(output, "h:hidden") {
		t.Error("expected hidden binding to be excluded")
	}
	if !strings.Contains(output, "q:quit") {
		t.Error("expected 'q:quit' always present")
	}
}

func TestRendererRenderBindingsEmpty(t *testing.T) {
	r := NewRenderer()
	output := r.RenderBindings(nil)
	if !strings.Contains(output, "q:quit") {
		t.Error("expected 'q:quit' even with no bindings")
	}
}

func TestRendererRenderHeader(t *testing.T) {
	r := NewRenderer()

	output := r.RenderHeader("bc", "agents")
	if !strings.Contains(output, "bc") {
		t.Error("expected title 'bc'")
	}
	if !strings.Contains(output, "[agents]") {
		t.Error("expected view ID '[agents]'")
	}
}

func TestRendererRenderHeaderNoView(t *testing.T) {
	r := NewRenderer()
	output := r.RenderHeader("myapp", "")
	if !strings.Contains(output, "myapp") {
		t.Error("expected title 'myapp'")
	}
	if strings.Contains(output, "[") {
		t.Error("expected no brackets when viewID is empty")
	}
}

func TestRendererRenderLoading(t *testing.T) {
	r := NewRenderer()

	output := r.RenderLoading("")
	if !strings.Contains(output, "Loading...") {
		t.Error("expected default loading message")
	}

	output = r.RenderLoading("Please wait")
	if !strings.Contains(output, "Please wait") {
		t.Error("expected custom loading message")
	}
}

func TestRendererSetSize(t *testing.T) {
	r := NewRenderer()
	r.SetSize(120, 40)

	if r.width != 120 {
		t.Errorf("expected width 120, got %d", r.width)
	}
	if r.height != 40 {
		t.Errorf("expected height 40, got %d", r.height)
	}
}

func TestRendererRenderTableSelected(t *testing.T) {
	r := NewRenderer()
	spec := &TableSpec{
		Columns: []ColumnSpec{{Name: "NAME", Width: 10}},
		Rows: []RowSpec{
			{ID: "1", Values: []string{"first"}},
			{ID: "2", Values: []string{"second"}},
		},
	}

	// Cursor on row 1 (second row)
	output := r.RenderTable(spec, 1)
	if !strings.Contains(output, "first") {
		t.Error("expected first row")
	}
	if !strings.Contains(output, "second") {
		t.Error("expected second row")
	}
}
