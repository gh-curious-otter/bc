package cmd

import (
	"testing"
	"time"
)

func TestParseAuditDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  0,
		},
		{
			name:  "7 days",
			input: "7d",
			want:  7 * 24 * time.Hour,
		},
		{
			name:  "30 days",
			input: "30d",
			want:  30 * 24 * time.Hour,
		},
		{
			name:  "24 hours",
			input: "24h",
			want:  24 * time.Hour,
		},
		{
			name:  "1h30m",
			input: "1h30m",
			want:  90 * time.Minute,
		},
		{
			name:    "invalid",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "bad days format",
			input:   "xd",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAuditDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAuditDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseAuditDuration(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAuditEventStruct(t *testing.T) {
	ev := AuditEvent{
		Timestamp: "2024-01-15T10:30:00Z",
		Type:      "agent.spawned",
		Agent:     "eng-01",
		Message:   "Test message",
		Data:      map[string]any{"key": "value"},
	}

	if ev.Timestamp != "2024-01-15T10:30:00Z" {
		t.Errorf("expected timestamp '2024-01-15T10:30:00Z', got %q", ev.Timestamp)
	}
	if ev.Type != "agent.spawned" {
		t.Errorf("expected type 'agent.spawned', got %q", ev.Type)
	}
	if ev.Agent != "eng-01" {
		t.Errorf("expected agent 'eng-01', got %q", ev.Agent)
	}
	if ev.Message != "Test message" {
		t.Errorf("expected message 'Test message', got %q", ev.Message)
	}
	if ev.Data["key"] != "value" {
		t.Errorf("expected data key 'value', got %v", ev.Data["key"])
	}
}

func TestAuditReportStruct(t *testing.T) {
	now := time.Now()
	start := now.Add(-7 * 24 * time.Hour)
	report := AuditReport{
		Generated:    now,
		Period:       "7d",
		PeriodStart:  start,
		PeriodEnd:    now,
		TotalEvents:  100,
		EventsByType: map[string]int{"agent.spawned": 10, "work.completed": 50},
	}

	if report.TotalEvents != 100 {
		t.Errorf("expected total events 100, got %d", report.TotalEvents)
	}
	if report.Period != "7d" {
		t.Errorf("expected period '7d', got %q", report.Period)
	}
	if !report.Generated.Equal(now) {
		t.Errorf("expected generated time to match")
	}
	if !report.PeriodStart.Equal(start) {
		t.Errorf("expected period start time to match")
	}
	if !report.PeriodEnd.Equal(now) {
		t.Errorf("expected period end time to match")
	}
	if report.EventsByType["agent.spawned"] != 10 {
		t.Errorf("expected 10 agent.spawned events, got %d", report.EventsByType["agent.spawned"])
	}
}

func TestCostAuditSummaryStruct(t *testing.T) {
	summary := CostAuditSummary{
		TotalCostUSD: 10.50,
		TotalTokens:  50000,
		InputTokens:  30000,
		OutputTokens: 20000,
		RecordCount:  25,
		CostByAgent:  map[string]float64{"eng-01": 5.25, "eng-02": 5.25},
		CostByModel:  map[string]float64{"claude-3": 10.50},
	}

	if summary.TotalCostUSD != 10.50 {
		t.Errorf("expected total cost 10.50, got %f", summary.TotalCostUSD)
	}
	if summary.TotalTokens != 50000 {
		t.Errorf("expected total tokens 50000, got %d", summary.TotalTokens)
	}
	if summary.InputTokens != 30000 {
		t.Errorf("expected input tokens 30000, got %d", summary.InputTokens)
	}
	if summary.OutputTokens != 20000 {
		t.Errorf("expected output tokens 20000, got %d", summary.OutputTokens)
	}
	if summary.RecordCount != 25 {
		t.Errorf("expected record count 25, got %d", summary.RecordCount)
	}
	if len(summary.CostByAgent) != 2 {
		t.Errorf("expected 2 agents in cost breakdown, got %d", len(summary.CostByAgent))
	}
	if summary.CostByModel["claude-3"] != 10.50 {
		t.Errorf("expected claude-3 cost 10.50, got %f", summary.CostByModel["claude-3"])
	}
}

func TestErrorAuditSummaryStruct(t *testing.T) {
	summary := ErrorAuditSummary{
		TotalFailures:   5,
		HealthFailures:  2,
		WorkFailures:    3,
		FailuresByAgent: map[string]int{"eng-01": 3, "eng-02": 2},
	}

	if summary.TotalFailures != 5 {
		t.Errorf("expected total failures 5, got %d", summary.TotalFailures)
	}
	if summary.HealthFailures != 2 {
		t.Errorf("expected health failures 2, got %d", summary.HealthFailures)
	}
	if summary.WorkFailures != 3 {
		t.Errorf("expected work failures 3, got %d", summary.WorkFailures)
	}
	if summary.HealthFailures+summary.WorkFailures != summary.TotalFailures {
		t.Errorf("failure counts don't add up")
	}
	if summary.FailuresByAgent["eng-01"] != 3 {
		t.Errorf("expected eng-01 failures 3, got %d", summary.FailuresByAgent["eng-01"])
	}
}
