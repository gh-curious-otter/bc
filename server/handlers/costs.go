package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/cost"
)

// CostHandler handles /api/costs routes.
type CostHandler struct {
	store    *cost.Store
	importer *cost.Importer
}

// NewCostHandler creates a CostHandler.
func NewCostHandler(store *cost.Store, importer *cost.Importer) *CostHandler {
	return &CostHandler{store: store, importer: importer}
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
	s, err := h.store.WorkspaceSummary(r.Context())
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
		summaries, err := h.store.SummaryByAgent(r.Context())
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaries)

	case "teams":
		summaries, err := h.store.SummaryByTeam(r.Context())
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaries)

	case "models":
		summaries, err := h.store.SummaryByModel(r.Context())
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, summaries)

	case "daily":
		h.daily(w, r)

	case "sync":
		h.sync(w, r)

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

// daily handles GET /api/costs/daily?days=30
func (h *CostHandler) daily(w http.ResponseWriter, r *http.Request) {
	days := 30
	if s := r.URL.Query().Get("days"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			days = n
		}
	}
	days = clampInt(days, 1, 365)
	since := time.Now().AddDate(0, 0, -days)
	costs, err := h.store.GetDailyCosts(r.Context(), since)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, costs)
}

// sync handles POST /api/costs/sync — triggers a fresh import from JSONL files.
func (h *CostHandler) sync(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if h.importer == nil {
		httpError(w, "importer not configured", http.StatusServiceUnavailable)
		return
	}
	n, err := h.importer.ImportAll(r.Context())
	if err != nil {
		httpError(w, "import failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": n})
}
