package queue

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	q := New("/tmp/test.json")
	if q == nil {
		t.Fatal("New returned nil")
	}
	if q.path != "/tmp/test.json" {
		t.Errorf("path = %q, want %q", q.path, "/tmp/test.json")
	}
}

func TestLoadMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.json")
	q := New(path)
	if err := q.Load(); err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if items := q.ListAll(); len(items) != 0 {
		t.Errorf("expected 0 items, got %d", len(items))
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("{not json"), 0644)

	q := New(path)
	if err := q.Load(); err == nil {
		t.Fatal("expected error loading invalid JSON")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "queue.json")
	q := New(path)
	q.Add("Task A", "desc A", "bead-1")
	q.Add("Task B", "desc B", "")

	if err := q.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists and is valid JSON.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("saved file is empty")
	}

	// Load into a fresh queue and verify.
	q2 := New(path)
	if err := q2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	items := q2.ListAll()
	if len(items) != 2 {
		t.Fatalf("loaded %d items, want 2", len(items))
	}
	if items[0].Title != "Task A" {
		t.Errorf("item[0].Title = %q, want %q", items[0].Title, "Task A")
	}
	if items[1].BeadsID != "" {
		t.Errorf("item[1].BeadsID = %q, want empty", items[1].BeadsID)
	}
}

func TestAddSequentialIDs(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	a := q.Add("first", "", "")
	b := q.Add("second", "", "")
	c := q.Add("third", "", "")

	if a.ID != "work-001" {
		t.Errorf("first ID = %q, want work-001", a.ID)
	}
	if b.ID != "work-002" {
		t.Errorf("second ID = %q, want work-002", b.ID)
	}
	if c.ID != "work-003" {
		t.Errorf("third ID = %q, want work-003", c.ID)
	}
}

func TestAddFieldsSet(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	item := q.Add("Fix bug", "segfault on nil", "bead-42")

	if item.Title != "Fix bug" {
		t.Errorf("Title = %q", item.Title)
	}
	if item.Description != "segfault on nil" {
		t.Errorf("Description = %q", item.Description)
	}
	if item.BeadsID != "bead-42" {
		t.Errorf("BeadsID = %q", item.BeadsID)
	}
	if item.Status != StatusPending {
		t.Errorf("Status = %q, want pending", item.Status)
	}
	if item.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
	if item.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestGetFound(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("task", "", "")

	item := q.Get("work-001")
	if item == nil {
		t.Fatal("Get returned nil for existing item")
	}
	if item.Title != "task" {
		t.Errorf("Title = %q", item.Title)
	}
}

func TestGetNotFound(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	if item := q.Get("nonexistent"); item != nil {
		t.Errorf("expected nil, got %+v", item)
	}
}

func TestAssignSuccess(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("task", "", "")

	if err := q.Assign("work-001", "agent-1"); err != nil {
		t.Fatalf("Assign: %v", err)
	}

	item := q.Get("work-001")
	if item.Status != StatusAssigned {
		t.Errorf("Status = %q, want assigned", item.Status)
	}
	if item.AssignedTo != "agent-1" {
		t.Errorf("AssignedTo = %q, want agent-1", item.AssignedTo)
	}
}

func TestAssignNotPending(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("task", "", "")
	q.Assign("work-001", "agent-1") // now assigned

	if err := q.Assign("work-001", "agent-2"); err == nil {
		t.Fatal("expected error assigning non-pending item")
	}
}

func TestAssignNotFound(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	if err := q.Assign("nope", "agent-1"); err == nil {
		t.Fatal("expected error for missing item")
	}
}

func TestUpdateStatusSuccess(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("task", "", "")

	if err := q.UpdateStatus("work-001", StatusWorking); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	if item := q.Get("work-001"); item.Status != StatusWorking {
		t.Errorf("Status = %q, want working", item.Status)
	}
}

func TestUpdateStatusNotFound(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	if err := q.UpdateStatus("missing", StatusDone); err == nil {
		t.Fatal("expected error for missing item")
	}
}

func TestListAllReturnsCopy(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("task", "", "")

	list := q.ListAll()
	list[0].Title = "mutated"

	if item := q.Get("work-001"); item.Title != "task" {
		t.Error("ListAll did not return a copy; internal state was mutated")
	}
}

func TestListPending(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("a", "", "")
	q.Add("b", "", "")
	q.Assign("work-001", "agent")

	pending := q.ListPending()
	if len(pending) != 1 {
		t.Fatalf("got %d pending, want 1", len(pending))
	}
	if pending[0].ID != "work-002" {
		t.Errorf("pending item ID = %q, want work-002", pending[0].ID)
	}
}

func TestListByAgent(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("a", "", "")
	q.Add("b", "", "")
	q.Assign("work-001", "alice")

	items := q.ListByAgent("alice")
	if len(items) != 1 {
		t.Fatalf("got %d items for alice, want 1", len(items))
	}
	if items[0].ID != "work-001" {
		t.Errorf("ID = %q", items[0].ID)
	}

	if items := q.ListByAgent("bob"); len(items) != 0 {
		t.Errorf("expected 0 items for bob, got %d", len(items))
	}
}

func TestListByStatus(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("a", "", "")
	q.Add("b", "", "")
	q.UpdateStatus("work-002", StatusFailed)

	failed := q.ListByStatus(StatusFailed)
	if len(failed) != 1 {
		t.Fatalf("got %d failed, want 1", len(failed))
	}
	if failed[0].ID != "work-002" {
		t.Errorf("ID = %q, want work-002", failed[0].ID)
	}
}

func TestHasBeadsID(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("task", "", "bead-7")

	if !q.HasBeadsID("bead-7") {
		t.Error("expected HasBeadsID to return true")
	}
	if q.HasBeadsID("bead-99") {
		t.Error("expected HasBeadsID to return false for missing ID")
	}
}

func TestHasBeadsIDEmpty(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	if q.HasBeadsID("anything") {
		t.Error("expected false on empty queue")
	}
}

func TestStats(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("a", "", "")
	q.Add("b", "", "")
	q.Add("c", "", "")
	q.Add("d", "", "")
	q.Add("e", "", "")

	q.Assign("work-001", "agent")
	q.UpdateStatus("work-002", StatusWorking)
	q.UpdateStatus("work-003", StatusDone)
	q.UpdateStatus("work-004", StatusFailed)
	// work-005 stays pending

	s := q.Stats()
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Pending != 1 {
		t.Errorf("Pending = %d, want 1", s.Pending)
	}
	if s.Assigned != 1 {
		t.Errorf("Assigned = %d, want 1", s.Assigned)
	}
	if s.Working != 1 {
		t.Errorf("Working = %d, want 1", s.Working)
	}
	if s.Done != 1 {
		t.Errorf("Done = %d, want 1", s.Done)
	}
	if s.Failed != 1 {
		t.Errorf("Failed = %d, want 1", s.Failed)
	}
}

func TestStatsEmpty(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	s := q.Stats()
	if s.Total != 0 {
		t.Errorf("Total = %d, want 0", s.Total)
	}
}

func TestUpdateStatusUpdatesTimestamp(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	item := q.Add("task", "", "")
	created := item.UpdatedAt

	q.UpdateStatus("work-001", StatusDone)

	updated := q.Get("work-001").UpdatedAt
	if !updated.After(created) && updated != created {
		t.Error("UpdatedAt was not advanced after UpdateStatus")
	}
}

func TestAssignUpdatesTimestamp(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	item := q.Add("task", "", "")
	created := item.UpdatedAt

	q.Assign("work-001", "agent")

	updated := q.Get("work-001").UpdatedAt
	if !updated.After(created) && updated != created {
		t.Error("UpdatedAt was not advanced after Assign")
	}
}
