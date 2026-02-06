package runtime

import (
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/rpuneet/bc/pkg/tui/style"
)

// Renderer interprets specs and produces rendered output.
type Renderer struct {
	styles style.Styles
	width  int
	height int
}

// NewRenderer creates a new spec renderer.
func NewRenderer() *Renderer {
	return &Renderer{
		styles: style.DefaultStyles(),
		width:  80,
		height: 24,
	}
}

// SetSize updates the terminal dimensions.
func (r *Renderer) SetSize(width, height int) {
	r.width = width
	r.height = height
}

// RenderTable renders a table spec to a string.
func (r *Renderer) RenderTable(spec *TableSpec, cursor int) string {
	var b strings.Builder

	// Title
	if spec.Title != "" {
		title := r.styles.Title.Render(spec.Title)
		if spec.Loading {
			title += r.styles.Muted.Render(" (loading...)")
		}
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	// Empty state
	if len(spec.Rows) == 0 && !spec.Loading {
		empty := spec.Empty
		if empty == "" {
			empty = "No data"
		}
		b.WriteString(r.styles.Muted.Render("  " + empty))
		b.WriteString("\n")
		return b.String()
	}

	// Header
	b.WriteString(r.renderTableHeader(spec.Columns))
	b.WriteString("\n")

	// Rows
	for i, row := range spec.Rows {
		selected := i == cursor
		b.WriteString(r.renderTableRow(spec.Columns, &row, selected))
		b.WriteString("\n")
	}

	// Loading indicator for streaming
	if spec.Loading && len(spec.Rows) > 0 {
		b.WriteString(r.styles.Muted.Render("  Loading more..."))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *Renderer) renderTableHeader(columns []ColumnSpec) string {
	cells := make([]string, 0, len(columns))
	for _, col := range columns {
		width := col.Width
		if width == 0 {
			width = 15
		}
		cell := r.styles.Bold.Width(width).Render(col.Name)
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}

func (r *Renderer) renderTableRow(columns []ColumnSpec, row *RowSpec, selected bool) string {
	cells := make([]string, 0, len(columns))
	for i, col := range columns {
		width := col.Width
		if width == 0 {
			width = 15
		}

		value := ""
		if i < len(row.Values) {
			value = row.Values[i]
		}

		cellStyle := r.styles.Normal.Width(width)

		// Alignment
		switch col.Align {
		case "right":
			cellStyle = cellStyle.Align(lipgloss.Right)
		case "center":
			cellStyle = cellStyle.Align(lipgloss.Center)
		}

		cells = append(cells, cellStyle.Render(value))
	}

	rowStr := lipgloss.JoinHorizontal(lipgloss.Top, cells...)

	if selected {
		return r.styles.Selected.Render(rowStr)
	}

	// Status-based styling
	if row.Status != "" {
		return r.styles.StatusStyle(row.Status).Render(rowStr)
	}

	return rowStr
}

// RenderDetail renders a detail spec to a string.
func (r *Renderer) RenderDetail(spec *DetailSpec) string {
	var b strings.Builder

	// Title
	if spec.Title != "" {
		title := r.styles.Title.Render(spec.Title)
		if spec.Loading {
			title += r.styles.Muted.Render(" (loading...)")
		}
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	// Sections
	for i, section := range spec.Sections {
		if section.Title != "" {
			b.WriteString(r.styles.Bold.Render(section.Title))
			b.WriteString("\n")
		}

		for _, field := range section.Fields {
			b.WriteString(r.renderField(&field))
			b.WriteString("\n")
		}

		if i < len(spec.Sections)-1 {
			b.WriteString("\n")
		}
	}

	// Loading placeholder
	if spec.Loading && len(spec.Sections) == 0 {
		b.WriteString(r.styles.Muted.Render("  Loading..."))
		b.WriteString("\n")
	}

	return b.String()
}

func (r *Renderer) renderField(field *FieldSpec) string {
	label := r.styles.Muted.Width(15).Render(field.Label + ":")

	valueStyle := r.styles.Normal
	switch field.Style {
	case "code":
		valueStyle = r.styles.Code
	case "success", "ok":
		valueStyle = r.styles.Success
	case "error":
		valueStyle = r.styles.Error
	case "warning":
		valueStyle = r.styles.Warning
	case "muted":
		valueStyle = r.styles.Muted
	}

	value := valueStyle.Render(field.Value)
	return label + " " + value
}

// RenderModal renders a modal spec to a string.
func (r *Renderer) RenderModal(spec *ModalSpec) string {
	var content strings.Builder

	// Title
	content.WriteString(r.styles.Bold.Render(spec.Title))
	content.WriteString("\n\n")

	// Message
	if spec.Message != "" {
		content.WriteString(spec.Message)
		content.WriteString("\n\n")
	}

	// Type-specific content
	switch spec.Type {
	case "confirm":
		content.WriteString(r.styles.Muted.Render("[y] Yes  [n] No"))
	case "input":
		for _, input := range spec.Inputs {
			content.WriteString(r.styles.Muted.Render(input.Label + ": "))
			if input.Value != "" {
				content.WriteString(input.Value)
			} else if input.Placeholder != "" {
				content.WriteString(r.styles.Muted.Render(input.Placeholder))
			}
			content.WriteString("\n")
		}
	case "select":
		for _, opt := range spec.Options {
			prefix := "  "
			if opt.Selected {
				prefix = "> "
			}
			content.WriteString(prefix + opt.Label + "\n")
		}
	}

	// Wrap in border
	modalStyle := r.styles.Border.
		Width(50).
		Padding(1, 2)

	return modalStyle.Render(content.String())
}

// RenderBindings renders key binding hints for the status bar.
func (r *Renderer) RenderBindings(bindings []BindingSpec) string {
	var hints []string
	for _, b := range bindings {
		if !b.Hidden {
			hints = append(hints, b.Key+":"+b.Label)
		}
	}
	hints = append(hints, "q:quit")
	return r.styles.StatusBar.Width(r.width).Render(strings.Join(hints, " | "))
}

// RenderHeader renders the app header.
func (r *Renderer) RenderHeader(title, viewID string) string {
	left := r.styles.Header.Render(title)
	if viewID != "" {
		left += r.styles.Muted.Render(" [" + viewID + "]")
	}
	return left
}

// RenderLoading renders a loading placeholder.
func (r *Renderer) RenderLoading(message string) string {
	if message == "" {
		message = "Loading..."
	}
	return r.styles.Muted.Render(message)
}

// --- JSON Path Helpers ---

// SetPath sets a value at a JSON path in a spec.
// Supports paths like: "title", "columns", "rows[0].status", "sections[1].fields"
func SetPath(spec any, path string, value any) error {
	// For now, use JSON marshal/unmarshal for simplicity
	// A production version would use reflection or a proper JSON path library
	data, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Simple path setting (no array indexing yet)
	parts := strings.Split(path, ".")
	current := m
	for i, part := range parts {
		if i == len(parts)-1 {
			current[part] = value
		} else {
			if next, ok := current[part].(map[string]any); ok {
				current = next
			} else {
				// Create intermediate map
				next := make(map[string]any)
				current[part] = next
				current = next
			}
		}
	}

	// Marshal back and unmarshal into spec
	data, err = json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, spec)
}

// AppendPath appends a value to an array at a JSON path.
func AppendPath(spec any, path string, value any) error {
	data, err := json.Marshal(spec)
	if err != nil {
		return err
	}

	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Navigate to parent and append
	parts := strings.Split(path, ".")
	current := m
	for i, part := range parts {
		if i == len(parts)-1 {
			// Append to array
			if arr, ok := current[part].([]any); ok {
				current[part] = append(arr, value)
			} else {
				// Create new array
				current[part] = []any{value}
			}
		} else {
			if next, ok := current[part].(map[string]any); ok {
				current = next
			}
		}
	}

	data, err = json.Marshal(m)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, spec)
}
