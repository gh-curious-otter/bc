package cmd

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/cost"
)

// Cost command tests use executeIntegrationCmd which captures os.Stdout
// because cost.go uses fmt.Printf directly rather than cmd.OutOrStdout()

// resetCostFlags resets the cost command flags between tests
func resetCostFlags() {
	costTeamFlag = ""
	costAgentFlag = ""
	costWorkspaceFlag = false
	costModelFlag = false
	costLimitFlag = 20
}

func TestCostShowEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("cost", "show")
	if err != nil {
		t.Fatalf("cost show failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No cost records found") {
		t.Errorf("expected 'No cost records found', got: %s", stdout)
	}
}

func TestCostShowWithRecords(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create cost records
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, err := store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	if err != nil {
		t.Fatalf("failed to record cost: %v", err)
	}
	_, err = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 0.03)
	if err != nil {
		t.Fatalf("failed to record cost: %v", err)
	}
	_ = store.Close()

	stdout, _, cmdErr := executeIntegrationCmd("cost", "show")
	if cmdErr != nil {
		t.Fatalf("cost show failed: %v\nOutput: %s", cmdErr, stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("output should contain engineer-01: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-02") {
		t.Errorf("output should contain engineer-02: %s", stdout)
	}
	if !strings.Contains(stdout, "claude-3-opus") {
		t.Errorf("output should contain model: %s", stdout)
	}
}

func TestCostShowByAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "show", "engineer-01")
	if err != nil {
		t.Fatalf("cost show agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("output should contain engineer-01: %s", stdout)
	}
	// Should not contain records from other agents
	if strings.Contains(stdout, "engineer-02") {
		t.Errorf("output should not contain engineer-02: %s", stdout)
	}
}

func TestCostShowLimit(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	// Create 5 records
	for i := 0; i < 5; i++ {
		_, _ = store.Record("engineer-01", "", "claude-3-opus", int64(1000+i*100), 500, 0.05)
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "show", "--limit", "2")
	if err != nil {
		t.Fatalf("cost show --limit failed: %v\nOutput: %s", err, stdout)
	}

	// Count lines with engineer-01 (excluding header)
	lines := strings.Split(stdout, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "engineer-01") {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 2 records with --limit 2, got %d\nOutput: %s", count, stdout)
	}
}

func TestCostSummaryEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("cost", "summary")
	if err != nil {
		t.Fatalf("cost summary failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Workspace Summary") {
		t.Errorf("expected 'Workspace Summary', got: %s", stdout)
	}
	if !strings.Contains(stdout, "API Calls:") {
		t.Errorf("expected 'API Calls:' header, got: %s", stdout)
	}
}

func TestCostSummaryWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, _ = store.Record("engineer-01", "engineering", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "engineering", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--workspace")
	if err != nil {
		t.Fatalf("cost summary --workspace failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Workspace Summary") {
		t.Errorf("expected 'Workspace Summary', got: %s", stdout)
	}
	if !strings.Contains(stdout, "Total Cost:") {
		t.Errorf("expected 'Total Cost:', got: %s", stdout)
	}
}

func TestCostSummaryByAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 2000, 1000, 0.08)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("cost summary --agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Agent:") {
		t.Errorf("expected 'Agent:' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("expected agent name in output: %s", stdout)
	}
}

func TestCostSummaryByTeam(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, _ = store.Record("engineer-01", "engineering", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "engineering", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--team", "engineering")
	if err != nil {
		t.Fatalf("cost summary --team failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Team:") {
		t.Errorf("expected 'Team:' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineering") {
		t.Errorf("expected team name in output: %s", stdout)
	}
}

func TestCostSummaryByModel(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--model")
	if err != nil {
		t.Fatalf("cost summary --model failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Cost by Model") {
		t.Errorf("expected 'Cost by Model' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "claude-3-opus") {
		t.Errorf("expected opus model in output: %s", stdout)
	}
	if !strings.Contains(stdout, "claude-3-sonnet") {
		t.Errorf("expected sonnet model in output: %s", stdout)
	}
}

func TestCostNoWorkspace(t *testing.T) {
	// Run outside workspace - setup temp dir without bc workspace
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, execErr := executeIntegrationCmd("cost", "show")
	if execErr == nil {
		t.Error("expected error when not in workspace")
	}
	if !strings.Contains(execErr.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", execErr)
	}
}

// TestCostAddBasic tests manual cost entry with amount only
func TestCostAddBasic(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-01",
		"--amount", "0.50",
	)
	if err != nil {
		t.Fatalf("cost add failed: %v\nOutput: %s", err, stdout)
	}

	if !strings.Contains(stdout, "Cost recorded") {
		t.Errorf("expected 'Cost recorded', got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("expected agent name in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "0.5000") {
		t.Errorf("expected cost amount in output, got: %s", stdout)
	}

	// Verify it was actually recorded
	store := cost.NewStore(wsDir)
	storeErr := store.Open()
	if storeErr != nil {
		t.Fatalf("failed to open store: %v", storeErr)
	}
	defer func() { _ = store.Close() }()

	records, err := store.GetByAgent("engineer-01", 10)
	if err != nil {
		t.Fatalf("failed to get records: %v", err)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	if records[0].CostUSD != 0.50 {
		t.Errorf("cost = %f, want 0.50", records[0].CostUSD)
	}
}

// TestCostAddWithTokens tests cost entry with token counts
func TestCostAddWithTokens(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-02",
		"--tokens-in", "5000",
		"--tokens-out", "2000",
		"--amount", "0.35",
		"--model", "claude-3-opus",
	)
	if err != nil {
		t.Fatalf("cost add with tokens failed: %v\nOutput: %s", err, stdout)
	}

	if !strings.Contains(stdout, "5000") && !strings.Contains(stdout, "input") {
		t.Errorf("expected token info in output, got: %s", stdout)
	}

	// Verify tokens were recorded
	store := cost.NewStore(wsDir)
	storeErr := store.Open()
	if storeErr != nil {
		t.Fatalf("failed to open store: %v", storeErr)
	}
	defer func() { _ = store.Close() }()

	records, getErr := store.GetByAgent("engineer-02", 10)
	if getErr != nil {
		t.Fatalf("failed to get records: %v", getErr)
	}
	if len(records) != 1 {
		t.Errorf("expected 1 record, got %d", len(records))
	}
	if records[0].InputTokens != 5000 {
		t.Errorf("input tokens = %d, want 5000", records[0].InputTokens)
	}
	if records[0].OutputTokens != 2000 {
		t.Errorf("output tokens = %d, want 2000", records[0].OutputTokens)
	}
}

// TestCostAddMissingAgent tests error when --agent is not provided
func TestCostAddMissingAgent(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--amount", "0.50",
	)
	if err == nil {
		t.Fatalf("expected error when --agent flag missing, got output: %s", stdout)
	}
}

// TestCostAddMissingAmount tests error when neither --amount nor tokens provided
func TestCostAddMissingAmount(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-01",
	)
	if err == nil {
		t.Fatalf("expected error when no cost info provided, got output: %s", stdout)
	}
}

// TestCostAddTokensWithoutAmount tests error when tokens provided without amount
func TestCostAddTokensWithoutAmount(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-01",
		"--tokens-in", "5000",
	)
	if err == nil {
		t.Fatalf("expected error when tokens provided without amount, got output: %s", stdout)
	}
}

// TestCostPeekMissingFlags tests error when neither --agent nor --workspace provided
func TestCostPeekMissingFlags(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	peekAgentFlag = ""
	peekWorkspaceFlag = false
	peekIntervalFlag = 5

	_, _, err := executeIntegrationCmd("cost", "peek")
	if err == nil {
		t.Fatal("expected error when no flags provided")
	}
}

// TestCostPeekBothFlags tests error when both --agent and --workspace provided
func TestCostPeekBothFlags(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	peekAgentFlag = ""
	peekWorkspaceFlag = false
	peekIntervalFlag = 5

	_, _, err := executeIntegrationCmd(
		"cost", "peek",
		"--agent", "engineer-01",
		"--workspace",
	)
	if err == nil {
		t.Fatal("expected error when both flags provided")
	}
}

// TestCostShowNegativeLimit tests that negative limits are rejected
func TestCostShowNegativeLimit(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	_, _, err := executeIntegrationCmd("cost", "show", "--limit", "-5")
	if err == nil {
		t.Error("expected error for negative limit")
	}
	if !strings.Contains(err.Error(), "must be a positive number") {
		t.Errorf("error should mention positive number: %v", err)
	}
}

// TestCostShowZeroLimit tests that zero limit is rejected
func TestCostShowZeroLimit(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	_, _, err := executeIntegrationCmd("cost", "show", "--limit", "0")
	if err == nil {
		t.Error("expected error for zero limit")
	}
	if !strings.Contains(err.Error(), "must be a positive number") {
		t.Errorf("error should mention positive number: %v", err)
	}
}

// =============================================================================
// Dashboard Command Tests (Additional)
// =============================================================================
// Note: Basic dashboard tests are in cmd_integration_test.go

// TestCostDashboardShowsPercentages tests that dashboard shows percentage breakdown
func TestCostDashboardShowsPercentages(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}

	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.80)
	_, _ = store.Record("engineer-02", "", "claude-3-opus", 1000, 500, 0.20)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("cost dashboard failed: %v\nOutput: %s", err, stdout)
	}
	// Should show percentage of total
	if !strings.Contains(stdout, "% OF TOTAL") {
		t.Errorf("expected percentage column, got: %s", stdout)
	}
}

// =============================================================================
// Budget Set Command Tests
// =============================================================================

// resetBudgetFlags resets budget-related flags between tests
func resetBudgetFlags() {
	budgetAgentFlag = ""
	budgetTeamFlag = ""
	budgetPeriodFlag = "monthly"
	budgetAlertAtFlag = 0.8
	budgetHardStop = false
}

// TestCostBudgetSetWorkspace tests setting a workspace budget
func TestCostBudgetSetWorkspace(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00")
	if err != nil {
		t.Fatalf("cost budget set failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Budget set for workspace") {
		t.Errorf("expected workspace budget confirmation, got: %s", stdout)
	}
	if !strings.Contains(stdout, "100.00") {
		t.Errorf("expected amount in output, got: %s", stdout)
	}
}

// TestCostBudgetSetAgent tests setting an agent-specific budget
func TestCostBudgetSetAgent(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "50.00", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("cost budget set --agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "agent:engineer-01") {
		t.Errorf("expected agent scope in output, got: %s", stdout)
	}
}

// TestCostBudgetSetTeam tests setting a team-specific budget
func TestCostBudgetSetTeam(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "200.00", "--team", "engineering")
	if err != nil {
		t.Fatalf("cost budget set --team failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "team:engineering") {
		t.Errorf("expected team scope in output, got: %s", stdout)
	}
}

// TestCostBudgetSetPeriods tests all budget period options
func TestCostBudgetSetPeriods(t *testing.T) {
	tests := []struct {
		name   string
		period string
	}{
		{"daily", "daily"},
		{"weekly", "weekly"},
		{"monthly", "monthly"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupIntegrationWorkspace(t)
			defer cleanup()
			resetBudgetFlags()
			defer resetBudgetFlags()

			stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00", "--period", tt.period)
			if err != nil {
				t.Fatalf("cost budget set --period %s failed: %v\nOutput: %s", tt.period, err, stdout)
			}
			if !strings.Contains(stdout, tt.period) {
				t.Errorf("expected period %s in output, got: %s", tt.period, stdout)
			}
		})
	}
}

// TestCostBudgetSetInvalidPeriod tests rejection of invalid periods
func TestCostBudgetSetInvalidPeriod(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00", "--period", "yearly")
	if err == nil {
		t.Fatal("expected error for invalid period")
	}
	if !strings.Contains(err.Error(), "invalid period") {
		t.Errorf("expected 'invalid period' error, got: %v", err)
	}
}

// TestCostBudgetSetAlertAt tests custom alert threshold
func TestCostBudgetSetAlertAt(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00", "--alert-at", "0.9")
	if err != nil {
		t.Fatalf("cost budget set --alert-at failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "90%") {
		t.Errorf("expected 90%% alert in output, got: %s", stdout)
	}
}

// TestCostBudgetSetAlertAtInvalid tests rejection of invalid alert thresholds
func TestCostBudgetSetAlertAtInvalid(t *testing.T) {
	tests := []struct {
		name    string
		alertAt string
	}{
		{"negative", "-0.1"},
		{"over_one", "1.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := setupIntegrationWorkspace(t)
			defer cleanup()
			resetBudgetFlags()
			defer resetBudgetFlags()

			_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00", "--alert-at", tt.alertAt)
			if err == nil {
				t.Fatal("expected error for invalid alert-at")
			}
			if !strings.Contains(err.Error(), "alert-at must be between") {
				t.Errorf("expected range error, got: %v", err)
			}
		})
	}
}

// TestCostBudgetSetHardStop tests hard-stop flag
func TestCostBudgetSetHardStop(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00", "--hard-stop")
	if err != nil {
		t.Fatalf("cost budget set --hard-stop failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Hard stop: true") {
		t.Errorf("expected hard-stop confirmation, got: %s", stdout)
	}
}

// TestCostBudgetSetZeroAmount tests rejection of zero amount
func TestCostBudgetSetZeroAmount(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	_, _, err := executeIntegrationCmd("cost", "budget", "set", "0")
	if err == nil {
		t.Fatal("expected error for zero budget")
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("expected positive amount error, got: %v", err)
	}
}

// TestCostBudgetSetNegativeAmount tests rejection of negative amount
// Note: CLI interprets "-50.00" as a flag, so this test expects a flag parsing error
// which is correct behavior - negative amounts are rejected at the CLI level
func TestCostBudgetSetNegativeAmount(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Using "--" to force argument interpretation doesn't work for Cobra's positional args
	// The CLI correctly rejects negative numbers that look like flags
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "-50.00")
	if err == nil {
		t.Fatal("expected error for negative budget")
	}
	// Accept either flag parsing error or positive amount error
	errStr := err.Error()
	if !strings.Contains(errStr, "must be positive") && !strings.Contains(errStr, "unknown shorthand flag") {
		t.Errorf("expected positive amount or flag error, got: %v", err)
	}
}

// TestCostBudgetSetInvalidAmount tests rejection of non-numeric amount
func TestCostBudgetSetInvalidAmount(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	_, _, err := executeIntegrationCmd("cost", "budget", "set", "abc")
	if err == nil {
		t.Fatal("expected error for invalid amount")
	}
	if !strings.Contains(err.Error(), "invalid amount") {
		t.Errorf("expected 'invalid amount' error, got: %v", err)
	}
}

// =============================================================================
// Budget Show Command Tests
// =============================================================================

// TestCostBudgetShowNoBudgets tests show when no budgets configured
func TestCostBudgetShowNoBudgets(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	stdout, _, err := executeIntegrationCmd("cost", "budget", "show")
	if err != nil {
		t.Fatalf("cost budget show failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No budget") {
		t.Errorf("expected 'No budget' message, got: %s", stdout)
	}
}

// TestCostBudgetShowWorkspace tests showing workspace budget
func TestCostBudgetShowWorkspace(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// First set a budget
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00")
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	resetBudgetFlags()
	stdout, _, err := executeIntegrationCmd("cost", "budget", "show")
	if err != nil {
		t.Fatalf("cost budget show failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "workspace") {
		t.Errorf("expected workspace budget, got: %s", stdout)
	}
	if !strings.Contains(stdout, "100.00") {
		t.Errorf("expected budget amount, got: %s", stdout)
	}
}

// TestCostBudgetShowWithSpending tests budget status with spending
func TestCostBudgetShowWithSpending(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set a budget
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00")
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	// Add some spending
	store := cost.NewStore(wsDir)
	if err = store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 50.00)
	_ = store.Close()

	resetBudgetFlags()
	stdout, _, showErr := executeIntegrationCmd("cost", "budget", "show")
	if showErr != nil {
		t.Fatalf("cost budget show failed: %v\nOutput: %s", showErr, stdout)
	}
	if !strings.Contains(stdout, "Spent:") {
		t.Errorf("expected spending info, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Remaining:") {
		t.Errorf("expected remaining info, got: %s", stdout)
	}
}

// TestCostBudgetShowNearLimit tests warning when near budget limit
func TestCostBudgetShowNearLimit(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set a budget with 80% alert threshold
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00", "--alert-at", "0.8")
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	// Add spending at 85%
	store := cost.NewStore(wsDir)
	if err = store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 85.00)
	_ = store.Close()

	resetBudgetFlags()
	stdout, _, showErr := executeIntegrationCmd("cost", "budget", "show")
	if showErr != nil {
		t.Fatalf("cost budget show failed: %v\nOutput: %s", showErr, stdout)
	}
	if !strings.Contains(stdout, "Near limit") {
		t.Errorf("expected near limit warning, got: %s", stdout)
	}
}

// TestCostBudgetShowOverBudget tests warning when over budget
func TestCostBudgetShowOverBudget(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set a small budget
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "50.00")
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	// Add spending over budget
	store := cost.NewStore(wsDir)
	if err = store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 75.00)
	_ = store.Close()

	resetBudgetFlags()
	stdout, _, showErr := executeIntegrationCmd("cost", "budget", "show")
	if showErr != nil {
		t.Fatalf("cost budget show failed: %v\nOutput: %s", showErr, stdout)
	}
	if !strings.Contains(stdout, "OVER BUDGET") {
		t.Errorf("expected over budget warning, got: %s", stdout)
	}
}

// =============================================================================
// Budget Delete Command Tests
// =============================================================================

// TestCostBudgetDelete tests deleting a workspace budget
func TestCostBudgetDelete(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set a budget first
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00")
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	resetBudgetFlags()
	stdout, _, err := executeIntegrationCmd("cost", "budget", "delete")
	if err != nil {
		t.Fatalf("cost budget delete failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Budget deleted") {
		t.Errorf("expected deletion confirmation, got: %s", stdout)
	}
}

// TestCostBudgetDeleteAgent tests deleting an agent budget
func TestCostBudgetDeleteAgent(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set an agent budget first
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "50.00", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("failed to set budget: %v", err)
	}

	resetBudgetFlags()
	stdout, _, err := executeIntegrationCmd("cost", "budget", "delete", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("cost budget delete --agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "agent:engineer-01") {
		t.Errorf("expected agent scope in output, got: %s", stdout)
	}
}

// =============================================================================
// Project Command Tests
// =============================================================================

// resetProjectFlags resets projection-related flags
func resetProjectFlags() {
	projectDurationFlag = "7d"
	projectLookbackFlag = 7
}

// TestCostProjectEmpty tests projection with no historical data
func TestCostProjectEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProjectFlags()
	defer resetProjectFlags()

	stdout, _, err := executeIntegrationCmd("cost", "project")
	if err != nil {
		t.Fatalf("cost project failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No historical cost data") {
		t.Errorf("expected no data message, got: %s", stdout)
	}
}

// TestCostProjectWithData tests projection with historical data
func TestCostProjectWithData(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProjectFlags()
	defer resetProjectFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	// Add some historical data
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 10.00)
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 2000, 1000, 15.00)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "project", "--duration", "7d")
	if err != nil {
		t.Fatalf("cost project failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Cost Projection") {
		t.Errorf("expected 'Cost Projection' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Projected cost:") {
		t.Errorf("expected projected cost, got: %s", stdout)
	}
}

// TestCostProjectDurations tests various duration formats
func TestCostProjectDurations(t *testing.T) {
	tests := []struct {
		name     string
		duration string
	}{
		{"one day", "1d"},
		{"seven days", "7d"},
		{"thirty days", "30d"},
		{"hours", "24h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsDir, cleanup := setupIntegrationWorkspace(t)
			defer cleanup()
			resetProjectFlags()
			defer resetProjectFlags()

			store := cost.NewStore(wsDir)
			if err := store.Open(); err != nil {
				t.Fatalf("failed to open store: %v", err)
			}
			_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 10.00)
			_ = store.Close()

			stdout, _, err := executeIntegrationCmd("cost", "project", "--duration", tt.duration)
			if err != nil {
				t.Fatalf("cost project --duration %s failed: %v\nOutput: %s", tt.duration, err, stdout)
			}
		})
	}
}

// TestCostProjectInvalidDuration tests rejection of invalid duration
func TestCostProjectInvalidDuration(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProjectFlags()
	defer resetProjectFlags()

	_, _, err := executeIntegrationCmd("cost", "project", "--duration", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
	if !strings.Contains(err.Error(), "invalid duration") {
		t.Errorf("expected 'invalid duration' error, got: %v", err)
	}
}

// TestCostProjectLookback tests custom lookback period
func TestCostProjectLookback(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProjectFlags()
	defer resetProjectFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 10.00)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "project", "--lookback", "14")
	if err != nil {
		t.Fatalf("cost project --lookback failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "14 days") {
		t.Errorf("expected 14 days lookback, got: %s", stdout)
	}
}

// =============================================================================
// Trends Command Tests
// =============================================================================

// resetTrendsFlags resets trends-related flags
func resetTrendsFlags() {
	trendsSinceFlag = "7d"
}

// TestCostTrendsEmpty tests trends with no data
func TestCostTrendsEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetTrendsFlags()
	defer resetTrendsFlags()

	stdout, _, err := executeIntegrationCmd("cost", "trends")
	if err != nil {
		t.Fatalf("cost trends failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No cost data") {
		t.Errorf("expected no data message, got: %s", stdout)
	}
}

// TestCostTrendsWithData tests trends with historical data
func TestCostTrendsWithData(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetTrendsFlags()
	defer resetTrendsFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	// Add multiple records
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 10.00)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 8.00)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "trends", "--since", "7d")
	if err != nil {
		t.Fatalf("cost trends failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Daily Cost Trends") {
		t.Errorf("expected 'Daily Cost Trends' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Total:") {
		t.Errorf("expected total summary, got: %s", stdout)
	}
}

// TestCostTrendsSincePeriods tests various since periods
func TestCostTrendsSincePeriods(t *testing.T) {
	tests := []struct {
		name  string
		since string
	}{
		{"24 hours", "24h"},
		{"7 days", "7d"},
		{"30 days", "30d"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wsDir, cleanup := setupIntegrationWorkspace(t)
			defer cleanup()
			resetTrendsFlags()
			defer resetTrendsFlags()

			store := cost.NewStore(wsDir)
			if err := store.Open(); err != nil {
				t.Fatalf("failed to open store: %v", err)
			}
			_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 10.00)
			_ = store.Close()

			stdout, _, err := executeIntegrationCmd("cost", "trends", "--since", tt.since)
			if err != nil {
				t.Fatalf("cost trends --since %s failed: %v\nOutput: %s", tt.since, err, stdout)
			}
		})
	}
}

// TestCostTrendsInvalidSince tests rejection of invalid since period
func TestCostTrendsInvalidSince(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetTrendsFlags()
	defer resetTrendsFlags()

	_, _, err := executeIntegrationCmd("cost", "trends", "--since", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid since")
	}
	if !strings.Contains(err.Error(), "invalid duration") {
		t.Errorf("expected 'invalid duration' error, got: %v", err)
	}
}

// =============================================================================
// By-Agent Command Tests
// =============================================================================

// resetByAgentFlags resets by-agent related flags
func resetByAgentFlags() {
	byAgentSinceFlag = "7d"
}

// TestCostByAgentEmpty tests by-agent with no data
func TestCostByAgentEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetByAgentFlags()
	defer resetByAgentFlags()

	stdout, _, err := executeIntegrationCmd("cost", "by-agent")
	if err != nil {
		t.Fatalf("cost by-agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No cost data") {
		t.Errorf("expected no data message, got: %s", stdout)
	}
}

// TestCostByAgentWithData tests by-agent with data
func TestCostByAgentWithData(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetByAgentFlags()
	defer resetByAgentFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 25.00)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 15.00)
	_, _ = store.Record("qa-01", "", "claude-3-haiku", 500, 250, 5.00)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "by-agent", "--since", "7d")
	if err != nil {
		t.Fatalf("cost by-agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Cost by Agent") {
		t.Errorf("expected 'Cost by Agent' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("expected engineer-01 in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "% OF TOTAL") {
		t.Errorf("expected percentage column, got: %s", stdout)
	}
}

// TestCostByAgentShowsPercentages tests that percentages are calculated
func TestCostByAgentShowsPercentages(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetByAgentFlags()
	defer resetByAgentFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	// Create two agents with known costs for percentage calculation
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 50.00)
	_, _ = store.Record("engineer-02", "", "claude-3-opus", 1000, 500, 50.00)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "by-agent", "--since", "7d")
	if err != nil {
		t.Fatalf("cost by-agent failed: %v\nOutput: %s", err, stdout)
	}
	// Each agent should be 50%
	if !strings.Contains(stdout, "50.0%") {
		t.Errorf("expected 50.0%% for each agent, got: %s", stdout)
	}
}

// TestCostByAgentInvalidSince tests rejection of invalid since period
func TestCostByAgentInvalidSince(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetByAgentFlags()
	defer resetByAgentFlags()

	_, _, err := executeIntegrationCmd("cost", "by-agent", "--since", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid since")
	}
	if !strings.Contains(err.Error(), "invalid duration") {
		t.Errorf("expected 'invalid duration' error, got: %v", err)
	}
}

// =============================================================================
// Additional Edge Case Tests
// =============================================================================

// TestCostAddLargeTokens tests handling of large token counts
func TestCostAddLargeTokens(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-01",
		"--tokens-in", "1000000",
		"--tokens-out", "500000",
		"--amount", "150.00",
		"--model", "claude-3-opus",
	)
	if err != nil {
		t.Fatalf("cost add with large tokens failed: %v\nOutput: %s", err, stdout)
	}

	// Verify large tokens were recorded
	store := cost.NewStore(wsDir)
	if err = store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	records, getErr := store.GetByAgent("engineer-01", 10)
	if getErr != nil {
		t.Fatalf("failed to get records: %v", getErr)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].InputTokens != 1000000 {
		t.Errorf("input tokens = %d, want 1000000", records[0].InputTokens)
	}
}

// TestCostShowLargeLimit tests showing many records
func TestCostShowLargeLimit(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	// Create many records
	for i := 0; i < 50; i++ {
		_, _ = store.Record("engineer-01", "", "claude-3-opus", int64(1000+i), 500, 0.05)
	}
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "show", "--limit", "100")
	if err != nil {
		t.Fatalf("cost show --limit 100 failed: %v\nOutput: %s", err, stdout)
	}

	// Count records
	lines := strings.Split(stdout, "\n")
	count := 0
	for _, line := range lines {
		if strings.Contains(line, "engineer-01") {
			count++
		}
	}
	if count != 50 {
		t.Errorf("expected 50 records, got %d", count)
	}
}

// TestCostSummaryMultipleAgents tests summary with many agents
func TestCostSummaryMultipleAgents(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	// Create records for multiple agents
	agents := []string{"eng-01", "eng-02", "eng-03", "qa-01", "qa-02", "pm-01"}
	for _, agent := range agents {
		_, _ = store.Record(agent, "", "claude-3-opus", 1000, 500, 5.00)
	}
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--workspace")
	if err != nil {
		t.Fatalf("cost summary failed: %v\nOutput: %s", err, stdout)
	}

	// Should show all agents in breakdown
	for _, agent := range agents {
		if !strings.Contains(stdout, agent) {
			t.Errorf("expected agent %s in output: %s", agent, stdout)
		}
	}
}

// TestCostAddWithModel tests specifying model explicitly
func TestCostAddWithModel(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Reset flags
	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-01",
		"--amount", "1.00",
		"--model", "gpt-4-turbo",
	)
	if err != nil {
		t.Fatalf("cost add with model failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "gpt-4-turbo") {
		t.Errorf("expected model in output, got: %s", stdout)
	}

	// Verify model was recorded
	store := cost.NewStore(wsDir)
	if err = store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	records, getErr := store.GetByAgent("engineer-01", 10)
	if getErr != nil {
		t.Fatalf("failed to get records: %v", getErr)
	}
	if records[0].Model != "gpt-4-turbo" {
		t.Errorf("model = %s, want gpt-4-turbo", records[0].Model)
	}
}

// TestCostShowJSON tests JSON output format
func TestCostShowJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "show", "--json")
	if err != nil {
		t.Fatalf("cost show --json failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "by_agent") {
		t.Errorf("expected JSON structure with by_agent, got: %s", stdout)
	}
	if !strings.Contains(stdout, "total_cost") {
		t.Errorf("expected total_cost in JSON, got: %s", stdout)
	}
}

// TestParseCostDuration tests duration parsing helper
func TestParseCostDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"days", "7d", false},
		{"hours", "24h", false},
		{"minutes", "30m", false},
		{"seconds", "60s", false},
		{"one day", "1d", false},
		{"thirty days", "30d", false},
		{"invalid", "invalid", true},
		{"empty", "", true},
		// Note: "-1d" actually parses as a negative duration which is technically valid
		// The function uses time.ParseDuration which accepts negatives
		{"negative hours", "-24h", false}, // Valid Go duration
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseCostDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseCostDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// TestCostShowNonExistentAgent tests showing costs for non-existent agent
func TestCostShowNonExistentAgent(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	stdout, _, err := executeIntegrationCmd("cost", "show", "nonexistent-agent")
	if err != nil {
		t.Fatalf("cost show nonexistent-agent should not error: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No cost records found") {
		t.Errorf("expected no records message, got: %s", stdout)
	}
}

// TestCostSummaryAgentNonExistent tests summary for non-existent agent
func TestCostSummaryAgentNonExistent(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--agent", "nonexistent")
	if err != nil {
		t.Fatalf("cost summary --agent nonexistent should not error: %v\nOutput: %s", err, stdout)
	}
	// Should show agent summary with zero values
	if !strings.Contains(stdout, "Agent:") {
		t.Errorf("expected agent header, got: %s", stdout)
	}
}

// TestCostBudgetSetMissingAmount tests budget set without amount argument
func TestCostBudgetSetMissingAmount(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	_, _, err := executeIntegrationCmd("cost", "budget", "set")
	if err == nil {
		t.Fatal("expected error when amount is missing")
	}
}

// TestCostDashboardTeamBreakdown tests dashboard shows team breakdown
func TestCostDashboardTeamBreakdown(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	// Create records with team affiliations
	_, _ = store.Record("eng-01", "engineering", "claude-3-opus", 1000, 500, 10.00)
	_, _ = store.Record("qa-01", "qa-team", "claude-3-opus", 1000, 500, 5.00)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("cost dashboard failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "BY TEAM") {
		t.Errorf("expected 'BY TEAM' section, got: %s", stdout)
	}
}

// =============================================================================
// Additional Edge Cases and Scenarios
// =============================================================================

// TestCostShowMultipleModels tests showing records with different models
func TestCostShowMultipleModels(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	// Create records with multiple models
	_, _ = store.Record("eng-01", "", "claude-3-opus", 1000, 500, 0.10)
	_, _ = store.Record("eng-01", "", "claude-3-sonnet", 1000, 500, 0.05)
	_, _ = store.Record("eng-01", "", "claude-3-haiku", 1000, 500, 0.01)
	_, _ = store.Record("eng-01", "", "gpt-4", 1000, 500, 0.08)
	_ = store.Close()

	stdout, _, err := executeIntegrationCmd("cost", "show")
	if err != nil {
		t.Fatalf("cost show failed: %v\nOutput: %s", err, stdout)
	}
	// Verify all models appear
	models := []string{"claude-3-opus", "claude-3-sonnet", "claude-3-haiku", "gpt-4"}
	for _, m := range models {
		if !strings.Contains(stdout, m) {
			t.Errorf("expected model %s in output: %s", m, stdout)
		}
	}
}

// TestCostBudgetUpdateExisting tests updating an existing budget
func TestCostBudgetUpdateExisting(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set initial budget
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "100.00")
	if err != nil {
		t.Fatalf("failed to set initial budget: %v", err)
	}

	resetBudgetFlags()
	// Update to new amount
	stdout, _, err := executeIntegrationCmd("cost", "budget", "set", "200.00")
	if err != nil {
		t.Fatalf("failed to update budget: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "200.00") {
		t.Errorf("expected updated amount 200.00, got: %s", stdout)
	}
}

// TestCostSummaryEmptyTeam tests summary for team with no records
func TestCostSummaryEmptyTeam(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--team", "nonexistent-team")
	if err != nil {
		t.Fatalf("cost summary --team nonexistent should not error: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "Team:") {
		t.Errorf("expected team header, got: %s", stdout)
	}
}

// TestCostAddZeroTokens tests adding cost with zero tokens
func TestCostAddZeroTokens(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	addCostAgentFlag = ""
	addCostAmountFlag = 0
	addCostToolFlag = ""
	addCostModelFlag = "manual"
	addCostInputTokens = 0
	addCostOutputTokens = 0

	stdout, _, err := executeIntegrationCmd(
		"cost", "add",
		"--agent", "engineer-01",
		"--tokens-in", "0",
		"--tokens-out", "0",
		"--amount", "1.00",
	)
	if err != nil {
		t.Fatalf("cost add with zero tokens failed: %v\nOutput: %s", err, stdout)
	}

	// Verify it was recorded
	store := cost.NewStore(wsDir)
	if err = store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	records, getErr := store.GetByAgent("engineer-01", 10)
	if getErr != nil {
		t.Fatalf("failed to get records: %v", getErr)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].CostUSD != 1.00 {
		t.Errorf("cost = %f, want 1.00", records[0].CostUSD)
	}
}

// TestCostBudgetShowAgentSpecific tests showing agent-specific budget
func TestCostBudgetShowAgentSpecific(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set agent budget
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "75.00", "--agent", "eng-01")
	if err != nil {
		t.Fatalf("failed to set agent budget: %v", err)
	}

	resetBudgetFlags()
	stdout, _, err := executeIntegrationCmd("cost", "budget", "show", "--agent", "eng-01")
	if err != nil {
		t.Fatalf("cost budget show --agent failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "75.00") {
		t.Errorf("expected budget amount 75.00, got: %s", stdout)
	}
}

// TestCostBudgetShowTeamSpecific tests showing team-specific budget
func TestCostBudgetShowTeamSpecific(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetBudgetFlags()
	defer resetBudgetFlags()

	// Set team budget
	_, _, err := executeIntegrationCmd("cost", "budget", "set", "300.00", "--team", "engineering")
	if err != nil {
		t.Fatalf("failed to set team budget: %v", err)
	}

	resetBudgetFlags()
	stdout, _, err := executeIntegrationCmd("cost", "budget", "show", "--team", "engineering")
	if err != nil {
		t.Fatalf("cost budget show --team failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "300.00") {
		t.Errorf("expected budget amount 300.00, got: %s", stdout)
	}
}

// TestCostProjectZeroLookback tests projection with zero lookback handled
func TestCostProjectZeroLookback(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetProjectFlags()
	defer resetProjectFlags()

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("eng-01", "", "claude-3-opus", 1000, 500, 10.00)
	_ = store.Close()

	// Zero lookback should still work (uses available data)
	stdout, _, err := executeIntegrationCmd("cost", "project", "--lookback", "0")
	if err != nil {
		t.Fatalf("cost project --lookback 0 failed: %v\nOutput: %s", err, stdout)
	}
}

// TestCostSummaryModelEmpty tests model summary with no records
func TestCostSummaryModelEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()
	resetCostFlags()
	defer resetCostFlags()

	stdout, _, err := executeIntegrationCmd("cost", "summary", "--model")
	if err != nil {
		t.Fatalf("cost summary --model failed: %v\nOutput: %s", err, stdout)
	}
	if !strings.Contains(stdout, "No cost records found") {
		t.Errorf("expected no records message, got: %s", stdout)
	}
}
