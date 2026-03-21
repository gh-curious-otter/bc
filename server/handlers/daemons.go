package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rpuneet/bc/pkg/daemon"
)

// DaemonHandler handles /api/daemons routes.
type DaemonHandler struct {
	mgr *daemon.Manager
}

// NewDaemonHandler creates a DaemonHandler.
func NewDaemonHandler(mgr *daemon.Manager) *DaemonHandler {
	return &DaemonHandler{mgr: mgr}
}

// Register mounts daemon routes on mux.
func (h *DaemonHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/daemons", h.list)
	mux.HandleFunc("/api/daemons/", h.byName)
}

func (h *DaemonHandler) list(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		daemons, err := h.mgr.List(r.Context())
		if err != nil {
			httpError(w, "list daemons: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if daemons == nil {
			daemons = []*daemon.Daemon{}
		}
		limit, offset := parsePagination(r, 50)
		if offset >= len(daemons) {
			daemons = []*daemon.Daemon{}
		} else {
			daemons = daemons[offset:]
			if len(daemons) > limit {
				daemons = daemons[:limit]
			}
		}
		writeJSON(w, http.StatusOK, daemons)

	case http.MethodPost:
		var req struct {
			Name    string   `json:"name"`
			Cmd     string   `json:"cmd"`
			Image   string   `json:"image"`
			Runtime string   `json:"runtime"`
			Restart string   `json:"restart"`
			Env     []string `json:"env"`
			Ports   []string `json:"ports"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		d, err := h.mgr.Run(r.Context(), daemon.RunOptions{
			Name:    req.Name,
			Cmd:     req.Cmd,
			Image:   req.Image,
			Env:     req.Env,
			Runtime: req.Runtime,
			Ports:   req.Ports,
			Restart: req.Restart,
		})
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, d)

	default:
		methodNotAllowed(w)
	}
}

func (h *DaemonHandler) byName(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/daemons/"), "/", 2)
	name := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	if name == "" {
		httpError(w, "daemon name required", http.StatusBadRequest)
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		d, err := h.mgr.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		if d == nil {
			httpError(w, "daemon not found", http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, d)

	case r.Method == http.MethodPost && action == "stop":
		if err := h.mgr.Stop(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})

	case r.Method == http.MethodPost && action == "restart":
		d, err := h.mgr.Restart(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, d)

	case r.Method == http.MethodDelete && action == "":
		if err := h.mgr.Remove(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}
