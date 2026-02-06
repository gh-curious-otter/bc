package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/rpuneet/bc/pkg/tui/style"
)

// TableView is a navigable table component with vim-style keybindings.
type TableView struct {
	id       string
	title    string
	columns  []Column
	rows     []Row
	bindings []KeyBinding

	// State
	cursor  int
	offset  int
	width   int
	height  int
	focused bool

	// Callbacks
	onSelect func(Row) Cmd
	onRender func(Row, int, bool) string // Custom row renderer

	// Styling
	styles style.Styles
}

// TableBuilder provides a fluent API for constructing TableView.
type TableBuilder struct {
	view *TableView
}

// NewTableView creates a new TableBuilder with the given ID.
func NewTableView(id string) *TableBuilder {
	return &TableBuilder{
		view: &TableView{
			id:      id,
			styles:  style.DefaultStyles(),
			focused: true,
		},
	}
}

// Title sets the table title.
func (b *TableBuilder) Title(title string) *TableBuilder {
	b.view.title = title
	return b
}

// Columns sets the table columns.
func (b *TableBuilder) Columns(cols ...Column) *TableBuilder {
	b.view.columns = cols
	return b
}

// Rows sets the initial row data.
func (b *TableBuilder) Rows(rows ...Row) *TableBuilder {
	b.view.rows = rows
	return b
}

// OnSelect sets the callback when a row is selected (Enter key).
func (b *TableBuilder) OnSelect(fn func(Row) Cmd) *TableBuilder {
	b.view.onSelect = fn
	return b
}

// OnRender sets a custom row renderer.
// The function receives the row, index, and whether it's selected.
func (b *TableBuilder) OnRender(fn func(Row, int, bool) string) *TableBuilder {
	b.view.onRender = fn
	return b
}

// Bind adds a key binding.
func (b *TableBuilder) Bind(key, label string, handler func() Cmd) *TableBuilder {
	b.view.bindings = append(b.view.bindings, KeyBinding{
		Key:     key,
		Label:   label,
		Handler: handler,
	})
	return b
}

// Bindings adds multiple key bindings.
func (b *TableBuilder) Bindings(bindings ...KeyBinding) *TableBuilder {
	b.view.bindings = append(b.view.bindings, bindings...)
	return b
}

// Styles sets custom styles.
func (b *TableBuilder) Styles(s style.Styles) *TableBuilder {
	b.view.styles = s
	return b
}

// Build returns the constructed TableView.
func (b *TableBuilder) Build() *TableView {
	return b.view
}

// --- Model Implementation ---

// Init implements Model.
func (t *TableView) Init() Cmd {
	return nil
}

// Update implements Model.
func (t *TableView) Update(msg Msg) (Model, Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case tea.KeyMsg:
		return t.handleKey(msg)
	}
	return t, nil
}

func (t *TableView) handleKey(msg tea.KeyMsg) (Model, Cmd) {
	// Check custom bindings first
	for _, binding := range t.bindings {
		if key.Matches(msg, key.NewBinding(key.WithKeys(binding.Key))) {
			if binding.Handler != nil {
				return t, binding.Handler()
			}
		}
	}

	// Built-in navigation
	keyStr := msg.String()
	switch keyStr {
	case "j", "down":
		t.MoveDown()
	case "k", "up":
		t.MoveUp()
	case "g", "home":
		t.MoveToTop()
	case "G", "end":
		t.MoveToBottom()
	}
	if isEnterKey(msg) && t.onSelect != nil && t.cursor < len(t.rows) {
		return t, t.onSelect(t.rows[t.cursor])
	}

	return t, nil
}

// View implements Model.
func (t *TableView) View() string {
	var b strings.Builder

	// Title
	if t.title != "" {
		b.WriteString(t.styles.Title.Render(t.title))
		b.WriteString("\n")
	}

	// Header
	b.WriteString(t.renderHeader())
	b.WriteString("\n")

	// Rows
	visibleRows := t.visibleRowCount()
	for i := t.offset; i < t.offset+visibleRows && i < len(t.rows); i++ {
		selected := i == t.cursor
		b.WriteString(t.renderRow(t.rows[i], i, selected))
		b.WriteString("\n")
	}

	// Empty state
	if len(t.rows) == 0 {
		b.WriteString(t.styles.Muted.Render("  No data"))
		b.WriteString("\n")
	}

	return b.String()
}

func (t *TableView) renderHeader() string {
	cells := make([]string, 0, len(t.columns))
	for _, col := range t.columns {
		cell := t.styles.Bold.Width(col.Width).Render(col.Name)
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (t *TableView) renderRow(row Row, index int, selected bool) string {
	// Use custom renderer if provided
	if t.onRender != nil {
		return t.onRender(row, index, selected)
	}

	// Default rendering
	var cells []string
	for i, col := range t.columns {
		value := ""
		if i < len(row.Values) {
			value = row.Values[i]
		}

		cellStyle := t.styles.Normal.Width(col.Width)

		// Apply alignment
		switch col.Alignment {
		case AlignRight:
			cellStyle = cellStyle.Align(lipgloss.Right)
		case AlignCenter:
			cellStyle = cellStyle.Align(lipgloss.Center)
		}

		cells = append(cells, cellStyle.Render(value))
	}

	rowStr := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

	if selected {
		return t.styles.Selected.Render(rowStr)
	}

	// Apply status-based styling
	if row.Status != "" {
		return t.styles.StatusStyle(row.Status).Render(rowStr)
	}

	return rowStr
}

func (t *TableView) visibleRowCount() int {
	// Reserve space for title and header
	reserved := 2
	if t.title != "" {
		reserved++
	}
	if t.height > reserved {
		return t.height - reserved
	}
	return 10 // Default
}

// --- Navigation Methods ---

// MoveDown moves the cursor down one row.
func (t *TableView) MoveDown() {
	if t.cursor < len(t.rows)-1 {
		t.cursor++
		t.scrollToView()
	}
}

// MoveUp moves the cursor up one row.
func (t *TableView) MoveUp() {
	if t.cursor > 0 {
		t.cursor--
		t.scrollToView()
	}
}

// MoveToTop moves the cursor to the first row.
func (t *TableView) MoveToTop() {
	t.cursor = 0
	t.offset = 0
}

// MoveToBottom moves the cursor to the last row.
func (t *TableView) MoveToBottom() {
	if len(t.rows) > 0 {
		t.cursor = len(t.rows) - 1
		t.scrollToView()
	}
}

func (t *TableView) scrollToView() {
	visible := t.visibleRowCount()
	if t.cursor < t.offset {
		t.offset = t.cursor
	} else if t.cursor >= t.offset+visible {
		t.offset = t.cursor - visible + 1
	}
}

// --- Data Methods ---

// SetRows replaces all rows.
func (t *TableView) SetRows(rows []Row) {
	t.rows = rows
	if t.cursor >= len(rows) {
		t.cursor = max(0, len(rows)-1)
	}
}

// SelectedRow returns the currently selected row, or nil if empty.
func (t *TableView) SelectedRow() *Row {
	if t.cursor < len(t.rows) {
		return &t.rows[t.cursor]
	}
	return nil
}

// ID returns the table's identifier.
func (t *TableView) ID() string {
	return t.id
}

// RowCount returns the number of rows.
func (t *TableView) RowCount() int {
	return len(t.rows)
}

// Focus sets the focused state.
func (t *TableView) Focus() {
	t.focused = true
}

// Blur removes focus.
func (t *TableView) Blur() {
	t.focused = false
}
