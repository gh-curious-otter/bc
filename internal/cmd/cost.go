package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/cost"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show cost information",
	Long: `Commands for viewing API cost information.

Examples:
  bc cost show
  bc cost show engineer-01
  bc cost summary --workspace
  bc cost summary --team engineering`,
}

var costShowCmd = &cobra.Command{
	Use:   "show [agent]",
	Short: "Show cost records",
	Long: `Show cost records, optionally filtered by agent.

Examples:
  bc cost show
  bc cost show engineer-01`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCostShow,
}

var costSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show cost summary",
	Long: `Show aggregated cost summary.

Examples:
  bc cost summary --workspace
  bc cost summary --team engineering
  bc cost summary --agent engineer-01
  bc cost summary --model`,
	RunE: runCostSummary,
}

var costDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show comprehensive cost dashboard",
	Long: `Display a comprehensive cost dashboard with all aggregations.

Shows:
  - Workspace totals (headline numbers)
  - Per-agent breakdown
  - Per-team breakdown
  - Per-model breakdown

Examples:
  bc cost dashboard`,
	RunE: runCostDashboard,
}

var costBudgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage cost budgets",
	Long: `Commands for managing cost budgets and limits.

Examples:
  bc cost budget show
  bc cost budget set 100.00
  bc cost budget set 50.00 --agent engineer-01
  bc cost budget set 500.00 --period monthly --alert-at 0.9`,
}

var costBudgetSetCmd = &cobra.Command{
	Use:   "set <amount>",
	Short: "Set a cost budget",
	Long: `Set a cost budget for the workspace, agent, or team.

Examples:
  bc cost budget set 100.00                          # Set workspace budget to $100
  bc cost budget set 50.00 --agent engineer-01       # Set agent budget
  bc cost budget set 500.00 --team engineering       # Set team budget
  bc cost budget set 100.00 --period weekly          # Weekly budget
  bc cost budget set 100.00 --alert-at 0.9           # Alert at 90%
  bc cost budget set 100.00 --hard-stop              # Stop when limit reached`,
	Args: cobra.ExactArgs(1),
	RunE: runCostBudgetSet,
}

var costBudgetShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show budget status",
	Long: `Show current budget configuration and status.

Examples:
  bc cost budget show                   # Show all budgets
  bc cost budget show --agent eng-01    # Show agent budget`,
	RunE: runCostBudgetShow,
}

var costBudgetDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a budget",
	Long: `Delete a budget configuration.

Examples:
  bc cost budget delete                  # Delete workspace budget
  bc cost budget delete --agent eng-01   # Delete agent budget`,
	RunE: runCostBudgetDelete,
}

var costProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Project future costs",
	Long: `Project future costs based on historical daily spending.

Uses the average daily cost over the lookback period to estimate
future costs for the specified duration.

Examples:
  bc cost project --duration 1d          # Estimate next day's cost
  bc cost project --duration 7d          # Weekly projection
  bc cost project --duration 30d         # Monthly projection
  bc cost project --lookback 14          # Use 14 days of history`,
	RunE: runCostProject,
}

var costTrendsCmd = &cobra.Command{
	Use:   "trends",
	Short: "Show spending trends",
	Long: `Show daily spending trends over a time period.

Displays daily cost breakdown to help identify patterns and anomalies.

Examples:
  bc cost trends --since 7d              # Last 7 days
  bc cost trends --since 30d             # Last 30 days
  bc cost trends --since 24h             # Last 24 hours`,
	RunE: runCostTrends,
}

var costByAgentCmd = &cobra.Command{
	Use:   "by-agent",
	Short: "Show costs by agent",
	Long: `Show cost breakdown by agent over a time period.

Helps identify which agents are consuming the most resources.

Examples:
  bc cost by-agent --since 7d            # Last 7 days
  bc cost by-agent --since 30d           # Last 30 days`,
	RunE: runCostByAgent,
}

var (
	costTeamFlag      string
	costAgentFlag     string
	costWorkspaceFlag bool
	costModelFlag     bool
	costLimitFlag     int

	// Budget flags
	budgetAgentFlag   string
	budgetTeamFlag    string
	budgetPeriodFlag  string
	budgetAlertAtFlag float64
	budgetHardStop    bool

	// Projection flags
	projectDurationFlag string
	projectLookbackFlag int

	// Trends/by-agent flags
	trendsSinceFlag  string
	byAgentSinceFlag string
)

func init() {
	costShowCmd.Flags().IntVarP(&costLimitFlag, "limit", "n", 20, "Number of records to show")

	costSummaryCmd.Flags().StringVar(&costTeamFlag, "team", "", "Show summary for a specific team")
	costSummaryCmd.Flags().StringVar(&costAgentFlag, "agent", "", "Show summary for a specific agent")
	costSummaryCmd.Flags().BoolVar(&costWorkspaceFlag, "workspace", false, "Show workspace-wide summary")
	costSummaryCmd.Flags().BoolVar(&costModelFlag, "model", false, "Show summary grouped by model")

	// Budget flags
	costBudgetSetCmd.Flags().StringVar(&budgetAgentFlag, "agent", "", "Set budget for specific agent")
	costBudgetSetCmd.Flags().StringVar(&budgetTeamFlag, "team", "", "Set budget for specific team")
	costBudgetSetCmd.Flags().StringVar(&budgetPeriodFlag, "period", "monthly", "Budget period (daily, weekly, monthly)")
	costBudgetSetCmd.Flags().Float64Var(&budgetAlertAtFlag, "alert-at", 0.8, "Alert when usage reaches this percentage (0.0-1.0)")
	costBudgetSetCmd.Flags().BoolVar(&budgetHardStop, "hard-stop", false, "Stop operations when budget is exceeded")

	costBudgetShowCmd.Flags().StringVar(&budgetAgentFlag, "agent", "", "Show budget for specific agent")
	costBudgetShowCmd.Flags().StringVar(&budgetTeamFlag, "team", "", "Show budget for specific team")

	costBudgetDeleteCmd.Flags().StringVar(&budgetAgentFlag, "agent", "", "Delete budget for specific agent")
	costBudgetDeleteCmd.Flags().StringVar(&budgetTeamFlag, "team", "", "Delete budget for specific team")

	costBudgetCmd.AddCommand(costBudgetSetCmd)
	costBudgetCmd.AddCommand(costBudgetShowCmd)
	costBudgetCmd.AddCommand(costBudgetDeleteCmd)

	// Projection flags
	costProjectCmd.Flags().StringVar(&projectDurationFlag, "duration", "7d", "Duration to project (e.g., 1d, 7d, 30d)")
	costProjectCmd.Flags().IntVar(&projectLookbackFlag, "lookback", 7, "Days of history to use for projection")

	// Trends flags
	costTrendsCmd.Flags().StringVar(&trendsSinceFlag, "since", "7d", "Time period to show (e.g., 24h, 7d, 30d)")

	// By-agent flags
	costByAgentCmd.Flags().StringVar(&byAgentSinceFlag, "since", "7d", "Time period to show (e.g., 24h, 7d, 30d)")

	costCmd.AddCommand(costShowCmd)
	costCmd.AddCommand(costSummaryCmd)
	costCmd.AddCommand(costDashboardCmd)
	costCmd.AddCommand(costBudgetCmd)
	costCmd.AddCommand(costProjectCmd)
	costCmd.AddCommand(costTrendsCmd)
	costCmd.AddCommand(costByAgentCmd)
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

func runCostDashboard(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	// Workspace summary (headline)
	wsSummary, err := store.WorkspaceSummary()
	if err != nil {
		return fmt.Errorf("failed to get workspace summary: %w", err)
	}

	cmd.Println("╔══════════════════════════════════════════════════════════════╗")
	cmd.Println("║                     COST DASHBOARD                           ║")
	cmd.Println("╚══════════════════════════════════════════════════════════════╝")
	cmd.Println()

	// Workspace totals
	cmd.Println("WORKSPACE TOTALS")
	cmd.Println("────────────────")
	cmd.Printf("  Total Cost:     $%.4f\n", wsSummary.TotalCostUSD)
	cmd.Printf("  API Calls:      %d\n", wsSummary.RecordCount)
	cmd.Printf("  Total Tokens:   %d\n", wsSummary.TotalTokens)
	cmd.Printf("  Input Tokens:   %d\n", wsSummary.InputTokens)
	cmd.Printf("  Output Tokens:  %d\n", wsSummary.OutputTokens)
	cmd.Println()

	// Per-agent breakdown
	agentSummaries, err := store.SummaryByAgent()
	if err == nil && len(agentSummaries) > 0 {
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

	// Per-team breakdown
	teamSummaries, err := store.SummaryByTeam()
	if err == nil && len(teamSummaries) > 0 {
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

	// Per-model breakdown
	modelSummaries, err := store.SummaryByModel()
	if err == nil && len(modelSummaries) > 0 {
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

func getBudgetScope() string {
	if budgetAgentFlag != "" {
		return "agent:" + budgetAgentFlag
	}
	if budgetTeamFlag != "" {
		return "team:" + budgetTeamFlag
	}
	return "workspace"
}

func runCostBudgetSet(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	// Parse amount
	var limitUSD float64
	if _, parseErr := fmt.Sscanf(args[0], "%f", &limitUSD); parseErr != nil {
		return fmt.Errorf("invalid amount: %s", args[0])
	}
	if limitUSD <= 0 {
		return fmt.Errorf("budget amount must be positive")
	}

	// Validate period
	period := cost.BudgetPeriod(budgetPeriodFlag)
	switch period {
	case cost.BudgetPeriodDaily, cost.BudgetPeriodWeekly, cost.BudgetPeriodMonthly:
		// Valid
	default:
		return fmt.Errorf("invalid period: %s (must be daily, weekly, or monthly)", budgetPeriodFlag)
	}

	// Validate alert threshold
	if budgetAlertAtFlag < 0 || budgetAlertAtFlag > 1 {
		return fmt.Errorf("alert-at must be between 0.0 and 1.0")
	}

	scope := getBudgetScope()
	budget, err := store.SetBudget(scope, period, limitUSD, budgetAlertAtFlag, budgetHardStop)
	if err != nil {
		return fmt.Errorf("failed to set budget: %w", err)
	}

	fmt.Printf("Budget set for %s:\n", scope)
	fmt.Printf("  Limit:     $%.2f/%s\n", budget.LimitUSD, budget.Period)
	fmt.Printf("  Alert at:  %.0f%%\n", budget.AlertAt*100)
	fmt.Printf("  Hard stop: %v\n", budget.HardStop)

	return nil
}

func runCostBudgetShow(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	scope := getBudgetScope()

	// If showing specific scope
	if budgetAgentFlag != "" || budgetTeamFlag != "" || scope == "workspace" {
		status, checkErr := store.CheckBudget(scope)
		if checkErr != nil {
			return fmt.Errorf("failed to check budget: %w", checkErr)
		}

		if status == nil {
			fmt.Printf("No budget configured for %s\n", scope)
			return nil
		}

		printBudgetStatus(scope, status)
		return nil
	}

	// Show all budgets
	budgets, err := store.GetAllBudgets()
	if err != nil {
		return fmt.Errorf("failed to get budgets: %w", err)
	}

	if len(budgets) == 0 {
		fmt.Println("No budgets configured")
		fmt.Println("\nUse 'bc cost budget set <amount>' to set a budget")
		return nil
	}

	fmt.Println("Configured Budgets")
	fmt.Println("==================")

	for _, b := range budgets {
		status, _ := store.CheckBudget(b.Scope)
		if status != nil {
			printBudgetStatus(b.Scope, status)
			fmt.Println()
		}
	}

	return nil
}

func printBudgetStatus(scope string, status *cost.BudgetStatus) {
	fmt.Printf("Budget: %s\n", scope)
	fmt.Printf("  Period:    %s\n", status.Budget.Period)
	fmt.Printf("  Limit:     $%.2f\n", status.Budget.LimitUSD)
	fmt.Printf("  Spent:     $%.2f (%.1f%%)\n", status.CurrentSpend, status.PercentUsed*100)
	fmt.Printf("  Remaining: $%.2f\n", status.Remaining)

	if status.IsOverBudget {
		fmt.Println("  Status:    ⚠️  OVER BUDGET")
	} else if status.IsNearLimit {
		fmt.Printf("  Status:    ⚠️  Near limit (alert at %.0f%%)\n", status.Budget.AlertAt*100)
	} else {
		fmt.Println("  Status:    ✓ Within budget")
	}

	if status.Budget.HardStop {
		fmt.Println("  Hard stop: Enabled")
	}
}

func runCostBudgetDelete(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	scope := getBudgetScope()

	if deleteErr := store.DeleteBudget(scope); deleteErr != nil {
		return fmt.Errorf("failed to delete budget: %w", deleteErr)
	}

	fmt.Printf("Budget deleted for %s\n", scope)
	return nil
}

func parseCostDuration(s string) (time.Duration, error) {
	// Support day notation (e.g., "7d" -> 7 * 24h)
	if len(s) > 0 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}

func runCostProject(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	duration, err := parseCostDuration(projectDurationFlag)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", projectDurationFlag, err)
	}

	proj, err := store.ProjectCost(projectLookbackFlag, duration)
	if err != nil {
		return fmt.Errorf("failed to project cost: %w", err)
	}

	if proj.DaysAnalyzed == 0 {
		fmt.Println("No historical cost data available for projection")
		fmt.Println("\nCost data will be available after agents make API calls")
		return nil
	}

	days := duration.Hours() / 24
	fmt.Println("Cost Projection")
	fmt.Println("===============")
	fmt.Printf("  Lookback period:   %d days\n", projectLookbackFlag)
	fmt.Printf("  Days with data:    %d\n", proj.DaysAnalyzed)
	fmt.Printf("  Historical total:  $%.4f\n", proj.TotalHistorical)
	fmt.Printf("  Daily average:     $%.4f/day\n", proj.DailyAvgCost)
	fmt.Println()
	fmt.Printf("  Projection period: %.0f days\n", days)
	fmt.Printf("  Projected cost:    $%.4f\n", proj.ProjectedCost)

	return nil
}

func runCostTrends(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	duration, err := parseCostDuration(trendsSinceFlag)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", trendsSinceFlag, err)
	}

	since := time.Now().Add(-duration)
	dailyCosts, err := store.GetDailyCosts(since)
	if err != nil {
		return fmt.Errorf("failed to get daily costs: %w", err)
	}

	if len(dailyCosts) == 0 {
		fmt.Printf("No cost data for the last %s\n", trendsSinceFlag)
		return nil
	}

	fmt.Printf("Daily Cost Trends (last %s)\n", trendsSinceFlag)
	fmt.Println("===========================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "DATE\tCOST\tCALLS\tTOKENS\tCHANGE")

	var prevCost float64
	var totalCost float64
	for i, dc := range dailyCosts {
		change := ""
		if i > 0 && prevCost > 0 {
			pctChange := ((dc.CostUSD - prevCost) / prevCost) * 100
			if pctChange > 0 {
				change = fmt.Sprintf("+%.1f%%", pctChange)
			} else {
				change = fmt.Sprintf("%.1f%%", pctChange)
			}
		}
		_, _ = fmt.Fprintf(w, "%s\t$%.4f\t%d\t%d\t%s\n",
			dc.Date,
			dc.CostUSD,
			dc.RecordCount,
			dc.TotalTokens,
			change,
		)
		prevCost = dc.CostUSD
		totalCost += dc.CostUSD
	}
	_ = w.Flush()

	fmt.Println()
	fmt.Printf("Total: $%.4f over %d days\n", totalCost, len(dailyCosts))
	if len(dailyCosts) > 0 {
		fmt.Printf("Average: $%.4f/day\n", totalCost/float64(len(dailyCosts)))
	}

	return nil
}

func runCostByAgent(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	duration, err := parseCostDuration(byAgentSinceFlag)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", byAgentSinceFlag, err)
	}

	since := time.Now().Add(-duration)
	summaries, err := store.GetAgentSummarySince(since)
	if err != nil {
		return fmt.Errorf("failed to get agent costs: %w", err)
	}

	if len(summaries) == 0 {
		fmt.Printf("No cost data for the last %s\n", byAgentSinceFlag)
		return nil
	}

	// Calculate total for percentage
	var totalCost float64
	for _, s := range summaries {
		totalCost += s.TotalCostUSD
	}

	fmt.Printf("Cost by Agent (last %s)\n", byAgentSinceFlag)
	fmt.Println("========================")

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "AGENT\tCOST\tCALLS\tTOKENS\t% OF TOTAL")

	for _, s := range summaries {
		pct := 0.0
		if totalCost > 0 {
			pct = (s.TotalCostUSD / totalCost) * 100
		}
		_, _ = fmt.Fprintf(w, "%s\t$%.4f\t%d\t%d\t%.1f%%\n",
			s.AgentID,
			s.TotalCostUSD,
			s.RecordCount,
			s.TotalTokens,
			pct,
		)
	}
	_ = w.Flush()

	fmt.Println()
	fmt.Printf("Total: $%.4f across %d agents\n", totalCost, len(summaries))

	return nil
}
