// Package ws implements a Server-Sent Events (SSE) hub for real-time event
// broadcasting to connected web clients.
package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gh-curious-otter/bc/pkg/log"
)

// Event is the payload broadcast to SSE subscribers.
type Event struct {
	Data any    `json:"data"`
	Type string `json:"type"`
}

// EventWriter persists SSE events for later retrieval.
type EventWriter interface {
	Write(eventType string, data any) error
}

type subscriber struct {
	ch   chan []byte
	done <-chan struct{}
}

// Hub manages SSE subscribers and broadcasts events.
// It implements agent.EventPublisher so it can be wired into AgentService.
type Hub struct {
	subscribers map[*subscriber]struct{}
	broadcast   chan []byte
	done        chan struct{}
	writer      EventWriter
	mu          sync.RWMutex
	stopOnce    sync.Once
}

// NewHub creates and returns a new Hub. Call Run() in a goroutine.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[*subscriber]struct{}),
		broadcast:   make(chan []byte, 256),
		done:        make(chan struct{}),
	}
}

// SetWriter attaches an EventWriter for persisting broadcast events.
func (h *Hub) SetWriter(w EventWriter) {
	h.writer = w
}

// Run processes the broadcast channel until Stop is called.
func (h *Hub) Run() {
	for {
		select {
		case msg := <-h.broadcast:
			h.send(msg)
		case <-h.done:
			return
		}
	}
}

// Stop shuts down the hub's Run loop. Safe to call multiple times.
func (h *Hub) Stop() {
	h.stopOnce.Do(func() { close(h.done) })
}

// Publish implements agent.EventPublisher.
// Data is redacted before broadcast to prevent secrets from leaking to the UI.
// If an EventWriter is attached, the event is also persisted to disk.
func (h *Hub) Publish(eventType string, data map[string]any) {
	redacted := RedactMap(data)
	evt := Event{Type: eventType, Data: redacted}
	msg, err := json.Marshal(evt)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- msg:
	default:
		log.Debug("event dropped: broadcast buffer full", "type", eventType)
	}

	// Persist to JSONL (best-effort)
	if h.writer != nil {
		if wErr := h.writer.Write(eventType, redacted); wErr != nil {
			log.Debug("event persist failed", "type", eventType, "error", wErr)
		}
	}
}

// ClientCount returns the number of active SSE connections.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

// ServeHTTP serves an SSE stream to a single client.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sub := &subscriber{
		ch:   make(chan []byte, 64),
		done: r.Context().Done(),
	}
	h.mu.Lock()
	h.subscribers[sub] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.subscribers, sub)
		h.mu.Unlock()
	}()

	// Send initial connected event.
	connected, _ := json.Marshal(Event{Type: "connected", Data: map[string]any{}})
	fmt.Fprintf(w, "data: %s\n\n", connected) //nolint:errcheck // writing to response
	flusher.Flush()

	for {
		select {
		case msg := <-sub.ch:
			fmt.Fprintf(w, "data: %s\n\n", msg) //nolint:errcheck // writing to response
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (h *Hub) send(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for sub := range h.subscribers {
		select {
		case sub.ch <- msg:
		default: // subscriber too slow — skip
		}
	}
}
