package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/workspace"
	"github.com/spf13/pflag"
)

// setupLogsWorkspace creates a temporary bc workspace, changes into it,
// and returns the root dir plus a cleanup function.
func setupLogsWorkspace(t *testing.T) (string, func()) {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	ws, err := workspace.Init(tmpDir)
	if err != nil {
		t.Fatalf("failed to init workspace: %v", err)
	}
	if err := ws.EnsureDirs(); err != nil {
		t.Fatalf("failed to ensure dirs: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	return tmpDir, func() { _ = os.Chdir(origDir) }
}

// seedLogsEvents writes events to the workspace events.jsonl file.
func seedLogsEvents(t *testing.T, wsDir string, evts []events.Event) {
	t.Helper()
	evtLog := events.NewLog(filepath.Join(wsDir, ".bc", "events.jsonl"))
	for _, ev := range evts {
		if err := evtLog.Append(ev); err != nil {
			t.Fatalf("failed to append event: %v", err)
		}
	}
}

// runLogsCmd executes the logs command capturing stdout. It resets all logs
// flags to defaults before execution to avoid leaking between tests.
func runLogsCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Reset all logs flags to defaults (both variable and cobra Changed state)
	logsAgent = ""
	logsTail = 0
	logsType = ""
	logsSince = ""
	logsFull = false
	logsCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })

	origStdout := os.Stdout
	r, w, pipeErr := os.Pipe()
	if pipeErr != nil {
		t.Fatalf("failed to create pipe: %v", pipeErr)
	}
	os.Stdout = w

	rootCmd.SetOut(w)
	rootCmd.SetErr(w)
	rootCmd.SetArgs(args)

	// Reset persistent flags too
	_ = rootCmd.PersistentFlags().Set("json", "false")

	err := rootCmd.Execute()

	_ = w.Close()
	var buf [64 * 1024]byte
	n, _ := r.Read(buf[:])
	os.Stdout = origStdout

	return string(buf[:n]), err
}

// --- truncateMessage tests ---

func TestTruncateMessage(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"short message unchanged", "hello", 80, "hello"},
		{"exact length unchanged", "abcde", 5, "abcde"},
		{"long message truncated", "this is a long message that exceeds the limit", 20, "this is a long me..."},
		{"very short limit", "hello world", 3, "hel"},
		{"empty string", "", 80, ""},
		{"limit of 4", "abcdefgh", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateMessage(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateMessage(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// --- parseSinceDuration tests ---

func TestParseSinceDuration(t *testing.T) {
	t.Run("valid 1h", func(t *testing.T) {
		cutoff, err := parseSinceDuration("1h")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// cutoff should be approximately 1 hour ago
		diff := time.Since(cutoff)
		if diff < 59*time.Minute || diff > 61*time.Minute {
			t.Errorf("expected ~1h ago, got %v ago", diff)
		}
	})

	t.Run("valid 30m", func(t *testing.T) {
		cutoff, err := parseSinceDuration("30m")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		diff := time.Since(cutoff)
		if diff < 29*time.Minute || diff > 31*time.Minute {
			t.Errorf("expected ~30m ago, got %v ago", diff)
		}
	})

	t.Run("valid 2h30m", func(t *testing.T) {
		cutoff, err := parseSinceDuration("2h30m")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		diff := time.Since(cutoff)
		expected := 2*time.Hour + 30*time.Minute
		if diff < expected-time.Minute || diff > expected+time.Minute {
			t.Errorf("expected ~2h30m ago, got %v ago", diff)
		}
	})

	t.Run("invalid duration", func(t *testing.T) {
		_, err := parseSinceDuration("bogus")
		if err == nil {
			t.Fatal("expected error for invalid duration")
		}
		if !strings.Contains(err.Error(), "invalid duration") {
			t.Errorf("expected 'invalid duration' in error, got: %v", err)
		}
	})
}

// --- Integration tests for --type flag ---

func TestLogs_TypeFilter(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentSpawned, Agent: "coord", Message: "spawned"},
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-01", Message: "working on auth"},
		{Timestamp: time.Now(), Type: events.WorkCompleted, Agent: "eng-01", Message: "auth done"},
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-02", Message: "working on tests"},
	})

	stdout, err := runLogsCmd(t, "logs", "--type", "agent.report")
	if err != nil {
		t.Fatalf("logs --type failed: %v", err)
	}

	if !strings.Contains(stdout, "agent.report") {
		t.Errorf("expected agent.report events, got: %s", stdout)
	}
	if strings.Contains(stdout, "agent.spawned") {
		t.Errorf("should not contain agent.spawned events, got: %s", stdout)
	}
	if strings.Contains(stdout, "work.completed") {
		t.Errorf("should not contain work.completed events, got: %s", stdout)
	}
}

func TestLogs_TypeFilter_NoMatch(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentSpawned, Agent: "coord", Message: "spawned"},
	})

	stdout, err := runLogsCmd(t, "logs", "--type", "work.failed")
	if err != nil {
		t.Fatalf("logs --type failed: %v", err)
	}

	if !strings.Contains(stdout, "No events found") {
		t.Errorf("expected 'No events found', got: %s", stdout)
	}
}

// --- Integration tests for --since flag ---

func TestLogs_SinceFilter(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now().Add(-2 * time.Hour), Type: events.AgentSpawned, Agent: "coord", Message: "old event"},
		{Timestamp: time.Now().Add(-10 * time.Minute), Type: events.AgentReport, Agent: "eng-01", Message: "recent event"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Type: events.WorkCompleted, Agent: "eng-01", Message: "very recent"},
	})

	stdout, err := runLogsCmd(t, "logs", "--since", "30m")
	if err != nil {
		t.Fatalf("logs --since failed: %v", err)
	}

	if strings.Contains(stdout, "old event") {
		t.Errorf("should not contain old event from 2h ago, got: %s", stdout)
	}
	if !strings.Contains(stdout, "recent event") {
		t.Errorf("should contain recent event from 10m ago, got: %s", stdout)
	}
	if !strings.Contains(stdout, "very recent") {
		t.Errorf("should contain very recent event, got: %s", stdout)
	}
}

func TestLogs_SinceFilter_InvalidDuration(t *testing.T) {
	_, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	_, err := runLogsCmd(t, "logs", "--since", "notaduration")
	if err == nil {
		t.Fatal("expected error for invalid --since duration")
	}
	if !strings.Contains(err.Error(), "invalid duration") {
		t.Errorf("expected 'invalid duration' in error, got: %v", err)
	}
}

// --- Integration tests for message truncation ---

func TestLogs_MessageTruncation(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	longMsg := strings.Repeat("x", 120)
	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-01", Message: longMsg},
	})

	stdout, err := runLogsCmd(t, "logs")
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}

	// Should be truncated - the full 120-char message should not appear
	if strings.Contains(stdout, longMsg) {
		t.Error("long message should be truncated by default")
	}
	if !strings.Contains(stdout, "...") {
		t.Errorf("truncated message should end with '...', got: %s", stdout)
	}
}

func TestLogs_FullFlag_NoTruncation(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	longMsg := strings.Repeat("y", 120)
	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-01", Message: longMsg},
	})

	stdout, err := runLogsCmd(t, "logs", "--full")
	if err != nil {
		t.Fatalf("logs --full failed: %v", err)
	}

	if !strings.Contains(stdout, longMsg) {
		t.Errorf("--full should show complete message, got: %s", stdout)
	}
}

func TestLogs_ShortMessage_NoTruncation(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	shortMsg := "short message"
	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-01", Message: shortMsg},
	})

	stdout, err := runLogsCmd(t, "logs")
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}

	if !strings.Contains(stdout, shortMsg) {
		t.Errorf("short message should appear in full, got: %s", stdout)
	}
}

// --- Composable filters tests ---

func TestLogs_AgentAndType(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-01", Message: "eng-01 report"},
		{Timestamp: time.Now(), Type: events.AgentSpawned, Agent: "eng-01", Message: "eng-01 spawned"},
		{Timestamp: time.Now(), Type: events.AgentReport, Agent: "eng-02", Message: "eng-02 report"},
	})

	stdout, err := runLogsCmd(t, "logs", "--agent", "eng-01", "--type", "agent.report")
	if err != nil {
		t.Fatalf("logs --agent --type failed: %v", err)
	}

	if !strings.Contains(stdout, "eng-01 report") {
		t.Errorf("expected eng-01 report, got: %s", stdout)
	}
	if strings.Contains(stdout, "eng-01 spawned") {
		t.Errorf("should not contain eng-01 spawned event, got: %s", stdout)
	}
	if strings.Contains(stdout, "eng-02 report") {
		t.Errorf("should not contain eng-02 events, got: %s", stdout)
	}
}

func TestLogs_AgentAndTypeAndSince(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now().Add(-2 * time.Hour), Type: events.AgentReport, Agent: "eng-01", Message: "old report"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Type: events.AgentReport, Agent: "eng-01", Message: "recent report"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Type: events.AgentSpawned, Agent: "eng-01", Message: "recent spawn"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Type: events.AgentReport, Agent: "eng-02", Message: "other agent"},
	})

	stdout, err := runLogsCmd(t, "logs", "--agent", "eng-01", "--type", "agent.report", "--since", "30m")
	if err != nil {
		t.Fatalf("logs --agent --type --since failed: %v", err)
	}

	if !strings.Contains(stdout, "recent report") {
		t.Errorf("expected recent report, got: %s", stdout)
	}
	if strings.Contains(stdout, "old report") {
		t.Errorf("should not contain old report, got: %s", stdout)
	}
	if strings.Contains(stdout, "recent spawn") {
		t.Errorf("should not contain spawn events, got: %s", stdout)
	}
	if strings.Contains(stdout, "other agent") {
		t.Errorf("should not contain other agent, got: %s", stdout)
	}
}

func TestLogs_AllFiltersComposed(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	// Seed 10 recent agent.report events from eng-01
	for i := 0; i < 10; i++ {
		seedLogsEvents(t, wsDir, []events.Event{
			{Timestamp: time.Now().Add(-time.Duration(10-i) * time.Minute), Type: events.AgentReport, Agent: "eng-01", Message: "report " + string(rune('A'+i))},
		})
	}
	// Seed some noise
	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now().Add(-3 * time.Hour), Type: events.AgentReport, Agent: "eng-01", Message: "old report"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Type: events.AgentSpawned, Agent: "eng-01", Message: "spawn"},
		{Timestamp: time.Now().Add(-5 * time.Minute), Type: events.AgentReport, Agent: "eng-02", Message: "other"},
	})

	stdout, err := runLogsCmd(t, "logs", "--agent", "eng-01", "--type", "agent.report", "--since", "30m", "--tail", "3")
	if err != nil {
		t.Fatalf("logs with all filters failed: %v", err)
	}

	// Should not contain old, spawn, or other agent events
	if strings.Contains(stdout, "old report") {
		t.Errorf("should not contain old report")
	}
	if strings.Contains(stdout, "spawn") {
		t.Errorf("should not contain spawn events")
	}
	if strings.Contains(stdout, "other") {
		t.Errorf("should not contain other agent")
	}

	// Count the number of event lines
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	eventLines := 0
	for _, l := range lines {
		if strings.Contains(l, "agent.report") {
			eventLines++
		}
	}
	if eventLines != 3 {
		t.Errorf("expected 3 event lines with --tail 3, got %d\noutput: %s", eventLines, stdout)
	}
}

// --- Edge cases ---

func TestLogs_EmptyLog(t *testing.T) {
	_, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	stdout, err := runLogsCmd(t, "logs")
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}

	if !strings.Contains(stdout, "No events found") {
		t.Errorf("expected 'No events found', got: %s", stdout)
	}
}

func TestLogs_TailOnly(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	for i := 0; i < 10; i++ {
		seedLogsEvents(t, wsDir, []events.Event{
			{Timestamp: time.Now().Add(-time.Duration(10-i) * time.Minute), Type: events.AgentReport, Agent: "eng-01", Message: "event " + string(rune('A'+i))},
		})
	}

	stdout, err := runLogsCmd(t, "logs", "--tail", "2")
	if err != nil {
		t.Fatalf("logs --tail failed: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
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

func TestLogs_SinceFilter_AllOld(t *testing.T) {
	wsDir, cleanup := setupLogsWorkspace(t)
	defer cleanup()

	seedLogsEvents(t, wsDir, []events.Event{
		{Timestamp: time.Now().Add(-5 * time.Hour), Type: events.AgentReport, Agent: "eng-01", Message: "old"},
		{Timestamp: time.Now().Add(-4 * time.Hour), Type: events.AgentReport, Agent: "eng-02", Message: "also old"},
	})

	stdout, err := runLogsCmd(t, "logs", "--since", "1h")
	if err != nil {
		t.Fatalf("logs --since failed: %v", err)
	}

	if !strings.Contains(stdout, "No events found") {
		t.Errorf("expected 'No events found' when all events are old, got: %s", stdout)
	}
}

func TestLogs_TailRejectsNegativeAndZero(t *testing.T) {
	tests := []struct {
		name string
		val  string
	}{
		{"zero", "0"},
		{"negative", "-5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := runLogsCmd(t, "logs", "--tail", tt.val)
			if err == nil {
				t.Fatal("expected error for --tail " + tt.val + ", got nil")
			}
			if !strings.Contains(err.Error(), "tail must be a positive number") {
				t.Errorf("expected 'tail must be a positive number', got: %v", err)
			}
		})
	}
}
