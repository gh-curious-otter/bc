package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/process"
)

// Process command tests use executeIntegrationCmd which captures os.Stdout
// because process.go uses fmt.Printf directly rather than cmd.OutOrStdout()

func TestProcessListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("process", "list")
	if err != nil {
		t.Fatalf("process list failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No processes") {
		t.Errorf("expected 'No processes', got: %s", stdout)
	}
}

func TestProcessListWithProcesses(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create a process registry and add a process
	reg := process.NewRegistry(wsDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	// Register a process manually
	proc := &process.Process{
		Name:      "test-web",
		Command:   "node server.js",
		PID:       12345,
		Port:      3000,
		Owner:     "engineer-01",
		Running:   true,
		StartedAt: time.Now(),
	}
	if err := reg.Register(proc); err != nil {
		t.Fatalf("failed to register process: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("process", "list")
	if err != nil {
		t.Fatalf("process list failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "test-web") {
		t.Errorf("output should contain process name: %s", stdout)
	}
	if !strings.Contains(stdout, "running") {
		t.Errorf("output should contain status: %s", stdout)
	}
	if !strings.Contains(stdout, "3000") {
		t.Errorf("output should contain port: %s", stdout)
	}
}

func TestProcessInfoFound(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	reg := process.NewRegistry(wsDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	proc := &process.Process{
		Name:      "test-api",
		Command:   "go run ./cmd/api",
		PID:       54321,
		Port:      8080,
		Owner:     "engineer-02",
		WorkDir:   "/app",
		Running:   true,
		StartedAt: time.Now(),
	}
	if err := reg.Register(proc); err != nil {
		t.Fatalf("failed to register process: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("process", "info", "test-api")
	if err != nil {
		t.Fatalf("process info failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "test-api") {
		t.Errorf("output should contain process name: %s", stdout)
	}
	if !strings.Contains(stdout, "go run ./cmd/api") {
		t.Errorf("output should contain command: %s", stdout)
	}
	if !strings.Contains(stdout, "running") {
		t.Errorf("output should contain status: %s", stdout)
	}
	if !strings.Contains(stdout, "8080") {
		t.Errorf("output should contain port: %s", stdout)
	}
}

func TestProcessInfoNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "info", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessLogsNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "logs", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessLogsNoLogs(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	reg := process.NewRegistry(wsDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	proc := &process.Process{
		Name:      "test-logs",
		Command:   "echo hello",
		PID:       99999,
		Running:   false,
		StartedAt: time.Now(),
	}
	if err := reg.Register(proc); err != nil {
		t.Fatalf("failed to register process: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("process", "logs", "test-logs")
	if err != nil {
		t.Fatalf("process logs failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No logs available") {
		t.Errorf("expected 'No logs available', got: %s", stdout)
	}
}

func TestProcessLogsWithContent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	reg := process.NewRegistry(wsDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	// Create log file
	logPath := reg.LogPath("test-logs")
	if err := os.MkdirAll(filepath.Dir(logPath), 0750); err != nil {
		t.Fatalf("failed to create log dir: %v", err)
	}
	logContent := "line 1\nline 2\nline 3\n"
	if err := os.WriteFile(logPath, []byte(logContent), 0600); err != nil {
		t.Fatalf("failed to write log file: %v", err)
	}

	proc := &process.Process{
		Name:      "test-logs",
		Command:   "echo hello",
		PID:       99999,
		LogFile:   logPath,
		Running:   false,
		StartedAt: time.Now(),
	}
	if err := reg.Register(proc); err != nil {
		t.Fatalf("failed to register process: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("process", "logs", "test-logs")
	if err != nil {
		t.Fatalf("process logs failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "line 1") {
		t.Errorf("output should contain log content: %s", stdout)
	}
}

func TestProcessStopNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "stop", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessStopNotRunning(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	reg := process.NewRegistry(wsDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	proc := &process.Process{
		Name:      "stopped-proc",
		Command:   "echo hello",
		PID:       12345,
		StartedAt: time.Now(),
	}
	if err := reg.Register(proc); err != nil {
		t.Fatalf("failed to register process: %v", err)
	}

	// Mark as stopped (Register sets Running = true by default)
	if err := reg.MarkStopped("stopped-proc"); err != nil {
		t.Fatalf("failed to mark stopped: %v", err)
	}

	_, _, err := executeIntegrationCmd("process", "stop", "stopped-proc")
	if err == nil {
		t.Error("expected error for stopped process")
	}
	if err != nil && !strings.Contains(err.Error(), "not running") {
		t.Errorf("error should mention not running: %v", err)
	}
}

func TestProcessAttachNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("process", "attach", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessAttachNotRunning(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	reg := process.NewRegistry(wsDir)
	if err := reg.Init(); err != nil {
		t.Fatalf("failed to init registry: %v", err)
	}

	proc := &process.Process{
		Name:      "attach-stopped",
		Command:   "echo hello",
		PID:       12345,
		StartedAt: time.Now(),
	}
	if err := reg.Register(proc); err != nil {
		t.Fatalf("failed to register process: %v", err)
	}

	// Mark as stopped (Register sets Running = true by default)
	if err := reg.MarkStopped("attach-stopped"); err != nil {
		t.Fatalf("failed to mark stopped: %v", err)
	}

	_, _, err := executeIntegrationCmd("process", "attach", "attach-stopped")
	if err == nil {
		t.Error("expected error for stopped process")
	}
	if err != nil && !strings.Contains(err.Error(), "not running") {
		t.Errorf("error should mention not running: %v", err)
	}
}

func TestProcessNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, execErr := executeIntegrationCmd("process", "list")
	if execErr == nil {
		t.Error("expected error when not in workspace")
	}
	if !strings.Contains(execErr.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", execErr)
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
	for _, tt := range tests {
		got := statusStr(tt.running)
		if got != tt.want {
			t.Errorf("statusStr(%v) = %q, want %q", tt.running, got, tt.want)
		}
	}
}
