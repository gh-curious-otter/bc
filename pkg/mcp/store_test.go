package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".bc"), 0750); err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestAddAndGet(t *testing.T) {
	s := setupTestStore(t)

	cfg := &ServerConfig{
		Name:      "github",
		Transport: TransportStdio,
		Command:   "npx",
		Args:      []string{"@modelcontextprotocol/server-github"},
		Env:       map[string]string{"GITHUB_TOKEN": "${secret:GH_TOKEN}"},
		Enabled:   true,
	}
	if err := s.Add(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("github")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected config, got nil")
	}
	if got.Name != "github" {
		t.Errorf("name = %q, want %q", got.Name, "github")
	}
	if got.Transport != TransportStdio {
		t.Errorf("transport = %q, want %q", got.Transport, TransportStdio)
	}
	if got.Command != "npx" {
		t.Errorf("command = %q, want %q", got.Command, "npx")
	}
	if len(got.Args) != 1 || got.Args[0] != "@modelcontextprotocol/server-github" {
		t.Errorf("args = %v, want [\"@modelcontextprotocol/server-github\"]", got.Args)
	}
	if got.Env["GITHUB_TOKEN"] != "${secret:GH_TOKEN}" {
		t.Errorf("env GITHUB_TOKEN = %q, want %q", got.Env["GITHUB_TOKEN"], "${secret:GH_TOKEN}")
	}
	if !got.Enabled {
		t.Error("expected enabled = true")
	}
}

func TestGetNotFound(t *testing.T) {
	s := setupTestStore(t)

	got, err := s.Get("nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}

func TestList(t *testing.T) {
	s := setupTestStore(t)

	// Empty list
	configs, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 0 {
		t.Errorf("expected 0 configs, got %d", len(configs))
	}

	// Add two configs
	if err := s.Add(&ServerConfig{Name: "beta", Transport: TransportStdio, Command: "cmd-b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(&ServerConfig{Name: "alpha", Transport: TransportSSE, URL: "https://example.com"}); err != nil {
		t.Fatal(err)
	}

	configs, err = s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 2 {
		t.Fatalf("expected 2 configs, got %d", len(configs))
	}
	// Sorted by name
	if configs[0].Name != "alpha" {
		t.Errorf("first config name = %q, want %q", configs[0].Name, "alpha")
	}
	if configs[1].Name != "beta" {
		t.Errorf("second config name = %q, want %q", configs[1].Name, "beta")
	}
}

func TestRemove(t *testing.T) {
	s := setupTestStore(t)

	if err := s.Add(&ServerConfig{Name: "test", Transport: TransportStdio, Command: "cmd"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("test"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Error("expected nil after remove")
	}
}

func TestRemoveNotFound(t *testing.T) {
	s := setupTestStore(t)

	err := s.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error for removing nonexistent server")
	}
}

func TestSetEnabled(t *testing.T) {
	s := setupTestStore(t)

	if err := s.Add(&ServerConfig{Name: "srv", Transport: TransportStdio, Command: "cmd", Enabled: true}); err != nil {
		t.Fatal(err)
	}

	if err := s.SetEnabled("srv", false); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("srv")
	if err != nil {
		t.Fatal(err)
	}
	if got.Enabled {
		t.Error("expected enabled = false after disable")
	}

	if err := s.SetEnabled("srv", true); err != nil {
		t.Fatal(err)
	}
	got, err = s.Get("srv")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled {
		t.Error("expected enabled = true after enable")
	}
}

func TestSetEnabledNotFound(t *testing.T) {
	s := setupTestStore(t)

	err := s.SetEnabled("nonexistent", true)
	if err == nil {
		t.Fatal("expected error for enabling nonexistent server")
	}
}

func TestAddDuplicate(t *testing.T) {
	s := setupTestStore(t)

	cfg := &ServerConfig{Name: "dup", Transport: TransportStdio, Command: "cmd"}
	if err := s.Add(cfg); err != nil {
		t.Fatal(err)
	}
	err := s.Add(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate add")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestValidation(t *testing.T) {
	s := setupTestStore(t)

	tests := []struct {
		name string
		cfg  *ServerConfig
	}{
		{"empty name", &ServerConfig{Transport: TransportStdio, Command: "cmd"}},
		{"stdio no command", &ServerConfig{Name: "bad", Transport: TransportStdio}},
		{"sse no url", &ServerConfig{Name: "bad", Transport: TransportSSE}},
		{"invalid transport", &ServerConfig{Name: "bad", Transport: "grpc"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := s.Add(tc.cfg); err == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestSSETransport(t *testing.T) {
	s := setupTestStore(t)

	cfg := &ServerConfig{
		Name:      "remote",
		Transport: TransportSSE,
		URL:       "https://api.example.com/mcp",
		Enabled:   true,
	}
	if err := s.Add(cfg); err != nil {
		t.Fatal(err)
	}

	got, err := s.Get("remote")
	if err != nil {
		t.Fatal(err)
	}
	if got.Transport != TransportSSE {
		t.Errorf("transport = %q, want %q", got.Transport, TransportSSE)
	}
	if got.URL != "https://api.example.com/mcp" {
		t.Errorf("url = %q, want %q", got.URL, "https://api.example.com/mcp")
	}
}
