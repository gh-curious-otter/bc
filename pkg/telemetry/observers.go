package telemetry

import (
	"context"

	"github.com/rpuneet/bc/pkg/log"
)

// LogObserver logs telemetry events using the log package.
type LogObserver struct {
	// Verbose enables debug-level logging
	Verbose bool
}

// NewLogObserver creates a new logging observer.
func NewLogObserver(verbose bool) *LogObserver {
	return &LogObserver{Verbose: verbose}
}

// OnEvent logs the telemetry event.
func (o *LogObserver) OnEvent(_ context.Context, event Event) {
	// Build attributes from event data
	attrs := []any{
		"type", string(event.Type),
	}

	if event.Agent != "" {
		attrs = append(attrs, "agent", event.Agent)
	}

	if event.Duration > 0 {
		attrs = append(attrs, "duration_ms", event.Duration.Milliseconds())
	}

	if event.Error != nil {
		attrs = append(attrs, "error", event.Error.Error())
	}

	// Add data fields
	for k, v := range event.Data {
		attrs = append(attrs, k, v)
	}

	// Log based on event type
	switch {
	case event.Error != nil:
		log.Error(event.Message, attrs...)
	case isErrorEvent(event.Type):
		log.Warn(event.Message, attrs...)
	case o.Verbose:
		log.Debug(event.Message, attrs...)
	default:
		log.Info(event.Message, attrs...)
	}
}

// isErrorEvent returns true if the event type indicates an error condition.
func isErrorEvent(t EventType) bool {
	switch t {
	case EventAgentError, EventWorkFailed, EventHealthUnhealthy, EventHealthDegraded, EventCostExceeded:
		return true
	default:
		return false
	}
}

// FilterObserver wraps an observer and only passes events matching the filter.
type FilterObserver struct {
	inner  Observer
	filter func(Event) bool
}

// NewFilterObserver creates a filtered observer.
func NewFilterObserver(inner Observer, filter func(Event) bool) *FilterObserver {
	return &FilterObserver{inner: inner, filter: filter}
}

// OnEvent passes the event to the inner observer if it matches the filter.
func (o *FilterObserver) OnEvent(ctx context.Context, event Event) {
	if o.filter(event) {
		o.inner.OnEvent(ctx, event)
	}
}

// TypeFilter returns a filter function that matches specific event types.
func TypeFilter(types ...EventType) func(Event) bool {
	typeSet := make(map[EventType]struct{}, len(types))
	for _, t := range types {
		typeSet[t] = struct{}{}
	}
	return func(e Event) bool {
		_, ok := typeSet[e.Type]
		return ok
	}
}

// AgentFilter returns a filter function that matches events for specific agents.
func AgentFilter(agents ...string) func(Event) bool {
	agentSet := make(map[string]struct{}, len(agents))
	for _, a := range agents {
		agentSet[a] = struct{}{}
	}
	return func(e Event) bool {
		_, ok := agentSet[e.Agent]
		return ok
	}
}

// ErrorFilter returns a filter function that matches error events.
func ErrorFilter() func(Event) bool {
	return func(e Event) bool {
		return e.Error != nil || isErrorEvent(e.Type)
	}
}

// BufferedObserver collects events in a buffer for batch processing.
//
//nolint:govet // fieldalignment: logical field grouping preferred
type BufferedObserver struct {
	mu     chan struct{} // mutex channel
	events []Event
	maxLen int
	onFull func([]Event)
}

// NewBufferedObserver creates a buffered observer.
// When the buffer reaches maxLen, onFull is called with the events and the buffer is cleared.
func NewBufferedObserver(maxLen int, onFull func([]Event)) *BufferedObserver {
	return &BufferedObserver{
		mu:     make(chan struct{}, 1),
		events: make([]Event, 0, maxLen),
		maxLen: maxLen,
		onFull: onFull,
	}
}

// OnEvent adds the event to the buffer.
func (o *BufferedObserver) OnEvent(_ context.Context, event Event) {
	o.mu <- struct{}{}
	defer func() { <-o.mu }()

	o.events = append(o.events, event)
	if len(o.events) >= o.maxLen {
		events := o.events
		o.events = make([]Event, 0, o.maxLen)
		if o.onFull != nil {
			go o.onFull(events)
		}
	}
}

// Flush processes any remaining buffered events.
func (o *BufferedObserver) Flush() []Event {
	o.mu <- struct{}{}
	defer func() { <-o.mu }()

	events := o.events
	o.events = make([]Event, 0, o.maxLen)
	return events
}

// MultiObserver dispatches events to multiple observers.
type MultiObserver struct {
	observers []Observer
}

// NewMultiObserver creates an observer that dispatches to multiple observers.
func NewMultiObserver(observers ...Observer) *MultiObserver {
	return &MultiObserver{observers: observers}
}

// OnEvent dispatches to all inner observers.
func (o *MultiObserver) OnEvent(ctx context.Context, event Event) {
	for _, obs := range o.observers {
		obs.OnEvent(ctx, event)
	}
}
