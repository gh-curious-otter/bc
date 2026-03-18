package mcp

// Backend is the storage interface for MCP server configuration persistence.
// Store is the default SQLite implementation.
type Backend interface {
	// Add inserts a new MCP server configuration.
	Add(cfg *ServerConfig) error
	// Get returns an MCP server config by name. Returns nil, nil if not found.
	Get(name string) (*ServerConfig, error)
	// List returns all MCP server configurations.
	List() ([]*ServerConfig, error)
	// Remove deletes an MCP server config by name.
	Remove(name string) error
	// SetEnabled enables or disables an MCP server.
	SetEnabled(name string, enabled bool) error
	// Close releases database resources.
	Close() error
}
