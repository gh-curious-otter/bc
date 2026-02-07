package demon

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStore_Create(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demon := &Demon{
		Name:     "cleanup",
		Schedule: "0 * * * *",
		Command:  "rm -rf /tmp/cache",
		Owner:    "engineer-01",
		Enabled:  true,
	}

	if err := store.Create(demon); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify it was saved
	loaded, err := store.Get("cleanup")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if loaded.Name != "cleanup" {
		t.Errorf("Name = %q, want cleanup", loaded.Name)
	}
	if loaded.Schedule != "0 * * * *" {
		t.Errorf("Schedule = %q, want '0 * * * *'", loaded.Schedule)
	}
	if loaded.Owner != "engineer-01" {
		t.Errorf("Owner = %q, want engineer-01", loaded.Owner)
	}
	if !loaded.Enabled {
		t.Error("Enabled should be true")
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestStore_CreateDuplicate(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demon := &Demon{
		Name:    "test",
		Command: "echo hello",
		Owner:   "engineer-01",
	}

	if err := store.Create(demon); err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	err := store.Create(demon)
	if err == nil {
		t.Error("Expected error for duplicate demon")
	}
}

func TestStore_Get(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	// Get nonexistent
	_, err := store.Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent demon")
	}

	// Create and get
	demon := &Demon{
		Name:    "test",
		Command: "echo test",
		Owner:   "qa-01",
	}
	if createErr := store.Create(demon); createErr != nil {
		t.Fatalf("Create failed: %v", createErr)
	}

	loaded, err := store.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if loaded.Owner != "qa-01" {
		t.Errorf("Owner = %q, want qa-01", loaded.Owner)
	}
}

func TestStore_Update(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demon := &Demon{
		Name:        "updatable",
		Command:     "old command",
		Description: "old desc",
		Owner:       "engineer-01",
	}
	if err := store.Create(demon); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	err := store.Update("updatable", func(d *Demon) {
		d.Command = "new command"
		d.Description = "new desc"
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	loaded, loadErr := store.Get("updatable")
	if loadErr != nil {
		t.Fatalf("Get failed: %v", loadErr)
	}
	if loaded.Command != "new command" {
		t.Errorf("Command = %q, want 'new command'", loaded.Command)
	}
	if loaded.Description != "new desc" {
		t.Errorf("Description = %q, want 'new desc'", loaded.Description)
	}
}

func TestStore_UpdateNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	err := store.Update("nonexistent", func(d *Demon) {
		d.Command = "test"
	})
	if err == nil {
		t.Error("Expected error for nonexistent demon")
	}
}

func TestStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demon := &Demon{
		Name:    "deletable",
		Command: "echo delete",
		Owner:   "engineer-01",
	}
	if err := store.Create(demon); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := store.Delete("deletable"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := store.Get("deletable")
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestStore_DeleteNonexistent(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	err := store.Delete("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent demon")
	}
}

func TestStore_ListByOwner(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	// Create demons with different owners
	demons := []Demon{
		{Name: "d1", Command: "cmd1", Owner: "engineer-01"},
		{Name: "d2", Command: "cmd2", Owner: "engineer-01"},
		{Name: "d3", Command: "cmd3", Owner: "qa-01"},
	}

	for i := range demons {
		if err := store.Create(&demons[i]); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	eng01Demons, err := store.ListByOwner("engineer-01")
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(eng01Demons) != 2 {
		t.Errorf("len = %d, want 2", len(eng01Demons))
	}

	qa01Demons, err := store.ListByOwner("qa-01")
	if err != nil {
		t.Fatalf("ListByOwner failed: %v", err)
	}
	if len(qa01Demons) != 1 {
		t.Errorf("len = %d, want 1", len(qa01Demons))
	}
}

func TestStore_ListEnabled(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demons := []Demon{
		{Name: "enabled1", Command: "cmd1", Owner: "eng", Enabled: true},
		{Name: "disabled1", Command: "cmd2", Owner: "eng", Enabled: false},
		{Name: "enabled2", Command: "cmd3", Owner: "eng", Enabled: true},
	}

	for i := range demons {
		if err := store.Create(&demons[i]); err != nil {
			t.Fatalf("Create failed: %v", err)
		}
	}

	enabled, err := store.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled failed: %v", err)
	}
	if len(enabled) != 2 {
		t.Errorf("len = %d, want 2", len(enabled))
	}
}

func TestStore_EnableDisable(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demon := &Demon{
		Name:    "toggleable",
		Command: "echo toggle",
		Owner:   "eng",
		Enabled: false,
	}
	if err := store.Create(demon); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Enable
	if err := store.Enable("toggleable"); err != nil {
		t.Fatalf("Enable failed: %v", err)
	}
	loaded, _ := store.Get("toggleable")
	if !loaded.Enabled {
		t.Error("Should be enabled")
	}

	// Disable
	if err := store.Disable("toggleable"); err != nil {
		t.Fatalf("Disable failed: %v", err)
	}
	loaded, _ = store.Get("toggleable")
	if loaded.Enabled {
		t.Error("Should be disabled")
	}
}

func TestStore_RecordRun(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demon := &Demon{
		Name:    "runner",
		Command: "echo run",
		Owner:   "eng",
	}
	if err := store.Create(demon); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	nextRun := time.Now().Add(time.Hour)
	if err := store.RecordRun("runner", nextRun); err != nil {
		t.Fatalf("RecordRun failed: %v", err)
	}

	loaded, _ := store.Get("runner")
	if loaded.RunCount != 1 {
		t.Errorf("RunCount = %d, want 1", loaded.RunCount)
	}
	if loaded.LastRunAt.IsZero() {
		t.Error("LastRunAt should be set")
	}
	if loaded.NextRunAt.IsZero() {
		t.Error("NextRunAt should be set")
	}

	// Run again
	if err := store.RecordRun("runner", nextRun.Add(time.Hour)); err != nil {
		t.Fatalf("Second RecordRun failed: %v", err)
	}
	loaded, _ = store.Get("runner")
	if loaded.RunCount != 2 {
		t.Errorf("RunCount = %d, want 2", loaded.RunCount)
	}
}

func TestStore_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, ".bc"))

	demons, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(demons) > 0 {
		t.Errorf("Expected empty list, got %d demons", len(demons))
	}
}

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/test/.bc")
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if store.path != "/tmp/test/.bc/demons.json" {
		t.Errorf("path = %q, want /tmp/test/.bc/demons.json", store.path)
	}
}
