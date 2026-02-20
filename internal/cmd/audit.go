package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/events"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Audit logging and compliance reporting",
	Long: `Access audit logs for compliance and reporting.

The audit system tracks all agent activities including:
- Agent spawns and stops
- Work assignments and completions
- Status reports and health checks
- Messages between agents

Examples:
  bc audit export --since 7d           # Export last 7 days
  bc audit export --format csv         # Export as CSV
  bc audit report                      # Summary report
  bc audit report --since 30d          # Monthly report`,
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit logs",
	Long: `Export audit logs to JSON or CSV format.

By default exports all events. Use filters to narrow down:
  --since DURATION   Events from the last duration (1h, 7d, 30d)
  --agent NAME       Events for specific agent
  --type TYPE        Events of specific type
  --format FORMAT    Output format (json, csv)
  --output FILE      Write to file instead of stdout

Examples:
  bc audit export --since 7d > audit.json
  bc audit export --format csv --output audit.csv
  bc audit export --agent eng-01 --since 24h`,
	RunE: runAuditExport,
}

var auditReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate audit summary report",
	Long: `Generate a summary report of audit events.

Shows:
- Total events by type
- Events by agent
- Timeline summary
- Error/warning counts

Examples:
  bc audit report              # Full report
  bc audit report --since 7d   # Weekly report
  bc audit report --since 30d  # Monthly report`,
	RunE: runAuditReport,
}

var (
	auditSince  string
	auditAgent  string
	auditType   string
	auditFormat string
	auditOutput string
)

func init() {
	// Export flags
	auditExportCmd.Flags().StringVar(&auditSince, "since", "", "Filter events since duration (e.g. 1h, 7d, 30d)")
	auditExportCmd.Flags().StringVar(&auditAgent, "agent", "", "Filter by agent name")
	auditExportCmd.Flags().StringVar(&auditType, "type", "", "Filter by event type")
	auditExportCmd.Flags().StringVar(&auditFormat, "format", "json", "Output format (json, csv)")
	auditExportCmd.Flags().StringVar(&auditOutput, "output", "", "Output file (default: stdout)")

	// Report flags
	auditReportCmd.Flags().StringVar(&auditSince, "since", "", "Filter events since duration (e.g. 1h, 7d, 30d)")

	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditReportCmd)
	rootCmd.AddCommand(auditCmd)
}

func runAuditExport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	evts, err := eventLog.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Apply filters
	evts = filterEvents(evts, auditSince, auditAgent, auditType)

	if len(evts) == 0 {
		cmd.Println("No events found matching criteria")
		return nil
	}

	// Determine output destination
	var out *os.File
	if auditOutput != "" {
		f, createErr := os.Create(auditOutput) //nolint:gosec // user-provided output path
		if createErr != nil {
			return fmt.Errorf("failed to create output file: %w", createErr)
		}
		defer func() { _ = f.Close() }()
		out = f
	} else {
		out = os.Stdout
	}

	// Export based on format
	switch auditFormat {
	case "csv":
		return exportCSV(out, evts)
	case "json":
		return exportJSON(out, evts)
	default:
		return fmt.Errorf("invalid format %q (use json or csv)", auditFormat)
	}
}

func runAuditReport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	evts, err := eventLog.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Apply time filter only for report
	evts = filterEvents(evts, auditSince, "", "")

	if len(evts) == 0 {
		cmd.Println("No events found")
		return nil
	}

	// Calculate stats
	typeCount := make(map[string]int)
	agentCount := make(map[string]int)
	var earliest, latest time.Time

	for _, ev := range evts {
		typeCount[string(ev.Type)]++
		if ev.Agent != "" {
			agentCount[ev.Agent]++
		}
		if earliest.IsZero() || ev.Timestamp.Before(earliest) {
			earliest = ev.Timestamp
		}
		if latest.IsZero() || ev.Timestamp.After(latest) {
			latest = ev.Timestamp
		}
	}

	// Print report
	cmd.Println("AUDIT REPORT")
	cmd.Println("============")
	cmd.Println()

	// Time range
	cmd.Printf("Period: %s to %s\n", earliest.Format("2006-01-02 15:04"), latest.Format("2006-01-02 15:04"))
	cmd.Printf("Duration: %s\n", latest.Sub(earliest).Round(time.Minute))
	cmd.Printf("Total Events: %d\n", len(evts))
	cmd.Println()

	// Events by type
	cmd.Println("Events by Type:")
	types := sortedKeys(typeCount)
	for _, t := range types {
		cmd.Printf("  %-25s %d\n", t, typeCount[t])
	}
	cmd.Println()

	// Events by agent
	if len(agentCount) > 0 {
		cmd.Println("Events by Agent:")
		agents := sortedKeys(agentCount)
		for _, a := range agents {
			cmd.Printf("  %-20s %d\n", a, agentCount[a])
		}
		cmd.Println()
	}

	// Summary stats
	spawns := typeCount[string(events.AgentSpawned)]
	stops := typeCount[string(events.AgentStopped)]
	errors := typeCount[string(events.HealthFailed)] + typeCount[string(events.WorkFailed)]

	cmd.Println("Summary:")
	cmd.Printf("  Agents started: %d\n", spawns)
	cmd.Printf("  Agents stopped: %d\n", stops)
	cmd.Printf("  Errors/failures: %d\n", errors)

	return nil
}

func filterEvents(evts []events.Event, since, agent, eventType string) []events.Event {
	// Filter by time
	if since != "" {
		cutoff, err := parseSinceDuration(since)
		if err == nil {
			filtered := evts[:0]
			for _, ev := range evts {
				if !ev.Timestamp.Before(cutoff) {
					filtered = append(filtered, ev)
				}
			}
			evts = filtered
		}
	}

	// Filter by agent
	if agent != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if ev.Agent == agent {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	// Filter by type
	if eventType != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if string(ev.Type) == eventType {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	return evts
}

func exportJSON(out *os.File, evts []events.Event) error {
	//nolint:govet // fieldalignment: JSON field order matters more than struct alignment
	export := struct {
		ExportedAt time.Time      `json:"exported_at"`
		Count      int            `json:"count"`
		Events     []events.Event `json:"events"`
	}{
		ExportedAt: time.Now().UTC(),
		Count:      len(evts),
		Events:     evts,
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(export)
}

func exportCSV(out *os.File, evts []events.Event) error {
	w := csv.NewWriter(out)
	defer w.Flush()

	// Header
	if err := w.Write([]string{"timestamp", "type", "agent", "message"}); err != nil {
		return err
	}

	// Data rows
	for _, ev := range evts {
		row := []string{
			ev.Timestamp.Format(time.RFC3339),
			string(ev.Type),
			ev.Agent,
			ev.Message,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
