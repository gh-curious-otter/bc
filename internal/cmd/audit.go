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

	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit log export and compliance reporting",
	Long: `Export audit logs and generate compliance reports for bc workspaces.

The audit command provides enterprise-grade logging capabilities:
- Export event logs in JSON or CSV format
- Generate compliance reports with activity summaries
- Filter by date range, agent, and event type

Examples:
  bc audit export --since 7d                    # Export last 7 days as JSON
  bc audit export --since 7d --format csv       # Export as CSV
  bc audit export --agent eng-01                # Export specific agent's events
  bc audit report --since 30d                   # Generate 30-day compliance report

See Also:
  bc logs      View event log interactively
  bc cost      Cost tracking and budgets`,
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit log in JSON or CSV format",
	Long: `Export the event audit log for compliance and analysis.

Supported Formats:
  json    JSON array (default)
  csv     Comma-separated values with headers

Examples:
  bc audit export --since 7d > audit.json
  bc audit export --since 7d --format csv > audit.csv
  bc audit export --agent eng-01 --since 24h
  bc audit export --type agent.report --format csv`,
	RunE: runAuditExport,
}

var auditReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate compliance report",
	Long: `Generate a compliance report summarizing agent activity.

The report includes:
- Total events by type
- Activity by agent
- Time range summary
- Error and failure counts

Examples:
  bc audit report --since 30d
  bc audit report --since 7d --json`,
	RunE: runAuditReport,
}

var (
	auditExportFormat string
	auditExportSince  string
	auditExportAgent  string
	auditExportType   string
	auditReportSince  string
)

func init() {
	// Export subcommand flags
	auditExportCmd.Flags().StringVar(&auditExportFormat, "format", "json", "Output format: json, csv")
	auditExportCmd.Flags().StringVar(&auditExportSince, "since", "", "Export events since duration (e.g., 7d, 24h, 30d)")
	auditExportCmd.Flags().StringVar(&auditExportAgent, "agent", "", "Filter by agent name")
	auditExportCmd.Flags().StringVar(&auditExportType, "type", "", "Filter by event type")

	// Report subcommand flags
	auditReportCmd.Flags().StringVar(&auditReportSince, "since", "30d", "Report period (e.g., 7d, 30d)")

	// Register commands
	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditReportCmd)
	rootCmd.AddCommand(auditCmd)
}

// parseAuditDuration parses duration strings like "7d", "30d", "24h"
func parseAuditDuration(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}

	// Handle days specially (not supported by time.ParseDuration)
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		var days int
		if _, err := fmt.Sscanf(daysStr, "%d", &days); err != nil {
			return time.Time{}, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		return time.Now().AddDate(0, 0, -days), nil
	}

	// Standard duration parsing for h, m, s
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration %q: %w (use e.g., 7d, 24h, 30m)", s, err)
	}
	return time.Now().Add(-d), nil
}

func runAuditExport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	log.Debug("audit export started", "format", auditExportFormat, "since", auditExportSince, "agent", auditExportAgent)

	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Read all events
	evts, err := eventLog.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Filter by time range
	if auditExportSince != "" {
		cutoff, parseErr := parseAuditDuration(auditExportSince)
		if parseErr != nil {
			return parseErr
		}
		filtered := evts[:0]
		for _, ev := range evts {
			if !ev.Timestamp.Before(cutoff) {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	// Filter by agent
	if auditExportAgent != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if ev.Agent == auditExportAgent {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	// Filter by event type
	if auditExportType != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if string(ev.Type) == auditExportType {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	log.Debug("events filtered for export", "count", len(evts))

	switch auditExportFormat {
	case "json":
		return exportJSON(evts)
	case "csv":
		return exportCSV(evts)
	default:
		return fmt.Errorf("unsupported format: %s (use json or csv)", auditExportFormat)
	}
}

func exportJSON(evts []events.Event) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(evts)
}

func exportCSV(evts []events.Event) error {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	// Write header
	header := []string{"timestamp", "type", "agent", "message", "data"}
	if err := w.Write(header); err != nil {
		return err
	}

	// Write rows
	for _, ev := range evts {
		dataJSON := ""
		if ev.Data != nil {
			if b, err := json.Marshal(ev.Data); err == nil {
				dataJSON = string(b)
			}
		}
		row := []string{
			ev.Timestamp.Format(time.RFC3339),
			string(ev.Type),
			ev.Agent,
			ev.Message,
			dataJSON,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return w.Error()
}

// AuditReport contains compliance report data
type AuditReport struct {
	AgentSummary  []AgentActivitySummary `json:"agent_summary"`
	EventsByType  map[string]int         `json:"events_by_type"`
	EventsByAgent map[string]int         `json:"events_by_agent"`
	GeneratedAt   time.Time              `json:"generated_at"`
	PeriodStart   time.Time              `json:"period_start"`
	PeriodEnd     time.Time              `json:"period_end"`
	ErrorSummary  ErrorSummary           `json:"error_summary"`
	TotalEvents   int                    `json:"total_events"`
}

// AgentActivitySummary summarizes an agent's activity
type AgentActivitySummary struct {
	FirstSeen     time.Time `json:"first_seen"`
	LastSeen      time.Time `json:"last_seen"`
	Agent         string    `json:"agent"`
	TotalEvents   int       `json:"total_events"`
	WorkAssigned  int       `json:"work_assigned"`
	WorkCompleted int       `json:"work_completed"`
	WorkFailed    int       `json:"work_failed"`
	Reports       int       `json:"reports"`
}

// ErrorSummary summarizes errors and failures
type ErrorSummary struct {
	FailuresByAgent map[string]int `json:"failures_by_agent"`
	TotalFailures   int            `json:"total_failures"`
	HealthFailures  int            `json:"health_failures"`
	WorkFailures    int            `json:"work_failures"`
}

func runAuditReport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	log.Debug("audit report started", "since", auditReportSince)

	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Read all events
	evts, err := eventLog.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Filter by time range
	cutoff, parseErr := parseAuditDuration(auditReportSince)
	if parseErr != nil {
		return parseErr
	}

	filtered := evts[:0]
	for _, ev := range evts {
		if !ev.Timestamp.Before(cutoff) {
			filtered = append(filtered, ev)
		}
	}
	evts = filtered

	log.Debug("events filtered for report", "count", len(evts))

	// Build report
	report := buildAuditReport(evts, cutoff)

	// Output
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	// Human-readable output
	printAuditReport(report)
	return nil
}

func buildAuditReport(evts []events.Event, periodStart time.Time) AuditReport {
	report := AuditReport{
		GeneratedAt:   time.Now(),
		PeriodStart:   periodStart,
		PeriodEnd:     time.Now(),
		TotalEvents:   len(evts),
		EventsByType:  make(map[string]int),
		EventsByAgent: make(map[string]int),
		ErrorSummary: ErrorSummary{
			FailuresByAgent: make(map[string]int),
		},
	}

	// Agent activity tracking
	agentData := make(map[string]*AgentActivitySummary)

	for _, ev := range evts {
		// Count by type
		report.EventsByType[string(ev.Type)]++

		// Count by agent
		if ev.Agent != "" {
			report.EventsByAgent[ev.Agent]++

			// Track agent activity
			if agentData[ev.Agent] == nil {
				agentData[ev.Agent] = &AgentActivitySummary{
					Agent:     ev.Agent,
					FirstSeen: ev.Timestamp,
					LastSeen:  ev.Timestamp,
				}
			}
			ad := agentData[ev.Agent]
			ad.TotalEvents++
			if ev.Timestamp.Before(ad.FirstSeen) {
				ad.FirstSeen = ev.Timestamp
			}
			if ev.Timestamp.After(ad.LastSeen) {
				ad.LastSeen = ev.Timestamp
			}

			// Track specific event types
			switch ev.Type {
			case events.WorkAssigned:
				ad.WorkAssigned++
			case events.WorkCompleted:
				ad.WorkCompleted++
			case events.WorkFailed:
				ad.WorkFailed++
				report.ErrorSummary.WorkFailures++
				report.ErrorSummary.TotalFailures++
				report.ErrorSummary.FailuresByAgent[ev.Agent]++
			case events.AgentReport:
				ad.Reports++
			case events.HealthFailed:
				report.ErrorSummary.HealthFailures++
				report.ErrorSummary.TotalFailures++
				report.ErrorSummary.FailuresByAgent[ev.Agent]++
			}
		}
	}

	// Convert agent data to sorted slice
	for _, ad := range agentData {
		report.AgentSummary = append(report.AgentSummary, *ad)
	}
	sort.Slice(report.AgentSummary, func(i, j int) bool {
		return report.AgentSummary[i].TotalEvents > report.AgentSummary[j].TotalEvents
	})

	return report
}

func printAuditReport(report AuditReport) {
	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Audit Compliance Report"))
	fmt.Println("  " + strings.Repeat("═", 50))
	fmt.Println()

	// Period info
	fmt.Printf("  %s\n", ui.GrayText("Report Period"))
	fmt.Printf("    From: %s\n", report.PeriodStart.Format("2006-01-02 15:04:05"))
	fmt.Printf("    To:   %s\n", report.PeriodEnd.Format("2006-01-02 15:04:05"))
	fmt.Printf("    Total Events: %d\n", report.TotalEvents)
	fmt.Println()

	// Events by type
	if len(report.EventsByType) > 0 {
		fmt.Printf("  %s\n", ui.GrayText("Events by Type"))
		// Sort types for consistent output
		types := make([]string, 0, len(report.EventsByType))
		for t := range report.EventsByType {
			types = append(types, t)
		}
		sort.Strings(types)
		for _, t := range types {
			fmt.Printf("    %-20s %d\n", t, report.EventsByType[t])
		}
		fmt.Println()
	}

	// Agent activity
	if len(report.AgentSummary) > 0 {
		fmt.Printf("  %s\n", ui.GrayText("Agent Activity"))
		for _, as := range report.AgentSummary {
			fmt.Printf("    %s\n", ui.CyanText(as.Agent))
			fmt.Printf("      Events: %d  |  Work: %d assigned, %d completed, %d failed\n",
				as.TotalEvents, as.WorkAssigned, as.WorkCompleted, as.WorkFailed)
			fmt.Printf("      Active: %s to %s\n",
				as.FirstSeen.Format("01-02 15:04"),
				as.LastSeen.Format("01-02 15:04"))
		}
		fmt.Println()
	}

	// Error summary
	if report.ErrorSummary.TotalFailures > 0 {
		fmt.Printf("  %s\n", ui.YellowText("Error Summary"))
		fmt.Printf("    Total Failures: %d\n", report.ErrorSummary.TotalFailures)
		fmt.Printf("    Work Failures:  %d\n", report.ErrorSummary.WorkFailures)
		fmt.Printf("    Health Failures: %d\n", report.ErrorSummary.HealthFailures)
		if len(report.ErrorSummary.FailuresByAgent) > 0 {
			fmt.Printf("    By Agent:\n")
			for agent, count := range report.ErrorSummary.FailuresByAgent {
				fmt.Printf("      %s: %d\n", agent, count)
			}
		}
		fmt.Println()
	} else {
		fmt.Printf("  %s\n", ui.GreenText("✓ No failures recorded"))
		fmt.Println()
	}

	fmt.Printf("  Generated: %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	fmt.Println()
}
