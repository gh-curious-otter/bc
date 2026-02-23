// Package tmux provides tmux session management for agent orchestration.
package tmux

import (
	"os/exec"
)

// CommandExecutor abstracts command execution for testability.
// This allows tests to mock exec.Command calls without actually running processes.
type CommandExecutor interface {
	// Command creates an exec.Cmd for the given command and arguments.
	Command(name string, arg ...string) *exec.Cmd
}

// execCommandFunc adapts a function to the CommandExecutor interface.
type execCommandFunc func(name string, arg ...string) *exec.Cmd

// Command implements CommandExecutor.
func (f execCommandFunc) Command(name string, arg ...string) *exec.Cmd {
	return f(name, arg...)
}

// DefaultExecutor returns a CommandExecutor using exec.Command.
func DefaultExecutor() CommandExecutor {
	return execCommandFunc(exec.Command)
}

// Session interface abstracts tmux session operations for testability.
// This allows agent code to work with mock implementations in tests.
type SessionManager interface {
	// HasSession checks if a session exists.
	HasSession(name string) bool
	// CreateSession creates a new tmux session.
	CreateSession(name, dir string) error
	// CreateSessionWithCommand creates a session and runs a command.
	CreateSessionWithCommand(name, dir, command string) error
	// CreateSessionWithEnv creates a session with env vars baked into the shell command.
	CreateSessionWithEnv(name, dir, command string, env map[string]string) error
	// KillSession kills a tmux session.
	KillSession(name string) error
	// RenameSession renames a tmux session.
	RenameSession(oldName, newName string) error
	// SendKeys sends keys to a session with Enter as submit key.
	SendKeys(name, keys string) error
	// SendKeysWithSubmit sends keys to a session with a specified submit key.
	SendKeysWithSubmit(name, keys, submitKey string) error
	// Capture captures the current pane content.
	Capture(name string, lines int) (string, error)
	// ListSessions lists all sessions with our prefix.
	ListSessions() ([]Session, error)
	// AttachCmd returns an exec.Cmd to attach to a session.
	AttachCmd(name string) *exec.Cmd
	// IsRunning checks if tmux server is running.
	IsRunning() bool
	// KillServer kills the tmux server (all sessions).
	KillServer() error
	// SetEnvironment sets an environment variable in a session.
	SetEnvironment(name, key, value string) error
	// SessionName returns the full session name with prefix.
	SessionName(name string) string
}

// Ensure Manager implements SessionManager.
var _ SessionManager = (*Manager)(nil)
