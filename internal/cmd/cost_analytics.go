package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/ui"
)

var costSummaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show workspace cost overview",
	Long: `Show cost summary with today, this week, this month, and all-time totals.

Examples:
  bc cost summary
  bc cost summary --json`,
	RunE: runCostSummary,
}

var costAgentCmd = &cobra.Command{
	Use:   "agent [name]",
	Short: "Show per-agent cost breakdown",
	Long: `Show cost breakdown by agent. If a name is given, shows detail for that agent.

Examples:
  bc cost agent                    # All agents
  bc cost agent swift-falcon       # Specific agent`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCostAgent,
}

var costModelCmd = &cobra.Command{
	Use:   "model [name]",
	Short: "Show per-model cost breakdown",
	Long: `Show cost breakdown by model.

Examples:
  bc cost model
  bc cost model claude-sonnet-4-6`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCostModel,
}

var costDailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Show daily cost totals",
	Long: `Show daily cost totals for the last N days.

Examples:
  bc cost daily              # Last 30 days (default)
  bc cost daily --days 7     # Last 7 days
  bc cost daily --json`,
	RunE: runCostDaily,
}

var costDashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Show rich cost dashboard",
	Long: `Show a rich formatted cost dashboard with summary, per-agent breakdown,
per-model breakdown, and budget status.

Examples:
  bc cost dashboard
  bc cost dashboard --json`,
	RunE: runCostDashboard,
}

var costDailyDaysFlag int

func initCostAnalyticsFlags() {
	costDailyCmd.Flags().IntVar(&costDailyDaysFlag, "days", 30, "Number of days to show")

	costCmd.AddCommand(costSummaryCmd)
	costCmd.AddCommand(costAgentCmd)
	costCmd.AddCommand(costModelCmd)
	costCmd.AddCommand(costDailyCmd)
	costCmd.AddCommand(costDashboardCmd)
}

func runCostSummary(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := todayStart.AddDate(0, 0, -int(now.Weekday()))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	today, err := store.GetSummarySince(todayStart)
	if err != nil {
		return err
	}
	week, err := store.GetSummarySince(weekStart)
	if err != nil {
		return err
	}
	month, err := store.GetSummarySince(monthStart)
	if err != nil {
		return err
	}
	allTime, err := store.WorkspaceSummary()
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		response := struct {
			TodayCost    float64 `json:"today_cost"`
			WeekCost     float64 `json:"week_cost"`
			MonthCost    float64 `json:"month_cost"`
			AllTimeCost  float64 `json:"all_time_cost"`
			TotalRecords int64   `json:"total_records"`
			TotalTokens  int64   `json:"total_tokens"`
		}{
			TodayCost:    today.TotalCostUSD,
			WeekCost:     week.TotalCostUSD,
			MonthCost:    month.TotalCostUSD,
			AllTimeCost:  allTime.TotalCostUSD,
			TotalRecords: allTime.RecordCount,
			TotalTokens:  allTime.TotalTokens,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

	fmt.Println("Cost Summary")
	fmt.Println("============")
	ui.SimpleTable(
		"Today", fmt.Sprintf("$%.4f", today.TotalCostUSD),
		"This Week", fmt.Sprintf("$%.4f", week.TotalCostUSD),
		"This Month", fmt.Sprintf("$%.4f", month.TotalCostUSD),
		"All Time", fmt.Sprintf("$%.4f", allTime.TotalCostUSD),
		"Total Records", fmt.Sprintf("%d", allTime.RecordCount),
		"Total Tokens", fmt.Sprintf("%d", allTime.TotalTokens),
	)
	return nil
}

func runCostAgent(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	jsonOutput, _ := cmd.Flags().GetBool("json")

	if len(args) > 0 {
		// Show specific agent
		agentID := args[0]
		summary, err := store.AgentSummary(agentID)
		if err != nil {
			return err
		}

		if jsonOutput {
			enc := json.NewEncoder(cmd.OutOrStdout())
			enc.SetIndent("", "  ")
			return enc.Encode(summary)
		}

		if summary.RecordCount == 0 {
			fmt.Printf("No cost records for agent %q\n", agentID)
			return nil
		}

		fmt.Printf("Cost: %s\n", agentID)
		fmt.Println("============")
		ui.SimpleTable(
			"Total Cost", fmt.Sprintf("$%.4f", summary.TotalCostUSD),
			"Input Tokens", fmt.Sprintf("%d", summary.InputTokens),
			"Output Tokens", fmt.Sprintf("%d", summary.OutputTokens),
			"Records", fmt.Sprintf("%d", summary.RecordCount),
		)
		return nil
	}

	// Show all agents
	summaries, err := store.SummaryByAgent()
	if err != nil {
		return err
	}

	if jsonOutput {
		response := struct {
			Agents []*costAgentSummary `json:"agents"`
		}{Agents: toAgentSummaries(summaries)}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

	if len(summaries) == 0 {
		fmt.Println("No cost records found")
		return nil
	}

	table := ui.NewTable("AGENT", "COST", "INPUT", "OUTPUT", "RECORDS")
	for _, s := range summaries {
		table.AddRow(
			s.AgentID,
			fmt.Sprintf("$%.4f", s.TotalCostUSD),
			fmt.Sprintf("%d", s.InputTokens),
			fmt.Sprintf("%d", s.OutputTokens),
			fmt.Sprintf("%d", s.RecordCount),
		)
	}
	table.Print()
	return nil
}

type costAgentSummary struct {
	AgentID      string  `json:"agent_id"`
	TotalCost    float64 `json:"total_cost"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	RecordCount  int64   `json:"record_count"`
}

func toAgentSummaries(summaries []*cost.Summary) []*costAgentSummary {
	result := make([]*costAgentSummary, len(summaries))
	for i, s := range summaries {
		result[i] = &costAgentSummary{
			AgentID:      s.AgentID,
			TotalCost:    s.TotalCostUSD,
			InputTokens:  s.InputTokens,
			OutputTokens: s.OutputTokens,
			RecordCount:  s.RecordCount,
		}
	}
	return result
}

func runCostModel(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	summaries, err := store.SummaryByModel()
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Filter to specific model if given
	if len(args) > 0 {
		modelName := args[0]
		var filtered []*cost.Summary
		for _, s := range summaries {
			if s.Model == modelName {
				filtered = append(filtered, s)
			}
		}
		summaries = filtered
	}

	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Models []*cost.Summary `json:"models"`
		}{Models: summaries})
	}

	if len(summaries) == 0 {
		fmt.Println("No cost records found")
		return nil
	}

	table := ui.NewTable("MODEL", "COST", "INPUT", "OUTPUT", "RECORDS")
	for _, s := range summaries {
		table.AddRow(
			s.Model,
			fmt.Sprintf("$%.4f", s.TotalCostUSD),
			fmt.Sprintf("%d", s.InputTokens),
			fmt.Sprintf("%d", s.OutputTokens),
			fmt.Sprintf("%d", s.RecordCount),
		)
	}
	table.Print()
	return nil
}

func runCostDaily(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	since := time.Now().AddDate(0, 0, -costDailyDaysFlag)
	dailyCosts, err := store.GetDailyCosts(since)
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(struct {
			Daily any `json:"daily"`
			Days  int `json:"days"`
		}{Daily: dailyCosts, Days: costDailyDaysFlag})
	}

	if len(dailyCosts) == 0 {
		fmt.Printf("No cost records in the last %d days\n", costDailyDaysFlag)
		return nil
	}

	table := ui.NewTable("DATE", "COST", "TOKENS", "RECORDS")
	for _, dc := range dailyCosts {
		table.AddRow(
			dc.Date,
			fmt.Sprintf("$%.4f", dc.CostUSD),
			fmt.Sprintf("%d", dc.TotalTokens),
			fmt.Sprintf("%d", dc.RecordCount),
		)
	}
	table.Print()
	return nil
}

func runCostDashboard(cmd *cobra.Command, args []string) error {
	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	now := time.Now().UTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	today, err := store.GetSummarySince(todayStart)
	if err != nil {
		return err
	}
	month, err := store.GetSummarySince(monthStart)
	if err != nil {
		return err
	}
	allTime, err := store.WorkspaceSummary()
	if err != nil {
		return err
	}
	agentSummaries, err := store.SummaryByAgent()
	if err != nil {
		return err
	}
	modelSummaries, err := store.SummaryByModel()
	if err != nil {
		return err
	}
	budgets, err := store.GetAllBudgets()
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		response := struct {
			TodayCost   float64             `json:"today_cost"`
			MonthCost   float64             `json:"month_cost"`
			AllTime     float64             `json:"all_time_cost"`
			ByAgent     []*costAgentSummary `json:"by_agent"`
			ByModel     []*cost.Summary     `json:"by_model"`
			BudgetCount int                 `json:"budget_count"`
		}{
			TodayCost:   today.TotalCostUSD,
			MonthCost:   month.TotalCostUSD,
			AllTime:     allTime.TotalCostUSD,
			ByAgent:     toAgentSummaries(agentSummaries),
			ByModel:     modelSummaries,
			BudgetCount: len(budgets),
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

	// Summary
	fmt.Println("Cost Dashboard")
	fmt.Println("==============")
	ui.SimpleTable(
		"Today", fmt.Sprintf("$%.4f", today.TotalCostUSD),
		"This Month", fmt.Sprintf("$%.4f", month.TotalCostUSD),
		"All Time", fmt.Sprintf("$%.4f", allTime.TotalCostUSD),
	)

	// Per-agent
	if len(agentSummaries) > 0 {
		fmt.Println("\nBy Agent")
		fmt.Println("--------")
		table := ui.NewTable("AGENT", "COST", "TOKENS")
		for _, s := range agentSummaries {
			table.AddRow(s.AgentID, fmt.Sprintf("$%.4f", s.TotalCostUSD), fmt.Sprintf("%d", s.TotalTokens))
		}
		table.Print()
	}

	// Per-model
	if len(modelSummaries) > 0 {
		fmt.Println("\nBy Model")
		fmt.Println("--------")
		table := ui.NewTable("MODEL", "COST", "TOKENS")
		for _, s := range modelSummaries {
			table.AddRow(s.Model, fmt.Sprintf("$%.4f", s.TotalCostUSD), fmt.Sprintf("%d", s.TotalTokens))
		}
		table.Print()
	}

	// Budget status
	if len(budgets) > 0 {
		fmt.Println("\nBudgets")
		fmt.Println("-------")
		for _, b := range budgets {
			status, _ := store.CheckBudget(b.Scope)
			if status != nil {
				printBudgetStatus(b.Scope, status)
			}
		}
	}

	return nil
}
