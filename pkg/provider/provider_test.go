package provider

import (
	"context"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if len(r.providers) != 0 {
		t.Errorf("expected empty registry, got %d providers", len(r.providers))
	}
}

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry()
	p := NewOpenCodeProvider()
	r.Register(p)

	got, ok := r.Get("opencode")
	if !ok {
		t.Fatal("expected to find registered provider")
	}
	if got.Name() != "opencode" {
		t.Errorf("expected name 'opencode', got %q", got.Name())
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry()
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected not to find unregistered provider")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry()
	r.Register(NewOpenCodeProvider())
	r.Register(NewClaudeProvider())

	list := r.List()
	if len(list) != 2 {
		t.Errorf("expected 2 providers, got %d", len(list))
	}
}

func TestDefaultRegistryHasProviders(t *testing.T) {
	// Default registry should have built-in providers
	if len(DefaultRegistry.providers) == 0 {
		t.Error("expected default registry to have providers")
	}

	// Check for expected providers
	if _, ok := DefaultRegistry.Get("opencode"); !ok {
		t.Error("expected opencode provider in default registry")
	}
	if _, ok := DefaultRegistry.Get("claude"); !ok {
		t.Error("expected claude provider in default registry")
	}
}

func TestGetProvider(t *testing.T) {
	p, err := GetProvider("opencode")
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}
	if p.Name() != "opencode" {
		t.Errorf("expected name 'opencode', got %q", p.Name())
	}
}

func TestGetProviderNotFound(t *testing.T) {
	_, err := GetProvider("nonexistent")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestOpenCodeProvider(t *testing.T) {
	p := NewOpenCodeProvider()

	if p.Name() != "opencode" {
		t.Errorf("expected name 'opencode', got %q", p.Name())
	}
	if p.Description() == "" {
		t.Error("expected non-empty description")
	}
	if p.Command() == "" {
		t.Error("expected non-empty command")
	}
}

func TestOpenCodeDetectState(t *testing.T) {
	p := NewOpenCodeProvider()
	ctx := context.Background()
	_ = ctx // unused in DetectState but keeps pattern consistent

	tests := []struct {
		name   string
		output string
		want   State
	}{
		{
			name:   "working spinner",
			output: "⠋ Processing files...\n",
			want:   StateWorking,
		},
		{
			name:   "working thinking",
			output: "thinking about your request\n",
			want:   StateWorking,
		},
		{
			name:   "done checkmark",
			output: "✓ Task complete\n",
			want:   StateDone,
		},
		{
			name:   "done finished",
			output: "Task finished successfully\n",
			want:   StateDone,
		},
		{
			name:   "error",
			output: "error: file not found\n",
			want:   StateError,
		},
		{
			name:   "idle prompt",
			output: "> ",
			want:   StateIdle,
		},
		{
			name:   "unknown",
			output: "some random output\n",
			want:   StateUnknown,
		},
		{
			name:   "empty",
			output: "",
			want:   StateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.DetectState(tt.output)
			if got != tt.want {
				t.Errorf("DetectState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClaudeProvider(t *testing.T) {
	p := NewClaudeProvider()

	if p.Name() != "claude" {
		t.Errorf("expected name 'claude', got %q", p.Name())
	}
	if p.Description() == "" {
		t.Error("expected non-empty description")
	}
	if p.Command() == "" {
		t.Error("expected non-empty command")
	}
}

func TestClaudeDetectState(t *testing.T) {
	p := NewClaudeProvider()

	tests := []struct {
		name   string
		output string
		want   State
	}{
		{
			name:   "working spinner 1",
			output: "✻ Reading file...\n",
			want:   StateWorking,
		},
		{
			name:   "working spinner 2",
			output: "✳ Analyzing code\n",
			want:   StateWorking,
		},
		{
			name:   "working tool",
			output: "⏺ Running bash command\n",
			want:   StateWorking,
		},
		{
			name:   "idle prompt",
			output: "❯ ",
			want:   StateIdle,
		},
		{
			name:   "unknown",
			output: "some output\n",
			want:   StateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.DetectState(tt.output)
			if got != tt.want {
				t.Errorf("DetectState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListProviders(t *testing.T) {
	providers := ListProviders()
	if len(providers) < 2 {
		t.Errorf("expected at least 2 providers, got %d", len(providers))
	}
}
