package client

import "context"

// ToolsClient provides tool management operations via the daemon.
type ToolsClient struct {
	client *Client
}

// ToolInfo represents a tool configuration returned by the daemon.
type ToolInfo struct {
	MCPServers []string `json:"mcp_servers,omitempty"`
	SlashCmds  []string `json:"slash_cmds,omitempty"`
	Name       string   `json:"name"`
	Command    string   `json:"command,omitempty"`
	InstallCmd string   `json:"install_cmd,omitempty"`
	UpgradeCmd string   `json:"upgrade_cmd,omitempty"`
	Enabled    bool     `json:"enabled"`
	Builtin    bool     `json:"builtin,omitempty"`
}

// List returns all tools.
func (t *ToolsClient) List(ctx context.Context) ([]*ToolInfo, error) {
	var tools []*ToolInfo
	if err := t.client.get(ctx, "/api/tools", &tools); err != nil {
		return nil, err
	}
	return tools, nil
}

// Get returns a specific tool by name.
func (t *ToolsClient) Get(ctx context.Context, name string) (*ToolInfo, error) {
	var tool ToolInfo
	if err := t.client.get(ctx, "/api/tools/"+name, &tool); err != nil {
		return nil, err
	}
	return &tool, nil
}
