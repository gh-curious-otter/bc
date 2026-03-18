package handlers

import (
	"net/http"
	"strconv"

	"github.com/rpuneet/bc/pkg/events"
)

// EventHandler handles /api/logs (historical event log).
type EventHandler struct {
	store events.EventStore
}

// NewEventHandler creates an EventHandler.
func NewEventHandler(store events.EventStore) *EventHandler {
	return &EventHandler{store: store}
}

// Register mounts event log routes on mux.
func (h *EventHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/logs", h.list)
	mux.HandleFunc("/api/logs/", h.byAgent)
}

func (h *EventHandler) list(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	tail := 0
	if s := r.URL.Query().Get("tail"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			tail = n
		}
	}
	var (
		evts []events.Event
		err  error
	)
	if tail > 0 {
		evts, err = h.store.ReadLast(tail)
	} else {
		evts, err = h.store.Read()
	}
	if err != nil {
		httpError(w, "read events: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if evts == nil {
		evts = []events.Event{}
	}
	writeJSON(w, http.StatusOK, evts)
}

func (h *EventHandler) byAgent(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	name := trimPrefix(r.URL.Path, "/api/logs/")
	if name == "" {
		httpError(w, "agent name required", http.StatusBadRequest)
		return
	}
	evts, err := h.store.ReadByAgent(name)
	if err != nil {
		httpError(w, "read events: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if evts == nil {
		evts = []events.Event{}
	}
	writeJSON(w, http.StatusOK, evts)
}

func trimPrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}
