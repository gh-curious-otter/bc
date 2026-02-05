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
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

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
	if err := q.Assign("work-001", "agent-1"); err != nil {
		t.Fatal(err)
	}

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
	if err := q.Assign("work-001", "agent"); err != nil {
		t.Fatal(err)
	}

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
	if err := q.Assign("work-001", "alice"); err != nil {
		t.Fatal(err)
	}

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
	if err := q.UpdateStatus("work-002", StatusFailed); err != nil {
		t.Fatal(err)
	}

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

	if err := q.Assign("work-001", "agent"); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus("work-002", StatusWorking); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus("work-003", StatusDone); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus("work-004", StatusFailed); err != nil {
		t.Fatal(err)
	}
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

	if err := q.UpdateStatus("work-001", StatusDone); err != nil {
		t.Fatal(err)
	}

	updated := q.Get("work-001").UpdatedAt
	if !updated.After(created) && updated != created {
		t.Error("UpdatedAt was not advanced after UpdateStatus")
	}
}

func TestAssignUpdatesTimestamp(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	item := q.Add("task", "", "")
	created := item.UpdatedAt

	if err := q.Assign("work-001", "agent"); err != nil {
		t.Fatal(err)
	}

	updated := q.Get("work-001").UpdatedAt
	if !updated.After(created) && updated != created {
		t.Error("UpdatedAt was not advanced after Assign")
	}
}

func TestFindByTitle(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("Fix auth bug", "", "")
	q.Add("Add dashboard", "", "bead-1")

	item := q.FindByTitle("Fix auth bug")
	if item == nil {
		t.Fatal("FindByTitle returned nil for existing title")
	}
	if item.ID != "work-001" {
		t.Errorf("ID = %q, want work-001", item.ID)
	}

	item = q.FindByTitle("Add dashboard")
	if item == nil {
		t.Fatal("FindByTitle returned nil for existing title")
	}
	if item.ID != "work-002" {
		t.Errorf("ID = %q, want work-002", item.ID)
	}

	if q.FindByTitle("nonexistent") != nil {
		t.Error("FindByTitle should return nil for missing title")
	}
}

func TestFindByTitleEmpty(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	if q.FindByTitle("anything") != nil {
		t.Error("expected nil on empty queue")
	}
}

func TestLinkBeadsID(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	q.Add("Fix auth bug", "", "")

	if err := q.LinkBeadsID("work-001", "bead-42"); err != nil {
		t.Fatalf("LinkBeadsID: %v", err)
	}

	item := q.Get("work-001")
	if item.BeadsID != "bead-42" {
		t.Errorf("BeadsID = %q, want bead-42", item.BeadsID)
	}

	// Now HasBeadsID should find it
	if !q.HasBeadsID("bead-42") {
		t.Error("HasBeadsID should return true after linking")
	}
}

func TestLinkBeadsIDNotFound(t *testing.T) {
	q := New(filepath.Join(t.TempDir(), "q.json"))
	if err := q.LinkBeadsID("missing", "bead-1"); err == nil {
		t.Fatal("expected error for missing item")
	}
}

func TestDedupByTitlePreventsDoubleAdd(t *testing.T) {
	// Simulate the real bug: manual item added without beads ID,
	// then bc queue load tries to add the same item from beads.
	q := New(filepath.Join(t.TempDir(), "q.json"))

	// Step 1: item added manually (no beads ID)
	q.Add("Fix SendKeys race condition", "", "")

	// Step 2: beads sync finds an issue with the same title
	beadsID := "bc-rgb.3"

	// Without dedup fix, HasBeadsID returns false and a duplicate is created.
	// With fix, FindByTitle catches it.
	if q.HasBeadsID(beadsID) {
		t.Fatal("HasBeadsID should be false before linking")
	}

	existing := q.FindByTitle("Fix SendKeys race condition")
	if existing == nil {
		t.Fatal("FindByTitle should find the manually added item")
	}

	// Link the beads ID to the existing item
	if err := q.LinkBeadsID(existing.ID, beadsID); err != nil {
		t.Fatalf("LinkBeadsID: %v", err)
	}

	// Verify: only 1 item in queue, not 2
	if items := q.ListAll(); len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}

	// Verify: future HasBeadsID check now succeeds
	if !q.HasBeadsID(beadsID) {
		t.Error("HasBeadsID should return true after linking")
	}
}
