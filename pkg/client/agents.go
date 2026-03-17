package client

import "context"

// AgentsClient provides agent operations via the daemon.
type AgentsClient struct {
	client *Client
}

// AgentInfo represents agent data returned by the daemon.
type AgentInfo struct {
	Name    string `json:"name"`
	Role    string `json:"role"`
	State   string `json:"state"`
	Task    string `json:"task,omitempty"`
	Team    string `json:"team,omitempty"`
	Tool    string `json:"tool,omitempty"`
	Session string `json:"session,omitempty"`
}

// List returns all agents from the daemon.
func (a *AgentsClient) List(ctx context.Context) ([]AgentInfo, error) {
	var agents []AgentInfo
	if err := a.client.get(ctx, "/api/agents", &agents); err != nil {
		return nil, err
	}
	return agents, nil
}
