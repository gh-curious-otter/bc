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
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/cron"
	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/secret"
	"github.com/rpuneet/bc/pkg/tool"
	"github.com/rpuneet/bc/pkg/workspace"
	"github.com/rpuneet/bc/server/handlers"
	servermcp "github.com/rpuneet/bc/server/mcp"
	"github.com/rpuneet/bc/server/ws"
)

const defaultAddr = "127.0.0.1:9374"

// Config holds server configuration.
type Config struct {
	Addr     string // default "127.0.0.1:9374"
	AddrFile string // path to write the resolved listen address on startup (e.g. .bc/bcd.addr)
	CORS     bool   // enable permissive CORS headers (safe for loopback)
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
	Tools        *tool.Store
	WS           *workspace.Workspace
}

// Server is the bcd HTTP server.
type Server struct {
	httpServer *http.Server
	handler    http.Handler
	addrFile   string
	addr       string
}

// New creates a bcd server with the given config, services, SSE hub, and optional static files.
func New(cfg Config, svc Services, hub *ws.Hub, staticFiles fs.FS) *Server {
	if cfg.Addr == "" {
		cfg.Addr = defaultAddr
	}

	mux := http.NewServeMux()

	// Health probe
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","addr":%q}`, cfg.Addr)
	})

	// SSE event stream
	if hub != nil {
		mux.Handle("/api/events", hub)
	}

	// Resource handlers (only registered when service is available)
	if svc.Agents != nil {
		handlers.NewAgentHandler(svc.Agents).Register(mux)
	}
	if svc.Channels != nil {
		handlers.NewChannelHandler(svc.Channels).Register(mux)
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
	if svc.WS != nil {
		handlers.NewWorkspaceHandler(svc.Agents, svc.WS).Register(mux)
		handlers.NewDoctorHandler(svc.WS).Register(mux)
	}

	// MCP protocol server (SSE transport) at /mcp/
	if svc.WS != nil {
		mcpSrv, mcpErr := servermcp.New(servermcp.Config{Workspace: svc.WS})
		if mcpErr != nil {
			log.Warn("MCP server unavailable", "error", mcpErr)
		} else {
			servermcp.MountOn(mux, mcpSrv, "/mcp")
		}
	}

	// Static web UI (served last so API routes win)
	if staticFiles != nil {
		mux.Handle("/", http.FileServer(http.FS(staticFiles)))
	}

	var handler http.Handler = mux
	if cfg.CORS {
		handler = handlers.CORS(mux)
	}

	return &Server{
		addr:     cfg.Addr,
		addrFile: cfg.AddrFile,
		handler:  handler,
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
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", s.addr, err)
	}
	s.addr = ln.Addr().String()

	log.Info("bcd listening", "addr", s.addr)

	// Write the resolved address to bcd.addr so the CLI can discover it.
	if s.addrFile != "" {
		httpAddr := "http://" + s.addr
		if err := writeAddrFile(s.addrFile, httpAddr); err != nil {
			log.Warn("failed to write addr file", "path", s.addrFile, "error", err)
		} else {
			defer os.Remove(s.addrFile) //nolint:errcheck // best-effort cleanup
		}
	}

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

func writeAddrFile(path, addr string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(addr+"\n"), 0600)
}
