// Package runtime provides a backend-agnostic interface for agent session management.
//
// Agents can run in tmux sessions (default) or Docker containers. The Backend
// interface abstracts the session lifecycle so the agent package doesn't need
// to know which runtime is in use.
package runtime

import (
	"context"
	"os/exec"
)

// Session represents an agent session regardless of backend.
type Session struct {
	Name      string
	Created   string
	Directory string
	Attached  bool
}

// Backend abstracts session management for agents.
// Implementations exist for tmux (default) and Docker containers.
type Backend interface {
	// HasSession checks if a session exists.
	HasSession(ctx context.Context, name string) bool
	// CreateSession creates a new session.
	CreateSession(ctx context.Context, name, dir string) error
	// CreateSessionWithCommand creates a session and runs a command.
	CreateSessionWithCommand(ctx context.Context, name, dir, command string) error
	// CreateSessionWithEnv creates a session with env vars.
	CreateSessionWithEnv(ctx context.Context, name, dir, command string, env map[string]string) error
	// KillSession kills a session.
	KillSession(ctx context.Context, name string) error
	// RenameSession renames a session.
	RenameSession(ctx context.Context, oldName, newName string) error
	// SendKeys sends keys to a session with Enter as submit key.
	SendKeys(ctx context.Context, name, keys string) error
	// SendKeysWithSubmit sends keys to a session with a specified submit key.
	SendKeysWithSubmit(ctx context.Context, name, keys, submitKey string) error
	// Capture captures session output.
	Capture(ctx context.Context, name string, lines int) (string, error)
	// ListSessions lists all sessions managed by this backend.
	ListSessions(ctx context.Context) ([]Session, error)
	// AttachCmd returns an exec.Cmd to attach to a session.
	AttachCmd(ctx context.Context, name string) *exec.Cmd
	// IsRunning checks if the backend is running.
	IsRunning(ctx context.Context) bool
	// KillServer kills all sessions.
	KillServer(ctx context.Context) error
	// SetEnvironment sets an environment variable in a session.
	SetEnvironment(ctx context.Context, name, key, value string) error
	// SessionName returns the full session name with prefix.
	SessionName(name string) string
	// PipePane starts/stops streaming session output to a log file.
	PipePane(ctx context.Context, name, logPath string) error
}
