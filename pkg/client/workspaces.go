package client

import "context"

// WorkspacesClient provides workspace operations via the daemon.
type WorkspacesClient struct {
	client *Client
}

// WorkspaceStatus represents workspace status from the daemon.
type WorkspaceStatus struct {
	Name         string `json:"name"`
	RootDir      string `json:"root_dir"`
	AgentCount   int    `json:"agent_count"`
	RunningCount int    `json:"running_count"`
	IsHealthy    bool   `json:"is_healthy"`
}

// Status returns the current workspace status.
func (w *WorkspacesClient) Status(ctx context.Context) (*WorkspaceStatus, error) {
	var status WorkspaceStatus
	if err := w.client.get(ctx, "/api/workspace/status", &status); err != nil {
		return nil, err
	}
	return &status, nil
}

// Up starts the workspace (creates/starts root agent).
func (w *WorkspacesClient) Up(ctx context.Context, tool, runtime string) (map[string]any, error) {
	body := map[string]string{"tool": tool, "runtime": runtime}
	var result map[string]any
	if err := w.client.post(ctx, "/api/workspace/up", body, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// Down stops all running agents. Returns the number stopped.
func (w *WorkspacesClient) Down(ctx context.Context) (int, error) {
	var result map[string]int
	if err := w.client.post(ctx, "/api/workspace/down", nil, &result); err != nil {
		return 0, err
	}
	return result["stopped"], nil
}
