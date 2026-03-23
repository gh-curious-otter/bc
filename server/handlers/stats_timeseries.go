package handlers

import (
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/stats"
)

// parseTimeRange extracts from, to, and interval query parameters.
func parseTimeRange(r *http.Request) stats.TimeRange {
	now := time.Now()
	tr := stats.TimeRange{
		From:     now.Add(-1 * time.Hour),
		To:       now,
		Interval: "5m",
	}
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			tr.From = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			tr.To = t
		}
	}
	if v := r.URL.Query().Get("interval"); v != "" {
		tr.Interval = v
	}
	return tr
}

// systemMetricsTimeseries handles GET /api/stats/metrics.
func (h *StatsHandler) systemMetricsTimeseries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()
	tr := parseTimeRange(r)

	// Use TimescaleDB if available
	if h.statsStore != nil {
		metrics, err := h.statsStore.QuerySystem(ctx, tr)
		if err != nil {
			httpInternalError(w, "query system metrics", err)
			return
		}
		if metrics == nil {
			metrics = []stats.SystemMetric{}
		}
		writeJSON(w, http.StatusOK, metrics)
		return
	}

	// Fallback: return current system metrics as single-item array
	rootDir := "/"
	if h.ws != nil {
		rootDir = h.ws.RootDir
	}
	sm := getSystemMetrics(ctx, rootDir)
	hostname, _ := os.Hostname() //nolint:errcheck // best-effort

	writeJSON(w, http.StatusOK, []stats.SystemMetric{
		{
			Time:       time.Now(),
			Hostname:   hostname,
			CPUPercent: sm.CPUUsagePercent,
			MemBytes:   int64(sm.MemoryUsedBytes),
			MemPercent: sm.MemoryPercent,
			DiskBytes:  int64(sm.DiskUsedBytes),
			Goroutines: runtime.NumGoroutine(),
		},
	})
}

// agentMetricsTimeseries handles GET /api/stats/agents/{name}/metrics.
func (h *StatsHandler) agentMetricsTimeseries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	// Parse agent name: /api/stats/agents/{name}/metrics
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/stats/agents/"), "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		httpError(w, "agent name required", http.StatusBadRequest)
		return
	}
	agentName := parts[0]

	// Verify the path ends with /metrics
	if len(parts) < 2 || parts[1] != "metrics" {
		httpError(w, "not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	tr := parseTimeRange(r)

	// Use TimescaleDB if available
	if h.statsStore != nil {
		metrics, err := h.statsStore.QueryAgent(ctx, agentName, tr)
		if err != nil {
			httpInternalError(w, "query agent metrics", err)
			return
		}
		if metrics == nil {
			metrics = []stats.AgentMetric{}
		}
		writeJSON(w, http.StatusOK, metrics)
		return
	}

	// Fallback: return agent stats from the agent store
	if h.agents == nil {
		writeJSON(w, http.StatusOK, []stats.AgentMetric{})
		return
	}

	a, err := h.agents.Get(ctx, agentName)
	if err != nil {
		httpError(w, "agent not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, []stats.AgentMetric{
		{
			Time:      time.Now(),
			AgentName: a.Name,
			AgentID:   a.ID,
			Role:      string(a.Role),
			State:     string(a.State),
		},
	})
}

// tokenMetricsTimeseries handles GET /api/stats/tokens.
func (h *StatsHandler) tokenMetricsTimeseries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()
	tr := parseTimeRange(r)
	agentID := r.URL.Query().Get("agent_id")

	// Use TimescaleDB if available
	if h.statsStore != nil {
		var metrics []stats.TokenMetric
		var err error
		if agentID != "" {
			metrics, err = h.statsStore.QueryTokensByAgent(ctx, agentID, tr)
		} else {
			metrics, err = h.statsStore.QueryTokens(ctx, tr)
		}
		if err != nil {
			httpInternalError(w, "query token metrics", err)
			return
		}
		if metrics == nil {
			metrics = []stats.TokenMetric{}
		}
		writeJSON(w, http.StatusOK, metrics)
		return
	}

	// Fallback: return cost store data
	if h.costs == nil {
		writeJSON(w, http.StatusOK, []stats.TokenMetric{})
		return
	}

	summary, err := h.costs.WorkspaceSummary(ctx)
	if err != nil {
		httpInternalError(w, "cost summary", err)
		return
	}
	if summary == nil {
		writeJSON(w, http.StatusOK, []stats.TokenMetric{})
		return
	}

	writeJSON(w, http.StatusOK, []stats.TokenMetric{
		{
			Time:         time.Now(),
			InputTokens:  summary.InputTokens,
			OutputTokens: summary.OutputTokens,
			CostUSD:      summary.TotalCostUSD,
		},
	})
}

// channelMetricsTimeseries handles GET /api/stats/channels/metrics.
func (h *StatsHandler) channelMetricsTimeseries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()
	tr := parseTimeRange(r)

	// Use TimescaleDB if available
	if h.statsStore != nil {
		metrics, err := h.statsStore.QueryChannels(ctx, tr)
		if err != nil {
			httpInternalError(w, "query channel metrics", err)
			return
		}
		if metrics == nil {
			metrics = []stats.ChannelMetric{}
		}
		writeJSON(w, http.StatusOK, metrics)
		return
	}

	// Fallback: return current channel stats
	if h.channels == nil {
		writeJSON(w, http.StatusOK, []stats.ChannelMetric{})
		return
	}

	channels, err := h.channels.List(ctx)
	if err != nil {
		httpInternalError(w, "list channels", err)
		return
	}

	result := make([]stats.ChannelMetric, 0, len(channels))
	now := time.Now()
	for _, ch := range channels {
		result = append(result, stats.ChannelMetric{
			Time:         now,
			ChannelName:  ch.Name,
			MessagesSent: int64(ch.MessageCount),
			Participants: len(ch.Members),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

// daemonMetricsTimeseries handles GET /api/stats/daemons.
func (h *StatsHandler) daemonMetricsTimeseries(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()
	tr := parseTimeRange(r)

	// Use TimescaleDB if available
	if h.statsStore != nil {
		metrics, err := h.statsStore.QueryDaemons(ctx, tr)
		if err != nil {
			httpInternalError(w, "query daemon metrics", err)
			return
		}
		if metrics == nil {
			metrics = []stats.DaemonMetric{}
		}
		writeJSON(w, http.StatusOK, metrics)
		return
	}

	// No fallback for daemon metrics without statsStore
	writeJSON(w, http.StatusOK, []stats.DaemonMetric{})
}

// overview handles GET /api/stats/overview.
func (h *StatsHandler) overview(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	ctx := r.Context()
	now := time.Now()

	// System metrics
	rootDir := "/"
	if h.ws != nil {
		rootDir = h.ws.RootDir
	}
	sm := getSystemMetrics(ctx, rootDir)
	hostname, _ := os.Hostname() //nolint:errcheck // best-effort

	systemMetric := stats.SystemMetric{
		Time:       now,
		Hostname:   hostname,
		CPUPercent: sm.CPUUsagePercent,
		MemBytes:   int64(sm.MemoryUsedBytes),
		MemPercent: sm.MemoryPercent,
		DiskBytes:  int64(sm.DiskUsedBytes),
		Goroutines: runtime.NumGoroutine(),
	}

	// Agent metrics
	var agentMetrics []stats.AgentMetric
	if h.agents != nil {
		agents, err := h.agents.List(ctx, agent.ListOptions{})
		if err == nil {
			agentMetrics = make([]stats.AgentMetric, 0, len(agents))
			for _, a := range agents {
				agentMetrics = append(agentMetrics, stats.AgentMetric{
					Time:      now,
					AgentName: a.Name,
					AgentID:   a.ID,
					Role:      string(a.Role),
					State:     string(a.State),
				})
			}
		}
	}

	// Token summary
	type tokenSummary struct {
		TotalInputTokens  int64   `json:"total_input_tokens"`
		TotalOutputTokens int64   `json:"total_output_tokens"`
		TotalCostUSD      float64 `json:"total_cost_usd"`
	}
	var tokens tokenSummary
	if h.costs != nil {
		summary, err := h.costs.WorkspaceSummary(ctx)
		if err == nil && summary != nil {
			tokens.TotalInputTokens = summary.InputTokens
			tokens.TotalOutputTokens = summary.OutputTokens
			tokens.TotalCostUSD = summary.TotalCostUSD
		}
	}

	// Channel metrics
	var channelMetrics []stats.ChannelMetric
	if h.channels != nil {
		channels, err := h.channels.List(ctx)
		if err == nil {
			channelMetrics = make([]stats.ChannelMetric, 0, len(channels))
			for _, ch := range channels {
				channelMetrics = append(channelMetrics, stats.ChannelMetric{
					Time:         now,
					ChannelName:  ch.Name,
					MessagesSent: int64(ch.MessageCount),
					Participants: len(ch.Members),
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"system":   systemMetric,
		"agents":   agentMetrics,
		"tokens":   tokens,
		"channels": channelMetrics,
	})
}
