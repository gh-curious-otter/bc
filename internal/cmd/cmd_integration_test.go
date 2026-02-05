package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/workspace"
)

func durationFromSeconds(s int) time.Duration {
	return time.Duration(s) * time.Second
}

// setupTestWorkspace creates a temporary bc workspace and changes into it.
// Returns the workspace root path and a cleanup function that restores
// the original working directory and removes the temp directory.
func setupTestWorkspace(t *testing.T) (string, func()) {
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
		os.Chdir(origDir)
	}
}

// executeCmd runs rootCmd with the given args, capturing real stdout output.
// Commands use fmt.Printf/Println (writing to os.Stdout), so we redirect
// os.Stdout to a pipe to capture output. Returns captured stdout and any error.
func executeCmd(args ...string) (string, string, error) {
	// Save and redirect os.Stdout
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	stderr := new(bytes.Buffer)
	rootCmd.SetOut(w)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()

	// Close writer and read all captured output
	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	os.Stdout = origStdout

	return buf.String(), stderr.String(), err
}

// seedQueue creates a queue.json file in the workspace with the given items.
func seedQueue(t *testing.T, wsDir string, items []queue.WorkItem) {
	t.Helper()
	queuePath := filepath.Join(wsDir, ".bc", "queue.json")
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal queue: %v", err)
	}
	if err := os.WriteFile(queuePath, data, 0644); err != nil {
		t.Fatalf("failed to write queue.json: %v", err)
	}
}

// --- Queue command tests ---

func TestQueueListEmpty(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	stdout, _, err := executeCmd("queue")
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
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, _, err = executeCmd("queue")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestQueueAddAndList(t *testing.T) {
	wsDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Add a work item
	stdout, _, err := executeCmd("queue", "add", "Fix the login bug")
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
	if _, err := os.Stat(queuePath); os.IsNotExist(err) {
		t.Fatal("queue.json was not created")
	}

	// Add a second item
	stdout, _, err = executeCmd("queue", "add", "Add user auth")
	if err != nil {
		t.Fatalf("second queue add returned error: %v", err)
	}
	if !strings.Contains(stdout, "Added work-002") {
		t.Errorf("expected 'Added work-002', got: %s", stdout)
	}
}

func TestQueueAddEmptyTitle(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	_, _, err := executeCmd("queue", "add", "   ")
	if err == nil {
		t.Fatal("expected error for empty title, got nil")
	}
	if !strings.Contains(err.Error(), "title cannot be empty") {
		t.Errorf("expected empty title error, got: %v", err)
	}
}

func TestQueueListWithItems(t *testing.T) {
	wsDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "First task", Status: queue.StatusPending},
		{ID: "work-002", Title: "Second task", Status: queue.StatusDone},
	})

	stdout, _, err := executeCmd("queue")
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
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	_, _, err := executeCmd("queue", "assign", "work-999", "worker-01")
	if err == nil {
		t.Fatal("expected error for missing item, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestQueueAssignSuccess(t *testing.T) {
	wsDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Test task", Status: queue.StatusPending},
	})

	stdout, _, err := executeCmd("queue", "assign", "work-001", "worker-01")
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
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	_, _, err := executeCmd("queue", "complete", "work-999")
	if err == nil {
		t.Fatal("expected error for missing item, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected not found error, got: %v", err)
	}
}

func TestQueueCompleteSuccess(t *testing.T) {
	wsDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	seedQueue(t, wsDir, []queue.WorkItem{
		{ID: "work-001", Title: "Test task", Status: queue.StatusWorking, AssignedTo: "worker-01"},
	})

	stdout, _, err := executeCmd("queue", "complete", "work-001")
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
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	stdout, _, err := executeCmd("queue", "load")
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
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, _, err = executeCmd("send", "worker-01", "hello")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestSendAgentNotFound(t *testing.T) {
	_, cleanup := setupTestWorkspace(t)
	defer cleanup()

	_, _, err := executeCmd("send", "nonexistent-agent", "hello")
	if err == nil {
		t.Fatal("expected error for missing agent, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected agent not found error, got: %v", err)
	}
}

func TestSendRequiresArgs(t *testing.T) {
	_, _, err := executeCmd("send")
	if err == nil {
		t.Fatal("expected error for missing args, got nil")
	}
}

// --- Report command tests ---

func TestReportNoAgentID(t *testing.T) {
	// Ensure BC_AGENT_ID is not set
	orig := os.Getenv("BC_AGENT_ID")
	os.Unsetenv("BC_AGENT_ID")
	defer func() {
		if orig != "" {
			os.Setenv("BC_AGENT_ID", orig)
		}
	}()

	_, _, err := executeCmd("report", "working", "testing")
	if err == nil {
		t.Fatal("expected error when BC_AGENT_ID not set, got nil")
	}
	if !strings.Contains(err.Error(), "BC_AGENT_ID not set") {
		t.Errorf("expected BC_AGENT_ID error, got: %v", err)
	}
}

func TestReportInvalidState(t *testing.T) {
	os.Setenv("BC_AGENT_ID", "test-agent")
	defer os.Unsetenv("BC_AGENT_ID")

	_, _, err := executeCmd("report", "invalid-state")
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
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	os.Setenv("BC_AGENT_ID", "test-agent")
	defer os.Unsetenv("BC_AGENT_ID")

	_, _, err = executeCmd("report", "working", "testing")
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
			os.Setenv("BC_AGENT_ID", "test-agent")
			defer os.Unsetenv("BC_AGENT_ID")

			// State validation happens before workspace lookup, but
			// invalid states are rejected. Valid states proceed to
			// workspace lookup, which we test fails correctly outside a workspace.
			origDir, _ := os.Getwd()
			tmpDir := t.TempDir()
			os.Chdir(tmpDir)
			defer os.Chdir(origDir)

			_, _, err := executeCmd("report", state, "test message")
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
	_, _, err := executeCmd("report")
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
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	_, _, err = executeCmd("status")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestStatusEmptyWorkspace(t *testing.T) {
	wsDir, cleanup := setupTestWorkspace(t)
	defer cleanup()

	// Create agents dir so LoadState doesn't warn
	os.MkdirAll(filepath.Join(wsDir, ".bc", "agents"), 0755)

	stdout, _, err := executeCmd("status")
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

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"zero", 0, "0s"},
		{"seconds", 45, "45s"},
		{"minutes", 125, "2m 5s"},
		{"hours", 3725, "1h 2m"},
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

func TestColorQueueStatus(t *testing.T) {
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

func TestColorState(t *testing.T) {
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
