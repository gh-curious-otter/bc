package remote

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace")
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	expectedPath := "/tmp/test-workspace/.bc/remote.json"
	if mgr.configPath != expectedPath {
		t.Errorf("configPath = %q, want %q", mgr.configPath, expectedPath)
	}
}

func TestManagerListHostsEmpty(t *testing.T) {
	mgr := NewManager("/tmp/test-workspace")

	hosts := mgr.ListHosts()
	if len(hosts) != 0 {
		t.Errorf("len(hosts) = %d, want 0", len(hosts))
	}
}

func TestManagerAddHost(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	host, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "Test server")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	if host.Name != "test-host" {
		t.Errorf("host.Name = %q, want %q", host.Name, "test-host")
	}
	if host.Hostname != "example.com" {
		t.Errorf("host.Hostname = %q, want %q", host.Hostname, "example.com")
	}
	if host.Port != 22 {
		t.Errorf("host.Port = %d, want 22", host.Port)
	}
	if host.User != "deploy" {
		t.Errorf("host.User = %q, want %q", host.User, "deploy")
	}
	if host.Status != StatusUnknown {
		t.Errorf("host.Status = %q, want %q", host.Status, StatusUnknown)
	}

	// Verify saved to disk
	configPath := filepath.Join(tmpDir, ".bc", "remote.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file not created")
	}
}

func TestManagerAddHostDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("first AddHost() error = %v", err)
	}

	_, err = mgr.AddHost("test-host", "other.com", 22, "user", "", "")
	if err == nil {
		t.Error("AddHost() should fail for duplicate name")
	}
}

func TestManagerAddHostDefaultPort(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	host, err := mgr.AddHost("test-host", "example.com", 0, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	if host.Port != 22 {
		t.Errorf("host.Port = %d, want 22 (default)", host.Port)
	}
}

func TestManagerGetHost(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Not found
	_, ok := mgr.GetHost("nonexistent")
	if ok {
		t.Error("GetHost() should return false for nonexistent host")
	}

	// Add and get
	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	host, ok := mgr.GetHost("test-host")
	if !ok {
		t.Error("GetHost() should return true for existing host")
	}
	if host.Hostname != "example.com" {
		t.Errorf("host.Hostname = %q, want %q", host.Hostname, "example.com")
	}
}

func TestManagerRemoveHost(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Add host
	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	// Remove
	if err := mgr.RemoveHost("test-host"); err != nil {
		t.Fatalf("RemoveHost() error = %v", err)
	}

	// Verify removed
	_, ok := mgr.GetHost("test-host")
	if ok {
		t.Error("host should not exist after removal")
	}
}

func TestManagerRemoveHostNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.RemoveHost("nonexistent")
	if err == nil {
		t.Error("RemoveHost() should fail for nonexistent host")
	}
}

func TestManagerRemoveHostWithAgents(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Add host
	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	// Spawn agent on host
	_, err = mgr.SpawnAgent(context.Background(), "test-agent", "test-host", "engineer")
	if err != nil {
		t.Fatalf("SpawnAgent() error = %v", err)
	}

	// Try to remove host (should fail)
	err = mgr.RemoveHost("test-host")
	if err == nil {
		t.Error("RemoveHost() should fail when agents are running")
	}
}

func TestManagerSpawnAgent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Add host first
	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	// Spawn agent
	agent, err := mgr.SpawnAgent(context.Background(), "test-agent", "test-host", "engineer")
	if err != nil {
		t.Fatalf("SpawnAgent() error = %v", err)
	}

	if agent.Name != "test-agent" {
		t.Errorf("agent.Name = %q, want %q", agent.Name, "test-agent")
	}
	if agent.Host != "test-host" {
		t.Errorf("agent.Host = %q, want %q", agent.Host, "test-host")
	}
	if agent.Role != "engineer" {
		t.Errorf("agent.Role = %q, want %q", agent.Role, "engineer")
	}
}

func TestManagerSpawnAgentHostNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.SpawnAgent(context.Background(), "test-agent", "nonexistent", "engineer")
	if err == nil {
		t.Error("SpawnAgent() should fail for nonexistent host")
	}
}

func TestManagerSpawnAgentDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	_, err = mgr.SpawnAgent(context.Background(), "test-agent", "test-host", "engineer")
	if err != nil {
		t.Fatalf("first SpawnAgent() error = %v", err)
	}

	_, err = mgr.SpawnAgent(context.Background(), "test-agent", "test-host", "manager")
	if err == nil {
		t.Error("SpawnAgent() should fail for duplicate agent name")
	}
}

func TestManagerStopAgent(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	_, err = mgr.SpawnAgent(context.Background(), "test-agent", "test-host", "engineer")
	if err != nil {
		t.Fatalf("SpawnAgent() error = %v", err)
	}

	// Stop agent
	if err := mgr.StopAgent(context.Background(), "test-agent"); err != nil {
		t.Fatalf("StopAgent() error = %v", err)
	}

	// Verify removed
	_, ok := mgr.GetAgent("test-agent")
	if ok {
		t.Error("agent should not exist after stop")
	}
}

func TestManagerStopAgentNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	err := mgr.StopAgent(context.Background(), "nonexistent")
	if err == nil {
		t.Error("StopAgent() should fail for nonexistent agent")
	}
}

func TestManagerListAgents(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Empty initially
	agents := mgr.ListAgents()
	if len(agents) != 0 {
		t.Errorf("len(agents) = %d, want 0", len(agents))
	}

	// Add agents
	_, _ = mgr.AddHost("host1", "h1.com", 22, "u", "", "")
	_, _ = mgr.AddHost("host2", "h2.com", 22, "u", "", "")
	_, _ = mgr.SpawnAgent(context.Background(), "agent1", "host1", "eng")
	_, _ = mgr.SpawnAgent(context.Background(), "agent2", "host2", "eng")

	agents = mgr.ListAgents()
	if len(agents) != 2 {
		t.Errorf("len(agents) = %d, want 2", len(agents))
	}
}

func TestManagerListAgentsByHost(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, _ = mgr.AddHost("host1", "h1.com", 22, "u", "", "")
	_, _ = mgr.AddHost("host2", "h2.com", 22, "u", "", "")
	_, _ = mgr.SpawnAgent(context.Background(), "agent1", "host1", "eng")
	_, _ = mgr.SpawnAgent(context.Background(), "agent2", "host1", "eng")
	_, _ = mgr.SpawnAgent(context.Background(), "agent3", "host2", "eng")

	agents := mgr.ListAgentsByHost("host1")
	if len(agents) != 2 {
		t.Errorf("len(agents) = %d, want 2", len(agents))
	}

	agents = mgr.ListAgentsByHost("host2")
	if len(agents) != 1 {
		t.Errorf("len(agents) = %d, want 1", len(agents))
	}
}

func TestManagerSSHCommand(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.AddHost("test-host", "example.com", 2222, "deploy", "/path/to/key", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	cmd, err := mgr.SSHCommand("test-host")
	if err != nil {
		t.Fatalf("SSHCommand() error = %v", err)
	}

	expected := "ssh -p 2222 -i /path/to/key deploy@example.com"
	if cmd != expected {
		t.Errorf("SSHCommand() = %q, want %q", cmd, expected)
	}
}

func TestManagerSSHCommandNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	_, err := mgr.SSHCommand("nonexistent")
	if err == nil {
		t.Error("SSHCommand() should fail for nonexistent host")
	}
}

func TestManagerLoadSave(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and save
	mgr1 := NewManager(tmpDir)
	_, _ = mgr1.AddHost("test-host", "example.com", 22, "deploy", "", "Test")
	_, _ = mgr1.SpawnAgent(context.Background(), "test-agent", "test-host", "engineer")

	// Load in new manager
	mgr2 := NewManager(tmpDir)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify host loaded
	host, ok := mgr2.GetHost("test-host")
	if !ok {
		t.Fatal("host not loaded")
	}
	if host.Hostname != "example.com" {
		t.Errorf("host.Hostname = %q, want %q", host.Hostname, "example.com")
	}

	// Verify agent loaded
	agent, ok := mgr2.GetAgent("test-agent")
	if !ok {
		t.Fatal("agent not loaded")
	}
	if agent.Role != "engineer" {
		t.Errorf("agent.Role = %q, want %q", agent.Role, "engineer")
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusUnknown != "unknown" {
		t.Errorf("StatusUnknown = %q, want %q", StatusUnknown, "unknown")
	}
	if StatusConnected != "connected" {
		t.Errorf("StatusConnected = %q, want %q", StatusConnected, "connected")
	}
	if StatusUnreachable != "unreachable" {
		t.Errorf("StatusUnreachable = %q, want %q", StatusUnreachable, "unreachable")
	}
	if StatusError != "error" {
		t.Errorf("StatusError = %q, want %q", StatusError, "error")
	}
}

func TestManagerTestConnection(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Add host
	_, err := mgr.AddHost("test-host", "example.com", 22, "deploy", "", "")
	if err != nil {
		t.Fatalf("AddHost() error = %v", err)
	}

	// Test connection
	ctx := context.Background()
	if err := mgr.TestConnection(ctx, "test-host"); err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}

	// Verify status updated
	host, ok := mgr.GetHost("test-host")
	if !ok {
		t.Fatal("host not found after TestConnection")
	}
	if host.Status != StatusConnected {
		t.Errorf("host.Status = %q, want %q", host.Status, StatusConnected)
	}
}

func TestManagerTestConnectionHostNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	ctx := context.Background()
	err := mgr.TestConnection(ctx, "nonexistent")
	if err == nil {
		t.Error("TestConnection() should fail for nonexistent host")
	}
}

func TestManagerListHostsWithHosts(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Add multiple hosts
	_, _ = mgr.AddHost("host1", "h1.example.com", 22, "user1", "", "Host 1")
	_, _ = mgr.AddHost("host2", "h2.example.com", 2222, "user2", "/key", "Host 2")
	_, _ = mgr.AddHost("host3", "h3.example.com", 22, "user3", "", "Host 3")

	hosts := mgr.ListHosts()
	if len(hosts) != 3 {
		t.Errorf("len(hosts) = %d, want 3", len(hosts))
	}

	// Verify hosts are in the list
	hostNames := make(map[string]bool)
	for _, h := range hosts {
		hostNames[h.Name] = true
	}
	for _, name := range []string{"host1", "host2", "host3"} {
		if !hostNames[name] {
			t.Errorf("host %q not found in ListHosts()", name)
		}
	}
}

func TestManagerLoadNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)

	// Load without any saved config should succeed (empty state)
	err := mgr.Load()
	if err != nil {
		t.Errorf("Load() on empty workspace should not error: %v", err)
	}
}

func TestManagerLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create invalid JSON config
	configDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(configDir, "remote.json")
	if err := os.WriteFile(configPath, []byte("invalid json"), 0600); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(tmpDir)
	err := mgr.Load()
	if err == nil {
		t.Error("Load() should fail for invalid JSON")
	}
}
