package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rpuneet/bc/pkg/events"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the event log",
	Long: `View the bc event log showing agent spawns, stops, work assignments, and reports.

Example:
  bc logs                    # all events
  bc logs --agent worker-01  # filter by agent
  bc logs --tail 20          # last N events
  bc logs --json             # JSON output`,
	RunE: runLogs,
}

var (
	logsAgent string
	logsTail  int
)

func init() {
	logsCmd.Flags().StringVar(&logsAgent, "agent", "", "Filter by agent name")
	logsCmd.Flags().IntVar(&logsTail, "tail", 0, "Show last N events")
	rootCmd.AddCommand(logsCmd)
}

func runLogs(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	var evts []events.Event

	if logsAgent != "" {
		evts, err = log.ReadByAgent(logsAgent)
	} else if logsTail > 0 {
		evts, err = log.ReadLast(logsTail)
	} else {
		evts, err = log.Read()
	}
	if err != nil {
		return fmt.Errorf("failed to read events: %w", err)
	}

	if len(evts) == 0 {
		fmt.Println("No events found")
		return nil
	}

	// Apply tail after agent filter if both are set
	if logsAgent != "" && logsTail > 0 && len(evts) > logsTail {
		evts = evts[len(evts)-logsTail:]
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
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
		msg := ""
		if ev.Message != "" {
			msg = " " + ev.Message
		}
		fmt.Printf("%s %-20s%s%s\n", ts, ev.Type, agentStr, msg)
	}

	return nil
}
