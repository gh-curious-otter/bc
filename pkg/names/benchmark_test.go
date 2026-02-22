package names

import (
	"testing"
)

// BenchmarkNew measures generator creation.
func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New()
	}
}

// BenchmarkGenerate measures name generation.
func BenchmarkGenerate(b *testing.B) {
	g := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Generate()
	}
}

// BenchmarkGenerateDefault measures default generator.
func BenchmarkGenerateDefault(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Generate()
	}
}

// BenchmarkGenerateUnique measures unique name generation.
func BenchmarkGenerateUnique(b *testing.B) {
	g := New()
	existing := make(map[string]bool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name, _ := g.GenerateUnique(existing, 100)
		existing[name] = true
	}
}

// BenchmarkGenerateUniqueFromList measures unique name from list.
func BenchmarkGenerateUniqueFromList(b *testing.B) {
	g := New()
	existingList := []string{"swift-falcon", "clever-otter", "bright-panda"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUniqueFromList(existingList, 100)
	}
}

// BenchmarkGenerateUniqueWithCollisions measures generation with many existing names.
func BenchmarkGenerateUniqueWithCollisions(b *testing.B) {
	sizes := []int{100, 500, 1000}

	for _, size := range sizes {
		b.Run(string(rune('0'+size/100)), func(b *testing.B) {
			g := New()
			existing := make(map[string]bool, size)

			// Pre-populate with names
			for i := 0; i < size; i++ {
				existing[g.Generate()] = true
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				name, _ := g.GenerateUnique(existing, 1000)
				// Don't add to existing so we don't exhaust combinations
				_ = name
			}
		})
	}
}

// BenchmarkGenerateConcurrent measures concurrent name generation.
func BenchmarkGenerateConcurrent(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		g := New() // Each goroutine gets own generator
		for pb.Next() {
			_ = g.Generate()
		}
	})
}
