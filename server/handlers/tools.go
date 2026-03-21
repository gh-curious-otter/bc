package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rpuneet/bc/pkg/tool"
)

// ToolHandler handles /api/tools routes.
type ToolHandler struct {
	store *tool.Store
}

// NewToolHandler creates a ToolHandler.
func NewToolHandler(store *tool.Store) *ToolHandler {
	return &ToolHandler{store: store}
}

// Register mounts tool routes on mux.
func (h *ToolHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/tools", h.list)
	mux.HandleFunc("/api/tools/", h.byName)
}

func (h *ToolHandler) list(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	tools, err := h.store.List(r.Context())
	if err != nil {
		httpError(w, "list tools: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if tools == nil {
		tools = []*tool.Tool{}
	}
	limit, offset := parsePagination(r, 50)
	if offset >= len(tools) {
		tools = []*tool.Tool{}
	} else {
		tools = tools[offset:]
		if len(tools) > limit {
			tools = tools[:limit]
		}
	}
	writeJSON(w, http.StatusOK, tools)
}

func (h *ToolHandler) byName(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/tools/"), "/", 2)
	name := parts[0]
	sub := ""
	if len(parts) > 1 {
		sub = parts[1]
	}
	if name == "" {
		httpError(w, "tool name required", http.StatusBadRequest)
		return
	}

	switch sub {
	case "":
		h.tool(w, r, name)
	case "enable":
		h.setEnabled(w, r, name, true)
	case "disable":
		h.setEnabled(w, r, name, false)
	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

func (h *ToolHandler) tool(w http.ResponseWriter, r *http.Request, name string) {
	switch r.Method {
	case http.MethodGet:
		t, err := h.store.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, t)

	case http.MethodPut:
		var t tool.Tool
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		t.Name = name
		if err := h.store.Update(r.Context(), &t); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		updated, err := h.store.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, updated)

	case http.MethodDelete:
		if err := h.store.Delete(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		methodNotAllowed(w)
	}
}

func (h *ToolHandler) setEnabled(w http.ResponseWriter, r *http.Request, name string, enabled bool) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	if err := h.store.SetEnabled(r.Context(), name, enabled); err != nil {
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}
