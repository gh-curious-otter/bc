// Package errors provides structured error types for bc CLI.
//
// This package defines sentinel errors and typed error structs for consistent
// error handling across the codebase. Use errors.Is() to check for sentinel
// errors and errors.As() to extract typed error details.
//
// # Sentinel Errors
//
// Sentinel errors represent common failure categories:
//
//	if errors.Is(err, bcerrors.ErrNotFound) {
//	    // Handle not found case
//	}
//
// # Typed Errors
//
// Typed errors provide additional context for specific operations:
//
//	var agentErr *bcerrors.AgentError
//	if errors.As(err, &agentErr) {
//	    fmt.Printf("agent %s failed: %s\n", agentErr.Agent, agentErr.Op)
//	}
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
// Use errors.Is() to check for these errors.
var (
	// ErrNotFound indicates a requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists indicates a resource already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidInput indicates invalid input was provided.
	ErrInvalidInput = errors.New("invalid input")

	// ErrPermission indicates a permission error.
	ErrPermission = errors.New("permission denied")

	// ErrTimeout indicates an operation timed out.
	ErrTimeout = errors.New("operation timeout")

	// ErrNotInWorkspace indicates the command was run outside a bc workspace.
	ErrNotInWorkspace = errors.New("not in a bc workspace")

	// ErrSessionNotFound indicates a tmux session was not found.
	ErrSessionNotFound = errors.New("session not found")

	// ErrAgentStopped indicates an agent has stopped.
	ErrAgentStopped = errors.New("agent stopped")

	// ErrChannelNotFound indicates a channel was not found.
	ErrChannelNotFound = errors.New("channel not found")

	// ErrRoleNotFound indicates a role was not found.
	ErrRoleNotFound = errors.New("role not found")
)

// AgentError represents an error related to agent operations.
type AgentError struct {
	Err   error  // Underlying error
	Agent string // Agent name
	Op    string // Operation that failed (e.g., "start", "stop", "send")
}

func (e *AgentError) Error() string {
	if e.Agent == "" {
		return fmt.Sprintf("agent: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("agent %q: %s: %v", e.Agent, e.Op, e.Err)
}

func (e *AgentError) Unwrap() error { return e.Err }

// NewAgentError creates a new AgentError.
func NewAgentError(agent, op string, err error) *AgentError {
	return &AgentError{Agent: agent, Op: op, Err: err}
}

// WorkspaceError represents an error related to workspace operations.
type WorkspaceError struct {
	Err  error  // Underlying error
	Path string // Workspace path
	Op   string // Operation that failed
}

func (e *WorkspaceError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("workspace: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("workspace %q: %s: %v", e.Path, e.Op, e.Err)
}

func (e *WorkspaceError) Unwrap() error { return e.Err }

// NewWorkspaceError creates a new WorkspaceError.
func NewWorkspaceError(path, op string, err error) *WorkspaceError {
	return &WorkspaceError{Path: path, Op: op, Err: err}
}

// ChannelError represents an error related to channel operations.
type ChannelError struct {
	Err     error  // Underlying error
	Channel string // Channel name
	Op      string // Operation that failed
}

func (e *ChannelError) Error() string {
	if e.Channel == "" {
		return fmt.Sprintf("channel: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("channel %q: %s: %v", e.Channel, e.Op, e.Err)
}

func (e *ChannelError) Unwrap() error { return e.Err }

// NewChannelError creates a new ChannelError.
func NewChannelError(channel, op string, err error) *ChannelError {
	return &ChannelError{Channel: channel, Op: op, Err: err}
}

// TmuxError represents an error related to tmux operations.
type TmuxError struct {
	Err     error  // Underlying error
	Session string // Session name (if applicable)
	Op      string // Operation that failed
}

func (e *TmuxError) Error() string {
	if e.Session == "" {
		return fmt.Sprintf("tmux: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("tmux session %q: %s: %v", e.Session, e.Op, e.Err)
}

func (e *TmuxError) Unwrap() error { return e.Err }

// NewTmuxError creates a new TmuxError.
func NewTmuxError(session, op string, err error) *TmuxError {
	return &TmuxError{Session: session, Op: op, Err: err}
}

// RoleError represents an error related to role operations.
type RoleError struct {
	Err  error  // Underlying error
	Role string // Role name
	Op   string // Operation that failed
}

func (e *RoleError) Error() string {
	if e.Role == "" {
		return fmt.Sprintf("role: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("role %q: %s: %v", e.Role, e.Op, e.Err)
}

func (e *RoleError) Unwrap() error { return e.Err }

// NewRoleError creates a new RoleError.
func NewRoleError(role, op string, err error) *RoleError {
	return &RoleError{Role: role, Op: op, Err: err}
}

// Is reports whether any error in err's tree matches target.
// This is a convenience wrapper around errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's tree that matches target.
// This is a convenience wrapper around errors.As.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Wrap returns an error annotating err with a message.
// If err is nil, Wrap returns nil.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf returns an error annotating err with a formatted message.
// If err is nil, Wrapf returns nil.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
