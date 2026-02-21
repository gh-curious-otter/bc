package events

import (
	"testing"
	"time"
)

func TestDetectStuck_NoEvents(t *testing.T) {
	config := DefaultStuckConfig()
	detection := DetectStuck(nil, config)

	if detection.IsStuck {
		t.Error("expected not stuck with no events")
	}
}

func TestDetectStuck_NoActivity(t *testing.T) {
	// Create an old event
	events := []Event{
		{
			Timestamp: time.Now().Add(-20 * time.Minute),
			Type:      AgentReport,
			Agent:     "test-agent",
		},
	}

	config := StuckConfig{
		ActivityTimeout: 10 * time.Minute,
		WorkTimeout:     30 * time.Minute,
		MaxFailures:     3,
	}

	detection := DetectStuck(events, config)

	if !detection.IsStuck {
		t.Error("expected stuck due to no activity")
	}
	if detection.Reason != StuckNoActivity {
		t.Errorf("expected reason %s, got %s", StuckNoActivity, detection.Reason)
	}
}

func TestDetectStuck_RepeatedFailures(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Timestamp: now.Add(-5 * time.Minute), Type: WorkFailed, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-4 * time.Minute), Type: WorkFailed, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-3 * time.Minute), Type: WorkFailed, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-1 * time.Minute), Type: AgentReport, Agent: "test-agent"},
	}

	config := StuckConfig{
		ActivityTimeout: 10 * time.Minute,
		WorkTimeout:     30 * time.Minute,
		MaxFailures:     3,
	}

	detection := DetectStuck(events, config)

	if !detection.IsStuck {
		t.Error("expected stuck due to repeated failures")
	}
	if detection.Reason != StuckRepeatedFailures {
		t.Errorf("expected reason %s, got %s", StuckRepeatedFailures, detection.Reason)
	}
}

func TestDetectStuck_WorkTimeout(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Timestamp: now.Add(-45 * time.Minute), Type: WorkStarted, Agent: "test-agent", Message: "long-task"},
		{Timestamp: now.Add(-1 * time.Minute), Type: AgentReport, Agent: "test-agent"},
	}

	config := StuckConfig{
		ActivityTimeout: 10 * time.Minute,
		WorkTimeout:     30 * time.Minute,
		MaxFailures:     3,
	}

	detection := DetectStuck(events, config)

	if !detection.IsStuck {
		t.Error("expected stuck due to work timeout")
	}
	if detection.Reason != StuckWorkTimeout {
		t.Errorf("expected reason %s, got %s", StuckWorkTimeout, detection.Reason)
	}
}

func TestDetectStuck_Healthy(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Timestamp: now.Add(-5 * time.Minute), Type: WorkStarted, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-2 * time.Minute), Type: WorkCompleted, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-1 * time.Minute), Type: AgentReport, Agent: "test-agent"},
	}

	config := DefaultStuckConfig()

	detection := DetectStuck(events, config)

	if detection.IsStuck {
		t.Errorf("expected not stuck, got reason: %s", detection.Reason)
	}
}

func TestDetectStuck_FailureResetBySuccess(t *testing.T) {
	now := time.Now()
	events := []Event{
		{Timestamp: now.Add(-10 * time.Minute), Type: WorkFailed, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-9 * time.Minute), Type: WorkFailed, Agent: "test-agent", Message: "task1"},
		{Timestamp: now.Add(-5 * time.Minute), Type: WorkCompleted, Agent: "test-agent", Message: "task2"},
		{Timestamp: now.Add(-4 * time.Minute), Type: WorkFailed, Agent: "test-agent", Message: "task3"},
		{Timestamp: now.Add(-1 * time.Minute), Type: AgentReport, Agent: "test-agent"},
	}

	config := StuckConfig{
		ActivityTimeout: 15 * time.Minute,
		WorkTimeout:     30 * time.Minute,
		MaxFailures:     3,
	}

	detection := DetectStuck(events, config)

	// Only 1 failure after the success, should not be stuck
	if detection.IsStuck {
		t.Errorf("expected not stuck after success reset, got reason: %s", detection.Reason)
	}
}

func TestDefaultStuckConfig(t *testing.T) {
	config := DefaultStuckConfig()

	if config.ActivityTimeout != 10*time.Minute {
		t.Errorf("expected ActivityTimeout 10m, got %s", config.ActivityTimeout)
	}
	if config.WorkTimeout != 30*time.Minute {
		t.Errorf("expected WorkTimeout 30m, got %s", config.WorkTimeout)
	}
	if config.MaxFailures != 3 {
		t.Errorf("expected MaxFailures 3, got %d", config.MaxFailures)
	}
}

func TestDetectAllStuck(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/events.jsonl"
	log := NewLog(logPath)

	now := time.Now()

	// Add events for agent-1 (healthy)
	if err := log.Append(Event{Timestamp: now.Add(-5 * time.Minute), Type: WorkStarted, Agent: "agent-1"}); err != nil {
		t.Fatalf("failed to append event: %v", err)
	}
	if err := log.Append(Event{Timestamp: now.Add(-2 * time.Minute), Type: WorkCompleted, Agent: "agent-1"}); err != nil {
		t.Fatalf("failed to append event: %v", err)
	}
	if err := log.Append(Event{Timestamp: now.Add(-1 * time.Minute), Type: AgentReport, Agent: "agent-1"}); err != nil {
		t.Fatalf("failed to append event: %v", err)
	}

	// Add events for agent-2 (stuck - no activity)
	if err := log.Append(Event{Timestamp: now.Add(-25 * time.Minute), Type: AgentReport, Agent: "agent-2"}); err != nil {
		t.Fatalf("failed to append event: %v", err)
	}

	config := StuckConfig{
		ActivityTimeout: 10 * time.Minute,
		WorkTimeout:     30 * time.Minute,
		MaxFailures:     3,
	}

	results, err := DetectAllStuck(log, []string{"agent-1", "agent-2"}, config)
	if err != nil {
		t.Fatalf("DetectAllStuck failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// agent-1 should be healthy
	if results[0].AgentName != "agent-1" {
		t.Errorf("expected agent-1 first, got %s", results[0].AgentName)
	}
	if results[0].IsStuck {
		t.Errorf("expected agent-1 to be healthy, got stuck: %s", results[0].Reason)
	}

	// agent-2 should be stuck
	if results[1].AgentName != "agent-2" {
		t.Errorf("expected agent-2 second, got %s", results[1].AgentName)
	}
	if !results[1].IsStuck {
		t.Error("expected agent-2 to be stuck")
	}
	if results[1].Reason != StuckNoActivity {
		t.Errorf("expected reason %s, got %s", StuckNoActivity, results[1].Reason)
	}
}

func TestDetectAllStuck_NoAgents(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/events.jsonl"
	log := NewLog(logPath)

	config := DefaultStuckConfig()

	results, err := DetectAllStuck(log, []string{}, config)
	if err != nil {
		t.Fatalf("DetectAllStuck failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty agent list, got %d", len(results))
	}
}

func TestDetectAllStuck_AgentWithNoEvents(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := tmpDir + "/events.jsonl"
	log := NewLog(logPath)

	config := DefaultStuckConfig()

	// Test with agent that has no events
	results, err := DetectAllStuck(log, []string{"nonexistent-agent"}, config)
	if err != nil {
		t.Fatalf("DetectAllStuck failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	// Agent with no events should not be marked stuck (no events = no data to analyze)
	if results[0].IsStuck {
		t.Error("expected agent with no events to not be stuck")
	}
}
