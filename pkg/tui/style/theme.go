// Package style provides theming and styling for the TUI components.
package style

import "github.com/charmbracelet/lipgloss"

// Theme defines the color palette and styles for the TUI.
type Theme struct {
	// Base colors
	Background lipgloss.Color
	Foreground lipgloss.Color
	Border     lipgloss.Color
	Muted      lipgloss.Color

	// Accent colors
	Primary   lipgloss.Color
	Secondary lipgloss.Color
	Accent    lipgloss.Color

	// Status colors
	Success lipgloss.Color
	Warning lipgloss.Color
	Error   lipgloss.Color
	Info    lipgloss.Color

	// UI element colors
	Selection   lipgloss.Color
	HeaderBg    lipgloss.Color
	StatusBarBg lipgloss.Color
}

// DefaultTheme returns the default Ayu-inspired dark theme.
func DefaultTheme() Theme {
	return Theme{
		// Base
		Background: lipgloss.Color("#0B0E14"),
		Foreground: lipgloss.Color("#BFBDB6"),
		Border:     lipgloss.Color("#565B66"),
		Muted:      lipgloss.Color("#565B66"),

		// Accent
		Primary:   lipgloss.Color("#E6B450"),
		Secondary: lipgloss.Color("#59C2FF"),
		Accent:    lipgloss.Color("#FF8F40"),

		// Status
		Success: lipgloss.Color("#AAD94C"),
		Warning: lipgloss.Color("#FF8F40"),
		Error:   lipgloss.Color("#F07178"),
		Info:    lipgloss.Color("#59C2FF"),

		// UI
		Selection:   lipgloss.Color("#409FFF"),
		HeaderBg:    lipgloss.Color("#1C2028"),
		StatusBarBg: lipgloss.Color("#1C2028"),
	}
}

// Styles contains pre-built lipgloss styles for common elements.
type Styles struct {
	theme Theme

	// Text styles
	Normal lipgloss.Style
	Bold   lipgloss.Style
	Muted  lipgloss.Style
	Code   lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Warning lipgloss.Style
	Error   lipgloss.Style
	Info    lipgloss.Style

	// UI element styles
	Header    lipgloss.Style
	StatusBar lipgloss.Style
	Border    lipgloss.Style
	Selected  lipgloss.Style
	Title     lipgloss.Style
}

// NewStyles creates a Styles instance from a theme.
func NewStyles(theme Theme) Styles {
	return Styles{
		theme: theme,

		Normal: lipgloss.NewStyle().
			Foreground(theme.Foreground),

		Bold: lipgloss.NewStyle().
			Foreground(theme.Foreground).
			Bold(true),

		Muted: lipgloss.NewStyle().
			Foreground(theme.Muted),

		Code: lipgloss.NewStyle().
			Foreground(theme.Secondary),

		Success: lipgloss.NewStyle().
			Foreground(theme.Success),

		Warning: lipgloss.NewStyle().
			Foreground(theme.Warning),

		Error: lipgloss.NewStyle().
			Foreground(theme.Error),

		Info: lipgloss.NewStyle().
			Foreground(theme.Info),

		Header: lipgloss.NewStyle().
			Background(theme.HeaderBg).
			Foreground(theme.Primary).
			Bold(true).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Background(theme.StatusBarBg).
			Foreground(theme.Foreground).
			Padding(0, 1),

		Border: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border),

		Selected: lipgloss.NewStyle().
			Background(theme.Selection).
			Foreground(theme.Background).
			Bold(true),

		Title: lipgloss.NewStyle().
			Foreground(theme.Primary).
			Bold(true).
			MarginBottom(1),
	}
}

// DefaultStyles returns styles using the default theme.
func DefaultStyles() Styles {
	return NewStyles(DefaultTheme())
}

// Theme returns the underlying theme.
func (s Styles) Theme() Theme {
	return s.theme
}

// StatusStyle returns the appropriate style for a status string.
func (s Styles) StatusStyle(status string) lipgloss.Style {
	switch status {
	case "ok", "running", "success", "active", "merged":
		return s.Success
	case "warning", "pending", "queued", "spawning":
		return s.Warning
	case "error", "failed", "stuck", "blocked":
		return s.Error
	case "info", "idle", "stopped":
		return s.Info
	default:
		return s.Normal
	}
}
