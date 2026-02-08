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

	// Agent role colors
	RoleCoordinator lipgloss.Color
	RoleEngineer    lipgloss.Color
	RoleQA          lipgloss.Color
	RoleTechLead    lipgloss.Color
	RolePM          lipgloss.Color
}

// ThemeName identifies a theme by name.
type ThemeName string

const (
	ThemeDark         ThemeName = "dark"
	ThemeLight        ThemeName = "light"
	ThemeHighContrast ThemeName = "high-contrast"
)

// AvailableThemes returns the list of available theme names.
func AvailableThemes() []ThemeName {
	return []ThemeName{ThemeDark, ThemeLight, ThemeHighContrast}
}

// DefaultTheme returns the default Ayu-inspired dark theme.
func DefaultTheme() Theme {
	return DarkTheme()
}

// DarkTheme returns the Ayu-inspired dark theme.
func DarkTheme() Theme {
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

		// Agent roles (muted/pastel for dark theme)
		RoleCoordinator: lipgloss.Color("#6B9FD4"), // Blue
		RoleEngineer:    lipgloss.Color("#7BC96F"), // Green
		RoleQA:          lipgloss.Color("#B48EAD"), // Purple
		RoleTechLead:    lipgloss.Color("#EBCB8B"), // Orange
		RolePM:          lipgloss.Color("#88C0D0"), // Teal
	}
}

// LightTheme returns a light theme suitable for bright environments.
func LightTheme() Theme {
	return Theme{
		// Base
		Background: lipgloss.Color("#FAFAFA"),
		Foreground: lipgloss.Color("#5C6166"),
		Border:     lipgloss.Color("#D4D5D6"),
		Muted:      lipgloss.Color("#8A9199"),

		// Accent
		Primary:   lipgloss.Color("#FF9940"),
		Secondary: lipgloss.Color("#399EE6"),
		Accent:    lipgloss.Color("#FA8D3E"),

		// Status
		Success: lipgloss.Color("#6CBF43"),
		Warning: lipgloss.Color("#F2AE49"),
		Error:   lipgloss.Color("#E65050"),
		Info:    lipgloss.Color("#399EE6"),

		// UI
		Selection:   lipgloss.Color("#035BD6"),
		HeaderBg:    lipgloss.Color("#E8E9EB"),
		StatusBarBg: lipgloss.Color("#E8E9EB"),

		// Agent roles (saturated for light theme)
		RoleCoordinator: lipgloss.Color("#2563EB"), // Blue
		RoleEngineer:    lipgloss.Color("#16A34A"), // Green
		RoleQA:          lipgloss.Color("#9333EA"), // Purple
		RoleTechLead:    lipgloss.Color("#EA580C"), // Orange
		RolePM:          lipgloss.Color("#0891B2"), // Teal
	}
}

// HighContrastTheme returns a high contrast theme for accessibility.
func HighContrastTheme() Theme {
	return Theme{
		// Base - pure black/white for maximum contrast
		Background: lipgloss.Color("#000000"),
		Foreground: lipgloss.Color("#FFFFFF"),
		Border:     lipgloss.Color("#FFFFFF"),
		Muted:      lipgloss.Color("#AAAAAA"),

		// Accent - bright, distinct colors
		Primary:   lipgloss.Color("#FFFF00"),
		Secondary: lipgloss.Color("#00FFFF"),
		Accent:    lipgloss.Color("#FF00FF"),

		// Status - vivid colors for clear distinction
		Success: lipgloss.Color("#00FF00"),
		Warning: lipgloss.Color("#FFFF00"),
		Error:   lipgloss.Color("#FF0000"),
		Info:    lipgloss.Color("#00FFFF"),

		// UI
		Selection:   lipgloss.Color("#0000FF"),
		HeaderBg:    lipgloss.Color("#333333"),
		StatusBarBg: lipgloss.Color("#333333"),

		// Agent roles (high contrast, distinct colors)
		RoleCoordinator: lipgloss.Color("#00BFFF"), // Blue
		RoleEngineer:    lipgloss.Color("#00FF00"), // Green
		RoleQA:          lipgloss.Color("#FF00FF"), // Purple
		RoleTechLead:    lipgloss.Color("#FFA500"), // Orange
		RolePM:          lipgloss.Color("#00FFFF"), // Teal
	}
}

// GetTheme returns a theme by name. Falls back to dark theme if not found.
func GetTheme(name ThemeName) Theme {
	switch name {
	case ThemeLight:
		return LightTheme()
	case ThemeHighContrast:
		return HighContrastTheme()
	case ThemeDark:
		fallthrough
	default:
		return DarkTheme()
	}
}

// GetThemeByString returns a theme by string name. Falls back to dark theme.
func GetThemeByString(name string) Theme {
	return GetTheme(ThemeName(name))
}

// Styles contains pre-built lipgloss styles for common elements.
type Styles struct {
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

	theme Theme
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

// MessageTypeStyle returns the appropriate style for a channel message type.
// Message types: text, task, review, approval, merge, status
func (s Styles) MessageTypeStyle(msgType string) lipgloss.Style {
	switch msgType {
	case "task":
		return s.Warning // Orange/yellow for tasks that need attention
	case "review":
		return s.Info // Blue for review requests
	case "approval":
		return s.Success // Green for approvals
	case "merge":
		return lipgloss.NewStyle().Foreground(s.theme.Accent) // Accent color for merge
	case "status":
		return s.Muted // Muted for status updates
	default:
		return s.Normal // Default for regular text messages
	}
}

// MessageTypeIcon returns an icon/emoji prefix for a channel message type.
func (s Styles) MessageTypeIcon(msgType string) string {
	switch msgType {
	case "task":
		return "📋 "
	case "review":
		return "👀 "
	case "approval":
		return "✅ "
	case "merge":
		return "🔀 "
	case "status":
		return "📊 "
	default:
		return ""
	}
}

// RoleColor returns the color for an agent role.
// Recognized roles: coordinator, engineer, qa, tech-lead, product-manager (or pm)
func (s Styles) RoleColor(role string) lipgloss.Color {
	switch role {
	case "coordinator":
		return s.theme.RoleCoordinator
	case "engineer":
		return s.theme.RoleEngineer
	case "qa":
		return s.theme.RoleQA
	case "tech-lead":
		return s.theme.RoleTechLead
	case "product-manager", "pm":
		return s.theme.RolePM
	default:
		return s.theme.Foreground
	}
}

// RoleStyle returns a lipgloss style with the role's color.
func (s Styles) RoleStyle(role string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(s.RoleColor(role))
}

// RoleBadge returns a pill-shaped badge style for the given role.
// The badge has the role color as background with contrasting text.
func (s Styles) RoleBadge(role string) lipgloss.Style {
	return lipgloss.NewStyle().
		Background(s.RoleColor(role)).
		Foreground(s.theme.Background).
		Padding(0, 1).
		Bold(true)
}

// RoleFromAgentName extracts the role from an agent name.
// Agent names follow the pattern: role-NN (e.g., "engineer-01", "tech-lead-02")
func RoleFromAgentName(agentName string) string {
	// Handle special cases
	switch agentName {
	case "coordinator":
		return "coordinator"
	case "manager":
		return "coordinator"
	case "product-manager":
		return "product-manager"
	}

	// Extract role prefix from names like "engineer-01", "tech-lead-02", "qa-01"
	// Find the last dash followed by digits
	for i := len(agentName) - 1; i >= 0; i-- {
		if agentName[i] == '-' {
			suffix := agentName[i+1:]
			if isNumeric(suffix) {
				return agentName[:i]
			}
		}
	}

	// No numeric suffix found, return the whole name as role
	return agentName
}

// isNumeric checks if a string contains only digits.
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
