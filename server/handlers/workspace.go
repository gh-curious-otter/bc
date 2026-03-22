package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

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
		httpInternalError(w, "list agents", err)
		return
	}
	runningCount := 0
	for _, a := range agents {
		if a.State != agent.StateStopped && a.State != agent.StateError {
			runningCount++
		}
	}
	nickname := ""
	if h.ws.Config != nil {
		nickname = h.ws.Config.User.Nickname
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":          h.ws.Name(),
		"nickname":      nickname,
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
		httpInternalError(w, "list roles", err)
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
	if r.ContentLength > 0 {
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}
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
		httpInternalError(w, "operation failed", err)
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
	return len(msg) > 0 && (strings.Contains(msg, "already running") || strings.Contains(msg, "session is alive"))
}
