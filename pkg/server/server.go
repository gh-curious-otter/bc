// Package server implements the bcd HTTP REST API.
//
// The server exposes agent, channel, workspace, and daemon state over HTTP
// so the bc CLI can operate as a thin client communicating with bcd.
//
// # Endpoints
//
//	GET  /health                    — liveness probe
//	GET  /api/agents                — list agents
//	GET  /api/agents/:name          — get agent
//	POST /api/agents/:name/stop     — stop agent
//	GET  /api/channels              — list channels
//	GET  /api/workspace/status      — workspace info
//	GET  /api/daemons               — list workspace daemons
//
// All responses are JSON. Errors are returned as {"error": "..."}.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Server is the bcd HTTP server.
type Server struct {
	agents     *agent.AgentService
	channels   *channel.ChannelService
	daemons    *daemon.Manager
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
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentByName)
	mux.HandleFunc("/api/channels", s.handleChannels)
	mux.HandleFunc("/api/workspace/status", s.handleWorkspaceStatus)
	mux.HandleFunc("/api/daemons", s.handleDaemons)

	s.httpServer = &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s
}

// Addr returns the resolved listen address after Start is called.
func (s *Server) Addr() string {
	return s.addr
}

// Start begins listening on the configured address.
// It blocks until the server is shut down.
func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
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
		type agentDTO struct {
			CreatedAt time.Time `json:"created_at"`
			Name      string    `json:"name"`
			Role      string    `json:"role"`
			State     string    `json:"state"`
			Task      string    `json:"task,omitempty"`
			Team      string    `json:"team,omitempty"`
			Tool      string    `json:"tool,omitempty"`
			Session   string    `json:"session,omitempty"`
		}
		dtos := make([]agentDTO, 0, len(agents))
		for _, a := range agents {
			dtos = append(dtos, agentDTO{
				Name:      a.Name,
				Role:      string(a.Role),
				State:     string(a.State),
				Task:      a.Task,
				Team:      a.Team,
				Tool:      a.Tool,
				Session:   a.Session,
				CreatedAt: a.CreatedAt,
			})
		}
		writeJSON(w, http.StatusOK, dtos)

	default:
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleAgentByName handles /api/agents/:name and /api/agents/:name/action
func (s *Server) handleAgentByName(w http.ResponseWriter, r *http.Request) {
	// Extract name from path: /api/agents/<name> or /api/agents/<name>/stop
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
		writeJSON(w, http.StatusOK, a)

	case r.Method == http.MethodPost && action == "stop":
		if err := s.agents.Stop(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})

	case r.Method == http.MethodPost && action == "start":
		a, err := s.agents.Start(r.Context(), name, agent.StartOptions{})
		if err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, a)

	case r.Method == http.MethodDelete && action == "":
		if err := s.agents.Delete(r.Context(), name); err != nil {
			httpError(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		httpError(w, "not found", http.StatusNotFound)
	}
}

func (s *Server) handleChannels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channels, err := s.channels.List(r.Context())
	if err != nil {
		httpError(w, fmt.Sprintf("list channels: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, channels)
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
