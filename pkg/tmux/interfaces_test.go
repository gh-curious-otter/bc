package tmux

import (
	"context"
	"os/exec"
	"testing"
)

func TestDefaultExecutor(t *testing.T) {
	executor := DefaultExecutor()
	if executor == nil {
		t.Fatal("DefaultExecutor() = nil, want non-nil")
	}

	// Test that it creates a command
	cmd := executor.Command("echo", "hello")
	if cmd == nil {
		t.Fatal("DefaultExecutor.Command() = nil, want non-nil")
	}
	if cmd.Path == "" {
		t.Error("DefaultExecutor.Command() path is empty")
	}
}

func TestExecCommandFunc(t *testing.T) {
	called := false
	var capturedName string
	var capturedArgs []string

	f := execCommandFunc(func(name string, arg ...string) *exec.Cmd {
		called = true
		capturedName = name
		capturedArgs = arg
		return exec.CommandContext(context.Background(), name, arg...)
	})

	cmd := f.Command("test", "arg1", "arg2")
	if !called {
		t.Error("execCommandFunc.Command() did not call the function")
	}
	if capturedName != "test" {
		t.Errorf("execCommandFunc.Command() name = %v, want test", capturedName)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "arg1" || capturedArgs[1] != "arg2" {
		t.Errorf("execCommandFunc.Command() args = %v, want [arg1 arg2]", capturedArgs)
	}
	if cmd == nil {
		t.Error("execCommandFunc.Command() = nil, want non-nil")
	}
}

func TestManagerImplementsSessionManager(t *testing.T) {
	// Verify Manager implements SessionManager
	var _ SessionManager = (*Manager)(nil)
}

// MockCommandExecutor is a test double for CommandExecutor.
type MockCommandExecutor struct {
	CommandFunc func(name string, arg ...string) *exec.Cmd
}

func (m *MockCommandExecutor) Command(name string, arg ...string) *exec.Cmd {
	if m.CommandFunc != nil {
		return m.CommandFunc(name, arg...)
	}
	return exec.CommandContext(context.Background(), name, arg...)
}

func TestMockCommandExecutor(t *testing.T) {
	// Verify MockCommandExecutor implements CommandExecutor
	var _ CommandExecutor = &MockCommandExecutor{}

	// Test with custom implementation
	mock := &MockCommandExecutor{
		CommandFunc: func(name string, arg ...string) *exec.Cmd {
			// Return a simple echo command regardless of input
			return exec.CommandContext(context.Background(), "echo", "mocked")
		},
	}

	cmd := mock.Command("anything", "ignored")
	if cmd == nil {
		t.Fatal("MockCommandExecutor.Command() = nil, want non-nil")
	}

	output, err := cmd.Output()
	if err != nil {
		t.Errorf("MockCommandExecutor command execution failed: %v", err)
	}
	if string(output) != "mocked\n" {
		t.Errorf("MockCommandExecutor output = %v, want mocked\\n", string(output))
	}
}

// MockSessionManager is a test double for SessionManager.
type MockSessionManager struct {
	HasSessionFunc           func(name string) bool
	CreateSessionFunc        func(name, dir string) error
	CreateSessionWithCmdFunc func(name, dir, command string) error
	CreateSessionWithEnvFunc func(name, dir, command string, env map[string]string) error
	KillSessionFunc          func(name string) error
	RenameSessionFunc        func(oldName, newName string) error
	SendKeysFunc             func(name, keys string) error
	SendKeysWithSubmitFunc   func(name, keys, submitKey string) error
	CaptureFunc              func(name string, lines int) (string, error)
	ListSessionsFunc         func() ([]Session, error)
	AttachCmdFunc            func(name string) *exec.Cmd
	IsRunningFunc            func() bool
	KillServerFunc           func() error
	SetEnvironmentFunc       func(name, key, value string) error
	SessionNameFunc          func(name string) string
}

func (m *MockSessionManager) HasSession(name string) bool {
	if m.HasSessionFunc != nil {
		return m.HasSessionFunc(name)
	}
	return false
}

func (m *MockSessionManager) CreateSession(name, dir string) error {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(name, dir)
	}
	return nil
}

func (m *MockSessionManager) CreateSessionWithCommand(name, dir, command string) error {
	if m.CreateSessionWithCmdFunc != nil {
		return m.CreateSessionWithCmdFunc(name, dir, command)
	}
	return nil
}

func (m *MockSessionManager) CreateSessionWithEnv(name, dir, command string, env map[string]string) error {
	if m.CreateSessionWithEnvFunc != nil {
		return m.CreateSessionWithEnvFunc(name, dir, command, env)
	}
	return nil
}

func (m *MockSessionManager) KillSession(name string) error {
	if m.KillSessionFunc != nil {
		return m.KillSessionFunc(name)
	}
	return nil
}

func (m *MockSessionManager) RenameSession(oldName, newName string) error {
	if m.RenameSessionFunc != nil {
		return m.RenameSessionFunc(oldName, newName)
	}
	return nil
}

func (m *MockSessionManager) SendKeys(name, keys string) error {
	if m.SendKeysFunc != nil {
		return m.SendKeysFunc(name, keys)
	}
	return nil
}

func (m *MockSessionManager) SendKeysWithSubmit(name, keys, submitKey string) error {
	if m.SendKeysWithSubmitFunc != nil {
		return m.SendKeysWithSubmitFunc(name, keys, submitKey)
	}
	return nil
}

func (m *MockSessionManager) Capture(name string, lines int) (string, error) {
	if m.CaptureFunc != nil {
		return m.CaptureFunc(name, lines)
	}
	return "", nil
}

func (m *MockSessionManager) ListSessions() ([]Session, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc()
	}
	return nil, nil
}

func (m *MockSessionManager) AttachCmd(name string) *exec.Cmd {
	if m.AttachCmdFunc != nil {
		return m.AttachCmdFunc(name)
	}
	return nil
}

func (m *MockSessionManager) IsRunning() bool {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc()
	}
	return false
}

func (m *MockSessionManager) KillServer() error {
	if m.KillServerFunc != nil {
		return m.KillServerFunc()
	}
	return nil
}

func (m *MockSessionManager) SetEnvironment(name, key, value string) error {
	if m.SetEnvironmentFunc != nil {
		return m.SetEnvironmentFunc(name, key, value)
	}
	return nil
}

func (m *MockSessionManager) SessionName(name string) string {
	if m.SessionNameFunc != nil {
		return m.SessionNameFunc(name)
	}
	return name
}

func TestMockSessionManager(t *testing.T) {
	// Verify MockSessionManager implements SessionManager
	var _ SessionManager = &MockSessionManager{}

	// Test with custom implementations
	mock := &MockSessionManager{
		HasSessionFunc: func(name string) bool {
			return name == "exists"
		},
		CaptureFunc: func(name string, lines int) (string, error) {
			return "captured output", nil
		},
	}

	if !mock.HasSession("exists") {
		t.Error("MockSessionManager.HasSession(exists) = false, want true")
	}
	if mock.HasSession("notexists") {
		t.Error("MockSessionManager.HasSession(notexists) = true, want false")
	}

	output, err := mock.Capture("test", 10)
	if err != nil {
		t.Errorf("MockSessionManager.Capture() error = %v, want nil", err)
	}
	if output != "captured output" {
		t.Errorf("MockSessionManager.Capture() = %v, want captured output", output)
	}
}
