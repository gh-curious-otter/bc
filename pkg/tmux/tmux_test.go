package tmux

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// mockExecer records calls and returns configurable responses.
type mockExecer struct {
	mu    sync.Mutex
	calls []mockCall

	combinedOutputFn func(name string, args ...string) ([]byte, error)
	outputFn         func(name string, args ...string) ([]byte, error)
	runFn            func(name string, args ...string) error
	runStderrFn      func(name string, args ...string) (error, string)
}

type mockCall struct {
	method string
	name   string
	args   []string
}

func (m *mockExecer) record(method, name string, args []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, mockCall{method, name, args})
}

func (m *mockExecer) getCalls() []mockCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]mockCall, len(m.calls))
	copy(cp, m.calls)
	return cp
}

func (m *mockExecer) combinedOutput(name string, args ...string) ([]byte, error) {
	m.record("combinedOutput", name, args)
	if m.combinedOutputFn != nil {
		return m.combinedOutputFn(name, args...)
	}
	return nil, nil
}

func (m *mockExecer) output(name string, args ...string) ([]byte, error) {
	m.record("output", name, args)
	if m.outputFn != nil {
		return m.outputFn(name, args...)
	}
	return nil, nil
}

func (m *mockExecer) run(name string, args ...string) error {
	m.record("run", name, args)
	if m.runFn != nil {
		return m.runFn(name, args...)
	}
	return nil
}

func (m *mockExecer) runStderr(name string, args ...string) (error, string) {
	m.record("runStderr", name, args)
	if m.runStderrFn != nil {
		return m.runStderrFn(name, args...)
	}
	return nil, ""
}

func (m *mockExecer) command(name string, args ...string) *exec.Cmd {
	m.record("command", name, args)
	return exec.Command(name, args...)
}

// newMockManager creates a Manager with a mock execer for testing.
func newMockManager(prefix string, mock *mockExecer) *Manager {
	return &Manager{
		SessionPrefix: prefix,
		exec:          mock,
	}
}

func newMockWorkspaceManager(prefix, path string, mock *mockExecer) *Manager {
	m := NewWorkspaceManager(prefix, path)
	m.exec = mock
	return m
}

// --- Constructor Tests ---

func TestNewDefaultManager_Prefix(t *testing.T) {
	m := NewDefaultManager()
	if m.SessionPrefix != "bc-" {
		t.Errorf("NewDefaultManager prefix = %q, want %q", m.SessionPrefix, "bc-")
	}
	if m.exec == nil {
		t.Error("NewDefaultManager exec should not be nil")
	}
}

func TestNewWorkspaceManager_DeterministicHash(t *testing.T) {
	m1 := NewWorkspaceManager("bc-", "/some/path")
	m2 := NewWorkspaceManager("bc-", "/some/path")
	if m1.workspaceHash != m2.workspaceHash {
		t.Errorf("same path should produce same hash: %q vs %q", m1.workspaceHash, m2.workspaceHash)
	}
}

func TestNewWorkspaceManager_DifferentPaths(t *testing.T) {
	m1 := NewWorkspaceManager("bc-", "/path/a")
	m2 := NewWorkspaceManager("bc-", "/path/b")
	if m1.workspaceHash == m2.workspaceHash {
		t.Errorf("different paths should produce different hashes: both %q", m1.workspaceHash)
	}
}

func TestNewManager_SetsExecer(t *testing.T) {
	m := NewManager("test-")
	if m.exec == nil {
		t.Error("NewManager exec should not be nil")
	}
}

// --- HasSession Tests ---

func TestHasSession_Exists(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	got := m.HasSession("agent1")
	if !got {
		t.Error("HasSession should return true when run succeeds")
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].method != "run" {
		t.Errorf("expected run call, got %s", calls[0].method)
	}
	wantArgs := []string{"has-session", "-t", "bc-agent1"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestHasSession_NotExists(t *testing.T) {
	mock := &mockExecer{
		runFn: func(name string, args ...string) error {
			return errors.New("session not found")
		},
	}
	m := newMockManager("bc-", mock)

	if m.HasSession("nonexistent") {
		t.Error("HasSession should return false when run fails")
	}
}

// --- CreateSession Tests ---

func TestCreateSession_NoDir(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.CreateSession("agent1", "")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	wantArgs := []string{"new-session", "-d", "-s", "bc-agent1"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestCreateSession_WithDir(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.CreateSession("agent1", "/tmp/work")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	calls := mock.getCalls()
	wantArgs := []string{"new-session", "-d", "-s", "bc-agent1", "-c", "/tmp/work"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestCreateSession_Error(t *testing.T) {
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("duplicate session"), errors.New("exit status 1")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.CreateSession("agent1", "")
	if err == nil {
		t.Fatal("CreateSession should return error")
	}
	if !strings.Contains(err.Error(), "failed to create session") {
		t.Errorf("error should mention failure: %v", err)
	}
	if !strings.Contains(err.Error(), "duplicate session") {
		t.Errorf("error should include output: %v", err)
	}
}

// --- CreateSessionWithCommand Tests ---

func TestCreateSessionWithCommand(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.CreateSessionWithCommand("agent1", "/tmp", "echo hello")
	if err != nil {
		t.Fatalf("CreateSessionWithCommand returned error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	// Should include bash -c and the command
	args := calls[0].args
	if args[len(args)-3] != "bash" || args[len(args)-2] != "-c" || args[len(args)-1] != "echo hello" {
		t.Errorf("expected bash -c 'echo hello' suffix, got %v", args[len(args)-3:])
	}
}

// --- CreateSessionWithEnv Tests ---

func TestCreateSessionWithEnv_WithVars(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	env := map[string]string{"FOO": "bar"}
	err := m.CreateSessionWithEnv("agent1", "/tmp", "run.sh", env)
	if err != nil {
		t.Fatalf("CreateSessionWithEnv returned error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	// The shell command should contain the export
	shellCmd := calls[0].args[len(calls[0].args)-1]
	if !strings.Contains(shellCmd, "export FOO=") {
		t.Errorf("shell command should contain export: %s", shellCmd)
	}
	if !strings.Contains(shellCmd, "run.sh") {
		t.Errorf("shell command should contain the command: %s", shellCmd)
	}
}

func TestCreateSessionWithEnv_NoDir(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.CreateSessionWithEnv("agent1", "", "cmd", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	args := calls[0].args
	// Should not contain -c flag for directory
	for i, a := range args {
		if a == "-c" && i > 0 && args[i-1] != "bash" {
			t.Error("should not include -c dir flag when dir is empty")
		}
	}
}

func TestCreateSessionWithEnv_NilEnv(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.CreateSessionWithEnv("agent1", "", "cmd", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	shellCmd := calls[0].args[len(calls[0].args)-1]
	if shellCmd != "cmd" {
		t.Errorf("shell command = %q, want %q", shellCmd, "cmd")
	}
}

func TestCreateSessionWithEnv_Error(t *testing.T) {
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("fail"), errors.New("exit status 1")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.CreateSessionWithEnv("agent1", "", "cmd", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to create session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- KillSession Tests ---

func TestKillSession_Success(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.KillSession("agent1")
	if err != nil {
		t.Fatalf("KillSession returned error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	wantArgs := []string{"kill-session", "-t", "bc-agent1"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestKillSession_Error(t *testing.T) {
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("no such session"), errors.New("exit status 1")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.KillSession("agent1")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to kill session") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- SendKeys / SendKeysWithSubmit Tests ---

func TestSendKeys_DelegatesToSendKeysWithSubmit(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.SendKeys("agent1", "hello")
	if err != nil {
		t.Fatalf("SendKeys returned error: %v", err)
	}

	calls := mock.getCalls()
	// Should make 2 calls: send-keys -l for the text, send-keys for Enter
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	// First call: literal text
	if !argsContain(calls[0].args, "-l") {
		t.Error("first call should use -l flag")
	}
	// Second call: Enter submit key
	lastArg := calls[1].args[len(calls[1].args)-1]
	if lastArg != "Enter" {
		t.Errorf("second call last arg = %q, want %q", lastArg, "Enter")
	}
}

func TestSendKeysWithSubmit_ShortMessage(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.SendKeysWithSubmit("agent1", "short msg", "Enter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	// send-keys -t bc-agent1 -l "short msg"
	wantArgs := []string{"send-keys", "-t", "bc-agent1", "-l", "short msg"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("call 0 args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestSendKeysWithSubmit_NoSubmitKey(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.SendKeysWithSubmit("agent1", "msg", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	// Should only make 1 call (no submit key)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call (no submit), got %d", len(calls))
	}
}

func TestSendKeysWithSubmit_TrimsTrailingNewlines(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.SendKeysWithSubmit("agent1", "msg\n\n\n", "Enter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	// The -l arg should be "msg" (newlines trimmed)
	sentKeys := calls[0].args[4] // send-keys -t X -l <keys>
	if sentKeys != "msg" {
		t.Errorf("keys = %q, want %q (newlines should be trimmed)", sentKeys, "msg")
	}
}

func TestSendKeysWithSubmit_ShortMessage_SendError(t *testing.T) {
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("err"), errors.New("send failed")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.SendKeysWithSubmit("agent1", "msg", "Enter")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to send keys") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_SubmitKeyError(t *testing.T) {
	callCount := 0
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			callCount++
			if callCount == 1 {
				return nil, nil // send-keys succeeds
			}
			return []byte("submit err"), errors.New("submit failed")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.SendKeysWithSubmit("agent1", "msg", "Enter")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to send submit key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_LongMessage(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	// Create a message longer than 500 chars
	longMsg := strings.Repeat("x", 501)

	err := m.SendKeysWithSubmit("agent1", longMsg, "Enter")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	// Long path: load-buffer, paste-buffer, send-keys (submit)
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls for long message, got %d", len(calls))
	}
	if !argsContain(calls[0].args, "load-buffer") {
		t.Errorf("call 0 should be load-buffer: %v", calls[0].args)
	}
	if !argsContain(calls[1].args, "paste-buffer") {
		t.Errorf("call 1 should be paste-buffer: %v", calls[1].args)
	}
	if !argsContain(calls[2].args, "send-keys") {
		t.Errorf("call 2 should be send-keys: %v", calls[2].args)
	}
}

func TestSendKeysWithSubmit_LongMessage_NoSubmit(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	longMsg := strings.Repeat("x", 501)
	err := m.SendKeysWithSubmit("agent1", longMsg, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	// No submit key: load-buffer, paste-buffer only
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls for long message no submit, got %d", len(calls))
	}
}

func TestSendKeysWithSubmit_LongMessage_LoadBufferError(t *testing.T) {
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			if argsContain(args, "load-buffer") {
				return []byte("load err"), errors.New("load failed")
			}
			return nil, nil
		},
	}
	m := newMockManager("bc-", mock)

	longMsg := strings.Repeat("x", 501)
	err := m.SendKeysWithSubmit("agent1", longMsg, "Enter")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to load buffer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSendKeysWithSubmit_LongMessage_PasteBufferError(t *testing.T) {
	mock := &mockExecer{
		combinedOutputFn: func(name string, args ...string) ([]byte, error) {
			if argsContain(args, "paste-buffer") {
				return []byte("paste err"), errors.New("paste failed")
			}
			return nil, nil
		},
	}
	m := newMockManager("bc-", mock)

	longMsg := strings.Repeat("x", 501)
	err := m.SendKeysWithSubmit("agent1", longMsg, "Enter")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to paste buffer") {
		t.Errorf("unexpected error: %v", err)
	}

	// Should also call delete-buffer for cleanup
	calls := mock.getCalls()
	foundDelete := false
	for _, c := range calls {
		if c.method == "run" && argsContain(c.args, "delete-buffer") {
			foundDelete = true
		}
	}
	if !foundDelete {
		t.Error("expected delete-buffer cleanup call on paste error")
	}
}

func TestSendKeysWithSubmit_CustomSubmitKey(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.SendKeysWithSubmit("agent1", "msg", "C-m")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	lastArg := calls[1].args[len(calls[1].args)-1]
	if lastArg != "C-m" {
		t.Errorf("submit key = %q, want %q", lastArg, "C-m")
	}
}

// --- Capture Tests ---

func TestCapture_NoLines(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("pane content\nline 2\n"), nil
		},
	}
	m := newMockManager("bc-", mock)

	got, err := m.Capture("agent1", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "pane content\nline 2\n" {
		t.Errorf("Capture = %q, want %q", got, "pane content\nline 2\n")
	}

	calls := mock.getCalls()
	wantArgs := []string{"capture-pane", "-t", "bc-agent1", "-p"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestCapture_WithLines(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("content"), nil
		},
	}
	m := newMockManager("bc-", mock)

	_, err := m.Capture("agent1", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	wantArgs := []string{"capture-pane", "-t", "bc-agent1", "-p", "-S", "-50"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestCapture_Error(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("no pane")
		},
	}
	m := newMockManager("bc-", mock)

	_, err := m.Capture("agent1", 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to capture pane") {
		t.Errorf("unexpected error: %v", err)
	}
}

// --- ListSessions Tests ---

func TestListSessions_ParsesOutput(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("bc-agent1|Mon Feb 05 10:00:00 2026|1|2|/home/user\nbc-agent2|Mon Feb 05 10:01:00 2026|0|1|/tmp\n"), nil
		},
	}
	m := newMockManager("bc-", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	if sessions[0].Name != "agent1" {
		t.Errorf("session 0 name = %q, want %q", sessions[0].Name, "agent1")
	}
	if !sessions[0].Attached {
		t.Error("session 0 should be attached")
	}
	if sessions[0].Directory != "/home/user" {
		t.Errorf("session 0 dir = %q, want %q", sessions[0].Directory, "/home/user")
	}

	if sessions[1].Name != "agent2" {
		t.Errorf("session 1 name = %q, want %q", sessions[1].Name, "agent2")
	}
	if sessions[1].Attached {
		t.Error("session 1 should not be attached")
	}
}

func TestListSessions_FiltersPrefix(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("bc-agent1|time|0|1|/dir\nother-session|time|0|1|/dir\nbc-agent2|time|1|1|/dir\n"), nil
		},
	}
	m := newMockManager("bc-", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions (filtered), got %d", len(sessions))
	}
	if sessions[0].Name != "agent1" || sessions[1].Name != "agent2" {
		t.Errorf("unexpected sessions: %v, %v", sessions[0].Name, sessions[1].Name)
	}
}

func TestListSessions_WorkspaceHashPrefix(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			m := NewWorkspaceManager("bc-", "/workspace")
			hash := m.workspaceHash
			return []byte(fmt.Sprintf("bc-%s-agent1|time|0|1|/dir\nbc-otherhash-agent2|time|0|1|/dir\n", hash)), nil
		},
	}
	m := newMockWorkspaceManager("bc-", "/workspace", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session (workspace filtered), got %d", len(sessions))
	}
	if sessions[0].Name != "agent1" {
		t.Errorf("session name = %q, want %q", sessions[0].Name, "agent1")
	}
}

func TestListSessions_NoServerRunning(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("no server running on /tmp/tmux-501/default")
		},
	}
	m := newMockManager("bc-", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("should not return error for 'no server running': %v", err)
	}
	if sessions != nil {
		t.Errorf("expected nil sessions, got %v", sessions)
	}
}

func TestListSessions_OtherError(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("connection refused")
		},
	}
	m := newMockManager("bc-", mock)

	_, err := m.ListSessions()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListSessions_SkipsEmptyLines(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("bc-a|time|0|1|/d\n\n\nbc-b|time|0|1|/d\n"), nil
		},
	}
	m := newMockManager("bc-", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestListSessions_SkipsShortLines(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte("bc-a|time|0\nbc-b|time|0|1|/d\n"), nil
		},
	}
	m := newMockManager("bc-", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session (short line skipped), got %d", len(sessions))
	}
}

func TestListSessions_EmptyOutput(t *testing.T) {
	mock := &mockExecer{
		outputFn: func(name string, args ...string) ([]byte, error) {
			return []byte(""), nil
		},
	}
	m := newMockManager("bc-", mock)

	sessions, err := m.ListSessions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

// --- AttachCmd Tests ---

func TestAttachCmd(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	cmd := m.AttachCmd("agent1")
	if cmd == nil {
		t.Fatal("AttachCmd returned nil")
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].method != "command" {
		t.Errorf("expected command call, got %s", calls[0].method)
	}
	wantArgs := []string{"attach-session", "-t", "bc-agent1"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestAttachCmd_WorkspaceManager(t *testing.T) {
	mock := &mockExecer{}
	m := newMockWorkspaceManager("bc-", "/workspace", mock)

	cmd := m.AttachCmd("agent1")
	if cmd == nil {
		t.Fatal("AttachCmd returned nil")
	}

	calls := mock.getCalls()
	expectedFullName := m.SessionName("agent1")
	wantArgs := []string{"attach-session", "-t", expectedFullName}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

// --- IsRunning Tests ---

func TestIsRunning_True(t *testing.T) {
	mock := &mockExecer{
		runStderrFn: func(name string, args ...string) (error, string) {
			return nil, ""
		},
	}
	m := newMockManager("bc-", mock)

	if !m.IsRunning() {
		t.Error("IsRunning should return true when no error")
	}
}

func TestIsRunning_NoServerRunning(t *testing.T) {
	mock := &mockExecer{
		runStderrFn: func(name string, args ...string) (error, string) {
			return errors.New("exit status 1"), "no server running on /tmp/tmux-501/default"
		},
	}
	m := newMockManager("bc-", mock)

	if m.IsRunning() {
		t.Error("IsRunning should return false for 'no server running'")
	}
}

func TestIsRunning_OtherError(t *testing.T) {
	mock := &mockExecer{
		runStderrFn: func(name string, args ...string) (error, string) {
			return errors.New("exit status 1"), "permission denied"
		},
	}
	m := newMockManager("bc-", mock)

	if m.IsRunning() {
		t.Error("IsRunning should return false for other errors")
	}
}

// --- KillServer Tests ---

func TestKillServer_Success(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.KillServer()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	wantArgs := []string{"kill-server"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestKillServer_Error(t *testing.T) {
	mock := &mockExecer{
		runFn: func(name string, args ...string) error {
			return errors.New("no server")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.KillServer()
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- SetEnvironment Tests ---

func TestSetEnvironment_Success(t *testing.T) {
	mock := &mockExecer{}
	m := newMockManager("bc-", mock)

	err := m.SetEnvironment("agent1", "MY_VAR", "my_value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	calls := mock.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	wantArgs := []string{"set-environment", "-t", "bc-agent1", "MY_VAR", "my_value"}
	if !argsEqual(calls[0].args, wantArgs) {
		t.Errorf("args = %v, want %v", calls[0].args, wantArgs)
	}
}

func TestSetEnvironment_Error(t *testing.T) {
	mock := &mockExecer{
		runFn: func(name string, args ...string) error {
			return errors.New("no session")
		},
	}
	m := newMockManager("bc-", mock)

	err := m.SetEnvironment("agent1", "KEY", "VAL")
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- SessionName with workspace hash ---

func TestSessionName_WorkspaceHash(t *testing.T) {
	m := NewWorkspaceManager("bc-", "/my/workspace")
	got := m.SessionName("agent1")
	want := "bc-" + m.workspaceHash + "-agent1"
	if got != want {
		t.Errorf("SessionName = %q, want %q", got, want)
	}
}

func TestSessionName_NoHash(t *testing.T) {
	m := NewManager("test-")
	got := m.SessionName("agent1")
	if got != "test-agent1" {
		t.Errorf("SessionName = %q, want %q", got, "test-agent1")
	}
}

// --- Helpers ---

func argsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func argsContain(args []string, target string) bool {
	for _, a := range args {
		if a == target {
			return true
		}
	}
	return false
}
