// Package queue manages the work queue for bc.
//
// Deprecated: This package is deprecated and will be removed in a future version.
// Use the channels package for message-based work routing instead.
// See pkg/channel for the new implementation.
//
// Work items are persisted to .bc/queue.json. Items flow from beads issues
// to agents: pending → assigned → working → done/failed.
package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// ItemStatus represents the lifecycle state of a work item.
type ItemStatus string

const (
	StatusPending  ItemStatus = "pending"
	StatusAssigned ItemStatus = "assigned"
	StatusWorking  ItemStatus = "working"
	StatusDone     ItemStatus = "done"
	StatusFailed   ItemStatus = "failed"
)

// MergeStatus tracks whether a completed work item's branch has been merged.
type MergeStatus string

const (
	MergeNone     MergeStatus = "" // not applicable (item not done)
	MergeUnmerged MergeStatus = "unmerged"
	MergeMerging  MergeStatus = "merging"
	MergeMerged   MergeStatus = "merged"
	MergeConflict MergeStatus = "conflict"
)

// WorkItem is a unit of work in the queue.
type WorkItem struct {
	ID          string     `json:"id"`
	BeadsID     string     `json:"beads_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      ItemStatus `json:"status"`
	AssignedTo  string     `json:"assigned_to,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Merge tracking
	Branch      string      `json:"branch,omitempty"`
	Merge       MergeStatus `json:"merge,omitempty"`
	MergedAt    time.Time   `json:"merged_at,omitempty"`
	MergeCommit string      `json:"merge_commit,omitempty"`
}

// Stats summarizes queue state.
type Stats struct {
	Total    int `json:"total"`
	Pending  int `json:"pending"`
	Assigned int `json:"assigned"`
	Working  int `json:"working"`
	Done     int `json:"done"`
	Failed   int `json:"failed"`
	Merged   int `json:"merged"`
	Unmerged int `json:"unmerged"`
}

// Queue manages work items persisted to a JSON file.
type Queue struct {
	path  string
	items []WorkItem
	mu    sync.RWMutex
}

// New creates a Queue backed by the given file path.
func New(path string) *Queue {
	return &Queue{path: path}
}

// Load reads queue state from disk.
func (q *Queue) Load() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := os.ReadFile(q.path)
	if err != nil {
		if os.IsNotExist(err) {
			q.items = nil
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &q.items)
}

// Save writes queue state to disk.
func (q *Queue) Save() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	data, err := json.MarshalIndent(q.items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(q.path, data, 0600)
}

// Add creates a new work item with an auto-generated ID.
func (q *Queue) Add(title, description, beadsID string) *WorkItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	id := fmt.Sprintf("work-%03d", len(q.items)+1)
	now := time.Now()
	item := WorkItem{
		ID:          id,
		BeadsID:     beadsID,
		Title:       title,
		Description: description,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	q.items = append(q.items, item)
	return &q.items[len(q.items)-1]
}

// Get returns a work item by ID, or nil if not found.
func (q *Queue) Get(id string) *WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for i := range q.items {
		if q.items[i].ID == id {
			return &q.items[i]
		}
	}
	return nil
}

// Assign transitions a pending item to assigned and sets the agent.
func (q *Queue) Assign(id, agentName string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].ID == id {
			if q.items[i].Status != StatusPending {
				return fmt.Errorf("item %s is %s, not pending", id, q.items[i].Status)
			}
			q.items[i].Status = StatusAssigned
			q.items[i].AssignedTo = agentName
			q.items[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("item %s not found", id)
}

// UpdateStatus changes the status of a work item.
// When transitioning to done, merge status is automatically set to unmerged.
func (q *Queue) UpdateStatus(id string, s ItemStatus) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].ID == id {
			q.items[i].Status = s
			q.items[i].UpdatedAt = time.Now()
			// Auto-set merge status when work is done
			if s == StatusDone && q.items[i].Merge == MergeNone {
				q.items[i].Merge = MergeUnmerged
			}
			return nil
		}
	}
	return fmt.Errorf("item %s not found", id)
}

// ListAll returns all work items.
func (q *Queue) ListAll() []WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	out := make([]WorkItem, len(q.items))
	copy(out, q.items)
	return out
}

// ListPending returns items with status pending.
func (q *Queue) ListPending() []WorkItem {
	return q.ListByStatus(StatusPending)
}

// ListByAgent returns items assigned to a specific agent.
func (q *Queue) ListByAgent(agentName string) []WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var out []WorkItem
	for _, item := range q.items {
		if item.AssignedTo == agentName {
			out = append(out, item)
		}
	}
	return out
}

// ListByStatus returns items with the given status.
func (q *Queue) ListByStatus(s ItemStatus) []WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var out []WorkItem
	for _, item := range q.items {
		if item.Status == s {
			out = append(out, item)
		}
	}
	return out
}

// HasBeadsID checks if any item already references the given beads issue ID.
func (q *Queue) HasBeadsID(beadsID string) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for _, item := range q.items {
		if item.BeadsID == beadsID {
			return true
		}
	}
	return false
}

// FindByTitle returns the first item matching the given title, or nil if none found.
func (q *Queue) FindByTitle(title string) *WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for i := range q.items {
		if q.items[i].Title == title {
			return &q.items[i]
		}
	}
	return nil
}

// FindByBranch returns the first item with the given branch name, or nil if none found.
func (q *Queue) FindByBranch(branch string) *WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	for i := range q.items {
		if q.items[i].Branch == branch {
			return &q.items[i]
		}
	}
	return nil
}

// LinkBeadsID sets the beads ID on an existing work item, linking it for future dedup.
func (q *Queue) LinkBeadsID(id, beadsID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].ID == id {
			q.items[i].BeadsID = beadsID
			q.items[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("item %s not found", id)
}

// SetBranch records the branch name associated with a work item.
func (q *Queue) SetBranch(id, branch string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].ID == id {
			q.items[i].Branch = branch
			q.items[i].UpdatedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("item %s not found", id)
}

// UpdateMergeStatus sets the merge status and optionally the commit hash.
func (q *Queue) UpdateMergeStatus(id string, ms MergeStatus, commitHash string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range q.items {
		if q.items[i].ID == id {
			q.items[i].Merge = ms
			q.items[i].UpdatedAt = time.Now()
			if ms == MergeMerged {
				q.items[i].MergedAt = time.Now()
				q.items[i].MergeCommit = commitHash
			}
			return nil
		}
	}
	return fmt.Errorf("item %s not found", id)
}

// ListMergeable returns done items that have not been merged yet.
// Items are returned in ID order (oldest first) for consistent merge ordering.
func (q *Queue) ListMergeable() []WorkItem {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var out []WorkItem
	for _, item := range q.items {
		if item.Status == StatusDone && item.Merge != MergeMerged {
			out = append(out, item)
		}
	}
	return out
}

// Stats returns a summary of queue state.
func (q *Queue) Stats() Stats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var s Stats
	s.Total = len(q.items)
	for _, item := range q.items {
		switch item.Status {
		case StatusPending:
			s.Pending++
		case StatusAssigned:
			s.Assigned++
		case StatusWorking:
			s.Working++
		case StatusDone:
			s.Done++
		case StatusFailed:
			s.Failed++
		}
		switch item.Merge {
		case MergeMerged:
			s.Merged++
		case MergeUnmerged, MergeMerging, MergeConflict:
			s.Unmerged++
		}
	}
	return s
}
