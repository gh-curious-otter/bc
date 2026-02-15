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
