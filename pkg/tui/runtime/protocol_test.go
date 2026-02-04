package runtime

import (
	"encoding/json"
	"testing"
)

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType MessageType
		wantErr  bool
	}{
		{
			name:     "view message",
			input:    `{"type": "view", "view": "table", "id": "test"}`,
			wantType: MsgView,
		},
		{
			name:     "set message",
			input:    `{"type": "set", "path": "title", "value": "Hello"}`,
			wantType: MsgSet,
		},
		{
			name:     "append message",
			input:    `{"type": "append", "path": "rows", "value": {"id": "1"}}`,
			wantType: MsgAppend,
		},
		{
			name:     "key event",
			input:    `{"type": "key", "key": "enter", "view": "agents"}`,
			wantType: MsgKey,
		},
		{
			name:     "done message",
			input:    `{"type": "done"}`,
			wantType: MsgDone,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMessage([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantType {
				t.Errorf("ParseMessage() = %v, want %v", got, tt.wantType)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	event := KeyEvent{
		Type: MsgKey,
		Key:  "enter",
		View: "agents",
		Selected: &RowRef{
			ID:     "1",
			Index:  0,
			Values: []string{"worker-01", "running"},
		},
	}

	data, err := Marshal(event)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Should end with newline
	if data[len(data)-1] != '\n' {
		t.Error("Marshal() should append newline")
	}

	// Should be valid JSON
	var decoded KeyEvent
	if err := json.Unmarshal(data[:len(data)-1], &decoded); err != nil {
		t.Errorf("Marshal() produced invalid JSON: %v", err)
	}

	if decoded.Key != "enter" {
		t.Errorf("decoded.Key = %v, want 'enter'", decoded.Key)
	}
}

func TestTableSpec(t *testing.T) {
	spec := TableSpec{
		ID:    "agents",
		Title: "Active Agents",
		Columns: []ColumnSpec{
			{Name: "NAME", Width: 15},
			{Name: "STATUS", Width: 10},
		},
		Rows: []RowSpec{
			{ID: "1", Values: []string{"worker-01", "running"}, Status: "ok"},
			{ID: "2", Values: []string{"worker-02", "error"}, Status: "error"},
		},
		Bindings: []BindingSpec{
			{Key: "enter", Label: "Select", Action: "select"},
		},
	}

	// Should serialize to JSON
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Should deserialize back
	var decoded TableSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.ID != "agents" {
		t.Errorf("decoded.ID = %v, want 'agents'", decoded.ID)
	}
	if len(decoded.Columns) != 2 {
		t.Errorf("len(decoded.Columns) = %v, want 2", len(decoded.Columns))
	}
	if len(decoded.Rows) != 2 {
		t.Errorf("len(decoded.Rows) = %v, want 2", len(decoded.Rows))
	}
}

func TestDetailSpec(t *testing.T) {
	spec := DetailSpec{
		ID:    "agent-detail",
		Title: "worker-01",
		Sections: []SectionSpec{
			{
				Title: "Info",
				Fields: []FieldSpec{
					{Label: "Name", Value: "worker-01"},
					{Label: "Status", Value: "running", Style: "success"},
				},
			},
		},
	}

	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded DetailSpec
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if len(decoded.Sections) != 1 {
		t.Errorf("len(decoded.Sections) = %v, want 1", len(decoded.Sections))
	}
	if len(decoded.Sections[0].Fields) != 2 {
		t.Errorf("len(decoded.Sections[0].Fields) = %v, want 2", len(decoded.Sections[0].Fields))
	}
}

func TestViewMessage(t *testing.T) {
	msg := ViewMessage{
		Type:    MsgView,
		View:    ViewTable,
		ID:      "agents",
		Title:   "Active Agents",
		Loading: true,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ViewMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Type != MsgView {
		t.Errorf("decoded.Type = %v, want 'view'", decoded.Type)
	}
	if decoded.View != ViewTable {
		t.Errorf("decoded.View = %v, want 'table'", decoded.View)
	}
	if !decoded.Loading {
		t.Error("decoded.Loading should be true")
	}
}
