package handlers

import (
	"net/http"
	"strconv"
	"strings"

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
	tail := 100
	if s := r.URL.Query().Get("tail"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			tail = n
		}
	}
	tail = clampInt(tail, 1, 10000)
	evts, err := h.store.ReadLast(tail)
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
	name := strings.TrimPrefix(r.URL.Path, "/api/logs/")
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

