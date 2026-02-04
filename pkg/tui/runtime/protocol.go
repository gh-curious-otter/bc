// Package runtime provides a streaming, spec-driven TUI runtime.
//
// The runtime allows an AI to control the TUI dynamically by sending
// JSON specs over stdin. User interactions are sent back as events.
//
// # Protocol
//
// AI → TUI (specs/commands):
//
//	{"type": "view", "view": "table", "id": "agents", "title": "Agents"}
//	{"type": "set", "path": "columns", "value": [{"name": "NAME", "width": 15}]}
//	{"type": "append", "path": "rows", "value": {"id": "1", "values": ["worker"]}}
//	{"type": "done"}
//
// TUI → AI (events):
//
//	{"type": "key", "key": "enter", "view": "agents", "selected": {"id": "1"}}
//	{"type": "ready"}
//
// # Streaming
//
// The runtime renders updates immediately as they arrive, providing
// a responsive feel even when the AI is still generating content.
package runtime

import "encoding/json"

// MessageType identifies the kind of message in the protocol.
type MessageType string

const (
	// AI → TUI message types
	MsgView   MessageType = "view"   // Create/switch to a view
	MsgSet    MessageType = "set"    // Set a value at a path
	MsgAppend MessageType = "append" // Append to an array at a path
	MsgDelete MessageType = "delete" // Delete a path
	MsgDone   MessageType = "done"   // Signal completion of an update batch
	MsgError  MessageType = "error"  // Signal an error

	// TUI → AI message types
	MsgKey    MessageType = "key"    // Key press event
	MsgSelect MessageType = "select" // Row/item selection
	MsgInput  MessageType = "input"  // Text input submitted
	MsgReady  MessageType = "ready"  // TUI is ready for commands
	MsgInit   MessageType = "init"   // Initial handshake
)

// Message is the base structure for all protocol messages.
type Message struct {
	Type MessageType `json:"type"`
}

// --- AI → TUI Messages ---

// ViewMessage creates or switches to a view.
type ViewMessage struct {
	Type    MessageType `json:"type"` // "view"
	View    ViewType    `json:"view"` // table, detail, form, modal
	ID      string      `json:"id"`
	Title   string      `json:"title,omitempty"`
	Loading bool        `json:"loading,omitempty"` // Show loading indicator
}

// SetMessage sets a value at a path.
type SetMessage struct {
	Type  MessageType `json:"type"` // "set"
	Path  string      `json:"path"` // JSON path: "title", "columns", "rows[0].status"
	Value any         `json:"value"`
}

// AppendMessage appends to an array at a path.
type AppendMessage struct {
	Type  MessageType `json:"type"` // "append"
	Path  string      `json:"path"` // Path to array: "rows", "sections[0].fields"
	Value any         `json:"value"`
}

// DeleteMessage removes a path.
type DeleteMessage struct {
	Type MessageType `json:"type"` // "delete"
	Path string      `json:"path"`
}

// DoneMessage signals the end of an update batch.
type DoneMessage struct {
	Type MessageType `json:"type"` // "done"
}

// ErrorMessage signals an error.
type ErrorMessage struct {
	Type    MessageType `json:"type"` // "error"
	Message string      `json:"message"`
	Code    string      `json:"code,omitempty"`
}

// --- TUI → AI Messages ---

// KeyEvent is sent when a key is pressed.
type KeyEvent struct {
	Type     MessageType `json:"type"` // "key"
	Key      string      `json:"key"`  // "enter", "p", "q", "ctrl+c"
	View     string      `json:"view"` // Current view ID
	Selected *RowRef     `json:"selected,omitempty"`
}

// SelectEvent is sent when a row/item is selected.
type SelectEvent struct {
	Type MessageType `json:"type"` // "select"
	View string      `json:"view"`
	Row  RowRef      `json:"row"`
}

// InputEvent is sent when text input is submitted.
type InputEvent struct {
	Type  MessageType `json:"type"` // "input"
	View  string      `json:"view"`
	Field string      `json:"field"`
	Value string      `json:"value"`
}

// ReadyEvent signals the TUI is ready.
type ReadyEvent struct {
	Type    MessageType `json:"type"` // "ready"
	Version string      `json:"version"`
	Width   int         `json:"width"`
	Height  int         `json:"height"`
}

// InitEvent is the initial handshake.
type InitEvent struct {
	Type    MessageType `json:"type"` // "init"
	Version string      `json:"version"`
}

// RowRef identifies a row in a table.
type RowRef struct {
	ID     string `json:"id"`
	Index  int    `json:"index"`
	Values []string `json:"values,omitempty"`
	Data   any    `json:"data,omitempty"`
}

// --- View Specs ---

// ViewType identifies the kind of view.
type ViewType string

const (
	ViewTable  ViewType = "table"
	ViewDetail ViewType = "detail"
	ViewForm   ViewType = "form"
	ViewModal  ViewType = "modal"
	ViewList   ViewType = "list"
)

// TableSpec defines a table view.
type TableSpec struct {
	ID       string       `json:"id"`
	Title    string       `json:"title,omitempty"`
	Columns  []ColumnSpec `json:"columns"`
	Rows     []RowSpec    `json:"rows,omitempty"`
	Loading  bool         `json:"loading,omitempty"`
	Empty    string       `json:"empty,omitempty"` // Empty state message
	Bindings []BindingSpec `json:"bindings,omitempty"`
}

// ColumnSpec defines a table column.
type ColumnSpec struct {
	Name  string `json:"name"`
	Width int    `json:"width,omitempty"` // 0 = auto
	Align string `json:"align,omitempty"` // left, center, right
}

// RowSpec defines a table row.
type RowSpec struct {
	ID     string   `json:"id"`
	Values []string `json:"values"`
	Status string   `json:"status,omitempty"` // ok, error, warning, info
	Data   any      `json:"data,omitempty"`   // Arbitrary attached data
}

// DetailSpec defines a detail view.
type DetailSpec struct {
	ID       string        `json:"id"`
	Title    string        `json:"title"`
	Sections []SectionSpec `json:"sections,omitempty"`
	Loading  bool          `json:"loading,omitempty"`
	Bindings []BindingSpec `json:"bindings,omitempty"`
}

// SectionSpec defines a section in a detail view.
type SectionSpec struct {
	Title  string      `json:"title,omitempty"`
	Fields []FieldSpec `json:"fields,omitempty"`
}

// FieldSpec defines a field in a detail section.
type FieldSpec struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Style string `json:"style,omitempty"` // normal, code, success, error, warning, muted
}

// ModalSpec defines a modal dialog.
type ModalSpec struct {
	ID      string       `json:"id"`
	Type    string       `json:"type"` // confirm, input, select
	Title   string       `json:"title"`
	Message string       `json:"message,omitempty"`
	Inputs  []InputSpec  `json:"inputs,omitempty"`  // For input/form modals
	Options []OptionSpec `json:"options,omitempty"` // For select modals
}

// InputSpec defines an input field.
type InputSpec struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// OptionSpec defines a selectable option.
type OptionSpec struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Selected bool   `json:"selected,omitempty"`
}

// BindingSpec defines a key binding.
type BindingSpec struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Action string `json:"action"` // Action name sent back to AI
	Hidden bool   `json:"hidden,omitempty"`
}

// --- Helpers ---

// ParseMessage parses a JSON message and returns its type.
func ParseMessage(data []byte) (MessageType, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return "", err
	}
	return msg.Type, nil
}

// Marshal serializes a message to JSON with a newline.
func Marshal(v any) ([]byte, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}
