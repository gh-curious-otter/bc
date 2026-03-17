package client

import "context"

// WorkspacesClient provides workspace operations via the daemon.
type WorkspacesClient struct {
	client *Client
}

// WorkspaceStatus represents workspace status from the daemon.
type WorkspaceStatus struct {
	Name       string `json:"name"`
	RootDir    string `json:"root_dir"`
	AgentCount int    `json:"agent_count"`
	IsHealthy  bool   `json:"is_healthy"`
}

// Status returns the current workspace status.
func (w *WorkspacesClient) Status(ctx context.Context) (*WorkspaceStatus, error) {
	var status WorkspaceStatus
	if err := w.client.get(ctx, "/api/workspace/status", &status); err != nil {
		return nil, err
	}
	return &status, nil
}
