package names

import (
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	g := New()

	for i := 0; i < 100; i++ {
		name := g.Generate()

		// Check format: adjective-animal
		parts := strings.Split(name, "-")
		if len(parts) != 2 {
			t.Errorf("expected adjective-animal format, got %q", name)
		}

		// Check lowercase
		if name != strings.ToLower(name) {
			t.Errorf("expected lowercase, got %q", name)
		}

		// Check non-empty parts
		if parts[0] == "" || parts[1] == "" {
			t.Errorf("expected non-empty parts, got %q", name)
		}
	}
}

func TestGenerateUnique(t *testing.T) {
	g := New()

	existing := map[string]bool{
		"swift-falcon": true,
		"clever-otter": true,
	}

	name, err := g.GenerateUnique(existing, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if existing[name] {
		t.Errorf("generated name %q already exists", name)
	}
}

func TestGenerateUniqueFromList(t *testing.T) {
	g := New()

	existingList := []string{"swift-falcon", "clever-otter", "BOLD-EAGLE"}

	name, err := g.GenerateUniqueFromList(existingList, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check not in list (case-insensitive)
	nameLower := strings.ToLower(name)
	for _, existing := range existingList {
		if strings.ToLower(existing) == nameLower {
			t.Errorf("generated name %q already exists in list", name)
		}
	}
}

func TestGenerateUniqueFallback(t *testing.T) {
	g := New()

	// Create a map with all possible combinations
	existing := make(map[string]bool)
	for _, adj := range adjectives {
		for _, animal := range animals {
			existing[adj+"-"+animal] = true
		}
	}

	// Should fall back to numeric suffix
	name, err := g.GenerateUnique(existing, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have numeric suffix
	parts := strings.Split(name, "-")
	if len(parts) != 3 {
		t.Errorf("expected numeric suffix, got %q", name)
	}
}

func TestDefaultGenerator(t *testing.T) {
	// Test package-level functions
	name := Generate()
	if name == "" {
		t.Error("expected non-empty name")
	}

	existing := map[string]bool{name: true}
	unique, err := GenerateUnique(existing, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unique == name {
		t.Error("expected different name")
	}
}

func TestPackageLevelGenerateUniqueFromList(t *testing.T) {
	existingList := []string{"swift-falcon", "clever-otter"}

	name, err := GenerateUniqueFromList(existingList, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check not in list (case-insensitive)
	nameLower := strings.ToLower(name)
	for _, existing := range existingList {
		if strings.ToLower(existing) == nameLower {
			t.Errorf("generated name %q already exists in list", name)
		}
	}
}

func TestWordListSizes(t *testing.T) {
	// Ensure we have enough variety
	if len(adjectives) < 50 {
		t.Errorf("expected at least 50 adjectives, got %d", len(adjectives))
	}
	if len(animals) < 50 {
		t.Errorf("expected at least 50 animals, got %d", len(animals))
	}

	// Check for duplicates in adjectives
	adjSet := make(map[string]bool)
	for _, adj := range adjectives {
		if adjSet[adj] {
			t.Errorf("duplicate adjective: %s", adj)
		}
		adjSet[adj] = true
	}

	// Check for duplicates in animals
	animalSet := make(map[string]bool)
	for _, animal := range animals {
		if animalSet[animal] {
			t.Errorf("duplicate animal: %s", animal)
		}
		animalSet[animal] = true
	}
}

// Benchmarks

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

func BenchmarkGenerateParallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		g := New() // Each goroutine gets its own generator
		for pb.Next() {
			_ = g.Generate()
		}
	})
}

func BenchmarkGenerateUnique_SmallSet(b *testing.B) {
	g := New()
	existing := map[string]bool{
		"swift-falcon": true,
		"clever-otter": true,
		"bold-eagle":   true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUnique(existing, 100)
	}
}

func BenchmarkGenerateUnique_LargeSet(b *testing.B) {
	g := New()
	// Create a large existing set (90% of possible combinations)
	existing := make(map[string]bool)
	count := 0
	target := len(adjectives) * len(animals) * 9 / 10
	for _, adj := range adjectives {
		for _, animal := range animals {
			if count >= target {
				break
			}
			existing[adj+"-"+animal] = true
			count++
		}
		if count >= target {
			break
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUnique(existing, 100)
	}
}

func BenchmarkGenerateUniqueFromList(b *testing.B) {
	g := New()
	existingList := []string{
		"swift-falcon", "clever-otter", "bold-eagle",
		"keen-tiger", "calm-panda", "eager-dolphin",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateUniqueFromList(existingList, 100)
	}
}

func BenchmarkDefaultGenerate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Generate()
	}
}
