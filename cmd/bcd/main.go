// Command bcd is the bc coordination daemon.
// It starts an HTTP server exposing workspace state via a REST API and
// SSE event stream so that bc CLI thin-client commands can talk to it.
//
// Usage:
//
//	bcd [--addr ADDR] [--workspace DIR] [--verbose]
//
// The server binds to 127.0.0.1:9374 by default.
// A PID file is written to <workspace>/.bc/bcd.pid on startup.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	bcagent "github.com/rpuneet/bc/pkg/agent"
	bcchannel "github.com/rpuneet/bc/pkg/channel"
	bccontainer "github.com/rpuneet/bc/pkg/container"
	bccost "github.com/rpuneet/bc/pkg/cost"
	bccron "github.com/rpuneet/bc/pkg/cron"
	bcdaemon "github.com/rpuneet/bc/pkg/daemon"
	bcevents "github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	bcmcp "github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/provider"
	bcsecret "github.com/rpuneet/bc/pkg/secret"
	bctool "github.com/rpuneet/bc/pkg/tool"
	bcworkspace "github.com/rpuneet/bc/pkg/workspace"
	"github.com/rpuneet/bc/server"
	bcws "github.com/rpuneet/bc/server/ws"
)

func main() {
	addr := flag.String("addr", server.DefaultConfig().Addr, "listen address")
	wsRoot := flag.String("workspace", ".", "workspace root directory")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	if *verbose {
		log.SetVerbose(true)
	}

	if err := run(*addr, *wsRoot); err != nil {
		fmt.Fprintf(os.Stderr, "bcd: %v\n", err)
		os.Exit(1)
	}
}

func run(addr, wsRoot string) error {
	// Try to load existing workspace; if none exists, initialize a minimal one.
	// This allows bcd to run in a fresh Docker container without a pre-existing workspace.
	ws, err := bcworkspace.Load(wsRoot)
	if err != nil {
		ws, err = bcworkspace.Init(wsRoot)
		if err != nil {
			return fmt.Errorf("init workspace %s: %w", wsRoot, err)
		}
	}

	// Write PID file
	pidPath := filepath.Join(ws.RootDir, ".bc", "bcd.pid")
	if err := writePID(pidPath); err != nil {
		log.Warn("failed to write PID file", "path", pidPath, "error", err)
	}
	defer os.Remove(pidPath) //nolint:errcheck // best-effort cleanup

	// Context — create early so goroutines can use it.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// SSE hub
	hub := bcws.NewHub()
	go hub.Run()
	defer hub.Stop()

	// Agent service
	agentMgr := newAgentManager(ws)
	if err := agentMgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}
	agentSvc := bcagent.NewAgentService(agentMgr, hub, nil)

	// Stats collector: polls Docker stats + consumes hook event files every 30s.
	statsCollector := bcagent.NewStatsCollector(agentMgr)
	go statsCollector.Run(ctx)

	// Channel service
	var channelSvc *bcchannel.ChannelService
	if chStore, err := bcchannel.OpenStore(ws.RootDir); err != nil {
		log.Warn("channel store unavailable", "error", err)
	} else {
		channelSvc = bcchannel.NewChannelService(chStore)
		defer chStore.Close() //nolint:errcheck // best-effort
	}

	// Daemon manager
	var daemonMgr *bcdaemon.Manager
	if mgr, err := bcdaemon.NewManager(ws.RootDir); err != nil {
		log.Warn("daemon manager unavailable", "error", err)
	} else {
		daemonMgr = mgr
		defer mgr.Close() //nolint:errcheck // best-effort
	}

	// Cost store + importer
	var costStore *bccost.Store
	var costImporter *bccost.Importer
	cs := bccost.NewStore(ws.RootDir)
	if err := cs.Open(); err != nil {
		log.Warn("cost store unavailable", "error", err)
	} else {
		costStore = cs
		defer cs.Close() //nolint:errcheck // best-effort

		costImporter = bccost.NewImporter(cs, ws.RootDir)
		// Run initial import and schedule periodic refresh every 5 minutes.
		go func() {
			if n, err := costImporter.ImportAll(ctx); err != nil {
				log.Warn("cost import failed", "error", err)
			} else if n > 0 {
				log.Info("cost import: imported records", "count", n)
			}
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if n, err := costImporter.ImportAll(ctx); err != nil {
						log.Warn("cost import failed", "error", err)
					} else if n > 0 {
						log.Info("cost import: imported records", "count", n)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Cron store
	var cronStore *bccron.Store
	if cr, err := bccron.Open(ws.RootDir); err != nil {
		log.Warn("cron store unavailable", "error", err)
	} else {
		cronStore = cr
		defer cr.Close() //nolint:errcheck // best-effort
	}

	// Secret store
	var secretStore *bcsecret.Store
	passphrase, passphraseErr := bcsecret.Passphrase()
	if passphraseErr != nil {
		log.Warn("secret passphrase unavailable — secret store disabled", "error", passphraseErr)
	}
	if passphraseErr == nil {
		if ss, err := bcsecret.NewStore(ws.RootDir, passphrase); err != nil {
			log.Warn("secret store unavailable", "error", err)
		} else {
			secretStore = ss
			defer ss.Close() //nolint:errcheck // best-effort
		}
	}

	// MCP store
	var mcpStore *bcmcp.Store
	if ms, err := bcmcp.NewStore(ws.RootDir); err != nil {
		log.Warn("mcp store unavailable", "error", err)
	} else {
		mcpStore = ms
		defer ms.Close() //nolint:errcheck // best-effort
	}

	// Tool store
	var toolStore *bctool.Store
	ts := bctool.NewStore(ws.StateDir())
	if err := ts.Open(); err != nil {
		log.Warn("tool store unavailable", "error", err)
	} else {
		toolStore = ts
		defer ts.Close() //nolint:errcheck // best-effort
	}

	// Event log
	var eventLog bcevents.EventStore
	if el, err := bcevents.NewSQLiteLog(filepath.Join(ws.StateDir(), "state.db")); err != nil {
		log.Warn("event log unavailable", "error", err)
	} else {
		eventLog = el
		defer el.Close() //nolint:errcheck // best-effort
	}

	svc := server.Services{
		Agents:       agentSvc,
		Channels:     channelSvc,
		Daemons:      daemonMgr,
		Costs:        costStore,
		CostImporter: costImporter,
		Cron:         cronStore,
		Secrets:      secretStore,
		MCP:          mcpStore,
		Tools:        toolStore,
		EventLog:     eventLog,
		WS:           ws,
	}

	cfg := server.DefaultConfig()
	if addr != "" {
		cfg.Addr = addr
	}

	srv := server.New(cfg, svc, hub, server.WebDist())
	return srv.Start(ctx)
}

// newAgentManager creates an agent manager with Docker support if available.
// bcd gets access to the host Docker socket via volume mount so it can
// manage agent containers.
func newAgentManager(ws *bcworkspace.Workspace) *bcagent.Manager {
	var wsCfg bcworkspace.DockerRuntimeConfig
	if ws.Config != nil {
		wsCfg = ws.Config.Runtime.Docker
	}
	dockerCfg := bccontainer.ConfigFromWorkspace(wsCfg)
	be, err := bccontainer.NewBackend(dockerCfg, "bc-", ws.RootDir, provider.DefaultRegistry)
	if err != nil {
		log.Debug("Docker not available in bcd container", "error", err)
		return bcagent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	}
	return bcagent.NewWorkspaceManagerWithRuntime(ws.AgentsDir(), ws.RootDir, be, "docker")
}

func writePID(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create pid dir: %w", err)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0600)
}
