package server

import (
	"context"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/container"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/provider"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Config holds server configuration.
type Config struct {
	Addr string // listen address, default "127.0.0.1:9374"
	Dir  string // workspace root directory
}

// Server is the bcd HTTP API server.
type Server struct {
	startedAt time.Time
	events    events.EventStore
	ws        *workspace.Workspace
	agents    *agent.Manager
	channels  *channel.SQLiteStore
	costs     *cost.Store
	httpSrv   *http.Server
	cfg       Config
}

// New creates a new Server from the given config.
func New(cfg Config) (*Server, error) {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:9374"
	}

	ws, err := workspace.Load(cfg.Dir)
	if err != nil {
		return nil, err
	}

	mgr := newAgentManager(ws)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	ch := channel.NewSQLiteStore(ws.RootDir)
	if openErr := ch.Open(); openErr != nil {
		return nil, openErr
	}

	cs := cost.NewStore(ws.RootDir)
	if openErr := cs.Open(); openErr != nil {
		_ = ch.Close()
		return nil, openErr
	}

	ev, err := events.NewSQLiteLog(filepath.Join(ws.StateDir(), "state.db"))
	if err != nil {
		_ = ch.Close()
		_ = cs.Close()
		return nil, err
	}

	return &Server{
		cfg:      cfg,
		ws:       ws,
		agents:   mgr,
		channels: ch,
		costs:    cs,
		events:   ev,
	}, nil
}

// newAgentManager creates an agent manager with the appropriate runtime backend.
func newAgentManager(ws *workspace.Workspace) *agent.Manager {
	if ws.Config != nil && ws.Config.Runtime.Backend == "docker" {
		dockerCfg := container.ConfigFromWorkspace(ws.Config.Runtime.Docker)
		backend, err := container.NewBackend(dockerCfg, "bc-", ws.RootDir, provider.DefaultRegistry)
		if err != nil {
			log.Warn("Docker unavailable, falling back to tmux", "error", err)
		} else {
			return agent.NewWorkspaceManagerWithRuntime(ws.AgentsDir(), ws.RootDir, backend)
		}
	}
	return agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
}

// Start sets up routes, starts the HTTP server, and blocks until ctx is canceled.
func (s *Server) Start(ctx context.Context) error {
	s.startedAt = time.Now()

	s.httpSrv = &http.Server{
		Addr:              s.cfg.Addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("bcd listening", "addr", s.cfg.Addr)
		if err := s.httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully stops the HTTP server and closes all stores.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("shutting down bcd")

	var errs []error
	if s.httpSrv != nil {
		if err := s.httpSrv.Shutdown(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	if err := s.channels.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.costs.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.events.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
