//nolint:errcheck // UI output errors to stdout are not recoverable
package ui

import (
	"fmt"
	"strings"
)

// Table represents a formatted table for CLI output.
type Table struct {
	headers []string
	rows    [][]string
	widths  []int
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		widths:  widths,
	}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(values ...string) {
	// Ensure row has correct number of columns
	row := make([]string, len(t.headers))
	for i := 0; i < len(t.headers) && i < len(values); i++ {
		row[i] = values[i]
		if len(values[i]) > t.widths[i] {
			t.widths[i] = len(values[i])
		}
	}
	t.rows = append(t.rows, row)
}

// String renders the table as a string.
func (t *Table) String() string {
	var sb strings.Builder

	// Render header
	t.renderRow(&sb, t.headers, true)

	// Render separator
	t.renderSeparator(&sb)

	// Render rows
	for _, row := range t.rows {
		t.renderRow(&sb, row, false)
	}

	return sb.String()
}

// Print renders and prints the table.
func (t *Table) Print() {
	fmt.Fprint(output, t.String())
}

// renderRow renders a single row.
func (t *Table) renderRow(sb *strings.Builder, values []string, isHeader bool) {
	for i, val := range values {
		if i > 0 {
			sb.WriteString("  ")
		}
		padded := padRight(val, t.widths[i])
		if isHeader && colorEnabled {
			sb.WriteString(Bold)
			sb.WriteString(padded)
			sb.WriteString(Reset)
		} else {
			sb.WriteString(padded)
		}
	}
	sb.WriteString("\n")
}

// renderSeparator renders a separator line.
func (t *Table) renderSeparator(sb *strings.Builder) {
	for i, w := range t.widths {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(strings.Repeat("─", w))
	}
	sb.WriteString("\n")
}

// padRight pads a string to the right with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// SimpleTable prints a simple key-value table.
func SimpleTable(pairs ...string) {
	if len(pairs)%2 != 0 {
		return
	}

	// Find max key width
	maxWidth := 0
	for i := 0; i < len(pairs); i += 2 {
		if len(pairs[i]) > maxWidth {
			maxWidth = len(pairs[i])
		}
	}

	// Print rows
	for i := 0; i < len(pairs); i += 2 {
		key := pairs[i]
		val := pairs[i+1]
		if colorEnabled {
			fmt.Fprintf(output, "%s%s%s  %s\n", Dim, padRight(key+":", maxWidth+1), Reset, val)
		} else {
			fmt.Fprintf(output, "%s  %s\n", padRight(key+":", maxWidth+1), val)
		}
	}
}
