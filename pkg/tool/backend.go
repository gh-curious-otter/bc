package tool

import "context"

// Backend is the storage interface for AI tool provider persistence.
// Store is the default SQLite implementation.
type Backend interface {
	// Open initializes the database and seeds built-in tools.
	Open() error
	// Add inserts a new tool. Returns an error if a tool with that name already exists.
	Add(ctx context.Context, t *Tool) error
	// Get returns a tool by name. Returns nil, nil if not found.
	Get(ctx context.Context, name string) (*Tool, error)
	// List returns all tools.
	List(ctx context.Context) ([]*Tool, error)
	// Update replaces a tool's mutable fields.
	Update(ctx context.Context, t *Tool) error
	// Delete removes a tool by name.
	Delete(ctx context.Context, name string) error
	// SetEnabled enables or disables a tool.
	SetEnabled(ctx context.Context, name string, enabled bool) error
	// Close releases database resources.
	Close() error
}
