package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManager(t *testing.T) {
	mgr := NewManager("/tmp/test-state")
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}

	expectedPath := "/tmp/test-state/remote.toml"
	if mgr.configPath != expectedPath {
		t.Errorf("configPath = %q, want %q", mgr.configPath, expectedPath)
	}
}

func TestHostAddress(t *testing.T) {
	tests := []struct {
		host *Host
		want string
	}{
		{
			host: &Host{Hostname: "example.com", Port: 22},
			want: "example.com:22",
		},
		{
			host: &Host{Hostname: "example.com", Port: 2222},
			want: "example.com:2222",
		},
		{
			host: &Host{Hostname: "example.com", Port: 0},
			want: "example.com:22",
		},
	}

	for _, tt := range tests {
		got := tt.host.Address()
		if got != tt.want {
			t.Errorf("Address() = %q, want %q", got, tt.want)
		}
	}
}

func TestManagerList(t *testing.T) {
	mgr := NewManager("/tmp/test-state")
	hosts := mgr.List()
	if len(hosts) != 0 {
		t.Errorf("len(hosts) = %d, want 0", len(hosts))
	}
}

func TestManagerGet(t *testing.T) {
	mgr := NewManager("/tmp/test-state")

	_, ok := mgr.Get("nonexistent")
	if ok {
		t.Error("Get should return false for nonexistent host")
	}
}

func TestManagerAddRemove(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "remote-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // cleanup in test

	mgr := NewManager(tmpDir)

	// Add host
	host := &Host{
		Name:     "test-host",
		Hostname: "example.com",
		User:     "deploy",
		Port:     22,
	}

	if err := mgr.Add(host); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Verify host exists
	got, ok := mgr.Get("test-host")
	if !ok {
		t.Fatal("host not found after add")
	}
	if got.Hostname != "example.com" {
		t.Errorf("hostname = %q, want %q", got.Hostname, "example.com")
	}

	// List hosts
	hosts := mgr.List()
	if len(hosts) != 1 {
		t.Errorf("len(hosts) = %d, want 1", len(hosts))
	}

	// Add duplicate should fail
	if err := mgr.Add(host); err == nil {
		t.Error("Add() should fail for duplicate host")
	}

	// Remove host
	if err := mgr.Remove("test-host"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify host removed
	_, ok = mgr.Get("test-host")
	if ok {
		t.Error("host should not exist after remove")
	}

	// Remove nonexistent should fail
	if err := mgr.Remove("nonexistent"); err == nil {
		t.Error("Remove() should fail for nonexistent host")
	}
}

func TestManagerAddValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remote-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // cleanup in test

	mgr := NewManager(tmpDir)

	tests := []struct {
		host    *Host
		name    string
		wantErr bool
	}{
		{
			name:    "missing name",
			host:    &Host{Hostname: "example.com", User: "deploy"},
			wantErr: true,
		},
		{
			name:    "missing hostname",
			host:    &Host{Name: "test", User: "deploy"},
			wantErr: true,
		},
		{
			name:    "missing user",
			host:    &Host{Name: "test", Hostname: "example.com"},
			wantErr: true,
		},
		{
			name:    "valid host",
			host:    &Host{Name: "test", Hostname: "example.com", User: "deploy"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset manager for each test
			mgr.hosts = make(map[string]*Host)

			err := mgr.Add(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestManagerSaveLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "remote-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir) //nolint:errcheck // cleanup in test

	// Create and save
	mgr1 := NewManager(tmpDir)
	host := &Host{
		Name:     "test-host",
		Hostname: "example.com",
		User:     "deploy",
		Port:     2222,
		KeyPath:  "~/.ssh/deploy_key",
	}
	if err := mgr1.Add(host); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Verify config file exists
	configPath := filepath.Join(tmpDir, "remote.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	// Create new manager and load
	mgr2 := NewManager(tmpDir)
	if err := mgr2.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Verify host loaded
	got, ok := mgr2.Get("test-host")
	if !ok {
		t.Fatal("host not found after load")
	}
	if got.Hostname != "example.com" {
		t.Errorf("hostname = %q, want %q", got.Hostname, "example.com")
	}
	if got.Port != 2222 {
		t.Errorf("port = %d, want %d", got.Port, 2222)
	}
}

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot get home dir")
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/.ssh/key", filepath.Join(home, ".ssh/key")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		got := expandPath(tt.input)
		if got != tt.want {
			t.Errorf("expandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParseHostsConfig(t *testing.T) {
	config := `# Remote hosts
[[hosts]]
name = "dev"
hostname = "dev.example.com"
user = "deploy"
port = 22
key_path = "~/.ssh/dev_key"

[[hosts]]
name = "prod"
hostname = "prod.example.com"
user = "admin"
port = 2222
`

	hosts := parseHostsConfig(config)
	if len(hosts) != 2 {
		t.Fatalf("len(hosts) = %d, want 2", len(hosts))
	}

	// Check first host
	if hosts[0].Name != "dev" {
		t.Errorf("hosts[0].Name = %q, want %q", hosts[0].Name, "dev")
	}
	if hosts[0].Port != 22 {
		t.Errorf("hosts[0].Port = %d, want %d", hosts[0].Port, 22)
	}

	// Check second host
	if hosts[1].Name != "prod" {
		t.Errorf("hosts[1].Name = %q, want %q", hosts[1].Name, "prod")
	}
	if hosts[1].Port != 2222 {
		t.Errorf("hosts[1].Port = %d, want %d", hosts[1].Port, 2222)
	}
}

func TestStateConstants(t *testing.T) {
	if StateOnline != "online" {
		t.Errorf("StateOnline = %q, want %q", StateOnline, "online")
	}
	if StateOffline != "offline" {
		t.Errorf("StateOffline = %q, want %q", StateOffline, "offline")
	}
	if StateUnknown != "unknown" {
		t.Errorf("StateUnknown = %q, want %q", StateUnknown, "unknown")
	}
}
