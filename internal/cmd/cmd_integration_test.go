package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/workspace"
)

func durationFromSeconds(s int) time.Duration {
	return time.Duration(s) * time.Second
}

// setupIntegrationWorkspace creates a temporary bc workspace and changes into it.
// Returns the workspace root path and a cleanup function that restores
// the original working directory and removes the temp directory.
func setupIntegrationWorkspace(t *testing.T) (string, func()) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()

	// Initialize a workspace using the workspace package
	ws, err := workspace.Init(tmpDir)
	if err != nil {
		t.Fatalf("failed to init workspace: %v", err)
	}
	if err := ws.EnsureDirs(); err != nil {
		t.Fatalf("failed to ensure dirs: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir to temp workspace: %v", err)
	}

	return tmpDir, func() {
		_ = os.Chdir(origDir)
	}
}

// executeIntegrationCmd runs rootCmd with the given args, capturing real stdout output.
// Commands use fmt.Printf/Println (writing to os.Stdout), so we redirect
// os.Stdout to a pipe to capture output. Returns captured stdout and any error.
func executeIntegrationCmd(args ...string) (string, string, error) {
	// Save and redirect os.Stdout
	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		return "", "", pipeErr
	}
	os.Stdout = w

	stderr := new(bytes.Buffer)
	rootCmd.SetOut(w)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	// Reset persistent flags to prevent leaking between tests
	_ = rootCmd.PersistentFlags().Set("json", "false")
	_ = rootCmd.PersistentFlags().Set("verbose", "false")
	defer func() { _ = rootCmd.PersistentFlags().Set("json", "false") }()
	defer func() { _ = rootCmd.PersistentFlags().Set("verbose", "false") }()

	// Reset subcommand flags (e.g. logs --tail) to prevent Changed state leaking
	for _, sub := range rootCmd.Commands() {
		sub.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
	}

	err := rootCmd.Execute()

	// Close writer and read all captured output
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = origStdout

	return buf.String(), stderr.String(), err
}

// seedAgents writes an agents.json file in the workspace's agents directory.
// The Manager stores agents as map[string]*Agent.
func seedAgents(t *testing.T, wsDir string, agents map[string]*agent.Agent) {
	t.Helper()
	agentsDir := filepath.Join(wsDir, ".bc", "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}
	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0600); err != nil {
		t.Fatalf("failed to write agents.json: %v", err)
	}
}

// seedQueue creates a queue.json file in the workspace with the given items.
func seedQueue(t *testing.T, wsDir string, items []queue.WorkItem) {
	t.Helper()
	queuePath := filepath.Join(wsDir, ".bc", "queue.json")
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal queue: %v", err)
	}
	if err := os.WriteFile(queuePath, data, 0600); err != nil {
		t.Fatalf("failed to write queue.json: %v", err)
	}
}

// --- Queue command tests ---

func TestQueueListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("queue")
	if err != nil {
		t.Fatalf("queue list returned error: %v", err)
	}

	if !strings.Contains(stdout, "No work items in queue") {
		t.Errorf("expected empty queue message, got: %s", stdout)
	}
}

func TestQueueListNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir() // No .bc directory
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("queue")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestQueueAddAndList(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Add a work item
	stdout, _, err := executeIntegrationCmd("queue", "add", "Fix the login bug")
	if err != nil {
		t.Fatalf("queue add returned error: %v", err)
	}
	if !strings.Contains(stdout, "Added work-001") {
		t.Errorf("expected 'Added work-001', got: %s", stdout)
	}
	if !strings.Contains(stdout, "Fix the login bug") {
		t.Errorf("expected title in output, got: %s", stdout)
	}

	// Verify queue.json was created
	queuePath := filepath.Join(wsDir, ".bc", "queue.json")
	if _, statErr := os.Stat(queuePath); os.IsNotExist(statErr) {
		t.Fatal("queue.json was not created")
	}

	// Add a second item
	stdout, _, err = executeIntegrationCmd("queue", "add", "Add user auth")
	if err != nil {
		t.Fatalf("second queue add returned error: %v", err)
	}
	if !strings.Contains(stdout, "Added work-002") {
		t.Errorf("expected 'Added work-002', got: %s", stdout)
	}
}

func TestQueueAddEmptyTitle(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("queue", "add", "   ")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	if !strings.Contains(err.Error(), "title cannot be empty") {
		t.Errorf("expected empty title error, got: %v", err)
	}
}

func TestQueueListWithItems(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "First task", Status: queue.StatusPending},
		{ID: "work-002", Title: "Second task", Status: queue.StatusDone},
	})

	stdout, _, err := executeIntegrationCmd("queue")
	if err != nil {
		t.Fatalf("queue list returned error: %v", err)
	}

	// Check header
	if !strings.Contains(stdout, "ID") || !strings.Contains(stdout, "STATUS") {
		t.Errorf("expected table header, got: %s", stdout)
	}

	// Check stats line
	if !strings.Contains(stdout, "Total: 2") {
		t.Errorf("expected 'Total: 2' in stats, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Pending: 1") {
		t.Errorf("expected 'Pending: 1' in stats, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Done: 1") {
		t.Errorf("expected 'Done: 1' in stats, got: %s", stdout)
	}
}

func TestQueueAssignNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("queue", "assign", "work-999", "worker-01")
	if err == nil {
		t.Fatal("expected error for missing item, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestQueueAssignSuccess(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Test task", Status: queue.StatusPending},
	})

	stdout, _, err := executeIntegrationCmd("queue", "assign", "work-001", "worker-01")
	if err != nil {
		t.Fatalf("queue assign returned error: %v", err)
	}
	if !strings.Contains(stdout, "Assigned work-001 to worker-01") {
		t.Errorf("expected assignment confirmation, got: %s", stdout)
	}

	// Verify the queue was updated on disk
	q := queue.New(filepath.Join(wsDir, ".bc", "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatalf("failed to reload queue: %v", err)
	}
	item := q.Get("work-001")
	if item == nil {
		t.Fatal("work-001 not found after assign")
	}
	if item.Status != queue.StatusAssigned {
		t.Errorf("expected status assigned, got: %s", item.Status)
	}
	if item.AssignedTo != "worker-01" {
		t.Errorf("expected assigned to worker-01, got: %s", item.AssignedTo)
	}
}

func TestQueueCompleteNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("queue", "complete", "work-999")
	if err == nil {
		t.Fatal("expected error for missing item, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestQueueCompleteSuccess(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Test task", Status: queue.StatusWorking, AssignedTo: "worker-01"},
	})

	stdout, _, err := executeIntegrationCmd("queue", "complete", "work-001")
	if err != nil {
		t.Fatalf("queue complete returned error: %v", err)
	}
	if !strings.Contains(stdout, "Marked work-001 done") {
		t.Errorf("expected completion message, got: %s", stdout)
	}

	// Verify on disk
	q := queue.New(filepath.Join(wsDir, ".bc", "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatalf("failed to reload queue: %v", err)
	}
	item := q.Get("work-001")
	if item == nil {
		t.Fatal("work-001 not found after complete")
	}
	if item.Status != queue.StatusDone {
		t.Errorf("expected status done, got: %s", item.Status)
	}
}

func TestQueueLoadNoBeads(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("queue", "load")
	if err != nil {
		t.Fatalf("queue load returned error: %v", err)
	}
	if !strings.Contains(stdout, "No beads issues found") {
		t.Errorf("expected no beads message, got: %s", stdout)
	}
}

// --- Send command tests ---

func TestSendNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("send", "worker-01", "hello")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestSendAgentNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("send", "nonexistent-agent", "hello")
	if err == nil {
		t.Fatal("expected error for missing agent, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected agent not found error, got: %v", err)
	}
}

func TestSendRequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("send")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
}

// --- Report command tests ---

func TestReportNoAgentID(t *testing.T) {
	// Ensure BC_AGENT_ID is not set
	orig := os.Getenv("BC_AGENT_ID")
	if err := os.Unsetenv("BC_AGENT_ID"); err != nil {
		t.Fatalf("failed to unsetenv: %v", err)
	}
	defer func() {
		if orig != "" {
			_ = os.Setenv("BC_AGENT_ID", orig)
		}
	}()

	_, _, err := executeIntegrationCmd("report", "working", "testing")
	if err == nil {
		t.Fatal("expected error when BC_AGENT_ID not set, got nil")
	}
	if !strings.Contains(err.Error(), "BC_AGENT_ID not set") {
		t.Errorf("expected BC_AGENT_ID error, got: %v", err)
	}
}

func TestReportInvalidState(t *testing.T) {
	t.Setenv("BC_AGENT_ID", "test-agent")

	_, _, err := executeIntegrationCmd("report", "invalid-state")
	if err == nil {
		t.Fatal("expected error for invalid state, got nil")
	}
	if !strings.Contains(err.Error(), "invalid state") {
		t.Errorf("expected invalid state error, got: %v", err)
	}
}

func TestReportNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	t.Setenv("BC_AGENT_ID", "test-agent")

	_, _, err = executeIntegrationCmd("report", "working", "testing")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestReportValidStates(t *testing.T) {
	validStates := []string{"idle", "working", "done", "stuck", "error"}

	for _, state := range validStates {
		t.Run(state, func(t *testing.T) {
			t.Setenv("BC_AGENT_ID", "test-agent")

			// State validation happens before workspace lookup, but
			// invalid states are rejected. Valid states proceed to
			// workspace lookup, which we test fails correctly outside a workspace.
			origDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get cwd: %v", err)
			}
			tmpDir := t.TempDir()
			if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
				t.Fatalf("failed to chdir: %v", chdirErr)
			}
			defer func() { _ = os.Chdir(origDir) }()

			_, _, err = executeIntegrationCmd("report", state, "test message")
			// Should fail with workspace error, NOT invalid state error
			if err == nil {
				t.Fatal("expected workspace error, got nil")
			}
			if strings.Contains(err.Error(), "invalid state") {
				t.Errorf("state %q should be valid but got invalid state error", state)
			}
		})
	}
}

func TestReportRequiresArgs(t *testing.T) {
	_, _, err := executeIntegrationCmd("report")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
}

// --- Status command tests ---

func TestStatusNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("status")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestStatusEmptyWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create agents dir so LoadState doesn't warn
	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(stdout, "No agents configured") {
		t.Errorf("expected 'No agents configured', got: %s", stdout)
	}
	if !strings.Contains(stdout, "bc workspace:") {
		t.Errorf("expected workspace path in output, got: %s", stdout)
	}
}

// --- Helper function tests ---

func TestFormatDurationIntegration(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		seconds  int
	}{
		{"zero", "0s", 0},
		{"seconds", "45s", 45},
		{"minutes", "2m 5s", 125},
		{"hours", "1h 2m", 3725},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := formatDuration(durationFromSeconds(tt.seconds))
			if d != tt.expected {
				t.Errorf("formatDuration(%ds) = %q, want %q", tt.seconds, d, tt.expected)
			}
		})
	}
}

func TestColorQueueStatusIntegration(t *testing.T) {
	statuses := []queue.ItemStatus{
		queue.StatusPending,
		queue.StatusAssigned,
		queue.StatusWorking,
		queue.StatusDone,
		queue.StatusFailed,
	}

	for _, s := range statuses {
		t.Run(string(s), func(t *testing.T) {
			result := colorQueueStatus(s)
			// Should contain the status text
			if !strings.Contains(result, string(s)) {
				t.Errorf("colorQueueStatus(%s) = %q, should contain status text", s, result)
			}
		})
	}
}

func TestColorStateIntegration(t *testing.T) {
	tests := []struct {
		state    string
		contains string
	}{
		{"idle", "idle"},
		{"working", "working"},
		{"done", "done"},
		{"stuck", "stuck"},
		{"error", "error"},
		{"stopped", "stopped"},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			result := colorState(agent.State(tt.state))
			if !strings.Contains(result, tt.contains) {
				t.Errorf("colorState(%s) = %q, should contain %q", tt.state, result, tt.contains)
			}
		})
	}
}

func TestColorState_Default(t *testing.T) {
	result := colorState(agent.State("unknown"))
	if !strings.Contains(result, "unknown") {
		t.Errorf("colorState(unknown) = %q, should contain 'unknown'", result)
	}
	// Default should NOT contain ANSI escape codes
	if strings.Contains(result, "\033[") {
		t.Errorf("colorState(unknown) should not have color codes, got: %q", result)
	}
}

func TestColorQueueStatus_Default(t *testing.T) {
	result := colorQueueStatus(queue.ItemStatus("custom"))
	if !strings.Contains(result, "custom") {
		t.Errorf("colorQueueStatus(custom) = %q, should contain 'custom'", result)
	}
	// Default should NOT contain ANSI escape codes
	if strings.Contains(result, "\033[") {
		t.Errorf("colorQueueStatus(custom) should not have color codes, got: %q", result)
	}
}

// --- Queue detail tests ---

func TestQueueDetailByPositionalArg(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	now := time.Now()
	seedQueue(t, wsDir, []queue.WorkItem{
		{
			ID:          "work-001",
			Title:       "Fix auth bug",
			Status:      queue.StatusWorking,
			AssignedTo:  "engineer-01",
			Description: "Authentication is broken for OAuth users",
			CreatedAt:   now.Add(-1 * time.Hour),
			UpdatedAt:   now,
		},
	})

	stdout, _, err := executeIntegrationCmd("queue", "work-001")
	if err != nil {
		t.Fatalf("queue detail returned error: %v", err)
	}
	if !strings.Contains(stdout, "ID:") {
		t.Errorf("expected 'ID:' field, got: %s", stdout)
	}
	if !strings.Contains(stdout, "work-001") {
		t.Errorf("expected work-001 in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Fix auth bug") {
		t.Errorf("expected title in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("expected assigned agent in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Authentication is broken") {
		t.Errorf("expected description in output, got: %s", stdout)
	}
}

func TestQueueDetailNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("queue", "work-999")
	if err == nil {
		t.Fatal("expected error for missing item, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

// --- Logs command tests ---

// seedEvents writes events to the workspace events.jsonl file.
func seedEvents(t *testing.T, wsDir string, evts []events.Event) {
	t.Helper()
	evtLog := events.NewLog(filepath.Join(wsDir, ".bc", "events.jsonl"))
	for _, ev := range evts {
		if err := evtLog.Append(ev); err != nil {
			t.Fatalf("failed to append event: %v", err)
		}
	}
}

func TestLogsNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("logs")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestLogsEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("logs")
	if err != nil {
		t.Fatalf("logs returned error: %v", err)
	}
	if !strings.Contains(stdout, "No events found") {
		t.Errorf("expected 'No events found', got: %s", stdout)
	}
}

func TestLogsWithEvents(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedEvents(t, wsDir, []events.Event{
		{
			Timestamp: time.Now().Add(-5 * time.Minute),
			Type:      events.AgentSpawned,
			Agent:     "worker-01",
			Message:   "spawned worker",
		},
		{
			Timestamp: time.Now().Add(-3 * time.Minute),
			Type:      events.WorkStarted,
			Agent:     "worker-01",
			Message:   "started task",
		},
		{
			Timestamp: time.Now().Add(-1 * time.Minute),
			Type:      events.WorkCompleted,
			Agent:     "worker-01",
			Message:   "completed task",
		},
	})

	stdout, _, err := executeIntegrationCmd("logs")
	if err != nil {
		t.Fatalf("logs returned error: %v", err)
	}
	if !strings.Contains(stdout, "agent.spawned") {
		t.Errorf("expected event type in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "[worker-01]") {
		t.Errorf("expected agent name in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "spawned worker") {
		t.Errorf("expected message in output, got: %s", stdout)
	}
}

func TestLogsTail(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Seed 5 events
	for i := 0; i < 5; i++ {
		seedEvents(t, wsDir, []events.Event{
			{
				Timestamp: time.Now().Add(time.Duration(-5+i) * time.Minute),
				Type:      events.AgentReport,
				Agent:     "worker-01",
				Message:   "event-" + string(rune('A'+i)),
			},
		})
	}

	// Reset the logsTail flag before test
	logsTail = 2
	defer func() { logsTail = 0 }()

	stdout, _, err := executeIntegrationCmd("logs", "--tail", "2")
	if err != nil {
		t.Fatalf("logs --tail returned error: %v", err)
	}

	// Should show last 2 events
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	// Filter to only lines with actual event content
	eventLines := 0
	for _, l := range lines {
		if strings.Contains(l, "agent.report") {
			eventLines++
		}
	}
	if eventLines != 2 {
		t.Errorf("expected 2 event lines with --tail 2, got %d\noutput: %s", eventLines, stdout)
	}
}

func TestLogsAgentFilter(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentSpawned, Agent: "worker-01", Message: "w1 spawned"},
		{Timestamp: time.Now(), Type: events.AgentSpawned, Agent: "worker-02", Message: "w2 spawned"},
		{Timestamp: time.Now(), Type: events.WorkStarted, Agent: "worker-01", Message: "w1 working"},
	})

	// Reset the logsAgent flag before test
	logsAgent = "worker-01"
	defer func() { logsAgent = "" }()

	stdout, _, err := executeIntegrationCmd("logs", "--agent", "worker-01")
	if err != nil {
		t.Fatalf("logs --agent returned error: %v", err)
	}
	if !strings.Contains(stdout, "w1 spawned") {
		t.Errorf("expected worker-01 events, got: %s", stdout)
	}
	if strings.Contains(stdout, "w2 spawned") {
		t.Errorf("should not contain worker-02 events, got: %s", stdout)
	}
}

// --- Stats command tests ---

func TestStatsNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("stats")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestStatsEmptyWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create agents dir
	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Reset flags
	statsJSON = false
	statsSave = false

	stdout, _, err := executeIntegrationCmd("stats")
	if err != nil {
		t.Fatalf("stats returned error: %v", err)
	}
	if !strings.Contains(stdout, "Workspace Stats") {
		t.Errorf("expected 'Workspace Stats' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Total:") {
		t.Errorf("expected 'Total:' in output, got: %s", stdout)
	}
}

func TestStatsWithQueue(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Task one", Status: queue.StatusPending},
		{ID: "work-002", Title: "Task two", Status: queue.StatusDone},
		{ID: "work-003", Title: "Task three", Status: queue.StatusWorking, AssignedTo: "worker-01"},
	})

	statsJSON = false
	statsSave = false

	stdout, _, err := executeIntegrationCmd("stats")
	if err != nil {
		t.Fatalf("stats returned error: %v", err)
	}
	if !strings.Contains(stdout, "Pending:  1") {
		t.Errorf("expected 'Pending:  1' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Done:     1") {
		t.Errorf("expected 'Done:     1' in output, got: %s", stdout)
	}
}

func TestStatsSave(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	statsSave = true
	statsJSON = false
	defer func() { statsSave = false }()

	stdout, _, err := executeIntegrationCmd("stats", "--save")
	if err != nil {
		t.Fatalf("stats --save returned error: %v", err)
	}
	if !strings.Contains(stdout, "Stats saved") {
		t.Errorf("expected 'Stats saved' message, got: %s", stdout)
	}

	// Verify file was created
	statsPath := filepath.Join(wsDir, ".bc", "stats.json")
	if _, err := os.Stat(statsPath); os.IsNotExist(err) {
		t.Error("stats.json was not created")
	}
}

// --- Dashboard command tests ---

func TestDashboardNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("dashboard")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestDashboardEmptyWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("dashboard")
	if err != nil {
		t.Fatalf("dashboard returned error: %v", err)
	}
	if !strings.Contains(stdout, "dashboard:") {
		t.Errorf("expected 'dashboard:' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "No agents configured") {
		t.Errorf("expected 'No agents configured', got: %s", stdout)
	}
	if !strings.Contains(stdout, "No work items") {
		t.Errorf("expected 'No work items', got: %s", stdout)
	}
	if !strings.Contains(stdout, "No events yet") {
		t.Errorf("expected 'No events yet', got: %s", stdout)
	}
}

// --- Channel command tests ---

// seedChannels creates a channels.json file in the workspace.
func seedChannels(t *testing.T, wsDir string, channels []*channel.Channel) {
	t.Helper()
	channelPath := filepath.Join(wsDir, ".bc", "channels.json")
	data, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal channels: %v", err)
	}
	if err := os.WriteFile(channelPath, data, 0600); err != nil {
		t.Fatalf("failed to write channels.json: %v", err)
	}
}

func TestChannelListNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("channel")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestChannelListEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("channel")
	if err != nil {
		t.Fatalf("channel list returned error: %v", err)
	}
	if !strings.Contains(stdout, "No channels defined") {
		t.Errorf("expected 'No channels defined', got: %s", stdout)
	}
}

func TestChannelListWithChannels(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "standup", Members: []string{"coordinator", "worker-01"}},
		{Name: "all", Members: []string{"coordinator", "worker-01", "qa-01"}},
	})

	stdout, _, err := executeIntegrationCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list returned error: %v", err)
	}
	if !strings.Contains(stdout, "standup") {
		t.Errorf("expected 'standup' channel, got: %s", stdout)
	}
	if !strings.Contains(stdout, "all") {
		t.Errorf("expected 'all' channel, got: %s", stdout)
	}
	if !strings.Contains(stdout, "CHANNEL") {
		t.Errorf("expected table header, got: %s", stdout)
	}
}

func TestChannelCreateAndDelete(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	// Create
	stdout, _, err := executeIntegrationCmd("channel", "create", "test-channel")
	if err != nil {
		t.Fatalf("channel create returned error: %v", err)
	}
	if !strings.Contains(stdout, "Created channel") {
		t.Errorf("expected creation confirmation, got: %s", stdout)
	}

	// Delete
	stdout, _, err = executeIntegrationCmd("channel", "delete", "test-channel")
	if err != nil {
		t.Fatalf("channel delete returned error: %v", err)
	}
	if !strings.Contains(stdout, "Deleted channel") {
		t.Errorf("expected deletion confirmation, got: %s", stdout)
	}
}

func TestChannelCreateDuplicate(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "existing", Members: []string{}},
	})

	_, _, err := executeIntegrationCmd("channel", "create", "existing")
	if err == nil {
		t.Fatal("expected error for duplicate channel, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestChannelAddMember(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "team", Members: []string{}},
	})

	stdout, _, err := executeIntegrationCmd("channel", "add", "team", "worker-01", "worker-02")
	if err != nil {
		t.Fatalf("channel add returned error: %v", err)
	}
	if !strings.Contains(stdout, "Added 2 member(s)") {
		t.Errorf("expected 'Added 2 member(s)', got: %s", stdout)
	}

	// Verify on disk
	store := channel.NewStore(wsDir)
	if loadErr := store.Load(); loadErr != nil {
		t.Fatalf("failed to load channels: %v", loadErr)
	}
	members, err := store.GetMembers("team")
	if err != nil {
		t.Fatalf("failed to get members: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestChannelRemoveMember(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "team", Members: []string{"worker-01", "worker-02"}},
	})

	stdout, _, err := executeIntegrationCmd("channel", "remove", "team", "worker-01")
	if err != nil {
		t.Fatalf("channel remove returned error: %v", err)
	}
	if !strings.Contains(stdout, "Removed") {
		t.Errorf("expected 'Removed' message, got: %s", stdout)
	}
}

func TestChannelDeleteNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "delete", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing channel, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

func TestChannelJoinNoAgentID(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	orig := os.Getenv("BC_AGENT_ID")
	if err := os.Unsetenv("BC_AGENT_ID"); err != nil {
		t.Fatalf("failed to unsetenv: %v", err)
	}
	defer func() {
		if orig != "" {
			_ = os.Setenv("BC_AGENT_ID", orig)
		}
	}()

	_, _, err := executeIntegrationCmd("channel", "join", "standup")
	if err == nil {
		t.Fatal("expected error when BC_AGENT_ID not set, got nil")
	}
	if !strings.Contains(err.Error(), "BC_AGENT_ID not set") {
		t.Errorf("expected BC_AGENT_ID error, got: %v", err)
	}
}

func TestChannelJoinSuccess(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "standup", Members: []string{}},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	stdout, _, err := executeIntegrationCmd("channel", "join", "standup")
	if err != nil {
		t.Fatalf("channel join returned error: %v", err)
	}
	if !strings.Contains(stdout, "Joined channel") {
		t.Errorf("expected 'Joined channel' message, got: %s", stdout)
	}
}

func TestChannelLeaveSuccess(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "standup", Members: []string{"test-agent"}},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	stdout, _, err := executeIntegrationCmd("channel", "leave", "standup")
	if err != nil {
		t.Fatalf("channel leave returned error: %v", err)
	}
	if !strings.Contains(stdout, "Left channel") {
		t.Errorf("expected 'Left channel' message, got: %s", stdout)
	}
}

func TestChannelLeaveNoAgentID(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	orig := os.Getenv("BC_AGENT_ID")
	if err := os.Unsetenv("BC_AGENT_ID"); err != nil {
		t.Fatalf("failed to unsetenv: %v", err)
	}
	defer func() {
		if orig != "" {
			_ = os.Setenv("BC_AGENT_ID", orig)
		}
	}()

	_, _, err := executeIntegrationCmd("channel", "leave", "standup")
	if err == nil {
		t.Fatal("expected error when BC_AGENT_ID not set, got nil")
	}
	if !strings.Contains(err.Error(), "BC_AGENT_ID not set") {
		t.Errorf("expected BC_AGENT_ID error, got: %v", err)
	}
}

func TestChannelHistoryEmpty(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{Name: "standup", Members: []string{}},
	})

	stdout, _, err := executeIntegrationCmd("channel", "history", "standup")
	if err != nil {
		t.Fatalf("channel history returned error: %v", err)
	}
	if !strings.Contains(stdout, "No message history") {
		t.Errorf("expected 'No message history', got: %s", stdout)
	}
}

func TestChannelHistoryWithMessages(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedChannels(t, wsDir, []*channel.Channel{
		{
			Name:    "standup",
			Members: []string{"worker-01"},
			History: []channel.HistoryEntry{
				{Time: time.Now().Add(-10 * time.Minute), Message: "first update"},
				{Time: time.Now().Add(-5 * time.Minute), Message: "second update"},
			},
		},
	})

	stdout, _, err := executeIntegrationCmd("channel", "history", "standup")
	if err != nil {
		t.Fatalf("channel history returned error: %v", err)
	}
	if !strings.Contains(stdout, "first update") {
		t.Errorf("expected 'first update' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "second update") {
		t.Errorf("expected 'second update' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "#standup") {
		t.Errorf("expected channel name in header, got: %s", stdout)
	}
}

func TestChannelHistoryNotFound(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("channel", "history", "nonexistent")
	if err == nil {
		t.Fatal("expected error for missing channel, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got: %v", err)
	}
}

// --- Report command tests (with workspace) ---

func TestReportWorkingInWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"test-agent": {
			Name:      "test-agent",
			Role:      agent.RoleWorker,
			State:     agent.StateIdle,
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	stdout, _, err := executeIntegrationCmd("report", "working", "fixing auth bug")
	if err != nil {
		t.Fatalf("report returned error: %v", err)
	}
	if !strings.Contains(stdout, "Reported: working fixing auth bug") {
		t.Errorf("expected report confirmation, got: %s", stdout)
	}
}

func TestReportDoneInWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"test-agent": {
			Name:      "test-agent",
			Role:      agent.RoleWorker,
			State:     agent.StateWorking,
			Task:      "some task",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	stdout, _, err := executeIntegrationCmd("report", "done", "auth bug fixed")
	if err != nil {
		t.Fatalf("report returned error: %v", err)
	}
	if !strings.Contains(stdout, "Reported: done auth bug fixed") {
		t.Errorf("expected report confirmation, got: %s", stdout)
	}

	// Verify event was logged
	evtLog := events.NewLog(filepath.Join(wsDir, ".bc", "events.jsonl"))
	evts, err := evtLog.Read()
	if err != nil {
		t.Fatalf("failed to read events: %v", err)
	}
	if len(evts) == 0 {
		t.Error("expected at least one event logged")
	}
	found := false
	for _, ev := range evts {
		if ev.Type == events.AgentReport && ev.Agent == "test-agent" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected agent.report event for test-agent")
	}
}

func TestReportStuckInWorkspace(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"test-agent": {
			Name:      "test-agent",
			Role:      agent.RoleWorker,
			State:     agent.StateWorking,
			Task:      "some task",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	stdout, _, err := executeIntegrationCmd("report", "stuck", "need database credentials")
	if err != nil {
		t.Fatalf("report returned error: %v", err)
	}
	if !strings.Contains(stdout, "Reported: stuck") {
		t.Errorf("expected report confirmation, got: %s", stdout)
	}
}

// --- Report + Queue integration test ---

func TestReportWorkingTransitionsQueueItem(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"test-agent": {
			Name:      "test-agent",
			Role:      agent.RoleWorker,
			State:     agent.StateIdle,
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	// Seed queue with assigned item
	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Test task", Status: queue.StatusAssigned, AssignedTo: "test-agent"},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	_, _, err := executeIntegrationCmd("report", "working", "starting task")
	if err != nil {
		t.Fatalf("report returned error: %v", err)
	}

	// Verify queue item transitioned to working
	q := queue.New(filepath.Join(wsDir, ".bc", "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatalf("failed to reload queue: %v", err)
	}
	item := q.Get("work-001")
	if item == nil {
		t.Fatal("work-001 not found after report")
	}
	if item.Status != queue.StatusWorking {
		t.Errorf("expected status working after report, got: %s", item.Status)
	}
}

func TestReportDoneTransitionsQueueItem(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"test-agent": {
			Name:      "test-agent",
			Role:      agent.RoleWorker,
			State:     agent.StateWorking,
			Task:      "some task",
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	// Seed queue with working item
	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Test task", Status: queue.StatusWorking, AssignedTo: "test-agent"},
	})

	t.Setenv("BC_AGENT_ID", "test-agent")

	_, _, err := executeIntegrationCmd("report", "done", "task completed")
	if err != nil {
		t.Fatalf("report returned error: %v", err)
	}

	// Verify queue item transitioned to done
	q := queue.New(filepath.Join(wsDir, ".bc", "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatalf("failed to reload queue: %v", err)
	}
	item := q.Get("work-001")
	if item == nil {
		t.Fatal("work-001 not found after report")
	}
	if item.Status != queue.StatusDone {
		t.Errorf("expected status done after report, got: %s", item.Status)
	}
}

// --- Version command tests ---

func TestVersionOutput(t *testing.T) {
	SetVersionInfo("2.0.0", "def456", "2025-06-01")
	defer SetVersionInfo("dev", "none", "unknown")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2.0.0") {
		t.Errorf("Expected version '2.0.0', got: %s", output)
	}
	if !strings.Contains(output, "def456") {
		t.Errorf("Expected commit 'def456', got: %s", output)
	}
	if !strings.Contains(output, "2025-06-01") {
		t.Errorf("Expected date '2025-06-01', got: %s", output)
	}
}

// --- Status with agents written to disk ---

// --- JSON output tests ---

func TestQueueListJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "First task", Status: queue.StatusPending},
		{ID: "work-002", Title: "Second task", Status: queue.StatusDone},
	})

	stdout, _, err := executeIntegrationCmd("queue", "--json")
	if err != nil {
		t.Fatalf("queue --json returned error: %v", err)
	}

	var items []queue.WorkItem
	if err := json.Unmarshal([]byte(stdout), &items); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestQueueDetailJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{
			ID:          "work-001",
			Title:       "Fix auth bug",
			Status:      queue.StatusPending,
			Description: "Auth is broken",
		},
	})

	stdout, _, err := executeIntegrationCmd("queue", "--json", "work-001")
	if err != nil {
		t.Fatalf("queue --json detail returned error: %v", err)
	}

	var item queue.WorkItem
	if err := json.Unmarshal([]byte(stdout), &item); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if item.ID != "work-001" {
		t.Errorf("expected ID work-001, got %s", item.ID)
	}
}

func TestLogsJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentSpawned, Agent: "w-01", Message: "spawned"},
	})

	stdout, _, err := executeIntegrationCmd("logs", "--json")
	if err != nil {
		t.Fatalf("logs --json returned error: %v", err)
	}

	var evts []events.Event
	if err := json.Unmarshal([]byte(stdout), &evts); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if len(evts) != 1 {
		t.Errorf("expected 1 event, got %d", len(evts))
	}
}

func TestStatsJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	statsJSON = true
	statsSave = false
	defer func() { statsJSON = false }()

	stdout, _, err := executeIntegrationCmd("stats", "--json")
	if err != nil {
		t.Fatalf("stats --json returned error: %v", err)
	}

	// Should be valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if _, ok := result["work_items"]; !ok {
		t.Error("expected 'work_items' key in JSON output")
	}
}

func TestDashboardJSON(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	if err := os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0750); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	stdout, _, err := executeIntegrationCmd("dashboard", "--json")
	if err != nil {
		t.Fatalf("dashboard --json returned error: %v", err)
	}

	var result dashboardOutput
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, stdout)
	}
	if result.Agents.Total != 0 {
		t.Errorf("expected 0 agents, got %d", result.Agents.Total)
	}
}

// --- Init command tests ---

func TestInitNewDirectory(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, "myproject")
	if err = os.MkdirAll(targetDir, 0750); err != nil {
		t.Fatalf("failed to create target dir: %v", err)
	}
	if err = os.Chdir(targetDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	stdout, _, err := executeIntegrationCmd("init")
	if err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if !strings.Contains(stdout, "Initialized bc v2 workspace") {
		t.Errorf("expected initialization message, got: %s", stdout)
	}

	// Verify .bc directory was created
	if _, err := os.Stat(filepath.Join(targetDir, ".bc")); os.IsNotExist(err) {
		t.Error(".bc directory was not created")
	}
}

func TestInitAlreadyInitialized(t *testing.T) {
	// setupIntegrationWorkspace creates a v1 workspace, which triggers v1 detection
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("init")
	if err == nil {
		t.Fatal("expected error for already initialized workspace, got nil")
	}
	// v1 workspace detection returns specific error
	if !strings.Contains(err.Error(), "v1 workspace exists") {
		t.Errorf("expected 'v1 workspace exists' error, got: %v", err)
	}
}

// --- Send command tests (more coverage) ---

func TestSendToStoppedAgent(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"worker-01": {
			Name:      "worker-01",
			Role:      agent.RoleWorker,
			State:     agent.StateStopped,
			StartedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	})

	_, _, err := executeIntegrationCmd("send", "worker-01", "hello")
	if err == nil {
		t.Fatal("expected error for stopped agent, got nil")
	}
	if !strings.Contains(err.Error(), "stopped") {
		t.Errorf("expected 'stopped' error, got: %v", err)
	}
}

// --- createDefaultChannels test ---

func TestCreateDefaultChannelsIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".bc"), 0750); err != nil {
		t.Fatalf("failed to create .bc dir: %v", err)
	}

	engineers := []string{"engineer-01", "engineer-02"}
	qaNames := []string{"qa-01"}
	allAgents := []string{"coordinator", "product-manager", "manager", "engineer-01", "engineer-02", "qa-01"}

	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	techLeads := []string{} // No tech leads in this test
	createDefaultChannels(tmpDir, techLeads, engineers, qaNames, allAgents)

	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	os.Stdout = origStdout

	output := buf.String()
	if !strings.Contains(output, "Created") {
		t.Errorf("expected creation message, got: %s", output)
	}

	// Verify channels were created
	store := channel.NewStore(tmpDir)
	if err := store.Load(); err != nil {
		t.Fatalf("failed to load channels: %v", err)
	}

	// Check required channels exist
	for _, name := range []string{"standup", "leadership", "engineering", "qa", "all"} {
		ch, exists := store.Get(name)
		if !exists {
			t.Errorf("expected channel %q to exist", name)
			continue
		}
		if len(ch.Members) == 0 {
			t.Errorf("channel %q should have members", name)
		}
	}
}

func TestStatusWithAgents(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"coordinator": {
			Name:      "coordinator",
			Role:      agent.RoleCoordinator,
			State:     agent.StateStopped,
			Session:   "bc-coord",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
		"worker-01": {
			Name:      "worker-01",
			Role:      agent.RoleWorker,
			State:     agent.StateStopped,
			Session:   "bc-worker-01",
			Task:      "fixing auth",
			StartedAt: time.Now().Add(-30 * time.Minute),
		},
	})

	stdout, _, err := executeIntegrationCmd("status")
	if err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if !strings.Contains(stdout, "coordinator") {
		t.Errorf("expected 'coordinator' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "worker-01") {
		t.Errorf("expected 'worker-01' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "AGENT") {
		t.Errorf("expected table header, got: %s", stdout)
	}
}
