package queue

import (
	"context"
	"fmt"
	"testing"
)

func setupBenchmarkStore(b *testing.B) *Store {
	b.Helper()
	dir := b.TempDir()
	store := NewStore(dir)
	if err := store.Open(context.Background()); err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func BenchmarkNewStore(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStore(dir)
	}
}

func BenchmarkStore_Open(b *testing.B) {
	dir := b.TempDir()
	for i := 0; i < b.N; i++ {
		store := NewStore(dir)
		err := store.Open(context.Background())
		if err != nil {
			b.Fatal(err)
		}
		_ = store.Close()
	}
}

func BenchmarkStore_AddWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:     "test-agent",
			Title:       fmt.Sprintf("Task %d", i),
			Description: "Benchmark task",
			Status:      StatusPending,
			Priority:    PriorityNormal,
			FromAgent:   "manager",
			IssueRef:    "#123",
		}
		_ = store.AddWork(ctx, item)
	}
}

func BenchmarkStore_GetWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add an item to retrieve
	item := &WorkItem{
		AgentID:     "test-agent",
		Title:       "Test task",
		Description: "Benchmark task",
		Status:      StatusPending,
		Priority:    PriorityNormal,
	}
	if err := store.AddWork(ctx, item); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetWork(ctx, item.ID)
	}
}

func BenchmarkStore_ListWork_Empty(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListWork(ctx, "test-agent", "")
	}
}

func BenchmarkStore_ListWork_WithItems(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add multiple items
	for i := 0; i < 100; i++ {
		item := &WorkItem{
			AgentID:  "test-agent",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: i % 4,
		}
		_ = store.AddWork(ctx, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListWork(ctx, "test-agent", "")
	}
}

func BenchmarkStore_ListWork_WithStatus(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add items with various statuses
	for i := 0; i < 100; i++ {
		status := StatusPending
		if i%3 == 0 {
			status = StatusInProgress
		}
		item := &WorkItem{
			AgentID:  "test-agent",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   status,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListWork(ctx, "test-agent", StatusPending)
	}
}

func BenchmarkStore_AcceptWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create items to accept
	items := make([]*WorkItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:  "test-agent",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.AcceptWork(ctx, items[i].ID)
	}
}

func BenchmarkStore_StartWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create and accept items
	items := make([]*WorkItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:  "test-agent",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
		_ = store.AcceptWork(ctx, item.ID)
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.StartWork(ctx, items[i].ID)
	}
}

func BenchmarkStore_CompleteWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create, accept, and start items
	items := make([]*WorkItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:  "test-agent",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
		_ = store.AcceptWork(ctx, item.ID)
		_ = store.StartWork(ctx, item.ID)
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.CompleteWork(ctx, items[i].ID, fmt.Sprintf("feature/task-%d", i))
	}
}

func BenchmarkStore_AddMerge(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &MergeItem{
			AgentID:   "manager",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("Merge %d", i),
			Status:    MergeStatusPending,
			FromAgent: "engineer",
			IssueRef:  "#123",
		}
		_ = store.AddMerge(ctx, item)
	}
}

func BenchmarkStore_GetMerge(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add an item to retrieve
	item := &MergeItem{
		AgentID:   "manager",
		Branch:    "feature/test",
		Title:     "Test merge",
		Status:    MergeStatusPending,
		FromAgent: "engineer",
	}
	if err := store.AddMerge(ctx, item); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetMerge(ctx, item.ID)
	}
}

func BenchmarkStore_GetMergeByBranch(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add an item to retrieve
	item := &MergeItem{
		AgentID:   "manager",
		Branch:    "feature/test",
		Title:     "Test merge",
		Status:    MergeStatusPending,
		FromAgent: "engineer",
	}
	if err := store.AddMerge(ctx, item); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetMergeByBranch(ctx, "manager", "feature/test")
	}
}

func BenchmarkStore_ListMerge_Empty(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListMerge(ctx, "manager", "")
	}
}

func BenchmarkStore_ListMerge_WithItems(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add multiple items
	for i := 0; i < 50; i++ {
		item := &MergeItem{
			AgentID:   "manager",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("Merge %d", i),
			Status:    MergeStatusPending,
			FromAgent: "engineer",
		}
		_ = store.AddMerge(ctx, item)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.ListMerge(ctx, "manager", "")
	}
}

func BenchmarkStore_ApproveMerge(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create items to approve
	items := make([]*MergeItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &MergeItem{
			AgentID:   "manager",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("Merge %d", i),
			Status:    MergeStatusPending,
			FromAgent: "engineer",
		}
		_ = store.AddMerge(ctx, item)
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.ApproveMerge(ctx, items[i].ID, "reviewer")
	}
}

func BenchmarkStore_RejectMerge(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create items to reject
	items := make([]*MergeItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &MergeItem{
			AgentID:   "manager",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("Merge %d", i),
			Status:    MergeStatusPending,
			FromAgent: "engineer",
		}
		_ = store.AddMerge(ctx, item)
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RejectMerge(ctx, items[i].ID, "reviewer", "needs changes")
	}
}

func BenchmarkStore_CompleteMerge(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create and approve items
	items := make([]*MergeItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &MergeItem{
			AgentID:   "manager",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("Merge %d", i),
			Status:    MergeStatusPending,
			FromAgent: "engineer",
		}
		_ = store.AddMerge(ctx, item)
		_ = store.ApproveMerge(ctx, item.ID, "reviewer")
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.CompleteMerge(ctx, items[i].ID)
	}
}

func BenchmarkStore_Submit(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Pre-create completed work items
	items := make([]*WorkItem, b.N)
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:  "engineer",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
		_ = store.AcceptWork(ctx, item.ID)
		_ = store.StartWork(ctx, item.ID)
		_ = store.CompleteWork(ctx, item.ID, fmt.Sprintf("feature/task-%d", i))
		items[i] = item
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Submit(ctx, items[i].ID, "manager")
	}
}

func BenchmarkStore_WorkLifecycle(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &WorkItem{
			AgentID:  "engineer",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
		_ = store.AcceptWork(ctx, item.ID)
		_ = store.StartWork(ctx, item.ID)
		_ = store.CompleteWork(ctx, item.ID, fmt.Sprintf("feature/task-%d", i))
	}
}

func BenchmarkStore_MergeLifecycle(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		item := &MergeItem{
			AgentID:   "manager",
			Branch:    fmt.Sprintf("feature/task-%d", i),
			Title:     fmt.Sprintf("Merge %d", i),
			Status:    MergeStatusPending,
			FromAgent: "engineer",
		}
		_ = store.AddMerge(ctx, item)
		_ = store.ApproveMerge(ctx, item.ID, "reviewer")
		_ = store.CompleteMerge(ctx, item.ID)
	}
}

func BenchmarkStore_Parallel_AddWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			item := &WorkItem{
				AgentID:  "test-agent",
				Title:    fmt.Sprintf("Parallel task %d", i),
				Status:   StatusPending,
				Priority: PriorityNormal,
			}
			_ = store.AddWork(ctx, item)
			i++
		}
	})
}

func BenchmarkStore_Parallel_GetWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add an item to retrieve
	item := &WorkItem{
		AgentID:  "test-agent",
		Title:    "Test task",
		Status:   StatusPending,
		Priority: PriorityNormal,
	}
	if err := store.AddWork(ctx, item); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.GetWork(ctx, item.ID)
		}
	})
}

func BenchmarkStore_Parallel_ListWork(b *testing.B) {
	store := setupBenchmarkStore(b)
	ctx := context.Background()

	// Add some items
	for i := 0; i < 20; i++ {
		item := &WorkItem{
			AgentID:  "test-agent",
			Title:    fmt.Sprintf("Task %d", i),
			Status:   StatusPending,
			Priority: PriorityNormal,
		}
		_ = store.AddWork(ctx, item)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = store.ListWork(ctx, "test-agent", "")
		}
	})
}
