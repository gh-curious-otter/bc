package memory

import (
	"os"
	"path/filepath"
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
