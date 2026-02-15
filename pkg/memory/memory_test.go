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
