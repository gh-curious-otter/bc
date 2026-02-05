package runtime

import "testing"

func TestSetPath(t *testing.T) {
	t.Run("simple path", func(t *testing.T) {
		spec := &TableSpec{Title: "old"}
		err := SetPath(spec, "title", "new")
		if err != nil {
			t.Fatalf("SetPath error: %v", err)
		}
		if spec.Title != "new" {
			t.Errorf("expected title 'new', got '%s'", spec.Title)
		}
	})

	t.Run("nested path", func(t *testing.T) {
		type nested struct {
			Outer struct {
				Inner string `json:"inner"`
			} `json:"outer"`
		}
		spec := &nested{}
		err := SetPath(spec, "outer.inner", "deep")
		if err != nil {
			t.Fatalf("SetPath error: %v", err)
		}
		if spec.Outer.Inner != "deep" {
			t.Errorf("expected 'deep', got '%s'", spec.Outer.Inner)
		}
	})

	t.Run("creates intermediate map", func(t *testing.T) {
		spec := &TableSpec{}
		// Setting a nested path where intermediate doesn't exist
		err := SetPath(spec, "id", "test-id")
		if err != nil {
			t.Fatalf("SetPath error: %v", err)
		}
		if spec.ID != "test-id" {
			t.Errorf("expected ID 'test-id', got '%s'", spec.ID)
		}
	})
}

func TestAppendPath(t *testing.T) {
	t.Run("append to existing array", func(t *testing.T) {
		spec := &TableSpec{
			Rows: []RowSpec{{ID: "1", Values: []string{"first"}}},
		}
		err := AppendPath(spec, "rows", map[string]any{"id": "2", "values": []string{"second"}})
		if err != nil {
			t.Fatalf("AppendPath error: %v", err)
		}
		if len(spec.Rows) != 2 {
			t.Errorf("expected 2 rows, got %d", len(spec.Rows))
		}
	})

	t.Run("append to new array", func(t *testing.T) {
		spec := &TableSpec{}
		err := AppendPath(spec, "rows", map[string]any{"id": "1", "values": []string{"x"}})
		if err != nil {
			t.Fatalf("AppendPath error: %v", err)
		}
		if len(spec.Rows) != 1 {
			t.Errorf("expected 1 row, got %d", len(spec.Rows))
		}
	})
}
