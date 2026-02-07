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
