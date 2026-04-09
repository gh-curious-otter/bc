package provider

import (
	"context"
	"testing"
)

// BenchmarkNewRegistry measures registry creation.
func BenchmarkNewRegistry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewRegistry()
	}
}

// BenchmarkRegistryRegister measures provider registration.
func BenchmarkRegistryRegister(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		r := NewRegistry()
		p := NewClaudeProvider()
		b.StartTimer()
		r.Register(p)
	}
}

// BenchmarkRegistryGet measures provider lookup.
func BenchmarkRegistryGet(b *testing.B) {
	r := NewRegistry()
	r.Register(NewClaudeProvider())
	r.Register(NewCodexProvider())
	r.Register(NewGeminiProvider())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get("claude")
	}
}

// BenchmarkRegistryGetMiss measures provider lookup miss.
func BenchmarkRegistryGetMiss(b *testing.B) {
	r := NewRegistry()
	r.Register(NewClaudeProvider())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Get("nonexistent")
	}
}

// BenchmarkRegistryList measures listing all providers.
func BenchmarkRegistryList(b *testing.B) {
	r := NewRegistry()
	r.Register(NewClaudeProvider())
	r.Register(NewCodexProvider())
	r.Register(NewGeminiProvider())
	r.Register(NewCursorProvider())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = r.List()
	}
}

// BenchmarkGetProvider measures default registry lookup.
func BenchmarkGetProvider(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetProvider("claude")
	}
}

// BenchmarkListProviders measures listing from default registry.
func BenchmarkListProviders(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ListProviders()
	}
}

// BenchmarkProviderName measures name getter.
func BenchmarkProviderName(b *testing.B) {
	p := NewClaudeProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Name()
	}
}

// BenchmarkProviderCommand measures command getter.
func BenchmarkProviderCommand(b *testing.B) {
	p := NewClaudeProvider()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.Command()
	}
}

// BenchmarkDetectStateWorking measures state detection for working state.
func BenchmarkDetectStateWorking(b *testing.B) {
	p := NewClaudeProvider()
	output := "Reading file...\n✻ Processing request\n✳ Thinking..."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.DetectState(output)
	}
}

// BenchmarkDetectStateIdle measures state detection for idle state.
func BenchmarkDetectStateIdle(b *testing.B) {
	p := NewClaudeProvider()
	output := "Done processing.\n❯ Ready for input"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.DetectState(output)
	}
}

// BenchmarkDetectStateUnknown measures state detection for unknown state.
func BenchmarkDetectStateUnknown(b *testing.B) {
	p := NewClaudeProvider()
	output := "Some random output\nwithout state indicators"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.DetectState(output)
	}
}

// BenchmarkDetectStateLongOutput measures state detection with long output.
func BenchmarkDetectStateLongOutput(b *testing.B) {
	p := NewClaudeProvider()
	output := ""
	for range 100 {
		output += "Some log line with content\n"
	}
	output += "✻ Working on task"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.DetectState(output)
	}
}

// BenchmarkAllProvidersDetectState measures state detection across all providers.
func BenchmarkAllProvidersDetectState(b *testing.B) {
	providers := []Provider{
		NewClaudeProvider(),
		NewCodexProvider(),
		NewGeminiProvider(),
		NewCursorProvider(),
	}
	output := "Processing...\n✻ Working"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, p := range providers {
			_ = p.DetectState(output)
		}
	}
}

// BenchmarkNewClaudeProvider measures provider creation.
func BenchmarkNewClaudeProvider(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewClaudeProvider()
	}
}

// BenchmarkNewCodexProvider measures provider creation.
func BenchmarkNewCodexProvider(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewCodexProvider()
	}
}

// BenchmarkIsInstalled measures binary check (fast path - binary exists).
func BenchmarkIsInstalled(b *testing.B) {
	p := NewClaudeProvider()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.IsInstalled(ctx)
	}
}

// BenchmarkRegistryGetConcurrent measures concurrent registry lookups.
func BenchmarkRegistryGetConcurrent(b *testing.B) {
	r := NewRegistry()
	r.Register(NewClaudeProvider())
	r.Register(NewCodexProvider())
	r.Register(NewGeminiProvider())

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = r.Get("claude")
		}
	})
}
