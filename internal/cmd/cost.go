package cmd

import (
	"fmt"
	"os"
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
		fmt.Println("No cost records found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
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

	// Specific agent summary
	if costAgentFlag != "" {
		summary, summaryErr := store.AgentSummary(costAgentFlag)
		if summaryErr != nil {
			return fmt.Errorf("failed to get agent summary: %w", summaryErr)
		}
		printSingleSummary("Agent", costAgentFlag, summary)
		return nil
	}

	// Specific team summary
	if costTeamFlag != "" {
		summary, summaryErr := store.TeamSummary(costTeamFlag)
		if summaryErr != nil {
			return fmt.Errorf("failed to get team summary: %w", summaryErr)
		}
		printSingleSummary("Team", costTeamFlag, summary)
		return nil
	}

	// Model summary
	if costModelFlag {
		summaries, summaryErr := store.SummaryByModel()
		if summaryErr != nil {
			return fmt.Errorf("failed to get model summary: %w", summaryErr)
		}
		printModelSummary(summaries)
		return nil
	}

	// Default: workspace summary
	if costWorkspaceFlag || (!costWorkspaceFlag && costTeamFlag == "" && costAgentFlag == "" && !costModelFlag) {
		summary, summaryErr := store.WorkspaceSummary()
		if summaryErr != nil {
			return fmt.Errorf("failed to get workspace summary: %w", summaryErr)
		}
		printWorkspaceSummary(summary)

		// Also show per-agent breakdown
		agentSummaries, agentErr := store.SummaryByAgent()
		if agentErr == nil && len(agentSummaries) > 0 {
			fmt.Println("\nBy Agent:")
			printCostAgentSummary(agentSummaries)
		}

		return nil
	}

	return nil
}

func printSingleSummary(label, name string, s *cost.Summary) {
	fmt.Printf("%s: %s\n", label, name)
	fmt.Printf("  Records:      %d\n", s.RecordCount)
	fmt.Printf("  Input Tokens: %d\n", s.InputTokens)
	fmt.Printf("  Output Tokens: %d\n", s.OutputTokens)
	fmt.Printf("  Total Tokens: %d\n", s.TotalTokens)
	fmt.Printf("  Total Cost:   $%.4f\n", s.TotalCostUSD)
}

func printWorkspaceSummary(s *cost.Summary) {
	fmt.Println("Workspace Summary")
	fmt.Println("=================")
	fmt.Printf("  API Calls:    %d\n", s.RecordCount)
	fmt.Printf("  Input Tokens: %d\n", s.InputTokens)
	fmt.Printf("  Output Tokens: %d\n", s.OutputTokens)
	fmt.Printf("  Total Tokens: %d\n", s.TotalTokens)
	fmt.Printf("  Total Cost:   $%.4f\n", s.TotalCostUSD)
}

func printCostAgentSummary(summaries []*cost.Summary) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "  AGENT\tCALLS\tTOKENS\tCOST")

	for _, s := range summaries {
		_, _ = fmt.Fprintf(w, "  %s\t%d\t%d\t$%.4f\n",
			s.AgentID,
			s.RecordCount,
			s.TotalTokens,
			s.TotalCostUSD,
		)
	}
	_ = w.Flush()
}

func printModelSummary(summaries []*cost.Summary) {
	if len(summaries) == 0 {
		fmt.Println("No cost records found")
		return
	}

	fmt.Println("Cost by Model")
	fmt.Println("=============")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "MODEL\tCALLS\tINPUT\tOUTPUT\tTOTAL\tCOST")

	var totalCost float64
	for _, s := range summaries {
		_, _ = fmt.Fprintf(w, "%s\t%d\t%d\t%d\t%d\t$%.4f\n",
			s.Model,
			s.RecordCount,
			s.InputTokens,
			s.OutputTokens,
			s.TotalTokens,
			s.TotalCostUSD,
		)
		totalCost += s.TotalCostUSD
	}
	_ = w.Flush()
	fmt.Printf("\nTotal: $%.4f\n", totalCost)
}
