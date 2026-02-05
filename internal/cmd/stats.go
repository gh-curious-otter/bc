package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/stats"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show workspace statistics",
	Long: `Display statistics about the current workspace including work item
metrics, agent utilization, and completion rates.

Example:
  bc stats             # human-readable summary
  bc stats --json      # JSON output for scripting
  bc stats --save      # save stats snapshot to .bc/stats.json`,
	RunE: runStats,
}

var (
	statsJSON bool
	statsSave bool
)

func init() {
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "Output as JSON")
	statsCmd.Flags().BoolVar(&statsSave, "save", false, "Save stats snapshot to disk")
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	s, err := stats.Load(ws.StateDir())
	if err != nil {
		return fmt.Errorf("failed to load stats: %w", err)
	}

	if statsSave {
		if err := s.Save(); err != nil {
			return fmt.Errorf("failed to save stats: %w", err)
		}
		fmt.Println("Stats saved to .bc/stats.json")
	}

	if statsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(s)
	}

	fmt.Print(s.Summary())

	// Show utilization as a quick indicator
	util := s.Utilization()
	if s.Agents.ActiveAgents > 0 {
		fmt.Printf("\nUtilization: %.0f%% (%d/%d agents working)\n",
			util*100, s.Agents.Working, s.Agents.ActiveAgents)
	}

	return nil
}
