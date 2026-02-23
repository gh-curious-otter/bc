// Package agent provides agent lifecycle management.
package agent

import (
	"io/fs"
	"os"
	"os/exec"

	"github.com/rpuneet/bc/pkg/tmux"
)

// FileSystem abstracts file system operations for testability.
// This allows tests to run without touching the real file system.
type FileSystem interface {
	// Stat returns the FileInfo for the file at path.
	Stat(path string) (fs.FileInfo, error)
	// ReadFile reads the file at path and returns its contents.
	ReadFile(path string) ([]byte, error)
	// WriteFile writes data to the file at path with the given permissions.
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// MkdirAll creates a directory along with any necessary parents.
	MkdirAll(path string, perm fs.FileMode) error
	// RemoveAll removes the path and any children it contains.
	RemoveAll(path string) error
}

// OSFileSystem implements FileSystem using the os package.
type OSFileSystem struct{}

// Stat implements FileSystem.
func (OSFileSystem) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(path)
}

// ReadFile implements FileSystem.
func (OSFileSystem) ReadFile(path string) ([]byte, error) {
	//nolint:gosec // path comes from trusted internal sources (workspace, config)
	return os.ReadFile(path)
}

// WriteFile implements FileSystem.
func (OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error {
	return os.WriteFile(path, data, perm)
}

// MkdirAll implements FileSystem.
func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

// RemoveAll implements FileSystem.
func (OSFileSystem) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// DefaultFileSystem returns an OSFileSystem.
func DefaultFileSystem() FileSystem {
	return OSFileSystem{}
}

// TmuxManager abstracts tmux session operations for testability.
// This interface matches the subset of tmux.Manager methods used by the agent package.
type TmuxManager interface {
	// HasSession checks if a session exists.
	HasSession(name string) bool
	// CreateSessionWithEnv creates a session with env vars baked into the shell command.
	CreateSessionWithEnv(name, dir, command string, env map[string]string) error
	// KillSession kills a tmux session.
	KillSession(name string) error
	// SendKeys sends keys to a session.
	SendKeys(name, keys string) error
	// Capture captures the current pane content.
	Capture(name string, lines int) (string, error)
	// ListSessions lists all sessions with our prefix.
	ListSessions() ([]tmux.Session, error)
	// AttachCmd returns an exec.Cmd to attach to a session.
	AttachCmd(name string) *exec.Cmd
}
