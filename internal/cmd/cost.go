package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/cost"
)

var costCmd = &cobra.Command{
	Use:     "cost",
	Aliases: []string{"co"},
	Short:   "Show cost information",
	Long: `Commands for viewing API cost information.

Shows Claude Code token usage, costs, and budget management.

Examples:
  bc cost                              # Show cost records (default)
  bc cost show eng-01                  # Show costs for specific agent
  bc cost usage                        # Claude Code usage via ccusage
  bc cost usage --monthly              # Monthly summary
  bc cost budget show                  # Show budget status

See Also:
  bc home           TUI dashboard with cost overview
  bc status         Agent status (includes cost info)`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCostShow,
}

var costShowCmd = &cobra.Command{
	Use:   "show [agent]",
	Short: "Show cost records",
	Long: `Show cost records, optionally filtered by agent.

You can specify the agent either as a positional argument or using --agent flag.

Examples:
  bc cost show
  bc cost show engineer-01
  bc cost show --agent engineer-01`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCostShow,
}

var (
	costShowAgentFlag string
	costLimitFlag     int
	costOffsetFlag    int
)

func init() {
	costShowCmd.Flags().IntVarP(&costLimitFlag, "limit", "n", 20, "Number of records to show")
	costShowCmd.Flags().IntVar(&costOffsetFlag, "offset", 0, "Number of records to skip (for pagination)")
	costShowCmd.Flags().StringVar(&costShowAgentFlag, "agent", "", "Filter by agent (alternative to positional argument)")

	// Budget flags (in cost_budget.go)
	initCostBudgetFlags()

	// Usage flags (in cost_usage.go)
	initCostUsageFlags()

	costCmd.AddCommand(costShowCmd)
	costCmd.AddCommand(costBudgetCmd)
	costCmd.AddCommand(costUsageCmd)
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

// ccusageRunner runs ccusage and returns raw JSON output. Overridable for testing.
var ccusageRunner = defaultCCUsageRunner

func defaultCCUsageRunner(ctx context.Context) ([]byte, error) {
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return nil, err
	}
	ccCmd := exec.CommandContext(ctx, npxPath, "ccusage@latest", "--json") //nolint:gosec // args are static
	ccCmd.Stderr = os.Stderr
	return ccCmd.Output()
}

// fetchCCUsageDailyReport calls ccusage and returns the daily report, or nil if unavailable.
func fetchCCUsageDailyReport(ctx context.Context) *ccusageDailyReport {
	output, err := ccusageRunner(ctx)
	if err != nil {
		return nil
	}
	var report ccusageDailyReport
	if unmarshalErr := json.Unmarshal(output, &report); unmarshalErr != nil {
		return nil
	}
	return &report
}

// costShowResponse is the enriched JSON response for 'bc cost show --json'.
type costShowResponse struct {
	ByAgent            map[string]float64 `json:"by_agent"`
	ByTeam             map[string]float64 `json:"by_team"`
	ByModel            map[string]float64 `json:"by_model"`
	CacheHitRate       *float64           `json:"cache_hit_rate,omitempty"`
	BurnRate           *float64           `json:"burn_rate,omitempty"`
	ProjectedTotal     *float64           `json:"projected_total,omitempty"`
	BillingWindowSpent *float64           `json:"billing_window_spent,omitempty"`
	TotalInputTokens   int64              `json:"total_input_tokens"`
	TotalOutputTokens  int64              `json:"total_output_tokens"`
	TotalCost          float64            `json:"total_cost"`
}

// enrichWithCCUsage merges ccusage daily report data into the cost show response.
func enrichWithCCUsage(resp *costShowResponse, report *ccusageDailyReport) {
	if report == nil {
		return
	}

	totals := report.Totals

	// Override totals from ccusage if internal DB had no data
	if resp.TotalCost == 0 && totals.TotalCost > 0 {
		resp.TotalCost = totals.TotalCost
		resp.TotalInputTokens = totals.InputTokens
		resp.TotalOutputTokens = totals.OutputTokens
	}

	// cache_hit_rate: cacheRead / (cacheRead + cacheCreation)
	cacheTotal := totals.CacheReadTokens + totals.CacheCreationTokens
	if cacheTotal > 0 {
		rate := float64(totals.CacheReadTokens) / float64(cacheTotal)
		resp.CacheHitRate = &rate
	}

	// burn_rate: average daily cost from ccusage daily entries
	if len(report.Daily) > 0 {
		burnRate := totals.TotalCost / float64(len(report.Daily))
		resp.BurnRate = &burnRate

		// projected_total: burn_rate * days_in_current_month
		now := time.Now()
		daysInMonth := float64(time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day())
		projected := burnRate * daysInMonth
		resp.ProjectedTotal = &projected
	}

	// billing_window_spent: total cost from ccusage
	if totals.TotalCost > 0 {
		spent := totals.TotalCost
		resp.BillingWindowSpent = &spent
	}

	// Enrich by_model from ccusage daily modelsUsed (frequency count, no cost attribution)
	if len(resp.ByModel) == 0 {
		modelSeen := make(map[string]bool)
		for _, d := range report.Daily {
			for _, m := range d.ModelsUsed {
				modelSeen[m] = true
			}
		}
		// Add models with zero cost — signals to TUI which models are in use
		for m := range modelSeen {
			resp.ByModel[m] = 0
		}
	}
}

func runCostShow(cmd *cobra.Command, args []string) error {
	// Validate limit parameter
	if cmd.Flags().Changed("limit") && costLimitFlag <= 0 {
		return fmt.Errorf("limit must be a positive number")
	}
	if cmd.Flags().Changed("offset") && costOffsetFlag < 0 {
		return fmt.Errorf("offset must be non-negative")
	}

	store, err := getCostStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	// Determine agent filter: --agent flag takes precedence, then positional arg
	agentID := costShowAgentFlag
	if agentID == "" && len(args) > 0 {
		agentID = args[0]
	}

	// Get total count for pagination hint
	var totalCount int64
	if agentID != "" {
		summary, summaryErr := store.AgentSummary(agentID)
		if summaryErr == nil && summary != nil {
			totalCount = summary.RecordCount
		}
	} else {
		summary, summaryErr := store.WorkspaceSummary()
		if summaryErr == nil && summary != nil {
			totalCount = summary.RecordCount
		}
	}

	var records []*cost.Record
	if agentID != "" {
		records, err = store.GetByAgentWithOffset(agentID, costLimitFlag, costOffsetFlag)
	} else {
		records, err = store.GetAllWithOffset(costLimitFlag, costOffsetFlag)
	}

	if err != nil {
		return fmt.Errorf("failed to get cost records: %w", err)
	}

	// Check for JSON output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Build a summary for TUI compatibility
		var totalCost, totalInput, totalOutput float64
		byAgent := make(map[string]float64)
		byTeam := make(map[string]float64)
		byModel := make(map[string]float64)

		for _, r := range records {
			totalCost += r.CostUSD
			totalInput += float64(r.InputTokens)
			totalOutput += float64(r.OutputTokens)
			byAgent[r.AgentID] += r.CostUSD
			byTeam[r.TeamID] += r.CostUSD
			byModel[r.Model] += r.CostUSD
		}

		response := &costShowResponse{
			ByAgent:           byAgent,
			ByTeam:            byTeam,
			ByModel:           byModel,
			TotalInputTokens:  int64(totalInput),
			TotalOutputTokens: int64(totalOutput),
			TotalCost:         totalCost,
		}

		// Enrich with ccusage data (graceful — nil if unavailable)
		ccReport := fetchCCUsageDailyReport(cmd.Context())
		enrichWithCCUsage(response, ccReport)

		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
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

	if flushErr := w.Flush(); flushErr != nil {
		return flushErr
	}

	// Show pagination hint if there are more records than displayed
	if totalCount > 0 {
		startIdx := costOffsetFlag + 1
		endIdx := costOffsetFlag + len(records)
		if int64(endIdx) < totalCount {
			fmt.Printf("\nShowing %d-%d of %d entries. Use --limit and --offset for more.\n", startIdx, endIdx, totalCount)
		} else if costOffsetFlag > 0 {
			// Show count when using offset even if we've reached the end
			fmt.Printf("\nShowing %d-%d of %d entries.\n", startIdx, endIdx, totalCount)
		}
	}

	return nil
}
