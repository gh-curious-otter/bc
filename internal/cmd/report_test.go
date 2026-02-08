package cmd

import (
	"os"
	"strings"
	"testing"
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
	if err != nil && !strings.Contains(err.Error(), "BC_AGENT_ID not set") {
		t.Errorf("expected BC_AGENT_ID error, got: %v", err)
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
