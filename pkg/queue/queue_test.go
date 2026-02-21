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

// --- Work Queue Status Transition Error Tests ---

func TestAcceptWorkWrongStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add work item
	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Test task",
		Description: "Test",
		Priority:    PriorityNormal,
		Status:      StatusPending,
	}
	if err := store.AddWork(ctx, item); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	// Accept it first
	if err := store.AcceptWork(ctx, item.ID); err != nil {
		t.Fatalf("AcceptWork() error = %v", err)
	}

	// Try to accept again - should fail
	err := store.AcceptWork(ctx, item.ID)
	if err == nil {
		t.Error("AcceptWork() should fail for already accepted item")
	}
}

func TestAcceptWorkNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Try to accept non-existent item
	err := store.AcceptWork(ctx, 99999)
	if err == nil {
		t.Error("AcceptWork() should fail for non-existent item")
	}
}

func TestStartWorkWrongStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add work item (pending status)
	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Test task",
		Description: "Test",
		Priority:    PriorityNormal,
		Status:      StatusPending,
	}
	if err := store.AddWork(ctx, item); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	// Try to start without accepting - should fail
	err := store.StartWork(ctx, item.ID)
	if err == nil {
		t.Error("StartWork() should fail for pending item (not accepted)")
	}
}

func TestStartWorkNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	err := store.StartWork(ctx, 99999)
	if err == nil {
		t.Error("StartWork() should fail for non-existent item")
	}
}

func TestCompleteWorkWrongStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add work item
	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Test task",
		Description: "Test",
		Priority:    PriorityNormal,
		Status:      StatusPending,
	}
	if err := store.AddWork(ctx, item); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	// Try to complete without starting - should fail
	err := store.CompleteWork(ctx, item.ID, "feature/test")
	if err == nil {
		t.Error("CompleteWork() should fail for pending item (not in progress)")
	}
}

func TestCompleteWorkNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	err := store.CompleteWork(ctx, 99999, "feature/test")
	if err == nil {
		t.Error("CompleteWork() should fail for non-existent item")
	}
}

// --- Merge Queue Status Transition Error Tests ---

func TestApproveMergeWrongStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add merge item
	item := &MergeItem{
		AgentID:   "eng-01",
		Branch:    "feature/test",
		Title:     "Test PR",
		Status:    MergeStatusPending,
		FromAgent: "eng-02",
	}
	if err := store.AddMerge(ctx, item); err != nil {
		t.Fatalf("AddMerge() error = %v", err)
	}

	// Approve it first
	if err := store.ApproveMerge(ctx, item.ID, "tl-01"); err != nil {
		t.Fatalf("ApproveMerge() error = %v", err)
	}

	// Try to approve again - should fail
	err := store.ApproveMerge(ctx, item.ID, "tl-02")
	if err == nil {
		t.Error("ApproveMerge() should fail for already approved item")
	}
}

func TestApproveMergeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	err := store.ApproveMerge(ctx, 99999, "tl-01")
	if err == nil {
		t.Error("ApproveMerge() should fail for non-existent item")
	}
}

func TestRejectMergeWrongStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add merge item
	item := &MergeItem{
		AgentID:   "eng-01",
		Branch:    "feature/test",
		Title:     "Test PR",
		Status:    MergeStatusPending,
		FromAgent: "eng-02",
	}
	if err := store.AddMerge(ctx, item); err != nil {
		t.Fatalf("AddMerge() error = %v", err)
	}

	// Approve it first
	if err := store.ApproveMerge(ctx, item.ID, "tl-01"); err != nil {
		t.Fatalf("ApproveMerge() error = %v", err)
	}

	// Try to reject approved item - should fail
	err := store.RejectMerge(ctx, item.ID, "tl-02", "Changed mind")
	if err == nil {
		t.Error("RejectMerge() should fail for approved item")
	}
}

func TestRejectMergeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	err := store.RejectMerge(ctx, 99999, "tl-01", "Not needed")
	if err == nil {
		t.Error("RejectMerge() should fail for non-existent item")
	}
}

func TestCompleteMergeWrongStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add merge item (pending status)
	item := &MergeItem{
		AgentID:   "eng-01",
		Branch:    "feature/test",
		Title:     "Test PR",
		Status:    MergeStatusPending,
		FromAgent: "eng-02",
	}
	if err := store.AddMerge(ctx, item); err != nil {
		t.Fatalf("AddMerge() error = %v", err)
	}

	// Try to complete without approving - should fail
	err := store.CompleteMerge(ctx, item.ID)
	if err == nil {
		t.Error("CompleteMerge() should fail for pending item (not approved)")
	}
}

func TestCompleteMergeNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	err := store.CompleteMerge(ctx, 99999)
	if err == nil {
		t.Error("CompleteMerge() should fail for non-existent item")
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

func TestStoreCloseNil(t *testing.T) {
	store := NewStore(t.TempDir())
	// Close without Open should not error
	err := store.Close()
	if err != nil {
		t.Errorf("Close() on unopened store error = %v", err)
	}
}

func TestStoreOpenInvalidPath(t *testing.T) {
	// Use path that will fail to create database
	store := NewStore("/nonexistent/path/that/does/not/exist")
	ctx := context.Background()

	err := store.Open(ctx)
	if err == nil {
		t.Error("Open() should fail for invalid path")
	}
}

func TestAddWorkWithMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	item := &WorkItem{
		AgentID:     "eng-01",
		Title:       "Test work",
		Description: "Test description",
		Priority:    PriorityNormal,
		Metadata:    `{"key": "value"}`,
	}

	if err := store.AddWork(ctx, item); err != nil {
		t.Fatalf("AddWork() error = %v", err)
	}

	if item.ID == 0 {
		t.Error("item.ID should be set after AddWork")
	}
}

func TestListWorkByStatus(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add items with different statuses
	item1 := &WorkItem{AgentID: "eng-01", Title: "Pending", Status: StatusPending}
	item2 := &WorkItem{AgentID: "eng-01", Title: "In Progress", Status: StatusInProgress}

	if err := store.AddWork(ctx, item1); err != nil {
		t.Fatal(err)
	}
	if err := store.AddWork(ctx, item2); err != nil {
		t.Fatal(err)
	}

	// List by status
	items, err := store.ListWork(ctx, "eng-01", StatusPending)
	if err != nil {
		t.Fatalf("ListWork() error = %v", err)
	}

	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}

func TestSubmitWorkAndMerge(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add work item
	work := &WorkItem{
		AgentID: "eng-01",
		Title:   "Feature work",
		Status:  StatusCompleted, // Must be completed to submit
		Branch:  "feature/test",
	}
	if err := store.AddWork(ctx, work); err != nil {
		t.Fatal(err)
	}

	// Submit work for merge
	merge, err := store.Submit(ctx, work.ID, "tl-01")
	if err != nil {
		t.Fatalf("Submit() error = %v", err)
	}

	if merge.FromAgent != "eng-01" {
		t.Errorf("merge.FromAgent = %q, want %q", merge.FromAgent, "eng-01")
	}
	if merge.AgentID != "tl-01" {
		t.Errorf("merge.AgentID = %q, want %q", merge.AgentID, "tl-01")
	}
}

func TestSubmitWorkNotFoundExtra(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	_, err := store.Submit(ctx, 99999, "tl-01")
	if err == nil {
		t.Error("Submit() should fail for non-existent work item")
	}
}

func TestListMergeByAgent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	ctx := context.Background()

	if err := store.Open(ctx); err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close() //nolint:errcheck

	// Add merge items for different agents
	item1 := &MergeItem{AgentID: "tl-01", Title: "PR 1", Branch: "b1", Status: MergeStatusPending}
	item2 := &MergeItem{AgentID: "tl-02", Title: "PR 2", Branch: "b2", Status: MergeStatusPending}

	if err := store.AddMerge(ctx, item1); err != nil {
		t.Fatal(err)
	}
	if err := store.AddMerge(ctx, item2); err != nil {
		t.Fatal(err)
	}

	// List for specific agent
	items, err := store.ListMerge(ctx, "tl-01", "")
	if err != nil {
		t.Fatalf("ListMerge() error = %v", err)
	}

	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1", len(items))
	}
}
