// Package audit provides audit logging for compliance tracking.
package audit

import (
	"encoding/json"
	"time"
)

// EventType represents the type of audit event.
type EventType string

const (
	// Agent events
	EventAgentCreate EventType = "agent.create"
	EventAgentStart  EventType = "agent.start"
	EventAgentStop   EventType = "agent.stop"
	EventAgentDelete EventType = "agent.delete"
	EventAgentSend   EventType = "agent.send"

	// Channel events
	EventChannelCreate  EventType = "channel.create"
	EventChannelMessage EventType = "channel.message"
	EventChannelJoin    EventType = "channel.join"
	EventChannelLeave   EventType = "channel.leave"

	// Cost events
	EventCostTransaction EventType = "cost.transaction"
	EventCostBudgetSet   EventType = "cost.budget_set"

	// Config events
	EventConfigChange EventType = "config.change"

	// Permission events
	EventPermissionGrant  EventType = "permission.grant"
	EventPermissionRevoke EventType = "permission.revoke"
)

// Event represents an audit log entry.
type Event struct {
	Timestamp time.Time         `json:"timestamp"`
	Details   map[string]string `json:"details"` // Additional context
	Type      EventType         `json:"type"`
	Actor     string            `json:"actor"`     // Who performed the action (agent name or "user")
	Target    string            `json:"target"`    // What was affected
	Workspace string            `json:"workspace"` // Workspace name
	ID        int64             `json:"id"`
}

// NewEvent creates a new audit event.
func NewEvent(eventType EventType, actor, target string) *Event {
	return &Event{
		Timestamp: time.Now().UTC(),
		Type:      eventType,
		Actor:     actor,
		Target:    target,
		Details:   make(map[string]string),
	}
}

// WithDetail adds a detail to the event.
func (e *Event) WithDetail(key, value string) *Event {
	e.Details[key] = value
	return e
}

// WithWorkspace sets the workspace for the event.
func (e *Event) WithWorkspace(ws string) *Event {
	e.Workspace = ws
	return e
}

// JSON returns the event as JSON bytes.
func (e *Event) JSON() ([]byte, error) {
	return json.Marshal(e)
}

// Store defines the interface for audit log storage.
type Store interface {
	// Log records an audit event.
	Log(event *Event) error

	// Query retrieves events matching the filter.
	Query(filter *Filter) ([]*Event, error)

	// Close closes the store.
	Close() error
}

// Filter defines criteria for querying audit events.
type Filter struct {
	Since     time.Time
	Until     time.Time
	Actor     string
	Target    string
	Workspace string
	Types     []EventType
	Limit     int
}

// NewFilter creates a new empty filter.
func NewFilter() *Filter {
	return &Filter{
		Limit: 100, // Default limit
	}
}

// WithTypes sets the event type filter.
func (f *Filter) WithTypes(types ...EventType) *Filter {
	f.Types = types
	return f
}

// WithActor sets the actor filter.
func (f *Filter) WithActor(actor string) *Filter {
	f.Actor = actor
	return f
}

// WithTarget sets the target filter.
func (f *Filter) WithTarget(target string) *Filter {
	f.Target = target
	return f
}

// WithTimeRange sets the time range filter.
func (f *Filter) WithTimeRange(since, until time.Time) *Filter {
	f.Since = since
	f.Until = until
	return f
}

// WithLimit sets the max number of results.
func (f *Filter) WithLimit(limit int) *Filter {
	f.Limit = limit
	return f
}
