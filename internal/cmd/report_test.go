package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/memory"
)

// --- Report Command Unit Tests ---

func TestReport_NoAgentID(t *testing.T) {
	setupTestWorkspace(t)

	// Clear BC_AGENT_ID env var
	origAgentID := os.Getenv("BC_AGENT_ID")
	_ = os.Unsetenv("BC_AGENT_ID")
	defer func() {
		if origAgentID != "" {
			_ = os.Setenv("BC_AGENT_ID", origAgentID)
		}
	}()

	// Report without BC_AGENT_ID should fail
	_, err := executeCmd("report", "working", "test message")
	if err == nil {
		t.Error("expected error when BC_AGENT_ID not set")
	}
	if err != nil && !strings.Contains(err.Error(), "this command can only be run by agents in the bc system") {
		t.Errorf("expected agent-only command error, got: %v", err)
	}
}

func TestReport_InvalidState(t *testing.T) {
	setupTestWorkspace(t)

	// Set BC_AGENT_ID
	origAgentID := os.Getenv("BC_AGENT_ID")
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() {
		if origAgentID != "" {
			_ = os.Setenv("BC_AGENT_ID", origAgentID)
		} else {
			_ = os.Unsetenv("BC_AGENT_ID")
		}
	}()

	// Report with invalid state should fail
	_, err := executeCmd("report", "invalid-state", "test message")
	if err == nil {
		t.Error("expected error for invalid state")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid state") {
		t.Errorf("expected invalid state error, got: %v", err)
	}
}

// --- Report Command Args Tests ---

func TestReportCommand_RequiresState(t *testing.T) {
	// Report requires at least 1 arg (state)
	err := reportCmd.Args(reportCmd, []string{})
	if err == nil {
		t.Error("expected error for missing state arg")
	}
}

func TestReportCommand_AcceptsStateOnly(t *testing.T) {
	// Report accepts state only
	err := reportCmd.Args(reportCmd, []string{"working"})
	if err != nil {
		t.Errorf("unexpected error for state-only args: %v", err)
	}
}

func TestReportCommand_AcceptsStateAndMessage(t *testing.T) {
	// Report accepts state + message
	err := reportCmd.Args(reportCmd, []string{"working", "test", "message"})
	if err != nil {
		t.Errorf("unexpected error for state + message args: %v", err)
	}
}

// --- Auto-Record Experience Tests ---

func TestRecordExperience(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	err := recordExperience(wsDir, "test-agent", "Completed task X", "success")
	if err != nil {
		t.Fatalf("recordExperience failed: %v", err)
	}

	// Verify experience was recorded
	store := memory.NewStore(wsDir, "test-agent")
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(experiences))
	}

	if experiences[0].Description != "Completed task X" {
		t.Errorf("expected 'Completed task X', got %q", experiences[0].Description)
	}
	if experiences[0].Outcome != "success" {
		t.Errorf("expected 'success', got %q", experiences[0].Outcome)
	}
}

func TestRecordExperience_Deduplication(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Record same experience twice
	err := recordExperience(wsDir, "dedup-agent", "Fixed bug", "success")
	if err != nil {
		t.Fatalf("first recordExperience failed: %v", err)
	}

	err = recordExperience(wsDir, "dedup-agent", "Fixed bug", "success")
	if err != nil {
		t.Fatalf("second recordExperience failed: %v", err)
	}

	// Verify only 1 experience recorded (deduplicated)
	store := memory.NewStore(wsDir, "dedup-agent")
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 1 {
		t.Errorf("expected 1 experience after dedup, got %d", len(experiences))
	}
}

func TestRecordExperience_DifferentDescriptions(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Record different experiences
	if err := recordExperience(wsDir, "multi-agent", "Task A", "success"); err != nil {
		t.Fatalf("first recordExperience failed: %v", err)
	}

	if err := recordExperience(wsDir, "multi-agent", "Task B", "success"); err != nil {
		t.Fatalf("second recordExperience failed: %v", err)
	}

	// Verify both recorded
	store := memory.NewStore(wsDir, "multi-agent")
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}

	if len(experiences) != 2 {
		t.Errorf("expected 2 experiences, got %d", len(experiences))
	}
}

func TestRecordExperience_InitializesMemory(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Record to non-existent agent memory
	err := recordExperience(wsDir, "new-agent", "First task", "success")
	if err != nil {
		t.Fatalf("recordExperience failed: %v", err)
	}

	// Verify memory was initialized
	store := memory.NewStore(wsDir, "new-agent")
	if !store.Exists() {
		t.Error("memory should be initialized after recording")
	}
}
