package cost

import (
	"testing"
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
