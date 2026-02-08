package style

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultTheme(t *testing.T) {
	theme := DefaultTheme()
	if theme.Background == "" {
		t.Error("DefaultTheme should have background color")
	}
	if theme.Foreground == "" {
		t.Error("DefaultTheme should have foreground color")
	}
}

func TestDarkTheme(t *testing.T) {
	theme := DarkTheme()
	// Dark theme should have dark background
	if theme.Background != lipgloss.Color("#0B0E14") {
		t.Errorf("DarkTheme background = %v, want #0B0E14", theme.Background)
	}
}

func TestLightTheme(t *testing.T) {
	theme := LightTheme()
	// Light theme should have light background
	if theme.Background != lipgloss.Color("#FAFAFA") {
		t.Errorf("LightTheme background = %v, want #FAFAFA", theme.Background)
	}
}

func TestHighContrastTheme(t *testing.T) {
	theme := HighContrastTheme()
	// High contrast should use pure black/white
	if theme.Background != lipgloss.Color("#000000") {
		t.Errorf("HighContrastTheme background = %v, want #000000", theme.Background)
	}
	if theme.Foreground != lipgloss.Color("#FFFFFF") {
		t.Errorf("HighContrastTheme foreground = %v, want #FFFFFF", theme.Foreground)
	}
}

func TestGetTheme(t *testing.T) {
	tests := []struct {
		name   ThemeName
		wantBg lipgloss.Color
	}{
		{ThemeDark, lipgloss.Color("#0B0E14")},
		{ThemeLight, lipgloss.Color("#FAFAFA")},
		{ThemeHighContrast, lipgloss.Color("#000000")},
		{"unknown", lipgloss.Color("#0B0E14")}, // Falls back to dark
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			theme := GetTheme(tt.name)
			if theme.Background != tt.wantBg {
				t.Errorf("GetTheme(%q) background = %v, want %v", tt.name, theme.Background, tt.wantBg)
			}
		})
	}
}

func TestGetThemeByString(t *testing.T) {
	theme := GetThemeByString("light")
	if theme.Background != lipgloss.Color("#FAFAFA") {
		t.Errorf("GetThemeByString(light) background = %v, want #FAFAFA", theme.Background)
	}
}

func TestAvailableThemes(t *testing.T) {
	themes := AvailableThemes()
	if len(themes) != 3 {
		t.Errorf("AvailableThemes() returned %d themes, want 3", len(themes))
	}

	expected := map[ThemeName]bool{
		ThemeDark:         false,
		ThemeLight:        false,
		ThemeHighContrast: false,
	}
	for _, theme := range themes {
		if _, ok := expected[theme]; ok {
			expected[theme] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("AvailableThemes() missing %q", name)
		}
	}
}

func TestNewStyles(t *testing.T) {
	theme := DarkTheme()
	styles := NewStyles(theme)

	// Verify styles render without error
	rendered := styles.Normal.Render("test")
	if rendered == "" {
		t.Error("Normal style should render text")
	}
	if styles.Theme().Background != theme.Background {
		t.Error("Styles.Theme() should return the theme used to create it")
	}
}

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()
	if styles.Theme().Background != DefaultTheme().Background {
		t.Error("DefaultStyles should use DefaultTheme")
	}
}

func TestStatusStyle(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		status    string
		wantStyle string
	}{
		{"success", "success"},
		{"running", "success"},
		{"warning", "warning"},
		{"pending", "warning"},
		{"error", "error"},
		{"failed", "error"},
		{"info", "info"},
		{"idle", "info"},
		{"unknown", "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			style := styles.StatusStyle(tt.status)
			// Just verify we get a valid style back (non-panic)
			_ = style.Render("test")
		})
	}
}

func TestMessageTypeStyle(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		msgType string
	}{
		{"text"},
		{"task"},
		{"review"},
		{"approval"},
		{"merge"},
		{"status"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.msgType, func(t *testing.T) {
			style := styles.MessageTypeStyle(tt.msgType)
			// Verify we get a valid style back (non-panic)
			rendered := style.Render("test message")
			if rendered == "" {
				t.Errorf("MessageTypeStyle(%q) should render text", tt.msgType)
			}
		})
	}
}

func TestMessageTypeIcon(t *testing.T) {
	styles := DefaultStyles()

	tests := []struct {
		msgType  string
		wantIcon string
	}{
		{"task", "📋 "},
		{"review", "👀 "},
		{"approval", "✅ "},
		{"merge", "🔀 "},
		{"status", "📊 "},
		{"text", ""},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.msgType, func(t *testing.T) {
			icon := styles.MessageTypeIcon(tt.msgType)
			if icon != tt.wantIcon {
				t.Errorf("MessageTypeIcon(%q) = %q, want %q", tt.msgType, icon, tt.wantIcon)
			}
		})
	}
}

func TestRoleColor(t *testing.T) {
	styles := DefaultStyles()
	theme := DarkTheme()

	tests := []struct {
		role      string
		wantColor lipgloss.Color
	}{
		{"coordinator", theme.RoleCoordinator},
		{"engineer", theme.RoleEngineer},
		{"qa", theme.RoleQA},
		{"tech-lead", theme.RoleTechLead},
		{"product-manager", theme.RolePM},
		{"pm", theme.RolePM},
		{"unknown", theme.Foreground},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			color := styles.RoleColor(tt.role)
			if color != tt.wantColor {
				t.Errorf("RoleColor(%q) = %v, want %v", tt.role, color, tt.wantColor)
			}
		})
	}
}

func TestRoleStyle(t *testing.T) {
	styles := DefaultStyles()

	roles := []string{"coordinator", "engineer", "qa", "tech-lead", "product-manager"}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			style := styles.RoleStyle(role)
			// Verify we get a valid style back (non-panic)
			rendered := style.Render("test")
			if rendered == "" {
				t.Errorf("RoleStyle(%q) should render text", role)
			}
		})
	}
}

func TestRoleBadge(t *testing.T) {
	styles := DefaultStyles()

	roles := []string{"coordinator", "engineer", "qa", "tech-lead", "product-manager"}

	for _, role := range roles {
		t.Run(role, func(t *testing.T) {
			badge := styles.RoleBadge(role)
			// Verify we get a valid style back (non-panic)
			rendered := badge.Render(role)
			if rendered == "" {
				t.Errorf("RoleBadge(%q) should render badge", role)
			}
		})
	}
}

func TestRoleFromAgentName(t *testing.T) {
	tests := []struct {
		agentName string
		wantRole  string
	}{
		{"engineer-01", "engineer"},
		{"engineer-02", "engineer"},
		{"tech-lead-01", "tech-lead"},
		{"tech-lead-02", "tech-lead"},
		{"qa-01", "qa"},
		{"qa-02", "qa"},
		{"coordinator", "coordinator"},
		{"manager", "coordinator"},
		{"product-manager", "product-manager"},
		{"unknown-agent", "unknown-agent"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.agentName, func(t *testing.T) {
			role := RoleFromAgentName(tt.agentName)
			if role != tt.wantRole {
				t.Errorf("RoleFromAgentName(%q) = %q, want %q", tt.agentName, role, tt.wantRole)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"01", true},
		{"123", true},
		{"0", true},
		{"", false},
		{"abc", false},
		{"12a", false},
		{"a12", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := isNumeric(tt.s)
			if got != tt.want {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestThemeHasRoleColors(t *testing.T) {
	themes := []struct {
		name  string
		theme Theme
	}{
		{"dark", DarkTheme()},
		{"light", LightTheme()},
		{"high-contrast", HighContrastTheme()},
	}

	for _, tt := range themes {
		t.Run(tt.name, func(t *testing.T) {
			if tt.theme.RoleCoordinator == "" {
				t.Error("Theme should have RoleCoordinator color")
			}
			if tt.theme.RoleEngineer == "" {
				t.Error("Theme should have RoleEngineer color")
			}
			if tt.theme.RoleQA == "" {
				t.Error("Theme should have RoleQA color")
			}
			if tt.theme.RoleTechLead == "" {
				t.Error("Theme should have RoleTechLead color")
			}
			if tt.theme.RolePM == "" {
				t.Error("Theme should have RolePM color")
			}
		})
	}
}

func TestMessageBubbleStyle(t *testing.T) {
	styles := DefaultStyles()

	// Verify MessageBubble style renders
	rendered := styles.MessageBubble.Render("test message")
	if rendered == "" {
		t.Error("MessageBubble style should render text")
	}
	if !strings.Contains(rendered, "test message") {
		t.Errorf("MessageBubble should contain message, got: %s", rendered)
	}

	// MessageBubbleOwn and MessageBubbleOthers both render and are theme-aware
	ownRendered := styles.MessageBubbleOwn.Render("own message")
	othersRendered := styles.MessageBubbleOthers.Render("others message")
	if ownRendered == "" || othersRendered == "" {
		t.Error("MessageBubbleOwn and MessageBubbleOthers should render")
	}
	if !strings.Contains(ownRendered, "own message") || !strings.Contains(othersRendered, "others message") {
		t.Error("own/others bubble styles should contain message content")
	}
}

func TestMessageBubbleThemeSupport(t *testing.T) {
	// All themes must define message bubble colors for own vs others
	themes := []struct {
		name  string
		theme Theme
	}{
		{"dark", DarkTheme()},
		{"light", LightTheme()},
		{"high-contrast", HighContrastTheme()},
	}
	for _, tt := range themes {
		t.Run(tt.name, func(t *testing.T) {
			if tt.theme.MessageBubbleOwnBg == "" {
				t.Error("theme must set MessageBubbleOwnBg")
			}
			if tt.theme.MessageBubbleOthersBg == "" {
				t.Error("theme must set MessageBubbleOthersBg")
			}
			styles := NewStyles(tt.theme)
			// Both styles should render without panic
			_ = styles.MessageBubbleOwn.Render("test")
			_ = styles.MessageBubbleOthers.Render("test")
		})
	}
}
