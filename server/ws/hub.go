// Package ws implements a Server-Sent Events (SSE) hub for real-time event
// broadcasting to connected web clients.
package ws

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// Event is the payload broadcast to SSE subscribers.
type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type subscriber struct {
	ch   chan []byte
	done <-chan struct{}
}

// Hub manages SSE subscribers and broadcasts events.
// It implements agent.EventPublisher so it can be wired into AgentService.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[*subscriber]struct{}
	broadcast   chan []byte
	done        chan struct{}
}

// NewHub creates and returns a new Hub. Call Run() in a goroutine.
func NewHub() *Hub {
	return &Hub{
		subscribers: make(map[*subscriber]struct{}),
		broadcast:   make(chan []byte, 256),
		done:        make(chan struct{}),
	}
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

// Stop shuts down the hub's Run loop.
func (h *Hub) Stop() {
	close(h.done)
}

// Publish implements agent.EventPublisher.
func (h *Hub) Publish(eventType string, data map[string]any) {
	evt := Event{Type: eventType, Payload: data}
	msg, err := json.Marshal(evt)
	if err != nil {
		return
	}
	select {
	case h.broadcast <- msg:
	default: // drop if full — clients should reconnect
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
	connected, _ := json.Marshal(Event{Type: "connected", Payload: map[string]any{}})
	fmt.Fprintf(w, "data: %s\n\n", connected)
	flusher.Flush()

	for {
		select {
		case msg := <-sub.ch:
			fmt.Fprintf(w, "data: %s\n\n", msg)
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
