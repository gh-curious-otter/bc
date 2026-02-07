package cmd

import (
	"os"
	"strings"
	"testing"
)

// resetProcessFlags resets process command flags between tests
func resetProcessFlags() {
	processCommand = ""
	processPort = 0
	processWorkDir = ""
	processLogLines = 50
}

func TestProcessListNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("process", "list")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestProcessListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("process", "list")
	if err != nil {
		t.Fatalf("process list returned error: %v", err)
	}
	if !strings.Contains(stdout, "No processes") {
		t.Errorf("expected 'No processes', got: %s", stdout)
	}
}

func TestProcessStopNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "stop", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessLogsNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "logs", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessInfoNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "info", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessAttachNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "attach", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing process, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessStartRequiresCmd(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProcessFlags()
	defer resetProcessFlags()

	_, _, err := executeIntegrationCmd("process", "start", "test-proc")
	if err == nil {
		t.Fatal("expected error for missing --cmd flag, got nil")
	}
	// The error message varies by Cobra version but should indicate missing flag
	if !strings.Contains(err.Error(), "required") && !strings.Contains(err.Error(), "cmd") {
		t.Errorf("expected flag requirement error, got: %v", err)
	}
}

func TestProcessFlagDefaults(t *testing.T) {
	// Check process list flag defaults
	linesFlag := processLogsCmd.Flags().Lookup("lines")
	if linesFlag == nil {
		t.Fatal("lines flag not found")
	}
	if linesFlag.DefValue != "50" {
		t.Errorf("lines default: got %q, want %q", linesFlag.DefValue, "50")
	}

	// Check process start flag defaults
	portFlag := processStartCmd.Flags().Lookup("port")
	if portFlag == nil {
		t.Fatal("port flag not found")
	}
	if portFlag.DefValue != "0" {
		t.Errorf("port default: got %q, want %q", portFlag.DefValue, "0")
	}
}

func TestProcessStartNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	resetProcessFlags()
	processCommand = "echo hello"
	defer resetProcessFlags()

	_, _, err = executeIntegrationCmd("process", "start", "test", "--cmd", "echo hello")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}
