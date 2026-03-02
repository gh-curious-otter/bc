package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStore_Init(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Check directory exists
	if !store.Exists() {
		t.Error("memory directory should exist after Init")
	}

	// Check experiences.jsonl exists
	experiencesPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	if _, err := os.Stat(experiencesPath); os.IsNotExist(err) {
		t.Error("experiences.jsonl should exist after Init")
	}

	// Check learnings.md exists
	learningsPath := filepath.Join(store.MemoryDir(), "learnings.md")
	if _, err := os.Stat(learningsPath); os.IsNotExist(err) {
		t.Error("learnings.md should exist after Init")
	}

	// Check learnings.md has initial content
	content, err := os.ReadFile(learningsPath) //nolint:gosec // test file path
	if err != nil {
		t.Fatalf("failed to read learnings.md: %v", err)
	}
	if len(content) == 0 {
		t.Error("learnings.md should have initial content")
	}
}

func TestStore_InitIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	// Init twice should not fail
	if err := store.Init(); err != nil {
		t.Fatalf("first init failed: %v", err)
	}
	if err := store.Init(); err != nil {
		t.Fatalf("second init failed: %v", err)
	}
}

func TestStore_RecordExperience(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	exp := Experience{
		TaskID:      "work-123",
		TaskType:    "code",
		Description: "Implemented feature X",
		Outcome:     "success",
		Learnings:   []string{"Use context for cancellation"},
	}

	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	// Read back
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}

	if experiences[0].TaskID != "work-123" {
		t.Errorf("expected task ID 'work-123', got %q", experiences[0].TaskID)
	}
	if experiences[0].Outcome != "success" {
		t.Errorf("expected outcome 'success', got %q", experiences[0].Outcome)
	}
}

func TestStore_RecordMultipleExperiences(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	for i := 0; i < 3; i++ {
		exp := Experience{
			TaskID:      "task-" + string(rune('A'+i)),
			Description: "Task description",
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record experience %d: %v", i, err)
		}
	}

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 3 {
		t.Errorf("expected 3 experiences, got %d", len(experiences))
	}
}

func TestStore_AddLearning(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	if err := store.AddLearning("Testing", "Always write tests first"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	content, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}

	if !contains(content, "Testing") {
		t.Error("learnings should contain 'Testing'")
	}
	if !contains(content, "Always write tests first") {
		t.Error("learnings should contain the learning text")
	}
}

func TestStore_AddLearning_NoDuplicateCategories(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add first learning under "patterns"
	if err := store.AddLearning("patterns", "Use context for cancellation"); err != nil {
		t.Fatalf("failed to add first learning: %v", err)
	}

	// Add second learning under same category
	if err := store.AddLearning("patterns", "Prefer composition over inheritance"); err != nil {
		t.Fatalf("failed to add second learning: %v", err)
	}

	// Add third learning under different category
	if err := store.AddLearning("testing", "Always write tests first"); err != nil {
		t.Fatalf("failed to add third learning: %v", err)
	}

	content, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}

	// Count occurrences of "## patterns" - should be exactly 1
	patternCount := countOccurrences(content, "## patterns")
	if patternCount != 1 {
		t.Errorf("expected exactly 1 '## patterns' header, got %d", patternCount)
	}

	// Both learnings should be present
	if !contains(content, "Use context for cancellation") {
		t.Error("learnings should contain first pattern learning")
	}
	if !contains(content, "Prefer composition over inheritance") {
		t.Error("learnings should contain second pattern learning")
	}
	if !contains(content, "Always write tests first") {
		t.Error("learnings should contain testing learning")
	}
}

func countOccurrences(s, substr string) int {
	count := 0
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			count++
		}
	}
	return count
}

func TestStore_ExperienceTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	before := time.Now().UTC().Add(-time.Second)

	exp := Experience{
		Description: "Test task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	after := time.Now().UTC().Add(time.Second)

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}

	ts := experiences[0].Timestamp
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v should be between %v and %v", ts, before, after)
	}
}

func TestStore_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if store.Exists() {
		t.Error("store should not exist before Init")
	}

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	if !store.Exists() {
		t.Error("store should exist after Init")
	}
}

func TestStore_GetExperiencesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 0 {
		t.Errorf("expected 0 experiences, got %d", len(experiences))
	}
}

func TestStore_GetExperiencesNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "nonexistent")

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("should not error for nonexistent: %v", err)
	}

	if experiences != nil {
		t.Errorf("expected nil experiences, got %v", experiences)
	}
}

func TestStore_GetLearningsNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "nonexistent")

	learnings, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("should not error for nonexistent: %v", err)
	}

	if learnings != "" {
		t.Errorf("expected empty learnings, got %q", learnings)
	}
}

func TestStore_ExperienceWithTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Record experience with explicit timestamp
	customTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	exp := Experience{
		Timestamp:   customTime,
		Description: "Test with timestamp",
		Outcome:     "success",
	}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}

	// Should preserve the custom timestamp
	if !experiences[0].Timestamp.Equal(customTime) {
		t.Errorf("expected timestamp %v, got %v", customTime, experiences[0].Timestamp)
	}
}

func TestStore_MemoryDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	expected := filepath.Join(tmpDir, ".bc", "memory", "test-agent")
	if store.MemoryDir() != expected {
		t.Errorf("MemoryDir() = %q, want %q", store.MemoryDir(), expected)
	}
}

func TestStore_ExperienceWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	exp := Experience{
		TaskID:      "work-456",
		TaskType:    "review",
		Description: "Reviewed PR #123",
		Outcome:     "success",
		Metadata: map[string]any{
			"pr_number": 123,
			"files":     []string{"main.go", "util.go"},
		},
	}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}

	if experiences[0].Metadata == nil {
		t.Error("expected metadata to be preserved")
	}
}

func TestStore_MultipleAgents(t *testing.T) {
	tmpDir := t.TempDir()

	// Create stores for different agents
	store1 := NewStore(tmpDir, "engineer-01")
	store2 := NewStore(tmpDir, "engineer-02")

	if err := store1.Init(); err != nil {
		t.Fatalf("failed to init store1: %v", err)
	}
	if err := store2.Init(); err != nil {
		t.Fatalf("failed to init store2: %v", err)
	}

	// Record different experiences
	if err := store1.RecordExperience(Experience{Description: "Task for eng1", Outcome: "success"}); err != nil {
		t.Fatalf("failed to record for store1: %v", err)
	}
	if err := store2.RecordExperience(Experience{Description: "Task for eng2", Outcome: "success"}); err != nil {
		t.Fatalf("failed to record for store2: %v", err)
	}

	// Verify isolation
	exp1, _ := store1.GetExperiences()
	exp2, _ := store2.GetExperiences()

	if len(exp1) != 1 || exp1[0].Description != "Task for eng1" {
		t.Error("store1 should only have eng1's experience")
	}
	if len(exp2) != 1 || exp2[0].Description != "Task for eng2" {
		t.Error("store2 should only have eng2's experience")
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"two lines", "hello\nworld", 2},
		{"trailing newline", "hello\n", 1},
		{"multiple newlines", "a\nb\nc\n", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines := splitLines([]byte(tt.input))
			if len(lines) != tt.want {
				t.Errorf("splitLines(%q) got %d lines, want %d", tt.input, len(lines), tt.want)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStore_GetMemoryContext_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// New agent with no memories should return empty string
	ctx, err := store.GetMemoryContext(DefaultMemoryLimit)
	if err != nil {
		t.Fatalf("GetMemoryContext failed: %v", err)
	}

	if ctx != "" {
		t.Errorf("expected empty context for new agent, got %q", ctx)
	}
}

func TestStore_GetMemoryContext_WithExperiences(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add some experiences
	for i := 0; i < 3; i++ {
		exp := Experience{
			TaskType:    "code",
			Description: "Implemented feature " + string(rune('A'+i)),
			Outcome:     "success",
			Learnings:   []string{"Learning " + string(rune('A'+i))},
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record experience: %v", err)
		}
	}

	ctx, err := store.GetMemoryContext(DefaultMemoryLimit)
	if err != nil {
		t.Fatalf("GetMemoryContext failed: %v", err)
	}

	// Should contain header
	if !contains(ctx, "# Agent Memory") {
		t.Error("context should contain header")
	}

	// Should contain experiences section
	if !contains(ctx, "## Recent Experiences") {
		t.Error("context should contain experiences section")
	}

	// Should contain experience content
	if !contains(ctx, "Implemented feature A") {
		t.Error("context should contain experience description")
	}
}

func TestStore_GetMemoryContext_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add 15 experiences
	for i := 0; i < 15; i++ {
		exp := Experience{
			TaskType:    "code",
			Description: "Task " + string(rune('A'+i)),
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record experience: %v", err)
		}
	}

	// With limit of 5, should only include last 5 experiences
	ctx, err := store.GetMemoryContext(5)
	if err != nil {
		t.Fatalf("GetMemoryContext failed: %v", err)
	}

	// Should contain later experiences (K, L, M, N, O)
	if !contains(ctx, "Task O") {
		t.Error("context should contain most recent task (O)")
	}

	// Should NOT contain early experiences (A, B, C)
	if contains(ctx, "Task A") {
		t.Error("context should NOT contain old task (A) when limited")
	}
}

func TestStore_GetMemoryContext_WithLearnings(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add substantial learnings
	for i := 0; i < 5; i++ {
		if err := store.AddLearning("Category "+string(rune('A'+i)), "Important learning about topic "+string(rune('A'+i))); err != nil {
			t.Fatalf("failed to add learning: %v", err)
		}
	}

	ctx, err := store.GetMemoryContext(DefaultMemoryLimit)
	if err != nil {
		t.Fatalf("GetMemoryContext failed: %v", err)
	}

	// Should contain learnings section
	if !contains(ctx, "## Key Learnings") {
		t.Error("context should contain learnings section")
	}
}

func TestStore_GetMemoryContext_DefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add an experience
	exp := Experience{
		TaskType:    "code",
		Description: "Test task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	// Passing 0 should use default limit
	ctx, err := store.GetMemoryContext(0)
	if err != nil {
		t.Fatalf("GetMemoryContext failed: %v", err)
	}

	if !contains(ctx, "Test task") {
		t.Error("context should contain the experience with default limit")
	}
}

func TestStore_GetMemoryContext_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "nonexistent-agent")

	// Store doesn't exist - should return empty without error
	ctx, err := store.GetMemoryContext(DefaultMemoryLimit)
	if err != nil {
		t.Fatalf("GetMemoryContext should not error for nonexistent store: %v", err)
	}

	if ctx != "" {
		t.Errorf("expected empty context for nonexistent agent, got %q", ctx)
	}
}

func TestStore_Prune_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add old experience (45 days ago)
	oldExp := Experience{
		Timestamp:   time.Now().Add(-45 * 24 * time.Hour),
		Description: "Old task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old experience: %v", err)
	}

	// Add recent experience
	recentExp := Experience{
		Timestamp:   time.Now().Add(-5 * 24 * time.Hour),
		Description: "Recent task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(recentExp); err != nil {
		t.Fatalf("failed to record recent experience: %v", err)
	}

	// Prune with dry run (30 day threshold)
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.TotalExperiences != 2 {
		t.Errorf("expected 2 total experiences, got %d", result.TotalExperiences)
	}
	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned experience, got %d", result.PrunedExperiences)
	}

	// Verify nothing was actually deleted (dry run)
	experiences, _ := store.GetExperiences()
	if len(experiences) != 2 {
		t.Errorf("dry run should not delete, but got %d experiences", len(experiences))
	}
}

func TestStore_Prune_ActualDeletion(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add old experience (45 days ago)
	oldExp := Experience{
		Timestamp:   time.Now().Add(-45 * 24 * time.Hour),
		Description: "Old task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old experience: %v", err)
	}

	// Add recent experience
	recentExp := Experience{
		Timestamp:   time.Now().Add(-5 * 24 * time.Hour),
		Description: "Recent task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(recentExp); err != nil {
		t.Fatalf("failed to record recent experience: %v", err)
	}

	// Prune without dry run (30 day threshold, no backup)
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
		Backup:    false,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned experience, got %d", result.PrunedExperiences)
	}

	// Verify old experience was deleted
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience after prune, got %d", len(experiences))
	}
	if experiences[0].Description != "Recent task" {
		t.Errorf("wrong experience kept: %s", experiences[0].Description)
	}
}

func TestStore_Prune_PreservesPinned(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add old pinned experience (should be preserved)
	pinnedExp := Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "Important pinned task",
		Outcome:     "success",
		Pinned:      true,
	}
	if err := store.RecordExperience(pinnedExp); err != nil {
		t.Fatalf("failed to record pinned experience: %v", err)
	}

	// Add old non-pinned experience (should be pruned)
	oldExp := Experience{
		Timestamp:   time.Now().Add(-45 * 24 * time.Hour),
		Description: "Old non-pinned task",
		Outcome:     "success",
		Pinned:      false,
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old experience: %v", err)
	}

	// Prune
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
		Backup:    false,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.PreservedPinned != 1 {
		t.Errorf("expected 1 preserved pinned, got %d", result.PreservedPinned)
	}
	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned, got %d", result.PrunedExperiences)
	}

	// Verify pinned was kept
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience after prune, got %d", len(experiences))
	}
	if !experiences[0].Pinned {
		t.Error("remaining experience should be pinned")
	}
}

func TestStore_Prune_WithBackup(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add old experience
	oldExp := Experience{
		Timestamp:   time.Now().Add(-45 * 24 * time.Hour),
		Description: "Old task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old experience: %v", err)
	}

	// Prune with backup
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
		Backup:    true,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.BackupPath == "" {
		t.Error("expected backup path to be set")
	}

	// Verify backup file exists
	if _, err := os.Stat(result.BackupPath); os.IsNotExist(err) {
		t.Error("backup file should exist")
	}
}

func TestStore_Prune_NothingToPrune(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add only recent experience
	recentExp := Experience{
		Timestamp:   time.Now().Add(-5 * 24 * time.Hour),
		Description: "Recent task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(recentExp); err != nil {
		t.Fatalf("failed to record recent experience: %v", err)
	}

	// Prune
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
		Backup:    true,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.PrunedExperiences != 0 {
		t.Errorf("expected 0 pruned, got %d", result.PrunedExperiences)
	}
	if result.BackupPath != "" {
		t.Error("backup should not be created when nothing to prune")
	}
}

func TestStore_GetSize(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add some content
	for i := 0; i < 10; i++ {
		exp := Experience{
			Description: "Task with some description content " + string(rune('A'+i)),
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record experience: %v", err)
		}
	}

	size, err := store.GetSize()
	if err != nil {
		t.Fatalf("GetSize failed: %v", err)
	}

	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}
}

func TestStore_NeedsPruning(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add some content
	exp := Experience{
		Description: "A task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	// With very low threshold, should need pruning
	needs, size, err := store.NeedsPruning(1) // 1 byte threshold
	if err != nil {
		t.Fatalf("NeedsPruning failed: %v", err)
	}
	if !needs {
		t.Error("should need pruning with 1 byte threshold")
	}
	if size <= 0 {
		t.Errorf("expected positive size, got %d", size)
	}

	// With high threshold, should not need pruning
	needs, _, err = store.NeedsPruning(1024 * 1024) // 1MB threshold
	if err != nil {
		t.Fatalf("NeedsPruning failed: %v", err)
	}
	if needs {
		t.Error("should not need pruning with 1MB threshold")
	}
}

func TestStore_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add some experiences
	for i := 0; i < 3; i++ {
		exp := Experience{Description: "Test experience"}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record experience: %v", err)
		}
	}

	// Add a learning
	if err := store.AddLearning("patterns", "Test learning"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Verify data exists
	experiences, _ := store.GetExperiences()
	if len(experiences) != 3 {
		t.Errorf("expected 3 experiences, got %d", len(experiences))
	}

	// Clear only experiences
	result, err := store.Clear(true, false)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if result.ExperiencesCleared != 3 {
		t.Errorf("expected 3 experiences cleared, got %d", result.ExperiencesCleared)
	}
	if result.LearningsCleared {
		t.Error("learnings should not be cleared")
	}

	// Verify experiences are cleared
	experiences, _ = store.GetExperiences()
	if len(experiences) != 0 {
		t.Errorf("expected 0 experiences after clear, got %d", len(experiences))
	}

	// Learnings should still exist
	learnings, _ := store.GetLearnings()
	if learnings == "" {
		t.Error("learnings should still exist after clearing only experiences")
	}
}

func TestStore_ClearBoth(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add experience and learning
	if err := store.RecordExperience(Experience{Description: "Test"}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}
	if err := store.AddLearning("tips", "Test tip"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Clear both
	result, err := store.Clear(true, true)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}
	if result.ExperiencesCleared != 1 {
		t.Errorf("expected 1 experience cleared, got %d", result.ExperiencesCleared)
	}
	if !result.LearningsCleared {
		t.Error("learnings should be cleared")
	}

	// Verify both cleared
	experiences, _ := store.GetExperiences()
	if len(experiences) != 0 {
		t.Errorf("expected 0 experiences, got %d", len(experiences))
	}

	learnings, _ := store.GetLearnings()
	// Learnings file should still have the header
	if learnings == "" {
		t.Error("learnings should have header after clear")
	}
}

func TestStore_ListTopics(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add learnings in different topics
	if err := store.AddLearning("patterns", "Use context"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := store.AddLearning("tips", "Check errors"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := store.AddLearning("gotchas", "Watch for nil"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	topics, err := store.ListTopics()
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}

	if len(topics) != 3 {
		t.Errorf("expected 3 topics, got %d", len(topics))
	}

	// Check topics exist
	topicMap := make(map[string]bool)
	for _, topic := range topics {
		topicMap[topic] = true
	}
	if !topicMap["patterns"] || !topicMap["tips"] || !topicMap["gotchas"] {
		t.Errorf("expected patterns, tips, gotchas; got %v", topics)
	}
}

func TestStore_ForgetTopic(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Add learnings in different topics
	if err := store.AddLearning("patterns", "Use context"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := store.AddLearning("patterns", "Use interfaces"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := store.AddLearning("tips", "Check errors"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Forget patterns topic
	removed, err := store.ForgetTopic("patterns")
	if err != nil {
		t.Fatalf("ForgetTopic failed: %v", err)
	}
	if removed != 2 {
		t.Errorf("expected 2 entries removed, got %d", removed)
	}

	// Verify patterns is gone
	topics, _ := store.ListTopics()
	for _, topic := range topics {
		if topic == "patterns" {
			t.Error("patterns topic should be removed")
		}
	}

	// Verify tips is still there
	learnings, _ := store.GetLearnings()
	if !strings.Contains(learnings, "## tips") {
		t.Error("tips topic should still exist")
	}
	if !strings.Contains(learnings, "Check errors") {
		t.Error("tips learning should still exist")
	}
}

func TestStore_ForgetTopic_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "engineer-01")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_, err := store.ForgetTopic("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent topic")
	}
}

// Additional tests for edge cases and error paths

func TestStore_RecordExperience_NoInit(t *testing.T) {
	// Test recording to uninitialized store (directory doesn't exist)
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")
	// Note: RecordExperience does NOT create the directory, only Init does

	exp := Experience{
		Description: "Test experience",
		Outcome:     "success",
	}

	// This should fail because the directory doesn't exist
	err := store.RecordExperience(exp)
	if err == nil {
		t.Fatal("expected error when recording to uninitialized store")
	}
	if !strings.Contains(err.Error(), "no such file or directory") {
		t.Errorf("expected 'no such file or directory' error, got: %v", err)
	}
}

func TestStore_GetExperiences_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Write malformed JSON directly
	expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	content := `{"description":"valid","outcome":"success"}
this is not valid json
{"description":"another valid","outcome":"success"}
`
	if err := os.WriteFile(expPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	// Should return valid experiences, skip malformed
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 valid experiences (skipping malformed line)
	if len(experiences) != 2 {
		t.Errorf("expected 2 valid experiences, got %d", len(experiences))
	}
}

func TestStore_GetExperiences_EmptyLines(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Write file with empty lines
	expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	content := `{"description":"first","outcome":"success"}

{"description":"second","outcome":"success"}

`
	if err := os.WriteFile(expPath, []byte(content), 0600); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(experiences) != 2 {
		t.Errorf("expected 2 experiences, got %d", len(experiences))
	}
}

func TestStore_GetLearnings_NotExist(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "nonexistent-agent")

	// Don't init - file doesn't exist
	content, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("unexpected error for nonexistent file: %v", err)
	}
	if content != "" {
		t.Errorf("expected empty string for nonexistent learnings, got %q", content)
	}
}

func TestStore_AddLearning_ToExistingCategory(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add first learning
	if err := store.AddLearning("patterns", "Use context"); err != nil {
		t.Fatalf("failed to add first learning: %v", err)
	}

	// Add second learning to same category
	if err := store.AddLearning("patterns", "Use interfaces"); err != nil {
		t.Fatalf("failed to add second learning: %v", err)
	}

	learnings, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}

	// Both learnings should be under the same ## patterns section
	if strings.Count(learnings, "## patterns") != 1 {
		t.Error("expected exactly one patterns section")
	}
	if !strings.Contains(learnings, "- Use context") {
		t.Error("first learning missing")
	}
	if !strings.Contains(learnings, "- Use interfaces") {
		t.Error("second learning missing")
	}
}

func TestStore_Clear_ExperiencesOnly(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add experience
	exp := Experience{Description: "test", Outcome: "success"}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	// Add learning
	if err := store.AddLearning("tips", "test tip"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Clear only experiences
	result, err := store.Clear(true, false)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if result.ExperiencesCleared != 1 {
		t.Errorf("expected 1 experience cleared, got %d", result.ExperiencesCleared)
	}

	// Learnings should still exist
	learnings, _ := store.GetLearnings()
	if !strings.Contains(learnings, "test tip") {
		t.Error("learnings should not be cleared")
	}
}

func TestStore_Clear_LearningsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add experience
	exp := Experience{Description: "test", Outcome: "success"}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	// Add learning
	if err := store.AddLearning("tips", "test tip"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Clear only learnings
	result, err := store.Clear(false, true)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if !result.LearningsCleared {
		t.Error("expected learnings to be cleared")
	}

	// Experiences should still exist
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Error("experiences should not be cleared")
	}
}

func TestStore_Clear_Neither(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Clear neither - should still succeed
	result, err := store.Clear(false, false)
	if err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if result.ExperiencesCleared != 0 || result.LearningsCleared {
		t.Error("nothing should be cleared")
	}
}

func TestStore_Prune_AllOld(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add old experience
	oldTime := time.Now().Add(-365 * 24 * time.Hour)
	exp := Experience{
		Timestamp:   oldTime,
		Description: "old experience",
		Outcome:     "success",
	}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	// Prune with 30 day retention
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned, got %d", result.PrunedExperiences)
	}

	// Verify experience is gone
	experiences, _ := store.GetExperiences()
	if len(experiences) != 0 {
		t.Error("old experience should be pruned")
	}
}

func TestStore_NeedsPruning_BelowThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Check with very high threshold
	needs, size, err := store.NeedsPruning(1024 * 1024 * 100) // 100MB threshold
	if err != nil {
		t.Fatalf("NeedsPruning failed: %v", err)
	}

	if needs {
		t.Error("should not need pruning with high threshold")
	}
	if size < 0 {
		t.Error("size should be non-negative")
	}
}

func TestStore_NeedsPruning_AboveThreshold(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add some content
	for i := 0; i < 10; i++ {
		exp := Experience{
			Description: strings.Repeat("x", 1000),
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record: %v", err)
		}
	}

	// Check with very low threshold
	needs, _, err := store.NeedsPruning(1) // 1 byte threshold
	if err != nil {
		t.Fatalf("NeedsPruning failed: %v", err)
	}

	if !needs {
		t.Error("should need pruning with 1 byte threshold")
	}
}

func TestStore_ListTopics_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// No learnings added - only has the default header
	topics, err := store.ListTopics()
	if err != nil {
		t.Fatalf("ListTopics failed: %v", err)
	}

	// Should be empty since no ## sections added yet
	if len(topics) != 0 {
		t.Errorf("expected 0 topics, got %d", len(topics))
	}
}

func TestStore_GetSize_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	size, err := store.GetSize()
	if err != nil {
		t.Fatalf("GetSize failed: %v", err)
	}

	if size < 0 {
		t.Error("size should be non-negative")
	}
}

func TestStore_RecordExperience_WithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	exp := Experience{
		Description: "test with metadata",
		Outcome:     "success",
		Metadata: map[string]any{
			"key1":   "value1",
			"key2":   42,
			"nested": map[string]string{"inner": "data"},
		},
	}

	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}

	if experiences[0].Metadata["key1"] != "value1" {
		t.Error("metadata key1 mismatch")
	}
}

func TestStore_Prune_MixedRetention(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add old experience
	oldExp := Experience{
		Timestamp:   time.Now().Add(-100 * 24 * time.Hour),
		Description: "old",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old: %v", err)
	}

	// Add recent experience
	newExp := Experience{
		Timestamp:   time.Now(),
		Description: "new",
		Outcome:     "success",
	}
	if err := store.RecordExperience(newExp); err != nil {
		t.Fatalf("failed to record new: %v", err)
	}

	// Prune with 30 day retention
	result, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned, got %d", result.PrunedExperiences)
	}
	retained := result.TotalExperiences - result.PrunedExperiences
	if retained != 1 {
		t.Errorf("expected 1 retained, got %d", retained)
	}

	// Verify only new experience remains
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}
	if experiences[0].Description != "new" {
		t.Error("wrong experience retained")
	}
}

func TestStore_DeleteExperience(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add 3 experiences
	for i := 0; i < 3; i++ {
		exp := Experience{
			Description: "Experience " + string(rune('A'+i)),
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record: %v", err)
		}
	}

	// Delete middle experience (index 2, which is B)
	deleted, err := store.DeleteExperience(2)
	if err != nil {
		t.Fatalf("DeleteExperience failed: %v", err)
	}

	if deleted.Description != "Experience B" {
		t.Errorf("expected deleted 'Experience B', got %q", deleted.Description)
	}

	// Verify 2 experiences remain
	experiences, _ := store.GetExperiences()
	if len(experiences) != 2 {
		t.Errorf("expected 2 experiences, got %d", len(experiences))
	}

	// Verify correct ones remain (A and C)
	if experiences[0].Description != "Experience A" {
		t.Errorf("expected 'Experience A', got %q", experiences[0].Description)
	}
	if experiences[1].Description != "Experience C" {
		t.Errorf("expected 'Experience C', got %q", experiences[1].Description)
	}
}

func TestStore_DeleteExperience_First(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add 2 experiences
	for i := 0; i < 2; i++ {
		exp := Experience{
			Description: "Experience " + string(rune('A'+i)),
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record: %v", err)
		}
	}

	// Delete first experience (index 1)
	deleted, err := store.DeleteExperience(1)
	if err != nil {
		t.Fatalf("DeleteExperience failed: %v", err)
	}

	if deleted.Description != "Experience A" {
		t.Errorf("expected deleted 'Experience A', got %q", deleted.Description)
	}

	// Verify B remains
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience, got %d", len(experiences))
	}
	if experiences[0].Description != "Experience B" {
		t.Errorf("expected 'Experience B', got %q", experiences[0].Description)
	}
}

func TestStore_DeleteExperience_Last(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add 2 experiences
	for i := 0; i < 2; i++ {
		exp := Experience{
			Description: "Experience " + string(rune('A'+i)),
			Outcome:     "success",
		}
		if err := store.RecordExperience(exp); err != nil {
			t.Fatalf("failed to record: %v", err)
		}
	}

	// Delete last experience (index 2)
	deleted, err := store.DeleteExperience(2)
	if err != nil {
		t.Fatalf("DeleteExperience failed: %v", err)
	}

	if deleted.Description != "Experience B" {
		t.Errorf("expected deleted 'Experience B', got %q", deleted.Description)
	}

	// Verify A remains
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience, got %d", len(experiences))
	}
	if experiences[0].Description != "Experience A" {
		t.Errorf("expected 'Experience A', got %q", experiences[0].Description)
	}
}

func TestStore_DeleteExperience_OutOfRange(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add 1 experience
	exp := Experience{Description: "Experience A", Outcome: "success"}
	if err := store.RecordExperience(exp); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	// Try to delete out of range
	_, err := store.DeleteExperience(2)
	if err == nil {
		t.Error("expected error for out of range index")
	}

	_, err = store.DeleteExperience(0)
	if err == nil {
		t.Error("expected error for index 0")
	}

	_, err = store.DeleteExperience(-1)
	if err == nil {
		t.Error("expected error for negative index")
	}
}

func TestStore_DeleteLearning(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add 3 learnings to a category
	for _, l := range []string{"first insight", "second insight", "third insight"} {
		if err := store.AddLearning("patterns", l); err != nil {
			t.Fatalf("failed to add learning: %v", err)
		}
	}

	// File order after AddLearning is newest-first: third, second, first
	// Delete middle item (index 2 = "second insight")
	deleted, err := store.DeleteLearning("patterns", 2)
	if err != nil {
		t.Fatalf("DeleteLearning failed: %v", err)
	}

	if deleted != "second insight" {
		t.Errorf("expected deleted 'second insight', got %q", deleted)
	}

	// Verify remaining learnings (third, first)
	topics := parseLearningsByTopic(mustGetLearnings(t, store))
	entries := topics["patterns"]
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0] != "third insight" {
		t.Errorf("expected 'third insight', got %q", entries[0])
	}
	if entries[1] != "first insight" {
		t.Errorf("expected 'first insight', got %q", entries[1])
	}
}

func TestStore_DeleteLearning_FirstAndLast(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// File order after AddLearning is newest-first: gamma, beta, alpha
	for _, l := range []string{"alpha", "beta", "gamma"} {
		if err := store.AddLearning("tips", l); err != nil {
			t.Fatalf("failed to add learning: %v", err)
		}
	}

	// Delete first in file (index 1 = "gamma")
	deleted, err := store.DeleteLearning("tips", 1)
	if err != nil {
		t.Fatalf("DeleteLearning first failed: %v", err)
	}
	if deleted != "gamma" {
		t.Errorf("expected 'gamma', got %q", deleted)
	}

	// Delete last in file (now index 2 = "alpha" after previous deletion)
	deleted, err = store.DeleteLearning("tips", 2)
	if err != nil {
		t.Fatalf("DeleteLearning last failed: %v", err)
	}
	if deleted != "alpha" {
		t.Errorf("expected 'alpha', got %q", deleted)
	}

	// Verify only beta remains
	topics := parseLearningsByTopic(mustGetLearnings(t, store))
	entries := topics["tips"]
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0] != "beta" {
		t.Errorf("expected 'beta', got %q", entries[0])
	}
}

func TestStore_DeleteLearning_LastItem(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	// Add one learning and another category
	if err := store.AddLearning("patterns", "only insight"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := store.AddLearning("tips", "survives"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Delete the only item in patterns — category header should be removed
	deleted, err := store.DeleteLearning("patterns", 1)
	if err != nil {
		t.Fatalf("DeleteLearning failed: %v", err)
	}
	if deleted != "only insight" {
		t.Errorf("expected 'only insight', got %q", deleted)
	}

	// Verify patterns category is gone but tips remains
	content := mustGetLearnings(t, store)
	if strings.Contains(content, "## patterns") {
		t.Error("expected patterns category header to be removed")
	}
	topics := parseLearningsByTopic(content)
	if _, ok := topics["tips"]; !ok {
		t.Error("expected tips category to remain")
	}
	if topics["tips"][0] != "survives" {
		t.Errorf("expected 'survives', got %q", topics["tips"][0])
	}
}

func TestStore_DeleteLearning_InvalidIndex(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	if err := store.AddLearning("patterns", "one insight"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Out of range
	_, err := store.DeleteLearning("patterns", 2)
	if err == nil {
		t.Error("expected error for out of range index")
	}

	// Zero index
	_, err = store.DeleteLearning("patterns", 0)
	if err == nil {
		t.Error("expected error for index 0")
	}

	// Negative index
	_, err = store.DeleteLearning("patterns", -1)
	if err == nil {
		t.Error("expected error for negative index")
	}

	// Missing category
	_, err = store.DeleteLearning("nonexistent", 1)
	if err == nil {
		t.Error("expected error for missing category")
	}
}

func TestStore_DeleteLearning_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir, "test-agent")

	if err := store.Init(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	_, err := store.DeleteLearning("patterns", 1)
	if err == nil {
		t.Error("expected error for empty learnings file")
	}
}

// mustGetLearnings is a test helper that reads learnings or fails.
func mustGetLearnings(t *testing.T, store *Store) string {
	t.Helper()
	content, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}
	return content
}

func TestStore_MergeLearnings(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source store with learnings
	srcStore := NewStore(tmpDir, "source-agent")
	if err := srcStore.Init(); err != nil {
		t.Fatalf("failed to init source: %v", err)
	}
	if err := srcStore.AddLearning("patterns", "Use context for cancellation"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := srcStore.AddLearning("tips", "Check all errors"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Create destination store
	dstStore := NewStore(tmpDir, "dest-agent")
	if err := dstStore.Init(); err != nil {
		t.Fatalf("failed to init dest: %v", err)
	}

	// Merge learnings
	added, err := dstStore.MergeLearnings(srcStore)
	if err != nil {
		t.Fatalf("MergeLearnings failed: %v", err)
	}

	if added != 2 {
		t.Errorf("expected 2 learnings added, got %d", added)
	}

	// Verify learnings exist in destination
	learnings, _ := dstStore.GetLearnings()
	if !strings.Contains(learnings, "Use context for cancellation") {
		t.Error("missing pattern learning in dest")
	}
	if !strings.Contains(learnings, "Check all errors") {
		t.Error("missing tips learning in dest")
	}
}

func TestStore_MergeLearnings_NoDuplicates(t *testing.T) {
	tmpDir := t.TempDir()

	// Create source store with learnings
	srcStore := NewStore(tmpDir, "source-agent")
	if err := srcStore.Init(); err != nil {
		t.Fatalf("failed to init source: %v", err)
	}
	if err := srcStore.AddLearning("patterns", "Shared learning"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}
	if err := srcStore.AddLearning("patterns", "Source only"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Create destination store with existing learning
	dstStore := NewStore(tmpDir, "dest-agent")
	if err := dstStore.Init(); err != nil {
		t.Fatalf("failed to init dest: %v", err)
	}
	if err := dstStore.AddLearning("patterns", "Shared learning"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	// Merge learnings
	added, err := dstStore.MergeLearnings(srcStore)
	if err != nil {
		t.Fatalf("MergeLearnings failed: %v", err)
	}

	// Should only add 1 new learning (Source only), not the duplicate
	if added != 1 {
		t.Errorf("expected 1 learning added (not duplicate), got %d", added)
	}

	// Verify both learnings exist
	learnings, _ := dstStore.GetLearnings()
	if !strings.Contains(learnings, "Shared learning") {
		t.Error("missing shared learning")
	}
	if !strings.Contains(learnings, "Source only") {
		t.Error("missing source only learning")
	}
}

func TestStore_MergeLearnings_EmptySource(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty source store
	srcStore := NewStore(tmpDir, "source-agent")
	if err := srcStore.Init(); err != nil {
		t.Fatalf("failed to init source: %v", err)
	}

	// Create destination store
	dstStore := NewStore(tmpDir, "dest-agent")
	if err := dstStore.Init(); err != nil {
		t.Fatalf("failed to init dest: %v", err)
	}

	// Merge learnings
	added, err := dstStore.MergeLearnings(srcStore)
	if err != nil {
		t.Fatalf("MergeLearnings failed: %v", err)
	}

	if added != 0 {
		t.Errorf("expected 0 learnings added from empty source, got %d", added)
	}
}

func TestStore_SaveRolePrompt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	prompt := "You are an engineer. Implement features and write tests."
	if err := store.SaveRolePrompt(prompt); err != nil {
		t.Fatalf("save role prompt failed: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(dir, ".bc", "memory", "test-agent", "role_prompt.md")) //nolint:gosec // test file path from t.TempDir
	if err != nil {
		t.Fatalf("failed to read role prompt file: %v", err)
	}
	if string(data) != prompt {
		t.Errorf("expected prompt %q, got %q", prompt, string(data))
	}
}

func TestStore_GetRolePrompt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	prompt := "You are a QA engineer."
	if err := store.SaveRolePrompt(prompt); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	got, err := store.GetRolePrompt()
	if err != nil {
		t.Fatalf("get role prompt failed: %v", err)
	}
	if got != prompt {
		t.Errorf("expected %q, got %q", prompt, got)
	}
}

func TestStore_GetRolePromptMissing(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	got, err := store.GetRolePrompt()
	if err != nil {
		t.Fatalf("get role prompt failed: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string for missing prompt, got %q", got)
	}
}

func TestStore_GetMemoryContextIncludesRolePrompt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Save a role prompt
	prompt := "You are an engineer specializing in Go backend development."
	if err := store.SaveRolePrompt(prompt); err != nil {
		t.Fatalf("save role prompt failed: %v", err)
	}

	// Add some experiences to ensure we get output
	if err := store.RecordExperience(Experience{
		TaskType:    "implementation",
		Description: "built log streaming",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("record experience failed: %v", err)
	}

	ctx, err := store.GetMemoryContext(10)
	if err != nil {
		t.Fatalf("get memory context failed: %v", err)
	}

	if !strings.Contains(ctx, "## Role") {
		t.Error("expected memory context to contain '## Role' section")
	}
	if !strings.Contains(ctx, prompt) {
		t.Error("expected memory context to include role prompt content")
	}
}

func TestParseLearningsByTopic(t *testing.T) {
	content := `# Agent Learnings

## patterns
- Use context for cancellation
- Prefer composition

## tips
- Check all errors
`

	topics := parseLearningsByTopic(content)

	if len(topics) != 2 {
		t.Errorf("expected 2 topics, got %d", len(topics))
	}

	patterns := topics["patterns"]
	if len(patterns) != 2 {
		t.Errorf("expected 2 patterns, got %d", len(patterns))
	}
	if patterns[0] != "Use context for cancellation" {
		t.Errorf("wrong first pattern: %s", patterns[0])
	}

	tips := topics["tips"]
	if len(tips) != 1 {
		t.Errorf("expected 1 tip, got %d", len(tips))
	}
}

// --- SaveRolePrompt coverage tests ---

func TestStore_SaveRolePrompt_WithoutInit(t *testing.T) {
	// SaveRolePrompt calls MkdirAll itself, so it should work without Init
	dir := t.TempDir()
	store := NewStore(dir, "no-init-agent")

	prompt := "You are an engineer."
	if err := store.SaveRolePrompt(prompt); err != nil {
		t.Fatalf("SaveRolePrompt without Init should succeed: %v", err)
	}

	got, err := store.GetRolePrompt()
	if err != nil {
		t.Fatalf("GetRolePrompt failed: %v", err)
	}
	if got != prompt {
		t.Errorf("expected %q, got %q", prompt, got)
	}
}

func TestStore_SaveRolePrompt_Overwrite(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Save first prompt
	if err := store.SaveRolePrompt("first prompt"); err != nil {
		t.Fatalf("first save failed: %v", err)
	}

	// Overwrite with second prompt
	if err := store.SaveRolePrompt("second prompt"); err != nil {
		t.Fatalf("second save failed: %v", err)
	}

	got, err := store.GetRolePrompt()
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "second prompt" {
		t.Errorf("expected 'second prompt', got %q", got)
	}
}

func TestStore_SaveRolePrompt_EmptyPrompt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.SaveRolePrompt(""); err != nil {
		t.Fatalf("SaveRolePrompt with empty string should succeed: %v", err)
	}

	got, err := store.GetRolePrompt()
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestStore_SaveRolePrompt_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Place a regular file where the memory dir would be, so MkdirAll fails
	memDir := filepath.Join(dir, ".bc", "memory")
	if err := os.MkdirAll(memDir, 0750); err != nil {
		t.Fatalf("setup mkdir failed: %v", err)
	}
	// Create a file where the agent subdir should go
	agentPath := filepath.Join(memDir, "blocked-agent")
	if err := os.WriteFile(agentPath, []byte("blocker"), 0600); err != nil {
		t.Fatalf("setup file write failed: %v", err)
	}

	store := NewStore(dir, "blocked-agent")
	err := store.SaveRolePrompt("test prompt")
	if err == nil {
		t.Error("expected error when MkdirAll fails due to file conflict")
	}
}

func TestStore_SaveRolePrompt_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make the memory directory read-only so WriteFile fails
	if err := os.Chmod(store.MemoryDir(), 0500); err != nil { //nolint:gosec // intentional permission for test
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(store.MemoryDir(), 0750) }) //nolint:gosec // restore permissions

	err := store.SaveRolePrompt("test prompt")
	if err == nil {
		t.Error("expected error when directory is read-only")
	}
	if !strings.Contains(err.Error(), "failed to save role prompt") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// --- Prune IncludeLearnings coverage ---

func TestStore_Prune_IncludeLearnings(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add old experience and a learning
	if err := store.RecordExperience(Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "old task",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if err := store.AddLearning("patterns", "will be cleared"); err != nil {
		t.Fatalf("add learning failed: %v", err)
	}

	result, err := store.Prune(PruneOptions{
		OlderThan:        30 * 24 * time.Hour,
		DryRun:           false,
		IncludeLearnings: true,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned, got %d", result.PrunedExperiences)
	}
	if !result.LearningsCleared {
		t.Error("expected LearningsCleared to be true")
	}

	// Verify learnings were reset to header only
	learnings, _ := store.GetLearnings()
	if strings.Contains(learnings, "will be cleared") {
		t.Error("learning should have been cleared")
	}
}

func TestStore_Prune_IncludeLearnings_DryRun(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add old experience and a learning
	if err := store.RecordExperience(Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "old task",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if err := store.AddLearning("tips", "keep this"); err != nil {
		t.Fatalf("add learning failed: %v", err)
	}

	result, err := store.Prune(PruneOptions{
		OlderThan:        30 * 24 * time.Hour,
		DryRun:           true,
		IncludeLearnings: true,
	})
	if err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	// DryRun returns early before the IncludeLearnings check, so LearningsCleared stays false
	if result.PrunedExperiences != 1 {
		t.Errorf("expected 1 pruned, got %d", result.PrunedExperiences)
	}

	// Verify learnings were NOT actually cleared (dry run)
	learnings, _ := store.GetLearnings()
	if !strings.Contains(learnings, "keep this") {
		t.Error("learning should still exist after dry run")
	}

	// Experiences should also still exist
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience still (dry run), got %d", len(experiences))
	}
}

// --- Init error path tests ---

func TestStore_Init_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	// Place a regular file where .bc should be a directory
	bcPath := filepath.Join(dir, ".bc")
	if err := os.WriteFile(bcPath, []byte("blocker"), 0600); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	store := NewStore(dir, "test-agent")
	err := store.Init()
	if err == nil {
		t.Error("expected error when MkdirAll fails")
	}
	if !strings.Contains(err.Error(), "failed to create memory directory") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStore_Init_ExperiencesCreateError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")

	// Create the memory dir but make it read-only before Init creates files
	if err := os.MkdirAll(store.MemoryDir(), 0750); err != nil { //nolint:gosec // test setup
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.Chmod(store.MemoryDir(), 0500); err != nil { //nolint:gosec // intentional permission for test
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(store.MemoryDir(), 0750) }) //nolint:gosec // restore permissions

	err := store.Init()
	if err == nil {
		t.Error("expected error when experiences file can't be created")
	}
}

// --- GetExperiences / GetLearnings / GetRolePrompt error paths ---

func TestStore_GetExperiences_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make the experiences file unreadable
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0600)
	})

	_, err := store.GetExperiences()
	if err == nil {
		t.Error("expected error for unreadable experiences file")
	}
	if !strings.Contains(err.Error(), "failed to read experiences") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStore_GetLearnings_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.GetLearnings()
	if err == nil {
		t.Error("expected error for unreadable learnings file")
	}
	if !strings.Contains(err.Error(), "failed to read learnings") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStore_GetRolePrompt_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Create and then make role_prompt.md unreadable
	if err := store.SaveRolePrompt("test"); err != nil {
		t.Fatalf("save failed: %v", err)
	}
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "role_prompt.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "role_prompt.md"), 0600)
	})

	_, err := store.GetRolePrompt()
	if err == nil {
		t.Error("expected error for unreadable role prompt file")
	}
	if !strings.Contains(err.Error(), "failed to read role prompt") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Clear error paths ---

func TestStore_Clear_ExperiencesWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{Description: "test", Outcome: "ok"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make the experiences file itself unwritable
	expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	if err := os.Chmod(expPath, 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(expPath, 0600) })

	_, err := store.Clear(true, false)
	if err == nil {
		t.Error("expected error when can't write experiences file")
	}
}

func TestStore_Clear_LearningsWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.AddLearning("tips", "test"); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Make learnings file read-only
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.Clear(false, true)
	if err == nil {
		t.Error("expected error when can't write learnings file")
	}
}

// --- DeleteExperience error paths ---

func TestStore_DeleteExperience_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{Description: "test", Outcome: "ok"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make experiences file unreadable
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0600)
	})

	_, err := store.DeleteExperience(1)
	if err == nil {
		t.Error("expected error when experiences file is unreadable")
	}
}

func TestStore_DeleteExperience_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{Description: "A", Outcome: "ok"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}
	if err := store.RecordExperience(Experience{Description: "B", Outcome: "ok"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make the file read-only: GetExperiences (ReadFile) can read,
	// but writeExperiences (os.Create with O_RDWR) cannot write
	expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	if err := os.Chmod(expPath, 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(expPath, 0600) })

	_, err := store.DeleteExperience(1)
	if err == nil {
		t.Error("expected error when can't write experiences")
	}
}

func TestStore_DeleteExperience_OnlyOne(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{Description: "lone", Outcome: "ok"}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	deleted, err := store.DeleteExperience(1)
	if err != nil {
		t.Fatalf("DeleteExperience failed: %v", err)
	}
	if deleted.Description != "lone" {
		t.Errorf("expected 'lone', got %q", deleted.Description)
	}

	experiences, _ := store.GetExperiences()
	if len(experiences) != 0 {
		t.Errorf("expected 0 experiences, got %d", len(experiences))
	}
}

// --- DeleteLearning error paths ---

func TestStore_DeleteLearning_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.AddLearning("patterns", "test"); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.DeleteLearning("patterns", 1)
	if err == nil {
		t.Error("expected error when learnings file is unreadable")
	}
}

func TestStore_DeleteLearning_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add multiple learnings so it takes the non-ForgetTopic path
	if err := store.AddLearning("patterns", "first"); err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if err := store.AddLearning("patterns", "second"); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Make learnings file read-only
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.DeleteLearning("patterns", 1)
	if err == nil {
		t.Error("expected error when can't write learnings file")
	}
}

// --- MergeLearnings error paths ---

func TestStore_MergeLearnings_SourceReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	srcStore := NewStore(dir, "source")
	if err := srcStore.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make source learnings unreadable
	if err := os.Chmod(filepath.Join(srcStore.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(srcStore.MemoryDir(), "learnings.md"), 0600)
	})

	dstStore := NewStore(dir, "dest")
	if err := dstStore.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	_, err := dstStore.MergeLearnings(srcStore)
	if err == nil {
		t.Error("expected error when source learnings unreadable")
	}
	if !strings.Contains(err.Error(), "source learnings") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStore_MergeLearnings_DestReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	srcStore := NewStore(dir, "source")
	if err := srcStore.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	dstStore := NewStore(dir, "dest")
	if err := dstStore.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make dest learnings unreadable
	if err := os.Chmod(filepath.Join(dstStore.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(dstStore.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := dstStore.MergeLearnings(srcStore)
	if err == nil {
		t.Error("expected error when dest learnings unreadable")
	}
	if !strings.Contains(err.Error(), "destination learnings") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- ForgetTopic error paths ---

func TestStore_ForgetTopic_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.AddLearning("patterns", "test"); err != nil {
		t.Fatalf("add failed: %v", err)
	}

	// Make learnings file read-only
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.ForgetTopic("patterns")
	if err == nil {
		t.Error("expected error when can't write learnings file")
	}
}

// --- AddLearning error paths ---

func TestStore_AddLearning_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	err := store.AddLearning("patterns", "test")
	if err == nil {
		t.Error("expected error when learnings file unreadable")
	}
	if !strings.Contains(err.Error(), "failed to read learnings") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStore_AddLearning_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make learnings file read-only (can read but not write)
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	err := store.AddLearning("patterns", "test")
	if err == nil {
		t.Error("expected error when learnings file is read-only")
	}
	if !strings.Contains(err.Error(), "failed to write learnings") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- RecordExperience error paths ---

func TestStore_RecordExperience_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make experiences file read-only
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0600)
	})

	err := store.RecordExperience(Experience{Description: "test", Outcome: "ok"})
	if err == nil {
		t.Error("expected error when experiences file is read-only")
	}
	if !strings.Contains(err.Error(), "failed to open experiences file") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- createBackup error paths ---

func TestStore_Prune_BackupReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "old",
		Outcome:     "ok",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make experiences file unreadable so backup read fails
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0600)
	})

	_, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
		Backup:    true,
	})
	if err == nil {
		t.Error("expected error when experiences file is unreadable for backup")
	}
}

// --- writeExperiences error path ---

func TestStore_Prune_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "old",
		Outcome:     "ok",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make the file read-only: GetExperiences (ReadFile) can read,
	// but writeExperiences (os.Create with O_RDWR) cannot write
	expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	if err := os.Chmod(expPath, 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(expPath, 0600) })

	_, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
	})
	if err == nil {
		t.Error("expected error when can't write pruned experiences")
	}
}

// --- NeedsPruning with GetSize error ---

func TestStore_NeedsPruning_NonExistentStore(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "ghost-agent")

	// Store doesn't exist — GetSize returns 0, no error
	needs, size, err := store.NeedsPruning(DefaultSizeThreshold)
	if err != nil {
		t.Fatalf("NeedsPruning failed: %v", err)
	}
	if needs {
		t.Error("non-existent store should not need pruning")
	}
	if size != 0 {
		t.Errorf("expected 0 size, got %d", size)
	}
}

// --- GetMemoryContext error paths ---

func TestStore_GetMemoryContext_WithLearningsAndRolePrompt(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Save a role prompt
	if err := store.SaveRolePrompt("You are a test engineer."); err != nil {
		t.Fatalf("save role prompt failed: %v", err)
	}

	// Add experiences
	if err := store.RecordExperience(Experience{
		TaskType:    "test",
		Description: "wrote unit tests",
		Outcome:     "success",
		Learnings:   []string{"Test edge cases"},
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Add substantial learnings (> 100 chars to trigger inclusion)
	for i := 0; i < 5; i++ {
		if err := store.AddLearning("category-"+string(rune('A'+i)),
			"Important learning about topic "+string(rune('A'+i))+" with enough detail to be meaningful"); err != nil {
			t.Fatalf("add learning failed: %v", err)
		}
	}

	ctx, err := store.GetMemoryContext(10)
	if err != nil {
		t.Fatalf("GetMemoryContext failed: %v", err)
	}

	// Should contain all three sections
	if !strings.Contains(ctx, "## Role") {
		t.Error("missing Role section")
	}
	if !strings.Contains(ctx, "## Recent Experiences") {
		t.Error("missing Recent Experiences section")
	}
	if !strings.Contains(ctx, "## Key Learnings") {
		t.Error("missing Key Learnings section")
	}
	if !strings.Contains(ctx, "Test edge cases") {
		t.Error("missing experience learning")
	}
}

func TestStore_GetMemoryContext_ExperiencesReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0600)
	})

	_, err := store.GetMemoryContext(10)
	if err == nil {
		t.Error("expected error when experiences file is unreadable")
	}
}

func TestStore_GetMemoryContext_LearningsReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Add an experience so we get past the experiences check
	if err := store.RecordExperience(Experience{
		TaskType: "test", Description: "test", Outcome: "ok",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.GetMemoryContext(10)
	if err == nil {
		t.Error("expected error when learnings file is unreadable")
	}
}

// --- ListTopics error path ---

func TestStore_ListTopics_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.ListTopics()
	if err == nil {
		t.Error("expected error when learnings file is unreadable")
	}
}

// --- clearLearnings error path ---

func TestStore_clearLearnings_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Make learnings file read-only
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	err := store.clearLearnings()
	if err == nil {
		t.Error("expected error when learnings file is read-only")
	}
	if !strings.Contains(err.Error(), "failed to reset learnings file") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- Prune clearLearnings error path ---

func TestStore_Prune_IncludeLearnings_ClearError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "old",
		Outcome:     "ok",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make learnings read-only so clearLearnings fails
	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0400); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.Prune(PruneOptions{
		OlderThan:        30 * 24 * time.Hour,
		DryRun:           false,
		IncludeLearnings: true,
	})
	if err == nil {
		t.Error("expected error when clearLearnings fails during prune")
	}
}

// --- ForgetTopic read error ---

func TestStore_ForgetTopic_ReadError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "learnings.md"), 0600)
	})

	_, err := store.ForgetTopic("patterns")
	if err == nil {
		t.Error("expected error when learnings file is unreadable")
	}
}

// --- Prune GetExperiences error path ---

func TestStore_Prune_GetExperiencesError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0000); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(filepath.Join(store.MemoryDir(), "experiences.jsonl"), 0600)
	})

	_, err := store.Prune(PruneOptions{OlderThan: 30 * 24 * time.Hour})
	if err == nil {
		t.Error("expected error when experiences file is unreadable")
	}
}

// --- Backup write error ---

func TestStore_Prune_BackupWriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	dir := t.TempDir()
	store := NewStore(dir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if err := store.RecordExperience(Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "old",
		Outcome:     "ok",
	}); err != nil {
		t.Fatalf("record failed: %v", err)
	}

	// Make dir read-only so backup write fails (but experiences.jsonl must be readable)
	expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
	if err := os.Chmod(expPath, 0444); err != nil { //nolint:gosec // intentional permission for test
		t.Fatalf("chmod exp failed: %v", err)
	}
	if err := os.Chmod(store.MemoryDir(), 0500); err != nil { //nolint:gosec // intentional permission for test
		t.Fatalf("chmod dir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(store.MemoryDir(), 0750) //nolint:gosec // restore permissions
		_ = os.Chmod(expPath, 0600)
	})

	_, err := store.Prune(PruneOptions{
		OlderThan: 30 * 24 * time.Hour,
		DryRun:    false,
		Backup:    true,
	})
	if err == nil {
		t.Error("expected error when backup write fails")
	}
}
