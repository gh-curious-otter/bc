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
	"strings"
	"time"

	"github.com/gh-curious-otter/bc/pkg/agent"
	"github.com/gh-curious-otter/bc/pkg/channel"
	"github.com/gh-curious-otter/bc/pkg/cost"
	"github.com/gh-curious-otter/bc/pkg/cron"
	"github.com/gh-curious-otter/bc/pkg/events"
	"github.com/gh-curious-otter/bc/pkg/gateway"
	"github.com/gh-curious-otter/bc/pkg/log"
	"github.com/gh-curious-otter/bc/pkg/mcp"
	"github.com/gh-curious-otter/bc/pkg/secret"
	"github.com/gh-curious-otter/bc/pkg/stats"
	"github.com/gh-curious-otter/bc/pkg/tool"
	"github.com/gh-curious-otter/bc/pkg/workspace"
	"github.com/gh-curious-otter/bc/server/handlers"
	servermcp "github.com/gh-curious-otter/bc/server/mcp"
	"github.com/gh-curious-otter/bc/server/ws"
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
	Costs        *cost.Store
	CostImporter *cost.Importer
	Cron         *cron.Store
	Secrets      *secret.Store
	MCP          *mcp.Store
	Tools        *tool.Store
	Stats        *stats.Store
	EventLog     events.EventStore
	WS           *workspace.Workspace
	Gateway      *gateway.Manager
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
		ah := handlers.NewAgentHandler(svc.Agents, svc.Costs, svc.WS, hub)
		if svc.EventLog != nil {
			ah.SetEventStore(svc.EventLog)
		}
		ah.SetTerminalHandler(handlers.NewTerminalHandler(svc.Agents, cfg.CORSOrigin))
		ah.Register(mux)
	}
	if svc.Channels != nil {
		svc.Channels.OnMessage = func(ch, sender, content string) {
			// Route outbound to gateway if this is a gateway channel
			if svc.Gateway != nil && svc.Gateway.IsGatewayChannel(ch) {
				// Don't re-send messages that came FROM the gateway (indicated by [platform] prefix)
				if !strings.HasPrefix(sender, "[telegram]") &&
					!strings.HasPrefix(sender, "[discord]") &&
					!strings.HasPrefix(sender, "[slack]") {
					if _, err := svc.Gateway.Send(context.Background(), ch, sender, content); err != nil {
						log.Warn("gateway outbound failed", "channel", ch, "error", err)
					}
				}
			}
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
			// Agent delivery via MCP SSE is wired below after MCP server
			// initialization — see the mcpBroker.SendToAgents call.
		}
		handlers.NewChannelHandler(svc.Channels).Register(mux)
		handlers.NewChannelStatsHandler(svc.Channels).Register(mux)
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
	if svc.Gateway != nil {
		handlers.NewGatewayHandler(svc.Gateway).Register(mux)
	}
	if svc.WS != nil {
		handlers.NewRolesHandler(svc.WS).Register(mux)
		handlers.NewWorkspaceHandler(svc.Agents, svc.WS).Register(mux)
		handlers.NewDoctorHandler(svc.WS).Register(mux)
		handlers.NewSettingsHandler(svc.WS).Register(mux)
	}

	// Stats endpoints (always registered; nil-safe internally)
	handlers.NewStatsHandler(svc.Agents, svc.Channels, svc.Costs, svc.Tools, svc.WS, svc.Stats).Register(mux)

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
			mcpBroker := servermcp.MountOn(mux, mcpSrv, "/mcp")

			// Wire MCP SSE delivery into the OnMessage callback.
			// When a channel message is stored, push a notification directly
			// to member agents via the MCP SSE broker — no poller needed.
			if svc.Channels != nil && mcpBroker != nil {
				prevOnMessage := svc.Channels.OnMessage
				svc.Channels.OnMessage = func(ch, sender, content string) {
					// Call previous OnMessage (gateway routing + web UI hub)
					if prevOnMessage != nil {
						prevOnMessage(ch, sender, content)
					}
					// Push MCP SSE notification to channel members
					members, err := svc.Channels.Store().GetMembers(ch)
					if err != nil || len(members) == 0 {
						return
					}
					memberSet := make(map[string]bool, len(members))
					for _, m := range members {
						memberSet[m] = true
					}
					notification := servermcp.NewChannelNotification(ch, sender, content, time.Now())
					mcpBroker.SendToAgents(notification, memberSet)
				}
			}
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
