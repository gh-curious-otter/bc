package cmd

import (
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/cost"
)

func TestCostDashboard(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create some cost records
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "team-a", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "team-a", "claude-3-sonnet", 500, 250, 0.01)
	_, _ = store.Record("engineer-03", "team-b", "claude-3-opus", 2000, 1000, 0.10)
	_ = store.Close()

	output, err := executeCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("cost dashboard failed: %v\nOutput: %s", err, output)
	}

	// Check workspace totals section
	if !strings.Contains(output, "COST DASHBOARD") {
		t.Errorf("output should contain dashboard header: %s", output)
	}
	if !strings.Contains(output, "WORKSPACE TOTALS") {
		t.Errorf("output should contain workspace totals: %s", output)
	}

	// Check agent breakdown
	if !strings.Contains(output, "BY AGENT") {
		t.Errorf("output should contain agent breakdown: %s", output)
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("output should contain agent engineer-01: %s", output)
	}

	// Check team aggregation
	if !strings.Contains(output, "BY TEAM") {
		t.Errorf("output should contain team breakdown: %s", output)
	}
	if !strings.Contains(output, "team-a") {
		t.Errorf("output should contain team-a: %s", output)
	}

	// Check model breakdown
	if !strings.Contains(output, "BY MODEL") {
		t.Errorf("output should contain model breakdown: %s", output)
	}
}

func TestCostDashboardEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("cost dashboard failed: %v\nOutput: %s", err, output)
	}

	// Should still show header and workspace totals (zeros)
	if !strings.Contains(output, "COST DASHBOARD") {
		t.Errorf("output should contain dashboard header: %s", output)
	}
	if !strings.Contains(output, "WORKSPACE TOTALS") {
		t.Errorf("output should contain workspace totals: %s", output)
	}
}

func TestCostShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create a cost record
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_ = store.Close()

	output, err := executeCmd("cost", "show")
	if err != nil {
		t.Fatalf("cost show failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "engineer-01") {
		t.Errorf("output should contain agent: %s", output)
	}
	if !strings.Contains(output, "claude-3-opus") {
		t.Errorf("output should contain model: %s", output)
	}
}

func TestCostShowAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create cost records for multiple agents
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 500, 250, 0.01)
	_ = store.Close()

	output, err := executeCmd("cost", "show", "engineer-01")
	if err != nil {
		t.Fatalf("cost show failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "engineer-01") {
		t.Errorf("output should contain engineer-01: %s", output)
	}
	// Should NOT contain engineer-02
	if strings.Contains(output, "engineer-02") {
		t.Errorf("output should NOT contain engineer-02 when filtered: %s", output)
	}
}

func TestCostShowEmpty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("cost", "show")
	if err != nil {
		t.Fatalf("cost show failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "No cost records") {
		t.Errorf("output should indicate no records: %s", output)
	}
}

func TestCostSummary(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create cost records
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("engineer-02", "", "claude-3-sonnet", 500, 250, 0.01)
	_ = store.Close()

	output, err := executeCmd("cost", "summary")
	if err != nil {
		t.Fatalf("cost summary failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Workspace Summary") {
		t.Errorf("output should contain workspace summary: %s", output)
	}
	if !strings.Contains(output, "By Agent") {
		t.Errorf("output should contain agent breakdown: %s", output)
	}
}

func TestCostSummaryAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("engineer-01", "", "claude-3-opus", 1000, 500, 0.05)
	_ = store.Close()

	output, err := executeCmd("cost", "summary", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("cost summary failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Agent: engineer-01") {
		t.Errorf("output should contain agent summary: %s", output)
	}
}

func TestCostSummaryModel(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Reset flags from previous tests
	costAgentFlag = ""
	costTeamFlag = ""
	costModelFlag = false
	costWorkspaceFlag = false

	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	_, _ = store.Record("agent-1", "", "claude-3-opus", 1000, 500, 0.05)
	_, _ = store.Record("agent-2", "", "claude-3-sonnet", 500, 250, 0.01)
	_ = store.Close()

	output, err := executeCmd("cost", "summary", "--model")
	if err != nil {
		t.Fatalf("cost summary failed: %v\nOutput: %s", err, output)
	}

	if !strings.Contains(output, "Cost by Model") {
		t.Errorf("output should contain model summary: %s", output)
	}
	if !strings.Contains(output, "claude-3-opus") {
		t.Errorf("output should contain opus: %s", output)
	}
}
