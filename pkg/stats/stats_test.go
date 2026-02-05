package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/queue"
)

// --- New ---

func TestNew(t *testing.T) {
	s := New("/tmp/test-state")
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.path != "/tmp/test-state/stats.json" {
		t.Errorf("path = %q, want %q", s.path, "/tmp/test-state/stats.json")
	}
	if s.CollectedAt.IsZero() {
		t.Error("CollectedAt should not be zero")
	}
}

// --- collectWorkItemMetrics ---

func TestCollectWorkItemMetricsEmpty(t *testing.T) {
	s := New(t.TempDir())
	q := queue.New(filepath.Join(t.TempDir(), "q.json"))

	s.collectWorkItemMetrics(q)

	if s.WorkItems.Total != 0 {
		t.Errorf("Total = %d, want 0", s.WorkItems.Total)
	}
	if s.WorkItems.CompletionRate != 0 {
		t.Errorf("CompletionRate = %f, want 0", s.WorkItems.CompletionRate)
	}
}

func TestCollectWorkItemMetricsStatusCounts(t *testing.T) {
	s := New(t.TempDir())
	q := queue.New(filepath.Join(t.TempDir(), "q.json"))

	q.Add("Pending task", "", "")
	q.Add("Assigned task", "", "")
	q.Add("Working task", "", "")
	q.Add("Done task", "", "")
	q.Add("Failed task", "", "")

	q.Assign("work-002", "agent-1")
	q.UpdateStatus("work-003", queue.StatusWorking)
	q.UpdateStatus("work-004", queue.StatusDone)
	q.UpdateStatus("work-005", queue.StatusFailed)

	s.collectWorkItemMetrics(q)

	if s.WorkItems.Total != 5 {
		t.Errorf("Total = %d, want 5", s.WorkItems.Total)
	}
	if s.WorkItems.Pending != 1 {
		t.Errorf("Pending = %d, want 1", s.WorkItems.Pending)
	}
	if s.WorkItems.Assigned != 1 {
		t.Errorf("Assigned = %d, want 1", s.WorkItems.Assigned)
	}
	if s.WorkItems.Working != 1 {
		t.Errorf("Working = %d, want 1", s.WorkItems.Working)
	}
	if s.WorkItems.Done != 1 {
		t.Errorf("Done = %d, want 1", s.WorkItems.Done)
	}
	if s.WorkItems.Failed != 1 {
		t.Errorf("Failed = %d, want 1", s.WorkItems.Failed)
	}
}

func TestCollectWorkItemMetricsRates(t *testing.T) {
	s := New(t.TempDir())
	q := queue.New(filepath.Join(t.TempDir(), "q.json"))

	// 4 items: 2 done, 1 failed, 1 pending
	q.Add("a", "", "")
	q.Add("b", "", "")
	q.Add("c", "", "")
	q.Add("d", "", "")

	q.UpdateStatus("work-001", queue.StatusDone)
	q.UpdateStatus("work-002", queue.StatusDone)
	q.UpdateStatus("work-003", queue.StatusFailed)

	s.collectWorkItemMetrics(q)

	expectedCompletion := 0.5 // 2/4
	if s.WorkItems.CompletionRate != expectedCompletion {
		t.Errorf("CompletionRate = %f, want %f", s.WorkItems.CompletionRate, expectedCompletion)
	}
	expectedFailure := 0.25 // 1/4
	if s.WorkItems.FailureRate != expectedFailure {
		t.Errorf("FailureRate = %f, want %f", s.WorkItems.FailureRate, expectedFailure)
	}
}

func TestCollectWorkItemMetricsTypeClassification(t *testing.T) {
	tests := []struct {
		title    string
		wantType string // "epic", "bug", "task", "other"
	}{
		{"[epic] Big project", "epic"},
		{"Epic: Redesign system", "epic"},
		{"[bug] Crash on login", "bug"},
		{"Bug: Fix null pointer", "bug"},
		{"Fix authentication", "bug"},
		{"fix broken tests", "bug"},
		{"[task] Add logging", "task"},
		{"Task: Implement cache", "task"},
		{"Add new feature", "other"},
		{"Refactor database layer", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			s := New(t.TempDir())
			q := queue.New(filepath.Join(t.TempDir(), "q.json"))
			q.Add(tt.title, "", "")

			s.collectWorkItemMetrics(q)

			var got int
			switch tt.wantType {
			case "epic":
				got = s.WorkItems.Epics
			case "bug":
				got = s.WorkItems.Bugs
			case "task":
				got = s.WorkItems.Tasks
			case "other":
				got = s.WorkItems.Other
			}

			if got != 1 {
				t.Errorf("%q classified as: epics=%d bugs=%d tasks=%d other=%d, want %s=1",
					tt.title, s.WorkItems.Epics, s.WorkItems.Bugs, s.WorkItems.Tasks, s.WorkItems.Other, tt.wantType)
			}
		})
	}
}

func TestCollectWorkItemMetricsHistorical(t *testing.T) {
	s := New(t.TempDir())
	s.TotalTasksEverCompleted = 10
	s.TotalTasksEverFailed = 3

	q := queue.New(filepath.Join(t.TempDir(), "q.json"))
	for i := 0; i < 15; i++ {
		q.Add("task", "", "")
		q.UpdateStatus(q.ListAll()[i].ID, queue.StatusDone)
	}

	s.collectWorkItemMetrics(q)

	// Should update historical totals when current exceeds them
	if s.TotalTasksEverCompleted != 15 {
		t.Errorf("TotalTasksEverCompleted = %d, want 15", s.TotalTasksEverCompleted)
	}
}

// --- Save / Load ---

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	s := New(dir)
	s.WorkspacePath = "/test/workspace"
	s.WorkItems.Total = 10
	s.WorkItems.Done = 7
	s.Agents.TotalAgents = 3
	s.TotalTasksEverCompleted = 15

	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	data, err := os.ReadFile(filepath.Join(dir, "stats.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("saved file is empty")
	}

	// Load into new struct
	var loaded Stats
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded.WorkspacePath != "/test/workspace" {
		t.Errorf("WorkspacePath = %q, want %q", loaded.WorkspacePath, "/test/workspace")
	}
	if loaded.WorkItems.Total != 10 {
		t.Errorf("Total = %d, want 10", loaded.WorkItems.Total)
	}
	if loaded.TotalTasksEverCompleted != 15 {
		t.Errorf("TotalTasksEverCompleted = %d, want 15", loaded.TotalTasksEverCompleted)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	s := New(dir)
	s.WorkItems.Total = 1

	if err := s.Save(); err != nil {
		t.Fatalf("Save to nested dir: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "stats.json")); err != nil {
		t.Errorf("stats.json not created: %v", err)
	}
}

// --- Utilization ---

func TestUtilization(t *testing.T) {
	tests := []struct {
		name    string
		active  int
		working int
		want    float64
	}{
		{"no agents", 0, 0, 0},
		{"all idle", 5, 0, 0},
		{"all working", 4, 4, 1.0},
		{"half working", 6, 3, 0.5},
		{"one of three", 3, 1, 1.0 / 3.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Stats{}
			s.Agents.ActiveAgents = tt.active
			s.Agents.Working = tt.working

			got := s.Utilization()
			if got != tt.want {
				t.Errorf("Utilization() = %f, want %f", got, tt.want)
			}
		})
	}
}

// --- formatDuration ---

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0s"},
		{"seconds", 45 * time.Second, "45s"},
		{"minutes and seconds", 3*time.Minute + 12*time.Second, "3m 12s"},
		{"hours and minutes", 2*time.Hour + 30*time.Minute, "2h 30m"},
		{"hours only", 1 * time.Hour, "1h 0m"},
		{"sub-second rounds down", 500 * time.Millisecond, "1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.d)
			if got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

// --- Summary ---

func TestSummaryContainsExpectedSections(t *testing.T) {
	s := &Stats{
		CollectedAt: time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		WorkItems: WorkItemMetrics{
			Total:   10,
			Done:    7,
			Pending: 3,
		},
		Agents: AgentMetrics{
			TotalAgents:  3,
			ActiveAgents: 2,
			Working:      1,
			AgentStats: []AgentStat{
				{Name: "coord", Role: "coordinator", State: "working", TasksCompleted: 5, Uptime: 1 * time.Hour},
			},
		},
	}

	summary := s.Summary()

	expectedParts := []string{
		"Workspace Stats",
		"Work Items",
		"Total:    10",
		"Done:     7",
		"Pending:  3",
		"Agents",
		"Total:  3 (2 active)",
		"Per Agent:",
		"coord",
	}

	for _, part := range expectedParts {
		if !strings.Contains(summary, part) {
			t.Errorf("Summary missing expected content: %q\nGot:\n%s", part, summary)
		}
	}
}

func TestSummaryAvgTimeShownWhenNonZero(t *testing.T) {
	s := &Stats{
		CollectedAt: time.Now(),
		WorkItems: WorkItemMetrics{
			AvgTimeToComplete: 30 * time.Minute,
		},
	}

	summary := s.Summary()
	if !strings.Contains(summary, "Avg Time:") {
		t.Error("Summary should show Avg Time when non-zero")
	}
}

func TestSummaryAvgTimeHiddenWhenZero(t *testing.T) {
	s := &Stats{
		CollectedAt: time.Now(),
	}

	summary := s.Summary()
	if strings.Contains(summary, "Avg Time:") {
		t.Error("Summary should not show Avg Time when zero")
	}
}

func TestSummaryNoAgentStatsSection(t *testing.T) {
	s := &Stats{
		CollectedAt: time.Now(),
	}

	summary := s.Summary()
	if strings.Contains(summary, "Per Agent:") {
		t.Error("Summary should not show Per Agent section with no agents")
	}
}
