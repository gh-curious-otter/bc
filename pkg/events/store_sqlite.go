package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/db"
)

// SQLiteLog stores events in a SQLite database.
// It implements the EventStore interface.
type SQLiteLog struct {
	db *db.DB
}

// NewSQLiteLog opens the events table using the shared workspace database.
// Returns an error if no shared database is available.
func NewSQLiteLog(dbPath string) (*SQLiteLog, error) {
	d := db.SharedWrapped()
	if d == nil {
		return nil, fmt.Errorf("events store requires shared database (none available, path hint: %s)", dbPath)
	}

	schema := `
		CREATE TABLE IF NOT EXISTS events (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			type      TEXT NOT NULL,
			agent     TEXT,
			message   TEXT,
			data      TEXT,
			timestamp TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_events_agent ON events(agent);
		CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);
	`
	if _, err := d.ExecContext(context.Background(), schema); err != nil {
		_ = d.Close()
		return nil, fmt.Errorf("create events table: %w", err)
	}

	return &SQLiteLog{db: d}, nil
}

// Append writes a single event to the database.
func (l *SQLiteLog) Append(event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	var dataJSON *string
	if event.Data != nil {
		b, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("marshal event data: %w", err)
		}
		s := string(b)
		dataJSON = &s
	}

	_, err := l.db.ExecContext(context.Background(),
		"INSERT INTO events (type, agent, message, data, timestamp) VALUES (?, ?, ?, ?, ?)",
		string(event.Type),
		nilStr(event.Agent),
		nilStr(event.Message),
		dataJSON,
		event.Timestamp.Format(time.RFC3339),
	)
	return err
}

// Read returns all events ordered by timestamp.
func (l *SQLiteLog) Read() ([]Event, error) {
	rows, err := l.db.QueryContext(context.Background(),
		"SELECT type, agent, message, data, timestamp FROM events ORDER BY id ASC LIMIT 1000",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEventRows(rows)
}

// ReadLast returns the last n events.
func (l *SQLiteLog) ReadLast(n int) ([]Event, error) {
	rows, err := l.db.QueryContext(context.Background(),
		"SELECT type, agent, message, data, timestamp FROM events ORDER BY id DESC LIMIT ?", n,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	events, err := scanEventRows(rows)
	if err != nil {
		return nil, err
	}

	// Reverse so oldest first
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	return events, nil
}

// ReadByAgent returns events for a specific agent.
func (l *SQLiteLog) ReadByAgent(name string) ([]Event, error) {
	rows, err := l.db.QueryContext(context.Background(),
		"SELECT type, agent, message, data, timestamp FROM events WHERE agent = ? ORDER BY id ASC LIMIT 1000", name,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	return scanEventRows(rows)
}

// Close is a no-op — the shared DB is owned by the caller.
func (l *SQLiteLog) Close() error {
	return nil
}

// --- helpers ---

// sqlRows is the subset of *sql.Rows used by scanEventRows.
type sqlRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func scanEventRows(rows sqlRows) ([]Event, error) {
	var events []Event
	for rows.Next() {
		var ev Event
		var evType string
		var agent, message, dataJSON *string
		var ts string

		if err := rows.Scan(&evType, &agent, &message, &dataJSON, &ts); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}

		ev.Type = EventType(evType)
		if agent != nil {
			ev.Agent = *agent
		}
		if message != nil {
			ev.Message = *message
		}
		if dataJSON != nil && *dataJSON != "" {
			_ = json.Unmarshal([]byte(*dataJSON), &ev.Data) //nolint:errcheck // best-effort
		}
		ev.Timestamp, _ = time.Parse(time.RFC3339, ts)
		events = append(events, ev)
	}
	return events, rows.Err()
}

func nilStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
