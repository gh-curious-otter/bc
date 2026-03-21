// Package server implements the bcd HTTP REST API.
//
// The server exposes agent, channel, workspace, and daemon state over HTTP
// so the bc CLI can operate as a thin client communicating with bcd.
//
// # Endpoints
//
//	GET  /health                           — liveness probe
//	GET  /api/agents                       — list agents
//	POST /api/agents                       — create agent
//	GET  /api/agents/generate-name         — generate a unique agent name
//	POST /api/agents/broadcast             — broadcast message to all agents
//	POST /api/agents/send-role             — send message to agents by role
//	POST /api/agents/send-pattern          — send message to agents by pattern
//	GET  /api/agents/:name                 — get agent
//	DELETE /api/agents/:name               — delete agent
//	POST /api/agents/:name/start           — start agent
//	POST /api/agents/:name/stop            — stop agent
//	POST /api/agents/:name/send            — send message to agent
//	POST /api/agents/:name/rename          — rename agent
//	GET  /api/agents/:name/peek            — peek at agent output
//	GET  /api/agents/:name/sessions        — list agent sessions
//	GET  /api/channels                     — list channels
//	POST /api/channels                     — create channel
//	GET  /api/channels/status              — channel status summary
//	GET  /api/channels/:name               — get channel
//	PUT  /api/channels/:name               — update channel
//	DELETE /api/channels/:name             — delete channel
//	POST /api/channels/:name/members       — add member
//	DELETE /api/channels/:name/members/:agent — remove member
//	POST /api/channels/:name/send          — send message to channel
//	GET  /api/channels/:name/history       — get channel history
//	POST /api/channels/:name/react         — react to message
//	GET  /api/workspace/status             — workspace info
//	POST /api/workspace/up                 — start workspace (root agent)
//	POST /api/workspace/down               — stop all agents
//	GET  /api/daemons                      — list workspace daemons
//	GET  /api/events                       — list events (supports ?agent=, ?tail=)
//
// All responses are JSON. Errors are returned as {"error": "..."}.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Server is the bcd HTTP server.
type Server struct {
	agents     *agent.AgentService
	channels   *channel.ChannelService
	daemons    *daemon.Manager
	eventStore events.EventStore
	ws         *workspace.Workspace
	httpServer *http.Server
	addr       string
}

// Config holds server configuration.
type Config struct {
	Addr string // e.g. ":4880" or "localhost:4880"
}

// DefaultConfig returns the default server configuration.
// The default address is localhost-only to prevent unintended network exposure.
func DefaultConfig() Config {
	return Config{Addr: "127.0.0.1:4880"}
}

// New creates a new bcd server.
func New(
	cfg Config,
	agentSvc *agent.AgentService,
	channelSvc *channel.ChannelService,
	daemonMgr *daemon.Manager,
	ws *workspace.Workspace,
) *Server {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:4880"
	}

	s := &Server{
		agents:   agentSvc,
		channels: channelSvc,
		daemons:  daemonMgr,
		ws:       ws,
		addr:     cfg.Addr,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)

	// Exact-path agent routes must be registered BEFORE the prefix route "/api/agents/"
	mux.HandleFunc("/api/agents/generate-name", s.handleAgentGenerateName)
	mux.HandleFunc("/api/agents/broadcast", s.handleAgentBroadcast)
	mux.HandleFunc("/api/agents/send-role", s.handleAgentSendRole)
	mux.HandleFunc("/api/agents/send-pattern", s.handleAgentSendPattern)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentByName)

	// Channel routes
	mux.HandleFunc("/api/channels/status", s.handleChannelStatus)
	mux.HandleFunc("/api/channels", s.handleChannels)
	mux.HandleFunc("/api/channels/", s.handleChannelByName)

	// Workspace routes
	mux.HandleFunc("/api/workspace/status", s.handleWorkspaceStatus)
	mux.HandleFunc("/api/workspace/up", s.handleWorkspaceUp)
	mux.HandleFunc("/api/workspace/down", s.handleWorkspaceDown)

	// Other routes
	mux.HandleFunc("/api/daemons", s.handleDaemons)
	mux.HandleFunc("/api/events", s.handleEvents)

	s.httpServer = &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// WithEventStore configures the server with an event store.
func (s *Server) WithEventStore(store events.EventStore) {
	s.eventStore = store
}

// Addr returns the resolved listen address after Start is called.
func (s *Server) Addr() string {
	return s.addr
}

// Start begins listening on the configured address.
// It blocks until the server is shut down.
func (s *Server) Start(ctx context.Context) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}
	s.addr = ln.Addr().String() // capture actual port if :0 was used

	log.Info("bcd listening", "addr", s.addr)

	// Shut down when context is canceled
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Warn("server shutdown error", "error", err)
		}
	}()

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// --- Agent DTO ---

type agentDTO struct {
	CreatedAt time.Time  `json:"created_at"`
	StartedAt time.Time  `json:"started_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
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

func toAgentDTO(a *agent.Agent) agentDTO {
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

// --- Handlers ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": "1",
		"addr":    s.addr,
	})
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		agents, err := s.agents.List(r.Context(), agent.ListOptions{})
		if err != nil {
			httpError(w, fmt.Sprintf("list agents: %v", err), http.StatusInternalServerError)
			return
		}
		dtos := make([]agentDTO, 0, len(agents))
		for _, a := range agents {
			dtos = append(dtos, toAgentDTO(a))
		}
		writeJSON(w, http.StatusOK, dtos)

	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Role    string `json:"role"`
			Tool    string `json:"tool,omitempty"`
			Runtime string `json:"runtime,omitempty"`
			Parent  string `json:"parent,omitempty"`
			Team    string `json:"team,omitempty"`
			EnvFile string `json:"env_file,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		a, err := s.agents.Create(r.Context(), agent.CreateOptions{
			Name:    req.Name,
			Role:    agent.Role(req.Role),
			Tool:    req.Tool,
			Runtime: req.Runtime,
			Parent:  req.Parent,
			Team:    req.Team,
			EnvFile: req.EnvFile,
		})
		if err != nil {
			httpError(w, fmt.Sprintf("create agent: %v", err), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, toAgentDTO(a))

	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAgentGenerateName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name, err := s.agents.GenerateName(r.Context())
	if err != nil {
		httpError(w, fmt.Sprintf("generate name: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": name})
}

func (s *Server) handleAgentBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}
	sent, err := s.agents.Broadcast(r.Context(), req.Message)
	if err != nil {
		httpError(w, fmt.Sprintf("broadcast: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"sent": sent})
}

func (s *Server) handleAgentSendRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Role    string `json:"role"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}
	result, err := s.agents.SendToRole(r.Context(), req.Role, req.Message)
	if err != nil {
		httpError(w, fmt.Sprintf("send-role: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAgentSendPattern(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Pattern string `json:"pattern"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
		return
	}
	result, err := s.agents.SendToPattern(r.Context(), req.Pattern, req.Message)
	if err != nil {
		httpError(w, fmt.Sprintf("send-pattern: %v", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// handleAgentByName handles /api/agents/:name and /api/agents/:name/action
func (s *Server) handleAgentByName(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		httpError(w, "agent name required", http.StatusBadRequest)
		return
	}
	name := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		a, err := s.agents.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, toAgentDTO(a))

	case r.Method == http.MethodDelete && action == "":
		force := r.URL.Query().Get("force") == "true"
		if err := s.agents.Delete(r.Context(), name, force); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case r.Method == http.MethodPost && action == "start":
		var req struct {
			Runtime  string `json:"runtime,omitempty"`
			ResumeID string `json:"resume_id,omitempty"`
			Fresh    bool   `json:"fresh,omitempty"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck // optional body
		a, err := s.agents.Start(r.Context(), name, agent.StartOptions{
			Runtime:  req.Runtime,
			ResumeID: req.ResumeID,
			Fresh:    req.Fresh,
		})
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, toAgentDTO(a))

	case r.Method == http.MethodPost && action == "stop":
		if err := s.agents.Stop(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})

	case r.Method == http.MethodPost && action == "send":
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		if err := s.agents.Send(r.Context(), name, req.Message); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "sent"})

	case r.Method == http.MethodPost && action == "rename":
		var req struct {
			NewName string `json:"new_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		if err := s.agents.Rename(r.Context(), name, req.NewName); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "renamed"})

	case r.Method == http.MethodGet && action == "peek":
		lines := 50
		if q := r.URL.Query().Get("lines"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n > 0 {
				lines = n
			}
		}
		output, err := s.agents.Peek(r.Context(), name, lines)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"output": output})

	case r.Method == http.MethodGet && action == "sessions":
		entries, err := s.agents.Sessions(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		if entries == nil {
			entries = []agent.SessionEntry{}
		}
		writeJSON(w, http.StatusOK, entries)

	case r.Method == http.MethodGet && action == "cost":
		summary, err := s.agents.Cost(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		if summary == nil {
			writeJSON(w, http.StatusOK, &agent.CostSummary{AgentID: name})
			return
		}
		writeJSON(w, http.StatusOK, summary)

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		channels, err := s.channels.List(r.Context())
		if err != nil {
			httpError(w, fmt.Sprintf("list channels: %v", err), http.StatusInternalServerError)
			return
		}
		writeJSON(w, http.StatusOK, channels)

	case http.MethodPost:
		var req channel.CreateChannelReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		ch, err := s.channels.Create(r.Context(), req)
		if err != nil {
			httpError(w, fmt.Sprintf("create channel: %v", err), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusCreated, ch)

	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleChannelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channels, err := s.channels.List(r.Context())
	if err != nil {
		httpError(w, fmt.Sprintf("list channels: %v", err), http.StatusInternalServerError)
		return
	}
	totalMembers := 0
	for _, ch := range channels {
		totalMembers += ch.MemberCount
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"channel_count": len(channels),
		"total_members": totalMembers,
	})
}

// handleChannelByName handles /api/channels/:name and sub-actions
func (s *Server) handleChannelByName(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/channels/")
	parts := strings.SplitN(path, "/", 3)
	if len(parts) == 0 || parts[0] == "" {
		httpError(w, "channel name required", http.StatusBadRequest)
		return
	}
	name := parts[0]
	action := ""
	if len(parts) > 1 {
		action = parts[1]
	}

	switch {
	case r.Method == http.MethodGet && action == "":
		ch, err := s.channels.Get(r.Context(), name)
		if err != nil {
			httpError(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, ch)

	case r.Method == http.MethodPut && action == "":
		var req channel.UpdateChannelReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		ch, err := s.channels.Update(r.Context(), name, req)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, ch)

	case r.Method == http.MethodDelete && action == "":
		if err := s.channels.Delete(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case r.Method == http.MethodPost && action == "members":
		var req struct {
			Agent string `json:"agent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		if err := s.channels.AddMember(r.Context(), name, req.Agent); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "added"})

	case r.Method == http.MethodDelete && action == "members":
		agentID := ""
		if len(parts) > 2 {
			agentID = parts[2]
		}
		if agentID == "" {
			httpError(w, "agent name required", http.StatusBadRequest)
			return
		}
		if err := s.channels.RemoveMember(r.Context(), name, agentID); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	case r.Method == http.MethodPost && action == "send":
		var req struct {
			Sender  string `json:"sender"`
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		msg, err := s.channels.Send(r.Context(), name, req.Sender, req.Message)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Also deliver to agent tmux sessions
		ch, getErr := s.channels.Get(r.Context(), name)
		if getErr == nil {
			for _, member := range ch.Members {
				if member == req.Sender {
					continue
				}
				if sendErr := s.agents.Send(r.Context(), member, req.Message); sendErr != nil {
					log.Debug("channel send: failed to deliver to agent", "agent", member, "error", sendErr)
				}
			}
		}
		writeJSON(w, http.StatusOK, msg)

	case r.Method == http.MethodGet && action == "history":
		q := r.URL.Query()
		opts := channel.HistoryOpts{}
		if n := q.Get("limit"); n != "" {
			if v, err := strconv.Atoi(n); err == nil {
				opts.Limit = v
			}
		}
		if n := q.Get("offset"); n != "" {
			if v, err := strconv.Atoi(n); err == nil {
				opts.Offset = v
			}
		}
		if a := q.Get("agent"); a != "" {
			opts.Agent = a
		}
		msgs, err := s.channels.History(r.Context(), name, opts)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, msgs)

	case r.Method == http.MethodPost && action == "react":
		var req struct {
			Emoji string `json:"emoji"`
			User  string `json:"user"`
			MsgID int    `json:"msg_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, fmt.Sprintf("invalid request: %v", err), http.StatusBadRequest)
			return
		}
		added, err := s.channels.React(r.Context(), name, req.MsgID, req.Emoji, req.User)
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"added": added})

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

func (s *Server) handleWorkspaceStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	agents, err := s.agents.List(r.Context(), agent.ListOptions{})
	if err != nil {
		httpError(w, fmt.Sprintf("list agents: %v", err), http.StatusInternalServerError)
		return
	}
	runningCount := 0
	for _, a := range agents {
		if a.State != agent.StateStopped && a.State != agent.StateError {
			runningCount++
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":          s.ws.Name(),
		"root_dir":      s.ws.RootDir,
		"agent_count":   len(agents),
		"running_count": runningCount,
		"is_healthy":    true,
	})
}

func (s *Server) handleWorkspaceUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Tool    string `json:"tool,omitempty"`
		Runtime string `json:"runtime,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck // optional body

	// Create root agent if not already running
	existing := s.agents.Manager().GetAgent("root")
	if existing != nil && existing.State != agent.StateStopped && existing.State != agent.StateError {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "already_running",
			"message": "root agent is already running",
		})
		return
	}

	var a *agent.Agent
	var err error
	if existing != nil {
		a, err = s.agents.Start(r.Context(), "root", agent.StartOptions{
			Runtime: req.Runtime,
		})
	} else {
		a, err = s.agents.Create(r.Context(), agent.CreateOptions{
			Name:    "root",
			Role:    agent.RoleRoot,
			Tool:    req.Tool,
			Runtime: req.Runtime,
		})
	}
	if err != nil {
		httpError(w, fmt.Sprintf("start workspace: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status": "started",
		"agent":  toAgentDTO(a),
	})
}

func (s *Server) handleWorkspaceDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	stopped, err := s.agents.StopAll(r.Context())
	if err != nil {
		httpError(w, fmt.Sprintf("stop all: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"stopped": stopped})
}

func (s *Server) handleDaemons(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	daemons, err := s.daemons.List(r.Context())
	if err != nil {
		httpError(w, fmt.Sprintf("list daemons: %v", err), http.StatusInternalServerError)
		return
	}
	if daemons == nil {
		daemons = []*daemon.Daemon{}
	}
	writeJSON(w, http.StatusOK, daemons)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.eventStore == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}

	q := r.URL.Query()
	agentFilter := q.Get("agent")
	tailStr := q.Get("tail")

	var evts []events.Event
	var err error

	switch {
	case agentFilter != "":
		evts, err = s.eventStore.ReadByAgent(agentFilter)
	case tailStr != "":
		n, parseErr := strconv.Atoi(tailStr)
		if parseErr != nil || n <= 0 {
			httpError(w, "invalid tail value", http.StatusBadRequest)
			return
		}
		evts, err = s.eventStore.ReadLast(n)
	default:
		evts, err = s.eventStore.Read()
	}

	if err != nil {
		httpError(w, fmt.Sprintf("read events: %v", err), http.StatusInternalServerError)
		return
	}
	if evts == nil {
		evts = []events.Event{}
	}
	writeJSON(w, http.StatusOK, evts)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Debug("failed to write JSON response", "error", err)
	}
}

func httpError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck // best-effort
}
