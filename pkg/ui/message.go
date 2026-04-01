//nolint:errcheck // UI output errors to stdout are not recoverable
package ui

import (
	"fmt"
	"io"
	"os"
)

// Default output writer (can be overridden for testing).
var output io.Writer = os.Stdout

// SetOutput overrides the output writer (for testing). Pass nil to reset to stdout.
func SetOutput(w io.Writer) {
	if w == nil {
		output = os.Stdout
	} else {
		output = w
	}
}

// Message prefixes with colors.
const (
	successPrefix = "✓"
	errorPrefix   = "✗"
	warningPrefix = "!"
	infoPrefix    = "→"
	debugPrefix   = "·"
)

// Warning prints a warning message with yellow exclamation.
func Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if colorEnabled {
		fmt.Fprintf(output, "%s%s%s %s\n", Yellow, warningPrefix, Reset, msg)
	} else {
		fmt.Fprintf(output, "%s %s\n", warningPrefix, msg)
	}
}

// Info prints an info message with blue arrow.
func Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if colorEnabled {
		fmt.Fprintf(output, "%s%s%s %s\n", Blue, infoPrefix, Reset, msg)
	} else {
		fmt.Fprintf(output, "%s %s\n", infoPrefix, msg)
	}
}

// BlankLine prints an empty line.
func BlankLine() {
	fmt.Fprintln(output)
}
