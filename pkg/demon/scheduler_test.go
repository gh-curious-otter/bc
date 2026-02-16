package demon

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewScheduler(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	if scheduler.rootDir != tmpDir {
		t.Errorf("rootDir = %q, want %q", scheduler.rootDir, tmpDir)
	}

	expectedDemonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if scheduler.demonsDir != expectedDemonsDir {
		t.Errorf("demonsDir = %q, want %q", scheduler.demonsDir, expectedDemonsDir)
	}

	expectedPidFile := filepath.Join(expectedDemonsDir, "scheduler.pid")
	if scheduler.pidFile != expectedPidFile {
		t.Errorf("pidFile = %q, want %q", scheduler.pidFile, expectedPidFile)
	}
}

func TestSchedulerStatus_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	status, err := scheduler.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.Running {
		t.Error("Status.Running = true, want false")
	}

	if status.PID != 0 {
		t.Errorf("Status.PID = %d, want 0", status.PID)
	}
}

func TestSchedulerIsRunning_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	if scheduler.IsRunning() {
		t.Error("IsRunning() = true, want false")
	}
}

func TestSchedulerIsRunning_StalePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory and stale PID file
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Write a PID that doesn't exist
	pidFile := filepath.Join(demonsDir, "scheduler.pid")
	if err := os.WriteFile(pidFile, []byte("99999999"), 0600); err != nil {
		t.Fatal(err)
	}

	// Should return false for stale PID
	if scheduler.IsRunning() {
		t.Error("IsRunning() = true, want false for stale PID")
	}
}

func TestSchedulerStatus_StalePIDCleansUp(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory and stale PID file
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	pidFile := filepath.Join(demonsDir, "scheduler.pid")
	if err := os.WriteFile(pidFile, []byte("99999999"), 0600); err != nil {
		t.Fatal(err)
	}

	status, err := scheduler.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if status.Running {
		t.Error("Status.Running = true, want false")
	}

	// PID file should be cleaned up
	if _, err := os.Stat(pidFile); !os.IsNotExist(err) {
		t.Error("Stale PID file should be removed")
	}
}

func TestSchedulerStop_NotRunning(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	err := scheduler.Stop()
	if err == nil {
		t.Error("Stop() should return error when not running")
	}
}

func TestSchedulerStart_AlreadyRunning(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory and PID file with current process PID
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Use current process PID to simulate running scheduler
	pid := os.Getpid()
	pidFile := filepath.Join(demonsDir, "scheduler.pid")
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0600); err != nil {
		t.Fatal(err)
	}

	err := scheduler.Start()
	if err == nil {
		t.Fatal("Start() should return error when already running")
	}
	if err.Error() != "scheduler is already running" {
		t.Errorf("Start() error = %q, want 'scheduler is already running'", err.Error())
	}
}

func TestSchedulerGetNextRuns(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a demon
	store := NewStore(tmpDir)
	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	runs, err := scheduler.GetNextRuns()
	if err != nil {
		t.Fatalf("GetNextRuns() error = %v", err)
	}

	if len(runs) != 1 {
		t.Fatalf("GetNextRuns() returned %d runs, want 1", len(runs))
	}

	if runs[0].Name != "test-demon" {
		t.Errorf("NextRun.Name = %q, want 'test-demon'", runs[0].Name)
	}

	if runs[0].Command != "echo hello" {
		t.Errorf("NextRun.Command = %q, want 'echo hello'", runs[0].Command)
	}

	if runs[0].NextRun.IsZero() {
		t.Error("NextRun.NextRun is zero")
	}
}

func TestSchedulerStatusSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Save status
	status := SchedulerStatus{
		Running:   true,
		PID:       12345,
		StartedAt: time.Now().UTC(),
	}
	if err := scheduler.saveStatus(status); err != nil {
		t.Fatalf("saveStatus() error = %v", err)
	}

	// Load status
	loaded, err := scheduler.loadStatus()
	if err != nil {
		t.Fatalf("loadStatus() error = %v", err)
	}

	if loaded.PID != status.PID {
		t.Errorf("loaded.PID = %d, want %d", loaded.PID, status.PID)
	}
}

func TestDemonNextRun(t *testing.T) {
	run := DemonNextRun{
		Name:    "test",
		NextRun: time.Now().Add(time.Hour),
		Command: "echo test",
	}

	if run.Name != "test" {
		t.Errorf("Name = %q, want 'test'", run.Name)
	}

	if run.Command != "echo test" {
		t.Errorf("Command = %q, want 'echo test'", run.Command)
	}

	if run.NextRun.IsZero() {
		t.Error("NextRun should not be zero")
	}
}
