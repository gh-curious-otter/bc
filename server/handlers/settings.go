package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rpuneet/bc/pkg/workspace"
)

// SettingsHandler handles /api/settings routes.
type SettingsHandler struct {
	ws *workspace.Workspace
}

// NewSettingsHandler creates a SettingsHandler.
func NewSettingsHandler(ws *workspace.Workspace) *SettingsHandler {
	return &SettingsHandler{ws: ws}
}

// Register mounts settings routes on mux.
func (h *SettingsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/settings", h.handle)
}

func (h *SettingsHandler) handle(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPut:
		h.put(w, r)
	default:
		methodNotAllowed(w)
	}
}

func (h *SettingsHandler) get(w http.ResponseWriter, _ *http.Request) {
	if h.ws.Config == nil {
		httpError(w, "no config loaded", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, h.ws.Config)
}

func (h *SettingsHandler) put(w http.ResponseWriter, r *http.Request) {
	if h.ws.Config == nil {
		httpError(w, "no config loaded", http.StatusInternalServerError)
		return
	}

	// Decode partial update into a map to detect which fields are present.
	var patch map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		httpError(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Start from current config (copy by value to avoid partial mutation on error).
	merged := *h.ws.Config

	// Apply each section if present in the patch.
	if raw, ok := patch["user"]; ok {
		if err := json.Unmarshal(raw, &merged.User); err != nil {
			httpError(w, "invalid user config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["providers"]; ok {
		if err := json.Unmarshal(raw, &merged.Providers); err != nil {
			httpError(w, "invalid providers config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["env"]; ok {
		if err := json.Unmarshal(raw, &merged.Env); err != nil {
			httpError(w, "invalid env config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["logs"]; ok {
		if err := json.Unmarshal(raw, &merged.Logs); err != nil {
			httpError(w, "invalid logs config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["runtime"]; ok {
		if err := json.Unmarshal(raw, &merged.Runtime); err != nil {
			httpError(w, "invalid runtime config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["performance"]; ok {
		if err := json.Unmarshal(raw, &merged.Performance); err != nil {
			httpError(w, "invalid performance config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["tui"]; ok {
		if err := json.Unmarshal(raw, &merged.TUI); err != nil {
			httpError(w, "invalid tui config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["workspace"]; ok {
		if err := json.Unmarshal(raw, &merged.Workspace); err != nil {
			httpError(w, "invalid workspace config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["roster"]; ok {
		if err := json.Unmarshal(raw, &merged.Roster); err != nil {
			httpError(w, "invalid roster config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if raw, ok := patch["services"]; ok {
		if err := json.Unmarshal(raw, &merged.Services); err != nil {
			httpError(w, "invalid services config: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	// Validate the merged config.
	if err := merged.Validate(); err != nil {
		httpError(w, "validation failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update in-memory config and persist to disk.
	*h.ws.Config = merged
	if err := h.ws.Save(); err != nil {
		httpError(w, "failed to save config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, h.ws.Config)
}
