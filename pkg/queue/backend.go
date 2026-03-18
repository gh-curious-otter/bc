package queue

import "context"

// Backend is the storage interface for the dual queue system.
// Store is the default SQLite implementation.
type Backend interface {
	// AddWork adds a new item to an agent's work queue.
	AddWork(ctx context.Context, item *WorkItem) error
	// GetWork retrieves a work item by ID.
	GetWork(ctx context.Context, id int64) (*WorkItem, error)
	// ListWork lists work items for an agent, optionally filtered by status.
	ListWork(ctx context.Context, agentID string, status string) ([]*WorkItem, error)
	// AcceptWork marks a work item as accepted.
	AcceptWork(ctx context.Context, id int64) error
	// StartWork marks a work item as in progress.
	StartWork(ctx context.Context, id int64) error
	// CompleteWork marks a work item as completed with the given branch.
	CompleteWork(ctx context.Context, id int64, branch string) error
	// AddMerge adds a new item to an agent's merge queue.
	AddMerge(ctx context.Context, item *MergeItem) error
	// GetMerge retrieves a merge item by ID.
	GetMerge(ctx context.Context, id int64) (*MergeItem, error)
	// GetMergeByBranch retrieves the latest active merge item for a branch.
	GetMergeByBranch(ctx context.Context, agentID, branch string) (*MergeItem, error)
	// ListMerge lists merge items for an agent, optionally filtered by status.
	ListMerge(ctx context.Context, agentID string, status string) ([]*MergeItem, error)
	// ApproveMerge approves a merge item.
	ApproveMerge(ctx context.Context, id int64, reviewer string) error
	// RejectMerge rejects a merge item with a reason.
	RejectMerge(ctx context.Context, id int64, reviewer, reason string) error
	// CompleteMerge marks a merge item as merged.
	CompleteMerge(ctx context.Context, id int64) error
	// Submit submits a completed work item to a target agent's merge queue.
	Submit(ctx context.Context, workID int64, toAgent string) (*MergeItem, error)
	// Close releases database resources.
	Close() error
}
