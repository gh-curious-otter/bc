package demon

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCron(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"every minute", "* * * * *", false},
		{"every hour", "0 * * * *", false},
		{"daily at 9am", "0 9 * * *", false},
		{"every 5 minutes", "*/5 * * * *", false},
		{"weekdays at 5pm", "0 17 * * 1-5", false},
		{"specific minutes", "0,15,30,45 * * * *", false},
		{"too few fields", "* * * *", true},
		{"too many fields", "* * * * * *", true},
		{"invalid minute", "60 * * * *", true},
		{"invalid hour", "0 24 * * *", true},
		{"invalid day", "0 0 32 * *", true},
		{"invalid month", "0 0 * 13 *", true},
		{"invalid weekday", "0 0 * * 7", true},
		{"invalid step", "*/0 * * * *", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCron(%q) error = %v, wantErr %v", tt.expr, err, tt.wantErr)
			}
		})
	}
}

func TestCronScheduleNext(t *testing.T) {
	// Test "every hour at minute 0"
	cron, err := ParseCron("0 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	// From 2024-01-15 10:30:00, next should be 2024-01-15 11:00:00
	after := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	next := cron.Next(after)
	expected := time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC)

	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronScheduleNextEvery5Min(t *testing.T) {
	cron, err := ParseCron("*/5 * * * *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	// From 10:32, next should be 10:35
	after := time.Date(2024, 1, 15, 10, 32, 0, 0, time.UTC)
	next := cron.Next(after)
	expected := time.Date(2024, 1, 15, 10, 35, 0, 0, time.UTC)

	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronScheduleNextWeekday(t *testing.T) {
	// Weekdays at 9am (Monday-Friday)
	cron, err := ParseCron("0 9 * * 1-5")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	// Saturday 2024-01-13 at 10am, next should be Monday 2024-01-15 at 9am
	after := time.Date(2024, 1, 13, 10, 0, 0, 0, time.UTC) // Saturday
	next := cron.Next(after)
	expected := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC) // Monday

	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestStoreCreateAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	demon, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if demon.Name != "test-demon" {
		t.Errorf("Name = %q, want %q", demon.Name, "test-demon")
	}
	if demon.Schedule != "0 * * * *" {
		t.Errorf("Schedule = %q, want %q", demon.Schedule, "0 * * * *")
	}
	if demon.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", demon.Command, "echo hello")
	}
	if !demon.Enabled {
		t.Error("Enabled should be true")
	}
	if demon.NextRun.IsZero() {
		t.Error("NextRun should be set")
	}

	// Get the demon
	got, err := store.Get("test-demon")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != demon.Name {
		t.Errorf("Got Name = %q, want %q", got.Name, demon.Name)
	}
}

func TestStoreCreateDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("First Create failed: %v", err)
	}

	_, err = store.Create("test-demon", "0 * * * *", "echo world")
	if err == nil {
		t.Error("Expected error for duplicate demon")
	}
}

func TestStoreCreateInvalidCron(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "invalid", "echo hello")
	if err == nil {
		t.Error("Expected error for invalid cron")
	}
}

func TestStoreList(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create some demons
	_, _ = store.Create("demon1", "0 * * * *", "echo one")
	_, _ = store.Create("demon2", "*/5 * * * *", "echo two")

	demons, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(demons) != 2 {
		t.Errorf("List returned %d demons, want 2", len(demons))
	}
}

func TestStoreListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	demons, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(demons) != 0 {
		t.Errorf("List returned %d demons, want 0", len(demons))
	}
}

func TestStoreDelete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if !store.Exists("test-demon") {
		t.Error("Demon should exist before delete")
	}

	err = store.Delete("test-demon")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	if store.Exists("test-demon") {
		t.Error("Demon should not exist after delete")
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("Expected error for deleting nonexistent demon")
	}
}

func TestStoreGetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	got, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got != nil {
		t.Error("Get should return nil for nonexistent demon")
	}
}

func TestDemonPath(t *testing.T) {
	store := NewStore("/tmp/test")
	expected := filepath.Join("/tmp/test", ".bc", "demons", "my-demon.json")
	got := store.demonPath("my-demon")
	if got != expected {
		t.Errorf("demonPath = %q, want %q", got, expected)
	}
}

func TestParseFieldSingleValue(t *testing.T) {
	vals, err := parseField("5", 0, 59)
	if err != nil {
		t.Fatalf("parseField failed: %v", err)
	}
	if len(vals) != 1 || vals[0] != 5 {
		t.Errorf("parseField(\"5\") = %v, want [5]", vals)
	}
}

func TestParseFieldRange(t *testing.T) {
	vals, err := parseField("1-5", 0, 10)
	if err != nil {
		t.Fatalf("parseField failed: %v", err)
	}
	expected := []int{1, 2, 3, 4, 5}
	if len(vals) != len(expected) {
		t.Fatalf("parseField(\"1-5\") len = %d, want %d", len(vals), len(expected))
	}
	for i, v := range vals {
		if v != expected[i] {
			t.Errorf("parseField(\"1-5\")[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestParseFieldComma(t *testing.T) {
	vals, err := parseField("0,15,30,45", 0, 59)
	if err != nil {
		t.Fatalf("parseField failed: %v", err)
	}
	expected := []int{0, 15, 30, 45}
	if len(vals) != len(expected) {
		t.Fatalf("parseField len = %d, want %d", len(vals), len(expected))
	}
	for i, v := range vals {
		if v != expected[i] {
			t.Errorf("parseField[%d] = %d, want %d", i, v, expected[i])
		}
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if _, err := os.Stat(demonsDir); os.IsNotExist(err) {
		t.Errorf("Demons directory not created: %s", demonsDir)
	}
}

func TestStoreUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.Update("test-demon", func(d *Demon) {
		d.Command = "echo updated"
		d.Description = "test description"
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := store.Get("test-demon")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Command != "echo updated" {
		t.Errorf("Command = %q, want %q", got.Command, "echo updated")
	}
	if got.Description != "test description" {
		t.Errorf("Description = %q, want %q", got.Description, "test description")
	}
}

func TestStoreUpdateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Update("nonexistent", func(d *Demon) {
		d.Command = "test"
	})
	if err == nil {
		t.Error("Expected error for updating nonexistent demon")
	}
}

func TestStoreListByOwner(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create demons with different owners
	d1, _ := store.Create("demon1", "0 * * * *", "echo one")
	_ = store.SetOwner(d1.Name, "engineer-01")

	d2, _ := store.Create("demon2", "*/5 * * * *", "echo two")
	_ = store.SetOwner(d2.Name, "engineer-01")

	d3, _ := store.Create("demon3", "0 0 * * *", "echo three")
	_ = store.SetOwner(d3.Name, "qa-01")

	// List by owner
	eng01Demons, err := store.ListByOwner("engineer-01")
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(eng01Demons) != 2 {
		t.Errorf("ListByOwner(engineer-01) len = %d, want 2", len(eng01Demons))
	}

	qa01Demons, err := store.ListByOwner("qa-01")
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(qa01Demons) != 1 {
		t.Errorf("ListByOwner(qa-01) len = %d, want 1", len(qa01Demons))
	}

	// No demons for this owner
	noneDemon, err := store.ListByOwner("unknown")
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(noneDemon) != 0 {
		t.Errorf("ListByOwner(unknown) len = %d, want 0", len(noneDemon))
	}
}

func TestStoreListEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, _ = store.Create("enabled1", "0 * * * *", "echo one")
	_, _ = store.Create("enabled2", "*/5 * * * *", "echo two")
	d3, _ := store.Create("disabled1", "0 0 * * *", "echo three")
	_ = store.Disable(d3.Name)

	enabled, err := store.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled failed: %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("ListEnabled len = %d, want 2", len(enabled))
	}

	// All should be enabled
	for _, d := range enabled {
		if !d.Enabled {
			t.Errorf("Demon %q should be enabled", d.Name)
		}
	}
}

func TestStoreEnableDisable(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Disable
	err = store.Disable("test-demon")
	if err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	got, _ := store.Get("test-demon")
	if got.Enabled {
		t.Error("Demon should be disabled")
	}

	// Enable
	err = store.Enable("test-demon")
	if err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	got, _ = store.Get("test-demon")
	if !got.Enabled {
		t.Error("Demon should be enabled")
	}
}

func TestStoreRecordRun(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Record a run
	err = store.RecordRun("test-demon")
	if err != nil {
		t.Fatalf("RecordRun failed: %v", err)
	}

	got, _ := store.Get("test-demon")
	if got.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", got.RunCount)
	}
	if got.LastRun.IsZero() {
		t.Error("LastRun should be set")
	}
	if got.NextRun.IsZero() {
		t.Error("NextRun should be set")
	}

	// Record another run
	err = store.RecordRun("test-demon")
	if err != nil {
		t.Fatalf("Second RecordRun failed: %v", err)
	}
	got, _ = store.Get("test-demon")
	if got.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2", got.RunCount)
	}
}

func TestStoreSetOwner(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.SetOwner("test-demon", "engineer-01")
	if err != nil {
		t.Fatalf("SetOwner failed: %v", err)
	}

	got, _ := store.Get("test-demon")
	if got.Owner != "engineer-01" {
		t.Errorf("Owner = %q, want %q", got.Owner, "engineer-01")
	}
}

func TestStoreSetDescription(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err = store.SetDescription("test-demon", "This is a test demon")
	if err != nil {
		t.Fatalf("SetDescription failed: %v", err)
	}

	got, _ := store.Get("test-demon")
	if got.Description != "This is a test demon" {
		t.Errorf("Description = %q, want %q", got.Description, "This is a test demon")
	}
}

func TestDemonOwnerField(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	demon, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Initially no owner
	if demon.Owner != "" {
		t.Errorf("Initial Owner = %q, want empty", demon.Owner)
	}

	// Set owner
	_ = store.SetOwner("test-demon", "engineer-01")

	// Verify persistence
	got, _ := store.Get("test-demon")
	if got.Owner != "engineer-01" {
		t.Errorf("Owner after reload = %q, want %q", got.Owner, "engineer-01")
	}
}

func TestRecordRunLog(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create a demon first
	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Record a successful run
	log1 := RunLog{
		Timestamp: time.Now().UTC(),
		Duration:  150,
		ExitCode:  0,
		Success:   true,
	}
	if recordErr := store.RecordRunLog("test-demon", log1); recordErr != nil {
		t.Fatalf("RecordRunLog failed: %v", recordErr)
	}

	// Record a failed run
	log2 := RunLog{
		Timestamp: time.Now().UTC(),
		Duration:  2500,
		ExitCode:  1,
		Success:   false,
	}
	if recordErr := store.RecordRunLog("test-demon", log2); recordErr != nil {
		t.Fatalf("RecordRunLog failed: %v", recordErr)
	}

	// Get all logs
	logs, err := store.GetRunLogs("test-demon", 0)
	if err != nil {
		t.Fatalf("GetRunLogs failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(logs))
	}

	// Verify first log
	if logs[0].Duration != 150 {
		t.Errorf("First log duration = %d, want 150", logs[0].Duration)
	}
	if !logs[0].Success {
		t.Error("First log should be successful")
	}

	// Verify second log
	if logs[1].ExitCode != 1 {
		t.Errorf("Second log exit code = %d, want 1", logs[1].ExitCode)
	}
	if logs[1].Success {
		t.Error("Second log should be failed")
	}
}

func TestGetRunLogsWithLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	_, err := store.Create("test-demon", "0 * * * *", "echo hello")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Record 5 runs
	for i := 0; i < 5; i++ {
		log := RunLog{
			Timestamp: time.Now().UTC(),
			Duration:  int64(i * 100),
			ExitCode:  0,
			Success:   true,
		}
		if recordErr := store.RecordRunLog("test-demon", log); recordErr != nil {
			t.Fatalf("RecordRunLog failed: %v", recordErr)
		}
	}

	// Get with limit
	logs, err := store.GetRunLogs("test-demon", 3)
	if err != nil {
		t.Fatalf("GetRunLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 logs with limit, got %d", len(logs))
	}

	// Should be the most recent ones (200ms, 300ms, 400ms)
	if logs[0].Duration != 200 {
		t.Errorf("First limited log duration = %d, want 200", logs[0].Duration)
	}
}

func TestGetRunLogsNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Get logs for non-existent demon (no log file)
	logs, err := store.GetRunLogs("nonexistent", 0)
	if err != nil {
		t.Errorf("GetRunLogs should not error for nonexistent: %v", err)
	}
	if logs != nil {
		t.Errorf("Expected nil logs, got %v", logs)
	}
}

func TestCreateWithPromptBothOptions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create prompt file for testing
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte("test prompt"), 0600); err != nil {
		t.Fatalf("failed to create prompt file: %v", err)
	}

	// Both prompt and prompt-file specified should error
	_, err := store.CreateWithPrompt("test-demon", "0 * * * *", "echo hello", "inline prompt", promptFile)
	if err == nil {
		t.Error("Expected error when both prompt and prompt-file specified")
	}
}

func TestCreateWithPromptFileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Prompt file doesn't exist should error
	_, err := store.CreateWithPrompt("test-demon", "0 * * * *", "echo hello", "", "/nonexistent/prompt.txt")
	if err == nil {
		t.Error("Expected error when prompt file doesn't exist")
	}
}

func TestCreateWithPromptSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Create prompt file
	promptFile := filepath.Join(tmpDir, "prompt.txt")
	if err := os.WriteFile(promptFile, []byte("test prompt content"), 0600); err != nil {
		t.Fatalf("failed to create prompt file: %v", err)
	}

	// Create with prompt file
	demon, err := store.CreateWithPrompt("test-demon", "0 * * * *", "echo hello", "", promptFile)
	if err != nil {
		t.Fatalf("CreateWithPrompt failed: %v", err)
	}

	if demon.PromptFile != promptFile {
		t.Errorf("PromptFile = %q, want %q", demon.PromptFile, promptFile)
	}

	// Create with inline prompt
	demon2, err := store.CreateWithPrompt("test-demon2", "0 * * * *", "echo world", "inline prompt", "")
	if err != nil {
		t.Fatalf("CreateWithPrompt failed: %v", err)
	}

	if demon2.Prompt != "inline prompt" {
		t.Errorf("Prompt = %q, want %q", demon2.Prompt, "inline prompt")
	}
}

func TestSchedulerProcessRunning(t *testing.T) {
	tmpDir := t.TempDir()
	sched := NewScheduler(tmpDir)

	// Current process should be running
	if !sched.processRunning(os.Getpid()) {
		t.Error("Current process should be detected as running")
	}

	// Non-existent PID should not be running
	if sched.processRunning(99999999) {
		t.Error("Non-existent process should not be running")
	}
}

func TestParseFieldInvalidValues(t *testing.T) {
	// Invalid step
	_, err := parseField("*/-1", 0, 59)
	if err == nil {
		t.Error("Expected error for invalid step")
	}

	// Invalid comma value
	_, err = parseField("1,abc", 0, 59)
	if err == nil {
		t.Error("Expected error for invalid comma value")
	}

	// Invalid range format
	_, err = parseField("1-2-3", 0, 59)
	if err == nil {
		t.Error("Expected error for invalid range format")
	}

	// Range out of bounds
	_, err = parseField("50-70", 0, 59)
	if err == nil {
		t.Error("Expected error for range out of bounds")
	}

	// Start > end in range
	_, err = parseField("10-5", 0, 59)
	if err == nil {
		t.Error("Expected error for start > end")
	}

	// Single value out of range
	_, err = parseField("100", 0, 59)
	if err == nil {
		t.Error("Expected error for value out of range")
	}
}

func TestCronMatchesEdgeCases(t *testing.T) {
	// Test matching for specific month/day
	cron, err := ParseCron("0 0 15 6 *")
	if err != nil {
		t.Fatalf("ParseCron failed: %v", err)
	}

	// June 15th at midnight
	matches := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	next := cron.Next(matches.Add(-time.Minute))
	if !next.Equal(matches) {
		t.Errorf("Next() = %v, want %v", next, matches)
	}
}

func TestLogPath(t *testing.T) {
	store := NewStore("/tmp/test")
	expected := filepath.Join("/tmp/test", ".bc", "demons", "my-demon.log.jsonl")
	got := store.logPath("my-demon")
	if got != expected {
		t.Errorf("logPath = %q, want %q", got, expected)
	}
}

func TestSchedulerReadPIDInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	sched := NewScheduler(tmpDir)

	// Create demons directory
	demonsDir := filepath.Join(tmpDir, ".bc", "demons")
	if err := os.MkdirAll(demonsDir, 0750); err != nil {
		t.Fatalf("failed to create demons dir: %v", err)
	}

	// Write invalid PID
	pidFile := filepath.Join(demonsDir, "scheduler.pid")
	if err := os.WriteFile(pidFile, []byte("not-a-number"), 0600); err != nil {
		t.Fatalf("failed to write PID file: %v", err)
	}

	_, err := sched.readPID()
	if err == nil {
		t.Error("Expected error when PID file contains invalid data")
	}
}
