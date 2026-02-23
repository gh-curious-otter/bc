// Package errors provides structured error types for bc.
//
// This package defines sentinel errors for common failure modes and typed
// error structs that wrap underlying errors with additional context.
// Using these types enables consistent error handling with errors.Is() and
// errors.As() throughout the codebase.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
// Use errors.Is(err, ErrNotFound) to check for these conditions.
var (
	// ErrNotFound indicates a requested resource was not found.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists indicates a resource already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidInput indicates invalid input was provided.
	ErrInvalidInput = errors.New("invalid input")

	// ErrPermission indicates a permission/authorization failure.
	ErrPermission = errors.New("permission denied")

	// ErrTimeout indicates an operation timed out.
	ErrTimeout = errors.New("operation timeout")

	// ErrNotInWorkspace indicates the command was run outside a bc workspace.
	ErrNotInWorkspace = errors.New("not in a bc workspace")

	// ErrInvalidState indicates an operation was attempted in an invalid state.
	ErrInvalidState = errors.New("invalid state")

	// ErrCanceled indicates an operation was canceled.
	ErrCanceled = errors.New("operation canceled")
)

// AgentError represents an error related to an agent operation.
type AgentError struct {
	Err   error  // Underlying error
	Agent string // Agent name or ID
	Op    string // Operation that failed (e.g., "start", "stop", "attach")
}

// Error returns the error message.
func (e *AgentError) Error() string {
	if e.Agent == "" {
		return fmt.Sprintf("agent: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("agent %q: %s: %v", e.Agent, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *AgentError) Unwrap() error { return e.Err }

// NewAgentError creates a new AgentError.
func NewAgentError(agent, op string, err error) *AgentError {
	return &AgentError{Agent: agent, Op: op, Err: err}
}

// AgentNotFoundError creates an AgentError with ErrNotFound.
func AgentNotFoundError(agent string) *AgentError {
	return &AgentError{Agent: agent, Op: "lookup", Err: ErrNotFound}
}

// ChannelError represents an error related to a channel operation.
type ChannelError struct {
	Err     error  // Underlying error
	Channel string // Channel name
	Op      string // Operation that failed
}

// Error returns the error message.
func (e *ChannelError) Error() string {
	if e.Channel == "" {
		return fmt.Sprintf("channel: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("channel %q: %s: %v", e.Channel, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ChannelError) Unwrap() error { return e.Err }

// NewChannelError creates a new ChannelError.
func NewChannelError(channel, op string, err error) *ChannelError {
	return &ChannelError{Channel: channel, Op: op, Err: err}
}

// WorkspaceError represents an error related to workspace operations.
type WorkspaceError struct {
	Err  error  // Underlying error
	Path string // Workspace path
	Op   string // Operation that failed
}

// Error returns the error message.
func (e *WorkspaceError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("workspace: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("workspace %q: %s: %v", e.Path, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *WorkspaceError) Unwrap() error { return e.Err }

// NewWorkspaceError creates a new WorkspaceError.
func NewWorkspaceError(path, op string, err error) *WorkspaceError {
	return &WorkspaceError{Path: path, Op: op, Err: err}
}

// ConfigError represents an error related to configuration.
type ConfigError struct {
	Err error  // Underlying error
	Key string // Config key or section
	Op  string // Operation that failed
}

// Error returns the error message.
func (e *ConfigError) Error() string {
	if e.Key == "" {
		return fmt.Sprintf("config: %s: %v", e.Op, e.Err)
	}
	return fmt.Sprintf("config %q: %s: %v", e.Key, e.Op, e.Err)
}

// Unwrap returns the underlying error for errors.Is/As support.
func (e *ConfigError) Unwrap() error { return e.Err }

// NewConfigError creates a new ConfigError.
func NewConfigError(key, op string, err error) *ConfigError {
	return &ConfigError{Key: key, Op: op, Err: err}
}

// ValidationError represents a validation failure with details.
type ValidationError struct {
	Field   string // Field that failed validation
	Value   any    // The invalid value
	Message string // Human-readable validation message
}

// Error returns the error message.
func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("validation error on %q: %s", e.Field, e.Message)
}

// Unwrap returns ErrInvalidInput for errors.Is support.
func (e *ValidationError) Unwrap() error { return ErrInvalidInput }

// NewValidationError creates a new ValidationError.
func NewValidationError(field string, value any, message string) *ValidationError {
	return &ValidationError{Field: field, Value: value, Message: message}
}

// Is reports whether any error in err's chain matches target.
// This is a convenience re-export of errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target.
// This is a convenience re-export of errors.As.
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Wrap wraps an error with additional context.
// Returns nil if err is nil.
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// Wrapf wraps an error with formatted context.
// Returns nil if err is nil.
func Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
}
