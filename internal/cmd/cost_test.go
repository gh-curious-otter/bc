package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/cost"
)

func TestCostShowEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("cost", "show")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "No cost records found") {
		t.Errorf("expected 'No cost records found', got: %s", output)
	}
}

func TestCostShowWithRecords(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Seed cost data
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, err := store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	if err != nil {
		t.Fatalf("failed to record cost: %v", err)
	}
	_ = store.Close()

	output, err := executeCmd("cost", "show")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "engineer-01") {
		t.Errorf("expected output to contain 'engineer-01', got: %s", output)
	}
	if !strings.Contains(output, "claude-3-opus") {
		t.Errorf("expected output to contain 'claude-3-opus', got: %s", output)
	}
}

func TestCostShowByAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Seed cost data for two agents
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	output, err := executeCmd("cost", "show", "engineer-01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "engineer-01") {
		t.Errorf("expected output to contain 'engineer-01', got: %s", output)
	}
	if strings.Contains(output, "engineer-02") {
		t.Errorf("expected output NOT to contain 'engineer-02', got: %s", output)
	}
}

func TestCostSummaryWorkspace(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags
	costAgentFlag = ""
	costTeamFlag = ""
	costModelFlag = false
	costWorkspaceFlag = false

	// Seed cost data
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	output, err := executeCmd("cost", "summary", "--workspace")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "Workspace Summary") {
		t.Errorf("expected 'Workspace Summary', got: %s", output)
	}
	if !strings.Contains(output, "API Calls") {
		t.Errorf("expected 'API Calls', got: %s", output)
	}
}

func TestCostSummaryByAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags
	costAgentFlag = ""
	costTeamFlag = ""
	costModelFlag = false
	costWorkspaceFlag = false

	// Seed cost data
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_ = store.Close()

	output, err := executeCmd("cost", "summary", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "Agent: engineer-01") {
		t.Errorf("expected 'Agent: engineer-01', got: %s", output)
	}
}

func TestCostSummaryByTeam(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags
	costAgentFlag = ""
	costTeamFlag = ""
	costModelFlag = false
	costWorkspaceFlag = false

	// Seed cost data with team
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, _ = store.Record("engineer-01", "engineering", "claude-3-opus", 1000, 500, 0.05)
	_ = store.Close()

	output, err := executeCmd("cost", "summary", "--team", "engineering")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "Team: engineering") {
		t.Errorf("expected 'Team: engineering', got: %s", output)
	}
}

func TestCostSummaryByModel(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags
	costAgentFlag = ""
	costTeamFlag = ""
	costModelFlag = false
	costWorkspaceFlag = false

	// Seed cost data
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	output, err := executeCmd("cost", "summary", "--model")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "Cost by Model") {
		t.Errorf("expected 'Cost by Model', got: %s", output)
	}
	if !strings.Contains(output, "claude-3-opus") {
		t.Errorf("expected 'claude-3-opus', got: %s", output)
	}
	if !strings.Contains(output, "claude-3-sonnet") {
		t.Errorf("expected 'claude-3-sonnet', got: %s", output)
	}
}

func TestCostDashboard(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Seed cost data
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	_, _ = store.Record("engineer-01", "engineering", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "engineering", "claude-3-sonnet", 2000, 1000, 0.03)
	_ = store.Close()

	output, err := executeCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "COST DASHBOARD") {
		t.Errorf("expected 'COST DASHBOARD', got: %s", output)
	}
	if !strings.Contains(output, "WORKSPACE TOTALS") {
		t.Errorf("expected 'WORKSPACE TOTALS', got: %s", output)
	}
	if !strings.Contains(output, "BY AGENT") {
		t.Errorf("expected 'BY AGENT', got: %s", output)
	}
	if !strings.Contains(output, "BY MODEL") {
		t.Errorf("expected 'BY MODEL', got: %s", output)
	}
}

func TestCostDashboardEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !strings.Contains(output, "COST DASHBOARD") {
		t.Errorf("expected 'COST DASHBOARD', got: %s", output)
	}
	// Should still display with zero values
	if !strings.Contains(output, "$0.0000") {
		t.Errorf("expected '$0.0000' for empty dashboard, got: %s", output)
	}
}

func TestCostNoWorkspace(t *testing.T) {
	// Run outside a workspace
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(origDir) }()

	_, err := executeCmd("cost", "show")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}

	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected 'not in a bc workspace' error, got: %v", err)
	}
}

func TestCostShowLimit(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Seed multiple cost records
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	for i := 0; i < 10; i++ {
		_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	}
	_ = store.Close()

	// Reset the limit flag
	costLimitFlag = 20

	output, err := executeCmd("cost", "show", "-n", "5")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	// Count lines (header + 5 records = 6 lines minus empty trailing)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Should have header line + exactly 5 data lines
	if len(lines) != 6 {
		t.Errorf("expected 6 lines (header + 5 records), got %d:\n%s", len(lines), output)
	}
}
