// Package queue implements dual queue system for bc.
//
// The dual queue system provides:
//   - Work Queue: Tasks assigned TO an agent (incoming work)
//   - Merge Queue: Completed branches awaiting review FROM children
//
// This enables hierarchical task flow:
//
//	ROOT -> MANAGER (work) -> ENGINEER (work) -> MANAGER (merge) -> ROOT (merge)
//
// Issue #1234: Dual Queue System for Phase 3 Enterprise
package queue

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Work item status constants
const (
	StatusPending    = "pending"
	StatusAccepted   = "accepted"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

// Merge item status constants
const (
	MergeStatusPending  = "pending"
	MergeStatusReviewed = "reviewed"
	MergeStatusApproved = "approved"
	MergeStatusMerged   = "merged"
	MergeStatusRejected = "rejected"
	MergeStatusConflict = "conflict"
)

// Priority levels
const (
	PriorityLow    = 0
	PriorityNormal = 1
	PriorityHigh   = 2
	PriorityUrgent = 3
)

// WorkItem represents a task in an agent's work queue
type WorkItem struct {
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	AcceptedAt  *time.Time `json:"acceptedAt,omitempty"`
	AgentID     string     `json:"agentId"`
	Title       string     `json:"title"`
	Description string     `json:"description,omitempty"`
	Status      string     `json:"status"`
	FromAgent   string     `json:"fromAgent,omitempty"`
	IssueRef    string     `json:"issueRef,omitempty"`
	Branch      string     `json:"branch,omitempty"`
	Metadata    string     `json:"metadata,omitempty"`
	ID          int64      `json:"id"`
	Priority    int        `json:"priority"`
}

// MergeItem represents a branch awaiting review in an agent's merge queue
type MergeItem struct {
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
	ReviewedAt *time.Time `json:"reviewedAt,omitempty"`
	MergedAt   *time.Time `json:"mergedAt,omitempty"`
	AgentID    string     `json:"agentId"`
	Branch     string     `json:"branch"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	FromAgent  string     `json:"fromAgent"`
	IssueRef   string     `json:"issueRef,omitempty"`
	Reviewer   string     `json:"reviewer,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	Metadata   string     `json:"metadata,omitempty"`
	ID         int64      `json:"id"`
}

// Store manages the dual queue system
type Store struct {
	db   *sql.DB
	path string
}

// NewStore creates a new queue store
func NewStore(stateDir string) *Store {
	return &Store{
		path: filepath.Join(stateDir, "bc.db"),
	}
}

// Open opens the database connection and initializes schema
func (s *Store) Open(ctx context.Context) error {
	db, err := sql.Open("sqlite3", s.path+"?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("failed to open queue database: %w", err)
	}

	// SQLite single-writer model
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Set pragmas for performance
	pragmas := `
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -2000;
		PRAGMA temp_store = MEMORY;
	`
	if _, err := db.ExecContext(ctx, pragmas); err != nil {
		db.Close() //nolint:errcheck // closing on error
		return fmt.Errorf("failed to set pragmas: %w", err)
	}

	if err := s.initSchema(ctx, db); err != nil {
		db.Close() //nolint:errcheck // closing on error
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	s.db = db
	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) initSchema(ctx context.Context, db *sql.DB) error {
	schema := `
		-- Work Queue: tasks assigned TO agents
		CREATE TABLE IF NOT EXISTS work_queue (
			id INTEGER PRIMARY KEY,
			agent_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			priority INTEGER NOT NULL DEFAULT 1,
			from_agent TEXT,
			issue_ref TEXT,
			branch TEXT,
			metadata TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			accepted_at TIMESTAMP,
			completed_at TIMESTAMP
		);

		-- Merge Queue: branches awaiting review FROM children
		CREATE TABLE IF NOT EXISTS merge_queue (
			id INTEGER PRIMARY KEY,
			agent_id TEXT NOT NULL,
			branch TEXT NOT NULL,
			title TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			from_agent TEXT NOT NULL,
			issue_ref TEXT,
			reviewer TEXT,
			reason TEXT,
			metadata TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			reviewed_at TIMESTAMP,
			merged_at TIMESTAMP
		);

		-- Indexes for work queue queries
		CREATE INDEX IF NOT EXISTS idx_work_queue_agent_status
			ON work_queue(agent_id, status);
		CREATE INDEX IF NOT EXISTS idx_work_queue_priority
			ON work_queue(priority DESC, created_at ASC);
		CREATE INDEX IF NOT EXISTS idx_work_queue_from_agent
			ON work_queue(from_agent);

		-- Indexes for merge queue queries
		CREATE INDEX IF NOT EXISTS idx_merge_queue_agent_status
			ON merge_queue(agent_id, status);
		CREATE INDEX IF NOT EXISTS idx_merge_queue_from_agent
			ON merge_queue(from_agent);
		CREATE INDEX IF NOT EXISTS idx_merge_queue_branch
			ON merge_queue(branch);
	`

	_, err := db.ExecContext(ctx, schema)
	return err
}

// === Work Queue Operations ===

// AddWork adds a new item to an agent's work queue
func (s *Store) AddWork(ctx context.Context, item *WorkItem) error {
	query := `
		INSERT INTO work_queue (agent_id, title, description, status, priority, from_agent, issue_ref, branch, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		item.AgentID, item.Title, item.Description, item.Status, item.Priority,
		item.FromAgent, item.IssueRef, item.Branch, item.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to add work item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get work item ID: %w", err)
	}
	item.ID = id
	return nil
}

// GetWork retrieves a work item by ID
func (s *Store) GetWork(ctx context.Context, id int64) (*WorkItem, error) {
	query := `
		SELECT id, agent_id, title, description, status, priority, from_agent,
			   issue_ref, branch, metadata, created_at, updated_at, accepted_at, completed_at
		FROM work_queue WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanWorkItem(row)
}

// ListWork lists work items for an agent, optionally filtered by status
func (s *Store) ListWork(ctx context.Context, agentID string, status string) ([]*WorkItem, error) {
	var query string
	var args []any

	if status == "" {
		query = `
			SELECT id, agent_id, title, description, status, priority, from_agent,
				   issue_ref, branch, metadata, created_at, updated_at, accepted_at, completed_at
			FROM work_queue WHERE agent_id = ?
			ORDER BY priority DESC, created_at ASC
		`
		args = []any{agentID}
	} else {
		query = `
			SELECT id, agent_id, title, description, status, priority, from_agent,
				   issue_ref, branch, metadata, created_at, updated_at, accepted_at, completed_at
			FROM work_queue WHERE agent_id = ? AND status = ?
			ORDER BY priority DESC, created_at ASC
		`
		args = []any{agentID, status}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list work items: %w", err)
	}
	defer rows.Close() //nolint:errcheck // deferred close

	var items []*WorkItem
	for rows.Next() {
		item, err := s.scanWorkItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// AcceptWork marks a work item as accepted
func (s *Store) AcceptWork(ctx context.Context, id int64) error {
	query := `
		UPDATE work_queue
		SET status = ?, accepted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`
	result, err := s.db.ExecContext(ctx, query, StatusAccepted, id, StatusPending)
	if err != nil {
		return fmt.Errorf("failed to accept work item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("work item %d not found or not pending", id)
	}
	return nil
}

// StartWork marks a work item as in progress
func (s *Store) StartWork(ctx context.Context, id int64) error {
	query := `
		UPDATE work_queue
		SET status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`
	result, err := s.db.ExecContext(ctx, query, StatusInProgress, id, StatusAccepted)
	if err != nil {
		return fmt.Errorf("failed to start work item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("work item %d not found or not accepted", id)
	}
	return nil
}

// CompleteWork marks a work item as completed
func (s *Store) CompleteWork(ctx context.Context, id int64, branch string) error {
	query := `
		UPDATE work_queue
		SET status = ?, branch = ?, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`
	result, err := s.db.ExecContext(ctx, query, StatusCompleted, branch, id, StatusInProgress)
	if err != nil {
		return fmt.Errorf("failed to complete work item: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("work item %d not found or not in progress", id)
	}
	return nil
}

// === Merge Queue Operations ===

// AddMerge adds a new item to an agent's merge queue
func (s *Store) AddMerge(ctx context.Context, item *MergeItem) error {
	query := `
		INSERT INTO merge_queue (agent_id, branch, title, status, from_agent, issue_ref, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		item.AgentID, item.Branch, item.Title, item.Status,
		item.FromAgent, item.IssueRef, item.Metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to add merge item: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get merge item ID: %w", err)
	}
	item.ID = id
	return nil
}

// GetMerge retrieves a merge item by ID
func (s *Store) GetMerge(ctx context.Context, id int64) (*MergeItem, error) {
	query := `
		SELECT id, agent_id, branch, title, status, from_agent, issue_ref,
			   reviewer, reason, metadata, created_at, updated_at, reviewed_at, merged_at
		FROM merge_queue WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)
	return s.scanMergeItem(row)
}

// GetMergeByBranch retrieves a merge item by branch name for an agent
func (s *Store) GetMergeByBranch(ctx context.Context, agentID, branch string) (*MergeItem, error) {
	query := `
		SELECT id, agent_id, branch, title, status, from_agent, issue_ref,
			   reviewer, reason, metadata, created_at, updated_at, reviewed_at, merged_at
		FROM merge_queue WHERE agent_id = ? AND branch = ? AND status NOT IN (?, ?)
		ORDER BY created_at DESC LIMIT 1
	`
	row := s.db.QueryRowContext(ctx, query, agentID, branch, MergeStatusMerged, MergeStatusRejected)
	return s.scanMergeItem(row)
}

// ListMerge lists merge items for an agent, optionally filtered by status
func (s *Store) ListMerge(ctx context.Context, agentID string, status string) ([]*MergeItem, error) {
	var query string
	var args []any

	if status == "" {
		query = `
			SELECT id, agent_id, branch, title, status, from_agent, issue_ref,
				   reviewer, reason, metadata, created_at, updated_at, reviewed_at, merged_at
			FROM merge_queue WHERE agent_id = ?
			ORDER BY created_at ASC
		`
		args = []any{agentID}
	} else {
		query = `
			SELECT id, agent_id, branch, title, status, from_agent, issue_ref,
				   reviewer, reason, metadata, created_at, updated_at, reviewed_at, merged_at
			FROM merge_queue WHERE agent_id = ? AND status = ?
			ORDER BY created_at ASC
		`
		args = []any{agentID, status}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge items: %w", err)
	}
	defer rows.Close() //nolint:errcheck // deferred close

	var items []*MergeItem
	for rows.Next() {
		item, err := s.scanMergeItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ApproveMerge approves a merge item
func (s *Store) ApproveMerge(ctx context.Context, id int64, reviewer string) error {
	query := `
		UPDATE merge_queue
		SET status = ?, reviewer = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN (?, ?)
	`
	result, err := s.db.ExecContext(ctx, query, MergeStatusApproved, reviewer, id, MergeStatusPending, MergeStatusReviewed)
	if err != nil {
		return fmt.Errorf("failed to approve merge: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("merge item %d not found or not pending", id)
	}
	return nil
}

// RejectMerge rejects a merge item with a reason
func (s *Store) RejectMerge(ctx context.Context, id int64, reviewer, reason string) error {
	query := `
		UPDATE merge_queue
		SET status = ?, reviewer = ?, reason = ?, reviewed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status IN (?, ?)
	`
	result, err := s.db.ExecContext(ctx, query, MergeStatusRejected, reviewer, reason, id, MergeStatusPending, MergeStatusReviewed)
	if err != nil {
		return fmt.Errorf("failed to reject merge: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("merge item %d not found or not pending", id)
	}
	return nil
}

// CompleteMerge marks a merge item as merged
func (s *Store) CompleteMerge(ctx context.Context, id int64) error {
	query := `
		UPDATE merge_queue
		SET status = ?, merged_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = ?
	`
	result, err := s.db.ExecContext(ctx, query, MergeStatusMerged, id, MergeStatusApproved)
	if err != nil {
		return fmt.Errorf("failed to complete merge: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check affected rows: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("merge item %d not found or not approved", id)
	}
	return nil
}

// === Submit Operation ===

// Submit submits a completed work item to a target agent's merge queue
func (s *Store) Submit(ctx context.Context, workID int64, toAgent string) (*MergeItem, error) {
	// Get the work item
	work, err := s.GetWork(ctx, workID)
	if err != nil {
		return nil, fmt.Errorf("failed to get work item: %w", err)
	}

	if work.Status != StatusCompleted {
		return nil, fmt.Errorf("work item %d is not completed", workID)
	}

	if work.Branch == "" {
		return nil, fmt.Errorf("work item %d has no branch", workID)
	}

	// Create merge item for target agent
	mergeItem := &MergeItem{
		AgentID:   toAgent,
		Branch:    work.Branch,
		Title:     work.Title,
		Status:    MergeStatusPending,
		FromAgent: work.AgentID,
		IssueRef:  work.IssueRef,
	}

	if err := s.AddMerge(ctx, mergeItem); err != nil {
		return nil, fmt.Errorf("failed to submit to merge queue: %w", err)
	}

	return mergeItem, nil
}

// === Helper functions ===

func (s *Store) scanWorkItem(row *sql.Row) (*WorkItem, error) {
	var item WorkItem
	var description, fromAgent, issueRef, branch, metadata sql.NullString
	var acceptedAt, completedAt sql.NullTime

	err := row.Scan(
		&item.ID, &item.AgentID, &item.Title, &description, &item.Status, &item.Priority,
		&fromAgent, &issueRef, &branch, &metadata,
		&item.CreatedAt, &item.UpdatedAt, &acceptedAt, &completedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("work item not found")
		}
		return nil, fmt.Errorf("failed to scan work item: %w", err)
	}

	item.Description = description.String
	item.FromAgent = fromAgent.String
	item.IssueRef = issueRef.String
	item.Branch = branch.String
	item.Metadata = metadata.String
	if acceptedAt.Valid {
		item.AcceptedAt = &acceptedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}

	return &item, nil
}

func (s *Store) scanWorkItemRows(rows *sql.Rows) (*WorkItem, error) {
	var item WorkItem
	var description, fromAgent, issueRef, branch, metadata sql.NullString
	var acceptedAt, completedAt sql.NullTime

	err := rows.Scan(
		&item.ID, &item.AgentID, &item.Title, &description, &item.Status, &item.Priority,
		&fromAgent, &issueRef, &branch, &metadata,
		&item.CreatedAt, &item.UpdatedAt, &acceptedAt, &completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan work item: %w", err)
	}

	item.Description = description.String
	item.FromAgent = fromAgent.String
	item.IssueRef = issueRef.String
	item.Branch = branch.String
	item.Metadata = metadata.String
	if acceptedAt.Valid {
		item.AcceptedAt = &acceptedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}

	return &item, nil
}

func (s *Store) scanMergeItem(row *sql.Row) (*MergeItem, error) {
	var item MergeItem
	var issueRef, reviewer, reason, metadata sql.NullString
	var reviewedAt, mergedAt sql.NullTime

	err := row.Scan(
		&item.ID, &item.AgentID, &item.Branch, &item.Title, &item.Status, &item.FromAgent,
		&issueRef, &reviewer, &reason, &metadata,
		&item.CreatedAt, &item.UpdatedAt, &reviewedAt, &mergedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("merge item not found")
		}
		return nil, fmt.Errorf("failed to scan merge item: %w", err)
	}

	item.IssueRef = issueRef.String
	item.Reviewer = reviewer.String
	item.Reason = reason.String
	item.Metadata = metadata.String
	if reviewedAt.Valid {
		item.ReviewedAt = &reviewedAt.Time
	}
	if mergedAt.Valid {
		item.MergedAt = &mergedAt.Time
	}

	return &item, nil
}

func (s *Store) scanMergeItemRows(rows *sql.Rows) (*MergeItem, error) {
	var item MergeItem
	var issueRef, reviewer, reason, metadata sql.NullString
	var reviewedAt, mergedAt sql.NullTime

	err := rows.Scan(
		&item.ID, &item.AgentID, &item.Branch, &item.Title, &item.Status, &item.FromAgent,
		&issueRef, &reviewer, &reason, &metadata,
		&item.CreatedAt, &item.UpdatedAt, &reviewedAt, &mergedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan merge item: %w", err)
	}

	item.IssueRef = issueRef.String
	item.Reviewer = reviewer.String
	item.Reason = reason.String
	item.Metadata = metadata.String
	if reviewedAt.Valid {
		item.ReviewedAt = &reviewedAt.Time
	}
	if mergedAt.Valid {
		item.MergedAt = &mergedAt.Time
	}

	return &item, nil
}
