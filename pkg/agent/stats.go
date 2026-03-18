package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// AgentStatsRecord holds a single Docker stats sample for an agent.
type AgentStatsRecord struct {
	CollectedAt  time.Time `json:"collected_at"`
	AgentName    string    `json:"agent_name"`
	CPUPct       float64   `json:"cpu_pct"`
	MemUsedMB    float64   `json:"mem_used_mb"`
	MemLimitMB   float64   `json:"mem_limit_mb"`
	NetRxMB      float64   `json:"net_rx_mb"`
	NetTxMB      float64   `json:"net_tx_mb"`
	BlockReadMB  float64   `json:"block_read_mb"`
	BlockWriteMB float64   `json:"block_write_mb"`
}

// dockerStatsJSON is the raw JSON emitted by `docker stats --format json --no-stream`.
type dockerStatsJSON struct {
	Name        string `json:"Name"`
	CPUPerc     string `json:"CPUPerc"`     // "0.50%"
	MemUsage    string `json:"MemUsage"`    // "150MiB / 7.77GiB"
	NetIO       string `json:"NetIO"`       // "1.5kB / 500B"
	BlockIO     string `json:"BlockIO"`     // "10MB / 5MB"
}

// statsCollectInterval is how often Docker stats are polled.
const statsCollectInterval = 30 * time.Second

// StatsCollector collects Docker container stats and processes hook events
// for all running agents. It runs as a background goroutine in bcd.
type StatsCollector struct {
	mgr *Manager
}

// NewStatsCollector creates a StatsCollector backed by the given Manager.
func NewStatsCollector(mgr *Manager) *StatsCollector {
	return &StatsCollector{mgr: mgr}
}

// Run starts collecting stats until ctx is cancelled.
// It polls every statsCollectInterval.
func (c *StatsCollector) Run(ctx context.Context) {
	ticker := time.NewTicker(statsCollectInterval)
	defer ticker.Stop()

	// Do an immediate pass on startup.
	c.collect(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect(ctx)
		}
	}
}

// collect performs one poll: consume hook events then gather Docker stats.
func (c *StatsCollector) collect(ctx context.Context) {
	c.consumeAllHookEvents()
	c.collectDockerStats(ctx)
}

// consumeAllHookEvents drains pending hook event files for every agent and
// updates the manager's in-memory + persisted state.
func (c *StatsCollector) consumeAllHookEvents() {
	agents := c.mgr.ListAgents()
	for _, a := range agents {
		ev, ok := ConsumeHookEvent(c.mgr.stateDir, a.Name)
		if !ok {
			continue
		}
		targetState, known := StateForHookEvent(ev)
		if !known {
			continue
		}
		if err := c.mgr.UpdateAgentState(a.Name, targetState, ""); err != nil {
			// Transition may be invalid (e.g., already stopped) — log at debug.
			log.Debug("hook state update skipped", "agent", a.Name, "event", ev, "error", err)
		} else {
			log.Debug("hook state applied", "agent", a.Name, "event", ev, "state", targetState)
		}
	}
}

// collectDockerStats fetches stats for all docker-backend agents.
func (c *StatsCollector) collectDockerStats(ctx context.Context) {
	agents := c.mgr.ListAgents()
	for _, a := range agents {
		if a.RuntimeBackend != "docker" {
			continue
		}
		if a.State == StateStopped || a.State == StateError {
			continue
		}
		rec, err := fetchDockerStats(ctx, a.Name)
		if err != nil {
			log.Debug("docker stats unavailable", "agent", a.Name, "error", err)
			continue
		}
		if err := c.mgr.saveAgentStats(rec); err != nil {
			log.Warn("failed to save agent stats", "agent", a.Name, "error", err)
		}
	}
}

// fetchDockerStats runs `docker stats --no-stream --format json <name>` and
// parses the result into an AgentStatsRecord.
func fetchDockerStats(ctx context.Context, containerName string) (*AgentStatsRecord, error) {
	cmd := exec.CommandContext(ctx, "docker", "stats", "--no-stream", "--format", "json", containerName)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("docker stats: %w", err)
	}

	// docker stats outputs one JSON object per line; we want the first non-empty line.
	var raw dockerStatsJSON
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("parse docker stats json: %w", err)
		}
		break
	}

	rec := &AgentStatsRecord{
		AgentName:   containerName,
		CollectedAt: time.Now(),
	}
	rec.CPUPct = parseDockerPct(raw.CPUPerc)
	rec.MemUsedMB, rec.MemLimitMB = parseDockerMemory(raw.MemUsage)
	rec.NetRxMB, rec.NetTxMB = parseDockerIO(raw.NetIO)
	rec.BlockReadMB, rec.BlockWriteMB = parseDockerIO(raw.BlockIO)
	return rec, nil
}

// parseDockerPct converts "0.50%" → 0.50.
func parseDockerPct(s string) float64 {
	s = strings.TrimSuffix(strings.TrimSpace(s), "%")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseDockerMemory converts "150MiB / 7.77GiB" → (used, limit) in MB.
func parseDockerMemory(s string) (float64, float64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	return parseDockerBytes(strings.TrimSpace(parts[0])), parseDockerBytes(strings.TrimSpace(parts[1]))
}

// parseDockerIO converts "1.5MB / 500kB" → (rx, tx) in MB.
func parseDockerIO(s string) (float64, float64) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	return parseDockerBytes(strings.TrimSpace(parts[0])), parseDockerBytes(strings.TrimSpace(parts[1]))
}

// parseDockerBytes converts human-readable Docker byte strings to megabytes.
// Supports B, kB, MB, GB, MiB, GiB, KiB suffixes.
func parseDockerBytes(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "--" {
		return 0
	}
	multipliers := []struct {
		suffix string
		factor float64
	}{
		{"GiB", 1024},
		{"MiB", 1},
		{"KiB", 1.0 / 1024},
		{"GB", 1000},
		{"MB", 1},
		{"kB", 1.0 / 1000},
		{"KB", 1.0 / 1000},
		{"B", 1.0 / (1024 * 1024)},
	}
	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			num, err := strconv.ParseFloat(strings.TrimSuffix(s, m.suffix), 64)
			if err != nil {
				return 0
			}
			return num * m.factor
		}
	}
	// Fallback: assume bytes.
	num, _ := strconv.ParseFloat(s, 64)
	return num / (1024 * 1024)
}
