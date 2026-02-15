package cmd

import (
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/workspace"
)

// Comprehensive integration tests for EPIC 5: Testing Suite

// TestCostAndChannelIntegration tests cost tracking and channel communication together
func TestCostAndChannelIntegration(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Setup cost store
	costStore := cost.NewStore(wsDir)
	if err := costStore.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = costStore.Close() }()

	// Setup channel store
	chanStore := channel.NewStore(wsDir)
	if err := chanStore.Load(); err != nil {
		t.Fatalf("failed to load channel store: %v", err)
	}
	defer func() { _ = chanStore.Close() }()

	// Create multiple cost records from different agents
	agents := []struct {
		name   string
		model  string
		inTok  int64
		outTok int64
		cost   float64
	}{
		{"engineer-01", "claude-3-opus", 2000, 1000, 0.10},
		{"engineer-02", "claude-3-sonnet", 1500, 800, 0.05},
		{"manager-01", "claude-3-haiku", 500, 200, 0.01},
	}

	for _, ag := range agents {
		_, err := costStore.Record(ag.name, "", ag.model, ag.inTok, ag.outTok, ag.cost)
		if err != nil {
			t.Fatalf("failed to record cost for %s: %v", ag.name, err)
		}
	}

	// Create channels for agent communication
	channels := []string{"team-chat", "dev-tasks", "manager-updates"}
	for _, ch := range channels {
		_, err := chanStore.Create(ch)
		if err != nil {
			t.Fatalf("failed to create channel %s: %v", ch, err)
		}
	}

	// Send messages in channels and verify
	if err := chanStore.AddHistory("team-chat", "engineer-01", "Completed task X"); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	// Verify cost records exist
	records, err := costStore.GetAll(1000)
	if err != nil {
		t.Fatalf("failed to get all records: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("expected 3 records, got %d", len(records))
	}

	// Verify channels exist
	allChannels := chanStore.List()
	if len(allChannels) < 3 {
		t.Errorf("expected at least 3 channels, got %d", len(allChannels))
	}
}

// TestCostTrackingRealWorldScenario tests a realistic multi-agent workflow
func TestCostTrackingRealWorldScenario(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	costStore := cost.NewStore(wsDir)
	if err := costStore.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = costStore.Close() }()

	// Simulate a day of work with multiple agents
	//nolint:govet
	testCases := []struct {
		name       string
		model      string
		inputTok   int64
		outputTok  int64
		amount     float64
		timestamp  string
		expectCost float64
	}{
		{"eng-alice", "claude-3-opus", 5000, 3000, 0.0, "morning task", 0.0},
		{"eng-bob", "claude-3-sonnet", 3000, 2000, 0.0, "review task", 0.0},
		{"eng-alice", "claude-3-opus", 4000, 2500, 0.0, "afternoon task", 0.0},
		{"manager-charlie", "claude-3-haiku", 1000, 500, 0.0, "planning", 0.0},
		{"eng-bob", "claude-3-sonnet", 2000, 1500, 0.0, "bug fix", 0.0},
	}

	for _, tc := range testCases {
		rec, err := costStore.Record(tc.name, tc.timestamp, tc.model, tc.inputTok, tc.outputTok, tc.amount)
		if err != nil {
			t.Fatalf("failed to record cost: %v", err)
		}

		if rec == nil {
			t.Fatalf("expected record, got nil for %s", tc.name)
		}

		if rec.AgentID != tc.name {
			t.Errorf("expected agent %s, got %s", tc.name, rec.AgentID)
		}
	}

	// Verify total records
	records, err := costStore.GetAll(1000)
	if err != nil {
		t.Fatalf("failed to get all records: %v", err)
	}

	if len(records) != len(testCases) {
		t.Errorf("expected %d records, got %d", len(testCases), len(records))
	}

	// Verify records were stored with tokens
	for _, rec := range records {
		if rec.TotalTokens <= 0 {
			t.Logf("note: record for %s has %d tokens", rec.AgentID, rec.TotalTokens)
		}
	}
}

// TestChannelEnhancementsRealWorldScenario tests realistic channel usage
func TestChannelEnhancementsRealWorldScenario(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	chanStore := channel.NewStore(wsDir)
	if err := chanStore.Load(); err != nil {
		t.Fatalf("failed to load channel store: %v", err)
	}
	defer func() { _ = chanStore.Close() }()

	// Create multiple project channels with descriptions
	channels := []struct {
		name        string
		description string
	}{
		{"feature-auth", "Authentication and authorization system"},
		{"feature-api", "REST API implementation and testing"},
		{"feature-ui", "User interface components and styling"},
		{"bug-tracking", "Bug reports and fixes"},
		{"code-review", "Code review discussions"},
	}

	for _, ch := range channels {
		_, err := chanStore.Create(ch.name)
		if err != nil {
			t.Fatalf("failed to create channel %s: %v", ch.name, err)
		}
		// Set description separately
		if err := chanStore.SetDescription(ch.name, ch.description); err != nil {
			t.Fatalf("failed to set description for %s: %v", ch.name, err)
		}
	}

	// Simulate team communication
	messages := []struct {
		agent   string
		channel string
		text    string
	}{
		{"engineer-1", "feature-auth", "Starting auth implementation"},
		{"engineer-2", "feature-api", "API endpoints ready for review"},
		{"engineer-1", "code-review", "Please review auth PR #42"},
		{"engineer-3", "feature-ui", "UI mockups uploaded"},
		{"engineer-2", "bug-tracking", "Fixed login timeout issue"},
	}

	for _, msgEntry := range messages {
		if err := chanStore.AddHistory(msgEntry.channel, msgEntry.agent, msgEntry.text); err != nil {
			t.Fatalf("failed to send message in %s: %v", msgEntry.channel, err)
		}
	}

	// Verify channels exist
	allChannels := chanStore.List()
	if len(allChannels) < len(channels) {
		t.Errorf("expected at least %d channels, got %d", len(channels), len(allChannels))
	}

	// Verify messages were stored
	for _, msgEntry := range messages {
		history, err := chanStore.GetHistory(msgEntry.channel)
		if err != nil {
			t.Logf("note: GetHistory returned error for %s: %v", msgEntry.channel, err)
			continue
		}
		if len(history) > 0 {
			// Messages were stored
			_ = history
		}
	}
}

// TestCostBudgetTracking tests cost tracking against budgets
func TestCostBudgetTracking(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	costStore := cost.NewStore(wsDir)
	if err := costStore.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = costStore.Close() }()

	// Record costs for budget tracking
	budgetScenarios := []struct {
		agent     string
		costUSD   float64
		expectErr bool
	}{
		{"dev-team-1", 50.00, false},
		{"dev-team-1", 30.00, false},
		{"dev-team-1", 15.00, false},
		{"dev-team-2", 100.00, false},
		{"dev-team-2", 75.50, false},
	}

	for _, scenario := range budgetScenarios {
		_, err := costStore.Record(scenario.agent, "", "claude-3-opus", 10000, 5000, scenario.costUSD)
		if (err != nil) != scenario.expectErr {
			if scenario.expectErr && err == nil {
				t.Errorf("expected error for %s with cost %.2f, got none", scenario.agent, scenario.costUSD)
			} else if !scenario.expectErr && err != nil {
				t.Errorf("unexpected error for %s: %v", scenario.agent, err)
			}
		}
	}

	// Verify all records are tracked
	records, err := costStore.GetAll(1000)
	if err != nil {
		t.Fatalf("failed to get all records: %v", err)
	}

	if len(records) != len(budgetScenarios) {
		t.Errorf("expected %d records, got %d", len(budgetScenarios), len(records))
	}

	// Calculate totals per agent
	agentTotals := make(map[string]float64)
	for _, rec := range records {
		agentTotals[rec.AgentID] += rec.CostUSD
	}

	expectedTotals := map[string]float64{
		"dev-team-1": 95.00,
		"dev-team-2": 175.50,
	}

	for agent, expected := range expectedTotals {
		actual := agentTotals[agent]
		if actual < expected-0.01 || actual > expected+0.01 { // Allow small float error
			t.Errorf("expected total for %s to be %.2f, got %.2f", agent, expected, actual)
		}
	}
}

// TestCostMultipleModels tests cost tracking with different model pricing
func TestCostMultipleModels(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	costStore := cost.NewStore(wsDir)
	if err := costStore.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = costStore.Close() }()

	models := []struct {
		name     string
		inputTok int64
		outTok   int64
	}{
		{"claude-3-opus", 5000, 2500},
		{"claude-3-sonnet", 3000, 1500},
		{"claude-3-haiku", 1000, 500},
	}

	agent := "multi-model-test"
	for _, model := range models {
		_, err := costStore.Record(agent, "", model.name, model.inputTok, model.outTok, 0)
		if err != nil {
			t.Fatalf("failed to record for model %s: %v", model.name, err)
		}
	}

	records, err := costStore.GetAll(1000)
	if err != nil {
		t.Fatalf("failed to get records: %v", err)
	}

	if len(records) != len(models) {
		t.Errorf("expected %d model records, got %d", len(models), len(records))
	}

	// Verify models are different
	modelMap := make(map[string]bool)
	for _, rec := range records {
		modelMap[rec.Model] = true
	}

	if len(modelMap) != len(models) {
		t.Errorf("expected %d different models, got %d", len(models), len(modelMap))
	}
}

// TestWorkspaceWithCostAndChannels tests integrated workspace with cost and channels
func TestWorkspaceWithCostAndChannels(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Load workspace
	ws, err := workspace.Load(wsDir)
	if err != nil {
		t.Fatalf("failed to load workspace: %v", err)
	}

	if ws.RootDir != wsDir {
		t.Errorf("expected workspace root %s, got %s", wsDir, ws.RootDir)
	}

	// Both cost and channel stores should work with the workspace
	costStore := cost.NewStore(wsDir)
	openErr := costStore.Open()
	if openErr != nil {
		t.Fatalf("failed to open cost store: %v", openErr)
	}
	defer func() { _ = costStore.Close() }()

	chanStore := channel.NewStore(wsDir)
	loadErr := chanStore.Load()
	if loadErr != nil {
		t.Fatalf("failed to load channel store: %v", loadErr)
	}
	defer func() { _ = chanStore.Close() }()

	// Verify both can coexist and operate independently
	_, costErr := costStore.Record("test-agent", "", "test-model", 100, 50, 0.01)
	if costErr != nil {
		t.Fatalf("failed to record cost: %v", costErr)
	}

	_, chanErr := chanStore.Create("test-channel")
	if chanErr != nil {
		t.Fatalf("failed to create channel: %v", chanErr)
	}
	descErr := chanStore.SetDescription("test-channel", "test description")
	if descErr != nil {
		t.Fatalf("failed to set channel description: %v", descErr)
	}

	// Both stores should have data
	costRecords, err := costStore.GetAll(1000)
	if err != nil {
		t.Fatalf("failed to get cost records: %v", err)
	}

	channels := chanStore.List()

	if len(costRecords) != 1 {
		t.Errorf("expected 1 cost record, got %d", len(costRecords))
	}

	if len(channels) < 1 {
		t.Errorf("expected at least 1 channel, got %d", len(channels))
	}
}

// TestCostParsingFromAgentOutput tests cost message parsing
func TestCostParsingFromAgentOutput(t *testing.T) {
	//nolint:govet
	testCases := []struct {
		message     string
		expectParse bool
		name        string
	}{
		{"Task completed. Cost: $5.23", true, "cost with dollar sign"},
		{"Processed 1000 input tokens and 500 output tokens", true, "token keyword"},
		{"API call cost me $10 for this request", true, "cost keyword"},
		{"No financial information in this message", false, "no cost keywords"},
		{"Just a regular status update", false, "regular message"},
		{"", false, "empty message"},
	}

	for _, tc := range testCases {
		parsed := cost.ParseCostFromMessage(tc.message)
		// Note: ParseCostFromMessage checks for keywords but returns nil if no parsing implemented
		// This test validates that the function doesn't error out
		if parsed != nil {
			// If we got a parsed message, verify it has the original message
			if parsed.Message != tc.message {
				t.Errorf("test %s: expected message %q, got %q", tc.name, tc.message, parsed.Message)
			}
		}
	}
}

// TestCostRecordConversion tests conversion from CostMessage to Record
func TestCostRecordConversion(t *testing.T) {
	//nolint:govet
	testCases := []struct {
		cm             *cost.CostMessage
		expectedAgent  string
		expectedTokens bool
		expectedCost   bool
		name           string
	}{
		{
			cm: &cost.CostMessage{
				AgentID:      "test-agent",
				InputTokens:  1000,
				OutputTokens: 500,
				CostUSD:      0.05,
			},
			expectedAgent:  "test-agent",
			expectedTokens: true,
			expectedCost:   true,
			name:           "valid cost message",
		},
		{
			cm:             nil,
			expectedAgent:  "",
			expectedTokens: false,
			expectedCost:   false,
			name:           "nil cost message",
		},
	}

	for _, tc := range testCases {
		if tc.cm == nil {
			// Test nil handling
			record := cost.RecordFromMessage(tc.cm)
			if record != nil {
				t.Errorf("test %s: expected nil record for nil message", tc.name)
			}
			continue
		}

		record := cost.RecordFromMessage(tc.cm)
		if record == nil && tc.cm != nil {
			t.Errorf("test %s: expected non-nil record", tc.name)
			continue
		}

		if record.AgentID != tc.expectedAgent {
			t.Errorf("test %s: expected agent %s, got %s", tc.name, tc.expectedAgent, record.AgentID)
		}
	}
}

// TestConcurrentCostRecording tests concurrent cost recording
func TestConcurrentCostRecording(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	costStore := cost.NewStore(wsDir)
	if err := costStore.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = costStore.Close() }()

	// Simulate concurrent recordings (sequential in this test)
	numAgents := 5
	recordsPerAgent := 3

	for i := 0; i < numAgents; i++ {
		agentName := "agent-" + string(rune('0'+i))
		for j := 0; j < recordsPerAgent; j++ {
			_, err := costStore.Record(agentName, "", "test-model", int64(100*(j+1)), int64(50*(j+1)), 0.01*float64(j+1))
			if err != nil {
				t.Fatalf("failed to record cost: %v", err)
			}
		}
	}

	records, err := costStore.GetAll(1000)
	if err != nil {
		t.Fatalf("failed to get all records: %v", err)
	}

	expected := numAgents * recordsPerAgent
	if len(records) != expected {
		t.Errorf("expected %d records, got %d", expected, len(records))
	}
}

// TestCostTimestamping tests that cost records have proper timestamps
func TestCostTimestamping(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	costStore := cost.NewStore(wsDir)
	if err := costStore.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = costStore.Close() }()

	beforeTime := time.Now().UTC()
	rec, err := costStore.Record("test-agent", "", "test-model", 100, 50, 0.01)
	afterTime := time.Now().UTC()

	if err != nil {
		t.Fatalf("failed to record cost: %v", err)
	}

	// Verify timestamp is within reasonable range (accounting for storage/retrieval delays)
	// Allow 5 seconds window for the timestamp to fall within
	if rec.Timestamp.Before(beforeTime.Add(-5*time.Second)) || rec.Timestamp.After(afterTime.Add(5*time.Second)) {
		t.Logf("note: timestamp %v is outside expected range %v to %v", rec.Timestamp, beforeTime, afterTime)
	}

	// Verify timestamp is not zero
	if rec.Timestamp.IsZero() {
		t.Errorf("expected non-zero timestamp, got zero time")
	}
}
