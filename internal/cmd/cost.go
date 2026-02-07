package cmd

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/cost"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "View cost information",
	Long: `Commands for viewing API cost information.

Example:
  bc cost show
  bc cost show engineer-01
  bc cost summary --workspace
  bc cost summary --team engineering`,
}

var costShowCmd = &cobra.Command{
	Use:   "show [agent]",
	Short: "Show cost records",
	Long: `Show cost records, optionally filtered by agent.

Example:
  bc cost show
  bc cost show engineer-01`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCostShow,
}

var costSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show cost summary",
	Long: `Show aggregated cost summary.

Example:
  bc cost summary --workspace
  bc cost summary --team engineering
  bc cost summary --agent engineer-01
  bc cost summary --model`,
	RunE: runCostSummary,
}

var costDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show cost dashboard with full breakdown",
	Long: `Display a comprehensive cost dashboard with workspace totals,
per-agent breakdown, and team aggregation.

Example:
  bc cost dashboard`,
	RunE: runCostDashboard,
}

var (
	costTeamFlag      string
	costAgentFlag     string
	costWorkspaceFlag bool
	costModelFlag     bool
	costLimitFlag     int
)

func init() {
	costShowCmd.Flags().IntVarP(&costLimitFlag, "limit", "n", 20, "Number of records to show")

	costSummaryCmd.Flags().StringVar(&costTeamFlag, "team", "", "Show summary for a specific team")
	costSummaryCmd.Flags().StringVar(&costAgentFlag, "agent", "", "Show summary for a specific agent")
	costSummaryCmd.Flags().BoolVar(&costWorkspaceFlag, "workspace", false, "Show workspace-wide summary")
	costSummaryCmd.Flags().BoolVar(&costModelFlag, "model", false, "Show summary grouped by model")

	costCmd.AddCommand(costShowCmd)
	costCmd.AddCommand(costSummaryCmd)
	costCmd.AddCommand(costDashboardCmd)
	rootCmd.AddCommand(costCmd)
}

func getCostStore() (*cost.Store, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, fmt.Errorf("not in a bc workspace: %w", err)
	}

	store := cost.NewStore(ws.RootDir)
	if err := store.Open(); err != nil {
		return nil, fmt.Errorf("failed to open cost store: %w", err)
	}

	return store, nil
}

func runCostShow(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	var records []*cost.Record
	if len(args) > 0 {
		agentID := args[0]
		records, err = store.GetByAgent(agentID, costLimitFlag)
	} else {
		records, err = store.GetAll(costLimitFlag)
	}

	if err != nil {
		return fmt.Errorf("failed to get cost records: %w", err)
	}

	if len(records) == 0 {
		cmd.Println("No cost records found")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "TIMESTAMP\tAGENT\tMODEL\tINPUT\tOUTPUT\tTOTAL\tCOST")

	for _, r := range records {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\t%d\t$%.4f\n",
			r.Timestamp.Format("2006-01-02 15:04"),
			r.AgentID,
			r.Model,
			r.InputTokens,
			r.OutputTokens,
			r.TotalTokens,
			r.CostUSD,
		)
	}

	return w.Flush()
}

func runCostSummary(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	out := cmd.OutOrStdout()

	// Specific agent summary
	if costAgentFlag != "" {
		summary, summaryErr := store.AgentSummary(costAgentFlag)
		if summaryErr != nil {
			return fmt.Errorf("failed to get agent summary: %w", summaryErr)
		}
		_, _ = fmt.Fprintf(out, "Agent: %s\n", costAgentFlag)
		_, _ = fmt.Fprintf(out, "  Records:      %d\n", summary.RecordCount)
		_, _ = fmt.Fprintf(out, "  Input Tokens: %d\n", summary.InputTokens)
		_, _ = fmt.Fprintf(out, "  Output Tokens: %d\n", summary.OutputTokens)
		_, _ = fmt.Fprintf(out, "  Total Tokens: %d\n", summary.TotalTokens)
		_, _ = fmt.Fprintf(out, "  Total Cost:   $%.4f\n", summary.TotalCostUSD)
		return nil
	}

	// Specific team summary
	if costTeamFlag != "" {
		summary, summaryErr := store.TeamSummary(costTeamFlag)
		if summaryErr != nil {
			return fmt.Errorf("failed to get team summary: %w", summaryErr)
		}
		_, _ = fmt.Fprintf(out, "Team: %s\n", costTeamFlag)
		_, _ = fmt.Fprintf(out, "  Records:      %d\n", summary.RecordCount)
		_, _ = fmt.Fprintf(out, "  Input Tokens: %d\n", summary.InputTokens)
		_, _ = fmt.Fprintf(out, "  Output Tokens: %d\n", summary.OutputTokens)
		_, _ = fmt.Fprintf(out, "  Total Tokens: %d\n", summary.TotalTokens)
		_, _ = fmt.Fprintf(out, "  Total Cost:   $%.4f\n", summary.TotalCostUSD)
		return nil
	}

	// Model summary
	if costModelFlag {
		summaries, summaryErr := store.SummaryByModel()
		if summaryErr != nil {
			return fmt.Errorf("failed to get model summary: %w", summaryErr)
		}
		if len(summaries) == 0 {
			_, _ = fmt.Fprintln(out, "No cost records found")
			return nil
		}
		_, _ = fmt.Fprintln(out, "Cost by Model")
		_, _ = fmt.Fprintln(out, "=============")
		w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "MODEL\tCALLS\tINPUT\tOUTPUT\tTOTAL\tCOST")
		var totalCost float64
		for _, s := range summaries {
			_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t$%.4f\n",
				s.Model, s.RecordCount, s.InputTokens, s.OutputTokens, s.TotalTokens, s.TotalCostUSD)
			totalCost += s.TotalCostUSD
		}
		_ = w.Flush()
		_, _ = fmt.Fprintf(out, "\nTotal: $%.4f\n", totalCost)
		return nil
	}

	// Default: workspace summary
	if costWorkspaceFlag || (!costWorkspaceFlag && costTeamFlag == "" && costAgentFlag == "" && !costModelFlag) {
		summary, summaryErr := store.WorkspaceSummary()
		if summaryErr != nil {
			return fmt.Errorf("failed to get workspace summary: %w", summaryErr)
		}
		_, _ = fmt.Fprintln(out, "Workspace Summary")
		_, _ = fmt.Fprintln(out, "=================")
		_, _ = fmt.Fprintf(out, "  API Calls:    %d\n", summary.RecordCount)
		_, _ = fmt.Fprintf(out, "  Input Tokens: %d\n", summary.InputTokens)
		_, _ = fmt.Fprintf(out, "  Output Tokens: %d\n", summary.OutputTokens)
		_, _ = fmt.Fprintf(out, "  Total Tokens: %d\n", summary.TotalTokens)
		_, _ = fmt.Fprintf(out, "  Total Cost:   $%.4f\n", summary.TotalCostUSD)

		// Also show per-agent breakdown
		agentSummaries, agentErr := store.SummaryByAgent()
		if agentErr == nil && len(agentSummaries) > 0 {
			_, _ = fmt.Fprintln(out, "\nBy Agent:")
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "  AGENT\tCALLS\tTOKENS\tCOST")
			for _, s := range agentSummaries {
				_, _ = fmt.Fprintf(w, "  %s\t%d\t%d\t$%.4f\n",
					s.AgentID, s.RecordCount, s.TotalTokens, s.TotalCostUSD)
			}
			_ = w.Flush()
		}

		return nil
	}

	return nil
}

func runCostDashboard(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	// Workspace totals
	wsSummary, wsErr := store.WorkspaceSummary()
	if wsErr != nil {
		return fmt.Errorf("failed to get workspace summary: %w", wsErr)
	}

	cmd.Println("╔══════════════════════════════════════════════════════════════╗")
	cmd.Println("║                     COST DASHBOARD                           ║")
	cmd.Println("╚══════════════════════════════════════════════════════════════╝")
	cmd.Println()

	// Workspace Summary
	cmd.Println("WORKSPACE TOTALS")
	cmd.Println("────────────────")
	cmd.Printf("  Total API Calls:   %d\n", wsSummary.RecordCount)
	cmd.Printf("  Input Tokens:      %d\n", wsSummary.InputTokens)
	cmd.Printf("  Output Tokens:     %d\n", wsSummary.OutputTokens)
	cmd.Printf("  Total Tokens:      %d\n", wsSummary.TotalTokens)
	cmd.Printf("  Total Cost:        $%.4f\n", wsSummary.TotalCostUSD)
	cmd.Println()

	// Per-agent breakdown
	agentSummaries, agentErr := store.SummaryByAgent()
	if agentErr != nil {
		return fmt.Errorf("failed to get agent summaries: %w", agentErr)
	}

	if len(agentSummaries) > 0 {
		cmd.Println("BY AGENT")
		cmd.Println("────────")
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "  AGENT\tCALLS\tTOKENS\tCOST\t% OF TOTAL")

		for _, s := range agentSummaries {
			pct := 0.0
			if wsSummary.TotalCostUSD > 0 {
				pct = (s.TotalCostUSD / wsSummary.TotalCostUSD) * 100
			}
			_, _ = fmt.Fprintf(w, "  %s\t%d\t%d\t$%.4f\t%.1f%%\n",
				s.AgentID,
				s.RecordCount,
				s.TotalTokens,
				s.TotalCostUSD,
				pct,
			)
		}
		_ = w.Flush()
		cmd.Println()
	}

	// Team aggregation
	teamSummaries, teamErr := store.SummaryByTeam()
	if teamErr != nil {
		return fmt.Errorf("failed to get team summaries: %w", teamErr)
	}

	if len(teamSummaries) > 0 {
		cmd.Println("BY TEAM")
		cmd.Println("───────")
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "  TEAM\tCALLS\tTOKENS\tCOST\t% OF TOTAL")

		for _, s := range teamSummaries {
			pct := 0.0
			if wsSummary.TotalCostUSD > 0 {
				pct = (s.TotalCostUSD / wsSummary.TotalCostUSD) * 100
			}
			_, _ = fmt.Fprintf(w, "  %s\t%d\t%d\t$%.4f\t%.1f%%\n",
				s.TeamID,
				s.RecordCount,
				s.TotalTokens,
				s.TotalCostUSD,
				pct,
			)
		}
		_ = w.Flush()
		cmd.Println()
	}

	// Model breakdown
	modelSummaries, modelErr := store.SummaryByModel()
	if modelErr != nil {
		return fmt.Errorf("failed to get model summaries: %w", modelErr)
	}

	if len(modelSummaries) > 0 {
		cmd.Println("BY MODEL")
		cmd.Println("────────")
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "  MODEL\tCALLS\tTOKENS\tCOST\t% OF TOTAL")

		for _, s := range modelSummaries {
			pct := 0.0
			if wsSummary.TotalCostUSD > 0 {
				pct = (s.TotalCostUSD / wsSummary.TotalCostUSD) * 100
			}
			_, _ = fmt.Fprintf(w, "  %s\t%d\t%d\t$%.4f\t%.1f%%\n",
				s.Model,
				s.RecordCount,
				s.TotalTokens,
				s.TotalCostUSD,
				pct,
			)
		}
		_ = w.Flush()
	}

	return nil
}
