// Package events provides an append-only event log for bc.
//
// Events are stored as JSONL (one JSON object per line) at .bc/events.jsonl.
// This provides an audit trail for agent spawns, stops, work assignments,
// status reports, and messages.
package events

import (
	"bufio"
	"encoding/json"
	"os"
	"time"
)

// EventType identifies what happened.
type EventType string

const (
	AgentSpawned  EventType = "agent.spawned"
	AgentStopped  EventType = "agent.stopped"
	AgentReport   EventType = "agent.report"
	WorkAssigned  EventType = "work.assigned"
	WorkStarted   EventType = "work.started"
	WorkCompleted EventType = "work.completed"
	WorkFailed    EventType = "work.failed"
	MessageSent   EventType = "message.sent"
	QueueLoaded   EventType = "queue.loaded"
)

// Event is a single log entry.
type Event struct {
	Timestamp time.Time      `json:"ts"`
	Type      EventType      `json:"type"`
	Agent     string         `json:"agent,omitempty"`
	Message   string         `json:"message,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}

// Log manages the append-only event log file.
type Log struct {
	path string
}

// NewLog creates a Log that writes to the given file path.
func NewLog(path string) *Log {
	return &Log{path: path}
}

// Append writes a single event to the log file.
func (l *Log) Append(event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = f.Write(data)
	return err
}

// Read returns all events from the log.
func (l *Log) Read() ([]Event, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			continue // skip malformed lines
		}
		events = append(events, ev)
	}
	return events, scanner.Err()
}

// ReadLast returns the last n events.
func (l *Log) ReadLast(n int) ([]Event, error) {
	all, err := l.Read()
	if err != nil {
		return nil, err
	}
	if len(all) <= n {
		return all, nil
	}
	return all[len(all)-n:], nil
}

// ReadByAgent returns events for a specific agent.
func (l *Log) ReadByAgent(name string) ([]Event, error) {
	all, err := l.Read()
	if err != nil {
		return nil, err
	}
	var filtered []Event
	for _, ev := range all {
		if ev.Agent == name {
			filtered = append(filtered, ev)
		}
	}
	return filtered, nil
}
