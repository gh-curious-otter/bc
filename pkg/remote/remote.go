// Package remote implements remote agent execution for bc.
//
// Remote execution allows spawning and managing agents on remote machines
// via SSH or cloud providers. This enables distributed agent workloads
// across multiple machines, cloud instances, or CI/CD environments.
//
// Features:
//   - SSH-based remote execution
//   - Remote host management
//   - Connection testing and health checks
//   - Secure credential management
//
// Issue #1219: Remote agent execution for Phase 3 Enterprise
package remote

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// Host states
const (
	StateOnline  = "online"
	StateOffline = "offline"
	StateUnknown = "unknown"
)

// Host represents a remote host configuration
type Host struct {
	LastChecked *time.Time `toml:"-" json:"lastChecked,omitempty"`
	Name        string     `toml:"name" json:"name"`
	Hostname    string     `toml:"hostname" json:"hostname"`
	User        string     `toml:"user" json:"user"`
	KeyPath     string     `toml:"key_path,omitempty" json:"keyPath,omitempty"`
	Password    string     `toml:"-" json:"-"` // Never serialize password
	State       string     `toml:"-" json:"state,omitempty"`
	Error       string     `toml:"-" json:"error,omitempty"`
	Port        int        `toml:"port,omitempty" json:"port,omitempty"`
}

// Address returns the SSH address (host:port)
func (h *Host) Address() string {
	port := h.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s:%d", h.Hostname, port)
}

// ConnectionResult contains the result of a connection test
type ConnectionResult struct {
	Host      string        `json:"host"`
	Error     string        `json:"error,omitempty"`
	BCVersion string        `json:"bcVersion,omitempty"`
	Latency   time.Duration `json:"latency"`
	Success   bool          `json:"success"`
}

// Manager manages remote hosts
type Manager struct {
	hosts      map[string]*Host
	configPath string
}

// NewManager creates a new remote manager
func NewManager(stateDir string) *Manager {
	return &Manager{
		configPath: filepath.Join(stateDir, "remote.toml"),
		hosts:      make(map[string]*Host),
	}
}

// Load loads remote hosts from config
func (m *Manager) Load() error {
	// Check if config exists
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return nil // No remote hosts configured
	}

	// Parse TOML config
	data, err := os.ReadFile(m.configPath) //nolint:gosec // remote.toml is internal config
	if err != nil {
		return fmt.Errorf("failed to read remote config: %w", err)
	}

	// Simple TOML parsing for hosts
	// Format: [[hosts]]\nname = "..."\nhostname = "..."\n...
	hosts := parseHostsConfig(string(data))
	for _, h := range hosts {
		m.hosts[h.Name] = h
	}

	return nil
}

// parseHostsConfig parses a simple TOML hosts configuration
func parseHostsConfig(data string) []*Host {
	var hosts []*Host
	var current *Host

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[[hosts]]" {
			if current != nil {
				hosts = append(hosts, current)
			}
			current = &Host{Port: 22}
			continue
		}

		if current == nil {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, "\"")

		switch key {
		case "name":
			current.Name = value
		case "hostname":
			current.Hostname = value
		case "port":
			fmt.Sscanf(value, "%d", &current.Port) //nolint:errcheck // best effort
		case "user":
			current.User = value
		case "key_path":
			current.KeyPath = expandPath(value)
		}
	}

	if current != nil && current.Name != "" {
		hosts = append(hosts, current)
	}

	return hosts
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

// Save saves remote hosts to config
func (m *Manager) Save() error {
	// Create directory if needed
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate TOML content
	var buf strings.Builder
	buf.WriteString("# Remote hosts configuration\n")
	buf.WriteString("# Managed by bc remote commands\n\n")

	for _, h := range m.hosts {
		buf.WriteString("[[hosts]]\n")
		fmt.Fprintf(&buf, "name = %q\n", h.Name)
		fmt.Fprintf(&buf, "hostname = %q\n", h.Hostname)
		if h.Port != 0 && h.Port != 22 {
			fmt.Fprintf(&buf, "port = %d\n", h.Port)
		}
		fmt.Fprintf(&buf, "user = %q\n", h.User)
		if h.KeyPath != "" {
			fmt.Fprintf(&buf, "key_path = %q\n", h.KeyPath)
		}
		buf.WriteString("\n")
	}

	if err := os.WriteFile(m.configPath, []byte(buf.String()), 0600); err != nil {
		return fmt.Errorf("failed to write remote config: %w", err)
	}

	return nil
}

// List returns all configured hosts
func (m *Manager) List() []*Host {
	hosts := make([]*Host, 0, len(m.hosts))
	for _, h := range m.hosts {
		hosts = append(hosts, h)
	}
	return hosts
}

// Get returns a host by name
func (m *Manager) Get(name string) (*Host, bool) {
	h, ok := m.hosts[name]
	return h, ok
}

// Add adds a new remote host
func (m *Manager) Add(host *Host) error {
	if host.Name == "" {
		return fmt.Errorf("host name is required")
	}
	if host.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	if host.User == "" {
		return fmt.Errorf("user is required")
	}

	if _, exists := m.hosts[host.Name]; exists {
		return fmt.Errorf("host %q already exists", host.Name)
	}

	if host.Port == 0 {
		host.Port = 22
	}

	m.hosts[host.Name] = host
	return m.Save()
}

// Remove removes a remote host
func (m *Manager) Remove(name string) error {
	if _, ok := m.hosts[name]; !ok {
		return fmt.Errorf("host %q not found", name)
	}

	delete(m.hosts, name)
	return m.Save()
}

// Test tests connection to a remote host
func (m *Manager) Test(ctx context.Context, name string) (*ConnectionResult, error) {
	host, ok := m.hosts[name]
	if !ok {
		return nil, fmt.Errorf("host %q not found", name)
	}

	result := &ConnectionResult{
		Host: name,
	}

	// Create SSH client config
	config, err := m.sshConfig(host)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	// Connect with timeout
	start := time.Now()
	client, err := sshDial(ctx, "tcp", host.Address(), config)
	if err != nil {
		result.Error = err.Error()
		host.State = StateOffline
		host.Error = err.Error()
		return result, nil
	}
	defer client.Close() //nolint:errcheck // best effort cleanup

	result.Latency = time.Since(start)
	result.Success = true

	// Try to get bc version
	session, err := client.NewSession()
	if err == nil {
		defer session.Close() //nolint:errcheck // best effort cleanup
		output, err := session.Output("bc version 2>/dev/null || echo 'not installed'")
		if err == nil {
			result.BCVersion = strings.TrimSpace(string(output))
		}
	}

	now := time.Now()
	host.State = StateOnline
	host.LastChecked = &now
	host.Error = ""

	return result, nil
}

// sshConfig creates SSH client config for a host
func (m *Manager) sshConfig(host *Host) (*ssh.ClientConfig, error) {
	var authMethods []ssh.AuthMethod

	// Try key-based auth first
	if host.KeyPath != "" {
		key, err := os.ReadFile(host.KeyPath) //nolint:gosec // user-specified key path
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Fall back to password if set
	if host.Password != "" {
		authMethods = append(authMethods, ssh.Password(host.Password))
	}

	// Try SSH agent
	if len(authMethods) == 0 {
		// TODO: Add SSH agent support
		return nil, fmt.Errorf("no authentication method available (set key_path or use ssh-agent)")
	}

	config := &ssh.ClientConfig{
		User:            host.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // TODO: proper host key verification
		Timeout:         10 * time.Second,
	}

	return config, nil
}

// sshDial wraps ssh.Dial with context support
func sshDial(ctx context.Context, network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	done := make(chan struct{})
	var client *ssh.Client
	var err error

	go func() {
		client, err = ssh.Dial(network, addr, config)
		close(done)
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return client, err
	}
}

// Exec executes a command on a remote host
func (m *Manager) Exec(ctx context.Context, name string, command string) (string, error) {
	host, ok := m.hosts[name]
	if !ok {
		return "", fmt.Errorf("host %q not found", name)
	}

	config, err := m.sshConfig(host)
	if err != nil {
		return "", err
	}

	client, err := sshDial(ctx, "tcp", host.Address(), config)
	if err != nil {
		return "", fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close() //nolint:errcheck // best effort cleanup

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close() //nolint:errcheck // best effort cleanup

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}
