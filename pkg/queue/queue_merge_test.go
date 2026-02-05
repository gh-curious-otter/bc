package queue

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeStatusConstants(t *testing.T) {
	// Verify merge status values
	if MergeNone != "" {
		t.Errorf("MergeNone should be empty string")
	}
	if MergeUnmerged != "unmerged" {
		t.Errorf("MergeUnmerged should be 'unmerged'")
	}
	if MergeMerging != "merging" {
		t.Errorf("MergeMerging should be 'merging'")
	}
	if MergeMerged != "merged" {
		t.Errorf("MergeMerged should be 'merged'")
	}
	if MergeConflict != "conflict" {
		t.Errorf("MergeConflict should be 'conflict'")
	}
}

func TestUpdateStatus_AutoSetsMergeUnmerged(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	item := q.Add("Test task", "", "")
	id := item.ID

	if err := q.UpdateStatus(id, StatusDone); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	got := q.Get(id)
	if got.Merge != MergeUnmerged {
		t.Errorf("expected MergeUnmerged after done, got %q", got.Merge)
	}
}

func TestUpdateStatus_DoesNotOverwriteExistingMerge(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	item := q.Add("Test task", "", "")
	id := item.ID

	// Set merge status to merged first
	q.UpdateMergeStatus(id, MergeMerged, "abc123")

	// Transition to done — should not overwrite merged status
	q.UpdateStatus(id, StatusDone)

	got := q.Get(id)
	if got.Merge != MergeMerged {
		t.Errorf("expected MergeMerged to be preserved, got %q", got.Merge)
	}
}

func TestUpdateMergeStatus(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	item := q.Add("Test task", "", "")
	id := item.ID

	if err := q.UpdateMergeStatus(id, MergeMerging, ""); err != nil {
		t.Fatalf("UpdateMergeStatus merging: %v", err)
	}

	got := q.Get(id)
	if got.Merge != MergeMerging {
		t.Errorf("expected MergeMerging, got %q", got.Merge)
	}

	if err := q.UpdateMergeStatus(id, MergeMerged, "abc123def"); err != nil {
		t.Fatalf("UpdateMergeStatus merged: %v", err)
	}

	got = q.Get(id)
	if got.Merge != MergeMerged {
		t.Errorf("expected MergeMerged, got %q", got.Merge)
	}
	if got.MergeCommit != "abc123def" {
		t.Errorf("expected commit hash 'abc123def', got %q", got.MergeCommit)
	}
	if got.MergedAt.IsZero() {
		t.Error("expected MergedAt to be set")
	}
}

func TestUpdateMergeStatus_NotFound(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))

	err := q.UpdateMergeStatus("nonexistent", MergeMerged, "abc")
	if err == nil {
		t.Error("expected error for nonexistent item")
	}
}

func TestUpdateMergeStatus_Conflict(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	item := q.Add("Test task", "", "")

	q.UpdateMergeStatus(item.ID, MergeConflict, "")

	got := q.Get(item.ID)
	if got.Merge != MergeConflict {
		t.Errorf("expected MergeConflict, got %q", got.Merge)
	}
	// MergedAt should NOT be set for conflict
	if !got.MergedAt.IsZero() {
		t.Error("MergedAt should be zero for conflict status")
	}
}

func TestSetBranch(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	item := q.Add("Test task", "", "")

	if err := q.SetBranch(item.ID, "engineer-01/work-108/fix-auth"); err != nil {
		t.Fatalf("SetBranch: %v", err)
	}

	got := q.Get(item.ID)
	if got.Branch != "engineer-01/work-108/fix-auth" {
		t.Errorf("expected branch, got %q", got.Branch)
	}
}

func TestSetBranch_NotFound(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))

	err := q.SetBranch("nonexistent", "some-branch")
	if err == nil {
		t.Error("expected error for nonexistent item")
	}
}

func TestFindByBranch(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	q.Add("Task 1", "", "")
	item2 := q.Add("Task 2", "", "")
	q.SetBranch(item2.ID, "engineer-01/work-200/feature")

	found := q.FindByBranch("engineer-01/work-200/feature")
	if found == nil {
		t.Fatal("expected to find item by branch")
	}
	if found.ID != item2.ID {
		t.Errorf("expected item %s, got %s", item2.ID, found.ID)
	}
}

func TestFindByBranch_NotFound(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	q.Add("Task 1", "", "")

	found := q.FindByBranch("nonexistent-branch")
	if found != nil {
		t.Error("expected nil for nonexistent branch")
	}
}

func TestListMergeable(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))

	// Add items in various states
	item1 := q.Add("Done unmerged", "", "")
	q.UpdateStatus(item1.ID, StatusDone) // auto-sets MergeUnmerged

	item2 := q.Add("Done merged", "", "")
	q.UpdateStatus(item2.ID, StatusDone)
	q.UpdateMergeStatus(item2.ID, MergeMerged, "abc")

	item3 := q.Add("Still working", "", "")
	q.UpdateStatus(item3.ID, StatusWorking)

	item4 := q.Add("Done conflict", "", "")
	q.UpdateStatus(item4.ID, StatusDone)
	q.UpdateMergeStatus(item4.ID, MergeConflict, "")

	mergeable := q.ListMergeable()

	// Should include item1 (unmerged) and item4 (conflict), but not item2 (merged) or item3 (working)
	if len(mergeable) != 2 {
		t.Fatalf("expected 2 mergeable items, got %d", len(mergeable))
	}
	if mergeable[0].ID != item1.ID {
		t.Errorf("expected first mergeable to be %s, got %s", item1.ID, mergeable[0].ID)
	}
	if mergeable[1].ID != item4.ID {
		t.Errorf("expected second mergeable to be %s, got %s", item4.ID, mergeable[1].ID)
	}
}

func TestListMergeable_Empty(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))
	q.Add("Pending task", "", "")

	mergeable := q.ListMergeable()
	if len(mergeable) != 0 {
		t.Errorf("expected 0 mergeable items for pending-only queue, got %d", len(mergeable))
	}
}

func TestStats_IncludesMergeCount(t *testing.T) {
	dir := t.TempDir()
	q := New(filepath.Join(dir, "q.json"))

	item1 := q.Add("Task 1", "", "")
	q.UpdateStatus(item1.ID, StatusDone) // auto MergeUnmerged

	item2 := q.Add("Task 2", "", "")
	q.UpdateStatus(item2.ID, StatusDone)
	q.UpdateMergeStatus(item2.ID, MergeMerged, "abc")

	item3 := q.Add("Task 3", "", "")
	_ = item3 // pending, no merge status

	s := q.Stats()
	if s.Merged != 1 {
		t.Errorf("expected 1 merged, got %d", s.Merged)
	}
	if s.Unmerged != 1 {
		t.Errorf("expected 1 unmerged, got %d", s.Unmerged)
	}
}

func TestMergeFields_Persist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "q.json")

	// Create and save
	q := New(path)
	item := q.Add("Test task", "", "")
	q.SetBranch(item.ID, "feature/test")
	q.UpdateMergeStatus(item.ID, MergeMerged, "deadbeef")
	if err := q.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("queue file not written: %v", err)
	}

	// Reload and verify
	q2 := New(path)
	if err := q2.Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := q2.Get(item.ID)
	if got == nil {
		t.Fatal("item not found after reload")
	}
	if got.Branch != "feature/test" {
		t.Errorf("expected branch 'feature/test', got %q", got.Branch)
	}
	if got.Merge != MergeMerged {
		t.Errorf("expected MergeMerged, got %q", got.Merge)
	}
	if got.MergeCommit != "deadbeef" {
		t.Errorf("expected commit 'deadbeef', got %q", got.MergeCommit)
	}
	if got.MergedAt.IsZero() {
		t.Error("expected MergedAt to persist")
	}
}
