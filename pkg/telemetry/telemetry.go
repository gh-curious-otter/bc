// Package telemetry provides observability hooks for bc operations.
//
// The telemetry package implements an observer pattern that allows
// multiple observers to receive notifications about key operations:
//   - Agent lifecycle (spawn, stop, state changes)
//   - Channel operations (send, receive)
//   - Cost events (API calls, budget alerts)
//   - Health events (checks, failures, recoveries)
//
// Observers can be used for:
//   - Structured logging
//   - Metrics collection
//   - Plugin hooks
//   - Future OpenTelemetry integration
//
// Issue #1661: Add telemetry/observability hooks
package telemetry

import (
	"context"
	"sync"
	"time"
)

// EventType identifies the category of telemetry event.
type EventType string

// Agent lifecycle events
const (
	EventAgentSpawn       EventType = "agent.spawn"
	EventAgentStop        EventType = "agent.stop"
	EventAgentStateChange EventType = "agent.state_change"
	EventAgentError       EventType = "agent.error"
)

// Channel events
const (
	EventChannelSend    EventType = "channel.send"
	EventChannelReceive EventType = "channel.receive"
	EventChannelCreate  EventType = "channel.create"
	EventChannelDelete  EventType = "channel.delete"
)

// Cost events
const (
	EventCostRecord   EventType = "cost.record"
	EventCostBudget   EventType = "cost.budget"
	EventCostAlert    EventType = "cost.alert"
	EventCostExceeded EventType = "cost.exceeded"
)

// Health events
const (
	EventHealthCheck     EventType = "health.check"
	EventHealthDegraded  EventType = "health.degraded"
	EventHealthUnhealthy EventType = "health.unhealthy"
	EventHealthRecovered EventType = "health.recovered"
)

// Work events
const (
	EventWorkAssigned  EventType = "work.assigned"
	EventWorkStarted   EventType = "work.started"
	EventWorkCompleted EventType = "work.completed"
	EventWorkFailed    EventType = "work.failed"
)

// Event represents a telemetry event with metadata.
//
//nolint:govet // fieldalignment: logical field grouping preferred over memory optimization
type Event struct {
	// Type identifies the event category
	Type EventType
	// Timestamp when the event occurred
	Timestamp time.Time
	// Agent name if applicable
	Agent string
	// Message is a human-readable description
	Message string
	// Data contains event-specific structured data
	Data map[string]any
	// Duration for timed operations (optional)
	Duration time.Duration
	// Error if the event represents a failure
	Error error
	// TraceID for distributed tracing (optional)
	TraceID string
	// SpanID for distributed tracing (optional)
	SpanID string
}

// Observer receives telemetry events.
type Observer interface {
	// OnEvent is called when a telemetry event occurs.
	// Implementations should be non-blocking.
	OnEvent(ctx context.Context, event Event)
}

// ObserverFunc is a function adapter for Observer.
type ObserverFunc func(ctx context.Context, event Event)

// OnEvent implements Observer.
func (f ObserverFunc) OnEvent(ctx context.Context, event Event) {
	f(ctx, event)
}

// Registry manages telemetry observers.
//
//nolint:govet // fieldalignment: struct is small, readability preferred
type Registry struct {
	mu        sync.RWMutex
	observers []Observer
	enabled   bool
}

// global is the default global registry.
var global = &Registry{enabled: true}

// Register adds an observer to the global registry.
func Register(observer Observer) {
	global.Register(observer)
}

// Emit sends an event to all registered observers.
func Emit(ctx context.Context, event Event) {
	global.Emit(ctx, event)
}

// EmitAsync sends an event to all registered observers asynchronously.
func EmitAsync(ctx context.Context, event Event) {
	global.EmitAsync(ctx, event)
}

// SetEnabled enables or disables telemetry globally.
func SetEnabled(enabled bool) {
	global.SetEnabled(enabled)
}

// Register adds an observer to this registry.
func (r *Registry) Register(observer Observer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = append(r.observers, observer)
}

// Emit sends an event to all registered observers synchronously.
func (r *Registry) Emit(ctx context.Context, event Event) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.enabled {
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	for _, obs := range r.observers {
		obs.OnEvent(ctx, event)
	}
}

// EmitAsync sends an event to all registered observers asynchronously.
// Each observer is called in its own goroutine.
func (r *Registry) EmitAsync(ctx context.Context, event Event) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.enabled {
		return
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	for _, obs := range r.observers {
		go obs.OnEvent(ctx, event)
	}
}

// SetEnabled enables or disables this registry.
func (r *Registry) SetEnabled(enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = enabled
}

// Clear removes all observers from this registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.observers = nil
}

// ObserverCount returns the number of registered observers.
func (r *Registry) ObserverCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.observers)
}
