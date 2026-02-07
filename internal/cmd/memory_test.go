package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
