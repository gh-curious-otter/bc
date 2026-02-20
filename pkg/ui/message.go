//nolint:errcheck // UI output errors to stdout are not recoverable
package ui

import (
	"fmt"
	"io"
	"os"
)

// Default output writer (can be overridden for testing).
var output io.Writer = os.Stdout

// SetOutput sets the output writer for all messages.
func SetOutput(w io.Writer) {
	output = w
}

// Message prefixes with colors.
const (
	successPrefix = "✓"
	errorPrefix   = "✗"
	warningPrefix = "!"
	infoPrefix    = "→"
	debugPrefix   = "·"
)

// Success prints a success message with green checkmark.
func Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if colorEnabled {
		fmt.Fprintf(output, "%s%s%s %s\n", Green, successPrefix, Reset, msg)
	} else {
		fmt.Fprintf(output, "%s %s\n", successPrefix, msg)
	}
}

// Successf is an alias for Success.
func Successf(format string, args ...any) {
	Success(format, args...)
}

// Error prints an error message with red X.
func Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if colorEnabled {
		fmt.Fprintf(output, "%s%s%s %s\n", Red, errorPrefix, Reset, msg)
	} else {
		fmt.Fprintf(output, "%s %s\n", errorPrefix, msg)
	}
}

// Errorf is an alias for Error.
func Errorf(format string, args ...any) {
	Error(format, args...)
}

// Warning prints a warning message with yellow exclamation.
func Warning(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if colorEnabled {
		fmt.Fprintf(output, "%s%s%s %s\n", Yellow, warningPrefix, Reset, msg)
	} else {
		fmt.Fprintf(output, "%s %s\n", warningPrefix, msg)
	}
}

// Warningf is an alias for Warning.
func Warningf(format string, args ...any) {
	Warning(format, args...)
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

// Infof is an alias for Info.
func Infof(format string, args ...any) {
	Info(format, args...)
}

// Debug prints a debug message with dim dot (only when verbose).
func Debug(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if colorEnabled {
		fmt.Fprintf(output, "%s%s %s%s\n", Dim, debugPrefix, msg, Reset)
	} else {
		fmt.Fprintf(output, "%s %s\n", debugPrefix, msg)
	}
}

// Debugf is an alias for Debug.
func Debugf(format string, args ...any) {
	Debug(format, args...)
}

// Print prints plain text.
func Print(format string, args ...any) {
	fmt.Fprintf(output, format, args...)
}

// Println prints plain text with newline.
func Println(format string, args ...any) {
	fmt.Fprintf(output, format+"\n", args...)
}

// Header prints a bold header line.
func Header(text string) {
	if colorEnabled {
		fmt.Fprintf(output, "%s%s%s\n", Bold, text, Reset)
	} else {
		fmt.Fprintf(output, "%s\n", text)
	}
}

// Divider prints a horizontal divider line.
func Divider(width int) {
	for range width {
		fmt.Fprint(output, "─")
	}
	fmt.Fprintln(output)
}

// BlankLine prints an empty line.
func BlankLine() {
	fmt.Fprintln(output)
}
