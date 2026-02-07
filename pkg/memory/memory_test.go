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
