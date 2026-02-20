//nolint:errcheck // UI output errors to stdout are not recoverable
package ui

import (
	"fmt"
)

// ListItem represents an item in a list.
type ListItem struct {
	Text   string
	Detail string
}

// List prints a bulleted list.
func List(items ...string) {
	for _, item := range items {
		if colorEnabled {
			fmt.Fprintf(output, "  %s•%s %s\n", Dim, Reset, item)
		} else {
			fmt.Fprintf(output, "  • %s\n", item)
		}
	}
}

// NumberedList prints a numbered list.
func NumberedList(items ...string) {
	for i, item := range items {
		if colorEnabled {
			fmt.Fprintf(output, "  %s%d.%s %s\n", Dim, i+1, Reset, item)
		} else {
			fmt.Fprintf(output, "  %d. %s\n", i+1, item)
		}
	}
}

// CheckList prints a list with checkmarks.
func CheckList(items ...ListItem) {
	for _, item := range items {
		if item.Detail != "" {
			if colorEnabled {
				fmt.Fprintf(output, "  %s✓%s %s %s(%s)%s\n",
					Green, Reset, item.Text, Dim, item.Detail, Reset)
			} else {
				fmt.Fprintf(output, "  ✓ %s (%s)\n", item.Text, item.Detail)
			}
		} else {
			if colorEnabled {
				fmt.Fprintf(output, "  %s✓%s %s\n", Green, Reset, item.Text)
			} else {
				fmt.Fprintf(output, "  ✓ %s\n", item.Text)
			}
		}
	}
}

// IndentedList prints a list with custom indent.
func IndentedList(indent int, items ...string) {
	prefix := ""
	for range indent {
		prefix += "  "
	}
	for _, item := range items {
		if colorEnabled {
			fmt.Fprintf(output, "%s%s•%s %s\n", prefix, Dim, Reset, item)
		} else {
			fmt.Fprintf(output, "%s• %s\n", prefix, item)
		}
	}
}

// KeyValueList prints a list of key-value pairs.
func KeyValueList(pairs map[string]string) {
	// Find max key length
	maxLen := 0
	for k := range pairs {
		if len(k) > maxLen {
			maxLen = len(k)
		}
	}

	for k, v := range pairs {
		if colorEnabled {
			fmt.Fprintf(output, "  %s%s:%s %s\n",
				Bold, padRight(k, maxLen), Reset, v)
		} else {
			fmt.Fprintf(output, "  %s: %s\n", padRight(k, maxLen), v)
		}
	}
}
