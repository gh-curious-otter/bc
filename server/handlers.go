package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
)

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, map[string]any{
		"status":    "ok",
		"uptime":    time.Since(s.startedAt).String(),
		"workspace": s.ws.Name(),
	})
}

// --- Agents ---

func (s *Server) handleAgentList(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, s.agents.ListAgents())
}

func (s *Server) handleAgentGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ag := s.agents.GetAgent(name)
	if ag == nil {
		Error(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}
	JSON(w, http.StatusOK, ag)
}

func (s *Server) handleAgentCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Role string `json:"role"`
		Tool string `json:"tool"`
	}
	if err := decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Role == "" {
		Error(w, http.StatusBadRequest, "role is required")
		return
	}

	role := agent.Role(req.Role)
	workspace := s.ws.RootDir
	tool := req.Tool
	if tool == "" {
		tool = s.ws.DefaultProvider()
	}

	var (
		ag  *agent.Agent
		err error
	)
	if tool != "" {
		ag, err = s.agents.SpawnAgentWithTool(req.Name, role, workspace, tool)
	} else {
		ag, err = s.agents.SpawnAgent(req.Name, role, workspace)
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create agent: "+err.Error())
		return
	}
	JSON(w, http.StatusCreated, ag)
}

func (s *Server) handleAgentDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ag := s.agents.GetAgent(name)
	if ag == nil {
		Error(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}
	if err := s.agents.StopAgent(name); err != nil {
		Error(w, http.StatusInternalServerError, "failed to stop agent: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"status": "stopped", "name": name})
}

func (s *Server) handleAgentStart(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ag := s.agents.GetAgent(name)
	if ag == nil {
		Error(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}

	workspace := s.ws.RootDir
	tool := ag.Tool
	if tool == "" {
		tool = s.ws.DefaultProvider()
	}

	var (
		spawned *agent.Agent
		err     error
	)
	if tool != "" {
		spawned, err = s.agents.SpawnAgentWithTool(name, ag.Role, workspace, tool)
	} else {
		spawned, err = s.agents.SpawnAgent(name, ag.Role, workspace)
	}
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to start agent: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, spawned)
}

func (s *Server) handleAgentStop(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.agents.StopAgent(name); err != nil {
		Error(w, http.StatusInternalServerError, "failed to stop agent: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"status": "stopped", "name": name})
}

func (s *Server) handleAgentPeek(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ag := s.agents.GetAgent(name)
	if ag == nil {
		Error(w, http.StatusNotFound, fmt.Sprintf("agent %q not found", name))
		return
	}

	lines := 50
	if v := r.URL.Query().Get("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			lines = n
		}
	}

	rt := s.agents.Runtime()
	sessionName := rt.SessionName(name)
	output, err := rt.Capture(r.Context(), sessionName, lines)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to capture output: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"name": name, "lines": lines, "output": output})
}

func (s *Server) handleAgentSend(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Message string `json:"message"`
	}
	if err := decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Message == "" {
		Error(w, http.StatusBadRequest, "message is required")
		return
	}
	if err := s.agents.SendToAgent(name, req.Message); err != nil {
		Error(w, http.StatusInternalServerError, "failed to send message: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"status": "sent", "name": name})
}

// --- Channels ---

func (s *Server) handleChannelList(w http.ResponseWriter, _ *http.Request) {
	channels, err := s.channels.ListChannels()
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to list channels: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, channels)
}

func (s *Server) handleChannelCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Name == "" {
		Error(w, http.StatusBadRequest, "name is required")
		return
	}
	ch, err := s.channels.CreateChannel(req.Name, channel.ChannelTypeGroup, req.Description)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to create channel: "+err.Error())
		return
	}
	JSON(w, http.StatusCreated, ch)
}

func (s *Server) handleChannelGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ch, err := s.channels.GetChannel(name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get channel: "+err.Error())
		return
	}
	if ch == nil {
		Error(w, http.StatusNotFound, fmt.Sprintf("channel %q not found", name))
		return
	}
	members, err := s.channels.GetMembers(name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get members: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"channel": ch, "members": members})
}

func (s *Server) handleChannelDelete(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.channels.DeleteChannel(name); err != nil {
		Error(w, http.StatusInternalServerError, "failed to delete channel: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{"status": "deleted", "name": name})
}

func (s *Server) handleChannelSend(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Sender  string `json:"sender"`
		Message string `json:"message"`
	}
	if err := decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}
	if req.Sender == "" {
		Error(w, http.StatusBadRequest, "sender is required")
		return
	}
	if req.Message == "" {
		Error(w, http.StatusBadRequest, "message is required")
		return
	}
	msg, err := s.channels.AddMessage(name, req.Sender, req.Message, channel.TypeText, "")
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to send message: "+err.Error())
		return
	}
	JSON(w, http.StatusCreated, msg)
}

func (s *Server) handleChannelHistory(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	messages, err := s.channels.GetHistory(name, limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get history: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, messages)
}

// --- Costs ---

func (s *Server) handleCostSummary(w http.ResponseWriter, _ *http.Request) {
	ws, err := s.costs.WorkspaceSummary()
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get workspace summary: "+err.Error())
		return
	}
	byAgent, err := s.costs.SummaryByAgent()
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get agent summary: "+err.Error())
		return
	}
	byModel, err := s.costs.SummaryByModel()
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get model summary: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"workspace": ws,
		"by_agent":  byAgent,
		"by_model":  byModel,
	})
}

func (s *Server) handleCostByAgent(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	summary, err := s.costs.AgentSummary(name)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get agent cost summary: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, summary)
}

func (s *Server) handleCostBudget(w http.ResponseWriter, _ *http.Request) {
	budgets, err := s.costs.GetAllBudgets()
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to get budgets: "+err.Error())
		return
	}
	status, err := s.costs.CheckBudget("workspace")
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to check budget: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, map[string]any{
		"budgets":          budgets,
		"workspace_status": status,
	})
}

// --- Workspace ---

func (s *Server) handleWorkspaceStatus(w http.ResponseWriter, _ *http.Request) {
	agentCount := s.agents.AgentCount()
	channels, err := s.channels.ListChannels()
	channelCount := 0
	if err == nil {
		channelCount = len(channels)
	}
	JSON(w, http.StatusOK, map[string]any{
		"name":          s.ws.Name(),
		"root_dir":      s.ws.RootDir,
		"state_dir":     s.ws.StateDir(),
		"agent_count":   agentCount,
		"channel_count": channelCount,
		"uptime":        time.Since(s.startedAt).String(),
	})
}

func (s *Server) handleWorkspaceConfig(w http.ResponseWriter, _ *http.Request) {
	JSON(w, http.StatusOK, s.ws.Config)
}

// --- Events ---

func (s *Server) handleEventList(w http.ResponseWriter, r *http.Request) {
	agentFilter := r.URL.Query().Get("agent")
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	if agentFilter != "" {
		evts, err := s.events.ReadByAgent(agentFilter)
		if err != nil {
			Error(w, http.StatusInternalServerError, "failed to read events: "+err.Error())
			return
		}
		JSON(w, http.StatusOK, evts)
		return
	}

	evts, err := s.events.ReadLast(limit)
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to read events: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, evts)
}

func (s *Server) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		Error(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	lastSeen := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			evts, err := s.events.Read()
			if err != nil {
				continue
			}
			if len(evts) > lastSeen {
				for _, ev := range evts[lastSeen:] {
					data, err := json.Marshal(ev)
					if err != nil {
						continue
					}
					_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
				}
				lastSeen = len(evts)
				flusher.Flush()
			}
		}
	}
}

// --- Roles ---

func (s *Server) handleRoleList(w http.ResponseWriter, _ *http.Request) {
	if s.ws.RoleManager == nil {
		Error(w, http.StatusInternalServerError, "role manager not initialized")
		return
	}
	roles, err := s.ws.RoleManager.LoadAllRoles()
	if err != nil {
		Error(w, http.StatusInternalServerError, "failed to load roles: "+err.Error())
		return
	}
	JSON(w, http.StatusOK, roles)
}

func (s *Server) handleRoleGet(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if s.ws.RoleManager == nil {
		Error(w, http.StatusInternalServerError, "role manager not initialized")
		return
	}
	role, err := s.ws.RoleManager.LoadRole(name)
	if err != nil {
		Error(w, http.StatusNotFound, fmt.Sprintf("role %q not found: %s", name, err.Error()))
		return
	}
	JSON(w, http.StatusOK, role)
}
