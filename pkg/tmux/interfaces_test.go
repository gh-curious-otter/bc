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
	HasSessionFunc           func(ctx context.Context, name string) bool
	CreateSessionFunc        func(ctx context.Context, name, dir string) error
	CreateSessionWithCmdFunc func(ctx context.Context, name, dir, command string) error
	CreateSessionWithEnvFunc func(ctx context.Context, name, dir, command string, env map[string]string) error
	KillSessionFunc          func(ctx context.Context, name string) error
	RenameSessionFunc        func(ctx context.Context, oldName, newName string) error
	SendKeysFunc             func(ctx context.Context, name, keys string) error
	SendKeysWithSubmitFunc   func(ctx context.Context, name, keys, submitKey string) error
	CaptureFunc              func(ctx context.Context, name string, lines int) (string, error)
	ListSessionsFunc         func(ctx context.Context) ([]Session, error)
	AttachCmdFunc            func(ctx context.Context, name string) *exec.Cmd
	IsRunningFunc            func(ctx context.Context) bool
	KillServerFunc           func(ctx context.Context) error
	SetEnvironmentFunc       func(ctx context.Context, name, key, value string) error
	SessionNameFunc          func(name string) string
}

func (m *MockSessionManager) HasSession(ctx context.Context, name string) bool {
	if m.HasSessionFunc != nil {
		return m.HasSessionFunc(ctx, name)
	}
	return false
}

func (m *MockSessionManager) CreateSession(ctx context.Context, name, dir string) error {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, name, dir)
	}
	return nil
}

func (m *MockSessionManager) CreateSessionWithCommand(ctx context.Context, name, dir, command string) error {
	if m.CreateSessionWithCmdFunc != nil {
		return m.CreateSessionWithCmdFunc(ctx, name, dir, command)
	}
	return nil
}

func (m *MockSessionManager) CreateSessionWithEnv(ctx context.Context, name, dir, command string, env map[string]string) error {
	if m.CreateSessionWithEnvFunc != nil {
		return m.CreateSessionWithEnvFunc(ctx, name, dir, command, env)
	}
	return nil
}

func (m *MockSessionManager) KillSession(ctx context.Context, name string) error {
	if m.KillSessionFunc != nil {
		return m.KillSessionFunc(ctx, name)
	}
	return nil
}

func (m *MockSessionManager) RenameSession(ctx context.Context, oldName, newName string) error {
	if m.RenameSessionFunc != nil {
		return m.RenameSessionFunc(ctx, oldName, newName)
	}
	return nil
}

func (m *MockSessionManager) SendKeys(ctx context.Context, name, keys string) error {
	if m.SendKeysFunc != nil {
		return m.SendKeysFunc(ctx, name, keys)
	}
	return nil
}

func (m *MockSessionManager) SendKeysWithSubmit(ctx context.Context, name, keys, submitKey string) error {
	if m.SendKeysWithSubmitFunc != nil {
		return m.SendKeysWithSubmitFunc(ctx, name, keys, submitKey)
	}
	return nil
}

func (m *MockSessionManager) Capture(ctx context.Context, name string, lines int) (string, error) {
	if m.CaptureFunc != nil {
		return m.CaptureFunc(ctx, name, lines)
	}
	return "", nil
}

func (m *MockSessionManager) ListSessions(ctx context.Context) ([]Session, error) {
	if m.ListSessionsFunc != nil {
		return m.ListSessionsFunc(ctx)
	}
	return nil, nil
}

func (m *MockSessionManager) AttachCmd(ctx context.Context, name string) *exec.Cmd {
	if m.AttachCmdFunc != nil {
		return m.AttachCmdFunc(ctx, name)
	}
	return nil
}

func (m *MockSessionManager) IsRunning(ctx context.Context) bool {
	if m.IsRunningFunc != nil {
		return m.IsRunningFunc(ctx)
	}
	return false
}

func (m *MockSessionManager) KillServer(ctx context.Context) error {
	if m.KillServerFunc != nil {
		return m.KillServerFunc(ctx)
	}
	return nil
}

func (m *MockSessionManager) SetEnvironment(ctx context.Context, name, key, value string) error {
	if m.SetEnvironmentFunc != nil {
		return m.SetEnvironmentFunc(ctx, name, key, value)
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

	ctx := context.Background()

	// Test with custom implementations
	mock := &MockSessionManager{
		HasSessionFunc: func(_ context.Context, name string) bool {
			return name == "exists"
		},
		CaptureFunc: func(_ context.Context, name string, lines int) (string, error) {
			return "captured output", nil
		},
	}

	if !mock.HasSession(ctx, "exists") {
		t.Error("MockSessionManager.HasSession(exists) = false, want true")
	}
	if mock.HasSession(ctx, "notexists") {
		t.Error("MockSessionManager.HasSession(notexists) = true, want false")
	}

	output, err := mock.Capture(ctx, "test", 10)
	if err != nil {
		t.Errorf("MockSessionManager.Capture() error = %v, want nil", err)
	}
	if output != "captured output" {
		t.Errorf("MockSessionManager.Capture() = %v, want captured output", output)
	}
}
