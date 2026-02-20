// Package remote implements remote agent execution via SSH.
//
// Enables spawning and managing agents on remote machines for
// distributed workloads and enterprise deployments.
//
// Issue #1219: Phase 3 Enterprise - Remote agent execution
package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Host represents a remote host configuration
//
//nolint:govet // JSON field order is more important than struct alignment
type Host struct {
	Name        string    `json:"name"`
	Hostname    string    `json:"hostname"`
	Port        int       `json:"port"`
	User        string    `json:"user"`
	KeyPath     string    `json:"key_path,omitempty"`
	Description string    `json:"description,omitempty"`
	AddedAt     time.Time `json:"added_at"`
	LastUsed    time.Time `json:"last_used,omitempty"`
	Status      string    `json:"status"`
}

// HostStatus constants
const (
	StatusUnknown     = "unknown"
	StatusConnected   = "connected"
	StatusUnreachable = "unreachable"
	StatusError       = "error"
)

// RemoteAgent represents an agent running on a remote host
//
//nolint:govet // JSON field order is more important than struct alignment
type RemoteAgent struct {
	Name      string    `json:"name"`
	Host      string    `json:"host"`
	Role      string    `json:"role"`
	PID       int       `json:"pid,omitempty"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
}

// Manager manages remote hosts and agents
type Manager struct {
	hosts      map[string]*Host
	agents     map[string]*RemoteAgent
	configPath string
}

// NewManager creates a new remote manager
func NewManager(workspaceDir string) *Manager {
	return &Manager{
		hosts:      make(map[string]*Host),
		agents:     make(map[string]*RemoteAgent),
		configPath: filepath.Join(workspaceDir, ".bc", "remote.json"),
	}
}

// Load loads remote configuration from disk
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath) //nolint:gosec // internal config file
	if os.IsNotExist(err) {
		return nil // No config yet
	}
	if err != nil {
		return fmt.Errorf("failed to read remote config: %w", err)
	}

	var config struct {
		Hosts  []*Host        `json:"hosts"`
		Agents []*RemoteAgent `json:"agents"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse remote config: %w", err)
	}

	for _, h := range config.Hosts {
		m.hosts[h.Name] = h
	}
	for _, a := range config.Agents {
		m.agents[a.Name] = a
	}

	return nil
}

// Save saves remote configuration to disk
func (m *Manager) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	hosts := make([]*Host, 0, len(m.hosts))
	for _, h := range m.hosts {
		hosts = append(hosts, h)
	}

	agents := make([]*RemoteAgent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}

	config := struct {
		Hosts  []*Host        `json:"hosts"`
		Agents []*RemoteAgent `json:"agents"`
	}{
		Hosts:  hosts,
		Agents: agents,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal remote config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write remote config: %w", err)
	}

	return nil
}

// AddHost adds a new remote host
func (m *Manager) AddHost(name, hostname string, port int, user, keyPath, description string) (*Host, error) {
	if _, exists := m.hosts[name]; exists {
		return nil, fmt.Errorf("host %q already exists", name)
	}

	if port == 0 {
		port = 22
	}

	host := &Host{
		Name:        name,
		Hostname:    hostname,
		Port:        port,
		User:        user,
		KeyPath:     keyPath,
		Description: description,
		AddedAt:     time.Now(),
		Status:      StatusUnknown,
	}

	m.hosts[name] = host

	if err := m.Save(); err != nil {
		delete(m.hosts, name)
		return nil, err
	}

	return host, nil
}

// RemoveHost removes a remote host
func (m *Manager) RemoveHost(name string) error {
	if _, exists := m.hosts[name]; !exists {
		return fmt.Errorf("host %q not found", name)
	}

	// Check for running agents on this host
	for _, a := range m.agents {
		if a.Host == name {
			return fmt.Errorf("host %q has running agents; stop them first", name)
		}
	}

	delete(m.hosts, name)
	return m.Save()
}

// GetHost returns a host by name
func (m *Manager) GetHost(name string) (*Host, bool) {
	h, ok := m.hosts[name]
	return h, ok
}

// ListHosts returns all configured hosts
func (m *Manager) ListHosts() []*Host {
	hosts := make([]*Host, 0, len(m.hosts))
	for _, h := range m.hosts {
		hosts = append(hosts, h)
	}
	return hosts
}

// TestConnection tests connectivity to a remote host
func (m *Manager) TestConnection(_ context.Context, name string) error {
	host, ok := m.hosts[name]
	if !ok {
		return fmt.Errorf("host %q not found", name)
	}

	// TODO: Implement actual SSH connection test
	// For now, just mark as connected
	host.Status = StatusConnected
	host.LastUsed = time.Now()

	return m.Save()
}

// SpawnAgent spawns an agent on a remote host
func (m *Manager) SpawnAgent(_ context.Context, agentName, hostName, role string) (*RemoteAgent, error) {
	host, ok := m.hosts[hostName]
	if !ok {
		return nil, fmt.Errorf("host %q not found", hostName)
	}

	if _, exists := m.agents[agentName]; exists {
		return nil, fmt.Errorf("agent %q already exists", agentName)
	}

	// TODO: Implement actual SSH agent spawning
	// For now, create a placeholder
	agent := &RemoteAgent{
		Name:      agentName,
		Host:      hostName,
		Role:      role,
		Status:    "starting",
		StartedAt: time.Now(),
	}

	m.agents[agentName] = agent
	host.LastUsed = time.Now()

	if err := m.Save(); err != nil {
		delete(m.agents, agentName)
		return nil, err
	}

	return agent, nil
}

// StopAgent stops a remote agent
func (m *Manager) StopAgent(_ context.Context, name string) error {
	agent, ok := m.agents[name]
	if !ok {
		return fmt.Errorf("remote agent %q not found", name)
	}

	// TODO: Implement actual SSH agent stop
	agent.Status = "stopped"

	delete(m.agents, name)
	return m.Save()
}

// GetAgent returns a remote agent by name
func (m *Manager) GetAgent(name string) (*RemoteAgent, bool) {
	a, ok := m.agents[name]
	return a, ok
}

// ListAgents returns all remote agents
func (m *Manager) ListAgents() []*RemoteAgent {
	agents := make([]*RemoteAgent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}
	return agents
}

// ListAgentsByHost returns agents on a specific host
func (m *Manager) ListAgentsByHost(hostName string) []*RemoteAgent {
	var agents []*RemoteAgent
	for _, a := range m.agents {
		if a.Host == hostName {
			agents = append(agents, a)
		}
	}
	return agents
}

// SSHCommand returns the SSH command to connect to a host
func (m *Manager) SSHCommand(name string) (string, error) {
	host, ok := m.hosts[name]
	if !ok {
		return "", fmt.Errorf("host %q not found", name)
	}

	cmd := fmt.Sprintf("ssh -p %d", host.Port)
	if host.KeyPath != "" {
		cmd += fmt.Sprintf(" -i %s", host.KeyPath)
	}
	cmd += fmt.Sprintf(" %s@%s", host.User, host.Hostname)

	return cmd, nil
}
