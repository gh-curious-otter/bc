// Package ui provides consistent CLI output formatting utilities.
package ui

import (
	"os"

	"github.com/charmbracelet/x/term"
)

// ANSI color codes
const (
	Reset  = "\033[0m"
	Bold   = "\033[1m"
	Dim    = "\033[2m"
	Italic = "\033[3m"
	Under  = "\033[4m"

	// Foreground colors
	Black   = "\033[30m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"

	// Bright foreground colors
	BrightBlack   = "\033[90m"
	BrightRed     = "\033[91m"
	BrightGreen   = "\033[92m"
	BrightYellow  = "\033[93m"
	BrightBlue    = "\033[94m"
	BrightMagenta = "\033[95m"
	BrightCyan    = "\033[96m"
	BrightWhite   = "\033[97m"
)

// colorEnabled caches whether colors should be used.
var colorEnabled = checkColorSupport()

// checkColorSupport determines if the terminal supports colors.
func checkColorSupport() bool {
	// Respect NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// Check if stdout is a terminal
	return term.IsTerminal(os.Stdout.Fd())
}

// SetColorEnabled overrides automatic color detection.
func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

// ColorEnabled returns whether colors are enabled.
func ColorEnabled() bool {
	return colorEnabled
}

// Color wraps text in the given color code if colors are enabled.
func Color(text, color string) string {
	if !colorEnabled {
		return text
	}
	return color + text + Reset
}

// Colorize applies color to text.
func Colorize(text string, codes ...string) string {
	if !colorEnabled || len(codes) == 0 {
		return text
	}
	var prefix string
	for _, c := range codes {
		prefix += c
	}
	return prefix + text + Reset
}

// Style helpers for common formatting patterns.

// BoldText makes text bold.
func BoldText(text string) string {
	return Color(text, Bold)
}

// DimText makes text dim.
func DimText(text string) string {
	return Color(text, Dim)
}

// RedText colors text red.
func RedText(text string) string {
	return Color(text, Red)
}

// GreenText colors text green.
func GreenText(text string) string {
	return Color(text, Green)
}

// YellowText colors text yellow.
func YellowText(text string) string {
	return Color(text, Yellow)
}

// BlueText colors text blue.
func BlueText(text string) string {
	return Color(text, Blue)
}

// CyanText colors text cyan.
func CyanText(text string) string {
	return Color(text, Cyan)
}

// MagentaText colors text magenta.
func MagentaText(text string) string {
	return Color(text, Magenta)
}
