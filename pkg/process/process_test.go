package process

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.processes == nil {
		t.Error("processes map should be initialized")
	}
}

func TestManager_StartStop(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Start a simple process (sleep)
	proc, err := mgr.Start("test-proc", "sleep", []string{"10"}, "", "engineer-01")
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}

	if proc.Name != "test-proc" {
		t.Errorf("Name = %q, want %q", proc.Name, "test-proc")
	}
	if proc.State != StateRunning {
		t.Errorf("State = %v, want %v", proc.State, StateRunning)
	}
	if proc.PID == 0 {
		t.Error("PID should be non-zero")
	}
	if proc.Owner != "engineer-01" {
		t.Errorf("Owner = %q, want %q", proc.Owner, "engineer-01")
	}

	// Stop the process
	if err := mgr.Stop("test-proc"); err != nil {
		t.Fatalf("failed to stop process: %v", err)
	}

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Verify stopped
	proc, ok := mgr.Get("test-proc")
	if !ok {
		t.Fatal("process should still exist after stop")
	}
	if proc.State == StateRunning {
		t.Error("process should not be running after stop")
	}
}

func TestManager_StartDuplicate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Start first process
	_, err := mgr.Start("test-proc", "sleep", []string{"10"}, "", "")
	if err != nil {
		t.Fatalf("failed to start first process: %v", err)
	}
	defer func() { _ = mgr.Stop("test-proc") }()

	// Try to start duplicate
	_, err = mgr.Start("test-proc", "sleep", []string{"10"}, "", "")
	if err == nil {
		t.Error("expected error when starting duplicate process")
	}
}

func TestManager_StopNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.Stop("nonexistent")
	if err == nil {
		t.Error("expected error when stopping nonexistent process")
	}
}

func TestManager_Get(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Get nonexistent
	_, ok := mgr.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent process")
	}

	// Start a process
	_, err := mgr.Start("test-proc", "sleep", []string{"10"}, "", "")
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	defer func() { _ = mgr.Stop("test-proc") }()

	// Get existing
	proc, ok := mgr.Get("test-proc")
	if !ok {
		t.Error("expected true for existing process")
	}
	if proc.Name != "test-proc" {
		t.Errorf("Name = %q, want %q", proc.Name, "test-proc")
	}
}

func TestManager_List(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Empty list
	procs := mgr.List()
	if len(procs) != 0 {
		t.Errorf("expected 0 processes, got %d", len(procs))
	}

	// Start some processes
	_, _ = mgr.Start("proc1", "sleep", []string{"10"}, "", "")
	_, _ = mgr.Start("proc2", "sleep", []string{"10"}, "", "")
	defer func() {
		_ = mgr.Stop("proc1")
		_ = mgr.Stop("proc2")
	}()

	procs = mgr.List()
	if len(procs) != 2 {
		t.Errorf("expected 2 processes, got %d", len(procs))
	}
}

func TestManager_ListRunning(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Start and stop one process
	_, _ = mgr.Start("stopped-proc", "sleep", []string{"10"}, "", "")
	_ = mgr.Stop("stopped-proc")

	// Start a running process
	_, _ = mgr.Start("running-proc", "sleep", []string{"10"}, "", "")
	defer func() { _ = mgr.Stop("running-proc") }()

	running := mgr.ListRunning()
	if len(running) != 1 {
		t.Errorf("expected 1 running process, got %d", len(running))
	}
	if running[0].Name != "running-proc" {
		t.Errorf("expected running-proc, got %s", running[0].Name)
	}
}

func TestManager_RunningCount(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	if mgr.RunningCount() != 0 {
		t.Errorf("expected 0 running, got %d", mgr.RunningCount())
	}

	_, _ = mgr.Start("proc1", "sleep", []string{"10"}, "", "")
	_, _ = mgr.Start("proc2", "sleep", []string{"10"}, "", "")
	defer func() {
		_ = mgr.Stop("proc1")
		_ = mgr.Stop("proc2")
	}()

	if mgr.RunningCount() != 2 {
		t.Errorf("expected 2 running, got %d", mgr.RunningCount())
	}
}

func TestManager_SaveLoadState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on Windows")
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Start a process
	_, err := mgr.Start("test-proc", "sleep", []string{"10"}, "", "owner1")
	if err != nil {
		t.Fatalf("failed to start process: %v", err)
	}
	defer func() { _ = mgr.Stop("test-proc") }()

	// Verify state file was created
	statePath := filepath.Join(tmpDir, ".bc", "processes", "processes.json")
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("state file should exist")
	}

	// Create new manager and load state
	mgr2 := NewManager(tmpDir)
	if err := mgr2.LoadState(); err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	proc, ok := mgr2.Get("test-proc")
	if !ok {
		t.Error("process should be loaded from state")
	}
	if proc.Owner != "owner1" {
		t.Errorf("Owner = %q, want %q", proc.Owner, "owner1")
	}
}

func TestManager_LoadStateEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Load from nonexistent file should not error
	if err := mgr.LoadState(); err != nil {
		t.Errorf("LoadState from empty should not error: %v", err)
	}
}

func TestProcess_isAlive(t *testing.T) {
	// Process with PID 0 is not alive
	p := &Process{PID: 0}
	if p.isAlive() {
		t.Error("PID 0 should not be alive")
	}

	// Current process should be alive
	p = &Process{PID: os.Getpid()}
	if !p.isAlive() {
		t.Error("current process should be alive")
	}

	// Non-existent high PID should not be alive
	p = &Process{PID: 999999999}
	if p.isAlive() {
		t.Error("non-existent PID should not be alive")
	}
}

func TestManager_RefreshState(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Add a process with fake PID that's not running
	mgr.processes["fake"] = &Process{
		Name:  "fake",
		PID:   999999999,
		State: StateRunning,
	}

	// Refresh should mark it as stopped
	if err := mgr.RefreshState(); err != nil {
		t.Fatalf("RefreshState failed: %v", err)
	}

	proc, _ := mgr.Get("fake")
	if proc.State != StateStopped {
		t.Errorf("State = %v, want %v", proc.State, StateStopped)
	}
}
