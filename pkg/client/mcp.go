package client

import "context"

// MCPClient provides MCP server configuration operations via the daemon.
type MCPClient struct {
	client *Client
}

// MCPServerConfig represents an MCP server configuration.
type MCPServerConfig struct {
	Env       map[string]string `json:"env,omitempty"`
	Name      string            `json:"name"`
	Transport string            `json:"transport"`
	Command   string            `json:"command,omitempty"`
	URL       string            `json:"url,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Enabled   bool              `json:"enabled"`
}

// List returns all MCP server configurations.
func (m *MCPClient) List(ctx context.Context) ([]*MCPServerConfig, error) {
	var configs []*MCPServerConfig
	if err := m.client.get(ctx, "/api/mcp", &configs); err != nil {
		return nil, err
	}
	return configs, nil
}

// Get returns a specific MCP server configuration.
func (m *MCPClient) Get(ctx context.Context, name string) (*MCPServerConfig, error) {
	var cfg MCPServerConfig
	if err := m.client.get(ctx, "/api/mcp/"+name, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Add adds a new MCP server configuration.
func (m *MCPClient) Add(ctx context.Context, cfg *MCPServerConfig) (*MCPServerConfig, error) {
	var created MCPServerConfig
	if err := m.client.post(ctx, "/api/mcp", cfg, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// Remove removes an MCP server configuration.
func (m *MCPClient) Remove(ctx context.Context, name string) error {
	return m.client.delete(ctx, "/api/mcp/"+name)
}

// Enable enables an MCP server configuration.
func (m *MCPClient) Enable(ctx context.Context, name string) error {
	return m.client.post(ctx, "/api/mcp/"+name+"/enable", nil, nil)
}

// Disable disables an MCP server configuration.
func (m *MCPClient) Disable(ctx context.Context, name string) error {
	return m.client.post(ctx, "/api/mcp/"+name+"/disable", nil, nil)
}
