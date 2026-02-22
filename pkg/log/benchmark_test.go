package log

import (
	"io"
	"testing"
)

// BenchmarkInfo measures info logging throughput.
func BenchmarkInfo(b *testing.B) {
	// Discard output for benchmarking
	SetOutput(io.Discard)
	defer SetOutput(io.Discard) // Keep discard after test

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message", "iteration", i)
	}
}

// BenchmarkDebug measures debug logging (disabled).
func BenchmarkDebugDisabled(b *testing.B) {
	SetOutput(io.Discard)
	SetVerbose(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("benchmark debug message", "iteration", i)
	}
}

// BenchmarkDebugEnabled measures debug logging (enabled).
func BenchmarkDebugEnabled(b *testing.B) {
	SetOutput(io.Discard)
	SetVerbose(true)
	defer SetVerbose(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("benchmark debug message", "iteration", i)
	}
}

// BenchmarkWarn measures warning logging.
func BenchmarkWarn(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Warn("benchmark warning", "code", 123)
	}
}

// BenchmarkError measures error logging.
func BenchmarkError(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Error("benchmark error", "code", 500)
	}
}

// BenchmarkSetVerbose measures verbose toggle.
func BenchmarkSetVerbose(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SetVerbose(i%2 == 0)
	}
}

// BenchmarkInfoConcurrent measures concurrent info logging.
func BenchmarkInfoConcurrent(b *testing.B) {
	SetOutput(io.Discard)

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			Info("concurrent message", "id", i)
			i++
		}
	})
}

// BenchmarkInfoWithManyArgs measures logging with multiple args.
func BenchmarkInfoWithManyArgs(b *testing.B) {
	SetOutput(io.Discard)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message",
			"arg1", "value1",
			"arg2", 42,
			"arg3", true,
			"arg4", 3.14,
		)
	}
}
