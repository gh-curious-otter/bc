package tui

import tea "github.com/charmbracelet/bubbletea"

// Cmd is an alias for Bubble Tea's Cmd type.
type Cmd = tea.Cmd

// Msg is an alias for Bubble Tea's Msg type.
type Msg = tea.Msg

// KeyMsg is an alias for Bubble Tea's KeyMsg type.
type KeyMsg = tea.KeyMsg

// Model is the interface that all TUI components must implement.
// It's a simplified version of Bubble Tea's Model.
type Model interface {
	// Init returns the initial command to run.
	Init() Cmd

	// Update handles a message and returns the updated model and command.
	Update(msg Msg) (Model, Cmd)

	// View renders the component to a string.
	View() string
}

// Row represents a single row of data in a table.
type Row struct {
	ID     string   // Unique identifier for the row
	Values []string // Cell values
	Data   any      // Optional attached data
	Status string   // Optional status for styling (ok, error, warning, etc.)
}

// Column defines a table column.
type Column struct {
	Name      string // Column header name
	Width     int    // Column width in characters (0 = auto)
	Alignment Alignment
}

// Alignment specifies text alignment within a column.
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// Col is a convenience function to create a Column.
func Col(name string, width int) Column {
	return Column{Name: name, Width: width, Alignment: AlignLeft}
}

// ColRight creates a right-aligned column.
func ColRight(name string, width int) Column {
	return Column{Name: name, Width: width, Alignment: AlignRight}
}

// ColCenter creates a center-aligned column.
func ColCenter(name string, width int) Column {
	return Column{Name: name, Width: width, Alignment: AlignCenter}
}

// KeyBinding represents a keyboard shortcut and its action.
type KeyBinding struct {
	Key         string     // Key to bind (e.g., "enter", "p", "ctrl+c")
	Label       string     // Human-readable label for help text
	Description string     // Longer description
	Handler     func() Cmd // Action to perform
	Hidden      bool       // If true, don't show in help
}

// Bind creates a KeyBinding with a key, label, and handler.
func Bind(key, label string, handler func() Cmd) KeyBinding {
	return KeyBinding{
		Key:     key,
		Label:   label,
		Handler: handler,
	}
}

// Section represents a grouping in a detail view.
type Section struct {
	Title  string
	Fields []Field
}

// Field represents a key-value pair in a detail view.
type Field struct {
	Label string
	Value string
	Style FieldStyle
}

// FieldStyle determines how a field value is rendered.
type FieldStyle int

const (
	FieldNormal FieldStyle = iota
	FieldCode              // Monospace
	FieldStatus            // Colored based on value
	FieldLink              // Clickable/highlighted
	FieldMuted             // Dimmed
)
