package log

import (
	"io"
	"testing"
)

// BenchmarkInfo measures Info logging performance.
func BenchmarkInfo(b *testing.B) {
	// Discard output to avoid I/O overhead
	SetOutput(io.Discard)
	defer SetOutput(io.Discard) // Reset after test

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("test message")
	}
}

// BenchmarkInfoWithArgs measures Info logging with arguments.
func BenchmarkInfoWithArgs(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("test message", "key1", "value1", "key2", 42)
	}
}

// BenchmarkDebugDisabled measures Debug logging when verbose is off.
func BenchmarkDebugDisabled(b *testing.B) {
	SetOutput(io.Discard)
	SetVerbose(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("test message", "key", "value")
	}
}

// BenchmarkDebugEnabled measures Debug logging when verbose is on.
func BenchmarkDebugEnabled(b *testing.B) {
	SetVerbose(true)
	SetOutput(io.Discard) // Must be after SetVerbose as SetVerbose recreates logger
	defer SetVerbose(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("test message", "key", "value")
	}
}

// BenchmarkWarn measures Warn logging performance.
func BenchmarkWarn(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Warn("warning message", "error", "something went wrong")
	}
}

// BenchmarkError measures Error logging performance.
func BenchmarkError(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Error("error message", "error", "fatal error occurred")
	}
}

// BenchmarkWith measures creating a child logger with attributes.
func BenchmarkWith(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = With("component", "test", "request_id", "abc123")
	}
}

// BenchmarkLogger measures getting the underlying logger.
func BenchmarkLogger(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Logger()
	}
}

// BenchmarkSetVerbose measures toggling verbose mode.
func BenchmarkSetVerbose(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetVerbose(i%2 == 0)
	}
}

// BenchmarkSetOutput measures changing output destination.
func BenchmarkSetOutput(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetOutput(io.Discard)
	}
}

// BenchmarkConcurrentInfo measures concurrent logging.
func BenchmarkConcurrentInfo(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("concurrent message", "goroutine", "test")
		}
	})
}

// BenchmarkConcurrentMixed measures mixed concurrent operations.
func BenchmarkConcurrentMixed(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			switch i % 4 {
			case 0:
				Info("info message")
			case 1:
				Debug("debug message")
			case 2:
				Warn("warn message")
			case 3:
				Error("error message")
			}
			i++
		}
	})
}
