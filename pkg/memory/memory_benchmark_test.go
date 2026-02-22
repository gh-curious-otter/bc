package memory

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkRecordExperience measures the performance of recording experiences.
func BenchmarkRecordExperience(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir, "benchmark-agent")
	if err := store.Init(); err != nil {
		b.Fatalf("failed to init store: %v", err)
	}

	exp := Experience{
		Timestamp:   time.Now().UTC(),
		Description: "Completed benchmark task",
		Outcome:     "success",
		TaskID:      "task-001",
		TaskType:    "benchmark",
		Learnings:   []string{"learned something"},
		Metadata:    map[string]any{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RecordExperience(exp)
	}
}

// BenchmarkRecordExperienceMinimal measures recording with minimal data.
func BenchmarkRecordExperienceMinimal(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir, "benchmark-agent")
	if err := store.Init(); err != nil {
		b.Fatalf("failed to init store: %v", err)
	}

	exp := Experience{
		Description: "Task done",
		Outcome:     "success",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RecordExperience(exp)
	}
}

// BenchmarkGetExperiences measures reading experiences.
func BenchmarkGetExperiences(b *testing.B) {
	counts := []int{10, 100, 500}

	for _, count := range counts {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			tmpDir := b.TempDir()
			store := NewStore(tmpDir, "benchmark-agent")
			if err := store.Init(); err != nil {
				b.Fatalf("failed to init store: %v", err)
			}

			// Seed with experiences
			for i := 0; i < count; i++ {
				exp := Experience{
					Description: fmt.Sprintf("Experience %d", i),
					Outcome:     "success",
					TaskID:      fmt.Sprintf("task-%d", i),
				}
				if err := store.RecordExperience(exp); err != nil {
					b.Fatalf("failed to record experience: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = store.GetExperiences()
			}
		})
	}
}

// BenchmarkAddLearning measures adding learnings.
func BenchmarkAddLearning(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir, "benchmark-agent")
	if err := store.Init(); err != nil {
		b.Fatalf("failed to init store: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Each iteration adds to a different category to avoid string manipulation overhead
		category := fmt.Sprintf("Category-%d", i%10)
		_ = store.AddLearning(category, "A learning about something important")
	}
}

// BenchmarkAddLearningExistingCategory measures adding to an existing category.
func BenchmarkAddLearningExistingCategory(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir, "benchmark-agent")
	if err := store.Init(); err != nil {
		b.Fatalf("failed to init store: %v", err)
	}

	// Pre-create the category
	if err := store.AddLearning("TestCategory", "Initial learning"); err != nil {
		b.Fatalf("failed to add initial learning: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.AddLearning("TestCategory", "Another learning")
	}
}

// BenchmarkGetLearnings measures reading learnings.
func BenchmarkGetLearnings(b *testing.B) {
	sizes := []int{10, 50, 100}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("learnings=%d", size), func(b *testing.B) {
			tmpDir := b.TempDir()
			store := NewStore(tmpDir, "benchmark-agent")
			if err := store.Init(); err != nil {
				b.Fatalf("failed to init store: %v", err)
			}

			// Seed with learnings
			for i := 0; i < size; i++ {
				category := fmt.Sprintf("Category-%d", i%5)
				if err := store.AddLearning(category, fmt.Sprintf("Learning %d", i)); err != nil {
					b.Fatalf("failed to add learning: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = store.GetLearnings()
			}
		})
	}
}

// BenchmarkStoreInit measures store initialization.
func BenchmarkStoreInit(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := NewStore(tmpDir, fmt.Sprintf("agent-%d", i))
		_ = store.Init()
	}
}

// BenchmarkStoreExists measures existence check.
func BenchmarkStoreExists(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir, "benchmark-agent")
	if err := store.Init(); err != nil {
		b.Fatalf("failed to init store: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Exists()
	}
}

// BenchmarkClear measures clearing memory.
func BenchmarkClear(b *testing.B) {
	tmpDir := b.TempDir()
	store := NewStore(tmpDir, "benchmark-agent")
	if err := store.Init(); err != nil {
		b.Fatalf("failed to init store: %v", err)
	}

	// Seed with data
	for i := 0; i < 100; i++ {
		exp := Experience{
			Description: fmt.Sprintf("Experience %d", i),
			Outcome:     "success",
		}
		_ = store.RecordExperience(exp)
		_ = store.AddLearning("Test", fmt.Sprintf("Learning %d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Clear(true, true)
		// Re-init for next iteration
		_ = store.Init()
	}
}

// BenchmarkNewStore measures store creation (memory only, no disk).
func BenchmarkNewStore(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStore(tmpDir, "benchmark-agent")
	}
}
