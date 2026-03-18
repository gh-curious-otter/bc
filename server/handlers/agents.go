package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
)

// AgentHandler handles /api/agents routes.
type AgentHandler struct {
	svc *agent.AgentService
}

// NewAgentHandler creates an AgentHandler.
func NewAgentHandler(svc *agent.AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

// Register mounts agent routes on mux.
func (h *AgentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/agents", h.list)
	mux.HandleFunc("/api/agents/", h.byName)
}

type agentDTO struct {
	CreatedAt time.Time  `json:"created_at"`
	StartedAt time.Time  `json:"started_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	State     string     `json:"state"`
	Task      string     `json:"task,omitempty"`
	Team      string     `json:"team,omitempty"`
	Tool      string     `json:"tool,omitempty"`
	Session   string     `json:"session,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	ParentID  string     `json:"parent_id,omitempty"`
	Children  []string   `json:"children,omitempty"`
}

func toDTO(a *agent.Agent) agentDTO {
	return agentDTO{
		ID:        a.ID,
		Name:      a.Name,
		Role:      string(a.Role),
		State:     string(a.State),
		Task:      a.Task,
		Team:      a.Team,
		Tool:      a.Tool,
		Session:   a.Session,
		SessionID: a.SessionID,
		ParentID:  a.ParentID,
		Children:  a.Children,
		CreatedAt: a.CreatedAt,
		StartedAt: a.StartedAt,
		UpdatedAt: a.UpdatedAt,
		StoppedAt: a.StoppedAt,
	}
}

func (h *AgentHandler) list(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		agents, err := h.svc.List(r.Context(), agent.ListOptions{})
		if err != nil {
			httpError(w, "list agents: "+err.Error(), http.StatusInternalServerError)
			return
		}
		dtos := make([]agentDTO, 0, len(agents))
		for _, a := range agents {
			dtos = append(dtos, toDTO(a))
		}
		writeJSON(w, http.StatusOK, dtos)

	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Role    string `json:"role"`
			Tool    string `json:"tool"`
			Runtime string `json:"runtime"`
			Parent  string `json:"parent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		a, err := h.svc.Create(r.Context(), agent.CreateOptions{
			Name:    req.Name,
			Role:    agent.Role(req.Role),
			Tool:    req.Tool,
			Runtime: req.Runtime,
			Parent:  req.Parent,
		})
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, toDTO(a))

	default:
		methodNotAllowed(w)
	}
}

func (h *AgentHandler) byName(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/", 2)
	name := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}
	if name == "" {
		httpError(w, "agent name required", http.StatusBadRequest)
		return
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		a, err := h.svc.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, toDTO(a))

	case r.Method == http.MethodPost && action == "start":
		a, err := h.svc.Start(r.Context(), name, agent.StartOptions{})
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, toDTO(a))

	case r.Method == http.MethodPost && action == "stop":
		if err := h.svc.Stop(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})

	case r.Method == http.MethodPost && action == "send":
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		if err := h.svc.Send(r.Context(), name, req.Message); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})

	case r.Method == http.MethodDelete && action == "":
		if err := h.svc.Delete(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case r.Method == http.MethodPost && action == "hook":
		// Receive a Claude Code hook event and update agent state.
		var req struct {
			Event string `json:"event"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, "invalid request body", http.StatusBadRequest)
			return
		}
		ev := agent.HookEvent(req.Event)
		targetState, ok := agent.StateForHookEvent(ev)
		if !ok {
			httpError(w, "unknown event: "+req.Event, http.StatusBadRequest)
			return
		}
		if err := h.svc.Manager().UpdateAgentState(name, targetState, ""); err != nil {
			// Transition may be invalid (agent stopped, etc.) — treat as no-op.
			writeJSON(w, http.StatusOK, map[string]any{"ok": true, "skipped": true})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})

	case r.Method == http.MethodGet && action == "stats":
		// Return recent Docker stats samples for this agent.
		limit := 20
		if lStr := r.URL.Query().Get("limit"); lStr != "" {
			if n, err := strconv.Atoi(lStr); err == nil && n > 0 {
				limit = n
			}
		}
		records, err := h.svc.Manager().QueryAgentStats(name, limit)
		if err != nil {
			httpError(w, "stats unavailable: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if records == nil {
			records = []*agent.AgentStatsRecord{}
		}
		writeJSON(w, http.StatusOK, records)

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}
