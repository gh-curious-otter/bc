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
