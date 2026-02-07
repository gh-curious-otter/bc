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
