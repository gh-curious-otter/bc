package handlers

import (
	"net/http"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/workspace"
)

// WorkspaceHandler handles /api/workspace routes.
type WorkspaceHandler struct {
	svc *agent.AgentService
	ws  *workspace.Workspace
}

// NewWorkspaceHandler creates a WorkspaceHandler.
func NewWorkspaceHandler(svc *agent.AgentService, ws *workspace.Workspace) *WorkspaceHandler {
	return &WorkspaceHandler{svc: svc, ws: ws}
}

// Register mounts workspace routes on mux.
func (h *WorkspaceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/workspace/status", h.status)
	mux.HandleFunc("/api/workspace/roles", h.roles)
}

func (h *WorkspaceHandler) status(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	agents, err := h.svc.List(r.Context(), agent.ListOptions{})
	if err != nil {
		httpError(w, "list agents: "+err.Error(), http.StatusInternalServerError)
		return
	}
	runningCount := 0
	for _, a := range agents {
		if a.State != agent.StateStopped && a.State != agent.StateError {
			runningCount++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":          h.ws.Name(),
		"root_dir":      h.ws.RootDir,
		"agent_count":   len(agents),
		"running_count": runningCount,
		"is_healthy":    true,
	})
}

func (h *WorkspaceHandler) roles(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	roles, err := h.ws.RoleManager.LoadAllRoles()
	if err != nil {
		httpError(w, "list roles: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, roles)
}
