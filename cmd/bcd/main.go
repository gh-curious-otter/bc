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

	bcagent "github.com/rpuneet/bc/pkg/agent"
	bcchannel "github.com/rpuneet/bc/pkg/channel"
	bccost "github.com/rpuneet/bc/pkg/cost"
	bccron "github.com/rpuneet/bc/pkg/cron"
	bcdaemon "github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	bcmcp "github.com/rpuneet/bc/pkg/mcp"
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
	ws, err := bcworkspace.Load(wsRoot)
	if err != nil {
		return fmt.Errorf("load workspace %s: %w", wsRoot, err)
	}

	// Write PID file
	pidPath := filepath.Join(ws.RootDir, ".bc", "bcd.pid")
	if err := writePID(pidPath); err != nil {
		log.Warn("failed to write PID file", "path", pidPath, "error", err)
	}
	defer os.Remove(pidPath) //nolint:errcheck // best-effort cleanup

	// SSE hub
	hub := bcws.NewHub()
	go hub.Run()
	defer hub.Stop()

	// Agent service
	agentMgr := bcagent.NewManager(ws.StateDir())
	if err := agentMgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}
	agentSvc := bcagent.NewAgentService(agentMgr, hub, nil)

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

	// Cost store
	var costStore *bccost.Store
	cs := bccost.NewStore(ws.RootDir)
	if err := cs.Open(); err != nil {
		log.Warn("cost store unavailable", "error", err)
	} else {
		costStore = cs
		defer cs.Close() //nolint:errcheck // best-effort
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

	svc := server.Services{
		Agents:   agentSvc,
		Channels: channelSvc,
		Daemons:  daemonMgr,
		Costs:    costStore,
		Cron:     cronStore,
		Secrets:  secretStore,
		MCP:      mcpStore,
		Tools:    toolStore,
		WS:       ws,
	}

	cfg := server.DefaultConfig()
	if addr != "" {
		cfg.Addr = addr
	}

	srv := server.New(cfg, svc, hub, server.WebDist())

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	return srv.Start(ctx)
}

func writePID(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create pid dir: %w", err)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0600)
}
