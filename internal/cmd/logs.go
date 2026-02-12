package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/events"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the event log",
	Long: `View the bc event log showing agent spawns, stops, work assignments, and reports.

Examples:
  bc logs                    # all events
  bc logs --agent worker-01  # filter by agent
  bc logs --type agent.report # filter by event type
  bc logs --since 1h         # events from last hour
  bc logs --tail 20          # last N events
  bc logs --full             # show full messages (no truncation)
  bc logs --json             # JSON output`,
	RunE: runLogs,
}

var (
	logsAgent string
	logsTail  int
	logsType  string
	logsSince string
	logsFull  bool
)

const logsMaxMessageLen = 80

func init() {
	logsCmd.Flags().StringVar(&logsAgent, "agent", "", "Filter by agent name")
	logsCmd.Flags().IntVar(&logsTail, "tail", 0, "Show last N events")
	logsCmd.Flags().StringVar(&logsType, "type", "", "Filter by event type (e.g. agent.report)")
	logsCmd.Flags().StringVar(&logsSince, "since", "", "Show events since duration ago (e.g. 1h, 30m)")
	logsCmd.Flags().BoolVar(&logsFull, "full", false, "Show full messages without truncation")
	rootCmd.AddCommand(logsCmd)
}

// parseSinceDuration parses a duration string like "1h", "30m", "2h30m" and
// returns the cutoff time (now minus duration).
func parseSinceDuration(s string) (time.Time, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid duration %q: %w (use e.g. 1h, 30m, 2h30m)", s, err)
	}
	return time.Now().Add(-d), nil
}

// truncateMessage truncates a string to maxLen characters, appending "..." if truncated.
func truncateMessage(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

func runLogs(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("tail") && logsTail <= 0 {
		return fmt.Errorf("tail must be a positive number")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Read all events, then filter in sequence
	evts, err := log.Read()
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	// Filter by agent
	if logsAgent != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if ev.Agent == logsAgent {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	// Filter by event type
	if logsType != "" {
		filtered := evts[:0]
		for _, ev := range evts {
			if string(ev.Type) == logsType {
				filtered = append(filtered, ev)
			}
		}
		evts = filtered
	}

	// Filter by time (--since)
	if logsSince != "" {
		cutoff, parseErr := parseSinceDuration(logsSince)
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

	// Apply tail last (take only the last N events)
	if logsTail > 0 && len(evts) > logsTail {
		evts = evts[len(evts)-logsTail:]
	}

	if len(evts) == 0 {
		fmt.Println("No events found")
		return nil
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(evts)
	}

	for _, ev := range evts {
		ts := ev.Timestamp.Format("15:04:05")
		agentStr := ""
		if ev.Agent != "" {
			agentStr = fmt.Sprintf(" [%s]", ev.Agent)
		}
		// For message.sent, show sender → recipient
		if ev.Type == events.MessageSent && ev.Data != nil {
			if recipient, ok := ev.Data["recipient"].(string); ok && recipient != "" {
				agentStr = fmt.Sprintf(" [%s] → [%s]", ev.Agent, recipient)
			}
		}
		msg := ""
		if ev.Message != "" {
			m := ev.Message
			if !logsFull {
				m = truncateMessage(m, logsMaxMessageLen)
			}
			msg = " " + m
		}
		fmt.Printf("%s %-20s%s%s\n", ts, ev.Type, agentStr, msg)
	}

	return nil
}
