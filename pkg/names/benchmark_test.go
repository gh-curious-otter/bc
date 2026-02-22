package names

import (
	"testing"
)

func BenchmarkNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = New()
	}
}

func BenchmarkGenerate(b *testing.B) {
	g := New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = g.Generate()
	}
}

func BenchmarkGenerate_Default(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Generate()
	}
}

func BenchmarkGenerateUnique_Empty(b *testing.B) {
	g := New()
	existing := make(map[string]bool)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUnique(existing, 100)
	}
}

func BenchmarkGenerateUnique_Small(b *testing.B) {
	g := New()
	// Pre-populate with 10 names
	existing := make(map[string]bool, 10)
	for i := 0; i < 10; i++ {
		existing[g.Generate()] = true
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name, _ := g.GenerateUnique(existing, 100)
		// Don't actually add to map to keep consistent benchmark
		_ = name
	}
}

func BenchmarkGenerateUnique_Medium(b *testing.B) {
	g := New()
	// Pre-populate with 100 names
	existing := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		existing[g.Generate()] = true
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name, _ := g.GenerateUnique(existing, 100)
		_ = name
	}
}

func BenchmarkGenerateUnique_Large(b *testing.B) {
	g := New()
	// Pre-populate with 500 names (significant portion of 50*50=2500 combinations)
	existing := make(map[string]bool, 500)
	for i := 0; i < 500; i++ {
		existing[g.Generate()] = true
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name, _ := g.GenerateUnique(existing, 100)
		_ = name
	}
}

func BenchmarkGenerateUniqueFromList_Empty(b *testing.B) {
	g := New()
	var existingNames []string
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUniqueFromList(existingNames, 100)
	}
}

func BenchmarkGenerateUniqueFromList_Small(b *testing.B) {
	g := New()
	// Pre-populate with 10 names
	existingNames := make([]string, 10)
	for i := 0; i < 10; i++ {
		existingNames[i] = g.Generate()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUniqueFromList(existingNames, 100)
	}
}

func BenchmarkGenerateUniqueFromList_Medium(b *testing.B) {
	g := New()
	// Pre-populate with 100 names
	existingNames := make([]string, 100)
	for i := 0; i < 100; i++ {
		existingNames[i] = g.Generate()
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUniqueFromList(existingNames, 100)
	}
}

func BenchmarkGenerateUnique_Parallel(b *testing.B) {
	existing := make(map[string]bool)
	b.RunParallel(func(pb *testing.PB) {
		g := New() // Each goroutine gets its own generator
		for pb.Next() {
			_, _ = g.GenerateUnique(existing, 100)
		}
	})
}
