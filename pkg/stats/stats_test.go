package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
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

// --- Helpers ---

func seedAgentsFile(t *testing.T, stateDir string, agents map[string]*agent.Agent) {
	t.Helper()
	agentsDir := filepath.Join(stateDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		t.Fatalf("marshal agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0644); err != nil {
		t.Fatalf("write agents.json: %v", err)
	}
}

func seedQueueFile(t *testing.T, stateDir string, items []queue.WorkItem) {
	t.Helper()
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		t.Fatalf("marshal queue: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "queue.json"), data, 0644); err != nil {
		t.Fatalf("write queue.json: %v", err)
	}
}

// --- collectAgentMetrics ---

func TestCollectAgentMetricsEmpty(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	s := New(stateDir)
	mgr := agent.NewWorkspaceManager(agentsDir, filepath.Dir(stateDir))
	q := queue.New(filepath.Join(stateDir, "queue.json"))

	s.collectAgentMetrics(mgr, q)

	if s.Agents.TotalAgents != 0 {
		t.Errorf("TotalAgents = %d, want 0", s.Agents.TotalAgents)
	}
	if s.Agents.ActiveAgents != 0 {
		t.Errorf("ActiveAgents = %d, want 0", s.Agents.ActiveAgents)
	}
	if len(s.Agents.AgentStats) != 0 {
		t.Errorf("AgentStats len = %d, want 0", len(s.Agents.AgentStats))
	}
}

func TestCollectAgentMetricsRoleCounts(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")

	agents := map[string]*agent.Agent{
		"coord-01": {Name: "coord-01", Role: agent.RoleCoordinator, State: agent.StateIdle, StartedAt: time.Now()},
		"coord-02": {Name: "coord-02", Role: agent.RoleCoordinator, State: agent.StateWorking, StartedAt: time.Now()},
		"eng-01":   {Name: "eng-01", Role: agent.RoleWorker, State: agent.StateIdle, StartedAt: time.Now()},
		"eng-02":   {Name: "eng-02", Role: agent.RoleWorker, State: agent.StateWorking, StartedAt: time.Now()},
		"eng-03":   {Name: "eng-03", Role: agent.RoleWorker, State: agent.StateDone, StartedAt: time.Now()},
	}
	seedAgentsFile(t, stateDir, agents)

	mgr := agent.NewWorkspaceManager(agentsDir, filepath.Dir(stateDir))
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	s := New(stateDir)
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	s.collectAgentMetrics(mgr, q)

	if s.Agents.TotalAgents != 5 {
		t.Errorf("TotalAgents = %d, want 5", s.Agents.TotalAgents)
	}
	if s.Agents.Coordinators != 2 {
		t.Errorf("Coordinators = %d, want 2", s.Agents.Coordinators)
	}
	if s.Agents.Workers != 3 {
		t.Errorf("Workers = %d, want 3", s.Agents.Workers)
	}
}

func TestCollectAgentMetricsStateCounts(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")

	agents := map[string]*agent.Agent{
		"a1": {Name: "a1", Role: agent.RoleWorker, State: agent.StateIdle, StartedAt: time.Now()},
		"a2": {Name: "a2", Role: agent.RoleWorker, State: agent.StateWorking, StartedAt: time.Now()},
		"a3": {Name: "a3", Role: agent.RoleWorker, State: agent.StateDone, StartedAt: time.Now()},
		"a4": {Name: "a4", Role: agent.RoleWorker, State: agent.StateStuck, StartedAt: time.Now()},
		"a5": {Name: "a5", Role: agent.RoleWorker, State: agent.StateError},
		"a6": {Name: "a6", Role: agent.RoleWorker, State: agent.StateStopped},
	}
	seedAgentsFile(t, stateDir, agents)

	mgr := agent.NewWorkspaceManager(agentsDir, filepath.Dir(stateDir))
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	s := New(stateDir)
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	s.collectAgentMetrics(mgr, q)

	if s.Agents.Idle != 1 {
		t.Errorf("Idle = %d, want 1", s.Agents.Idle)
	}
	if s.Agents.Working != 1 {
		t.Errorf("Working = %d, want 1", s.Agents.Working)
	}
	if s.Agents.Done != 1 {
		t.Errorf("Done = %d, want 1", s.Agents.Done)
	}
	if s.Agents.Stuck != 1 {
		t.Errorf("Stuck = %d, want 1", s.Agents.Stuck)
	}
	if s.Agents.Error != 1 {
		t.Errorf("Error = %d, want 1", s.Agents.Error)
	}
	if s.Agents.Stopped != 1 {
		t.Errorf("Stopped = %d, want 1", s.Agents.Stopped)
	}
	// Active = idle + working + done + stuck = 4
	if s.Agents.ActiveAgents != 4 {
		t.Errorf("ActiveAgents = %d, want 4", s.Agents.ActiveAgents)
	}
}

func TestCollectAgentMetricsUptime(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")

	startTime := time.Now().Add(-2 * time.Hour)
	agents := map[string]*agent.Agent{
		"running": {Name: "running", Role: agent.RoleWorker, State: agent.StateWorking, StartedAt: startTime},
		"stopped": {Name: "stopped", Role: agent.RoleWorker, State: agent.StateStopped, StartedAt: startTime},
		"no-time": {Name: "no-time", Role: agent.RoleWorker, State: agent.StateIdle},
	}
	seedAgentsFile(t, stateDir, agents)

	mgr := agent.NewWorkspaceManager(agentsDir, filepath.Dir(stateDir))
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	s := New(stateDir)
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	s.collectAgentMetrics(mgr, q)

	for _, stat := range s.Agents.AgentStats {
		switch stat.Name {
		case "running":
			if stat.Uptime < 1*time.Hour {
				t.Errorf("running agent uptime = %v, want >= 1h", stat.Uptime)
			}
		case "stopped":
			if stat.Uptime != 0 {
				t.Errorf("stopped agent uptime = %v, want 0", stat.Uptime)
			}
		case "no-time":
			// StartedAt is zero, so uptime should still be computed but may be huge;
			// the code checks !a.StartedAt.IsZero(), so it should be 0
			if stat.Uptime != 0 {
				t.Errorf("no-time agent uptime = %v, want 0", stat.Uptime)
			}
		}
	}
}

func TestCollectAgentMetricsTaskCounts(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")

	agents := map[string]*agent.Agent{
		"worker-1": {Name: "worker-1", Role: agent.RoleWorker, State: agent.StateWorking, StartedAt: time.Now()},
	}
	seedAgentsFile(t, stateDir, agents)

	mgr := agent.NewWorkspaceManager(agentsDir, filepath.Dir(stateDir))
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	// Create queue with items assigned to worker-1
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	q.Add("task 1", "", "")
	q.Add("task 2", "", "")
	q.Add("task 3", "", "")
	q.Assign("work-001", "worker-1")
	q.Assign("work-002", "worker-1")
	q.Assign("work-003", "worker-1")
	q.UpdateStatus("work-001", queue.StatusDone)
	q.UpdateStatus("work-002", queue.StatusDone)
	q.UpdateStatus("work-003", queue.StatusFailed)

	s := New(stateDir)
	s.collectAgentMetrics(mgr, q)

	if len(s.Agents.AgentStats) != 1 {
		t.Fatalf("AgentStats len = %d, want 1", len(s.Agents.AgentStats))
	}
	stat := s.Agents.AgentStats[0]
	if stat.TasksCompleted != 2 {
		t.Errorf("TasksCompleted = %d, want 2", stat.TasksCompleted)
	}
	if stat.TasksFailed != 1 {
		t.Errorf("TasksFailed = %d, want 1", stat.TasksFailed)
	}
}

func TestCollectAgentMetricsPerAgentFields(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")

	agents := map[string]*agent.Agent{
		"coord": {Name: "coord", Role: agent.RoleCoordinator, State: agent.StateIdle, StartedAt: time.Now()},
	}
	seedAgentsFile(t, stateDir, agents)

	mgr := agent.NewWorkspaceManager(agentsDir, filepath.Dir(stateDir))
	if err := mgr.LoadState(); err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	s := New(stateDir)
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	s.collectAgentMetrics(mgr, q)

	if len(s.Agents.AgentStats) != 1 {
		t.Fatalf("AgentStats len = %d, want 1", len(s.Agents.AgentStats))
	}
	stat := s.Agents.AgentStats[0]
	if stat.Name != "coord" {
		t.Errorf("Name = %q, want %q", stat.Name, "coord")
	}
	if stat.Role != string(agent.RoleCoordinator) {
		t.Errorf("Role = %q, want %q", stat.Role, agent.RoleCoordinator)
	}
	if stat.State != string(agent.StateIdle) {
		t.Errorf("State = %q, want %q", stat.State, agent.StateIdle)
	}
}

// --- Load integration ---

func TestLoadEmptyStateDir(t *testing.T) {
	stateDir := t.TempDir()

	s, err := Load(stateDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s.WorkItems.Total != 0 {
		t.Errorf("WorkItems.Total = %d, want 0", s.WorkItems.Total)
	}
	if s.Agents.TotalAgents != 0 {
		t.Errorf("Agents.TotalAgents = %d, want 0", s.Agents.TotalAgents)
	}
	if s.CollectedAt.IsZero() {
		t.Error("CollectedAt should not be zero")
	}
	if s.WorkspacePath != filepath.Dir(stateDir) {
		t.Errorf("WorkspacePath = %q, want %q", s.WorkspacePath, filepath.Dir(stateDir))
	}
}

func TestLoadWithQueueData(t *testing.T) {
	stateDir := t.TempDir()

	now := time.Now()
	items := []queue.WorkItem{
		{ID: "work-001", Title: "pending task", Status: queue.StatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: "work-002", Title: "[bug] fix crash", Status: queue.StatusDone, CreatedAt: now.Add(-1 * time.Hour), UpdatedAt: now},
		{ID: "work-003", Title: "[epic] redesign", Status: queue.StatusWorking, CreatedAt: now, UpdatedAt: now},
	}
	seedQueueFile(t, stateDir, items)

	s, err := Load(stateDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s.WorkItems.Total != 3 {
		t.Errorf("Total = %d, want 3", s.WorkItems.Total)
	}
	if s.WorkItems.Pending != 1 {
		t.Errorf("Pending = %d, want 1", s.WorkItems.Pending)
	}
	if s.WorkItems.Done != 1 {
		t.Errorf("Done = %d, want 1", s.WorkItems.Done)
	}
	if s.WorkItems.Working != 1 {
		t.Errorf("Working = %d, want 1", s.WorkItems.Working)
	}
	if s.WorkItems.Bugs != 1 {
		t.Errorf("Bugs = %d, want 1", s.WorkItems.Bugs)
	}
	if s.WorkItems.Epics != 1 {
		t.Errorf("Epics = %d, want 1", s.WorkItems.Epics)
	}
}

func TestLoadWithAgentsData(t *testing.T) {
	stateDir := t.TempDir()

	// Seed agents as already stopped so RefreshState won't change their state
	agents := map[string]*agent.Agent{
		"coord": {Name: "coord", Role: agent.RoleCoordinator, State: agent.StateStopped},
		"eng-1": {Name: "eng-1", Role: agent.RoleWorker, State: agent.StateStopped},
		"eng-2": {Name: "eng-2", Role: agent.RoleWorker, State: agent.StateStopped},
	}
	seedAgentsFile(t, stateDir, agents)

	s, err := Load(stateDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if s.Agents.TotalAgents != 3 {
		t.Errorf("TotalAgents = %d, want 3", s.Agents.TotalAgents)
	}
	if s.Agents.Coordinators != 1 {
		t.Errorf("Coordinators = %d, want 1", s.Agents.Coordinators)
	}
	if s.Agents.Workers != 2 {
		t.Errorf("Workers = %d, want 2", s.Agents.Workers)
	}
	if s.Agents.Stopped != 3 {
		t.Errorf("Stopped = %d, want 3", s.Agents.Stopped)
	}
}

func TestLoadPreservesHistorical(t *testing.T) {
	stateDir := t.TempDir()

	// Write existing stats.json with historical data
	existing := &Stats{
		TotalTasksEverCompleted: 50,
		TotalTasksEverFailed:    10,
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "stats.json"), data, 0644); err != nil {
		t.Fatalf("write stats.json: %v", err)
	}

	// Seed queue with fewer done items than historical
	now := time.Now()
	items := []queue.WorkItem{
		{ID: "work-001", Title: "task 1", Status: queue.StatusDone, CreatedAt: now, UpdatedAt: now},
		{ID: "work-002", Title: "task 2", Status: queue.StatusDone, CreatedAt: now, UpdatedAt: now},
	}
	seedQueueFile(t, stateDir, items)

	s, err := Load(stateDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Historical should be preserved (50 > current 2)
	if s.TotalTasksEverCompleted != 50 {
		t.Errorf("TotalTasksEverCompleted = %d, want 50", s.TotalTasksEverCompleted)
	}
	if s.TotalTasksEverFailed != 10 {
		t.Errorf("TotalTasksEverFailed = %d, want 10", s.TotalTasksEverFailed)
	}
}

func TestLoadInvalidQueueFile(t *testing.T) {
	stateDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(stateDir, "queue.json"), []byte("not json{{{"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(stateDir)
	if err == nil {
		t.Fatal("expected error for invalid queue.json")
	}
	if !strings.Contains(err.Error(), "failed to load queue") {
		t.Errorf("error = %q, want contains 'failed to load queue'", err)
	}
}

func TestLoadInvalidAgentsFile(t *testing.T) {
	stateDir := t.TempDir()
	agentsDir := filepath.Join(stateDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), []byte("not json{{{"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := Load(stateDir)
	if err == nil {
		t.Fatal("expected error for invalid agents.json")
	}
	if !strings.Contains(err.Error(), "failed to load agents") {
		t.Errorf("error = %q, want contains 'failed to load agents'", err)
	}
}
