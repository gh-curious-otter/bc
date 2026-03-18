package handlers

import (
	"encoding/json"
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
	mux.HandleFunc("/api/workspace", h.status) // root = status
	mux.HandleFunc("/api/workspace/status", h.status)
	mux.HandleFunc("/api/workspace/roles", h.roles)
	mux.HandleFunc("/api/workspace/up", h.up)
	mux.HandleFunc("/api/workspace/down", h.down)
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

	// Resolve each role via BFS inheritance so response includes
	// inherited MCP servers, secrets, commands, rules, etc.
	resolved := make(map[string]*workspace.ResolvedRole, len(roles))
	for name := range roles {
		if res, resolveErr := h.ws.RoleManager.ResolveRole(name); resolveErr == nil {
			resolved[name] = res
		}
	}
	writeJSON(w, http.StatusOK, resolved)
}

func (h *WorkspaceHandler) up(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req struct {
		Tool    string `json:"tool"`
		Runtime string `json:"runtime"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck // optional body
	a, err := h.svc.Create(r.Context(), agent.CreateOptions{
		Name:    "root",
		Role:    agent.RoleRoot,
		Tool:    req.Tool,
		Runtime: req.Runtime,
	})
	if err != nil {
		if isAlreadyRunning(err) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "already_running"})
			return
		}
		httpError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "started", "session": a.Session})
}

func (h *WorkspaceHandler) down(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	stopped, err := h.svc.StopAll(r.Context())
	if err != nil {
		httpError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"stopped": stopped})
}

// isAlreadyRunning detects "already running" errors from Create/Start.
func isAlreadyRunning(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return len(msg) > 0 && (contains(msg, "already running") || contains(msg, "session is alive"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && indexBytes(s, sub) >= 0)
}

func indexBytes(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
