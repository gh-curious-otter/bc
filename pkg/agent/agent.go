// Package agent provides agent lifecycle management.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/tmux"
)

// Role defines the type of agent.
type Role string

const (
	RoleCoordinator Role = "coordinator"
	RoleWorker      Role = "worker"
)

// State represents the current state of an agent.
type State string

const (
	StateIdle     State = "idle"
	StateStarting State = "starting"
	StateWorking  State = "working"
	StateDone     State = "done"
	StateStuck    State = "stuck"
	StateError    State = "error"
	StateStopped  State = "stopped"
)

// Agent represents a running AI agent.
type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	State     State     `json:"state"`
	Workspace string    `json:"workspace"`
	Task      string    `json:"task,omitempty"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Session info
	Session   string `json:"session"`
	
	// For workers
	HookedWork string `json:"hooked_work,omitempty"`
}

// Manager handles agent lifecycle.
type Manager struct {
	mu       sync.RWMutex
	agents   map[string]*Agent
	tmux     *tmux.Manager
	stateDir string
	
	// Agent command (e.g., "claude" or "claude --dangerously-skip-permissions")
	agentCmd string
}

// NewManager creates a new agent manager.
func NewManager(stateDir string) *Manager {
	return &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager(config.Tmux.SessionPrefix),
		stateDir: stateDir,
		agentCmd: config.Agent.Command,
	}
}

// SetAgentCommand sets the command to run for agents.
// Use config.Agents to see available agent types.
func (m *Manager) SetAgentCommand(cmd string) {
	m.agentCmd = cmd
}

// SetAgentByName sets the agent command by looking up the agent name in config.
func (m *Manager) SetAgentByName(name string) bool {
	for _, a := range config.Agents {
		if a.Name == name {
			m.agentCmd = a.Command
			return true
		}
	}
	return false
}

// SpawnAgent creates and starts a new agent.
func (m *Manager) SpawnAgent(name string, role Role, workspace string) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Check if already exists
	if _, exists := m.agents[name]; exists {
		return nil, fmt.Errorf("agent %s already exists", name)
	}
	
	// Create agent
	agent := &Agent{
		ID:        name,
		Name:      name,
		Role:      role,
		State:     StateStarting,
		Workspace: workspace,
		Session:   name,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	// Create tmux session with claude
	if err := m.tmux.CreateSessionWithCommand(name, workspace, m.agentCmd); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}
	
	// Set environment variables for the agent
	m.tmux.SetEnvironment(name, "BC_AGENT_ID", name)
	m.tmux.SetEnvironment(name, "BC_AGENT_ROLE", string(role))
	m.tmux.SetEnvironment(name, "BC_WORKSPACE", workspace)
	
	// Update state
	agent.State = StateIdle
	agent.UpdatedAt = time.Now()
	m.agents[name] = agent
	
	// Save state
	m.saveState()
	
	return agent, nil
}

// StopAgent stops an agent.
func (m *Manager) StopAgent(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}
	
	// Kill tmux session
	if err := m.tmux.KillSession(name); err != nil {
		// Session might already be dead
	}
	
	agent.State = StateStopped
	agent.UpdatedAt = time.Now()
	
	m.saveState()
	
	return nil
}

// StopAll stops all agents.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for name, agent := range m.agents {
		m.tmux.KillSession(name)
		agent.State = StateStopped
		agent.UpdatedAt = time.Now()
	}
	
	m.saveState()
	return nil
}

// GetAgent returns an agent by name.
func (m *Manager) GetAgent(name string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[name]
}

// ListAgents returns all agents.
func (m *Manager) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	agents := make([]*Agent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}
	return agents
}

// RefreshState updates agent states from tmux.
func (m *Manager) RefreshState() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return err
	}
	
	// Build map of active sessions
	active := make(map[string]bool)
	for _, s := range sessions {
		active[s.Name] = true
	}
	
	// Update agent states
	for name, agent := range m.agents {
		if !active[name] && agent.State != StateStopped {
			agent.State = StateStopped
			agent.UpdatedAt = time.Now()
		}
	}
	
	return nil
}

// UpdateAgentState updates an agent's state and task.
func (m *Manager) UpdateAgentState(name string, state State, task string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}
	
	agent.State = state
	agent.Task = task
	agent.UpdatedAt = time.Now()
	
	m.saveState()
	return nil
}

// SendToAgent sends a message/command to an agent's session.
func (m *Manager) SendToAgent(name, message string) error {
	return m.tmux.SendKeys(name, message)
}

// CaptureOutput captures recent output from an agent's session.
func (m *Manager) CaptureOutput(name string, lines int) (string, error) {
	return m.tmux.Capture(name, lines)
}

// AttachToAgent returns the command to attach to an agent's session.
func (m *Manager) AttachToAgent(name string) error {
	cmd := m.tmux.AttachCmd(name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// saveState persists agent state to disk.
func (m *Manager) saveState() error {
	if m.stateDir == "" {
		return nil
	}
	
	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(m.agents, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(filepath.Join(m.stateDir, "agents.json"), data, 0644)
}

// LoadState loads agent state from disk.
func (m *Manager) LoadState() error {
	if m.stateDir == "" {
		return nil
	}
	
	data, err := os.ReadFile(filepath.Join(m.stateDir, "agents.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return json.Unmarshal(data, &m.agents)
}

// Tmux returns the underlying tmux manager.
func (m *Manager) Tmux() *tmux.Manager {
	return m.tmux
}
