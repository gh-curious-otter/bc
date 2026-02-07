package cost

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/test")
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	expected := filepath.Join("/tmp/test", ".bc", "costs")
	if store.costsDir != expected {
		t.Errorf("costsDir = %q, want %q", store.costsDir, expected)
	}
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	err := store.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	costsDir := filepath.Join(tmpDir, ".bc", "costs")
	if !dirExists(costsDir) {
		t.Errorf("Costs directory not created: %s", costsDir)
	}
}

func TestRecord(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	record := &Record{
		Agent:        "engineer-01",
		Model:        "claude-3-opus",
		Operation:    "completion",
		InputTokens:  1000,
		OutputTokens: 500,
		CostUSD:      0.05,
	}

	err := store.Record(record)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	// Verify timestamp was set
	if record.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	// Verify record was saved
	records, err := store.GetAgentRecords("engineer-01")
	if err != nil {
		t.Fatalf("GetAgentRecords failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Records len = %d, want 1", len(records))
	}
	if records[0].CostUSD != 0.05 {
		t.Errorf("CostUSD = %f, want 0.05", records[0].CostUSD)
	}
}

func TestRecordEmptyAgent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	record := &Record{
		Model:   "claude-3-opus",
		CostUSD: 0.05,
	}

	err := store.Record(record)
	if err == nil {
		t.Error("Expected error for empty agent")
	}
}

func TestRecordMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	records := []Record{
		{Agent: "engineer-01", Model: "claude-3-opus", CostUSD: 0.05},
		{Agent: "engineer-01", Model: "claude-3-sonnet", CostUSD: 0.02},
		{Agent: "engineer-01", Model: "claude-3-opus", CostUSD: 0.08},
	}

	for i := range records {
		if err := store.Record(&records[i]); err != nil {
			t.Fatalf("Record %d failed: %v", i, err)
		}
	}

	// Verify all were saved
	saved, err := store.GetAgentRecords("engineer-01")
	if err != nil {
		t.Fatalf("GetAgentRecords failed: %v", err)
	}
	if len(saved) != 3 {
		t.Errorf("Records len = %d, want 3", len(saved))
	}
}

func TestGetAgentSummary(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	records := []Record{
		{Agent: "engineer-01", InputTokens: 100, OutputTokens: 50, CostUSD: 0.01},
		{Agent: "engineer-01", InputTokens: 200, OutputTokens: 100, CostUSD: 0.02},
		{Agent: "engineer-01", InputTokens: 150, OutputTokens: 75, CostUSD: 0.015},
	}

	for i := range records {
		_ = store.Record(&records[i])
	}

	summary, err := store.GetAgentSummary("engineer-01")
	if err != nil {
		t.Fatalf("GetAgentSummary failed: %v", err)
	}

	if summary.RecordCount != 3 {
		t.Errorf("RecordCount = %d, want 3", summary.RecordCount)
	}
	if summary.TotalInputTokens != 450 {
		t.Errorf("TotalInputTokens = %d, want 450", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 225 {
		t.Errorf("TotalOutputTokens = %d, want 225", summary.TotalOutputTokens)
	}
	expectedCost := 0.045
	if summary.TotalCostUSD < expectedCost-0.001 || summary.TotalCostUSD > expectedCost+0.001 {
		t.Errorf("TotalCostUSD = %f, want %f", summary.TotalCostUSD, expectedCost)
	}
}

func TestGetAgentSummaryNoRecords(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	summary, err := store.GetAgentSummary("nonexistent")
	if err != nil {
		t.Fatalf("GetAgentSummary failed: %v", err)
	}

	if summary.RecordCount != 0 {
		t.Errorf("RecordCount = %d, want 0", summary.RecordCount)
	}
	if summary.TotalCostUSD != 0 {
		t.Errorf("TotalCostUSD = %f, want 0", summary.TotalCostUSD)
	}
}

func TestGetWorkspaceSummary(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	records := []Record{
		{Agent: "engineer-01", CostUSD: 0.10},
		{Agent: "engineer-02", CostUSD: 0.15},
		{Agent: "qa-01", CostUSD: 0.05},
	}

	for i := range records {
		_ = store.Record(&records[i])
	}

	summary, err := store.GetWorkspaceSummary()
	if err != nil {
		t.Fatalf("GetWorkspaceSummary failed: %v", err)
	}

	if summary.RecordCount != 3 {
		t.Errorf("RecordCount = %d, want 3", summary.RecordCount)
	}
	expectedCost := 0.30
	if summary.TotalCostUSD < expectedCost-0.001 || summary.TotalCostUSD > expectedCost+0.001 {
		t.Errorf("TotalCostUSD = %f, want %f", summary.TotalCostUSD, expectedCost)
	}
}

func TestGetWorkspaceSummaryEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	summary, err := store.GetWorkspaceSummary()
	if err != nil {
		t.Fatalf("GetWorkspaceSummary failed: %v", err)
	}

	if summary.RecordCount != 0 {
		t.Errorf("RecordCount = %d, want 0", summary.RecordCount)
	}
}

func TestListAgents(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	// Record for multiple agents
	_ = store.Record(&Record{Agent: "engineer-01", CostUSD: 0.01})
	_ = store.Record(&Record{Agent: "engineer-02", CostUSD: 0.02})
	_ = store.Record(&Record{Agent: "qa-01", CostUSD: 0.03})

	agents, err := store.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	if len(agents) != 3 {
		t.Errorf("Agents len = %d, want 3", len(agents))
	}
}

func TestListAgentsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	agents, err := store.ListAgents()
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("Agents len = %d, want 0", len(agents))
	}
}

func TestSummaryTimeRange(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	now := time.Now().UTC()
	records := []Record{
		{Agent: "agent-01", Timestamp: now.Add(-2 * time.Hour), CostUSD: 0.01},
		{Agent: "agent-01", Timestamp: now.Add(-1 * time.Hour), CostUSD: 0.02},
		{Agent: "agent-01", Timestamp: now, CostUSD: 0.03},
	}

	for i := range records {
		_ = store.Record(&records[i])
	}

	summary, err := store.GetAgentSummary("agent-01")
	if err != nil {
		t.Fatalf("GetAgentSummary failed: %v", err)
	}

	// FirstRecord should be the earliest
	if summary.FirstRecord.After(now.Add(-1 * time.Hour)) {
		t.Error("FirstRecord should be the earliest record")
	}

	// LastRecord should be the latest
	if summary.LastRecord.Before(now.Add(-30 * time.Minute)) {
		t.Error("LastRecord should be the latest record")
	}
}

func TestAgentPath(t *testing.T) {
	store := NewStore("/tmp/test")
	expected := filepath.Join("/tmp/test", ".bc", "costs", "my-agent.json")
	got := store.agentPath("my-agent")
	if got != expected {
		t.Errorf("agentPath = %q, want %q", got, expected)
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
