package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

// Issue #1648: Extracted from agent.go for better code organization
// Health monitoring and stuck agent detection commands

// agentHealthCmd displays health status of agents
var agentHealthCmd = &cobra.Command{
	Use:   "health [agent]",
	Short: "Check agent health status",
	Long: `Check health status of agents including tmux session and state freshness.

Examples:
  bc agent health              # Check all agents
  bc agent health eng-01       # Check specific agent
  bc agent health --json       # Output as JSON
  bc agent health --detect-stuck --alert eng  # Detect stuck and alert`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentHealth,
}

// Health command flags
var (
	agentHealthJSON      bool
	agentHealthTimeout   string
	agentHealthDetect    bool
	agentHealthWorkTmout string
	agentHealthMaxFail   int
	agentHealthAlert     string
)

func initAgentHealthFlags() {
	agentHealthCmd.Flags().BoolVar(&agentHealthJSON, "json", false, "Output as JSON")
	agentHealthCmd.Flags().StringVar(&agentHealthTimeout, "timeout", "60s", "Stale state threshold (e.g., 30s, 2m)")
	agentHealthCmd.Flags().BoolVar(&agentHealthDetect, "detect-stuck", false, "Enable stuck detection analysis")
	agentHealthCmd.Flags().StringVar(&agentHealthWorkTmout, "work-timeout", "30m", "Work timeout for stuck detection (e.g., 30m, 1h)")
	agentHealthCmd.Flags().IntVar(&agentHealthMaxFail, "max-failures", 3, "Max consecutive failures before considered stuck")
	agentHealthCmd.Flags().StringVar(&agentHealthAlert, "alert", "", "Send alert to channel when stuck agents detected (requires --detect-stuck)")
}

// AgentHealth represents the health status of an agent.
type AgentHealth struct {
	Name          string `json:"name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	LastUpdated   string `json:"last_updated"`
	StaleDuration string `json:"stale_duration,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	StuckReason   string `json:"stuck_reason,omitempty"`
	StuckDetails  string `json:"stuck_details,omitempty"`
	TmuxAlive     bool   `json:"tmux_alive"`
	StateFresh    bool   `json:"state_fresh"`
	IsStuck       bool   `json:"is_stuck,omitempty"`
}

func runAgentHealth(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Parse timeout duration
	timeout, parseErr := time.ParseDuration(agentHealthTimeout)
	if parseErr != nil {
		return fmt.Errorf("invalid timeout format: %w", parseErr)
	}

	// Parse work timeout for stuck detection
	workTimeout, workParseErr := time.ParseDuration(agentHealthWorkTmout)
	if workParseErr != nil {
		return fmt.Errorf("invalid work-timeout format: %w", workParseErr)
	}

	// Validate --alert flag requires --detect-stuck
	if agentHealthAlert != "" && !agentHealthDetect {
		return fmt.Errorf("--alert requires --detect-stuck to be enabled")
	}

	ctx := cmd.Context()
	mgr := newAgentManager(ws)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	if refreshErr := mgr.RefreshState(); refreshErr != nil {
		log.Warn("failed to refresh agent state", "error", refreshErr)
	}

	// Get agents to check
	var agents []*agent.Agent
	if len(args) > 0 {
		// Check specific agent
		a := mgr.GetAgent(args[0])
		if a == nil {
			return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", args[0])
		}
		agents = []*agent.Agent{a}
	} else {
		// Check all agents
		agents = mgr.ListAgents()
	}

	if len(agents) == 0 {
		fmt.Println("No agents found")
		return nil
	}

	// Prepare stuck detection if enabled
	var eventLog events.EventStore
	var stuckConfig events.StuckConfig
	if agentHealthDetect {
		eventLog = openEventLog(ws)
		if eventLog != nil {
			defer func() { _ = eventLog.Close() }()
		}
		stuckConfig = events.StuckConfig{
			ActivityTimeout: timeout,
			WorkTimeout:     workTimeout,
			MaxFailures:     agentHealthMaxFail,
		}
	}

	// Compute health for each agent
	healthResults := make([]AgentHealth, 0, len(agents))
	for _, a := range agents {
		health := computeAgentHealth(ctx, a, mgr, timeout)

		// Add stuck detection if enabled
		if agentHealthDetect && eventLog != nil {
			agentEvents, readErr := eventLog.ReadByAgent(a.Name)
			if readErr != nil {
				log.Warn("failed to read agent events", "agent", a.Name, "error", readErr)
			} else {
				stuck := events.DetectStuck(agentEvents, stuckConfig)
				if stuck.IsStuck {
					health.IsStuck = true
					health.StuckReason = string(stuck.Reason)
					health.StuckDetails = stuck.Details
					// Override status if stuck
					if health.Status == "healthy" || health.Status == "degraded" {
						health.Status = "stuck"
						health.ErrorMessage = stuck.Details
					}
				}
			}
		}

		healthResults = append(healthResults, health)
	}

	// Send alert to channel if --alert is set and there are stuck agents
	if agentHealthAlert != "" {
		if alertErr := sendStuckAlert(ws.RootDir, agentHealthAlert, healthResults, mgr); alertErr != nil {
			log.Warn("failed to send stuck alert", "error", alertErr)
		}
	}

	// Output
	if agentHealthJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(healthResults)
	}

	// Table output
	fmt.Printf("%-15s %-12s %-10s %-8s %-8s %s\n", "AGENT", "ROLE", "STATUS", "TMUX", "FRESH", "LAST UPDATED")
	fmt.Println(strings.Repeat("-", 75))

	for _, h := range healthResults {
		tmuxStr := "✗"
		if h.TmuxAlive {
			tmuxStr = "✓"
		}
		freshStr := "✗"
		if h.StateFresh {
			freshStr = "✓"
		}

		statusColor := h.Status
		switch h.Status {
		case "healthy":
			statusColor = ui.GreenText(h.Status)
		case "degraded":
			statusColor = ui.YellowText(h.Status)
		case "unhealthy":
			statusColor = ui.RedText(h.Status)
		case "stuck":
			statusColor = ui.MagentaText(h.Status)
		}

		fmt.Printf("%-15s %-12s %-10s %-8s %-8s %s\n",
			h.Name,
			h.Role,
			statusColor,
			tmuxStr,
			freshStr,
			h.LastUpdated,
		)

		if h.ErrorMessage != "" {
			fmt.Printf("  └─ %s\n", h.ErrorMessage)
		}
	}

	// Summary
	var healthy, degraded, unhealthy, stuck int
	for _, h := range healthResults {
		switch h.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "unhealthy":
			unhealthy++
		case "stuck":
			stuck++
		}
	}
	if agentHealthDetect {
		fmt.Printf("\nSummary: %d healthy, %d degraded, %d unhealthy, %d stuck (threshold: %s, work-timeout: %s)\n",
			healthy, degraded, unhealthy, stuck, timeout, agentHealthWorkTmout)
	} else {
		fmt.Printf("\nSummary: %d healthy, %d degraded, %d unhealthy (threshold: %s)\n",
			healthy, degraded, unhealthy, timeout)
	}

	return nil
}

func computeAgentHealth(ctx context.Context, a *agent.Agent, mgr *agent.Manager, timeout time.Duration) AgentHealth {
	health := AgentHealth{
		Name:        a.Name,
		Role:        string(a.Role),
		LastUpdated: a.UpdatedAt.Format(time.RFC3339),
	}

	// Check tmux session
	health.TmuxAlive = mgr.Runtime().HasSession(ctx, a.Name)

	// Check state freshness
	staleDuration := time.Since(a.UpdatedAt)
	health.StateFresh = staleDuration < timeout
	if !health.StateFresh {
		health.StaleDuration = staleDuration.Round(time.Second).String()
	}

	// Determine overall status
	switch {
	case a.State == agent.StateStopped:
		health.Status = "unhealthy"
		health.ErrorMessage = "agent stopped"
	case a.State == agent.StateError:
		health.Status = "unhealthy"
		health.ErrorMessage = "agent in error state"
	case !health.TmuxAlive:
		health.Status = "unhealthy"
		health.ErrorMessage = "tmux session not found"
	case !health.StateFresh:
		health.Status = "degraded"
		health.ErrorMessage = fmt.Sprintf("state stale (%s since last update)", health.StaleDuration)
	default:
		health.Status = "healthy"
	}

	return health
}

// sendStuckAlert sends an alert to the specified channel when stuck agents are detected.
func sendStuckAlert(rootDir, channelName string, healthResults []AgentHealth, mgr *agent.Manager) error {
	// Collect stuck agents
	var stuckAgents []AgentHealth
	for _, h := range healthResults {
		if h.IsStuck || h.Status == "stuck" {
			stuckAgents = append(stuckAgents, h)
		}
	}

	if len(stuckAgents) == 0 {
		// No stuck agents, no alert needed
		return nil
	}

	// Build alert message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚠️ ALERT: %d stuck agent(s) detected\n", len(stuckAgents)))
	for _, h := range stuckAgents {
		reason := h.StuckReason
		if reason == "" {
			reason = "unknown"
		}
		details := h.StuckDetails
		if details == "" {
			details = h.ErrorMessage
		}
		sb.WriteString(fmt.Sprintf("  • %s (%s): %s - %s\n", h.Name, h.Role, reason, details))
	}

	message := sb.String()

	// Load channel store
	store, err := channel.OpenStore(rootDir)
	if err != nil {
		return fmt.Errorf("failed to open channel store: %w", err)
	}
	defer func() { _ = store.Close() }()

	if loadErr := store.Load(); loadErr != nil {
		return fmt.Errorf("failed to load channel store: %w", loadErr)
	}

	// Get channel members
	members, membersErr := store.GetMembers(channelName)
	if membersErr != nil {
		return fmt.Errorf("channel %q not found: %w", channelName, membersErr)
	}

	if len(members) == 0 {
		fmt.Printf("Alert: channel %q has no members, alert not sent\n", channelName)
		return nil
	}

	// Record in channel history
	if err := store.AddHistory(channelName, "bc-health", message); err != nil {
		log.Warn("failed to record alert history", "error", err)
	}
	if err := store.Save(); err != nil {
		log.Warn("failed to save alert history", "error", err)
	}

	// Send to all members
	sent := 0
	for _, member := range members {
		a := mgr.GetAgent(member)
		if a == nil || a.State == agent.StateStopped {
			continue
		}
		formattedMsg := fmt.Sprintf("[#%s] bc-health: %s", channelName, message)
		if sendErr := mgr.SendToAgent(member, formattedMsg); sendErr != nil {
			log.Warn("failed to send alert to agent", "agent", member, "error", sendErr)
			continue
		}
		sent++
	}

	fmt.Printf("Alert sent to %d member(s) in channel %q\n", sent, channelName)
	return nil
}
