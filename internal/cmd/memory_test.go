package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/memory"
)

func TestMemoryRecord(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Set agent ID
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	output, err := executeCmd("memory", "record", "--outcome", "success", "Fixed a bug")
	if err != nil {
		t.Fatalf("memory record failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Recorded experience") {
		t.Errorf("expected confirmation message, got: %s", output)
	}

	// Verify experience was recorded
	store := memory.NewStore(wsDir, "test-agent")
	exps, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}
	if len(exps) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(exps))
	}
	if exps[0].Description != "Fixed a bug" {
		t.Errorf("unexpected description: %s", exps[0].Description)
	}
	if exps[0].Outcome != "success" {
		t.Errorf("unexpected outcome: %s", exps[0].Outcome)
	}
}

func TestMemoryLearn(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Set agent ID
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	output, err := executeCmd("memory", "learn", "patterns", "Always check errors")
	if err != nil {
		t.Fatalf("memory learn failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Added learning") {
		t.Errorf("expected confirmation message, got: %s", output)
	}

	// Verify learning was added
	store := memory.NewStore(wsDir, "test-agent")
	learnings, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}
	if !strings.Contains(learnings, "patterns") {
		t.Errorf("learnings should contain category 'patterns': %s", learnings)
	}
	if !strings.Contains(learnings, "Always check errors") {
		t.Errorf("learnings should contain the learning: %s", learnings)
	}
}

func TestMemoryShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Set agent ID
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	// Initialize memory with some content
	store := memory.NewStore(wsDir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "Test experience",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}
	if err := store.AddLearning("tips", "Test tip"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	output, err := executeCmd("memory", "show")
	if err != nil {
		t.Fatalf("memory show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Experiences") {
		t.Errorf("output should contain 'Experiences': %s", output)
	}
	if !strings.Contains(output, "Test experience") {
		t.Errorf("output should contain the experience: %s", output)
	}
	if !strings.Contains(output, "Learnings") {
		t.Errorf("output should contain 'Learnings': %s", output)
	}
}

func TestMemoryShowSpecificAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memory for a specific agent
	store := memory.NewStore(wsDir, "other-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "Other agent experience",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	output, err := executeCmd("memory", "show", "other-agent")
	if err != nil {
		t.Fatalf("memory show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "other-agent") {
		t.Errorf("output should reference the agent: %s", output)
	}
	if !strings.Contains(output, "Other agent experience") {
		t.Errorf("output should contain the experience: %s", output)
	}
}

func TestMemorySearch(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memory with searchable content
	store := memory.NewStore(wsDir, "search-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "Fixed authentication bug",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	output, err := executeCmd("memory", "search", "authentication")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "authentication") {
		t.Errorf("output should contain search result: %s", output)
	}
}

func TestMemorySearchNoResults(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create empty memory
	store := memory.NewStore(wsDir, "empty-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	output, err := executeCmd("memory", "search", "nonexistent")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No results found") {
		t.Errorf("output should indicate no results: %s", output)
	}
}

func TestMemoryRecordRequiresAgentID(t *testing.T) {
	setupTestWorkspace(t)

	// Ensure BC_AGENT_ID is not set
	_ = os.Unsetenv("BC_AGENT_ID")

	_, err := executeCmd("memory", "record", "Test")
	if err == nil {
		t.Error("expected error when BC_AGENT_ID not set")
	}
	if !strings.Contains(err.Error(), "BC_AGENT_ID") {
		t.Errorf("error should mention BC_AGENT_ID: %v", err)
	}
}

func TestMemoryShowNoMemory(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Ensure the memory directory doesn't exist
	memoryDir := filepath.Join(wsDir, ".bc", "memory", "nonexistent-agent")
	_ = os.RemoveAll(memoryDir)

	output, err := executeCmd("memory", "show", "nonexistent-agent")
	if err != nil {
		t.Fatalf("memory show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No memory found") {
		t.Errorf("output should indicate no memory: %s", output)
	}
}

func TestMemoryRecordRejectsEmpty(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "record", "")
	if err == nil {
		t.Error("expected error for empty experience")
	}
	if !strings.Contains(err.Error(), "experience cannot be empty") {
		t.Errorf("error should mention empty experience: %v", err)
	}
}

func TestMemoryRecordRejectsWhitespace(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "record", "   ")
	if err == nil {
		t.Error("expected error for whitespace-only experience")
	}
	if !strings.Contains(err.Error(), "experience cannot be empty") {
		t.Errorf("error should mention empty experience: %v", err)
	}
}

func TestMemoryLearnRejectsEmptyCategory(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "learn", "", "some learning")
	if err == nil {
		t.Error("expected error for empty category")
	}
	if !strings.Contains(err.Error(), "category cannot be empty") {
		t.Errorf("error should mention empty category: %v", err)
	}
}

func TestMemoryLearnRejectsEmptyLearning(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "learn", "patterns", "")
	if err == nil {
		t.Error("expected error for empty learning")
	}
	if !strings.Contains(err.Error(), "learning cannot be empty") {
		t.Errorf("error should mention empty learning: %v", err)
	}
}

func TestMemorySearchRankedResults(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memory with multiple experiences to test ranking
	store := memory.NewStore(wsDir, "ranked-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Experience with auth in description (should rank higher)
	if err := store.RecordExperience(memory.Experience{
		Description: "Fixed authentication auth flow",
		Outcome:     "success",
		TaskType:    "bugfix",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	// Experience with auth in task type only (lower rank)
	if err := store.RecordExperience(memory.Experience{
		Description: "Updated login page",
		Outcome:     "success",
		TaskType:    "auth",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	output, err := executeCmd("memory", "search", "auth")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}

	// Should show count of results
	if !strings.Contains(output, "2 found") {
		t.Errorf("output should show result count: %s", output)
	}

	// Should show score
	if !strings.Contains(output, "score:") {
		t.Errorf("output should show relevance score: %s", output)
	}

	// First result should have higher score (auth in description)
	if !strings.Contains(output, "1. [ranked-agent]") {
		t.Errorf("output should number results: %s", output)
	}
}

func TestMemorySearchMultipleAgents(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memories for two agents
	store1 := memory.NewStore(wsDir, "agent-one")
	if err := store1.Init(); err != nil {
		t.Fatalf("failed to init store1: %v", err)
	}
	if err := store1.RecordExperience(memory.Experience{
		Description: "Fixed auth bug in agent one",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	store2 := memory.NewStore(wsDir, "agent-two")
	if err := store2.Init(); err != nil {
		t.Fatalf("failed to init store2: %v", err)
	}
	if err := store2.RecordExperience(memory.Experience{
		Description: "Auth system redesign",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	output, err := executeCmd("memory", "search", "auth")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}

	// Should find results from both agents
	if !strings.Contains(output, "agent-one") {
		t.Errorf("output should contain agent-one: %s", output)
	}
	if !strings.Contains(output, "agent-two") {
		t.Errorf("output should contain agent-two: %s", output)
	}
}

func TestMemorySearchSpecificAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memories for two agents
	store1 := memory.NewStore(wsDir, "target-agent")
	if err := store1.Init(); err != nil {
		t.Fatalf("failed to init store1: %v", err)
	}
	if err := store1.RecordExperience(memory.Experience{
		Description: "Fixed bug in target",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	store2 := memory.NewStore(wsDir, "other-agent")
	if err := store2.Init(); err != nil {
		t.Fatalf("failed to init store2: %v", err)
	}
	if err := store2.RecordExperience(memory.Experience{
		Description: "Fixed bug in other",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	// Search only target-agent
	output, err := executeCmd("memory", "search", "--agent", "target-agent", "bug")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}

	// Should only find target-agent
	if !strings.Contains(output, "target-agent") {
		t.Errorf("output should contain target-agent: %s", output)
	}
	if strings.Contains(output, "other-agent") {
		t.Errorf("output should NOT contain other-agent: %s", output)
	}
}

func TestScoreExperience(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment doesn't matter
		name     string
		exp      memory.Experience
		query    string
		minScore int
	}{
		{
			name: "match in description",
			exp: memory.Experience{
				Description: "Fixed authentication bug",
			},
			query:    "auth",
			minScore: 10,
		},
		{
			name: "match in task type",
			exp: memory.Experience{
				Description: "Updated login",
				TaskType:    "auth",
			},
			query:    "auth",
			minScore: 5,
		},
		{
			name: "match in outcome",
			exp: memory.Experience{
				Description: "Some task",
				Outcome:     "auth failed",
			},
			query:    "auth",
			minScore: 3,
		},
		{
			name: "no match",
			exp: memory.Experience{
				Description: "Updated UI",
			},
			query:    "auth",
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreExperience(tt.exp, tt.query)
			if score < tt.minScore {
				t.Errorf("scoreExperience() = %d, want >= %d", score, tt.minScore)
			}
		})
	}
}

func TestScoreLearning(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		query    string
		minScore int
	}{
		{
			name:     "simple match",
			line:     "Always check authentication",
			query:    "auth",
			minScore: 5,
		},
		{
			name:     "header match",
			line:     "## Authentication Patterns",
			query:    "auth",
			minScore: 8, // 5 + 3 for header
		},
		{
			name:     "no match",
			line:     "Some other content",
			query:    "auth",
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreLearning(tt.line, tt.query)
			if score < tt.minScore {
				t.Errorf("scoreLearning() = %d, want >= %d", score, tt.minScore)
			}
		})
	}
}

// --- Memory Prune Tests ---

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"1h", 1 * time.Hour, false},
		{"60m", 60 * time.Minute, false},
		{"x", 0, true},   // too short
		{"abc", 0, true}, // invalid number
		{"7x", 0, true},  // unknown unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseDuration(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseDuration(%q) error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMemoryPruneDryRun(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags (they persist between tests)
	memoryPruneDryRun = false
	memoryPruneAgent = ""
	memoryPruneOlder = "30d"

	// Create memory with old experience
	store := memory.NewStore(wsDir, "prune-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Record an old experience (manually set timestamp in the past)
	oldExp := memory.Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour), // 60 days ago
		Description: "Old task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old experience: %v", err)
	}

	// Record a recent experience
	newExp := memory.Experience{
		Description: "Recent task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(newExp); err != nil {
		t.Fatalf("failed to record new experience: %v", err)
	}

	output, err := executeCmd("memory", "prune", "--agent", "prune-agent", "--older-than", "30d", "--dry-run")
	if err != nil {
		t.Fatalf("memory prune failed: %v\nOutput: %s", err, output)
	}

	// Should mention dry run
	if !strings.Contains(output, "Dry run") {
		t.Errorf("output should mention dry run: %s", output)
	}

	// Should still have both experiences (dry run doesn't delete)
	experiences, _ := store.GetExperiences()
	if len(experiences) != 2 {
		t.Errorf("expected 2 experiences after dry run, got %d", len(experiences))
	}
}

func TestMemoryPruneActual(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags via Cobra (they persist between tests)
	_ = memoryPruneCmd.Flags().Set("dry-run", "false")

	// Create memory with old experience
	store := memory.NewStore(wsDir, "prune-real")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Record an old experience
	oldExp := memory.Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour), // 60 days ago
		Description: "Old task to prune",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record old experience: %v", err)
	}

	// Record a recent experience
	newExp := memory.Experience{
		Description: "Recent task to keep",
		Outcome:     "success",
	}
	if err := store.RecordExperience(newExp); err != nil {
		t.Fatalf("failed to record new experience: %v", err)
	}

	output, err := executeCmd("memory", "prune", "--agent", "prune-real", "--older-than", "30d", "--dry-run=false")
	if err != nil {
		t.Fatalf("memory prune failed: %v\nOutput: %s", err, output)
	}

	// Should report pruning
	if !strings.Contains(output, "Pruned") {
		t.Errorf("output should report pruning: %s", output)
	}

	// Should only have 1 experience now
	experiences, _ := store.GetExperiences()
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience after prune, got %d", len(experiences))
	}
	if experiences[0].Description != "Recent task to keep" {
		t.Errorf("wrong experience kept: %s", experiences[0].Description)
	}
}

func TestMemoryPruneCreatesBackup(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags via Cobra (they persist between tests)
	_ = memoryPruneCmd.Flags().Set("dry-run", "false")

	// Create memory with old experience
	store := memory.NewStore(wsDir, "backup-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Record an old experience
	oldExp := memory.Experience{
		Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		Description: "Old task",
		Outcome:     "success",
	}
	if err := store.RecordExperience(oldExp); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	_, err := executeCmd("memory", "prune", "--agent", "backup-agent", "--older-than", "30d", "--dry-run=false")
	if err != nil {
		t.Fatalf("memory prune failed: %v", err)
	}

	// Check backup was created
	backupPath := filepath.Join(store.MemoryDir(), "experiences.jsonl.bak")
	if _, statErr := os.Stat(backupPath); os.IsNotExist(statErr) {
		t.Error("backup file should exist after prune")
	}
}
