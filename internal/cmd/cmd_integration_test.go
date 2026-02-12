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
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/events"
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
	if !strings.Contains(stdout, "Workspace:") {
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
			Role:      agent.Role("worker"),
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
			Role:      agent.Role("worker"),
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
			Role:      agent.Role("worker"),
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
	if _, ok := result["agents"]; !ok {
		t.Error("expected 'agents' key in JSON output")
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
			Role:      agent.Role("worker"),
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

	allAgents := []string{"coordinator", "product-manager", "manager", "engineer-01", "engineer-02", "qa-01"}

	// Capture stdout to verify output
	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	// Call with new signature: only rootDir and allAgents
	createDefaultChannels(tmpDir, allAgents)

	_ = w.Close()
	// Restore stdout
	os.Stdout = origStdout

	// Read captured output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	// output := buf.String()
	// The new implementation might not print anything if successful or print warnings.
	// We primarily care about the side effects (channels created).

	// Verify channels were created in SQLite store
	store := channel.NewSQLiteStore(tmpDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open channel store: %v", err)
	}
	defer func() { _ = store.Close() }()

	// Check required channels exist
	// In the new implementation, only "all" and per-agent channels are created.
	expectedChannels := make([]string, 0, 1+len(allAgents))
	expectedChannels = append(expectedChannels, "all")
	expectedChannels = append(expectedChannels, allAgents...)

	for _, name := range expectedChannels {
		ch, getErr := store.GetChannel(name)
		if getErr != nil {
			t.Errorf("failed to get channel %s: %v", name, getErr)
			continue
		}
		if ch == nil {
			t.Errorf("channel %s not created", name)
		}
	}
}

func TestStatusWithAgents(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedAgents(t, wsDir, map[string]*agent.Agent{
		"coordinator": {
			Name:      "coordinator",
			Role:      agent.RoleRoot,
			State:     agent.StateStopped,
			Session:   "bc-coord",
			StartedAt: time.Now().Add(-1 * time.Hour),
		},
		"worker-01": {
			Name:      "worker-01",
			Role:      agent.Role("worker"),
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

// --- Cost dashboard command tests ---
// (Basic cost show/summary tests are in cost_test.go)

// seedCostRecords creates cost records in the workspace database.
func seedCostRecords(t *testing.T, wsDir string, records []costRecordInput) {
	t.Helper()
	store := cost.NewStore(wsDir)
	if err := store.Open(); err != nil {
		t.Fatalf("failed to open cost store: %v", err)
	}
	defer func() { _ = store.Close() }()

	for _, r := range records {
		if _, err := store.Record(r.AgentID, r.TeamID, r.Model, r.InputTokens, r.OutputTokens, r.CostUSD); err != nil {
			t.Fatalf("failed to record cost: %v", err)
		}
	}
}

type costRecordInput struct {
	AgentID      string
	TeamID       string
	Model        string
	InputTokens  int64
	OutputTokens int64
	CostUSD      float64
}

func TestCostDashboardNoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, _, err = executeIntegrationCmd("cost", "dashboard")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
}

func TestCostDashboardEmpty(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	stdout, _, err := executeIntegrationCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("cost dashboard returned error: %v", err)
	}
	if !strings.Contains(stdout, "COST DASHBOARD") {
		t.Errorf("expected 'COST DASHBOARD' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "WORKSPACE TOTALS") {
		t.Errorf("expected 'WORKSPACE TOTALS' section, got: %s", stdout)
	}
}

func TestCostDashboardWithRecords(t *testing.T) {
	wsDir, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	seedCostRecords(t, wsDir, []costRecordInput{
		{AgentID: "engineer-01", TeamID: "engineering", Model: "claude-sonnet", InputTokens: 1000, OutputTokens: 500, CostUSD: 0.015},
		{AgentID: "engineer-02", TeamID: "engineering", Model: "claude-opus", InputTokens: 2000, OutputTokens: 1000, CostUSD: 0.045},
		{AgentID: "qa-01", TeamID: "qa", Model: "claude-sonnet", InputTokens: 500, OutputTokens: 250, CostUSD: 0.0075},
	})

	stdout, _, err := executeIntegrationCmd("cost", "dashboard")
	if err != nil {
		t.Fatalf("cost dashboard returned error: %v", err)
	}
	// Check all sections are present
	if !strings.Contains(stdout, "COST DASHBOARD") {
		t.Errorf("expected 'COST DASHBOARD' header, got: %s", stdout)
	}
	if !strings.Contains(stdout, "WORKSPACE TOTALS") {
		t.Errorf("expected 'WORKSPACE TOTALS' section, got: %s", stdout)
	}
	if !strings.Contains(stdout, "BY AGENT") {
		t.Errorf("expected 'BY AGENT' section, got: %s", stdout)
	}
	if !strings.Contains(stdout, "BY TEAM") {
		t.Errorf("expected 'BY TEAM' section, got: %s", stdout)
	}
	if !strings.Contains(stdout, "BY MODEL") {
		t.Errorf("expected 'BY MODEL' section, got: %s", stdout)
	}
	// Check agent data
	if !strings.Contains(stdout, "engineer-01") {
		t.Errorf("expected 'engineer-01' in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "engineer-02") {
		t.Errorf("expected 'engineer-02' in output, got: %s", stdout)
	}
	// Check percentages are shown
	if !strings.Contains(stdout, "%") {
		t.Errorf("expected percentage values, got: %s", stdout)
	}
}

// Note: Process command tests are in process_test.go
