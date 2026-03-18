package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/rpuneet/bc/pkg/cron"
)

// CronHandler handles /api/cron routes.
type CronHandler struct {
	store *cron.Store
}

// NewCronHandler creates a CronHandler.
func NewCronHandler(store *cron.Store) *CronHandler {
	return &CronHandler{store: store}
}

// Register mounts cron routes on mux.
func (h *CronHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/cron", h.list)
	mux.HandleFunc("/api/cron/", h.byName)
}

func (h *CronHandler) list(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		jobs, err := h.store.ListJobs(r.Context())
		if err != nil {
			httpError(w, "list jobs: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if jobs == nil {
			jobs = []*cron.Job{}
		}
		writeJSON(w, http.StatusOK, jobs)

	case http.MethodPost:
		var job cron.Job
		if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := h.store.AddJob(r.Context(), &job); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, job)

	default:
		methodNotAllowed(w)
	}
}

func (h *CronHandler) byName(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/cron/"), "/", 2)
	name := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	if name == "" {
		httpError(w, "job name required", http.StatusBadRequest)
		return
	}

	switch sub {
	case "":
		h.job(w, r, name)
	case "enable":
		h.setEnabled(w, r, name, true)
	case "disable":
		h.setEnabled(w, r, name, false)
	case "logs":
		h.logs(w, r, name)
	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

func (h *CronHandler) job(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		job, err := h.store.GetJob(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, job)

	case http.MethodDelete:
		if err := h.store.DeleteJob(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w)
	}
}

func (h *CronHandler) setEnabled(w http.ResponseWriter, r *http.Request, name string, enabled bool) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := h.store.SetEnabled(r.Context(), name, enabled); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

func (h *CronHandler) logs(w http.ResponseWriter, r *http.Request, name string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	last := 20
	if s := r.URL.Query().Get("last"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			last = n
		}
	}
	logs, err := h.store.GetLogs(r.Context(), name, last)
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, logs)
}
