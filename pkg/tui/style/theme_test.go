package style

import (
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
