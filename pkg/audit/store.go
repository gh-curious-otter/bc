package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteStore creates a new SQLite audit store.
func NewSQLiteStore(workspaceDir string) *SQLiteStore {
	return &SQLiteStore{
		path: filepath.Join(workspaceDir, ".bc", "audit.db"),
	}
}

// Open opens the database connection and creates tables if needed.
func (s *SQLiteStore) Open() error {
	db, err := sql.Open("sqlite3", s.path)
	if err != nil {
		return fmt.Errorf("failed to open audit database: %w", err)
	}
	s.db = db

	// Create tables
	ctx := context.Background()
	_, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			type TEXT NOT NULL,
			actor TEXT NOT NULL,
			target TEXT NOT NULL,
			details TEXT,
			workspace TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp);
		CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_events(type);
		CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_events(actor);
	`)
	if err != nil {
		return fmt.Errorf("failed to create audit tables: %w", err)
	}

	return nil
}

// Log records an audit event.
func (s *SQLiteStore) Log(event *Event) error {
	if s.db == nil {
		return fmt.Errorf("audit store not opened")
	}

	details, err := json.Marshal(event.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}

	ctx := context.Background()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO audit_events (timestamp, type, actor, target, details, workspace)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.Timestamp.Format(time.RFC3339), event.Type, event.Actor, event.Target, string(details), event.Workspace)
	if err != nil {
		return fmt.Errorf("failed to insert audit event: %w", err)
	}

	return nil
}

// Query retrieves events matching the filter.
func (s *SQLiteStore) Query(filter *Filter) ([]*Event, error) {
	if s.db == nil {
		return nil, fmt.Errorf("audit store not opened")
	}

	// Build query
	query := "SELECT id, timestamp, type, actor, target, details, workspace FROM audit_events WHERE 1=1"
	var args []any

	if len(filter.Types) > 0 {
		placeholders := make([]string, len(filter.Types))
		for i, t := range filter.Types {
			placeholders[i] = "?"
			args = append(args, string(t))
		}
		query += " AND type IN (" + strings.Join(placeholders, ",") + ")"
	}

	if filter.Actor != "" {
		query += " AND actor = ?"
		args = append(args, filter.Actor)
	}

	if filter.Target != "" {
		query += " AND target = ?"
		args = append(args, filter.Target)
	}

	if !filter.Since.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.Since.Format(time.RFC3339))
	}

	if !filter.Until.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.Until.Format(time.RFC3339))
	}

	if filter.Workspace != "" {
		query += " AND workspace = ?"
		args = append(args, filter.Workspace)
	}

	query += " ORDER BY timestamp DESC"

	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit events: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var events []*Event
	for rows.Next() {
		var e Event
		var timestamp string
		var detailsJSON string
		var workspace sql.NullString

		err := rows.Scan(&e.ID, &timestamp, &e.Type, &e.Actor, &e.Target, &detailsJSON, &workspace)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit event: %w", err)
		}

		e.Timestamp, _ = time.Parse(time.RFC3339, timestamp)
		if detailsJSON != "" {
			_ = json.Unmarshal([]byte(detailsJSON), &e.Details)
		}
		if e.Details == nil {
			e.Details = make(map[string]string)
		}
		if workspace.Valid {
			e.Workspace = workspace.String
		}

		events = append(events, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit events: %w", err)
	}

	return events, nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Export exports events to JSON format.
func (s *SQLiteStore) Export(filter *Filter) ([]byte, error) {
	events, err := s.Query(filter)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(events, "", "  ")
}

// ExportCSV exports events to CSV format.
func (s *SQLiteStore) ExportCSV(filter *Filter) (string, error) {
	events, err := s.Query(filter)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString("id,timestamp,type,actor,target,workspace,details\n")
	for _, e := range events {
		details, _ := json.Marshal(e.Details)
		sb.WriteString(fmt.Sprintf("%d,%s,%s,%s,%s,%s,%q\n",
			e.ID,
			e.Timestamp.Format(time.RFC3339),
			e.Type,
			e.Actor,
			e.Target,
			e.Workspace,
			string(details),
		))
	}
	return sb.String(), nil
}
