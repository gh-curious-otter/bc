package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/audit"
	"github.com/rpuneet/bc/pkg/ui"
)

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "View and export audit logs for compliance",
	Long: `Query and export audit logs tracking agent actions.

Audit logs record:
- Agent lifecycle events (create, start, stop, delete)
- Channel messages
- Cost transactions
- Configuration changes
- Permission changes

Examples:
  bc audit list                          # List recent audit events
  bc audit list --type agent.create      # Filter by event type
  bc audit list --actor eng-01           # Filter by actor
  bc audit list --since 24h              # Events in last 24 hours
  bc audit export --format json          # Export to JSON
  bc audit export --format csv           # Export to CSV`,
}

var auditListCmd = &cobra.Command{
	Use:   "list",
	Short: "List audit events",
	RunE:  runAuditList,
}

var auditExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export audit logs to file",
	Long: `Export audit logs to JSON or CSV format.

Examples:
  bc audit export --format json > audit.json
  bc audit export --format csv --since 7d > audit.csv`,
	RunE: runAuditExport,
}

var auditStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show audit statistics",
	RunE:  runAuditStats,
}

// Flags
var (
	auditType   string
	auditActor  string
	auditTarget string
	auditSince  string
	auditUntil  string
	auditLimit  int
	auditFormat string
	auditJSON   bool
)

func init() {
	auditCmd.AddCommand(auditListCmd)
	auditCmd.AddCommand(auditExportCmd)
	auditCmd.AddCommand(auditStatsCmd)

	// List flags
	auditListCmd.Flags().StringVar(&auditType, "type", "", "Filter by event type (e.g., agent.create)")
	auditListCmd.Flags().StringVar(&auditActor, "actor", "", "Filter by actor")
	auditListCmd.Flags().StringVar(&auditTarget, "target", "", "Filter by target")
	auditListCmd.Flags().StringVar(&auditSince, "since", "", "Events since duration (e.g., 24h, 7d)")
	auditListCmd.Flags().StringVar(&auditUntil, "until", "", "Events until duration")
	auditListCmd.Flags().IntVar(&auditLimit, "limit", 50, "Max number of events")
	auditListCmd.Flags().BoolVar(&auditJSON, "json", false, "Output in JSON format")

	// Export flags
	auditExportCmd.Flags().StringVar(&auditFormat, "format", "json", "Export format (json, csv)")
	auditExportCmd.Flags().StringVar(&auditType, "type", "", "Filter by event type")
	auditExportCmd.Flags().StringVar(&auditActor, "actor", "", "Filter by actor")
	auditExportCmd.Flags().StringVar(&auditSince, "since", "", "Events since duration")
	auditExportCmd.Flags().IntVar(&auditLimit, "limit", 1000, "Max number of events")

	rootCmd.AddCommand(auditCmd)
}

func getAuditStore() (*audit.SQLiteStore, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, fmt.Errorf("not in a bc workspace: %w", err)
	}

	store := audit.NewSQLiteStore(ws.RootDir)
	if err := store.Open(); err != nil {
		return nil, fmt.Errorf("failed to open audit store: %w", err)
	}

	return store, nil
}

func buildFilter() (*audit.Filter, error) {
	filter := audit.NewFilter().WithLimit(auditLimit)

	if auditType != "" {
		filter.WithTypes(audit.EventType(auditType))
	}

	if auditActor != "" {
		filter.WithActor(auditActor)
	}

	if auditTarget != "" {
		filter.WithTarget(auditTarget)
	}

	if auditSince != "" {
		since, err := parseAuditDuration(auditSince)
		if err != nil {
			return nil, fmt.Errorf("invalid --since value: %w", err)
		}
		filter.Since = time.Now().Add(-since)
	}

	if auditUntil != "" {
		until, err := parseAuditDuration(auditUntil)
		if err != nil {
			return nil, fmt.Errorf("invalid --until value: %w", err)
		}
		filter.Until = time.Now().Add(-until)
	}

	return filter, nil
}

func parseAuditDuration(s string) (time.Duration, error) {
	// Support d (days) suffix
	if strings.HasSuffix(s, "d") {
		days := strings.TrimSuffix(s, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, err
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}
	return time.ParseDuration(s)
}

func runAuditList(cmd *cobra.Command, _ []string) error {
	store, err := getAuditStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	filter, err := buildFilter()
	if err != nil {
		return err
	}

	events, err := store.Query(filter)
	if err != nil {
		return fmt.Errorf("failed to query audit events: %w", err)
	}

	if auditJSON {
		data, err := json.MarshalIndent(events, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	if len(events) == 0 {
		fmt.Println("No audit events found")
		return nil
	}

	fmt.Printf("Audit Events (%d)\n", len(events))
	fmt.Println(strings.Repeat("-", 80))

	for _, e := range events {
		timestamp := e.Timestamp.Format("2006-01-02 15:04:05")
		typeColor := getEventTypeColor(e.Type)

		fmt.Printf("%s  %s  %s → %s\n",
			ui.GrayText(timestamp),
			typeColor(string(e.Type)),
			e.Actor,
			e.Target,
		)

		if len(e.Details) > 0 {
			for k, v := range e.Details {
				fmt.Printf("           %s: %s\n", k, v)
			}
		}
	}

	return nil
}

func getEventTypeColor(t audit.EventType) func(string) string {
	switch {
	case strings.HasPrefix(string(t), "agent."):
		return ui.CyanText
	case strings.HasPrefix(string(t), "channel."):
		return ui.GreenText
	case strings.HasPrefix(string(t), "cost."):
		return ui.YellowText
	case strings.HasPrefix(string(t), "config."):
		return ui.MagentaText
	case strings.HasPrefix(string(t), "permission."):
		return ui.RedText
	default:
		return func(s string) string { return s }
	}
}

func runAuditExport(cmd *cobra.Command, _ []string) error {
	store, err := getAuditStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	filter, err := buildFilter()
	if err != nil {
		return err
	}

	switch auditFormat {
	case "json":
		data, err := store.Export(filter)
		if err != nil {
			return err
		}
		fmt.Println(string(data))

	case "csv":
		data, err := store.ExportCSV(filter)
		if err != nil {
			return err
		}
		fmt.Print(data)

	default:
		return fmt.Errorf("unknown format: %s (use json or csv)", auditFormat)
	}

	return nil
}

func runAuditStats(_ *cobra.Command, _ []string) error {
	store, err := getAuditStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	// Get all events for stats
	filter := audit.NewFilter().WithLimit(10000)
	events, err := store.Query(filter)
	if err != nil {
		return fmt.Errorf("failed to query audit events: %w", err)
	}

	if len(events) == 0 {
		fmt.Println("No audit events found")
		return nil
	}

	// Count by type
	typeCounts := make(map[audit.EventType]int)
	actorCounts := make(map[string]int)
	var oldest, newest time.Time

	for _, e := range events {
		typeCounts[e.Type]++
		actorCounts[e.Actor]++

		if oldest.IsZero() || e.Timestamp.Before(oldest) {
			oldest = e.Timestamp
		}
		if newest.IsZero() || e.Timestamp.After(newest) {
			newest = e.Timestamp
		}
	}

	fmt.Println("Audit Statistics")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Printf("Total Events:  %d\n", len(events))
	fmt.Printf("Time Range:    %s to %s\n",
		oldest.Format("2006-01-02"),
		newest.Format("2006-01-02"),
	)
	fmt.Println()

	fmt.Println("Events by Type:")
	for t, count := range typeCounts {
		fmt.Printf("  %-25s %d\n", t, count)
	}
	fmt.Println()

	fmt.Println("Events by Actor:")
	for actor, count := range actorCounts {
		fmt.Printf("  %-20s %d\n", actor, count)
	}

	return nil
}

// LogAuditEvent is a helper to log an audit event from other commands.
func LogAuditEvent(eventType audit.EventType, actor, target string, details map[string]string) error {
	ws, err := getWorkspace()
	if err != nil {
		return nil // Silently fail if not in workspace
	}

	store := audit.NewSQLiteStore(ws.RootDir)
	if err := store.Open(); err != nil {
		return nil // Silently fail
	}
	defer func() { _ = store.Close() }()

	event := audit.NewEvent(eventType, actor, target).WithWorkspace(filepath.Base(ws.RootDir))
	for k, v := range details {
		event.WithDetail(k, v)
	}

	return store.Log(event)
}
