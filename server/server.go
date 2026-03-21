// Package server implements the bcd HTTP API server.
//
// The server exposes workspace state over HTTP so the bc CLI can operate as a
// thin client. It binds to localhost only by default and serves:
//
//   - REST API at /api/…  (JSON, one handler file per resource)
//   - SSE stream at /api/events  (real-time agent state updates)
//   - Static web UI at /  (embedded web/dist, served when built)
//   - Health probe at /health
//
// Default address: 127.0.0.1:9374
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/cron"
	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/secret"
	"github.com/rpuneet/bc/pkg/team"
	"github.com/rpuneet/bc/pkg/tool"
	"github.com/rpuneet/bc/pkg/workspace"
	"github.com/rpuneet/bc/server/handlers"
	servermcp "github.com/rpuneet/bc/server/mcp"
	"github.com/rpuneet/bc/server/ws"
)

const defaultAddr = "127.0.0.1:9374"

// BuildInfo holds build-time metadata injected via ldflags.
type BuildInfo struct {
	Commit  string // short git commit hash
	BuiltAt string // UTC build timestamp (RFC 3339)
}

// Config holds server configuration.
type Config struct {
	Build      BuildInfo // build-time metadata
	Addr       string    // default "127.0.0.1:9374"
	CORSOrigin string    // allowed origin (default "*")
	CORS       bool      // enable permissive CORS headers (safe for loopback)
}

// DefaultConfig returns the default server configuration.
func DefaultConfig() Config {
	return Config{Addr: defaultAddr, CORS: true}
}

// Services bundles all service/store dependencies for the handlers.
type Services struct {
	Agents       *agent.AgentService
	Channels     *channel.ChannelService
	Daemons      *daemon.Manager
	Costs        *cost.Store
	CostImporter *cost.Importer
	Cron         *cron.Store
	Secrets      *secret.Store
	MCP          *mcp.Store
	Teams        *team.Store
	Tools        *tool.Store
	EventLog     events.EventStore
	WS           *workspace.Workspace
}

// Server is the bcd HTTP server.
type Server struct {
	httpServer *http.Server
	handler    http.Handler
	addr       string
}

// New creates a bcd server with the given config, services, SSE hub, and optional static files.
func New(cfg Config, svc Services, hub *ws.Hub, staticFiles fs.FS) *Server {
	if cfg.Addr == "" {
		cfg.Addr = defaultAddr
	}

	mux := http.NewServeMux()

	// Health probes
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","addr":%q,"commit":%q,"built_at":%q}`, cfg.Addr, cfg.Build.Commit, cfg.Build.BuiltAt) //nolint:errcheck // writing to response
	})

	// Readiness probe — verifies downstream dependencies
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		checks := map[string]string{}
		status := "ok"

		// Check database connectivity
		if svc.Costs != nil {
			if _, err := svc.Costs.WorkspaceSummary(r.Context()); err != nil {
				checks["db"] = "error: " + err.Error()
				status = "degraded"
			} else {
				checks["db"] = "ok"
			}
		}

		// Check agent runtime
		if svc.Agents != nil {
			checks["agents"] = fmt.Sprintf("%d total", len(svc.Agents.Manager().ListAgents()))
		}

		w.Header().Set("Content-Type", "application/json")
		if status != "ok" {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		writeJSON := func(v any) {
			_ = json.NewEncoder(w).Encode(v) //nolint:errcheck
		}
		writeJSON(map[string]any{"status": status, "checks": checks})
	})

	// SSE event stream
	if hub != nil {
		mux.Handle("/api/events", hub)
	}

	// Resource handlers (only registered when service is available)
	if svc.Agents != nil {
		handlers.NewAgentHandler(svc.Agents, svc.Costs, svc.WS).Register(mux)
	}
	if svc.Channels != nil {
		svc.Channels.OnMessage = func(ch, sender, content string) {
			// Publish SSE event for web UI (non-blocking)
			if hub != nil {
				hub.Publish("channel.message", map[string]any{
					"channel": ch,
					"message": map[string]any{
						"sender":  sender,
						"content": content,
						"type":    "text",
					},
				})
			}
			// Deliver to agent tmux/docker sessions asynchronously.
			// Messages are already persisted to SQLite before OnMessage fires,
			// so delivery is best-effort — agents can read history on reconnect.
			if svc.Agents != nil {
				go func() {
					formatted := fmt.Sprintf("[bc-mcp][%s][#%s] %s: %s", time.Now().UTC().Format(time.RFC3339), ch, sender, content)
					chDTO, err := svc.Channels.Get(context.Background(), ch)
					if err != nil {
						log.Debug("channel send: failed to get channel", "channel", ch, "error", err)
						return
					}
					// Parse @mentions to filter delivery targets.
					// If mentions exist, only deliver to mentioned agents.
					// If no mentions, deliver to all members (backward compat).
					mentionedAgents, hasAll := channel.ExtractMentionedAgents(content)
					hasMentions := hasAll || len(mentionedAgents) > 0

					for _, member := range chDTO.Members {
						if member == sender {
							continue
						}
						// If mentions are present, only deliver to mentioned agents
						if hasMentions && !channel.ContainsMention(content, member) {
							continue
						}
						// Retry delivery up to 3 times
						var sendErr error
						for attempt := 0; attempt < 3; attempt++ {
							sendErr = svc.Agents.Send(context.Background(), member, formatted)
							if sendErr == nil {
								break
							}
							time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
						}
						if sendErr != nil {
							log.Warn("channel send: delivery failed after retries", "channel", ch, "agent", member, "error", sendErr)
						}
					}
				}()
			}
		}
		handlers.NewChannelHandler(svc.Channels).Register(mux)
		handlers.NewChannelStatsHandler(svc.Channels).Register(mux)
	}
	if svc.Daemons != nil {
		handlers.NewDaemonHandler(svc.Daemons).Register(mux)
	}
	if svc.Costs != nil {
		handlers.NewCostHandler(svc.Costs, svc.CostImporter).Register(mux)
	}
	if svc.Cron != nil {
		handlers.NewCronHandler(svc.Cron).Register(mux)
	}
	if svc.Secrets != nil {
		handlers.NewSecretHandler(svc.Secrets).Register(mux)
	}
	if svc.MCP != nil {
		handlers.NewMCPHandler(svc.MCP).Register(mux)
	}
	if svc.Tools != nil {
		handlers.NewToolHandler(svc.Tools).Register(mux)
	}
	if svc.EventLog != nil {
		handlers.NewEventHandler(svc.EventLog).Register(mux)
	}
	if svc.Teams != nil {
		handlers.NewTeamHandler(svc.Teams).Register(mux)
	}
	if svc.WS != nil {
		handlers.NewRolesHandler(svc.WS).Register(mux)
		handlers.NewWorkspaceHandler(svc.Agents, svc.WS).Register(mux)
		handlers.NewDoctorHandler(svc.WS).Register(mux)
		handlers.NewSettingsHandler(svc.WS).Register(mux)
	}

	// Stats endpoints (always registered; nil-safe internally)
	handlers.NewStatsHandler(svc.Agents, svc.Channels, svc.Costs, svc.Tools, svc.WS).Register(mux)

	// MCP protocol server (SSE transport) at /mcp/
	if svc.WS != nil {
		mcpCfg := servermcp.Config{Workspace: svc.WS, Costs: svc.Costs}
		if svc.Agents != nil {
			mcpCfg.Agents = svc.Agents.Manager()
		}
		if svc.Channels != nil {
			mcpCfg.Channels = svc.Channels.Store()
			mcpCfg.ChannelService = svc.Channels
		}
		mcpSrv, mcpErr := servermcp.New(mcpCfg)
		if mcpErr != nil {
			log.Warn("MCP server unavailable", "error", mcpErr)
		} else {
			servermcp.MountOn(mux, mcpSrv, "/mcp")
		}
	}

	// Static web UI with SPA fallback — serves files if they exist,
	// otherwise falls back to index.html for client-side routing.
	if staticFiles != nil {
		fileServer := http.FileServer(http.FS(staticFiles))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Try serving the exact file first
			path := r.URL.Path
			if path != "/" {
				if f, err := staticFiles.Open(path[1:]); err == nil {
					_ = f.Close() //nolint:errcheck // best-effort close
					fileServer.ServeHTTP(w, r)
					return
				}
			}
			// Fallback: serve index.html for SPA client-side routes
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	// Middleware chain (outermost runs first):
	// RateLimit → RequestID → RequestLogger → Recovery → Gzip → MaxBodySize → CORS → mux
	var handler http.Handler = mux
	if cfg.CORS {
		origin := cfg.CORSOrigin
		if origin == "" {
			origin = "*"
		}
		handler = handlers.CORSWithOrigin(origin, mux)
	}
	handler = handlers.MaxBodySize(1 << 20)(handler) // 1MB request body limit
	handler = handlers.Gzip(handler)
	handler = handlers.Recovery(handler)
	handler = handlers.RequestLogger(handler)
	handler = handlers.RequestID(handler)
	limiter := handlers.NewRateLimiter(100, 200)
	handler = handlers.RateLimit(limiter)(handler)

	return &Server{
		addr:    cfg.Addr,
		handler: handler,
		httpServer: &http.Server{
			Addr:        cfg.Addr,
			Handler:     handler,
			ReadTimeout: 30 * time.Second,
			// WriteTimeout must be 0 for SSE connections (/api/events) which are long-lived.
			// Per-handler timeouts are used instead where needed.
			WriteTimeout: 0,
			IdleTimeout:  120 * time.Second,
		},
	}
}

// Handler returns the HTTP handler (useful for httptest.NewServer in tests).
func (s *Server) Handler() http.Handler {
	return s.handler
}

// Addr returns the resolved listen address (updated after Start is called with :0).
func (s *Server) Addr() string {
	return s.addr
}

// Start begins listening. It blocks until ctx is canceled or an error occurs.
func (s *Server) Start(ctx context.Context) error {
	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}
	s.addr = ln.Addr().String()

	log.Info("bcd listening", "addr", s.addr)

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := s.httpServer.Shutdown(shutCtx); err != nil {
			log.Warn("server shutdown error", "error", err)
		}
	}()

	if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}
