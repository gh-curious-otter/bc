package queue

import (
	"context"
	"testing"
)

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/test-state")
	if store == nil {
		t.Fatal("NewStore returned nil")
	}

	expectedPath := "/tmp/test-state/queues.db"
	if store.path != expectedPath {
		t.Errorf("path = %q, want %q", store.path, expectedPath)
	}
}

func TestStoreOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestWorkQueueOperations(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Add work item
	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Implement feature X",
		Description: "Build the feature",
		Status:      StatusPending,
		Priority:    PriorityHigh,
		FromAgent:   "mgr-01",
		IssueRef:    "#123",
	}

	if err := store.AddWork(ctx, item); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	if item.ID == 0 {
		t.Error("AddWork() did not set ID")
	}

	// Get work item
	got, err := store.GetWork(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetWork() error = %v", err)
	}

	if got.Title != "Implement feature X" {
		t.Errorf("Title = %q, want %q", got.Title, "Implement feature X")
	}
	if got.Status != StatusPending {
		t.Errorf("Status = %q, want %q", got.Status, StatusPending)
	}

	// List work items
	items, err := store.ListWork(ctx, "eng-01", "")
	if err != nil {
		t.Fatalf("ListWork() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}

	// List with status filter
	items, err = store.ListWork(ctx, "eng-01", StatusPending)
	if err != nil {
		t.Fatalf("ListWork() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}

	// Accept work
	if err := store.AcceptWork(ctx, item.ID); err != nil {
		t.Fatalf("AcceptWork() error = %v", err)
	}

	got, _ = store.GetWork(ctx, item.ID)
	if got.Status != StatusAccepted {
		t.Errorf("Status = %q, want %q", got.Status, StatusAccepted)
	}

	// Start work
	if err := store.StartWork(ctx, item.ID); err != nil {
		t.Fatalf("StartWork() error = %v", err)
	}

	got, _ = store.GetWork(ctx, item.ID)
	if got.Status != StatusInProgress {
		t.Errorf("Status = %q, want %q", got.Status, StatusInProgress)
	}

	// Complete work
	if err := store.CompleteWork(ctx, item.ID, "eng-01/issue-123/feature"); err != nil {
		t.Fatalf("CompleteWork() error = %v", err)
	}

	got, _ = store.GetWork(ctx, item.ID)
	if got.Status != StatusCompleted {
		t.Errorf("Status = %q, want %q", got.Status, StatusCompleted)
	}
	if got.Branch != "eng-01/issue-123/feature" {
		t.Errorf("Branch = %q, want %q", got.Branch, "eng-01/issue-123/feature")
	}
}

func TestMergeQueueOperations(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Add merge item
	item := &MergeItem{
		AgentID:   "mgr-01",
		Branch:    "eng-01/issue-123/feature",
		Title:     "Feature X implementation",
		Status:    MergeStatusPending,
		FromAgent: "eng-01",
		IssueRef:  "#123",
	}

	if err := store.AddMerge(ctx, item); err != nil {
		t.Fatalf("AddMerge() error = %v", err)
	}

	if item.ID == 0 {
		t.Error("AddMerge() did not set ID")
	}

	// Get merge item
	got, err := store.GetMerge(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetMerge() error = %v", err)
	}

	if got.Branch != "eng-01/issue-123/feature" {
		t.Errorf("Branch = %q, want %q", got.Branch, "eng-01/issue-123/feature")
	}

	// Get by branch
	got, err = store.GetMergeByBranch(ctx, "mgr-01", "eng-01/issue-123/feature")
	if err != nil {
		t.Fatalf("GetMergeByBranch() error = %v", err)
	}
	if got.ID != item.ID {
		t.Errorf("ID = %d, want %d", got.ID, item.ID)
	}

	// List merge items
	items, err := store.ListMerge(ctx, "mgr-01", "")
	if err != nil {
		t.Fatalf("ListMerge() error = %v", err)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}

	// Approve merge
	if err := store.ApproveMerge(ctx, item.ID, "mgr-01"); err != nil {
		t.Fatalf("ApproveMerge() error = %v", err)
	}

	got, _ = store.GetMerge(ctx, item.ID)
	if got.Status != MergeStatusApproved {
		t.Errorf("Status = %q, want %q", got.Status, MergeStatusApproved)
	}
	if got.Reviewer != "mgr-01" {
		t.Errorf("Reviewer = %q, want %q", got.Reviewer, "mgr-01")
	}

	// Complete merge
	if err := store.CompleteMerge(ctx, item.ID); err != nil {
		t.Fatalf("CompleteMerge() error = %v", err)
	}

	got, _ = store.GetMerge(ctx, item.ID)
	if got.Status != MergeStatusMerged {
		t.Errorf("Status = %q, want %q", got.Status, MergeStatusMerged)
	}
}

func TestMergeReject(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	item := &MergeItem{
		AgentID:   "mgr-01",
		Branch:    "eng-01/issue-456/bugfix",
		Title:     "Bugfix attempt",
		Status:    MergeStatusPending,
		FromAgent: "eng-01",
	}

	if err := store.AddMerge(ctx, item); err != nil {
		t.Fatalf("AddMerge() error = %v", err)
	}

	// Reject merge
	if err := store.RejectMerge(ctx, item.ID, "mgr-01", "Tests failing"); err != nil {
		t.Fatalf("RejectMerge() error = %v", err)
	}

	got, _ := store.GetMerge(ctx, item.ID)
	if got.Status != MergeStatusRejected {
		t.Errorf("Status = %q, want %q", got.Status, MergeStatusRejected)
	}
	if got.Reason != "Tests failing" {
		t.Errorf("Reason = %q, want %q", got.Reason, "Tests failing")
	}
}

func TestSubmit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Create and complete a work item
	work := &WorkItem{
		AgentID:  "eng-01",
		Title:    "Feature Y",
		Status:   StatusPending,
		Priority: PriorityNormal,
		IssueRef: "#789",
	}

	if err := store.AddWork(ctx, work); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	// Progress through states
	_ = store.AcceptWork(ctx, work.ID)
	_ = store.StartWork(ctx, work.ID)
	_ = store.CompleteWork(ctx, work.ID, "eng-01/issue-789/feature-y")

	// Submit to manager
	merge, err := store.Submit(ctx, work.ID, "mgr-01")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	if merge.AgentID != "mgr-01" {
		t.Errorf("AgentID = %q, want %q", merge.AgentID, "mgr-01")
	}
	if merge.FromAgent != "eng-01" {
		t.Errorf("FromAgent = %q, want %q", merge.FromAgent, "eng-01")
	}
	if merge.Branch != "eng-01/issue-789/feature-y" {
		t.Errorf("Branch = %q, want %q", merge.Branch, "eng-01/issue-789/feature-y")
	}
	if merge.Status != MergeStatusPending {
		t.Errorf("Status = %q, want %q", merge.Status, MergeStatusPending)
	}
}

func TestSubmitErrors(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Work item not completed
	work := &WorkItem{
		AgentID:  "eng-01",
		Title:    "Incomplete work",
		Status:   StatusPending,
		Priority: PriorityNormal,
	}

	if err := store.AddWork(ctx, work); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	_, submitErr := store.Submit(ctx, work.ID, "mgr-01")
	if submitErr == nil {
		t.Error("Submit() should fail for non-completed work")
	}

	// Work item with no branch
	_ = store.AcceptWork(ctx, work.ID)
	_ = store.StartWork(ctx, work.ID)
	_ = store.CompleteWork(ctx, work.ID, "") // Empty branch

	// Force status to completed without branch for test
	work2 := &WorkItem{
		AgentID:  "eng-02",
		Title:    "No branch work",
		Status:   StatusPending,
		Priority: PriorityNormal,
	}
	_ = store.AddWork(ctx, work2)
	_ = store.AcceptWork(ctx, work2.ID)
	_ = store.StartWork(ctx, work2.ID)

	// Can't submit in-progress work
	_, submitErr2 := store.Submit(ctx, work2.ID, "mgr-01")
	if submitErr2 == nil {
		t.Error("Submit() should fail for in-progress work")
	}
}

func TestWorkItemNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	_, getErr := store.GetWork(ctx, 9999)
	if getErr == nil {
		t.Error("GetWork() should return error for non-existent item")
	}
}

func TestMergeItemNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	_, getErr := store.GetMerge(ctx, 9999)
	if getErr == nil {
		t.Error("GetMerge() should return error for non-existent item")
	}
}

func TestConstants(t *testing.T) {
	// Work status
	if StatusPending != "pending" {
		t.Errorf("StatusPending = %q, want %q", StatusPending, "pending")
	}
	if StatusCompleted != "completed" {
		t.Errorf("StatusCompleted = %q, want %q", StatusCompleted, "completed")
	}

	// Merge status
	if MergeStatusPending != "pending" {
		t.Errorf("MergeStatusPending = %q, want %q", MergeStatusPending, "pending")
	}
	if MergeStatusMerged != "merged" {
		t.Errorf("MergeStatusMerged = %q, want %q", MergeStatusMerged, "merged")
	}

	// Priority
	if PriorityUrgent != 3 {
		t.Errorf("PriorityUrgent = %d, want %d", PriorityUrgent, 3)
	}
}

func TestCloseNilDB(t *testing.T) {
	store := NewStore("/tmp/test")
	// Close without opening should not error
	if err := store.Close(); err != nil {
		t.Errorf("Close() on nil db should not error, got %v", err)
	}
}

func TestAcceptWorkNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// AcceptWork on non-existent item
	if err := store.AcceptWork(ctx, 9999); err == nil {
		t.Error("AcceptWork() should fail for non-existent item")
	}
}

func TestStartWorkNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// StartWork on non-existent item
	if err := store.StartWork(ctx, 9999); err == nil {
		t.Error("StartWork() should fail for non-existent item")
	}
}

func TestCompleteWorkNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// CompleteWork on non-existent item
	if err := store.CompleteWork(ctx, 9999, "branch"); err == nil {
		t.Error("CompleteWork() should fail for non-existent item")
	}
}

func TestApproveMergeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// ApproveMerge on non-existent item
	if err := store.ApproveMerge(ctx, 9999, "reviewer"); err == nil {
		t.Error("ApproveMerge() should fail for non-existent item")
	}
}

func TestRejectMergeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// RejectMerge on non-existent item
	if err := store.RejectMerge(ctx, 9999, "reviewer", "reason"); err == nil {
		t.Error("RejectMerge() should fail for non-existent item")
	}
}

func TestCompleteMergeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// CompleteMerge on non-existent item
	if err := store.CompleteMerge(ctx, 9999); err == nil {
		t.Error("CompleteMerge() should fail for non-existent item")
	}
}

func TestSubmitNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Submit on non-existent item
	if _, err := store.Submit(ctx, 9999, "mgr"); err == nil {
		t.Error("Submit() should fail for non-existent item")
	}
}

func TestListWorkEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// List work for non-existent agent
	items, err := store.ListWork(ctx, "nonexistent", "")
	if err != nil {
		t.Fatalf("ListWork() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestListMergeEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// List merge for non-existent agent
	items, err := store.ListMerge(ctx, "nonexistent", "")
	if err != nil {
		t.Fatalf("ListMerge() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("len(items) = %d, want 0", len(items))
	}
}

func TestGetMergeByBranchNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Get merge by branch for non-existent branch
	_, err := store.GetMergeByBranch(ctx, "agent", "nonexistent/branch")
	if err == nil {
		t.Error("GetMergeByBranch() should fail for non-existent branch")
	}
}

func TestListMergeWithStatusFilter(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Add merge items with different statuses
	item1 := &MergeItem{
		AgentID:   "mgr-01",
		Branch:    "branch1",
		Title:     "Merge 1",
		Status:    MergeStatusPending,
		FromAgent: "eng-01",
	}
	item2 := &MergeItem{
		AgentID:   "mgr-01",
		Branch:    "branch2",
		Title:     "Merge 2",
		Status:    MergeStatusPending,
		FromAgent: "eng-02",
	}

	_ = store.AddMerge(ctx, item1)
	_ = store.AddMerge(ctx, item2)

	// Approve one
	_ = store.ApproveMerge(ctx, item1.ID, "mgr-01")

	// List pending only
	pending, err := store.ListMerge(ctx, "mgr-01", MergeStatusPending)
	if err != nil {
		t.Fatalf("ListMerge() error = %v", err)
	}
	if len(pending) != 1 {
		t.Errorf("len(pending) = %d, want 1", len(pending))
	}

	// List approved only
	approved, err := store.ListMerge(ctx, "mgr-01", MergeStatusApproved)
	if err != nil {
		t.Fatalf("ListMerge() error = %v", err)
	}
	if len(approved) != 1 {
		t.Errorf("len(approved) = %d, want 1", len(approved))
	}
}

func TestWorkItemFields(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck // cleanup in test

	// Test all fields are saved and retrieved correctly
	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Full Feature Test",
		Description: "Detailed description",
		Status:      StatusPending,
		Priority:    PriorityUrgent,
		FromAgent:   "mgr-01",
		IssueRef:    "#999",
	}

	if err := store.AddWork(ctx, item); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	got, err := store.GetWork(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetWork() error = %v", err)
	}

	if got.Description != "Detailed description" {
		t.Errorf("Description = %q, want %q", got.Description, "Detailed description")
	}
	if got.Priority != PriorityUrgent {
		t.Errorf("Priority = %d, want %d", got.Priority, PriorityUrgent)
	}
	if got.FromAgent != "mgr-01" {
		t.Errorf("FromAgent = %q, want %q", got.FromAgent, "mgr-01")
	}
	if got.IssueRef != "#999" {
		t.Errorf("IssueRef = %q, want %q", got.IssueRef, "#999")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}
