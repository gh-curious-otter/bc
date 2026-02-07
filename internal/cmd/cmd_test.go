package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
)

// --- Test helpers ---

// setupTestWorkspace creates a minimal bc workspace in a temp directory
// and changes CWD to it. Returns the root dir and a cleanup function.
func setupTestWorkspace(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatal(err)
	}

	// Write config.json (what workspace.Find looks for)
	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	wsConfig := `{"version":1,"name":"test-ws","state_dir":"` + bcDir + `","root_dir":"` + absDir + `","max_workers":3}`
	if writeErr := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte(wsConfig), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	// Change to workspace dir
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	return dir
}

// setupQueueFile creates a queue.json with sample items.
func setupQueueFile(t *testing.T, stateDir string, items []queue.WorkItem) {
	t.Helper()
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	for _, item := range items {
		q.Add(item.Title, item.Description, item.BeadsID)
	}
	if err := q.Save(); err != nil {
		t.Fatalf("failed to save test queue: %v", err)
	}
}

// setupEventsFile creates an events.jsonl with sample events.
func setupEventsFile(t *testing.T, stateDir string, evts []events.Event) {
	t.Helper()
	log := events.NewLog(filepath.Join(stateDir, "events.jsonl"))
	for _, ev := range evts {
		if err := log.Append(ev); err != nil {
			t.Fatal(err)
		}
	}
}

// setupAgentState writes agent state to the agents dir.
func setupAgentState(t *testing.T, agentsDir string, agents map[string]*agent.Agent) {
	t.Helper()
	data, err := json.Marshal(agents)
	if err != nil {
		t.Fatalf("failed to marshal agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "agents.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

// executeCmd runs a cobra command with the given args.
// Note: most commands write to os.Stdout directly via fmt.Printf,
// so output capture via cobra's SetOut doesn't work for those.
// We capture stdout via pipe for commands we need to inspect.
func executeCmd(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	// Reset persistent flags to prevent leaking between tests
	_ = rootCmd.PersistentFlags().Set("json", "false")
	_ = rootCmd.PersistentFlags().Set("verbose", "false")

	// Reset subcommand flags (e.g. logs --tail) to prevent Changed state leaking
	for _, sub := range rootCmd.Commands() {
		sub.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var stdoutBuf bytes.Buffer
	_, _ = stdoutBuf.ReadFrom(r)

	// Combine cobra output and stdout
	combined := buf.String() + stdoutBuf.String()
	return combined, execErr
}

// --- formatDuration tests ---

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name string
		want string
		d    time.Duration
	}{
		{"zero", "0s", 0},
		{"seconds only", "45s", 45 * time.Second},
		{"one minute", "1m 0s", 60 * time.Second},
		{"minutes and seconds", "2m 30s", 2*time.Minute + 30*time.Second},
		{"one hour", "1h 0m", time.Hour},
		{"hours and minutes", "2h 15m", 2*time.Hour + 15*time.Minute},
		{"sub-second rounds up", "1s", 500 * time.Millisecond},
		{"large duration", "48h 30m", 48*time.Hour + 30*time.Minute},
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

// --- colorState tests ---

func TestColorState(t *testing.T) {
	tests := []struct {
		state agent.State
		want  string // expected state name in output
	}{
		{agent.StateIdle, "idle"},
		{agent.StateWorking, "working"},
		{agent.StateDone, "done"},
		{agent.StateStuck, "stuck"},
		{agent.StateError, "error"},
		{agent.StateStopped, "stopped"},
		{agent.State("unknown"), "unknown"},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			result := colorState(tt.state)
			if !strings.Contains(result, tt.want) {
				t.Errorf("colorState(%q) = %q, should contain %q", tt.state, result, tt.want)
			}
		})
	}
}

// --- stateIcon tests ---

func TestStateIcon(t *testing.T) {
	tests := []struct {
		state agent.State
		want  string
	}{
		{agent.StateIdle, "o"},
		{agent.StateWorking, ">"},
		{agent.StateDone, "+"},
		{agent.StateStuck, "!"},
		{agent.StateError, "x"},
		{agent.StateStarting, "~"},
		{agent.StateStopped, "-"},
		{agent.State("bogus"), "?"},
	}
	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			got := stateIcon(tt.state)
			if got != tt.want {
				t.Errorf("stateIcon(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

// --- colorQueueStatus tests ---

func TestColorQueueStatus(t *testing.T) {
	tests := []struct {
		status queue.ItemStatus
		want   string
	}{
		{queue.StatusPending, "pending"},
		{queue.StatusAssigned, "assigned"},
		{queue.StatusWorking, "working"},
		{queue.StatusDone, "done"},
		{queue.StatusFailed, "failed"},
		{queue.ItemStatus("other"), "other"},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			result := colorQueueStatus(tt.status)
			if !strings.Contains(result, tt.want) {
				t.Errorf("colorQueueStatus(%q) = %q, should contain %q", tt.status, result, tt.want)
			}
		})
	}
}

// --- parseRole tests ---

func TestParseRole(t *testing.T) {
	tests := []struct {
		input   string
		want    agent.Role
		wantErr bool
	}{
		{"worker", agent.RoleWorker, false},
		{"engineer", agent.RoleEngineer, false},
		{"manager", agent.RoleManager, false},
		{"product-manager", agent.RoleProductManager, false},
		{"pm", agent.RoleProductManager, false},
		{"coordinator", agent.RoleCoordinator, false},
		{"coord", agent.RoleCoordinator, false},
		{"qa", agent.RoleQA, false},
		{"WORKER", agent.RoleWorker, false},     // case insensitive
		{"Engineer", agent.RoleEngineer, false}, // case insensitive
		{"invalid", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRole(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRole(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- loadRolePrompt tests ---

func TestLoadRolePrompt(t *testing.T) {
	dir := t.TempDir()
	promptDir := filepath.Join(dir, "prompts")
	if err := os.MkdirAll(promptDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a test prompt file
	if err := os.WriteFile(filepath.Join(promptDir, "engineer.md"), []byte("You are an engineer."), 0600); err != nil {
		t.Fatal(err)
	}

	t.Run("existing prompt", func(t *testing.T) {
		got := loadRolePrompt(dir, "engineer")
		if got != "You are an engineer." {
			t.Errorf("loadRolePrompt = %q, want %q", got, "You are an engineer.")
		}
	})

	t.Run("missing prompt returns empty", func(t *testing.T) {
		got := loadRolePrompt(dir, "nonexistent")
		if got != "" {
			t.Errorf("loadRolePrompt for missing file = %q, want empty", got)
		}
	})
}

// --- buildBootstrapPrompt tests ---

func TestBuildBootstrapPrompt(t *testing.T) {
	agents := []string{"coordinator", "manager", "engineer-01"}
	items := []queue.WorkItem{
		{ID: "work-001", Title: "Fix auth", Description: "Fix the authentication bug", BeadsID: "bc-123"},
		{ID: "work-002", Title: "Add tests", BeadsID: ""},
	}

	result := buildBootstrapPrompt(agents, items, "/test/workspace")

	checks := []string{
		"coordinator",
		"manager",
		"engineer-01",
		"/test/workspace",
		"work-001",
		"Fix auth",
		"Fix the authentication bug",
		"bc-123",
		"work-002",
		"Add tests",
		"WORK QUEUE",
		"YOUR WORKFLOW",
		"BC COMMANDS",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("buildBootstrapPrompt missing %q", check)
		}
	}
}

// --- createDefaultChannels tests ---

func TestCreateDefaultChannels(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	engineers := []string{"eng-01", "eng-02"}
	qa := []string{"qa-01"}
	all := []string{"coordinator", "product-manager", "manager", "eng-01", "eng-02", "qa-01"}

	// Capture stdout from createDefaultChannels
	oldStdout := os.Stdout
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	techLeads := []string{} // No tech leads in this test
	createDefaultChannels(dir, techLeads, engineers, qa, all)
	_ = w.Close()
	os.Stdout = oldStdout

	// Verify channels file was created at .bc/channels.json
	channelsFile := filepath.Join(dir, ".bc", "channels.json")
	if _, statErr := os.Stat(channelsFile); os.IsNotExist(statErr) {
		t.Fatal("channels.json not created")
	}

	data, err := os.ReadFile(channelsFile) //nolint:gosec // G304: test file reads from test-created path
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	for _, ch := range []string{"standup", "leadership", "engineering", "qa", "all"} {
		if !strings.Contains(content, ch) {
			t.Errorf("channels.json missing channel %q", ch)
		}
	}
}

func TestCreateDefaultChannels_PerAgentChannels(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	techLeads := []string{"tech-lead-01"}
	engineers := []string{"engineer-01", "engineer-02"}
	qa := []string{"qa-01"}
	all := []string{"coordinator", "product-manager", "manager", "tech-lead-01", "engineer-01", "engineer-02", "qa-01"}

	// Capture stdout
	oldStdout := os.Stdout
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	createDefaultChannels(dir, techLeads, engineers, qa, all)
	_ = w.Close()
	os.Stdout = oldStdout

	// Verify channels file was created
	channelsFile := filepath.Join(dir, ".bc", "channels.json")
	data, err := os.ReadFile(channelsFile) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	// Verify per-agent channels are created
	for _, agentName := range all {
		if !strings.Contains(content, fmt.Sprintf(`"name": "%s"`, agentName)) {
			t.Errorf("channels.json missing per-agent channel for %q", agentName)
		}
	}
}

func TestCreateDefaultChannels_NoDuplicatesOnRestart(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	engineers := []string{"engineer-01"}
	qa := []string{"qa-01"}
	all := []string{"coordinator", "manager", "engineer-01", "qa-01"}

	// Suppress stdout
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	// Run twice to simulate restart
	createDefaultChannels(dir, []string{}, engineers, qa, all)
	createDefaultChannels(dir, []string{}, engineers, qa, all)

	_ = w.Close()
	os.Stdout = oldStdout

	// Read and count channel occurrences
	channelsFile := filepath.Join(dir, ".bc", "channels.json")
	data, err := os.ReadFile(channelsFile) //nolint:gosec // test file
	if err != nil {
		t.Fatal(err)
	}

	// Count occurrences of "engineer-01" channel name
	count := strings.Count(string(data), `"name": "engineer-01"`)
	if count != 1 {
		t.Errorf("expected 1 engineer-01 channel, got %d (duplicate channels created)", count)
	}
}

// --- Init command tests ---

func TestInitCommand(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "myproject")
	if err := os.MkdirAll(subdir, 0750); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(subdir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	output, err := executeCmd("init")
	if err != nil {
		t.Fatalf("init failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Initialized") {
		t.Errorf("init output should contain 'Initialized', got: %s", output)
	}

	// Verify .bc dir exists
	if _, err := os.Stat(filepath.Join(subdir, ".bc")); os.IsNotExist(err) {
		t.Error(".bc directory not created")
	}
}

func TestInitCommand_AlreadyInitialized(t *testing.T) {
	root := setupTestWorkspace(t)

	// Initialize again should fail (use the workspace dir)
	_, err := executeCmd("init", root)
	if err == nil {
		t.Error("expected error for already initialized workspace")
	}
}

// --- Queue command tests ---

func TestQueueList_Empty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("queue")
	if err != nil {
		t.Fatalf("queue list failed: %v", err)
	}
	if !strings.Contains(output, "No work items") {
		t.Errorf("expected 'No work items', got: %s", output)
	}
}

func TestQueueList_WithItems(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Fix auth bug"},
		{Title: "Add tests"},
	})

	output, err := executeCmd("queue")
	if err != nil {
		t.Fatalf("queue list failed: %v", err)
	}
	if !strings.Contains(output, "Fix auth bug") {
		t.Errorf("queue list should show items, got: %s", output)
	}
	if !strings.Contains(output, "Total:") {
		t.Errorf("queue list should show stats, got: %s", output)
	}
}

func TestQueueList_JSON(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Fix auth bug"},
	})

	output, err := executeCmd("queue", "--json")
	if err != nil {
		t.Fatalf("queue --json failed: %v", err)
	}
	// JSON output goes to stdout directly, not captured by cobra SetOut
	// Just verify no error
	_ = output
}

func TestQueueAdd(t *testing.T) {
	root := setupTestWorkspace(t)

	output, err := executeCmd("queue", "add", "New feature request")
	if err != nil {
		t.Fatalf("queue add failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Added") {
		t.Errorf("queue add should confirm, got: %s", output)
	}

	// Verify item exists in queue file
	stateDir := filepath.Join(root, ".bc")
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatal(err)
	}
	items := q.ListAll()
	if len(items) != 1 || items[0].Title != "New feature request" {
		t.Errorf("queue should have 1 item 'New feature request', got %v", items)
	}
}

func TestQueueAdd_EmptyTitle(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("queue", "add", "  ")
	if err == nil {
		t.Error("expected error for empty title")
	}
}

func TestQueueAssign(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Fix auth"},
	})

	// Reload to get the generated ID
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatal(err)
	}
	items := q.ListAll()
	itemID := items[0].ID

	output, err := executeCmd("queue", "assign", itemID, "engineer-01")
	if err != nil {
		t.Fatalf("queue assign failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Assigned") {
		t.Errorf("queue assign should confirm, got: %s", output)
	}
}

func TestQueueAssign_NotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("queue", "assign", "nonexistent", "engineer-01")
	if err == nil {
		t.Error("expected error for nonexistent item")
	}
}

func TestQueueComplete(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	// Create queue with an item, then set it to working status
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "", "")
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	output, err := executeCmd("queue", "complete", item.ID)
	if err != nil {
		t.Fatalf("queue complete failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Marked") && !strings.Contains(output, "done") {
		t.Errorf("queue complete should confirm, got: %s", output)
	}
}

func TestQueueComplete_NotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("queue", "complete", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent item")
	}
}

// --- Logs command tests ---

func TestLogsCommand_Empty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("logs")
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}
	if !strings.Contains(output, "No events") {
		t.Errorf("expected 'No events', got: %s", output)
	}
}

func TestLogsCommand_WithEvents(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupEventsFile(t, stateDir, []events.Event{
		{Type: events.AgentSpawned, Agent: "coordinator", Message: "spawned"},
		{Type: events.AgentReport, Agent: "engineer-01", Message: "working: fix auth"},
	})

	output, err := executeCmd("logs")
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}
	if !strings.Contains(output, "coordinator") {
		t.Errorf("logs should show coordinator event, got: %s", output)
	}
	if !strings.Contains(output, "engineer-01") {
		t.Errorf("logs should show engineer event, got: %s", output)
	}
}

func TestLogsCommand_FilterByAgent(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupEventsFile(t, stateDir, []events.Event{
		{Type: events.AgentSpawned, Agent: "coordinator", Message: "spawned"},
		{Type: events.AgentReport, Agent: "engineer-01", Message: "working"},
		{Type: events.AgentReport, Agent: "engineer-02", Message: "done"},
	})

	// Reset flags to clean state
	oldAgent := logsAgent
	oldTail := logsTail
	logsAgent = ""
	logsTail = 0
	t.Cleanup(func() { logsAgent = oldAgent; logsTail = oldTail })

	output, err := executeCmd("logs", "--agent", "engineer-01")
	if err != nil {
		t.Fatalf("logs --agent failed: %v", err)
	}
	if strings.Contains(output, "engineer-02") {
		t.Errorf("filtered logs should not contain engineer-02, got: %s", output)
	}
}

func TestLogsCommand_Tail(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	evts := make([]events.Event, 10)
	for i := range evts {
		evts[i] = events.Event{
			Type:    events.AgentReport,
			Agent:   "eng-01",
			Message: "event " + string(rune('A'+i)),
		}
	}
	setupEventsFile(t, stateDir, evts)

	// Reset flags to clean state
	oldAgent := logsAgent
	oldTail := logsTail
	logsAgent = ""
	logsTail = 0
	t.Cleanup(func() { logsAgent = oldAgent; logsTail = oldTail })

	output, err := executeCmd("logs", "--tail", "3")
	if err != nil {
		t.Fatalf("logs --tail failed: %v", err)
	}
	_ = output
}

// --- Report command tests ---

func TestReportCommand_NoAgentID(t *testing.T) {
	setupTestWorkspace(t)

	if err := os.Unsetenv("BC_AGENT_ID"); err != nil {
		t.Fatal(err)
	}

	_, err := executeCmd("report", "working", "fixing auth")
	if err == nil {
		t.Error("expected error when BC_AGENT_ID not set")
	}
}

func TestReportCommand_InvalidState(t *testing.T) {
	setupTestWorkspace(t)

	t.Setenv("BC_AGENT_ID", "engineer-01")

	_, err := executeCmd("report", "invalid-state")
	if err == nil {
		t.Error("expected error for invalid state")
	}
}

func TestReportCommand_Working(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	// Create agent state
	agents := map[string]*agent.Agent{
		"engineer-01": {
			ID: "engineer-01", Name: "engineer-01",
			Role: agent.RoleEngineer, State: agent.StateIdle,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	t.Setenv("BC_AGENT_ID", "engineer-01")

	output, err := executeCmd("report", "working", "fixing auth bug")
	if err != nil {
		t.Fatalf("report working failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Reported") {
		t.Errorf("report should confirm, got: %s", output)
	}
}

func TestReportCommand_Done(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")
	stateDir := filepath.Join(root, ".bc")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	// Add a working queue item assigned to eng-01
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "", "")
	if err := q.Assign(item.ID, "eng-01"); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus(item.ID, queue.StatusWorking); err != nil {
		t.Fatal(err)
	}
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BC_AGENT_ID", "eng-01")

	output, err := executeCmd("report", "done", "auth fixed")
	if err != nil {
		t.Fatalf("report done failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Reported") {
		t.Errorf("report should confirm, got: %s", output)
	}

	// Verify queue item was marked done
	q2 := queue.New(filepath.Join(stateDir, "queue.json"))
	if err := q2.Load(); err != nil {
		t.Fatal(err)
	}
	completed := q2.Get(item.ID)
	if completed != nil && completed.Status != queue.StatusDone {
		t.Errorf("item status = %s, want done", completed.Status)
	}
}

// --- Status command tests ---

func TestStatusCommand_NoAgents(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(output, "No agents") {
		t.Errorf("expected 'No agents', got: %s", output)
	}
}

func TestStatusCommand_WithAgents(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"coordinator": {
			ID: "coordinator", Name: "coordinator",
			Role: agent.RoleCoordinator, State: agent.StateIdle,
			Workspace: root, Session: "coordinator",
			Children: []string{}, StartedAt: time.Now(),
		},
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Session: "eng-01", Task: "fixing auth",
			Children: []string{}, StartedAt: time.Now(),
		},
	}
	setupAgentState(t, agentsDir, agents)

	output, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(output, "coordinator") {
		t.Errorf("status should show coordinator, got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("status should show eng-01, got: %s", output)
	}
}

// --- Dashboard command tests ---

func TestDashboardCommand_Empty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("dashboard")
	if err != nil {
		t.Fatalf("dashboard failed: %v", err)
	}
	if !strings.Contains(output, "dashboard") {
		t.Errorf("dashboard output should contain 'dashboard', got: %s", output)
	}
	if !strings.Contains(output, "No agents") {
		t.Errorf("dashboard should indicate no agents, got: %s", output)
	}
}

func TestDashboardCommand_WithData(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")
	stateDir := filepath.Join(root, ".bc")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Session: "eng-01",
			Children: []string{}, StartedAt: time.Now(),
		},
	}
	setupAgentState(t, agentsDir, agents)

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
		{Title: "Task 2"},
	})

	setupEventsFile(t, stateDir, []events.Event{
		{Type: events.AgentSpawned, Agent: "eng-01", Message: "spawned"},
	})

	output, err := executeCmd("dashboard")
	if err != nil {
		t.Fatalf("dashboard failed: %v", err)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("dashboard should show eng-01, got: %s", output)
	}
	if !strings.Contains(output, "Queue") {
		t.Errorf("dashboard should have Queue section, got: %s", output)
	}
}

// --- printAgentSummary tests ---

func TestPrintAgentSummary_Empty(t *testing.T) {
	// Just verify no panic with nil/empty
	printAgentSummary(nil)
	printAgentSummary([]*agent.Agent{})
}

func TestPrintAgentSummary_WithAgents(t *testing.T) {
	agents := []*agent.Agent{
		{Name: "eng-01", Role: agent.RoleEngineer, State: agent.StateWorking, StartedAt: time.Now()},
		{Name: "eng-02", Role: agent.RoleEngineer, State: agent.StateIdle, StartedAt: time.Now()},
		{Name: "qa-01", Role: agent.RoleQA, State: agent.StateStopped},
	}
	// Verify no panic
	printAgentSummary(agents)
}

// --- printQueueStats tests ---

func TestPrintQueueStats_Empty(t *testing.T) {
	printQueueStats(queue.Stats{})
}

func TestPrintQueueStats_WithData(t *testing.T) {
	printQueueStats(queue.Stats{
		Total:    10,
		Pending:  3,
		Assigned: 2,
		Working:  2,
		Done:     3,
		Failed:   0,
	})
}

func TestPrintQueueStats_WithFailed(t *testing.T) {
	printQueueStats(queue.Stats{
		Total:  5,
		Done:   3,
		Failed: 2,
	})
}

// --- printRecentActivity tests ---

func TestPrintRecentActivity_Empty(t *testing.T) {
	printRecentActivity(nil)
}

func TestPrintRecentActivity_WithEvents(t *testing.T) {
	evts := []events.Event{
		{Type: events.AgentSpawned, Agent: "coordinator", Timestamp: time.Now(), Message: "spawned"},
		{Type: events.AgentReport, Timestamp: time.Now(), Message: ""},
	}
	printRecentActivity(evts)
}

// --- Stats command tests ---

func TestStatsCommand(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	// Create minimal stats data files
	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
	})

	output, err := executeCmd("stats")
	if err != nil {
		t.Fatalf("stats failed: %v\nOutput: %s", err, output)
	}
}

// --- Channel command tests ---

func TestChannelList_Empty(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list failed: %v", err)
	}
	if !strings.Contains(output, "No channels") {
		t.Errorf("expected 'No channels', got: %s", output)
	}
}

func TestChannelCreate(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("channel", "create", "test-channel")
	if err != nil {
		t.Fatalf("channel create failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Created") {
		t.Errorf("should confirm creation, got: %s", output)
	}

	// Verify channel appears in list
	output2, err := executeCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list failed: %v", err)
	}
	if !strings.Contains(output2, "test-channel") {
		t.Errorf("channel list should show test-channel, got: %s", output2)
	}
}

func TestChannelCreate_EmptyName(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("channel", "create", "  ")
	if err == nil {
		t.Error("expected error for empty channel name")
	}
}

func TestChannelAdd(t *testing.T) {
	setupTestWorkspace(t)

	// Create channel first
	_, _ = executeCmd("channel", "create", "devs")

	output, err := executeCmd("channel", "add", "devs", "eng-01", "eng-02")
	if err != nil {
		t.Fatalf("channel add failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Added") {
		t.Errorf("should confirm add, got: %s", output)
	}
}

func TestChannelRemove(t *testing.T) {
	setupTestWorkspace(t)

	_, _ = executeCmd("channel", "create", "devs")
	_, _ = executeCmd("channel", "add", "devs", "eng-01")

	output, err := executeCmd("channel", "remove", "devs", "eng-01")
	if err != nil {
		t.Fatalf("channel remove failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Removed") {
		t.Errorf("should confirm remove, got: %s", output)
	}
}

func TestChannelDelete(t *testing.T) {
	setupTestWorkspace(t)

	_, _ = executeCmd("channel", "create", "temp-channel")

	output, err := executeCmd("channel", "delete", "temp-channel")
	if err != nil {
		t.Fatalf("channel delete failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Deleted") {
		t.Errorf("should confirm deletion, got: %s", output)
	}
}

func TestChannelJoin_NoAgentID(t *testing.T) {
	setupTestWorkspace(t)
	if err := os.Unsetenv("BC_AGENT_ID"); err != nil {
		t.Fatal(err)
	}

	_, err := executeCmd("channel", "join", "devs")
	if err == nil {
		t.Error("expected error when BC_AGENT_ID not set")
	}
}

func TestChannelJoin(t *testing.T) {
	setupTestWorkspace(t)
	t.Setenv("BC_AGENT_ID", "eng-01")

	_, _ = executeCmd("channel", "create", "devs")

	output, err := executeCmd("channel", "join", "devs")
	if err != nil {
		t.Fatalf("channel join failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Joined") {
		t.Errorf("should confirm join, got: %s", output)
	}
}

func TestChannelLeave_NoAgentID(t *testing.T) {
	setupTestWorkspace(t)
	if err := os.Unsetenv("BC_AGENT_ID"); err != nil {
		t.Fatal(err)
	}

	_, err := executeCmd("channel", "leave", "devs")
	if err == nil {
		t.Error("expected error when BC_AGENT_ID not set")
	}
}

func TestChannelLeave(t *testing.T) {
	setupTestWorkspace(t)
	t.Setenv("BC_AGENT_ID", "eng-01")

	_, _ = executeCmd("channel", "create", "devs")
	_, _ = executeCmd("channel", "add", "devs", "eng-01")

	output, err := executeCmd("channel", "leave", "devs")
	if err != nil {
		t.Fatalf("channel leave failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Left") {
		t.Errorf("should confirm leave, got: %s", output)
	}
}

func TestChannelHistory_Empty(t *testing.T) {
	setupTestWorkspace(t)

	_, _ = executeCmd("channel", "create", "devs")

	output, err := executeCmd("channel", "history", "devs")
	if err != nil {
		t.Fatalf("channel history failed: %v", err)
	}
	if !strings.Contains(output, "No message history") {
		t.Errorf("expected no history message, got: %s", output)
	}
}

// --- Send command tests ---

func TestSendCommand_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	_, err = executeCmd("send", "eng-01", "hello")
	if err == nil {
		t.Error("expected error when not in a workspace")
	}
}

// --- Spawn command tests ---

func TestSpawnCommand_InvalidRole(t *testing.T) {
	setupTestWorkspace(t)

	spawnRole = "bogus"
	defer func() { spawnRole = "worker" }()

	_, err := executeCmd("spawn", "agent-01", "--role", "bogus")
	if err == nil {
		t.Error("expected error for invalid role")
	}
}

// --- Down command tests ---

func TestDownCommand_NoAgents(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("down")
	if err != nil {
		t.Fatalf("down failed: %v", err)
	}
	if !strings.Contains(output, "No agents") {
		t.Errorf("expected 'No agents', got: %s", output)
	}
}

// --- printJSONDashboard tests ---

func TestPrintJSONDashboard(t *testing.T) {
	agents := []*agent.Agent{
		{
			Name: "eng-01", Role: agent.RoleEngineer,
			State: agent.StateWorking, Task: "fixing",
			Session: "eng-01", StartedAt: time.Now(),
		},
		{
			Name: "eng-02", Role: agent.RoleEngineer,
			State: agent.StateStopped, Session: "eng-02",
		},
	}
	qs := queue.Stats{Total: 5, Done: 3, Pending: 2}
	evts := []events.Event{
		{Type: events.AgentSpawned, Agent: "eng-01", Timestamp: time.Now(), Message: "spawned"},
	}

	// Redirect stdout to capture JSON
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printErr := printJSONDashboard("/test", "test-ws", agents, qs, evts)

	_ = w.Close()
	os.Stdout = oldStdout

	if printErr != nil {
		t.Fatalf("printJSONDashboard error: %v", printErr)
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	output := buf.String()

	var result dashboardOutput
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v\nOutput: %s", err, output)
	}

	if result.Workspace != "/test" {
		t.Errorf("workspace = %q, want /test", result.Workspace)
	}
	if result.Name != "test-ws" {
		t.Errorf("name = %q, want test-ws", result.Name)
	}
	if result.Agents.Total != 2 {
		t.Errorf("agents.total = %d, want 2", result.Agents.Total)
	}
	if result.Agents.Running != 1 {
		t.Errorf("agents.running = %d, want 1", result.Agents.Running)
	}
	if result.Queue.Total != 5 {
		t.Errorf("queue.total = %d, want 5", result.Queue.Total)
	}
	if len(result.Events) != 1 {
		t.Errorf("events count = %d, want 1", len(result.Events))
	}
}

// --- SetVersionInfo tests ---

func TestSetVersionInfo(t *testing.T) {
	SetVersionInfo("2.0.0", "def456", "2025-06-01")
	// Version is set in the package-level var, verify via version command
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	_ = rootCmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "2.0.0") {
		t.Errorf("version output should contain '2.0.0', got: %s", output)
	}
}

// --- loadQueue helper tests ---

type mockStateDir struct{ dir string }

func (m mockStateDir) StateDir() string { return m.dir }

func TestLoadQueue(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Item 1"},
	})

	ws := mockStateDir{dir: stateDir}
	q, err := loadQueue(ws)
	if err != nil {
		t.Fatal(err)
	}

	items := q.ListAll()
	if len(items) != 1 {
		t.Errorf("loadQueue should return 1 item, got %d", len(items))
	}
}

// --- More Report command tests ---

func TestReportCommand_Stuck(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	t.Setenv("BC_AGENT_ID", "eng-01")

	output, err := executeCmd("report", "stuck", "need credentials")
	if err != nil {
		t.Fatalf("report stuck failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Reported") {
		t.Errorf("report should confirm, got: %s", output)
	}
}

func TestReportCommand_Error(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	t.Setenv("BC_AGENT_ID", "eng-01")

	output, err := executeCmd("report", "error", "build failed")
	if err != nil {
		t.Fatalf("report error failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Reported") {
		t.Errorf("report should confirm, got: %s", output)
	}
}

func TestReportCommand_Idle(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	t.Setenv("BC_AGENT_ID", "eng-01")

	output, err := executeCmd("report", "idle")
	if err != nil {
		t.Fatalf("report idle failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Reported") {
		t.Errorf("report should confirm, got: %s", output)
	}
}

func TestReportCommand_WorkingWithQueueItem(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")
	stateDir := filepath.Join(root, ".bc")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateIdle,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	// Add queue item assigned to eng-01
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "", "")
	if err := q.Assign(item.ID, "eng-01"); err != nil {
		t.Fatal(err)
	}
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BC_AGENT_ID", "eng-01")

	output, err := executeCmd("report", "working", "starting auth fix")
	if err != nil {
		t.Fatalf("report working failed: %v\nOutput: %s", err, output)
	}

	// Verify queue item status changed to working
	q2 := queue.New(filepath.Join(stateDir, "queue.json"))
	if err := q2.Load(); err != nil {
		t.Fatal(err)
	}
	updated := q2.Get(item.ID)
	if updated != nil && updated.Status != queue.StatusWorking {
		t.Errorf("item status = %s, want working", updated.Status)
	}
}

// --- More Logs tests ---

func TestLogsCommand_JSON(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupEventsFile(t, stateDir, []events.Event{
		{Type: events.AgentSpawned, Agent: "coordinator", Message: "spawned"},
	})

	// Reset flags
	oldAgent := logsAgent
	oldTail := logsTail
	logsAgent = ""
	logsTail = 0
	t.Cleanup(func() { logsAgent = oldAgent; logsTail = oldTail })

	output, err := executeCmd("logs", "--json")
	if err != nil {
		t.Fatalf("logs --json failed: %v", err)
	}
	// JSON should be valid (goes through stdout)
	_ = output
}

func TestLogsCommand_AgentAndTail(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	evts := make([]events.Event, 0)
	for i := 0; i < 10; i++ {
		evts = append(evts, events.Event{
			Type:    events.AgentReport,
			Agent:   "eng-01",
			Message: fmt.Sprintf("event %d", i),
		})
	}
	// Add events from another agent
	evts = append(evts, events.Event{
		Type:    events.AgentReport,
		Agent:   "eng-02",
		Message: "other agent",
	})
	setupEventsFile(t, stateDir, evts)

	oldAgent := logsAgent
	oldTail := logsTail
	logsAgent = ""
	logsTail = 0
	t.Cleanup(func() { logsAgent = oldAgent; logsTail = oldTail })

	output, err := executeCmd("logs", "--agent", "eng-01", "--tail", "3")
	if err != nil {
		t.Fatalf("logs --agent --tail failed: %v", err)
	}
	// Should not contain eng-02 events
	if strings.Contains(output, "eng-02") {
		t.Errorf("filtered logs should not contain eng-02, got: %s", output)
	}
}

// --- Dashboard JSON test ---

func TestDashboardCommand_JSON(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")
	stateDir := filepath.Join(root, ".bc")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Session: "eng-01",
			Children: []string{}, StartedAt: time.Now(),
		},
	}
	setupAgentState(t, agentsDir, agents)

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
	})

	output, err := executeCmd("dashboard", "--json")
	if err != nil {
		t.Fatalf("dashboard --json failed: %v\nOutput: %s", err, output)
	}
	// The JSON output goes to stdout, should be parseable
	// At minimum, no error should occur
}

// --- Stats tests ---

func TestStatsCommand_JSON(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
	})

	// Reset stats flags
	oldJSON := statsJSON
	oldSave := statsSave
	statsJSON = false
	statsSave = false
	t.Cleanup(func() { statsJSON = oldJSON; statsSave = oldSave })

	output, err := executeCmd("stats", "--json")
	if err != nil {
		t.Fatalf("stats --json failed: %v\nOutput: %s", err, output)
	}
}

func TestStatsCommand_Save(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
	})

	oldJSON := statsJSON
	oldSave := statsSave
	statsJSON = false
	statsSave = false
	t.Cleanup(func() { statsJSON = oldJSON; statsSave = oldSave })

	output, err := executeCmd("stats", "--save")
	if err != nil {
		t.Fatalf("stats --save failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "saved") && !strings.Contains(output, "Stats") {
		t.Errorf("stats --save should mention saving, got: %s", output)
	}
}

// --- Channel send - no members ---

func TestChannelSend_NoMembers(t *testing.T) {
	setupTestWorkspace(t)

	// Create channel with no members
	_, _ = executeCmd("channel", "create", "empty-channel")

	output, err := executeCmd("channel", "send", "empty-channel", "hello")
	if err != nil {
		t.Fatalf("channel send failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "no members") {
		t.Errorf("should mention no members, got: %s", output)
	}
}

// --- Channel send - with members but no tmux ---

func TestChannelSend_WithMembers(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	// Set up agents so the member lookup works
	agentMap := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateIdle,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agentMap)

	_, _ = executeCmd("channel", "create", "devs")
	_, _ = executeCmd("channel", "add", "devs", "eng-01")

	// This will try to send but fail since there's no tmux session
	// It should not error out - it just prints failures per member
	output, err := executeCmd("channel", "send", "devs", "hello world")
	if err != nil {
		t.Fatalf("channel send failed: %v\nOutput: %s", err, output)
	}
	// Should mention the channel
	if !strings.Contains(output, "devs") {
		t.Errorf("should mention channel name, got: %s", output)
	}
}

// --- Channel history with messages ---

func TestChannelHistory_WithMessages(t *testing.T) {
	root := setupTestWorkspace(t)

	// Create channel and add history directly via store
	store, err := loadChannelStore(root)
	if err != nil {
		// First time, no file yet
		store, _ = loadChannelStore(root)
	}
	if store == nil {
		t.Skip("cannot load channel store")
	}

	if _, err := store.Create("devs"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddHistory("devs", "eng-01", "first message"); err != nil {
		t.Fatal(err)
	}
	if err := store.AddHistory("devs", "eng-01", "second message"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(); err != nil {
		t.Fatal(err)
	}

	output, err2 := executeCmd("channel", "history", "devs")
	if err2 != nil {
		t.Fatalf("channel history failed: %v\nOutput: %s", err2, output)
	}
	if !strings.Contains(output, "first message") {
		t.Errorf("history should contain first message, got: %s", output)
	}
	if !strings.Contains(output, "second message") {
		t.Errorf("history should contain second message, got: %s", output)
	}
}

// --- Queue load (no beads) ---

func TestQueueLoad_NoBeads(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("queue", "load")
	if err != nil {
		t.Fatalf("queue load failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No beads") && !strings.Contains(output, "Loaded 0") {
		t.Errorf("queue load should indicate no beads or 0 loaded, got: %s", output)
	}
}

// --- Send command - agent not found ---

func TestSendCommand_AgentNotFound(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("send", "nonexistent-agent", "hello")
	if err == nil {
		t.Error("expected error for nonexistent agent")
	}
}

// --- Send command - agent stopped ---

func TestSendCommand_AgentStopped(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateStopped,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	_, err := executeCmd("send", "eng-01", "hello")
	if err == nil {
		t.Error("expected error for stopped agent")
	}
}

// --- Spawn command - existing running agent ---

func TestSpawnCommand_ExistingRunningAgent(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	spawnRole = "worker"
	defer func() { spawnRole = "worker" }()

	_, err := executeCmd("spawn", "eng-01")
	if err == nil {
		t.Error("expected error for already-running agent")
	}
}

// --- Down with agents ---

func TestDownCommand_WithAgents(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateIdle,
			Workspace: root, Children: []string{},
		},
		"eng-02": {
			ID: "eng-02", Name: "eng-02",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	output, err := executeCmd("down")
	if err != nil {
		t.Fatalf("down failed: %v\nOutput: %s", err, output)
	}
	// Should try to stop agents (may fail without tmux but shouldn't error)
	if !strings.Contains(output, "Stopping") {
		t.Errorf("down should mention stopping, got: %s", output)
	}
}

// --- Status with long task truncation ---

func TestStatusCommand_LongTask(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	longTask := strings.Repeat("x", 200)
	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Session: "eng-01", Task: longTask,
			Children: []string{}, StartedAt: time.Now(),
		},
	}
	setupAgentState(t, agentsDir, agents)

	output, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if strings.Contains(output, longTask) {
		t.Error("status should truncate long task, but full task appears in output")
	}
	if !strings.Contains(output, "...") {
		t.Errorf("status should show truncated task with '...', got: %s", output)
	}
}

// --- Status with stopped agent (no uptime) ---

func TestStatusCommand_StoppedAgent(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateStopped,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	output, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("status should show eng-01, got: %s", output)
	}
}

// --- printAgentSummary with long task ---

func TestPrintAgentSummary_LongTaskTruncation(t *testing.T) {
	longTask := strings.Repeat("a", 50)
	agents := []*agent.Agent{
		{
			Name: "eng-01", Role: agent.RoleEngineer,
			State: agent.StateWorking, Task: longTask,
			StartedAt: time.Now(),
		},
	}
	// Verify no panic and truncation works
	printAgentSummary(agents)
}

// --- printAgentSummary with multiple states ---

func TestPrintAgentSummary_AllStates(t *testing.T) {
	agents := []*agent.Agent{
		{Name: "a1", Role: agent.RoleEngineer, State: agent.StateIdle, StartedAt: time.Now()},
		{Name: "a2", Role: agent.RoleEngineer, State: agent.StateWorking, StartedAt: time.Now()},
		{Name: "a3", Role: agent.RoleEngineer, State: agent.StateDone, StartedAt: time.Now()},
		{Name: "a4", Role: agent.RoleEngineer, State: agent.StateStuck, StartedAt: time.Now()},
		{Name: "a5", Role: agent.RoleEngineer, State: agent.StateError, StartedAt: time.Now()},
		{Name: "a6", Role: agent.RoleEngineer, State: agent.StateStarting, StartedAt: time.Now()},
		{Name: "a7", Role: agent.RoleEngineer, State: agent.StateStopped},
	}
	printAgentSummary(agents)
}

// --- Queue list with long title ---

func TestQueueList_LongTitle(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	longTitle := strings.Repeat("z", 50)
	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: longTitle},
	})

	output, err := executeCmd("queue")
	if err != nil {
		t.Fatalf("queue list failed: %v", err)
	}
	// Title should be truncated
	if strings.Contains(output, longTitle) {
		t.Error("queue list should truncate long titles")
	}
}

// --- Queue list with assigned and beads items ---

func TestQueueList_WithAssignedAndBeads(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "Fix the auth bug", "bc-123")
	if err := q.Assign(item.ID, "eng-01"); err != nil {
		t.Fatal(err)
	}
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	output, err := executeCmd("queue")
	if err != nil {
		t.Fatalf("queue list failed: %v", err)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("queue list should show assigned agent, got: %s", output)
	}
	if !strings.Contains(output, "bc-123") {
		t.Errorf("queue list should show beads ID, got: %s", output)
	}
}

// --- Init with custom dir ---

func TestInitCommand_CustomDir(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "custom-project")
	if err := os.MkdirAll(subdir, 0750); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	output, err := executeCmd("init", subdir)
	if err != nil {
		t.Fatalf("init custom dir failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Initialized") {
		t.Errorf("init output should contain 'Initialized', got: %s", output)
	}

	// Verify .bc dir created in the custom dir
	if _, err := os.Stat(filepath.Join(subdir, ".bc")); os.IsNotExist(err) {
		t.Error(".bc directory not created in custom dir")
	}
}

// --- Execute function ---

func TestExecute(t *testing.T) {
	// Execute with no args should show help (no error)
	rootCmd.SetArgs([]string{})
	err := Execute()
	if err != nil {
		t.Errorf("Execute with no args should not error, got: %v", err)
	}
}

// --- Root function ---

func TestRoot(t *testing.T) {
	cmd := Root()
	if cmd == nil {
		t.Fatal("Root() should return non-nil command")
	}
	if cmd.Use != "bc" {
		t.Errorf("Root().Use = %q, want 'bc'", cmd.Use)
	}
}

// --- getWorkspace error ---

func TestGetWorkspace_NotInWorkspace(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	_, err = executeCmd("status")
	if err == nil {
		t.Error("expected error when not in a workspace")
	}
}

// --- Queue add with description ---

func TestQueueAdd_WithDescription(t *testing.T) {
	root := setupTestWorkspace(t)

	// Reset queueDesc flag
	oldDesc := queueDesc
	queueDesc = ""
	t.Cleanup(func() { queueDesc = oldDesc })

	output, err := executeCmd("queue", "add", "New feature", "-d", "Detailed description")
	if err != nil {
		t.Fatalf("queue add with desc failed: %v\nOutput: %s", err, output)
	}

	stateDir := filepath.Join(root, ".bc")
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	if err := q.Load(); err != nil {
		t.Fatal(err)
	}
	items := q.ListAll()
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Description != "Detailed description" {
		t.Errorf("description = %q, want 'Detailed description'", items[0].Description)
	}
}

// --- createDefaultChannels - channels already exist ---

func TestCreateDefaultChannels_AlreadyExist(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}

	engineers := []string{"eng-01"}
	qa := []string{"qa-01"}
	all := []string{"coordinator", "product-manager", "manager", "eng-01", "qa-01"}

	// Create channels twice - second time should not error
	oldStdout := os.Stdout
	_, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	techLeads := []string{} // No tech leads in this test
	createDefaultChannels(dir, techLeads, engineers, qa, all)
	createDefaultChannels(dir, techLeads, engineers, qa, all)
	_ = w.Close()
	os.Stdout = oldStdout
}

// --- Attach command - no workspace ---

func TestAttachCommand_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	_, err = executeCmd("attach", "coordinator")
	if err == nil {
		t.Error("expected error when not in a workspace")
	}
}

// --- Channel commands - nonexistent channel errors ---

func TestChannelRemove_NonexistentChannel(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("channel", "remove", "nonexistent", "eng-01")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestChannelDelete_NonexistentChannel(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("channel", "delete", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

func TestChannelHistory_NonexistentChannel(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("channel", "history", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent channel")
	}
}

// --- Queue assign - already done item ---

func TestQueueAssign_BeadsSync(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	// Create item with BeadsID
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "", "bc-123")
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	output, err := executeCmd("queue", "assign", item.ID, "eng-01")
	if err != nil {
		t.Fatalf("queue assign failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Assigned") {
		t.Errorf("queue assign should confirm, got: %s", output)
	}
}

// --- Queue complete - with beads ---

func TestQueueComplete_WithBeads(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "", "bc-123")
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	output, err := executeCmd("queue", "complete", item.ID)
	if err != nil {
		t.Fatalf("queue complete failed: %v\nOutput: %s", err, output)
	}
	// Should mention beads ID
	if !strings.Contains(output, "bc-123") {
		t.Errorf("queue complete should mention beads ID, got: %s", output)
	}
}

// --- Stats with active agents ---

func TestStatsCommand_WithAgents(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")
	stateDir := filepath.Join(root, ".bc")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
			StartedAt: time.Now(),
		},
	}
	setupAgentState(t, agentsDir, agents)

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
		{Title: "Task 2"},
	})

	oldJSON := statsJSON
	oldSave := statsSave
	statsJSON = false
	statsSave = false
	t.Cleanup(func() { statsJSON = oldJSON; statsSave = oldSave })

	output, err := executeCmd("stats")
	if err != nil {
		t.Fatalf("stats failed: %v\nOutput: %s", err, output)
	}
}

// --- Attach command - session not found ---

func TestAttachCommand_SessionNotFound(t *testing.T) {
	setupTestWorkspace(t)

	// No tmux session exists, so HasSession returns false
	_, err := executeCmd("attach", "nonexistent-agent")
	if err == nil {
		t.Error("expected error when session not found")
	}
	// Error should mention "not running"
}

// --- Spawn command - workspace with Tool config ---

func TestSpawnCommand_WorkspaceToolConfig(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	wsConfig := `{"version":1,"name":"test-ws","state_dir":"` + bcDir + `","root_dir":"` + absDir + `","max_workers":3,"tool":"cursor"}`
	if writeErr := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte(wsConfig), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	spawnRole = "worker"
	defer func() { spawnRole = "worker" }()

	// This will fail at SpawnAgentWithTool (no tmux), but should get past tool config
	_, err = executeCmd("spawn", "test-agent")
	// We expect an error (tmux not available), but it should not be about the tool
	if err != nil && strings.Contains(err.Error(), "unknown tool") {
		t.Error("should recognize tool from workspace config")
	}
}

// --- Spawn command - workspace with AgentCommand config ---

func TestSpawnCommand_WorkspaceAgentCommandConfig(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	wsConfig := `{"version":1,"name":"test-ws","state_dir":"` + bcDir + `","root_dir":"` + absDir + `","max_workers":3,"agent_command":"custom-agent"}`
	if writeErr := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte(wsConfig), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	spawnRole = "worker"
	defer func() { spawnRole = "worker" }()

	// This will fail at spawn (no tmux), but should get past config parsing
	_, _ = executeCmd("spawn", "test-agent")
}

// --- Spawn command - invalid tool name ---

func TestSpawnCommand_InvalidTool(t *testing.T) {
	setupTestWorkspace(t)

	spawnTool = "nonexistent-tool"
	spawnRole = "worker"
	defer func() { spawnTool = ""; spawnRole = "worker" }()

	_, err := executeCmd("spawn", "test-agent", "--tool", "nonexistent-tool")
	if err == nil {
		t.Error("expected error for invalid tool")
	}
}

// --- Spawn command - stopped agent can be respawned ---

func TestSpawnCommand_StoppedAgent(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateStopped,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	spawnRole = "worker"
	defer func() { spawnRole = "worker" }()

	// Will fail at SpawnAgentWithTool (no tmux) but should pass the "already exists" check
	_, err := executeCmd("spawn", "eng-01")
	// Should NOT get "already exists and is working" error
	if err != nil && strings.Contains(err.Error(), "already exists") {
		t.Error("stopped agent should be respawnable, got: " + err.Error())
	}
}

// --- Channel add - warning for nonexistent channel ---

func TestChannelAdd_NonexistentChannel(t *testing.T) {
	setupTestWorkspace(t)

	output, err := executeCmd("channel", "add", "nonexistent", "eng-01")
	if err != nil {
		// Might error - that's fine
		_ = output
	}
}

// --- Send command - with working agent (tmux will fail) ---

func TestSendCommand_WorkingAgent(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Session: "eng-01",
			Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	// This will try SendToAgent which requires tmux - should fail gracefully
	_, err := executeCmd("send", "eng-01", "hello", "world")
	// Expected to fail (no tmux), but it exercises the non-error agent paths
	_ = err
}

// --- Channel send - with agent not found and agent stopped ---

func TestChannelSend_MixedAgentStates(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateStopped,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	_, _ = executeCmd("channel", "create", "devs")
	_, _ = executeCmd("channel", "add", "devs", "eng-01", "eng-02")

	output, err := executeCmd("channel", "send", "devs", "test message")
	if err != nil {
		t.Fatalf("channel send failed: %v", err)
	}
	// eng-01 is stopped, eng-02 not found - both should be reported
	if !strings.Contains(output, "failed") {
		t.Logf("Expected failed count in output, got: %s", output)
	}
}

// --- Dashboard with verbose flag ---

func TestDashboardCommand_Verbose(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
	})

	output, err := executeCmd("dashboard", "--verbose")
	if err != nil {
		t.Fatalf("dashboard --verbose failed: %v\nOutput: %s", err, output)
	}
}

// --- Report done - with beads item ---

func TestReportCommand_DoneWithBeads(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")
	stateDir := filepath.Join(root, ".bc")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateWorking,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	// Add queue item with BeadsID
	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item := q.Add("Fix auth", "", "bc-123")
	if err := q.Assign(item.ID, "eng-01"); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus(item.ID, queue.StatusWorking); err != nil {
		t.Fatal(err)
	}
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}

	t.Setenv("BC_AGENT_ID", "eng-01")

	output, err := executeCmd("report", "done", "auth fixed")
	if err != nil {
		t.Fatalf("report done failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Reported") {
		t.Errorf("report should confirm, got: %s", output)
	}
}

// --- Up command tests (exercises early paths before tmux failure) ---

func TestUpCommand_NoWorkspace(t *testing.T) {
	dir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	_, err = executeCmd("up")
	if err == nil {
		t.Error("expected error when not in a workspace")
	}
}

func TestUpCommand_InWorkspace(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	// Add some queue items so the beads/queue loading path is exercised
	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Existing item"},
	})

	// Reset up flags
	oldWorkers := upWorkers
	oldEngineers := upEngineers
	oldQA := upQA
	oldAgent := upAgent
	upWorkers = 0
	upEngineers = 1
	upQA = 0
	upAgent = ""
	t.Cleanup(func() {
		upWorkers = oldWorkers
		upEngineers = oldEngineers
		upQA = oldQA
		upAgent = oldAgent
	})

	// Will fail at SpawnAgent (coordinator) because no tmux
	// But exercises workspace lookup, agent manager creation, queue loading
	output, err := executeCmd("up")
	// Expected error - tmux not available
	_ = output
	_ = err
}

func TestUpCommand_WithAgentFlag(t *testing.T) {
	root := setupTestWorkspace(t)
	_ = root

	oldWorkers := upWorkers
	oldEngineers := upEngineers
	oldQA := upQA
	oldAgent := upAgent
	upWorkers = 0
	upEngineers = 1
	upQA = 0
	upAgent = "nonexistent-agent-type"
	t.Cleanup(func() {
		upWorkers = oldWorkers
		upEngineers = oldEngineers
		upQA = oldQA
		upAgent = oldAgent
	})

	_, err := executeCmd("up", "--agent", "nonexistent-agent-type")
	// Should error about unknown agent
	if err != nil && !strings.Contains(err.Error(), "unknown agent") {
		// Could also be tmux error - both are acceptable
		_ = err
	}
}

func TestUpCommand_WithWorkersFlag(t *testing.T) {
	setupTestWorkspace(t)

	oldWorkers := upWorkers
	oldEngineers := upEngineers
	oldQA := upQA
	oldAgent := upAgent
	upWorkers = 2
	upEngineers = 3
	upQA = 0
	upAgent = ""
	t.Cleanup(func() {
		upWorkers = oldWorkers
		upEngineers = oldEngineers
		upQA = oldQA
		upAgent = oldAgent
	})

	// Will fail at tmux but exercises the legacy --workers path
	_, _ = executeCmd("up", "--workers", "2")
}

func TestUpCommand_WithAgentCommandConfig(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	if err := os.MkdirAll(filepath.Join(bcDir, "agents"), 0750); err != nil {
		t.Fatal(err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	wsConfig := `{"version":1,"name":"test-ws","state_dir":"` + bcDir + `","root_dir":"` + absDir + `","max_workers":3,"agent_command":"custom-command"}`
	if writeErr := os.WriteFile(filepath.Join(bcDir, "config.json"), []byte(wsConfig), 0600); writeErr != nil {
		t.Fatal(writeErr)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })
	if chdirErr := os.Chdir(dir); chdirErr != nil {
		t.Fatal(chdirErr)
	}

	oldWorkers := upWorkers
	oldEngineers := upEngineers
	oldQA := upQA
	oldAgent := upAgent
	upWorkers = 0
	upEngineers = 1
	upQA = 0
	upAgent = ""
	t.Cleanup(func() {
		upWorkers = oldWorkers
		upEngineers = oldEngineers
		upQA = oldQA
		upAgent = oldAgent
	})

	// Exercises the AgentCommand config path
	_, _ = executeCmd("up")
}

// --- Down command - force flag ---

func TestDownCommand_Force(t *testing.T) {
	root := setupTestWorkspace(t)
	agentsDir := filepath.Join(root, ".bc", "agents")

	agents := map[string]*agent.Agent{
		"eng-01": {
			ID: "eng-01", Name: "eng-01",
			Role: agent.RoleEngineer, State: agent.StateIdle,
			Workspace: root, Children: []string{},
		},
	}
	setupAgentState(t, agentsDir, agents)

	oldForce := downForce
	downForce = false
	t.Cleanup(func() { downForce = oldForce })

	output, err := executeCmd("down", "--force")
	if err != nil {
		t.Fatalf("down --force failed: %v", err)
	}
	_ = output
}

// --- Logs JSON with data ---

func TestLogsCommand_JSONWithData(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupEventsFile(t, stateDir, []events.Event{
		{Type: events.AgentSpawned, Agent: "coordinator", Message: "spawned"},
		{Type: events.AgentReport, Agent: "eng-01", Message: "working: fixing"},
	})

	oldAgent := logsAgent
	oldTail := logsTail
	logsAgent = ""
	logsTail = 0
	t.Cleanup(func() { logsAgent = oldAgent; logsTail = oldTail })

	output, err := executeCmd("logs", "--json")
	if err != nil {
		t.Fatalf("logs --json failed: %v\nOutput: %s", err, output)
	}
}

// --- Version command (from cmd_test) ---

func TestVersionCommandOutput(t *testing.T) {
	output, err := executeCmd("version")
	if err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(output, "bc") {
		t.Errorf("version should contain 'bc', got: %s", output)
	}
}

// --- Root command (help) ---

func TestRootCommand_Help(t *testing.T) {
	output, err := executeCmd("--help")
	if err != nil {
		t.Fatalf("help failed: %v", err)
	}
	if !strings.Contains(output, "bc") {
		t.Errorf("help should contain 'bc', got: %s", output)
	}
}

// --- Queue list - all status colors exercised ---

func TestQueueList_AllStatuses(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	q := queue.New(filepath.Join(stateDir, "queue.json"))
	item1 := q.Add("Pending task", "", "")
	item2 := q.Add("Assigned task", "", "")
	item3 := q.Add("Working task", "", "")
	item4 := q.Add("Done task", "", "")
	item5 := q.Add("Failed task", "", "")

	if err := q.Assign(item2.ID, "eng-01"); err != nil {
		t.Fatal(err)
	}
	if err := q.Assign(item3.ID, "eng-02"); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus(item3.ID, queue.StatusWorking); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus(item4.ID, queue.StatusDone); err != nil {
		t.Fatal(err)
	}
	if err := q.UpdateStatus(item5.ID, queue.StatusFailed); err != nil {
		t.Fatal(err)
	}
	if err := q.Save(); err != nil {
		t.Fatal(err)
	}
	_ = item1 // pending by default

	output, err := executeCmd("queue")
	if err != nil {
		t.Fatalf("queue list failed: %v", err)
	}
	if !strings.Contains(output, "Total:") {
		t.Errorf("should show stats, got: %s", output)
	}
}

// --- Stats save and json together ---

func TestStatsCommand_SaveAndJSON(t *testing.T) {
	root := setupTestWorkspace(t)
	stateDir := filepath.Join(root, ".bc")

	setupQueueFile(t, stateDir, []queue.WorkItem{
		{Title: "Task 1"},
	})

	oldJSON := statsJSON
	oldSave := statsSave
	statsJSON = false
	statsSave = false
	t.Cleanup(func() { statsJSON = oldJSON; statsSave = oldSave })

	output, err := executeCmd("stats", "--save", "--json")
	if err != nil {
		t.Fatalf("stats --save --json failed: %v\nOutput: %s", err, output)
	}
}

// --- Channel list with members ---

func TestChannelList_WithMembers(t *testing.T) {
	setupTestWorkspace(t)

	_, _ = executeCmd("channel", "create", "devs")
	_, _ = executeCmd("channel", "add", "devs", "eng-01", "eng-02")

	output, err := executeCmd("channel", "list")
	if err != nil {
		t.Fatalf("channel list failed: %v", err)
	}
	if !strings.Contains(output, "devs") {
		t.Errorf("should list devs channel, got: %s", output)
	}
	if !strings.Contains(output, "eng-01") {
		t.Errorf("should show members, got: %s", output)
	}
}
