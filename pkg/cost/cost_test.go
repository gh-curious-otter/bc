package cost

import (
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	store := NewStore("/tmp/test")
	if store == nil {
		t.Fatal("NewStore returned nil")
	}
	if store.path != "/tmp/test/.bc/costs.db" {
		t.Errorf("path = %q, want %q", store.path, "/tmp/test/.bc/costs.db")
	}
}

func TestStoreOpenClose(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)

	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if store.DB() == nil {
		t.Error("DB() should not be nil after Open")
	}

	if err := store.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestStoreRecord(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	record, err := store.Record("engineer-01", "team-a", "claude-3-opus", 1000, 500, 0.05)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if record.AgentID != "engineer-01" {
		t.Errorf("AgentID = %q, want %q", record.AgentID, "engineer-01")
	}
	if record.TeamID != "team-a" {
		t.Errorf("TeamID = %q, want %q", record.TeamID, "team-a")
	}
	if record.Model != "claude-3-opus" {
		t.Errorf("Model = %q, want %q", record.Model, "claude-3-opus")
	}
	if record.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want %d", record.InputTokens, 1000)
	}
	if record.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want %d", record.OutputTokens, 500)
	}
	if record.TotalTokens != 1500 {
		t.Errorf("TotalTokens = %d, want %d", record.TotalTokens, 1500)
	}
	if record.CostUSD != 0.05 {
		t.Errorf("CostUSD = %f, want %f", record.CostUSD, 0.05)
	}
	if record.ID == 0 {
		t.Error("ID should not be 0")
	}
	if record.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestStoreRecordNoTeam(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	record, err := store.Record("engineer-01", "", "claude-3-sonnet", 500, 250, 0.01)
	if err != nil {
		t.Fatalf("Record failed: %v", err)
	}

	if record.TeamID != "" {
		t.Errorf("TeamID = %q, want empty", record.TeamID)
	}
}

func TestStoreGetByID(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	record, _ := store.Record("engineer-01", "team-a", "claude-3-opus", 1000, 500, 0.05)

	got, err := store.GetByID(record.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}
	if got.AgentID != "engineer-01" {
		t.Errorf("AgentID = %q, want %q", got.AgentID, "engineer-01")
	}
}

func TestStoreGetByIDNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	got, err := store.GetByID(999)
	if err != nil {
		t.Fatalf("GetByID should not error: %v", err)
	}
	if got != nil {
		t.Error("GetByID should return nil for non-existent ID")
	}
}

func TestStoreGetByAgent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("engineer-01", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("engineer-01", "", "model-b", 200, 100, 0.02)
	_, _ = store.Record("engineer-02", "", "model-a", 300, 150, 0.03)

	records, err := store.GetByAgent("engineer-01", 10)
	if err != nil {
		t.Fatalf("GetByAgent failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(records))
	}
}

func TestStoreGetByTeam(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("engineer-01", "team-a", "model-a", 100, 50, 0.01)
	_, _ = store.Record("engineer-02", "team-a", "model-a", 200, 100, 0.02)
	_, _ = store.Record("engineer-03", "team-b", "model-a", 300, 150, 0.03)

	records, err := store.GetByTeam("team-a", 10)
	if err != nil {
		t.Fatalf("GetByTeam failed: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("len(records) = %d, want 2", len(records))
	}
}

func TestStoreGetAll(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.02)
	_, _ = store.Record("agent-3", "", "model-a", 300, 150, 0.03)

	records, err := store.GetAll(10)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(records) != 3 {
		t.Errorf("len(records) = %d, want 3", len(records))
	}
}

func TestStoreSummaryByAgent(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("engineer-01", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("engineer-01", "", "model-a", 200, 100, 0.02)
	_, _ = store.Record("engineer-02", "", "model-a", 300, 150, 0.03)

	summaries, err := store.SummaryByAgent()
	if err != nil {
		t.Fatalf("SummaryByAgent failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}

	// Find engineer-01 summary
	var eng01 *Summary
	for _, s := range summaries {
		if s.AgentID == "engineer-01" {
			eng01 = s
			break
		}
	}
	if eng01 == nil {
		t.Fatal("engineer-01 not found in summaries")
	}
	if eng01.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", eng01.InputTokens)
	}
	if eng01.TotalCostUSD != 0.03 {
		t.Errorf("TotalCostUSD = %f, want 0.03", eng01.TotalCostUSD)
	}
	if eng01.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2", eng01.RecordCount)
	}
}

func TestStoreSummaryByTeam(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "team-a", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "team-a", "model-a", 200, 100, 0.02)
	_, _ = store.Record("agent-3", "team-b", "model-a", 300, 150, 0.03)
	_, _ = store.Record("agent-4", "", "model-a", 400, 200, 0.04) // No team

	summaries, err := store.SummaryByTeam()
	if err != nil {
		t.Fatalf("SummaryByTeam failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2 (records without team should be excluded)", len(summaries))
	}
}

func TestStoreSummaryByModel(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "", "claude-3-opus", 1000, 500, 0.10)
	_, _ = store.Record("agent-2", "", "claude-3-opus", 2000, 1000, 0.20)
	_, _ = store.Record("agent-3", "", "claude-3-sonnet", 500, 250, 0.01)

	summaries, err := store.SummaryByModel()
	if err != nil {
		t.Fatalf("SummaryByModel failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len(summaries) = %d, want 2", len(summaries))
	}

	// Should be sorted by cost DESC
	if summaries[0].Model != "claude-3-opus" {
		t.Errorf("First model = %q, want claude-3-opus (highest cost)", summaries[0].Model)
	}
}

func TestStoreWorkspaceSummary(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-b", 200, 100, 0.02)
	_, _ = store.Record("agent-3", "", "model-c", 300, 150, 0.03)

	summary, err := store.WorkspaceSummary()
	if err != nil {
		t.Fatalf("WorkspaceSummary failed: %v", err)
	}

	if summary.InputTokens != 600 {
		t.Errorf("InputTokens = %d, want 600", summary.InputTokens)
	}
	if summary.OutputTokens != 300 {
		t.Errorf("OutputTokens = %d, want 300", summary.OutputTokens)
	}
	if summary.TotalTokens != 900 {
		t.Errorf("TotalTokens = %d, want 900", summary.TotalTokens)
	}
	if summary.TotalCostUSD != 0.06 {
		t.Errorf("TotalCostUSD = %f, want 0.06", summary.TotalCostUSD)
	}
	if summary.RecordCount != 3 {
		t.Errorf("RecordCount = %d, want 3", summary.RecordCount)
	}
}

func TestStoreWorkspaceSummaryEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	summary, err := store.WorkspaceSummary()
	if err != nil {
		t.Fatalf("WorkspaceSummary failed: %v", err)
	}

	if summary.TotalCostUSD != 0 {
		t.Errorf("TotalCostUSD = %f, want 0", summary.TotalCostUSD)
	}
	if summary.RecordCount != 0 {
		t.Errorf("RecordCount = %d, want 0", summary.RecordCount)
	}
}

func TestStoreAgentSummary(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("engineer-01", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("engineer-01", "", "model-b", 200, 100, 0.02)
	_, _ = store.Record("engineer-02", "", "model-a", 300, 150, 0.03)

	summary, err := store.AgentSummary("engineer-01")
	if err != nil {
		t.Fatalf("AgentSummary failed: %v", err)
	}

	if summary.AgentID != "engineer-01" {
		t.Errorf("AgentID = %q, want engineer-01", summary.AgentID)
	}
	if summary.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", summary.InputTokens)
	}
	if summary.TotalCostUSD != 0.03 {
		t.Errorf("TotalCostUSD = %f, want 0.03", summary.TotalCostUSD)
	}
	if summary.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2", summary.RecordCount)
	}
}

func TestStoreTeamSummary(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "team-a", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "team-a", "model-a", 200, 100, 0.02)
	_, _ = store.Record("agent-3", "team-b", "model-a", 300, 150, 0.03)

	summary, err := store.TeamSummary("team-a")
	if err != nil {
		t.Fatalf("TeamSummary failed: %v", err)
	}

	if summary.TeamID != "team-a" {
		t.Errorf("TeamID = %q, want team-a", summary.TeamID)
	}
	if summary.InputTokens != 300 {
		t.Errorf("InputTokens = %d, want 300", summary.InputTokens)
	}
	if summary.TotalCostUSD != 0.03 {
		t.Errorf("TotalCostUSD = %f, want 0.03", summary.TotalCostUSD)
	}
	if summary.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2", summary.RecordCount)
	}
}

func TestStoreClear(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.02)

	if err := store.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	records, err := store.GetAll(100)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("len(records) = %d, want 0 after Clear", len(records))
	}
}

func TestRecordStruct(t *testing.T) {
	r := Record{
		ID:          1,
		TotalTokens: 1500,
	}

	if r.ID != 1 {
		t.Errorf("ID = %d, want 1", r.ID)
	}
	if r.TotalTokens != 1500 {
		t.Errorf("TotalTokens = %d, want 1500", r.TotalTokens)
	}
}

func TestSummaryStruct(t *testing.T) {
	s := Summary{
		AgentID:     "agent-1",
		RecordCount: 10,
	}

	if s.AgentID != "agent-1" {
		t.Errorf("AgentID = %q, want agent-1", s.AgentID)
	}
	if s.RecordCount != 10 {
		t.Errorf("RecordCount = %d, want 10", s.RecordCount)
	}
}

func TestGetDailyCosts(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some records
	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.02)

	// Get daily costs for the last day
	dailyCosts, err := store.GetDailyCosts(time.Now().AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("GetDailyCosts failed: %v", err)
	}

	// Should have at least one day of data
	if len(dailyCosts) < 1 {
		t.Error("expected at least 1 day of cost data")
	}
}

func TestGetDailyCostsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Get daily costs with no data
	dailyCosts, err := store.GetDailyCosts(time.Now().AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("GetDailyCosts failed: %v", err)
	}

	if len(dailyCosts) != 0 {
		t.Errorf("expected 0 days, got %d", len(dailyCosts))
	}
}

func TestGetAgentDailyCosts(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add records for multiple agents
	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.02)

	// Get agent daily costs
	agentCosts, err := store.GetAgentDailyCosts(time.Now().AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("GetAgentDailyCosts failed: %v", err)
	}

	// Should have data for both agents
	if len(agentCosts) < 2 {
		t.Errorf("expected at least 2 agent cost entries, got %d", len(agentCosts))
	}
}

func TestGetSummarySince(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.02)

	summary, err := store.GetSummarySince(time.Now().AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("GetSummarySince failed: %v", err)
	}

	if summary.RecordCount != 2 {
		t.Errorf("RecordCount = %d, want 2", summary.RecordCount)
	}
	if summary.TotalCostUSD != 0.03 {
		t.Errorf("TotalCostUSD = %f, want 0.03", summary.TotalCostUSD)
	}
}

func TestGetAgentSummarySince(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.01)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.02)

	summaries, err := store.GetAgentSummarySince(time.Now().AddDate(0, 0, -1))
	if err != nil {
		t.Fatalf("GetAgentSummarySince failed: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("expected 2 agents, got %d", len(summaries))
	}
}

func TestProjectCost(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add some records
	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 0.10)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 0.20)

	// Project for 7 days based on 7 days of history
	proj, err := store.ProjectCost(7, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ProjectCost failed: %v", err)
	}

	// Should have 1 day of data
	if proj.DaysAnalyzed < 1 {
		t.Errorf("DaysAnalyzed = %d, expected >= 1", proj.DaysAnalyzed)
	}

	// Total historical should be approximately 0.30
	if proj.TotalHistorical < 0.29 || proj.TotalHistorical > 0.31 {
		t.Errorf("TotalHistorical = %f, want ~0.30", proj.TotalHistorical)
	}
}

func TestProjectCostEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Project with no data
	proj, err := store.ProjectCost(7, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("ProjectCost failed: %v", err)
	}

	if proj.DaysAnalyzed != 0 {
		t.Errorf("DaysAnalyzed = %d, want 0", proj.DaysAnalyzed)
	}
	if proj.ProjectedCost != 0 {
		t.Errorf("ProjectedCost = %f, want 0", proj.ProjectedCost)
	}
}

func TestDailyCostStruct(t *testing.T) {
	dc := DailyCost{
		Date:    "2026-01-01",
		CostUSD: 1.50,
	}

	if dc.Date != "2026-01-01" {
		t.Errorf("Date = %q, want 2026-01-01", dc.Date)
	}
	if dc.CostUSD != 1.50 {
		t.Errorf("CostUSD = %f, want 1.50", dc.CostUSD)
	}
}

func TestProjectionStruct(t *testing.T) {
	proj := Projection{
		DailyAvgCost:  1.00,
		ProjectedCost: 7.00,
		DaysAnalyzed:  7,
	}

	if proj.DailyAvgCost != 1.00 {
		t.Errorf("DailyAvgCost = %f, want 1.00", proj.DailyAvgCost)
	}
	if proj.ProjectedCost != 7.00 {
		t.Errorf("ProjectedCost = %f, want 7.00", proj.ProjectedCost)
	}
	if proj.DaysAnalyzed != 7 {
		t.Errorf("DaysAnalyzed = %d, want 7", proj.DaysAnalyzed)
	}
}

func TestParseCostFromMessage(t *testing.T) {
	tests := []struct { //nolint:govet
		name    string
		message string
		want    *CostMessage
	}{
		{
			name:    "empty message",
			message: "",
			want:    nil,
		},
		{
			name:    "no cost info",
			message: "This is a regular message",
			want:    nil,
		},
		{
			name:    "message with cost keyword but no data",
			message: "The cost of this operation is undefined",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseCostFromMessage(tt.message)
			if got != tt.want {
				t.Errorf("ParseCostFromMessage(%q) = %v, want %v", tt.message, got, tt.want)
			}
		})
	}
}

func TestRecordFromMessage(t *testing.T) {
	cm := &CostMessage{
		AgentID:      "engineer-01",
		InputTokens:  1000,
		OutputTokens: 500,
		CostUSD:      0.05,
		Message:      "test message",
	}

	record := RecordFromMessage(cm)
	if record == nil {
		t.Fatal("expected non-nil record")
	}

	if record.AgentID != "engineer-01" {
		t.Errorf("AgentID = %q, want engineer-01", record.AgentID)
	}
	if record.InputTokens != 1000 {
		t.Errorf("InputTokens = %d, want 1000", record.InputTokens)
	}
	if record.OutputTokens != 500 {
		t.Errorf("OutputTokens = %d, want 500", record.OutputTokens)
	}
	if record.TotalTokens != 1500 {
		t.Errorf("TotalTokens = %d, want 1500", record.TotalTokens)
	}
	if record.CostUSD != 0.05 {
		t.Errorf("CostUSD = %f, want 0.05", record.CostUSD)
	}
	if record.Model != "extracted" {
		t.Errorf("Model = %q, want extracted", record.Model)
	}
}

func TestRecordFromMessageNil(t *testing.T) {
	record := RecordFromMessage(nil)
	if record != nil {
		t.Errorf("expected nil for nil input, got %v", record)
	}
}

func TestSetBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget, err := store.SetBudget("workspace", BudgetPeriodDaily, 100.0, 0.8, true)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	if budget == nil {
		t.Fatal("SetBudget returned nil budget")
	}
	if budget.Scope != "workspace" {
		t.Errorf("Scope = %q, want workspace", budget.Scope)
	}
	if budget.Period != BudgetPeriodDaily {
		t.Errorf("Period = %q, want daily", budget.Period)
	}
	if budget.LimitUSD != 100.0 {
		t.Errorf("LimitUSD = %f, want 100.0", budget.LimitUSD)
	}
	if budget.AlertAt != 0.8 {
		t.Errorf("AlertAt = %f, want 0.8", budget.AlertAt)
	}
	if !budget.HardStop {
		t.Error("HardStop should be true")
	}
}

func TestSetBudgetUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create initial budget
	_, err := store.SetBudget("agent:eng-01", BudgetPeriodWeekly, 50.0, 0.7, false)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	// Update the budget
	budget, err := store.SetBudget("agent:eng-01", BudgetPeriodMonthly, 200.0, 0.9, true)
	if err != nil {
		t.Fatalf("SetBudget update failed: %v", err)
	}

	if budget.Period != BudgetPeriodMonthly {
		t.Errorf("Period = %q, want monthly", budget.Period)
	}
	if budget.LimitUSD != 200.0 {
		t.Errorf("LimitUSD = %f, want 200.0", budget.LimitUSD)
	}
	if !budget.HardStop {
		t.Error("HardStop should be true after update")
	}
}

func TestGetBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a budget
	_, err := store.SetBudget("team:backend", BudgetPeriodWeekly, 500.0, 0.75, false)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	// Get the budget
	budget, err := store.GetBudget("team:backend")
	if err != nil {
		t.Fatalf("GetBudget failed: %v", err)
	}

	if budget == nil {
		t.Fatal("GetBudget returned nil")
	}
	if budget.Scope != "team:backend" {
		t.Errorf("Scope = %q, want team:backend", budget.Scope)
	}
	if budget.LimitUSD != 500.0 {
		t.Errorf("LimitUSD = %f, want 500.0", budget.LimitUSD)
	}
}

func TestGetBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget, err := store.GetBudget("nonexistent")
	if err != nil {
		t.Fatalf("GetBudget should not error for missing budget: %v", err)
	}
	if budget != nil {
		t.Error("GetBudget should return nil for missing budget")
	}
}

func TestGetAllBudgets(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create multiple budgets
	_, _ = store.SetBudget("workspace", BudgetPeriodMonthly, 1000.0, 0.8, true)
	_, _ = store.SetBudget("agent:eng-01", BudgetPeriodDaily, 50.0, 0.9, false)
	_, _ = store.SetBudget("team:backend", BudgetPeriodWeekly, 300.0, 0.7, false)

	budgets, err := store.GetAllBudgets()
	if err != nil {
		t.Fatalf("GetAllBudgets failed: %v", err)
	}

	if len(budgets) != 3 {
		t.Errorf("len(budgets) = %d, want 3", len(budgets))
	}
}

func TestGetAllBudgetsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budgets, err := store.GetAllBudgets()
	if err != nil {
		t.Fatalf("GetAllBudgets failed: %v", err)
	}

	if len(budgets) != 0 {
		t.Errorf("len(budgets) = %d, want 0", len(budgets))
	}
}

func TestDeleteBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a budget
	_, err := store.SetBudget("workspace", BudgetPeriodDaily, 100.0, 0.8, false)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	// Delete it
	err = store.DeleteBudget("workspace")
	if err != nil {
		t.Fatalf("DeleteBudget failed: %v", err)
	}

	// Verify it's gone
	budget, err := store.GetBudget("workspace")
	if err != nil {
		t.Fatalf("GetBudget failed: %v", err)
	}
	if budget != nil {
		t.Error("Budget should be deleted")
	}
}

func TestDeleteBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	err := store.DeleteBudget("nonexistent")
	if err == nil {
		t.Error("DeleteBudget should fail for nonexistent budget")
	}
}

func TestCheckBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a budget
	_, err := store.SetBudget("workspace", BudgetPeriodDaily, 100.0, 0.8, true)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	// Add some cost records
	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 50.0)
	_, _ = store.Record("agent-2", "", "model-a", 200, 100, 30.0)

	// Check budget status
	status, err := store.CheckBudget("workspace")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}

	if status == nil {
		t.Fatal("CheckBudget returned nil status")
	}
	if status.Budget == nil {
		t.Fatal("CheckBudget status has nil Budget")
	}
	if status.CurrentSpend != 80.0 {
		t.Errorf("CurrentSpend = %f, want 80.0", status.CurrentSpend)
	}
	if status.Remaining != 20.0 {
		t.Errorf("Remaining = %f, want 20.0", status.Remaining)
	}
	if status.PercentUsed != 0.8 {
		t.Errorf("PercentUsed = %f, want 0.8", status.PercentUsed)
	}
	if status.IsOverBudget {
		t.Error("Should not be over budget")
	}
	if !status.IsNearLimit {
		t.Error("Should be near limit (80% >= 80% alert threshold)")
	}
}

func TestCheckBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	status, err := store.CheckBudget("nonexistent")
	if err != nil {
		t.Fatalf("CheckBudget should not error for missing budget: %v", err)
	}
	if status != nil {
		t.Error("CheckBudget should return nil for missing budget")
	}
}

func TestCheckBudgetOverLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a small budget
	_, err := store.SetBudget("workspace", BudgetPeriodDaily, 10.0, 0.5, true)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	// Exceed the budget
	_, _ = store.Record("agent-1", "", "model-a", 100, 50, 15.0)

	status, err := store.CheckBudget("workspace")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}

	if !status.IsOverBudget {
		t.Error("Should be over budget")
	}
	// Remaining is clamped to 0 when over budget
	if status.Remaining != 0 {
		t.Errorf("Remaining = %f, want 0 (clamped)", status.Remaining)
	}
	if status.PercentUsed < 1.0 {
		t.Errorf("PercentUsed = %f, want >= 1.0", status.PercentUsed)
	}
}

func TestBudgetPeriodConstants(t *testing.T) {
	if BudgetPeriodDaily != "daily" {
		t.Errorf("BudgetPeriodDaily = %q, want daily", BudgetPeriodDaily)
	}
	if BudgetPeriodWeekly != "weekly" {
		t.Errorf("BudgetPeriodWeekly = %q, want weekly", BudgetPeriodWeekly)
	}
	if BudgetPeriodMonthly != "monthly" {
		t.Errorf("BudgetPeriodMonthly = %q, want monthly", BudgetPeriodMonthly)
	}
}

func TestBudgetStruct(t *testing.T) {
	b := Budget{
		Scope:    "workspace",
		Period:   BudgetPeriodDaily,
		LimitUSD: 100.0,
		AlertAt:  0.8,
		HardStop: true,
	}

	if b.Scope != "workspace" {
		t.Errorf("Scope = %q, want workspace", b.Scope)
	}
	if b.Period != BudgetPeriodDaily {
		t.Errorf("Period = %q, want daily", b.Period)
	}
	if b.LimitUSD != 100.0 {
		t.Errorf("LimitUSD = %f, want 100.0", b.LimitUSD)
	}
	if b.AlertAt != 0.8 {
		t.Errorf("AlertAt = %f, want 0.8", b.AlertAt)
	}
	if !b.HardStop {
		t.Error("HardStop should be true")
	}
}

func TestBudgetStatusStruct(t *testing.T) {
	status := BudgetStatus{
		CurrentSpend: 80.0,
		Remaining:    20.0,
		PercentUsed:  0.8,
		IsOverBudget: false,
		IsNearLimit:  true,
	}

	if status.CurrentSpend != 80.0 {
		t.Errorf("CurrentSpend = %f, want 80.0", status.CurrentSpend)
	}
	if status.Remaining != 20.0 {
		t.Errorf("Remaining = %f, want 20.0", status.Remaining)
	}
	if status.PercentUsed != 0.8 {
		t.Errorf("PercentUsed = %f, want 0.8", status.PercentUsed)
	}
	if status.IsOverBudget {
		t.Error("IsOverBudget should be false")
	}
	if !status.IsNearLimit {
		t.Error("IsNearLimit should be true")
	}
}

func TestStoreRecordCostFromMessage(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// RecordCostFromMessage with no cost data should return nil, nil
	record, err := store.RecordCostFromMessage("engineer-01", "No cost information here")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record != nil {
		t.Errorf("expected nil record for message with no cost data, got %v", record)
	}
}

func TestStoreRecordCostFromMessageWithCost(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a CostMessage to test the RecordCostFromMessage flow
	// Note: With current implementation, ParseCostFromMessage returns nil
	// but the infrastructure is in place for future enhancements
	cm := &CostMessage{
		AgentID:      "engineer-01",
		InputTokens:  1000,
		OutputTokens: 500,
		CostUSD:      0.05,
		Message:      "test",
	}

	record := RecordFromMessage(cm)
	if record == nil {
		t.Fatal("expected non-nil record from RecordFromMessage")
	}

	// Verify the record can be stored
	if record.AgentID != "engineer-01" {
		t.Errorf("AgentID mismatch in extracted record")
	}
}

func TestBudgetStruct(t *testing.T) {
	now := time.Now()
	b := Budget{
		ID:        1,
		Scope:     "workspace",
		Period:    BudgetPeriodMonthly,
		LimitUSD:  100.0,
		AlertAt:   0.8,
		HardStop:  true,
		UpdatedAt: now,
	}

	if b.ID != 1 {
		t.Errorf("ID = %d, want 1", b.ID)
	}
	if b.Scope != "workspace" {
		t.Errorf("Scope = %q, want workspace", b.Scope)
	}
	if b.Period != BudgetPeriodMonthly {
		t.Errorf("Period = %q, want monthly", b.Period)
	}
	if b.LimitUSD != 100.0 {
		t.Errorf("LimitUSD = %f, want 100.0", b.LimitUSD)
	}
	if b.AlertAt != 0.8 {
		t.Errorf("AlertAt = %f, want 0.8", b.AlertAt)
	}
	if !b.HardStop {
		t.Error("HardStop should be true")
	}
	if b.UpdatedAt != now {
		t.Error("UpdatedAt should match")
	}
}

func TestBudgetStatusStruct(t *testing.T) {
	budget := &Budget{
		LimitUSD: 100.0,
		AlertAt:  0.8,
	}
	status := BudgetStatus{
		Budget:       budget,
		CurrentSpend: 85.0,
		Remaining:    15.0,
		PercentUsed:  0.85,
		IsOverBudget: false,
		IsNearLimit:  true,
	}

	if status.Budget != budget {
		t.Error("Budget pointer mismatch")
	}
	if status.CurrentSpend != 85.0 {
		t.Errorf("CurrentSpend = %f, want 85.0", status.CurrentSpend)
	}
	if status.Remaining != 15.0 {
		t.Errorf("Remaining = %f, want 15.0", status.Remaining)
	}
	if status.PercentUsed != 0.85 {
		t.Errorf("PercentUsed = %f, want 0.85", status.PercentUsed)
	}
	if status.IsOverBudget {
		t.Error("IsOverBudget should be false")
	}
	if !status.IsNearLimit {
		t.Error("IsNearLimit should be true")
	}
}

func TestBudgetPeriodConstants(t *testing.T) {
	if BudgetPeriodDaily != "daily" {
		t.Errorf("BudgetPeriodDaily = %q, want daily", BudgetPeriodDaily)
	}
	if BudgetPeriodWeekly != "weekly" {
		t.Errorf("BudgetPeriodWeekly = %q, want weekly", BudgetPeriodWeekly)
	}
	if BudgetPeriodMonthly != "monthly" {
		t.Errorf("BudgetPeriodMonthly = %q, want monthly", BudgetPeriodMonthly)
	}
}

func TestAgentDailyCostStruct(t *testing.T) {
	adc := AgentDailyCost{
		AgentID:      "engineer-01",
		Date:         "2026-02-21",
		CostUSD:      2.50,
		TotalTokens:  5000,
		RecordCount:  10,
		InputTokens:  3000,
		OutputTokens: 2000,
	}

	if adc.AgentID != "engineer-01" {
		t.Errorf("AgentID = %q, want engineer-01", adc.AgentID)
	}
	if adc.Date != "2026-02-21" {
		t.Errorf("Date = %q, want 2026-02-21", adc.Date)
	}
	if adc.CostUSD != 2.50 {
		t.Errorf("CostUSD = %f, want 2.50", adc.CostUSD)
	}
	if adc.TotalTokens != 5000 {
		t.Errorf("TotalTokens = %d, want 5000", adc.TotalTokens)
	}
	if adc.RecordCount != 10 {
		t.Errorf("RecordCount = %d, want 10", adc.RecordCount)
	}
	if adc.InputTokens != 3000 {
		t.Errorf("InputTokens = %d, want 3000", adc.InputTokens)
	}
	if adc.OutputTokens != 2000 {
		t.Errorf("OutputTokens = %d, want 2000", adc.OutputTokens)
	}
}

func TestSetBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget, err := store.SetBudget("workspace", BudgetPeriodMonthly, 100.0, 0.8, true)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	if budget == nil {
		t.Fatal("SetBudget returned nil budget")
	}
	if budget.Scope != "workspace" {
		t.Errorf("Scope = %q, want workspace", budget.Scope)
	}
	if budget.Period != BudgetPeriodMonthly {
		t.Errorf("Period = %q, want monthly", budget.Period)
	}
	if budget.LimitUSD != 100.0 {
		t.Errorf("LimitUSD = %f, want 100.0", budget.LimitUSD)
	}
	if budget.AlertAt != 0.8 {
		t.Errorf("AlertAt = %f, want 0.8", budget.AlertAt)
	}
	if !budget.HardStop {
		t.Error("HardStop should be true")
	}
	if budget.ID == 0 {
		t.Error("ID should not be 0")
	}
}

func TestSetBudgetUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create initial budget
	_, err := store.SetBudget("workspace", BudgetPeriodMonthly, 100.0, 0.8, false)
	if err != nil {
		t.Fatalf("SetBudget (create) failed: %v", err)
	}

	// Update the budget
	budget, err := store.SetBudget("workspace", BudgetPeriodWeekly, 50.0, 0.9, true)
	if err != nil {
		t.Fatalf("SetBudget (update) failed: %v", err)
	}

	if budget.Period != BudgetPeriodWeekly {
		t.Errorf("Period = %q, want weekly", budget.Period)
	}
	if budget.LimitUSD != 50.0 {
		t.Errorf("LimitUSD = %f, want 50.0", budget.LimitUSD)
	}
	if budget.AlertAt != 0.9 {
		t.Errorf("AlertAt = %f, want 0.9", budget.AlertAt)
	}
	if !budget.HardStop {
		t.Error("HardStop should be true after update")
	}
}

func TestSetBudgetAgentScope(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget, err := store.SetBudget("agent:engineer-01", BudgetPeriodDaily, 10.0, 0.5, false)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	if budget.Scope != "agent:engineer-01" {
		t.Errorf("Scope = %q, want agent:engineer-01", budget.Scope)
	}
	if budget.Period != BudgetPeriodDaily {
		t.Errorf("Period = %q, want daily", budget.Period)
	}
}

func TestSetBudgetTeamScope(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget, err := store.SetBudget("team:frontend", BudgetPeriodWeekly, 200.0, 0.75, true)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	if budget.Scope != "team:frontend" {
		t.Errorf("Scope = %q, want team:frontend", budget.Scope)
	}
}

func TestGetBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create a budget
	_, err := store.SetBudget("workspace", BudgetPeriodMonthly, 100.0, 0.8, true)
	if err != nil {
		t.Fatalf("SetBudget failed: %v", err)
	}

	// Retrieve it
	budget, err := store.GetBudget("workspace")
	if err != nil {
		t.Fatalf("GetBudget failed: %v", err)
	}
	if budget == nil {
		t.Fatal("GetBudget returned nil")
	}
	if budget.Scope != "workspace" {
		t.Errorf("Scope = %q, want workspace", budget.Scope)
	}
	if budget.LimitUSD != 100.0 {
		t.Errorf("LimitUSD = %f, want 100.0", budget.LimitUSD)
	}
}

func TestGetBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budget, err := store.GetBudget("nonexistent")
	if err != nil {
		t.Fatalf("GetBudget should not error for non-existent: %v", err)
	}
	if budget != nil {
		t.Error("GetBudget should return nil for non-existent budget")
	}
}

func TestGetAllBudgets(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create multiple budgets
	_, _ = store.SetBudget("workspace", BudgetPeriodMonthly, 100.0, 0.8, false)
	_, _ = store.SetBudget("agent:eng-01", BudgetPeriodDaily, 10.0, 0.5, false)
	_, _ = store.SetBudget("team:backend", BudgetPeriodWeekly, 50.0, 0.75, true)

	budgets, err := store.GetAllBudgets()
	if err != nil {
		t.Fatalf("GetAllBudgets failed: %v", err)
	}
	if len(budgets) != 3 {
		t.Errorf("len(budgets) = %d, want 3", len(budgets))
	}

	// Verify sorted by scope
	scopes := make([]string, len(budgets))
	for i, b := range budgets {
		scopes[i] = b.Scope
	}
	// agent:eng-01 < team:backend < workspace (alphabetically)
	if scopes[0] != "agent:eng-01" {
		t.Errorf("First scope = %q, want agent:eng-01", scopes[0])
	}
}

func TestGetAllBudgetsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	budgets, err := store.GetAllBudgets()
	if err != nil {
		t.Fatalf("GetAllBudgets failed: %v", err)
	}
	if len(budgets) != 0 {
		t.Errorf("len(budgets) = %d, want 0", len(budgets))
	}
}

func TestDeleteBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create and then delete
	_, _ = store.SetBudget("workspace", BudgetPeriodMonthly, 100.0, 0.8, false)

	if err := store.DeleteBudget("workspace"); err != nil {
		t.Fatalf("DeleteBudget failed: %v", err)
	}

	// Verify deletion
	budget, err := store.GetBudget("workspace")
	if err != nil {
		t.Fatalf("GetBudget failed: %v", err)
	}
	if budget != nil {
		t.Error("Budget should be nil after deletion")
	}
}

func TestDeleteBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	err := store.DeleteBudget("nonexistent")
	if err == nil {
		t.Error("DeleteBudget should error for non-existent budget")
	}
}

func TestCheckBudgetWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create budget
	_, _ = store.SetBudget("workspace", BudgetPeriodDaily, 1.0, 0.8, false)

	// Add some costs
	_, _ = store.Record("eng-01", "", "model-a", 1000, 500, 0.50)
	_, _ = store.Record("eng-02", "", "model-a", 1000, 500, 0.30)

	// Check budget status
	status, err := store.CheckBudget("workspace")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}
	if status == nil {
		t.Fatal("CheckBudget returned nil status")
	}

	// Use tolerance for floating point comparison
	const tolerance = 0.001
	if status.CurrentSpend < 0.80-tolerance || status.CurrentSpend > 0.80+tolerance {
		t.Errorf("CurrentSpend = %f, want ~0.80", status.CurrentSpend)
	}
	if status.Remaining < 0.20-tolerance || status.Remaining > 0.20+tolerance {
		t.Errorf("Remaining = %f, want ~0.20", status.Remaining)
	}
	if status.PercentUsed < 0.80-tolerance || status.PercentUsed > 0.80+tolerance {
		t.Errorf("PercentUsed = %f, want ~0.80", status.PercentUsed)
	}
	if status.IsOverBudget {
		t.Error("IsOverBudget should be false at 80%")
	}
	if !status.IsNearLimit {
		t.Error("IsNearLimit should be true at 80% (equals AlertAt)")
	}
}

func TestCheckBudgetOverBudget(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create budget
	_, _ = store.SetBudget("workspace", BudgetPeriodDaily, 1.0, 0.8, true)

	// Add costs exceeding budget
	_, _ = store.Record("eng-01", "", "model-a", 2000, 1000, 1.50)

	status, err := store.CheckBudget("workspace")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}

	if !status.IsOverBudget {
		t.Error("IsOverBudget should be true")
	}
	if status.Remaining != 0 {
		t.Errorf("Remaining = %f, want 0 (capped)", status.Remaining)
	}
	if status.PercentUsed != 1.50 {
		t.Errorf("PercentUsed = %f, want 1.50", status.PercentUsed)
	}
}

func TestCheckBudgetAgentScope(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create agent-scoped budget
	_, _ = store.SetBudget("agent:eng-01", BudgetPeriodDaily, 1.0, 0.5, false)

	// Add costs for multiple agents
	_, _ = store.Record("eng-01", "", "model-a", 1000, 500, 0.40)
	_, _ = store.Record("eng-02", "", "model-a", 1000, 500, 0.60) // Different agent

	status, err := store.CheckBudget("agent:eng-01")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}

	// Should only count eng-01's costs
	if status.CurrentSpend != 0.40 {
		t.Errorf("CurrentSpend = %f, want 0.40 (only eng-01)", status.CurrentSpend)
	}
}

func TestCheckBudgetTeamScope(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create team-scoped budget
	_, _ = store.SetBudget("team:frontend", BudgetPeriodDaily, 2.0, 0.5, false)

	// Add costs for different teams
	_, _ = store.Record("eng-01", "frontend", "model-a", 1000, 500, 0.50)
	_, _ = store.Record("eng-02", "frontend", "model-a", 1000, 500, 0.30)
	_, _ = store.Record("eng-03", "backend", "model-a", 1000, 500, 1.00) // Different team

	status, err := store.CheckBudget("team:frontend")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}

	// Should only count frontend team's costs
	if status.CurrentSpend != 0.80 {
		t.Errorf("CurrentSpend = %f, want 0.80 (only frontend)", status.CurrentSpend)
	}
}

func TestCheckBudgetNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	status, err := store.CheckBudget("nonexistent")
	if err != nil {
		t.Fatalf("CheckBudget should not error for non-existent: %v", err)
	}
	if status != nil {
		t.Error("CheckBudget should return nil for non-existent budget")
	}
}

func TestCheckBudgetZeroLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Create budget with zero limit
	_, _ = store.SetBudget("workspace", BudgetPeriodDaily, 0.0, 0.8, false)

	// Add some costs
	_, _ = store.Record("eng-01", "", "model-a", 1000, 500, 0.50)

	status, err := store.CheckBudget("workspace")
	if err != nil {
		t.Fatalf("CheckBudget failed: %v", err)
	}

	// With zero limit, PercentUsed should be 0 (no division by zero)
	if status.PercentUsed != 0 {
		t.Errorf("PercentUsed = %f, want 0 for zero limit", status.PercentUsed)
	}
}

func TestCostMessageStruct(t *testing.T) {
	cm := CostMessage{
		AgentID:      "engineer-01",
		Message:      "test message with cost info",
		InputTokens:  1500,
		OutputTokens: 750,
		CostUSD:      0.075,
	}

	if cm.AgentID != "engineer-01" {
		t.Errorf("AgentID = %q, want engineer-01", cm.AgentID)
	}
	if cm.Message != "test message with cost info" {
		t.Errorf("Message mismatch")
	}
	if cm.InputTokens != 1500 {
		t.Errorf("InputTokens = %d, want 1500", cm.InputTokens)
	}
	if cm.OutputTokens != 750 {
		t.Errorf("OutputTokens = %d, want 750", cm.OutputTokens)
	}
	if cm.CostUSD != 0.075 {
		t.Errorf("CostUSD = %f, want 0.075", cm.CostUSD)
	}
}

func TestGetByAgentDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add more than 100 records
	for range 150 {
		_, _ = store.Record("engineer-01", "", "model-a", 100, 50, 0.01)
	}

	// Get with zero/negative limit (should default to 100)
	records, err := store.GetByAgent("engineer-01", 0)
	if err != nil {
		t.Fatalf("GetByAgent failed: %v", err)
	}
	if len(records) != 100 {
		t.Errorf("len(records) = %d, want 100 (default limit)", len(records))
	}
}

func TestGetByTeamDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add more than 100 records
	for range 150 {
		_, _ = store.Record("engineer-01", "team-a", "model-a", 100, 50, 0.01)
	}

	// Get with negative limit (should default to 100)
	records, err := store.GetByTeam("team-a", -1)
	if err != nil {
		t.Fatalf("GetByTeam failed: %v", err)
	}
	if len(records) != 100 {
		t.Errorf("len(records) = %d, want 100 (default limit)", len(records))
	}
}

func TestGetAllDefaultLimit(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Add more than 100 records
	for range 150 {
		_, _ = store.Record("engineer-01", "", "model-a", 100, 50, 0.01)
	}

	// Get with negative limit (should default to 100)
	records, err := store.GetAll(-5)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}
	if len(records) != 100 {
		t.Errorf("len(records) = %d, want 100 (default limit)", len(records))
	}
}

func TestCloseNilDB(t *testing.T) {
	store := NewStore("/tmp/test")
	// Close without opening - should not error
	if err := store.Close(); err != nil {
		t.Errorf("Close on nil DB should not error: %v", err)
	}
}
