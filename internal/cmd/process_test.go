package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/process"
)

func TestProcessStart(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	output, err := executeCmd("process", "start", "test-proc", "--cmd", "sleep 60")
	if err != nil {
		t.Fatalf("process start failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Started process") {
		t.Errorf("expected confirmation message, got: %s", output)
	}
	if !strings.Contains(output, "test-proc") {
		t.Errorf("output should contain process name: %s", output)
	}
	if !strings.Contains(output, "PID") {
		t.Errorf("output should contain PID: %s", output)
	}

	// Verify process was registered
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	p := registry.Get("test-proc")
	if p == nil {
		t.Fatal("process not found in registry")
	}
	if p.Command != "sleep 60" {
		t.Errorf("unexpected command: %s", p.Command)
	}
	if !p.Running {
		t.Error("process should be marked as running")
	}

	// Cleanup: stop the process
	_, _ = executeCmd("process", "stop", "test-proc")
}

func TestProcessStartWithPort(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	output, err := executeCmd("process", "start", "port-proc", "--cmd", "sleep 60", "--port", "8080")
	if err != nil {
		t.Fatalf("process start failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Port:") {
		t.Errorf("output should show port: %s", output)
	}

	// Verify port was recorded
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	p := registry.Get("port-proc")
	if p.Port != 8080 {
		t.Errorf("port = %d, want 8080", p.Port)
	}

	// Cleanup
	_, _ = executeCmd("process", "stop", "port-proc")
}

func TestProcessStartDuplicate(t *testing.T) {
	setupTestWorkspace(t)

	// Start first process
	_, err := executeCmd("process", "start", "dup-proc", "--cmd", "sleep 60")
	if err != nil {
		t.Fatalf("first start failed: %v", err)
	}

	// Try to start duplicate
	_, err = executeCmd("process", "start", "dup-proc", "--cmd", "sleep 60")
	if err == nil {
		t.Error("expected error for duplicate process")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error should mention already running: %v", err)
	}

	// Cleanup
	_, _ = executeCmd("process", "stop", "dup-proc")
}

func TestProcessStartPortConflict(t *testing.T) {
	setupTestWorkspace(t)

	// Start first process with port
	_, err := executeCmd("process", "start", "port1", "--cmd", "sleep 60", "--port", "9000")
	if err != nil {
		t.Fatalf("first start failed: %v", err)
	}

	// Try to start another with same port
	_, err = executeCmd("process", "start", "port2", "--cmd", "sleep 60", "--port", "9000")
	if err == nil {
		t.Error("expected error for port conflict")
	}
	if !strings.Contains(err.Error(), "already in use") {
		t.Errorf("error should mention port in use: %v", err)
	}

	// Cleanup
	_, _ = executeCmd("process", "stop", "port1")
}

func TestProcessList(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Register some processes
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.Register(&process.Process{
		Name:    "proc1",
		Command: "echo one",
		PID:     1001,
	})
	_ = registry.Register(&process.Process{
		Name:    "proc2",
		Command: "echo two",
		PID:     1002,
		Port:    8080,
	})

	output, err := executeCmd("process", "list")
	if err != nil {
		t.Fatalf("process list failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "proc1") {
		t.Errorf("output should contain proc1: %s", output)
	}
	if !strings.Contains(output, "proc2") {
		t.Errorf("output should contain proc2: %s", output)
	}
	if !strings.Contains(output, "NAME") {
		t.Errorf("output should contain header: %s", output)
	}
}

func TestProcessListEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("process", "list")
	if err != nil {
		t.Fatalf("process list failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No processes registered") {
		t.Errorf("output should indicate no processes: %s", output)
	}
}

func TestProcessStop(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Start a process
	_, err := executeCmd("process", "start", "stop-proc", "--cmd", "sleep 60")
	if err != nil {
		t.Fatalf("start failed: %v", err)
	}

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop it
	output, err := executeCmd("process", "stop", "stop-proc")
	if err != nil {
		t.Fatalf("process stop failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Stopped") || !strings.Contains(output, "stop-proc") {
		t.Errorf("output should confirm stop: %s", output)
	}

	// Verify it's marked as stopped
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	p := registry.Get("stop-proc")
	if p.Running {
		t.Error("process should be marked as stopped")
	}
}

func TestProcessStopNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("process", "stop", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent process")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestProcessShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Register a process
	registry := process.NewRegistry(wsDir)
	_ = registry.Init()
	_ = registry.Register(&process.Process{
		Name:    "show-proc",
		Command: "echo hello",
		PID:     1234,
		Port:    3000,
		Owner:   "test-agent",
	})

	output, err := executeCmd("process", "show", "show-proc")
	if err != nil {
		t.Fatalf("process show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "show-proc") {
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
	if !strings.Contains(output, "test-agent") {
		t.Errorf("output should contain owner: %s", output)
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
