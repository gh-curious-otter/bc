package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestProcessListEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("process", "list")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "No processes") {
		t.Errorf("expected 'No processes', got: %s", output)
	}
}

func TestProcessStopNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "stop", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent process")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessLogsNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "logs", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent process")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessInfoNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "info", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent process")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessAttachNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "attach", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent process")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestProcessNoWorkspace(t *testing.T) {
	// Run outside a workspace
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("process", "list")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}

	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected 'not in a bc workspace' error, got: %v", err)
	}
}

func TestProcessStartRequiresCmd(t *testing.T) {
	setupTestWorkspace(t)

	// Reset flag
	processCommand = ""

	_, err := executeCmd("process", "start", "test")
	if err == nil {
		t.Fatal("expected error when --cmd not provided")
	}

	if !strings.Contains(err.Error(), "required") {
		t.Errorf("expected 'required' error, got: %v", err)
	}
}

func TestStatusStr(t *testing.T) {
	tests := []struct {
		want    string
		running bool
	}{
		{"running", true},
		{"stopped", false},
	}

	for _, tc := range tests {
		got := statusStr(tc.running)
		if got != tc.want {
			t.Errorf("statusStr(%v) = %q, want %q", tc.running, got, tc.want)
		}
	}
}
