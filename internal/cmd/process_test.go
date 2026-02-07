package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/process"
)

func TestProcessStartWithLogs(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	output, err := executeCmd("process", "start", "log-proc", "--cmd", "echo 'hello from test'")
	if err != nil {
		t.Fatalf("process start failed: %v\nOutput: %s", err, output)
	}

	// Give it a moment to write logs
	time.Sleep(200 * time.Millisecond)

	// Check log file exists
	logPath := filepath.Join(wsDir, ".bc", "processes", "log-proc.logs", "output.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Errorf("log file should exist: %s", logPath)
	}
}

func TestProcessLogs(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create log file manually
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.EnsureLogsDir("logs-test")
	logPath := registry.GetLogPath("logs-test")
	_ = os.WriteFile(logPath, []byte("line1\nline2\nline3\n"), 0600)

	_ = registry.Register(&process.Process{
		Name:    "logs-test",
		Command: "test",
		PID:     1234,
	})

	output, err := executeCmd("process", "logs", "logs-test")
	if err != nil {
		t.Fatalf("process logs failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "line1") {
		t.Errorf("output should contain log content: %s", output)
	}
}

func TestProcessLogsEmpty(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.Register(&process.Process{
		Name:    "empty-logs",
		Command: "test",
		PID:     1234,
	})

	output, err := executeCmd("process", "logs", "empty-logs")
	if err != nil {
		t.Fatalf("process logs failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No logs") {
		t.Errorf("output should indicate no logs: %s", output)
	}
}

func TestProcessLogsNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "logs", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessLogsTail(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create log file with many lines
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.EnsureLogsDir("tail-test")
	logPath := registry.GetLogPath("tail-test")

	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, "line"+string(rune('0'+(i%10))))
	}
	_ = os.WriteFile(logPath, []byte(strings.Join(lines, "\n")+"\n"), 0600)

	_ = registry.Register(&process.Process{
		Name:    "tail-test",
		Command: "test",
		PID:     1234,
	})

	output, err := executeCmd("process", "logs", "tail-test", "--tail", "10")
	if err != nil {
		t.Fatalf("process logs failed: %v\nOutput: %s", err, output)
	}
	// Should have fewer lines than original
	outputLines := strings.Split(strings.TrimSpace(output), "\n")
	if len(outputLines) > 15 {
		t.Errorf("expected around 10 lines, got %d", len(outputLines))
	}
}

func TestProcessShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.Register(&process.Process{
		Name:    "show-test",
		Command: "echo hello",
		PID:     1234,
		Port:    3000,
		Owner:   "test-agent",
	})

	output, err := executeCmd("process", "show", "show-test")
	if err != nil {
		t.Fatalf("process show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "show-test") {
		t.Errorf("output should contain name: %s", output)
	}
	if !strings.Contains(output, "echo hello") {
		t.Errorf("output should contain command: %s", output)
	}
	if !strings.Contains(output, "1234") {
		t.Errorf("output should contain PID: %s", output)
	}
	if !strings.Contains(output, "3000") {
		t.Errorf("output should contain port: %s", output)
	}
	if !strings.Contains(output, "LogFile") {
		t.Errorf("output should contain log file path: %s", output)
	}
}

func TestProcessShowNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "show", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessAttachNotRunning(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.Register(&process.Process{
		Name:    "stopped-proc",
		Command: "echo",
		PID:     0,
		Running: false,
	})
	_ = registry.MarkStopped("stopped-proc")

	_, err := executeCmd("process", "attach", "stopped-proc")
	if err == nil {
		t.Error("expected error for stopped process")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error should mention not running: %v", err)
	}
}

func TestProcessAttachNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "attach", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}
