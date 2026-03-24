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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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

	agentMgr, agentErr := newAgentManager(ws)
	if agentErr != nil {
		return fmt.Errorf("agent manager: %w", agentErr)
	}
	if err := agentMgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}
	defer agentMgr.Close() //nolint:errcheck // best-effort
	agentSvc := bcagent.NewAgentService(agentMgr, hub, nil)

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

func newAgentManager(ws *bcworkspace.Workspace) (*bcagent.Manager, error) {
	var wsCfg bcworkspace.DockerRuntimeConfig
	if ws.Config != nil {
		wsCfg = ws.Config.Runtime.Docker
	}
	dockerCfg := bccontainer.ConfigFromWorkspace(wsCfg)
	be, err := bccontainer.NewBackend(dockerCfg, "bc-", ws.RootDir, provider.DefaultRegistry)
	if err != nil {
		log.Warn("Docker not available — agents will use tmux runtime only", "error", err)
		return bcagent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir), nil
	}
	return bcagent.NewWorkspaceManagerWithRuntime(ws.AgentsDir(), ws.RootDir, be, "docker"), nil
}

func writePID(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("create pid dir: %w", err)
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0600)
}

// dockerStatsEntry represents one line of `docker stats --no-stream --format '{{json .}}'`.
type dockerStatsEntry struct {
	Container string `json:"Container"` // container ID
	Name      string `json:"Name"`      // container name
	CPUPerc   string `json:"CPUPerc"`
	MemUsage  string `json:"MemUsage"`
	MemPerc   string `json:"MemPerc"`
	NetIO     string `json:"NetIO"`
	BlockIO   string `json:"BlockIO"`
}

// runStatsCollector periodically samples system and agent metrics into TimescaleDB.
// It shells out to `docker stats --no-stream` every 30s, classifies containers as
// system (bc-sql, bc-stats, *-daemon) or agent (bc-*-agent-*), and records resource
// usage. Channel metrics come from the channel service.
func runStatsCollector(ctx context.Context, ss *bcstats.Store, agents *bcagent.AgentService, channels *bcchannel.ChannelService, ws *bcworkspace.Workspace) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Build an agent lookup by name for enriching agent metrics.
	agentLookup := func() map[string]*bcagent.Agent {
		if agents == nil {
			return nil
		}
		list, err := agents.List(ctx, bcagent.ListOptions{})
		if err != nil {
			log.Debug("stats: agent list failed", "error", err)
			return nil
		}
		m := make(map[string]*bcagent.Agent, len(list))
		for _, a := range list {
			m[a.Name] = a
		}
		return m
	}

	for {
		select {
		case <-ticker.C:
			now := time.Now()

			// ── docker stats ────────────────────────────────────────
			entries := collectDockerStats(ctx)
			agentsByName := agentLookup()

			for _, e := range entries {
				cpu := parsePercent(e.CPUPerc)
				memUsed, memLimit := parseMemUsage(e.MemUsage)
				memPct := parsePercent(e.MemPerc)
				netRx, netTx := parseIOPair(e.NetIO)
				diskR, diskW := parseIOPair(e.BlockIO)

				name := e.Name

				switch {
				case isSystemContainer(name):
					if err := ss.RecordSystem(ctx, bcstats.SystemMetric{
						Time:           now,
						SystemName:     name,
						CPUPercent:     cpu,
						MemUsedBytes:   memUsed,
						MemLimitBytes:  memLimit,
						MemPercent:     memPct,
						NetRxBytes:     netRx,
						NetTxBytes:     netTx,
						DiskReadBytes:  diskR,
						DiskWriteBytes: diskW,
					}); err != nil {
						log.Debug("stats: record system metric", "name", name, "error", err)
					}

				case isAgentContainer(name):
					agentName := extractAgentName(name)
					var role, tool, state string
					if a, ok := agentsByName[agentName]; ok {
						role = string(a.Role)
						tool = a.Tool
						state = string(a.State)
					}
					if err := ss.RecordAgent(ctx, bcstats.AgentMetric{
						Time:           now,
						AgentName:      agentName,
						Role:           role,
						Tool:           tool,
						Runtime:        "docker",
						State:          state,
						CPUPercent:     cpu,
						MemUsedBytes:   memUsed,
						MemLimitBytes:  memLimit,
						MemPercent:     memPct,
						NetRxBytes:     netRx,
						NetTxBytes:     netTx,
						DiskReadBytes:  diskR,
						DiskWriteBytes: diskW,
					}); err != nil {
						log.Debug("stats: record agent metric", "agent", agentName, "error", err)
					}
				}
			}

			// ── channel metrics ─────────────────────────────────────
			if channels != nil {
				chList, err := channels.List(ctx)
				if err != nil {
					log.Debug("stats: channel list failed", "error", err)
				} else {
					for _, ch := range chList {
						if err := ss.RecordChannel(ctx, bcstats.ChannelMetric{
							Time:         now,
							ChannelName:  ch.Name,
							MessageCount: int64(ch.MessageCount),
							MemberCount:  ch.MemberCount,
						}); err != nil {
							log.Debug("stats: record channel metric", "channel", ch.Name, "error", err)
						}
					}
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// collectDockerStats runs `docker stats --no-stream` and returns parsed entries.
func collectDockerStats(ctx context.Context) []dockerStatsEntry {
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "{{json .}}")
	out, err := cmd.Output()
	if err != nil {
		log.Debug("stats: docker stats failed", "error", err)
		return nil
	}

	var entries []dockerStatsEntry
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e dockerStatsEntry
		if err := json.Unmarshal(line, &e); err != nil {
			log.Debug("stats: parse docker stats line", "error", err)
			continue
		}
		entries = append(entries, e)
	}
	return entries
}

// isSystemContainer returns true for bc-sql, bc-stats, or *-daemon containers.
func isSystemContainer(name string) bool {
	if name == "bc-sql" || name == "bc-stats" {
		return true
	}
	return strings.Contains(name, "-daemon")
}

// isAgentContainer returns true for bc-<hash>-<name> containers that are NOT system containers.
func isAgentContainer(name string) bool {
	if !strings.HasPrefix(name, "bc-") {
		return false
	}
	return !isSystemContainer(name)
}

// extractAgentName extracts the agent name from a container name like bc-<hash>-<name>.
// The hash is always 6 hex chars, so prefix is "bc-XXXXXX-" (10 chars).
func extractAgentName(containerName string) string {
	// bc-a08b6d-agent-01 → agent-01
	if len(containerName) > 10 && strings.HasPrefix(containerName, "bc-") {
		return containerName[10:]
	}
	return containerName
}

// parsePercent parses a percentage string like "5.00%" into a float64.
func parsePercent(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, "%")
	v, _ := strconv.ParseFloat(s, 64) //nolint:errcheck // returns 0 on failure
	return v
}

// parseMemUsage splits "100MiB / 2GiB" into (used bytes, limit bytes).
func parseMemUsage(s string) (int64, int64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	return parseBytes(strings.TrimSpace(parts[0])), parseBytes(strings.TrimSpace(parts[1]))
}

// parseIOPair splits an IO string like "1.5kB / 2.3kB" into (in, out) bytes.
func parseIOPair(s string) (int64, int64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	return parseBytes(strings.TrimSpace(parts[0])), parseBytes(strings.TrimSpace(parts[1]))
}

// parseBytes converts a human-readable byte string (e.g. "100MiB", "1.5kB", "0B")
// into an int64 byte count.
func parseBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Find where the numeric part ends and the unit begins.
	unitIdx := len(s)
	for i, c := range s {
		if c != '.' && (c < '0' || c > '9') {
			unitIdx = i
			break
		}
	}

	numStr := s[:unitIdx]
	unit := strings.ToLower(s[unitIdx:])

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	switch unit {
	case "b":
		return int64(val)
	case "kb":
		return int64(val * 1000)
	case "mb":
		return int64(val * 1000 * 1000)
	case "gb":
		return int64(val * 1000 * 1000 * 1000)
	case "tb":
		return int64(val * 1000 * 1000 * 1000 * 1000)
	case "kib":
		return int64(val * 1024)
	case "mib":
		return int64(val * 1024 * 1024)
	case "gib":
		return int64(val * 1024 * 1024 * 1024)
	case "tib":
		return int64(val * 1024 * 1024 * 1024 * 1024)
	default:
		return int64(val)
	}
}
