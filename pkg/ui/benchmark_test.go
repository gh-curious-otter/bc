package ui

import (
	"testing"
)

// BenchmarkColor measures basic color application.
func BenchmarkColor(b *testing.B) {
	SetColorEnabled(true)
	defer SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Color("test text", Green)
	}
}

// BenchmarkColorDisabled measures color when disabled.
func BenchmarkColorDisabled(b *testing.B) {
	SetColorEnabled(false)
	defer SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Color("test text", Green)
	}
}

// BenchmarkColorize measures multi-code colorization.
func BenchmarkColorize(b *testing.B) {
	SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Colorize("test text", Bold, Green)
	}
}

// BenchmarkColorizeMany measures colorization with many codes.
func BenchmarkColorizeMany(b *testing.B) {
	SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Colorize("test text", Bold, Italic, Under, Green)
	}
}

// BenchmarkRedText measures common helper.
func BenchmarkRedText(b *testing.B) {
	SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RedText("error message")
	}
}

// BenchmarkGreenText measures success helper.
func BenchmarkGreenText(b *testing.B) {
	SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GreenText("success message")
	}
}

// BenchmarkBoldText measures bold helper.
func BenchmarkBoldText(b *testing.B) {
	SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BoldText("important text")
	}
}

// BenchmarkDimText measures dim helper.
func BenchmarkDimText(b *testing.B) {
	SetColorEnabled(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = DimText("secondary text")
	}
}

// BenchmarkColorEnabled measures enabled check.
func BenchmarkColorEnabled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ColorEnabled()
	}
}

// BenchmarkSetColorEnabled measures toggle.
func BenchmarkSetColorEnabled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		SetColorEnabled(i%2 == 0)
	}
}

// BenchmarkColorConcurrent measures concurrent color application.
func BenchmarkColorConcurrent(b *testing.B) {
	SetColorEnabled(true)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = GreenText("concurrent text")
		}
	})
}
