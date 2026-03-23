// Command bcd is the bc coordination daemon.
// It starts an HTTP server exposing workspace state via a REST API and
// SSE event stream so that bc CLI thin-client commands can talk to it.
//
// Usage:
//
//	bcd [--addr ADDR] [--workspace DIR] [--verbose] [--log-format text|json] [--cors-origin ORIGIN]
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
	bcevents "github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	bcmcp "github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/provider"
	bcsecret "github.com/rpuneet/bc/pkg/secret"
	bcstats "github.com/rpuneet/bc/pkg/stats"
	bcteam "github.com/rpuneet/bc/pkg/team"
	bctool "github.com/rpuneet/bc/pkg/tool"
	bcworkspace "github.com/rpuneet/bc/pkg/workspace"
	"github.com/rpuneet/bc/server"
	bcws "github.com/rpuneet/bc/server/ws"
)

// Build information set by ldflags during build.
var (
	commit = "unknown"
	date   = "unknown"
)

func main() {
	addr := flag.String("addr", server.DefaultConfig().Addr, "listen address")
	wsRoot := flag.String("workspace", ".", "workspace root directory")
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	logFormat := flag.String("log-format", "text", "log output format (text|json)")
	corsOrigin := flag.String("cors-origin", "*", "CORS allowed origin (* for permissive, or specific origin)")
	flag.Parse()

	if *logFormat == "json" {
		log.SetFormat("json")
	}
	if *verbose {
		log.SetVerbose(true)
	}

	if err := run(*addr, *wsRoot, *corsOrigin); err != nil {
		fmt.Fprintf(os.Stderr, "bcd: %v\n", err)
		os.Exit(1)
	}
}

func run(addr, wsRoot, corsOrigin string) error {
	ws, err := bcworkspace.Load(wsRoot)
	if err != nil {
		ws, err = bcworkspace.Init(wsRoot)
		if err != nil {
			return fmt.Errorf("init workspace %s: %w", wsRoot, err)
		}
	}

	pidPath := filepath.Join(ws.RootDir, ".bc", "bcd.pid")
	if err := writePID(pidPath); err != nil {
		log.Warn("failed to write PID file", "path", pidPath, "error", err)
	}
	defer os.Remove(pidPath) //nolint:errcheck // best-effort cleanup

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	hub := bcws.NewHub()
	go hub.Run()
	defer hub.Stop()

	agentMgr := newAgentManager(ws)
	if err := agentMgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}
	defer agentMgr.Close() //nolint:errcheck // best-effort
	go agentMgr.RunReconciler(ctx, 10*time.Second)
	agentSvc := bcagent.NewAgentService(agentMgr, hub, nil)

	statsCollector := bcagent.NewStatsCollector(agentMgr)
	go statsCollector.Run(ctx)

	var channelSvc *bcchannel.ChannelService
	if chStore, err := bcchannel.OpenStore(ws.RootDir); err != nil {
		log.Warn("channel store unavailable", "error", err)
	} else {
		channelSvc = bcchannel.NewChannelService(chStore)
		defer chStore.Close() //nolint:errcheck // best-effort
	}

	var costStore *bccost.Store
	var costImporter *bccost.Importer
	if cs, err := bccost.OpenStore(ws.RootDir); err != nil {
		log.Warn("cost store unavailable", "error", err)
	} else {
		costStore = cs
		defer cs.Close() //nolint:errcheck // best-effort

		costImporter = bccost.NewImporter(cs, ws.RootDir)
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

	var cronStore *bccron.Store
	if cr, err := bccron.Open(ws.RootDir); err != nil {
		log.Warn("cron store unavailable", "error", err)
	} else {
		cronStore = cr
		defer cr.Close() //nolint:errcheck // best-effort

		cronSched := bccron.NewScheduler(cr)
		go cronSched.Run(ctx)
	}

	var secretStore *bcsecret.Store
	passphrase, passphraseErr := bcsecret.Passphrase()
	if passphraseErr != nil {
		log.Warn("secret passphrase unavailable \u2014 secret store disabled", "error", passphraseErr)
	}
	if passphraseErr == nil {
		if ss, err := bcsecret.NewStore(ws.RootDir, passphrase); err != nil {
			log.Warn("secret store unavailable", "error", err)
		} else {
			secretStore = ss
			defer ss.Close() //nolint:errcheck // best-effort
		}
	}

	var mcpStore *bcmcp.Store
	if ms, err := bcmcp.NewStore(ws.RootDir); err != nil {
		log.Warn("mcp store unavailable", "error", err)
	} else {
		mcpStore = ms
		defer ms.Close() //nolint:errcheck // best-effort
	}

	var toolStore *bctool.Store
	ts := bctool.NewStore(ws.StateDir())
	if err := ts.Open(); err != nil {
		log.Warn("tool store unavailable", "error", err)
	} else {
		toolStore = ts
		defer ts.Close() //nolint:errcheck // best-effort
	}

	var eventLog bcevents.EventStore
	if el, err := bcevents.OpenLog(ws.RootDir, filepath.Join(ws.StateDir(), "state.db")); err != nil {
		log.Warn("event log unavailable", "error", err)
	} else {
		eventLog = el
		defer el.Close() //nolint:errcheck // best-effort
	}

	// TimescaleDB stats store (optional — nil when STATS_DATABASE_URL is not set)
	var statsStore *bcstats.Store
	if dsn := bcstats.StatsDSN(); dsn != bcstats.DefaultStatsDSN || os.Getenv("STATS_DATABASE_URL") != "" {
		if ss, err := bcstats.NewStore(dsn); err != nil {
			log.Warn("stats store unavailable (TimescaleDB)", "error", err)
		} else {
			statsStore = ss
			defer ss.Close() //nolint:errcheck // best-effort
			log.Info("stats store: using TimescaleDB", "dsn", dsn)

			// Background system metrics collector
			go runStatsCollector(ctx, ss, agentSvc, channelSvc, ws)
		}
	}

	teamStore := bcteam.NewStore(ws.RootDir)

	svc := server.Services{
		Agents:       agentSvc,
		Channels:     channelSvc,
		Costs:        costStore,
		CostImporter: costImporter,
		Cron:         cronStore,
		Secrets:      secretStore,
		MCP:          mcpStore,
		Tools:        toolStore,
		Stats:        statsStore,
		EventLog:     eventLog,
		Teams:        teamStore,
		WS:           ws,
	}

	cfg := server.DefaultConfig()
	if addr != "" {
		cfg.Addr = addr
	}
	cfg.CORSOrigin = corsOrigin
	cfg.Build = server.BuildInfo{
		Commit:  commit,
		BuiltAt: date,
	}

	srv := server.New(cfg, svc, hub, server.WebDist())
	return srv.Start(ctx)
}

func newAgentManager(ws *bcworkspace.Workspace) *bcagent.Manager {
	var wsCfg bcworkspace.DockerRuntimeConfig
	if ws.Config != nil {
		wsCfg = ws.Config.Runtime.Docker
	}
	dockerCfg := bccontainer.ConfigFromWorkspace(wsCfg)
	be, err := bccontainer.NewBackend(dockerCfg, "bc-", ws.RootDir, provider.DefaultRegistry)
	if err != nil {
		log.Warn("Docker not available — agents will use tmux runtime only", "error", err)
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

// runStatsCollector periodically samples system and agent metrics into TimescaleDB.
func runStatsCollector(ctx context.Context, ss *bcstats.Store, agents *bcagent.AgentService, channels *bcchannel.ChannelService, ws *bcworkspace.Workspace) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	hostname, _ := os.Hostname() //nolint:errcheck // best-effort

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			// System metrics
			_ = ss.RecordSystem(ctx, bcstats.SystemMetric{
				Time:     now,
				Hostname: hostname,
			})

			// Agent metrics
			if agents != nil {
				agentList, err := agents.List(ctx, bcagent.ListOptions{})
				if err == nil {
					for _, a := range agentList {
						_ = ss.RecordAgent(ctx, bcstats.AgentMetric{
							Time:      now,
							AgentName: a.Name,
							AgentID:   a.ID,
							Role:      string(a.Role),
							State:     string(a.State),
						})
					}
				}
			}

			// Channel metrics
			if channels != nil {
				chList, err := channels.List(ctx)
				if err == nil {
					for _, ch := range chList {
						_ = ss.RecordChannel(ctx, bcstats.ChannelMetric{
							Time:         now,
							ChannelName:  ch.Name,
							MessagesSent: int64(ch.MessageCount),
							Participants: len(ch.Members),
						})
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}
