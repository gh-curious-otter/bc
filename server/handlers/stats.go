package handlers

import (
	"math"
	"net/http"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/tool"
	"github.com/rpuneet/bc/pkg/workspace"
)

// serverStartTime is used to compute uptime.
var serverStartTime = time.Now() //nolint:gochecknoglobals // intentional: tracks server start

// StatsHandler handles /api/stats routes.
type StatsHandler struct {
	agents   *agent.AgentService
	channels *channel.ChannelService
	costs    *cost.Store
	tools    *tool.Store
	ws       *workspace.Workspace
}

// NewStatsHandler creates a StatsHandler.
func NewStatsHandler(
	agents *agent.AgentService,
	channels *channel.ChannelService,
	costs *cost.Store,
	tools *tool.Store,
	ws *workspace.Workspace,
) *StatsHandler {
	return &StatsHandler{
		agents:   agents,
		channels: channels,
		costs:    costs,
		tools:    tools,
		ws:       ws,
	}
}

// Register mounts stats routes on mux.
func (h *StatsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/api/stats/system", h.system)
	mux.HandleFunc("/api/stats/summary", h.summary)
}

func (h *StatsHandler) system(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	hostname, _ := os.Hostname() //nolint:errcheck // best-effort

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// System memory via Sysinfo
	var sysInfo syscall.Sysinfo_t
	var memTotal, memUsed uint64
	var memPercent float64
	if err := syscall.Sysinfo(&sysInfo); err == nil {
		memTotal = sysInfo.Totalram * uint64(sysInfo.Unit)
		freeRAM := sysInfo.Freeram * uint64(sysInfo.Unit)
		memUsed = memTotal - freeRAM
		if memTotal > 0 {
			memPercent = roundTo(float64(memUsed)/float64(memTotal)*100, 1)
		}
	}

	// Disk usage via Statfs on workspace root
	var diskTotal, diskUsed uint64
	var diskPercent float64
	rootDir := "/"
	if h.ws != nil {
		rootDir = h.ws.RootDir
	}
	var statfs syscall.Statfs_t
	if err := syscall.Statfs(rootDir, &statfs); err == nil && statfs.Bsize > 0 {
		bsize := uint64(statfs.Bsize) //nolint:gosec // Bsize is always positive from the kernel
		diskTotal = statfs.Blocks * bsize
		diskFree := statfs.Bavail * bsize
		diskUsed = diskTotal - diskFree
		if diskTotal > 0 {
			diskPercent = roundTo(float64(diskUsed)/float64(diskTotal)*100, 1)
		}
	}

	// CPU usage approximation: ratio of Go's Sys memory to total (not ideal,
	// but avoids cgo/proc parsing). We report 0 when unavailable.
	cpuPercent := 0.0

	writeJSON(w, http.StatusOK, map[string]any{
		"hostname":             hostname,
		"os":                   runtime.GOOS,
		"arch":                 runtime.GOARCH,
		"cpus":                 runtime.NumCPU(),
		"cpu_usage_percent":    cpuPercent,
		"memory_total_bytes":   memTotal,
		"memory_used_bytes":    memUsed,
		"memory_usage_percent": memPercent,
		"disk_total_bytes":     diskTotal,
		"disk_used_bytes":      diskUsed,
		"disk_usage_percent":   diskPercent,
		"go_version":           runtime.Version(),
		"uptime_seconds":       int64(time.Since(serverStartTime).Seconds()),
		"goroutines":           runtime.NumGoroutine(),
	})
}

func (h *StatsHandler) summary(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()

	var agentsTotal, agentsRunning, agentsStopped int
	if h.agents != nil {
		agents, err := h.agents.List(ctx, agent.ListOptions{})
		if err != nil {
			httpError(w, "list agents: "+err.Error(), http.StatusInternalServerError)
			return
		}
		agentsTotal = len(agents)
		for _, a := range agents {
			if a.State == agent.StateStopped || a.State == agent.StateError {
				agentsStopped++
			} else {
				agentsRunning++
			}
		}
	}

	var channelsTotal, messagesTotal int
	if h.channels != nil {
		channels, err := h.channels.List(ctx)
		if err != nil {
			httpError(w, "list channels: "+err.Error(), http.StatusInternalServerError)
			return
		}
		channelsTotal = len(channels)
		for _, ch := range channels {
			messagesTotal += ch.MessageCount
		}
	}

	var totalCostUSD float64
	if h.costs != nil {
		summary, err := h.costs.WorkspaceSummary(ctx)
		if err != nil {
			httpError(w, "cost summary: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if summary != nil {
			totalCostUSD = summary.TotalCostUSD
		}
	}

	var rolesTotal int
	if h.ws != nil {
		roles, err := h.ws.RoleManager.LoadAllRoles()
		if err == nil {
			rolesTotal = len(roles)
		}
	}

	var toolsTotal int
	if h.tools != nil {
		tools, err := h.tools.List(ctx)
		if err == nil {
			toolsTotal = len(tools)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"agents_total":   agentsTotal,
		"agents_running": agentsRunning,
		"agents_stopped": agentsStopped,
		"channels_total": channelsTotal,
		"messages_total": messagesTotal,
		"total_cost_usd": totalCostUSD,
		"roles_total":    rolesTotal,
		"tools_total":    toolsTotal,
		"uptime_seconds": int64(time.Since(serverStartTime).Seconds()),
	})
}

// roundTo rounds f to the given number of decimal places.
func roundTo(f float64, places int) float64 {
	shift := math.Pow(10, float64(places))
	return math.Round(f*shift) / shift
}
