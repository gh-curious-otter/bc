package client

import (
	"context"
	"fmt"
	"time"
)

// AgentsClient provides agent operations via the daemon.
type AgentsClient struct {
	client *Client
}

// AgentInfo represents agent data returned by the daemon.
type AgentInfo struct {
	CreatedAt time.Time  `json:"created_at"`
	StartedAt time.Time  `json:"started_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	ID        string     `json:"id,omitempty"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	State     string     `json:"state"`
	Task      string     `json:"task,omitempty"`
	Team      string     `json:"team,omitempty"`
	Tool      string     `json:"tool,omitempty"`
	Session   string     `json:"session,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	ParentID  string     `json:"parent_id,omitempty"`
	Children  []string   `json:"children,omitempty"`
}

// CreateAgentReq is the request to create an agent.
type CreateAgentReq struct {
	Name    string `json:"name"`
	Role    string `json:"role"`
	Tool    string `json:"tool,omitempty"`
	Runtime string `json:"runtime,omitempty"`
	Parent  string `json:"parent,omitempty"`
	Team    string `json:"team,omitempty"`
	EnvFile string `json:"env_file,omitempty"`
}

// SendResultInfo holds the result of a broadcast/role/pattern send.
type SendResultInfo struct {
	Matched []string `json:"matched"`
	Sent    int      `json:"sent"`
	Skipped int      `json:"skipped"`
	Failed  int      `json:"failed"`
}

// SessionInfo represents a single session history entry.
type SessionInfo struct {
	Timestamp time.Time `json:"timestamp,omitempty"`
	ID        string    `json:"id"`
	Current   bool      `json:"current,omitempty"`
}

// List returns all agents from the daemon.
func (a *AgentsClient) List(ctx context.Context) ([]AgentInfo, error) {
	var agents []AgentInfo
	if err := a.client.get(ctx, "/api/agents", &agents); err != nil {
		return nil, err
	}
	return agents, nil
}

// ListByRole returns agents filtered by role.
func (a *AgentsClient) ListByRole(ctx context.Context, role string) ([]AgentInfo, error) {
	agents, err := a.List(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]AgentInfo, 0)
	for _, ag := range agents {
		if ag.Role == role {
			filtered = append(filtered, ag)
		}
	}
	return filtered, nil
}

// Get returns a single agent by name.
func (a *AgentsClient) Get(ctx context.Context, name string) (*AgentInfo, error) {
	var info AgentInfo
	if err := a.client.get(ctx, "/api/agents/"+name, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Create creates a new agent.
func (a *AgentsClient) Create(ctx context.Context, req CreateAgentReq) (*AgentInfo, error) {
	var info AgentInfo
	if err := a.client.post(ctx, "/api/agents", req, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Start starts a stopped agent. resumeID optionally specifies a session ID to resume.
func (a *AgentsClient) Start(ctx context.Context, name, runtime, resumeID string, fresh bool) (*AgentInfo, error) {
	body := map[string]any{"runtime": runtime, "fresh": fresh, "resume_id": resumeID}
	var info AgentInfo
	if err := a.client.post(ctx, "/api/agents/"+name+"/start", body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Stop stops a running agent.
func (a *AgentsClient) Stop(ctx context.Context, name string) error {
	return a.client.post(ctx, "/api/agents/"+name+"/stop", nil, nil)
}

// Delete permanently removes an agent.
func (a *AgentsClient) Delete(ctx context.Context, name string) error {
	return a.client.delete(ctx, "/api/agents/"+name)
}

// Send sends a message to a running agent.
func (a *AgentsClient) Send(ctx context.Context, name, message string) error {
	body := map[string]string{"message": message}
	return a.client.post(ctx, "/api/agents/"+name+"/send", body, nil)
}

// Rename renames an agent.
func (a *AgentsClient) Rename(ctx context.Context, oldName, newName string) error {
	body := map[string]string{"new_name": newName}
	return a.client.post(ctx, "/api/agents/"+oldName+"/rename", body, nil)
}

// Peek returns recent output from an agent.
func (a *AgentsClient) Peek(ctx context.Context, name string, lines int) (string, error) {
	path := fmt.Sprintf("/api/agents/%s/peek?lines=%d", name, lines)
	var result map[string]string
	if err := a.client.get(ctx, path, &result); err != nil {
		return "", err
	}
	return result["output"], nil
}

// Sessions returns session history for an agent.
func (a *AgentsClient) Sessions(ctx context.Context, name string) ([]SessionInfo, error) {
	var sessions []SessionInfo
	if err := a.client.get(ctx, "/api/agents/"+name+"/sessions", &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// Broadcast sends a message to all running agents.
func (a *AgentsClient) Broadcast(ctx context.Context, message string) (int, error) {
	body := map[string]string{"message": message}
	var result map[string]int
	if err := a.client.post(ctx, "/api/agents/broadcast", body, &result); err != nil {
		return 0, err
	}
	return result["sent"], nil
}

// SendToRole sends a message to all agents with the given role.
func (a *AgentsClient) SendToRole(ctx context.Context, role, message string) (*SendResultInfo, error) {
	body := map[string]string{"role": role, "message": message}
	var result SendResultInfo
	if err := a.client.post(ctx, "/api/agents/send-role", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SendToPattern sends a message to agents matching the given glob pattern.
func (a *AgentsClient) SendToPattern(ctx context.Context, pattern, message string) (*SendResultInfo, error) {
	body := map[string]string{"pattern": pattern, "message": message}
	var result SendResultInfo
	if err := a.client.post(ctx, "/api/agents/send-pattern", body, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GenerateName generates a unique agent name.
func (a *AgentsClient) GenerateName(ctx context.Context) (string, error) {
	var result map[string]string
	if err := a.client.get(ctx, "/api/agents/generate-name", &result); err != nil {
		return "", err
	}
	return result["name"], nil
}

// StopAll stops all running agents. Returns the number of agents stopped.
func (a *AgentsClient) StopAll(ctx context.Context) (int, error) {
	var result map[string]int
	if err := a.client.post(ctx, "/api/workspace/down", nil, &result); err != nil {
		return 0, err
	}
	return result["stopped"], nil
}
