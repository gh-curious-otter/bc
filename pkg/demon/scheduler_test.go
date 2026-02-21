package demon

import (
	"context"
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

func TestSchedulerLoadStatusNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// loadStatus without creating status file
	_, err := scheduler.loadStatus()
	if err == nil {
		t.Error("loadStatus() should error when status file doesn't exist")
	}
}

func TestSchedulerLoadStatusInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory with invalid JSON
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	statusFile := filepath.Join(demonsDir, "scheduler.json")
	if err := os.WriteFile(statusFile, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := scheduler.loadStatus()
	if err == nil {
		t.Error("loadStatus() should error on invalid JSON")
	}
}

func TestSchedulerStatusStruct(t *testing.T) {
	now := time.Now().UTC()
	status := SchedulerStatus{
		Running:   true,
		PID:       12345,
		StartedAt: now,
		Uptime:    "1h30m",
	}

	if !status.Running {
		t.Error("Running should be true")
	}
	if status.PID != 12345 {
		t.Errorf("PID = %d, want 12345", status.PID)
	}
	if status.Uptime != "1h30m" {
		t.Errorf("Uptime = %q, want '1h30m'", status.Uptime)
	}
	if status.StartedAt != now {
		t.Errorf("StartedAt = %v, want %v", status.StartedAt, now)
	}
}

func TestSchedulerReadPIDNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// readPID without PID file
	_, err := scheduler.readPID()
	if err == nil {
		t.Error("readPID() should error when PID file doesn't exist")
	}
}

func TestSchedulerReadPIDInvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory with invalid PID file
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	pidFile := filepath.Join(demonsDir, "scheduler.pid")
	if err := os.WriteFile(pidFile, []byte("not-a-number"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := scheduler.readPID()
	if err == nil {
		t.Error("readPID() should error on invalid PID content")
	}
}

func TestSchedulerProcessRunningInvalidPID(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Process with PID -1 should not be running
	if scheduler.processRunning(-1) {
		t.Error("processRunning(-1) should return false")
	}

	// Process with PID 0 should not be running (or may be kernel)
	// This test is platform-dependent, so we just verify no panic
	_ = scheduler.processRunning(0)
}

// --- Additional tests for runtime functions and edge cases ---

func TestGetRunLogsWithMalformedLines(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create demons directory
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Write log file with mix of valid and invalid lines
	logPath := filepath.Join(demonsDir, "test-demon.log.jsonl")
	logContent := `{"timestamp":"2024-01-15T10:00:00Z","duration_ms":100,"exit_code":0,"success":true}
invalid json line
{"timestamp":"2024-01-15T11:00:00Z","duration_ms":200,"exit_code":0,"success":true}
`
	if err := os.WriteFile(logPath, []byte(logContent), 0600); err != nil {
		t.Fatal(err)
	}

	// Should skip invalid lines and return valid ones
	logs, err := store.GetRunLogs("test-demon", 0)
	if err != nil {
		t.Fatalf("GetRunLogs() error = %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("GetRunLogs() returned %d logs, want 2", len(logs))
	}
}

func TestSchedulerSaveStatusDirCreation(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Demons directory doesn't exist yet
	status := SchedulerStatus{
		Running:   true,
		PID:       12345,
		StartedAt: time.Now().UTC(),
	}

	// saveStatus should fail if directory doesn't exist
	err := scheduler.saveStatus(status)
	if err == nil {
		t.Error("saveStatus should fail when directory doesn't exist")
	}
}

func TestStoreListReadsDir(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create a valid demon
	_, err := store.Create("demon1", "0 * * * *", "echo one")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Create non-json file (should be skipped)
	nonJsonPath := filepath.Join(store.demonsDir, "readme.txt")
	if writeErr := os.WriteFile(nonJsonPath, []byte("readme"), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Create directory (should be skipped)
	subDir := filepath.Join(store.demonsDir, "subdir.json")
	if mkdirErr := os.Mkdir(subDir, 0750); mkdirErr != nil {
		t.Fatal(mkdirErr)
	}

	demons, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(demons) != 1 {
		t.Errorf("List returned %d demons, want 1", len(demons))
	}
}

func TestCronScheduleNextNoMatch(t *testing.T) {
	// Create a cron schedule that matches specific conditions
	cron := &CronSchedule{
		Minute:     []int{0},
		Hour:       []int{0},
		DayOfMonth: []int{31},
		Month:      []int{2},
		DayOfWeek:  []int{0, 1, 2, 3, 4, 5, 6},
	}

	// Feb 31 never exists, so after exhausting iterations it returns zero
	after := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	next := cron.Next(after)

	if !next.IsZero() {
		t.Errorf("Next() for impossible schedule should return zero time, got %v", next)
	}
}

func TestStoreDeleteError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Try to delete from non-existent directory
	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("Delete should fail for non-existent demon")
	}
}

func TestStoreGetReadError(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create demons directory
	if err := os.MkdirAll(store.demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a directory with the same name as a demon file
	dirPath := filepath.Join(store.demonsDir, "dir-demon.json")
	if err := os.Mkdir(dirPath, 0750); err != nil {
		t.Fatal(err)
	}

	_, err := store.Get("dir-demon")
	if err == nil {
		t.Error("Get should fail when path is a directory")
	}
}

// --- Runtime function tests for checkAndRunDemons, runDemon, and log ---

func TestSchedulerCheckAndRunDemonsNoDue(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Create a demon with next run far in the future
	_, err := store.Create("future-demon", "0 0 1 1 *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// This should not run any demons (none are due)
	scheduler.checkAndRunDemons(store)

	// Verify demon was not run
	demon, _ := store.Get("future-demon")
	if demon.RunCount != 0 {
		t.Errorf("RunCount = %d, want 0", demon.RunCount)
	}
}

func TestSchedulerCheckAndRunDemonsWithDue(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Create a demon
	_, err := store.Create("due-demon", "* * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Manually set NextRun to the past to make it due
	err = store.Update("due-demon", func(d *Demon) {
		d.NextRun = time.Now().Add(-time.Hour)
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Run check - this should execute the demon
	scheduler.checkAndRunDemons(store)

	// Give it a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Verify demon was run
	demon, _ := store.Get("due-demon")
	if demon.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", demon.RunCount)
	}
	if demon.LastRun.IsZero() {
		t.Error("LastRun should be set")
	}
}

func TestSchedulerRunDemonSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Create a demon with a simple command
	demon, err := store.Create("run-test", "0 * * * *", "echo success")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Run the demon directly
	scheduler.runDemon(store, demon)

	// Verify run was recorded
	got, _ := store.Get("run-test")
	if got.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", got.RunCount)
	}

	// Verify log was recorded
	logs, _ := store.GetRunLogs("run-test", 0)
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
	if len(logs) > 0 && !logs[0].Success {
		t.Error("Log should show success")
	}
}

func TestSchedulerRunDemonFailure(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Create a demon with a failing command
	demon, err := store.Create("fail-test", "0 * * * *", "exit 1")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Run the demon
	scheduler.runDemon(store, demon)

	// Verify log was recorded with failure
	logs, _ := store.GetRunLogs("fail-test", 0)
	if len(logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(logs))
	}
	if len(logs) > 0 {
		if logs[0].Success {
			t.Error("Log should show failure")
		}
		if logs[0].ExitCode != 1 {
			t.Errorf("ExitCode = %d, want 1", logs[0].ExitCode)
		}
	}
}

func TestSchedulerCheckAndRunDemonsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Initialize store but don't create any demons
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	// Should not panic on empty store
	scheduler.checkAndRunDemons(store)
}

func TestSchedulerLog(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Just verify log doesn't panic
	scheduler.log("Test message: %s", "hello")
	scheduler.log("Test number: %d", 42)
}

func TestSchedulerRunLoopCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Initialize store
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	// Create a demon to be processed
	_, err := store.Create("loop-test", "* * * * *", "echo test")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Set NextRun to past so it runs immediately
	err = store.Update("loop-test", func(d *Demon) {
		d.NextRun = time.Now().Add(-time.Hour)
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Run loop with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// RunLoop should return context.DeadlineExceeded
	err = scheduler.RunLoop(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("RunLoop() error = %v, want context.DeadlineExceeded", err)
	}

	// Verify the demon was executed during the loop
	demon, _ := store.Get("loop-test")
	if demon.RunCount < 1 {
		t.Errorf("RunCount = %d, want >= 1", demon.RunCount)
	}
}

func TestSchedulerRunLoopEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)
	store := NewStore(tmpDir)

	// Initialize store but no demons
	if err := store.Init(); err != nil {
		t.Fatal(err)
	}

	// Run loop with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := scheduler.RunLoop(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("RunLoop() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestSchedulerStatusRunning(t *testing.T) {
	tmpDir := t.TempDir()
	scheduler := NewScheduler(tmpDir)

	// Create demons directory
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Use current process PID
	pid := os.Getpid()
	pidFile := filepath.Join(demonsDir, "scheduler.pid")
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0600); err != nil {
		t.Fatal(err)
	}

	// Write status file
	status := SchedulerStatus{
		Running:   true,
		PID:       pid,
		StartedAt: time.Now().Add(-30 * time.Minute).UTC(),
	}
	if err := scheduler.saveStatus(status); err != nil {
		t.Fatal(err)
	}

	// Get status
	got, err := scheduler.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}

	if !got.Running {
		t.Error("Status.Running should be true")
	}
	if got.PID != pid {
		t.Errorf("Status.PID = %d, want %d", got.PID, pid)
	}
	if got.Uptime == "" {
		t.Error("Status.Uptime should be set")
	}
	if got.StartedAt.IsZero() {
		t.Error("Status.StartedAt should be set")
	}
}
