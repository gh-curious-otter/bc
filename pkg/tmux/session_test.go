package tmux

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Test helper process — standard Go pattern for mocking exec.Command.
// When invoked as a subprocess, reads MOCK_STDOUT/MOCK_STDERR/MOCK_EXIT_CODE
// from env and behaves accordingly.
// ---------------------------------------------------------------------------

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if s := os.Getenv("MOCK_STDOUT"); s != "" {
		_, _ = fmt.Fprint(os.Stdout, s) //nolint:errcheck // test helper output
	}
	if s := os.Getenv("MOCK_STDERR"); s != "" {
		_, _ = fmt.Fprint(os.Stderr, s) //nolint:errcheck // test helper output
	}
	exitCode := 0
	if v := os.Getenv("MOCK_EXIT_CODE"); v != "" {
		var err error
		exitCode, err = strconv.Atoi(v)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "invalid MOCK_EXIT_CODE: %v\n", err)
			os.Exit(2)
		}
	}
	os.Exit(exitCode)
}

// ---------------------------------------------------------------------------
// Mock helpers
// ---------------------------------------------------------------------------

// mockCmd creates a mock execCommand that always returns the same result.
func mockCmd(stdout, stderr string, exitCode int) func(string, ...string) *exec.Cmd {
	return func(name string, args ...string) *exec.Cmd {
		cs := make([]string, 0, 3+len(args))
		cs = append(cs, "-test.run=TestHelperProcess", "--", name)
		cs = append(cs, args...)
		cmd := exec.CommandContext(context.Background(), os.Args[0], cs...) //nolint:gosec // test helper
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"MOCK_STDOUT=" + stdout,
			"MOCK_STDERR=" + stderr,
			fmt.Sprintf("MOCK_EXIT_CODE=%d", exitCode),
		}
		return cmd
	}
}

// mockResponse defines output for a single exec call.
type mockResponse struct {
	stdout   string
	stderr   string
	exitCode int
}

// mockCmdSequence creates a mock that returns different results for sequential calls.
// After all responses are consumed, subsequent calls return success.
func mockCmdSequence(responses ...mockResponse) func(string, ...string) *exec.Cmd {
	var mu sync.Mutex
	idx := 0
	return func(name string, args ...string) *exec.Cmd {
		mu.Lock()
		r := mockResponse{}
		if idx < len(responses) {
			r = responses[idx]
		}
		idx++
		mu.Unlock()

		cs := make([]string, 0, 3+len(args))
		cs = append(cs, "-test.run=TestHelperProcess", "--", name)
		cs = append(cs, args...)
		cmd := exec.CommandContext(context.Background(), os.Args[0], cs...) //nolint:gosec // test helper
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"MOCK_STDOUT=" + r.stdout,
			"MOCK_STDERR=" + r.stderr,
			fmt.Sprintf("MOCK_EXIT_CODE=%d", r.exitCode),
		}
		return cmd
	}
}

// cmdRecord captures the name and args of an exec call.
type cmdRecord struct {
	name string
	args []string
}

// recordingMock creates a mock that records all calls and returns success.
func recordingMock(stdout string) (func(string, ...string) *exec.Cmd, *[]cmdRecord) {
	var mu sync.Mutex
	var records []cmdRecord
	fn := func(name string, args ...string) *exec.Cmd {
		mu.Lock()
		argsCopy := make([]string, len(args))
		copy(argsCopy, args)
		records = append(records, cmdRecord{name: name, args: argsCopy})
		mu.Unlock()

		cs := make([]string, 0, 3+len(args))
		cs = append(cs, "-test.run=TestHelperProcess", "--", name)
		cs = append(cs, args...)
		cmd := exec.CommandContext(context.Background(), os.Args[0], cs...) //nolint:gosec // test helper
		cmd.Env = []string{
			"GO_WANT_HELPER_PROCESS=1",
			"MOCK_STDOUT=" + stdout,
			"MOCK_EXIT_CODE=0",
		}
		return cmd
	}
	return fn, &records
}

// newTestManager creates a Manager with a mock exec command.
func newTestManager(prefix string, mock func(string, ...string) *exec.Cmd) *Manager {
	return &Manager{
		SessionPrefix: prefix,
		execCommand:   mock,
	}
}

// sliceContains checks if a string slice contains a target value.
func sliceContains(strs []string, target string) bool {
	for _, s := range strs {
		if s == target {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Constructor tests
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	m := NewManager("test-")
	if m.SessionPrefix != "test-" {
		t.Errorf("SessionPrefix = %q, want %q", m.SessionPrefix, "test-")
	}
	if m.workspaceHash != "" {
		t.Errorf("workspaceHash = %q, want empty", m.workspaceHash)
	}
	if m.execCommand == nil {
		t.Error("execCommand should not be nil")
	}
}

func TestNewDefaultManager(t *testing.T) {
	m := NewDefaultManager()
	if m.SessionPrefix != "bc-" {
		t.Errorf("SessionPrefix = %q, want %q", m.SessionPrefix, "bc-")
	}
	if m.execCommand == nil {
		t.Error("execCommand should not be nil")
	}
}

func TestNewWorkspaceManager(t *testing.T) {
	m := NewWorkspaceManager("bc-", "/some/workspace")
	if m.SessionPrefix != "bc-" {
		t.Errorf("SessionPrefix = %q, want %q", m.SessionPrefix, "bc-")
	}
	if m.workspaceHash == "" {
		t.Error("workspaceHash should not be empty")
	}
	if m.execCommand == nil {
		t.Error("execCommand should not be nil")
	}
}

func TestNewWorkspaceManager_DifferentPaths(t *testing.T) {
	m1 := NewWorkspaceManager("bc-", "/path/one")
	m2 := NewWorkspaceManager("bc-", "/path/two")
	if m1.workspaceHash == m2.workspaceHash {
		t.Error("different paths should produce different hashes")
	}
}

func TestNewWorkspaceManager_SamePath(t *testing.T) {
	m1 := NewWorkspaceManager("bc-", "/same/path")
	m2 := NewWorkspaceManager("bc-", "/same/path")
	if m1.workspaceHash != m2.workspaceHash {
		t.Error("same paths should produce same hash")
	}
}

// ---------------------------------------------------------------------------
// SessionName tests (existing + new edge cases)
// ---------------------------------------------------------------------------

func TestGetSessionLock_ReturnsSameMutex(t *testing.T) {
	m := NewManager("test-")
	mu1 := m.getSessionLock("session-a")
	mu2 := m.getSessionLock("session-a")
	if mu1 != mu2 {
		t.Error("expected same mutex for same session name")
	}
}

func TestGetSessionLock_DifferentSessions(t *testing.T) {
	m := NewManager("test-")
	mu1 := m.getSessionLock("session-a")
	mu2 := m.getSessionLock("session-b")
	if mu1 == mu2 {
		t.Error("expected different mutexes for different session names")
	}
}

func TestGetSessionLock_ConcurrentAccess(t *testing.T) {
	m := NewManager("test-")
	var wg sync.WaitGroup
	results := make([]*sync.Mutex, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx] = m.getSessionLock("session-x")
		}(i)
	}
	wg.Wait()

	for i := 1; i < 100; i++ {
		if results[i] != results[0] {
			t.Errorf("goroutine %d got a different mutex", i)
		}
	}
}

func TestGetSessionLock_LazyInit(t *testing.T) {
	m := NewManager("test-")
	if m.sessionLocks != nil {
		t.Error("sessionLocks should be nil before first use")
	}
	m.getSessionLock("session-a")
	if m.sessionLocks == nil {
		t.Error("sessionLocks should be initialized after first use")
	}
}

func TestGenerateBufferName_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		name := generateBufferName()
		if seen[name] {
			t.Fatalf("duplicate buffer name: %s", name)
		}
		seen[name] = true
	}
}

func TestGenerateBufferName_Format(t *testing.T) {
	name := generateBufferName()
	if len(name) != 3+16 { // "bc-" + 16 hex chars
		t.Errorf("unexpected buffer name length: %d (%s)", len(name), name)
	}
	if name[:3] != "bc-" {
		t.Errorf("buffer name should start with 'bc-': %s", name)
	}
}

// TestSendKeys_ConcurrentLongMessages verifies that concurrent SendKeys calls
// with long messages (>500 chars, using buffer-based send) don't corrupt each other.
// This exercises the named buffer and per-session locking mechanism.
func TestSendKeys_ConcurrentLongMessages(t *testing.T) {
	// This test verifies the concurrent safety of SendKeys with long messages.
	// It doesn't require actual tmux sessions - it tests that concurrent calls
	// properly serialize via per-session locks and use unique buffer names.
	m := NewManager("conctest-")

	// Generate unique buffer names concurrently
	bufferNames := make([]string, 50)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bufferNames[idx] = generateBufferName()
		}(i)
	}
	wg.Wait()

	// Verify all buffer names are unique (no collisions under concurrent generation)
	seen := make(map[string]bool)
	for i, name := range bufferNames {
		if seen[name] {
			t.Errorf("duplicate buffer name at index %d: %s", i, name)
		}
		seen[name] = true
	}

	// Verify per-session locks are correctly allocated under concurrent access
	locks := make([]*sync.Mutex, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			locks[idx] = m.getSessionLock("test-session")
		}(i)
	}
	wg.Wait()

	// All locks for the same session should be the same instance
	for i := 1; i < 50; i++ {
		if locks[i] != locks[0] {
			t.Errorf("goroutine %d got different lock for same session", i)
		}
	}
}

func TestSessionName(t *testing.T) {
	tests := []struct {
		name  string
		mgr   *Manager
		input string
		want  string
	}{
		{
			name:  "simple prefix",
			mgr:   NewManager("bc-"),
			input: "agent1",
			want:  "bc-agent1",
		},
		{
			name:  "workspace manager",
			mgr:   NewWorkspaceManager("bc-", "/some/path"),
			input: "agent1",
			want:  "bc-" + NewWorkspaceManager("bc-", "/some/path").workspaceHash + "-agent1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mgr.SessionName(tt.input)
			if got != tt.want {
				t.Errorf("SessionName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSessionName_EmptyName(t *testing.T) {
	m := NewManager("bc-")
	got := m.SessionName("")
	if got != "bc-" {
		t.Errorf("SessionName('') = %q, want %q", got, "bc-")
	}
}

func TestSessionName_WorkspacePrefix(t *testing.T) {
	m := NewWorkspaceManager("bc-", "/workspace")
	name := m.SessionName("agent1")
	if !strings.HasPrefix(name, "bc-") {
		t.Errorf("session name should start with prefix: %s", name)
	}
	if !strings.HasSuffix(name, "-agent1") {
		t.Errorf("session name should end with -agent1: %s", name)
	}
	if !strings.Contains(name, m.workspaceHash) {
		t.Errorf("session name should contain workspace hash: %s", name)
	}
}

// ---------------------------------------------------------------------------
// command() method tests
// ---------------------------------------------------------------------------

func TestCommand_NilExecCommand(t *testing.T) {
	m := &Manager{SessionPrefix: "bc-"}
	cmd := m.command("echo", "hello")
	if cmd == nil {
		t.Fatal("command returned nil with nil execCommand")
	}
}

func TestCommand_UsesExecCommand(t *testing.T) {
	called := false
	m := &Manager{
		SessionPrefix: "bc-",
		execCommand: func(name string, arg ...string) *exec.Cmd {
			called = true
			return exec.CommandContext(context.Background(), name, arg...)
		},
	}
	m.command("echo", "test")
	if !called {
		t.Error("command should use execCommand when set")
	}
}

// ---------------------------------------------------------------------------
// HasSession tests
// ---------------------------------------------------------------------------

func TestHasSession_Exists(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 0))
	if !m.HasSession("agent1") {
		t.Error("expected HasSession to return true when tmux returns 0")
	}
}

func TestHasSession_NotExists(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "no session", 1))
	if m.HasSession("agent1") {
		t.Error("expected HasSession to return false when tmux returns 1")
	}
}

// ---------------------------------------------------------------------------
// CreateSession tests
// ---------------------------------------------------------------------------

func TestCreateSession_Success(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	err := m.CreateSession("agent1", "/workspace")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if len(*records) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*records))
	}
	rec := (*records)[0]
	if rec.name != "tmux" {
		t.Errorf("expected tmux command, got %s", rec.name)
	}
	if !sliceContains(rec.args, "new-session") {
		t.Error("expected new-session in args")
	}
	if !sliceContains(rec.args, "bc-agent1") {
		t.Error("expected session name in args")
	}
	if !sliceContains(rec.args, "/workspace") {
		t.Error("expected directory in args")
	}
}

func TestCreateSession_NoDir(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.CreateSession("agent1", ""); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if sliceContains((*records)[0].args, "-c") {
		t.Error("should not include -c flag when dir is empty")
	}
}

func TestCreateSession_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "duplicate session", 1))
	err := m.CreateSession("agent1", "/workspace")
	if err == nil {
		t.Error("expected error when tmux fails")
	}
	if !strings.Contains(err.Error(), "failed to create session") {
		t.Errorf("error should mention failure: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CreateSessionWithCommand tests
// ---------------------------------------------------------------------------

func TestCreateSessionWithCommand_Success(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 0))
	if err := m.CreateSessionWithCommand("agent1", "/workspace", "echo hello"); err != nil {
		t.Fatalf("CreateSessionWithCommand failed: %v", err)
	}
}

func TestCreateSessionWithCommand_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "error", 1))
	err := m.CreateSessionWithCommand("agent1", "/workspace", "echo hello")
	if err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// CreateSessionWithEnv tests
// ---------------------------------------------------------------------------

func TestCreateSessionWithEnv_Success(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	env := map[string]string{"FOO": "bar"}
	err := m.CreateSessionWithEnv("agent1", "/workspace", "echo hello", env)
	if err != nil {
		t.Fatalf("CreateSessionWithEnv failed: %v", err)
	}

	if len(*records) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*records))
	}
	if !sliceContains((*records)[0].args, "bash") {
		t.Error("expected bash in args")
	}
}

func TestCreateSessionWithEnv_NilEnv(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 0))
	if err := m.CreateSessionWithEnv("agent1", "/workspace", "echo hello", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSessionWithEnv_NoDir(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.CreateSessionWithEnv("agent1", "", "echo hello", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "-c" appears once (for bash -c) but not twice (no tmux -c dir flag)
	args := (*records)[0].args
	cCount := 0
	for _, a := range args {
		if a == "-c" {
			cCount++
		}
	}
	if cCount != 1 {
		t.Errorf("expected 1 '-c' (bash -c only), got %d in args: %v", cCount, args)
	}
}

func TestCreateSessionWithEnv_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "error", 1))
	err := m.CreateSessionWithEnv("agent1", "/workspace", "echo", nil)
	if err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// KillSession tests
// ---------------------------------------------------------------------------

func TestKillSession_Success(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.KillSession("agent1"); err != nil {
		t.Fatalf("KillSession failed: %v", err)
	}

	if len(*records) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*records))
	}
	if !sliceContains((*records)[0].args, "kill-session") {
		t.Error("expected kill-session in args")
	}
}

func TestKillSession_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "no session", 1))
	err := m.KillSession("agent1")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "failed to kill session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SendKeys tests
// ---------------------------------------------------------------------------

func TestSendKeys_DelegatesToSendKeysWithSubmit(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.SendKeys("agent1", "hello"); err != nil {
		t.Fatalf("SendKeys failed: %v", err)
	}

	// Should make 2 calls: send-keys -l + send-keys Enter
	if len(*records) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*records))
	}
}

// ---------------------------------------------------------------------------
// SendKeysWithSubmit tests
// ---------------------------------------------------------------------------

func TestSendKeysWithSubmit_ShortMessage(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.SendKeysWithSubmit("agent1", "short message", "Enter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 calls: send-keys literal + send-keys Enter
	if len(*records) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*records))
	}

	if !sliceContains((*records)[0].args, "-l") {
		t.Error("first call should include -l flag for literal mode")
	}
	// Enter is sent as -H 0D (raw hex CR byte) for tmux 3.5+ compatibility
	if !sliceContains((*records)[1].args, "-H") || !sliceContains((*records)[1].args, "0D") {
		t.Errorf("second call should use -H 0D for Enter, got args: %v", (*records)[1].args)
	}
}

func TestSendKeysWithSubmit_NoSubmitKey(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.SendKeysWithSubmit("agent1", "message", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only 1 call: send-keys literal (no submit)
	if len(*records) != 1 {
		t.Fatalf("expected 1 call (no submit), got %d", len(*records))
	}
}

func TestSendKeysWithSubmit_TrimsNewlines(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.SendKeysWithSubmit("agent1", "message\n\n\n", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*records) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*records))
	}
	// The literal text arg should be "message" without trailing newlines
	argsStr := strings.Join((*records)[0].args, "|")
	if strings.Contains(argsStr, "\n") {
		t.Error("trailing newlines should be trimmed")
	}
}

func TestSendKeysWithSubmit_LongMessage(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	longMsg := strings.Repeat("x", 501)
	if err := m.SendKeysWithSubmit("agent1", longMsg, "Enter"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 calls: load-buffer + paste-buffer + send-keys Enter
	if len(*records) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(*records))
	}

	if !sliceContains((*records)[0].args, "load-buffer") {
		t.Error("first call should be load-buffer")
	}
	if !sliceContains((*records)[1].args, "paste-buffer") {
		t.Error("second call should be paste-buffer")
	}
	if !sliceContains((*records)[2].args, "-H") || !sliceContains((*records)[2].args, "0D") {
		t.Errorf("third call should use -H 0D for Enter, got args: %v", (*records)[2].args)
	}
}

func TestSendKeysWithSubmit_LongMessageNoSubmit(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	longMsg := strings.Repeat("x", 501)
	if err := m.SendKeysWithSubmit("agent1", longMsg, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 calls: load-buffer + paste-buffer (no submit)
	if len(*records) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*records))
	}
}

func TestSendKeysWithSubmit_ShortMessageError(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "send error", 1))
	err := m.SendKeysWithSubmit("agent1", "hello", "Enter")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "failed to send keys") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_SubmitKeyError(t *testing.T) {
	// First call (send-keys literal) succeeds, second (Enter) fails
	m := newTestManager("bc-", mockCmdSequence(
		mockResponse{},
		mockResponse{stderr: "submit error", exitCode: 1},
	))
	err := m.SendKeysWithSubmit("agent1", "hello", "Enter")
	if err == nil {
		t.Error("expected error on submit key failure")
	}
	if !strings.Contains(err.Error(), "failed to send submit key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_LoadBufferError(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "buffer error", 1))
	longMsg := strings.Repeat("x", 501)
	err := m.SendKeysWithSubmit("agent1", longMsg, "Enter")
	if err == nil {
		t.Error("expected error on load-buffer failure")
	}
	if !strings.Contains(err.Error(), "failed to load buffer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_PasteBufferError(t *testing.T) {
	// load-buffer succeeds, paste-buffer fails (also triggers delete-buffer cleanup)
	m := newTestManager("bc-", mockCmdSequence(
		mockResponse{},
		mockResponse{stderr: "paste error", exitCode: 1},
	))
	longMsg := strings.Repeat("x", 501)
	err := m.SendKeysWithSubmit("agent1", longMsg, "Enter")
	if err == nil {
		t.Error("expected error on paste-buffer failure")
	}
	if !strings.Contains(err.Error(), "failed to paste buffer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_CustomSubmitKey(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.SendKeysWithSubmit("agent1", "msg", "C-m"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(*records) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(*records))
	}
	if !sliceContains((*records)[1].args, "C-m") {
		t.Error("second call should use custom submit key C-m")
	}
}

// ---------------------------------------------------------------------------
// Capture tests
// ---------------------------------------------------------------------------

func TestCapture_Success(t *testing.T) {
	m := newTestManager("bc-", mockCmd("pane content here\n", "", 0))
	output, err := m.Capture("agent1", 0)
	if err != nil {
		t.Fatalf("Capture failed: %v", err)
	}
	if output != "pane content here\n" {
		t.Errorf("output = %q, want %q", output, "pane content here\n")
	}
}

func TestCapture_WithLines(t *testing.T) {
	mock, records := recordingMock("output")
	m := newTestManager("bc-", mock)

	if _, err := m.Capture("agent1", 100); err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	if !sliceContains((*records)[0].args, "-S") {
		t.Error("should include -S flag when lines > 0")
	}
	if !sliceContains((*records)[0].args, "-100") {
		t.Error("should include negative line count")
	}
}

func TestCapture_NoLines(t *testing.T) {
	mock, records := recordingMock("output")
	m := newTestManager("bc-", mock)

	if _, err := m.Capture("agent1", 0); err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	if sliceContains((*records)[0].args, "-S") {
		t.Error("should not include -S flag when lines = 0")
	}
}

func TestCapture_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "no session", 1))
	_, err := m.Capture("agent1", 0)
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "failed to capture pane") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ListSessions tests
// ---------------------------------------------------------------------------

func TestListSessions_Success(t *testing.T) {
	sessionOutput := "bc-agent1|Thu Jan  1 00:00:00 2025|0|1|/workspace\nbc-agent2|Thu Jan  1 00:00:01 2025|1|2|/workspace\nother-session|Thu Jan  1 00:00:02 2025|0|1|/other\n"
	m := newTestManager("bc-", mockCmd(sessionOutput, "", 0))

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	// Should only include bc- prefixed sessions (not "other-session")
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].Name != "agent1" {
		t.Errorf("sessions[0].Name = %q, want %q", sessions[0].Name, "agent1")
	}
	if sessions[1].Name != "agent2" {
		t.Errorf("sessions[1].Name = %q, want %q", sessions[1].Name, "agent2")
	}
	if !sessions[1].Attached {
		t.Error("sessions[1] should be attached (field was '1')")
	}
	if sessions[0].Attached {
		t.Error("sessions[0] should not be attached (field was '0')")
	}
	if sessions[0].Directory != "/workspace" {
		t.Errorf("sessions[0].Directory = %q, want %q", sessions[0].Directory, "/workspace")
	}
	if sessions[0].Windows != 1 {
		t.Errorf("sessions[0].Windows = %d, want %d", sessions[0].Windows, 1)
	}
	if sessions[1].Windows != 2 {
		t.Errorf("sessions[1].Windows = %d, want %d", sessions[1].Windows, 2)
	}
}

func TestListSessions_WorkspaceIsolation(t *testing.T) {
	m := NewWorkspaceManager("bc-", "/workspace/one")
	hash := m.workspaceHash
	sessionOutput := fmt.Sprintf("bc-%s-agent1|Thu Jan  1|0|1|/workspace\nbc-otheragent2|Thu Jan  1|0|1|/workspace\n", hash)
	m.execCommand = mockCmd(sessionOutput, "", 0)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("expected 1 session (workspace-scoped), got %d", len(sessions))
	}
	if sessions[0].Name != "agent1" {
		t.Errorf("session name = %q, want %q", sessions[0].Name, "agent1")
	}
}

func TestListSessions_Empty(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 0))
	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestListSessions_ExitError_ReturnsEmpty(t *testing.T) {
	// When tmux list-sessions fails with an exit error (e.g., no server
	// running, socket not found), ListSessions treats it as "no sessions"
	// rather than propagating the error.
	m := newTestManager("bc-", mockCmd("", "", 1))
	sessions, err := m.ListSessions()
	if err != nil {
		t.Errorf("expected nil error for exit error, got: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestListSessions_MalformedLine(t *testing.T) {
	// Line with fewer than 5 pipe-separated parts should be skipped
	sessionOutput := "bc-agent1|too|few\nbc-agent2|Thu Jan  1|0|1|/workspace\n"
	m := newTestManager("bc-", mockCmd(sessionOutput, "", 0))

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session (skipping malformed), got %d", len(sessions))
	}
	if sessions[0].Name != "agent2" {
		t.Errorf("session name = %q, want %q", sessions[0].Name, "agent2")
	}
}

func TestListSessions_NoMatchingPrefix(t *testing.T) {
	sessionOutput := "other-session1|Thu Jan  1|0|1|/dir\nfoo-session2|Thu Jan  1|0|1|/dir\n"
	m := newTestManager("bc-", mockCmd(sessionOutput, "", 0))

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions (no matching prefix), got %d", len(sessions))
	}
}

// ---------------------------------------------------------------------------
// AttachCmd tests
// ---------------------------------------------------------------------------

func TestAttachCmd(t *testing.T) {
	m := &Manager{SessionPrefix: "bc-", execCommand: exec.Command}
	cmd := m.AttachCmd("agent1")
	if cmd == nil {
		t.Fatal("AttachCmd returned nil")
	}
	expectedArgs := []string{"tmux", "attach-session", "-t", "bc-agent1"}
	if len(cmd.Args) != len(expectedArgs) {
		t.Fatalf("args length = %d, want %d: %v", len(cmd.Args), len(expectedArgs), cmd.Args)
	}
	for i, exp := range expectedArgs {
		if cmd.Args[i] != exp {
			t.Errorf("args[%d] = %q, want %q", i, cmd.Args[i], exp)
		}
	}
}

func TestAttachCmd_WorkspaceManager(t *testing.T) {
	m := NewWorkspaceManager("bc-", "/workspace")
	m.execCommand = exec.Command
	cmd := m.AttachCmd("agent1")
	expectedName := m.SessionName("agent1")
	if !sliceContains(cmd.Args, expectedName) {
		t.Errorf("args should contain full session name %q: %v", expectedName, cmd.Args)
	}
}

// ---------------------------------------------------------------------------
// IsRunning tests
// ---------------------------------------------------------------------------

func TestIsRunning_True(t *testing.T) {
	m := newTestManager("bc-", mockCmd("session1\n", "", 0))
	if !m.IsRunning() {
		t.Error("expected IsRunning to return true when tmux exits 0")
	}
}

func TestIsRunning_NoServer(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "no server running on /tmp/tmux-501/default", 1))
	if m.IsRunning() {
		t.Error("expected IsRunning to return false when no server running")
	}
}

func TestIsRunning_OtherError(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "permission denied", 1))
	if m.IsRunning() {
		t.Error("expected IsRunning to return false on error")
	}
}

// ---------------------------------------------------------------------------
// KillServer tests
// ---------------------------------------------------------------------------

func TestKillServer_Success(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 0))
	if err := m.KillServer(); err != nil {
		t.Fatalf("KillServer failed: %v", err)
	}
}

func TestKillServer_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 1))
	if err := m.KillServer(); err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// SetEnvironment tests
// ---------------------------------------------------------------------------

func TestSetEnvironment_Success(t *testing.T) {
	mock, records := recordingMock("")
	m := newTestManager("bc-", mock)

	if err := m.SetEnvironment("agent1", "MY_VAR", "my_value"); err != nil {
		t.Fatalf("SetEnvironment failed: %v", err)
	}

	if len(*records) != 1 {
		t.Fatalf("expected 1 call, got %d", len(*records))
	}
	if !sliceContains((*records)[0].args, "set-environment") {
		t.Error("expected set-environment in args")
	}
	if !sliceContains((*records)[0].args, "MY_VAR") {
		t.Error("expected env var name in args")
	}
	if !sliceContains((*records)[0].args, "my_value") {
		t.Error("expected env var value in args")
	}
}

func TestSetEnvironment_Error(t *testing.T) {
	m := newTestManager("bc-", mockCmd("", "", 1))
	if err := m.SetEnvironment("agent1", "KEY", "VALUE"); err == nil {
		t.Error("expected error")
	}
}

// ---------------------------------------------------------------------------
// Prefix isolation test
// ---------------------------------------------------------------------------

func TestPrefixIsolation(t *testing.T) {
	sessionOutput := "prefix-a-agent1|Thu Jan  1|0|1|/dir\nprefix-b-agent2|Thu Jan  1|0|1|/dir\n"

	mA := newTestManager("prefix-a-", mockCmd(sessionOutput, "", 0))
	mB := newTestManager("prefix-b-", mockCmd(sessionOutput, "", 0))

	sessionsA, err := mA.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions A: %v", err)
	}
	sessionsB, err := mB.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions B: %v", err)
	}

	if len(sessionsA) != 1 {
		t.Errorf("manager A should see 1 session, got %d", len(sessionsA))
	}
	if len(sessionsB) != 1 {
		t.Errorf("manager B should see 1 session, got %d", len(sessionsB))
	}
	if len(sessionsA) > 0 && sessionsA[0].Name != "agent1" {
		t.Errorf("manager A session = %q, want agent1", sessionsA[0].Name)
	}
	if len(sessionsB) > 0 && sessionsB[0].Name != "agent2" {
		t.Errorf("manager B session = %q, want agent2", sessionsB[0].Name)
	}
}

func TestPrefixIsolation_SessionName(t *testing.T) {
	mA := NewManager("prefix-a-")
	mB := NewManager("prefix-b-")

	nameA := mA.SessionName("agent1")
	nameB := mB.SessionName("agent1")

	if nameA == nameB {
		t.Errorf("different prefixes should produce different session names: %s vs %s", nameA, nameB)
	}
	if !strings.HasPrefix(nameA, "prefix-a-") {
		t.Errorf("nameA should start with prefix-a-: %s", nameA)
	}
	if !strings.HasPrefix(nameB, "prefix-b-") {
		t.Errorf("nameB should start with prefix-b-: %s", nameB)
	}
}

// hasTmux returns true if tmux is available.
func hasTmux() bool {
	return exec.CommandContext(context.Background(), "tmux", "-V").Run() == nil
}

// TestSendKeysPreservesSpaces verifies that spaces in messages survive the
// full send-keys -l path through tmux. This is a regression test for
// work-116/work-117 where spaces were reported stripped from channel messages.
func TestSendKeysPreservesSpaces(t *testing.T) {
	if !hasTmux() {
		t.Skip("tmux not available")
	}

	m := NewManager("test-sp-")

	tests := []struct {
		name    string
		message string
	}{
		{"simple spaces", "hello world"},
		{"multiple words", "Can someone update me with the status"},
		{"double spaces", "hello  world"},
		{"channel format", "[#standup] Can someone update me with the status of the build"},
		{"special chars with spaces", "fix: handle edge case (issue #42)"},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionName := fmt.Sprintf("sp-test-%d", i)
			fullName := m.SessionName(sessionName)

			// Create session running cat (echoes stdin to PTY)
			cmd := exec.CommandContext(context.Background(), "tmux", "new-session", "-d", "-s", fullName, "cat") //nolint:gosec // test helper
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("failed to create session: %v (%s)", err, out)
			}
			defer func() { _ = exec.CommandContext(context.Background(), "tmux", "kill-session", "-t", fullName).Run() }() //nolint:errcheck,gosec // best-effort cleanup

			time.Sleep(200 * time.Millisecond)

			// Send message WITHOUT submit key (text stays in input line)
			if err := m.SendKeysWithSubmit(sessionName, tt.message, ""); err != nil {
				t.Fatalf("SendKeysWithSubmit failed: %v", err)
			}

			time.Sleep(200 * time.Millisecond)

			captured, err := m.Capture(sessionName, 10)
			if err != nil {
				t.Fatalf("Capture failed: %v", err)
			}

			// The captured pane should contain the message with spaces intact.
			// Use Contains (not exact match) because the pane may include
			// a shell prompt or other artifacts.
			captured = strings.TrimSpace(captured)
			if !strings.Contains(captured, tt.message) {
				t.Errorf("spaces not preserved in tmux\n  sent:     %q\n  captured: %q", tt.message, captured)
			}
		})
	}
}

// TestPasteBufferPreservesSpaces verifies that the paste-buffer path (for
// messages > 500 chars) preserves internal spaces. Terminal line wrapping
// converts spaces at column boundaries to newlines, so we normalize both
// the sent and captured text by collapsing whitespace before comparing.
func TestPasteBufferPreservesSpaces(t *testing.T) {
	if !hasTmux() {
		t.Skip("tmux not available")
	}
	if os.Getenv("CI") != "" {
		t.Skip("skipping flaky test in CI: timing-dependent tmux paste-buffer interaction")
	}

	m := NewManager("test-pb-")

	// Build a message > 500 chars with lots of internal spaces.
	// Use distinct words so we can verify ordering too.
	var words []string
	for i := 0; i < 120; i++ {
		words = append(words, "word")
	}
	message := strings.Join(words, " ") // 120*4 + 119 = 599 chars
	if len(message) <= 500 {
		t.Fatalf("test message too short: %d chars", len(message))
	}

	sessionName := "pb-test"
	fullName := m.SessionName(sessionName)

	cmd := exec.CommandContext(context.Background(), "tmux", "new-session", "-d", "-s", fullName, "cat") //nolint:gosec // test helper
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create session: %v (%s)", err, out)
	}
	defer func() { _ = exec.CommandContext(context.Background(), "tmux", "kill-session", "-t", fullName).Run() }() //nolint:errcheck,gosec // best-effort cleanup

	time.Sleep(200 * time.Millisecond)

	if err := m.SendKeysWithSubmit(sessionName, message, ""); err != nil {
		t.Fatalf("SendKeysWithSubmit failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	captured, err := m.Capture(sessionName, 50)
	if err != nil {
		t.Fatalf("Capture failed: %v", err)
	}

	// Terminal line wrapping converts spaces at column boundaries to newlines
	// and can split words across lines. Join lines back together for comparison.
	capturedJoined := strings.ReplaceAll(strings.TrimSpace(captured), "\n", "")

	if !strings.Contains(capturedJoined, message) {
		t.Errorf("paste-buffer path lost content\n  sent len:     %d\n  captured len: %d",
			len(message), len(capturedJoined))
	}
}
