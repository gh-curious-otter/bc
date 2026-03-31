package ui

import (
	"strings"
	"testing"
)

func TestColor(t *testing.T) {
	// Enable colors for testing
	SetColorEnabled(true)

	tests := []struct {
		name     string
		text     string
		color    string
		contains string
	}{
		{"red text", "error", Red, "\033[31m"},
		{"green text", "success", Green, "\033[32m"},
		{"yellow text", "warning", Yellow, "\033[33m"},
		{"blue text", "info", Blue, "\033[34m"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Color(tc.text, tc.color)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("Color(%q, %q) = %q, want contains %q", tc.text, tc.color, result, tc.contains)
			}
			if !strings.Contains(result, tc.text) {
				t.Errorf("Color(%q, %q) = %q, want contains %q", tc.text, tc.color, result, tc.text)
			}
			if !strings.HasSuffix(result, Reset) {
				t.Errorf("Color(%q, %q) = %q, want suffix %q", tc.text, tc.color, result, Reset)
			}
		})
	}
}

func TestColorDisabled(t *testing.T) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)

	result := Color("test", Red)
	if result != "test" {
		t.Errorf("Color with colors disabled = %q, want %q", result, "test")
	}
}

func TestColorize(t *testing.T) {
	SetColorEnabled(true)

	result := Colorize("test", Bold, Red)
	if !strings.Contains(result, Bold) {
		t.Errorf("Colorize should contain Bold code")
	}
	if !strings.Contains(result, Red) {
		t.Errorf("Colorize should contain Red code")
	}
}

func TestColorEnabled(t *testing.T) {
	SetColorEnabled(true)
	if !ColorEnabled() {
		t.Error("ColorEnabled should return true")
	}

	SetColorEnabled(false)
	if ColorEnabled() {
		t.Error("ColorEnabled should return false")
	}
}

func TestGrayText(t *testing.T) {
	SetColorEnabled(true)
	result := GrayText("test")
	if !strings.Contains(result, "test") {
		t.Error("GrayText should contain text")
	}
}

// Benchmarks

func BenchmarkColor(b *testing.B) {
	SetColorEnabled(true)
	for i := 0; i < b.N; i++ {
		_ = Color("test message", Red)
	}
}

func BenchmarkColorDisabled(b *testing.B) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)
	for i := 0; i < b.N; i++ {
		_ = Color("test message", Red)
	}
}

func BenchmarkColorize(b *testing.B) {
	SetColorEnabled(true)
	for i := 0; i < b.N; i++ {
		_ = Colorize("test message", Bold, Red, Under)
	}
}

func BenchmarkStyleHelpers(b *testing.B) {
	SetColorEnabled(true)
	b.Run("BoldText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = BoldText("test")
		}
	})
	b.Run("RedText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = RedText("test")
		}
	})
	b.Run("GreenText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = GreenText("test")
		}
	})
}

func BenchmarkNewTable(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewTable("Name", "Value", "Status", "Date")
	}
}

func BenchmarkTableAddRow(b *testing.B) {
	table := NewTable("Name", "Value", "Status")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		table.AddRow("test-name", "test-value", "active")
	}
}

func BenchmarkTableString_SmallTable(b *testing.B) {
	SetColorEnabled(false)
	table := NewTable("Name", "Value")
	table.AddRow("foo", "bar")
	table.AddRow("baz", "qux")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.String()
	}
}

func BenchmarkTableString_LargeTable(b *testing.B) {
	SetColorEnabled(false)
	table := NewTable("ID", "Name", "Status", "Created", "Updated")
	for i := 0; i < 100; i++ {
		table.AddRow("id-123", "test-name", "active", "2024-01-01", "2024-01-02")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = table.String()
	}
}

func BenchmarkPadRight(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = padRight("short", 20)
	}
}

func BenchmarkPadRight_NoOp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = padRight("already long enough text", 10)
	}
}
