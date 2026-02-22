package queue

import (
	"context"
	"fmt"
	"testing"
)

// setupTestStore creates a temp store for benchmarking.
func setupTestStore(b *testing.B) *Store {
	b.Helper()
	tmpDir := b.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()
	if err := store.Open(ctx); err != nil {
		b.Fatalf("failed to open store: %v", err)
	}
	b.Cleanup(func() {
		store.Close() //nolint:errcheck
	})
	return store
}

// BenchmarkNewStore measures store creation.
func BenchmarkNewStore(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStore(tmpDir)
	}
}

// BenchmarkStoreOpen measures store initialization with schema.
func BenchmarkStoreOpen(b *testing.B) {
	tmpDir := b.TempDir()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store := NewStore(tmpDir)
		if err := store.Open(ctx); err != nil {
			b.Fatalf("failed to open store: %v", err)
		}
		store.Close() //nolint:errcheck
	}
}

// BenchmarkAddWork measures adding work items.
func BenchmarkAddWork(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:     "eng-01",
			Title:       fmt.Sprintf("Task %d", i),
			Description: "Benchmark task description",
			Status:      StatusPending,
			Priority:    PriorityNormal,
		}
		if err := store.AddWork(ctx, item); err != nil {
			b.Fatalf("failed to add work: %v", err)
		}
	}
}

// BenchmarkGetWork measures retrieving work items.
func BenchmarkGetWork(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	// Create a work item to retrieve
	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Test task",
		Description: "Test description",
		Status:      StatusPending,
		Priority:    PriorityNormal,
	}
	if err := store.AddWork(ctx, item); err != nil {
		b.Fatalf("failed to add work: %v", err)
	}
	id := item.ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetWork(ctx, id)
		if err != nil {
			b.Fatalf("failed to get work: %v", err)
		}
	}
}

// BenchmarkListWork measures listing work items.
func BenchmarkListWork(b *testing.B) {
	counts := []int{10, 50, 100}

	for _, count := range counts {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			store := setupTestStore(b)
			ctx := context.Background()

			// Create work items
			for i := 0; i < count; i++ {
				item := &WorkItem{
					AgentID:  "eng-01",
					Title:    fmt.Sprintf("Task %d", i),
					Status:   StatusPending,
					Priority: i % 4, // Vary priority
				}
				if err := store.AddWork(ctx, item); err != nil {
					b.Fatalf("failed to add work: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.ListWork(ctx, "eng-01", "")
				if err != nil {
					b.Fatalf("failed to list work: %v", err)
				}
			}
		})
	}
}

// BenchmarkAcceptWork measures accepting work items.
func BenchmarkAcceptWork(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		item := &WorkItem{
			AgentID:  "eng-01",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		if err := store.AddWork(ctx, item); err != nil {
			b.Fatalf("failed to add work: %v", err)
		}
		b.StartTimer()
		if err := store.AcceptWork(ctx, item.ID); err != nil {
			b.Fatalf("failed to accept work: %v", err)
		}
	}
}

// BenchmarkAddMerge measures adding merge items.
func BenchmarkAddMerge(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &MergeItem{
			AgentID:   "mgr-01",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("PR %d", i),
			Status:    MergeStatusPending,
			FromAgent: "eng-01",
		}
		if err := store.AddMerge(ctx, item); err != nil {
			b.Fatalf("failed to add merge: %v", err)
		}
	}
}

// BenchmarkGetMerge measures retrieving merge items.
func BenchmarkGetMerge(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	// Create a merge item to retrieve
	item := &MergeItem{
		AgentID:   "mgr-01",
		Branch:    "feature/test",
		Title:     "Test PR",
		Status:    MergeStatusPending,
		FromAgent: "eng-01",
	}
	if err := store.AddMerge(ctx, item); err != nil {
		b.Fatalf("failed to add merge: %v", err)
	}
	id := item.ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.GetMerge(ctx, id)
		if err != nil {
			b.Fatalf("failed to get merge: %v", err)
		}
	}
}

// BenchmarkListMerge measures listing merge items.
func BenchmarkListMerge(b *testing.B) {
	counts := []int{10, 50, 100}

	for _, count := range counts {
		b.Run(fmt.Sprintf("count=%d", count), func(b *testing.B) {
			store := setupTestStore(b)
			ctx := context.Background()

			// Create merge items
			for i := 0; i < count; i++ {
				item := &MergeItem{
					AgentID:   "mgr-01",
					Branch:    fmt.Sprintf("feature/task-%d", i),
					Title:     fmt.Sprintf("PR %d", i),
					Status:    MergeStatusPending,
					FromAgent: "eng-01",
				}
				if err := store.AddMerge(ctx, item); err != nil {
					b.Fatalf("failed to add merge: %v", err)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.ListMerge(ctx, "mgr-01", "")
				if err != nil {
					b.Fatalf("failed to list merge: %v", err)
				}
			}
		})
	}
}

// BenchmarkListWorkByStatus measures filtering work items by status.
func BenchmarkListWorkByStatus(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	// Create work items with different statuses
	statuses := []string{StatusPending, StatusAccepted, StatusCompleted}
	for i := range 50 {
		statusIdx := i % len(statuses)
		item := &WorkItem{
			AgentID:  "eng-01",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   statuses[statusIdx], //nolint:gosec // index is always in bounds
			Priority: PriorityNormal,
		}
		if err := store.AddWork(ctx, item); err != nil {
			b.Fatalf("failed to add work: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.ListWork(ctx, "eng-01", StatusPending)
		if err != nil {
			b.Fatalf("failed to list work: %v", err)
		}
	}
}

// BenchmarkListMergeByStatus measures filtering merge items by status.
func BenchmarkListMergeByStatus(b *testing.B) {
	store := setupTestStore(b)
	ctx := context.Background()

	// Create merge items with different statuses
	statuses := []string{MergeStatusPending, MergeStatusReviewed, MergeStatusMerged}
	for i := range 50 {
		statusIdx := i % len(statuses)
		item := &MergeItem{
			AgentID:   "mgr-01",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("PR %d", i),
			Status:    statuses[statusIdx], //nolint:gosec // index is always in bounds
			FromAgent: "eng-01",
		}
		if err := store.AddMerge(ctx, item); err != nil {
			b.Fatalf("failed to add merge: %v", err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.ListMerge(ctx, "mgr-01", MergeStatusPending)
		if err != nil {
			b.Fatalf("failed to list merge: %v", err)
		}
	}
}
