package names

import (
	"fmt"
	"testing"
)

// BenchmarkNew measures generator creation overhead.
func BenchmarkNew(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = New()
	}
}

// BenchmarkGenerate measures name generation performance.
func BenchmarkGenerate(b *testing.B) {
	g := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Generate()
	}
}

// BenchmarkGenerateDefault measures default generator performance.
func BenchmarkGenerateDefault(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Generate()
	}
}

// BenchmarkGenerateUnique measures unique name generation.
func BenchmarkGenerateUnique(b *testing.B) {
	g := New()

	// Small existing set - should find unique quickly
	b.Run("small-set", func(b *testing.B) {
		existing := map[string]bool{
			"swift-falcon": true,
			"clever-otter": true,
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = g.GenerateUnique(existing, 100)
		}
	})

	// Medium set
	b.Run("medium-set", func(b *testing.B) {
		existing := make(map[string]bool)
		for i := 0; i < 100; i++ {
			existing[g.Generate()] = true
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = g.GenerateUnique(existing, 100)
		}
	})
}

// BenchmarkGenerateUniqueFromList measures unique generation from slice.
func BenchmarkGenerateUniqueFromList(b *testing.B) {
	g := New()

	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("list-%d", size), func(b *testing.B) {
			existingNames := make([]string, size)
			for i := 0; i < size; i++ {
				existingNames[i] = g.Generate()
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = g.GenerateUniqueFromList(existingNames, 100)
			}
		})
	}
}

// BenchmarkGenerateUniqueDefault measures default generator unique generation.
func BenchmarkGenerateUniqueDefault(b *testing.B) {
	existing := map[string]bool{
		"swift-falcon": true,
		"clever-otter": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateUnique(existing, 100)
	}
}

// BenchmarkGenerateUniqueFromListDefault measures default generator with list.
func BenchmarkGenerateUniqueFromListDefault(b *testing.B) {
	existingNames := []string{"swift-falcon", "clever-otter", "bright-panda"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = GenerateUniqueFromList(existingNames, 100)
	}
}

// BenchmarkGenerateCollision measures performance when many names are taken.
// This simulates a scenario where most simple names are already used.
func BenchmarkGenerateCollision(b *testing.B) {
	g := New()

	// Take half of all possible combinations
	numPossible := len(adjectives) * len(animals)
	existing := make(map[string]bool)
	for i := 0; i < numPossible/2; i++ {
		existing[g.Generate()] = true
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUnique(existing, 100)
	}
}
