package handlers

import (
	"net/http"
	"strings"

	"github.com/rpuneet/bc/pkg/workspace"
)

// RolesHandler handles /api/roles routes.
type RolesHandler struct {
	ws *workspace.Workspace
}

// NewRolesHandler creates a RolesHandler.
func NewRolesHandler(ws *workspace.Workspace) *RolesHandler {
	return &RolesHandler{ws: ws}
}

// Register mounts roles routes on mux.
func (h *RolesHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/roles", h.list)
	mux.HandleFunc("/api/roles/", h.byName)
}

// list handles GET /api/roles — returns all resolved roles.
func (h *RolesHandler) list(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	roles, err := h.ws.RoleManager.LoadAllRoles()
	if err != nil {
		httpError(w, "list roles: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resolved := make(map[string]*workspace.ResolvedRole, len(roles))
	for name := range roles {
		if res, resolveErr := h.ws.RoleManager.ResolveRole(name); resolveErr == nil {
			resolved[name] = res
		}
	}
	writeJSON(w, http.StatusOK, resolved)
}

// byName handles GET /api/roles/{name} — returns a single resolved role.
func (h *RolesHandler) byName(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/roles/")
	if name == "" {
		httpError(w, "role name required", http.StatusBadRequest)
		return
	}

	resolved, err := h.ws.RoleManager.ResolveRole(name)
	if err != nil {
		httpError(w, "role not found: "+err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, resolved)
}
