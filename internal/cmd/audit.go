package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit log export and compliance reporting",
	Long: `Export audit logs and generate compliance reports.

The audit command provides enterprise-grade audit logging and compliance
features for tracking agent activities, costs, and system events.

Subcommands:
  export    Export audit logs to JSON or CSV format
  report    Generate a compliance report summary

Examples:
  bc audit export --since 7d --format json > audit.json
  bc audit export --since 30d --format csv > audit.csv
  bc audit export --agent eng-01 > agent-audit.json
  bc audit report --since 30d`,
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit logs to JSON or CSV",
	Long: `Export workspace audit logs for compliance and analysis.

Exports all events (agent lifecycle, work assignments, messages, health checks)
to a structured format suitable for external analysis or compliance reporting.

Formats:
  json  JSON array of event objects (default)
  csv   CSV with columns: timestamp,type,agent,message,data

Examples:
  bc audit export --since 7d --format json > audit.json
  bc audit export --since 7d --format csv > audit.csv
  bc audit export --agent eng-01 > agent-audit.json
  bc audit export --type agent.report --since 24h`,
	RunE: runAuditExport,
}

var auditReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate compliance report",
	Long: `Generate a compliance report summarizing workspace activity.

The report includes:
  - Total events and event breakdown by type
  - Agent activity summary (events per agent)
  - Cost summary (total spend, by agent)
  - Error summary (failures, stuck agents)
  - Timeline overview

Examples:
  bc audit report --since 30d
  bc audit report --since 7d --json`,
	RunE: runAuditReport,
}

var (
	auditSince  string
	auditFormat string
	auditAgent  string
	auditType   string
)

func init() {
	// Export flags
	auditExportCmd.Flags().StringVar(&auditSince, "since", "", "Export events since duration ago (e.g. 7d, 30d, 24h)")
	auditExportCmd.Flags().StringVar(&auditFormat, "format", "json", "Output format: json, csv")
	auditExportCmd.Flags().StringVar(&auditAgent, "agent", "", "Filter by agent name")
	auditExportCmd.Flags().StringVar(&auditType, "type", "", "Filter by event type")

	// Report flags
	auditReportCmd.Flags().StringVar(&auditSince, "since", "30d", "Report period (e.g. 7d, 30d)")

	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditReportCmd)
	rootCmd.AddCommand(auditCmd)
}

// parseAuditDuration parses duration strings like "7d", "30d", "24h", "1h30m".
func parseAuditDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	// Handle day format (e.g., "7d", "30d")
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var n int
		if _, err := fmt.Sscanf(days, "%d", &n); err != nil {
			return 0, fmt.Errorf("invalid duration %q: use e.g. 7d, 30d, 24h", s)
		}
		return time.Duration(n) * 24 * time.Hour, nil
	}

	// Standard Go duration
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q: use e.g. 7d, 30d, 24h, 1h30m", s)
	}
	return d, nil
}

func runAuditExport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	log.Debug("audit export started", "since", auditSince, "format", auditFormat, "agent", auditAgent)

	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Read all events
	evts, err := eventLog.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Apply filters
	if auditSince != "" {
		duration, parseErr := parseAuditDuration(auditSince)
		if parseErr != nil {
			return parseErr
		}
		cutoff := time.Now().Add(-duration)
		filtered := evts[:0]
		for _, ev := range evts {
			if !ev.Timestamp.Before(cutoff) {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	if auditAgent != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if ev.Agent == auditAgent {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	if auditType != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if string(ev.Type) == auditType {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	log.Debug("events filtered", "count", len(evts))

	if len(evts) == 0 {
		ui.Warning("No events found matching criteria")
		return nil
	}

	// Export in requested format
	switch strings.ToLower(auditFormat) {
	case "json":
		return exportJSON(evts)
	case "csv":
		return exportCSV(evts)
	default:
		return fmt.Errorf("unknown format %q: use json or csv", auditFormat)
	}
}

// AuditEvent wraps events.Event with additional export fields.
type AuditEvent struct {
	Data      map[string]any `json:"data,omitempty"`
	Timestamp string         `json:"timestamp"`
	Type      string         `json:"type"`
	Agent     string         `json:"agent,omitempty"`
	Message   string         `json:"message,omitempty"`
}

func exportJSON(evts []events.Event) error {
	// Convert to export format
	export := make([]AuditEvent, len(evts))
	for i, ev := range evts {
		export[i] = AuditEvent{
			Timestamp: ev.Timestamp.Format(time.RFC3339),
			Type:      string(ev.Type),
			Agent:     ev.Agent,
			Message:   ev.Message,
			Data:      ev.Data,
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(export)
}

func exportCSV(evts []events.Event) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Header
	if err := w.Write([]string{"timestamp", "type", "agent", "message", "data"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Rows
	for _, ev := range evts {
		dataJSON := ""
		if ev.Data != nil {
			b, _ := json.Marshal(ev.Data)
			dataJSON = string(b)
		}

		row := []string{
			ev.Timestamp.Format(time.RFC3339),
			string(ev.Type),
			ev.Agent,
			ev.Message,
			dataJSON,
		}
		if err := w.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// AuditReport represents a compliance report.
//
//nolint:govet // field alignment acceptable for JSON serialization order
type AuditReport struct {
	AgentSummary []AgentAuditSummary `json:"agent_summary"`
	Generated    time.Time           `json:"generated"`
	PeriodStart  time.Time           `json:"period_start"`
	PeriodEnd    time.Time           `json:"period_end"`
	Period       string              `json:"period"`
	EventsByType map[string]int      `json:"events_by_type"`
	CostSummary  *CostAuditSummary   `json:"cost_summary,omitempty"`
	ErrorSummary *ErrorAuditSummary  `json:"error_summary"`
	TotalEvents  int                 `json:"total_events"`
}

// AgentAuditSummary summarizes activity for a single agent.
type AgentAuditSummary struct {
	AgentID      string  `json:"agent_id"`
	TotalEvents  int     `json:"total_events"`
	WorkStarted  int     `json:"work_started"`
	WorkComplete int     `json:"work_completed"`
	WorkFailed   int     `json:"work_failed"`
	Messages     int     `json:"messages"`
	CostUSD      float64 `json:"cost_usd,omitempty"`
}

// CostAuditSummary summarizes cost data.
type CostAuditSummary struct {
	CostByAgent  map[string]float64 `json:"cost_by_agent,omitempty"`
	CostByModel  map[string]float64 `json:"cost_by_model,omitempty"`
	TotalCostUSD float64            `json:"total_cost_usd"`
	TotalTokens  int64              `json:"total_tokens"`
	InputTokens  int64              `json:"input_tokens"`
	OutputTokens int64              `json:"output_tokens"`
	RecordCount  int64              `json:"record_count"`
}

// ErrorAuditSummary summarizes errors and issues.
type ErrorAuditSummary struct {
	FailuresByAgent map[string]int `json:"failures_by_agent,omitempty"`
	TotalFailures   int            `json:"total_failures"`
	HealthFailures  int            `json:"health_failures"`
	WorkFailures    int            `json:"work_failures"`
}

func runAuditReport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	log.Debug("audit report started", "since", auditSince)

	duration, err := parseAuditDuration(auditSince)
	if err != nil {
		return err
	}

	periodStart := time.Now().Add(-duration)
	periodEnd := time.Now()

	// Read events
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	allEvents, err := eventLog.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Filter to period
	var evts []events.Event
	for _, ev := range allEvents {
		if !ev.Timestamp.Before(periodStart) {
			evts = append(evts, ev)
		}
	}

	// Build report
	report := &AuditReport{
		Generated:    time.Now(),
		Period:       auditSince,
		PeriodStart:  periodStart,
		PeriodEnd:    periodEnd,
		TotalEvents:  len(evts),
		EventsByType: make(map[string]int),
	}

	// Aggregate by type
	agentStats := make(map[string]*AgentAuditSummary)
	errorSummary := &ErrorAuditSummary{
		FailuresByAgent: make(map[string]int),
	}

	for _, ev := range evts {
		report.EventsByType[string(ev.Type)]++

		// Ensure agent entry exists
		if ev.Agent != "" {
			if agentStats[ev.Agent] == nil {
				agentStats[ev.Agent] = &AgentAuditSummary{AgentID: ev.Agent}
			}
			agentStats[ev.Agent].TotalEvents++

			// Categorize events
			switch ev.Type {
			case events.WorkStarted:
				agentStats[ev.Agent].WorkStarted++
			case events.WorkCompleted:
				agentStats[ev.Agent].WorkComplete++
			case events.WorkFailed:
				agentStats[ev.Agent].WorkFailed++
				errorSummary.WorkFailures++
				errorSummary.TotalFailures++
				errorSummary.FailuresByAgent[ev.Agent]++
			case events.MessageSent:
				agentStats[ev.Agent].Messages++
			case events.HealthFailed:
				errorSummary.HealthFailures++
				errorSummary.TotalFailures++
				errorSummary.FailuresByAgent[ev.Agent]++
			}
		}
	}

	// Convert agent stats to slice and sort
	for _, stats := range agentStats {
		report.AgentSummary = append(report.AgentSummary, *stats)
	}
	sort.Slice(report.AgentSummary, func(i, j int) bool {
		return report.AgentSummary[i].TotalEvents > report.AgentSummary[j].TotalEvents
	})

	report.ErrorSummary = errorSummary

	// Load cost data
	costStore := cost.NewStore(ws.RootDir)
	if openErr := costStore.Open(); openErr == nil {
		defer func() { _ = costStore.Close() }()

		costSum, costErr := costStore.GetSummarySince(periodStart)
		if costErr == nil && costSum != nil {
			report.CostSummary = &CostAuditSummary{
				TotalCostUSD: costSum.TotalCostUSD,
				TotalTokens:  costSum.TotalTokens,
				InputTokens:  costSum.InputTokens,
				OutputTokens: costSum.OutputTokens,
				RecordCount:  costSum.RecordCount,
				CostByAgent:  make(map[string]float64),
				CostByModel:  make(map[string]float64),
			}

			// Get per-agent costs
			agentCosts, _ := costStore.GetAgentSummarySince(periodStart)
			for _, ac := range agentCosts {
				report.CostSummary.CostByAgent[ac.AgentID] = ac.TotalCostUSD
				// Update agent summary with cost
				for i := range report.AgentSummary {
					if report.AgentSummary[i].AgentID == ac.AgentID {
						report.AgentSummary[i].CostUSD = ac.TotalCostUSD
						break
					}
				}
			}
		}
	}

	// Output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Human-readable output
	printAuditReport(report)
	return nil
}

func printAuditReport(r *AuditReport) {
	fmt.Printf("Compliance Report\n")
	fmt.Printf("=================\n\n")
	fmt.Printf("Generated: %s\n", r.Generated.Format(time.RFC3339))
	fmt.Printf("Period:    %s (%s to %s)\n\n",
		r.Period,
		r.PeriodStart.Format("2006-01-02"),
		r.PeriodEnd.Format("2006-01-02"))

	// Event Summary
	fmt.Printf("Event Summary\n")
	fmt.Printf("-------------\n")
	fmt.Printf("Total Events: %d\n\n", r.TotalEvents)

	if len(r.EventsByType) > 0 {
		fmt.Printf("By Type:\n")
		// Sort types for consistent output
		types := make([]string, 0, len(r.EventsByType))
		for t := range r.EventsByType {
			types = append(types, t)
		}
		sort.Strings(types)
		for _, t := range types {
			fmt.Printf("  %-20s %d\n", t, r.EventsByType[t])
		}
		fmt.Println()
	}

	// Agent Summary
	if len(r.AgentSummary) > 0 {
		fmt.Printf("Agent Activity\n")
		fmt.Printf("--------------\n")
		for _, a := range r.AgentSummary {
			fmt.Printf("  %s: %d events", a.AgentID, a.TotalEvents)
			if a.WorkComplete > 0 {
				fmt.Printf(", %d completed", a.WorkComplete)
			}
			if a.WorkFailed > 0 {
				fmt.Printf(", %d failed", a.WorkFailed)
			}
			if a.CostUSD > 0 {
				fmt.Printf(", $%.2f", a.CostUSD)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	// Cost Summary
	if r.CostSummary != nil {
		fmt.Printf("Cost Summary\n")
		fmt.Printf("------------\n")
		fmt.Printf("Total Cost:    $%.2f\n", r.CostSummary.TotalCostUSD)
		fmt.Printf("Total Tokens:  %d\n", r.CostSummary.TotalTokens)
		fmt.Printf("  Input:       %d\n", r.CostSummary.InputTokens)
		fmt.Printf("  Output:      %d\n", r.CostSummary.OutputTokens)
		fmt.Printf("API Calls:     %d\n\n", r.CostSummary.RecordCount)
	}

	// Error Summary
	if r.ErrorSummary.TotalFailures > 0 {
		fmt.Printf("Error Summary\n")
		fmt.Printf("-------------\n")
		fmt.Printf("Total Failures:  %d\n", r.ErrorSummary.TotalFailures)
		if r.ErrorSummary.WorkFailures > 0 {
			fmt.Printf("  Work Failures: %d\n", r.ErrorSummary.WorkFailures)
		}
		if r.ErrorSummary.HealthFailures > 0 {
			fmt.Printf("  Health Fails:  %d\n", r.ErrorSummary.HealthFailures)
		}
		if len(r.ErrorSummary.FailuresByAgent) > 0 {
			fmt.Printf("\nBy Agent:\n")
			for agent, count := range r.ErrorSummary.FailuresByAgent {
				fmt.Printf("  %-15s %d\n", agent, count)
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("Error Summary: No failures recorded\n\n")
	}
}
