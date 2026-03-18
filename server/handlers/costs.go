package handlers

import (
	"net/http"
	"strings"

	"github.com/rpuneet/bc/pkg/cost"
)

// CostHandler handles /api/costs routes.
type CostHandler struct {
	store *cost.Store
}

// NewCostHandler creates a CostHandler.
func NewCostHandler(store *cost.Store) *CostHandler {
	return &CostHandler{store: store}
}

// Register mounts cost routes on mux.
func (h *CostHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/costs", h.summary)
	mux.HandleFunc("/api/costs/", h.byResource)
}

func (h *CostHandler) summary(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	s, err := h.store.WorkspaceSummary()
	if err != nil {
		httpError(w, "workspace summary: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

func (h *CostHandler) byResource(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/costs/"), "/", 2)
	resource := parts[0]

	switch resource {
	case "agents":
		summaries, err := h.store.SummaryByAgent()
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaries)

	case "teams":
		summaries, err := h.store.SummaryByTeam()
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaries)

	case "models":
		summaries, err := h.store.SummaryByModel()
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaries)

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}
