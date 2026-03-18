package daemon

import "context"

// Backend is the storage and runtime interface for workspace daemon management.
// Manager is the default implementation using tmux/Docker.
type Backend interface {
	// Run starts a new daemon or restarts an existing stopped one.
	Run(ctx context.Context, opts RunOptions) (*Daemon, error)
	// Stop stops a running daemon.
	Stop(ctx context.Context, name string) error
	// Restart stops and restarts a daemon using its saved configuration.
	Restart(ctx context.Context, name string) (*Daemon, error)
	// Remove permanently deletes a daemon record. The daemon must be stopped first.
	Remove(ctx context.Context, name string) error
	// List returns all daemons.
	List(ctx context.Context) ([]*Daemon, error)
	// Get returns a daemon by name or nil if not found.
	Get(ctx context.Context, name string) (*Daemon, error)
	// Logs returns recent log lines for a daemon.
	Logs(ctx context.Context, name string, lines int) (string, error)
	// Close releases database resources.
	Close() error
}
