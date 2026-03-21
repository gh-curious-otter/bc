// Package events provides an append-only event log for bc.
//
// Events are stored as JSONL (one JSON object per line) at .bc/events.jsonl.
// This provides an audit trail for agent spawns, stops, work assignments,
// status reports, and messages.
package events

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// EventType identifies what happened.
type EventType string

const (
	AgentSpawned    EventType = "agent.spawned"
	AgentStopped    EventType = "agent.stopped"
	AgentReport     EventType = "agent.report"
	WorkAssigned    EventType = "work.assigned"
	WorkStarted     EventType = "work.started"
	WorkCompleted   EventType = "work.completed"
	WorkFailed      EventType = "work.failed"
	MessageSent     EventType = "message.sent"
	QueueLoaded     EventType = "queue.loaded"
	HealthCheck     EventType = "health.check"
	HealthFailed    EventType = "health.failed"
	HealthRecovered EventType = "health.recovered"
)

const (
	// DefaultMaxFileSize is the size threshold (in bytes) that triggers rotation.
	DefaultMaxFileSize int64 = 10 * 1024 * 1024 // 10 MB
	// DefaultMaxRotatedFiles is the number of rotated files to keep.
	DefaultMaxRotatedFiles = 5
	// DefaultReadLimit caps the number of events returned by Read and ReadByAgent
	// to prevent unbounded memory usage. Matches the SQLite store limit.
	DefaultReadLimit = 1000
	// MaxReadLastLimit caps the value of n in ReadLast to prevent abuse.
	MaxReadLastLimit = 10000
)

// Event is a single log entry.
type Event struct {
	Data      map[string]any `json:"data,omitempty"`
	Timestamp time.Time      `json:"ts"`
	Type      EventType      `json:"type"`
	Agent     string         `json:"agent,omitempty"`
	Message   string         `json:"message,omitempty"`
}

// EventStore is the interface for reading and writing events.
// Both the file-based Log and SQLiteLog implement this interface.
type EventStore interface {
	Append(event Event) error
	Read() ([]Event, error)
	ReadLast(n int) ([]Event, error)
	ReadByAgent(name string) ([]Event, error)
	Close() error
}

// Log manages the append-only event log file.
type Log struct {
	path            string
	maxFileSize     int64
	maxRotatedFiles int
}

// NewLog creates a Log that writes to the given file path.
func NewLog(path string) *Log {
	return &Log{
		path:            path,
		maxFileSize:     DefaultMaxFileSize,
		maxRotatedFiles: DefaultMaxRotatedFiles,
	}
}

// Append writes a single event to the log file.
// After writing, if the file exceeds maxFileSize the log is rotated.
func (l *Log) Append(event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	data, err := json.Marshal(event)
	if err != nil {
		_ = f.Close() //nolint:errcheck // closing on error path
		return err
	}
	data = append(data, '\n')
	if _, err = f.Write(data); err != nil {
		_ = f.Close() //nolint:errcheck // closing on error path
		return err
	}

	if l.maxFileSize > 0 {
		if info, statErr := f.Stat(); statErr == nil && info.Size() >= l.maxFileSize {
			_ = f.Close() //nolint:errcheck // closing before rotate
			l.rotate()
			return nil
		}
	}

	return f.Close()
}

// rotate shifts rotated log files and renames the current file.
// events.jsonl -> events.jsonl.1, events.jsonl.1 -> events.jsonl.2, etc.
// Files beyond maxRotatedFiles are removed.
func (l *Log) rotate() {
	oldest := fmt.Sprintf("%s.%d", l.path, l.maxRotatedFiles)
	_ = os.Remove(oldest) //nolint:errcheck // best-effort rotation cleanup
	for i := l.maxRotatedFiles - 1; i >= 1; i-- {
		from := fmt.Sprintf("%s.%d", l.path, i)
		to := fmt.Sprintf("%s.%d", l.path, i+1)
		_ = os.Rename(from, to) //nolint:errcheck // best-effort rotation
	}
	_ = os.Rename(l.path, fmt.Sprintf("%s.1", l.path)) //nolint:errcheck // best-effort rotation
}

// Read returns events from the log, capped at DefaultReadLimit to prevent
// unbounded memory usage.
func (l *Log) Read() ([]Event, error) {
	return l.readWithLimit(DefaultReadLimit)
}

// readAll returns all events without a cap. Used internally by ReadLast
// which applies its own limit.
func (l *Log) readAll() ([]Event, error) {
	return l.readWithLimit(0)
}

// readWithLimit reads events from the log file. If limit > 0, at most limit
// events are returned. If limit <= 0, all events are returned.
func (l *Log) readWithLimit(limit int) ([]Event, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // deferred close

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // skip malformed lines
		}
		events = append(events, ev)
		if limit > 0 && len(events) >= limit {
			break
		}
	}
	return events, scanner.Err()
}

// ReadLast returns the last n events. The value of n is capped at
// MaxReadLastLimit to prevent excessive memory usage.
func (l *Log) ReadLast(n int) ([]Event, error) {
	if n <= 0 {
		return nil, nil
	}
	if n > MaxReadLastLimit {
		n = MaxReadLastLimit
	}
	all, err := l.readAll()
	if err != nil {
		return nil, err
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// Close is a no-op for the file-based log (satisfies EventStore interface).
func (l *Log) Close() error {
	return nil
}

// ReadByAgent returns events for a specific agent, capped at DefaultReadLimit.
func (l *Log) ReadByAgent(name string) ([]Event, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }() //nolint:errcheck // deferred close

	var filtered []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // skip malformed lines
		}
		if ev.Agent == name {
			filtered = append(filtered, ev)
			if len(filtered) >= DefaultReadLimit {
				break
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return filtered, nil
}
