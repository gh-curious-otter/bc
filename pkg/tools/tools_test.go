package tools

import (
	"context"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if r.tools == nil {
		t.Fatal("NewRegistry tools map is nil")
	}
}

func TestRegister(t *testing.T) {
	r := NewRegistry()

	// Register valid tool
	tool := &Tool{Name: "test", Command: "echo hello", Enabled: true}
	if err := r.Register(tool); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify registration
	got, ok := r.Get("test")
	if !ok {
		t.Fatal("Get failed to find registered tool")
	}
	if got.Name != "test" {
		t.Errorf("got name %q, want %q", got.Name, "test")
	}

	// Test missing name
	err := r.Register(&Tool{Command: "cmd"})
	if err == nil {
		t.Error("Register with missing name should fail")
	}

	// Test missing command
	err = r.Register(&Tool{Name: "bad"})
	if err == nil {
		t.Error("Register with missing command should fail")
	}
}

func TestList(t *testing.T) {
	r := NewRegistry()

	// Register some tools
	tools := []*Tool{
		{Name: "tool1", Command: "cmd1", Enabled: true},
		{Name: "tool2", Command: "cmd2", Enabled: false},
		{Name: "tool3", Command: "cmd3", Enabled: true},
	}

	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			t.Fatalf("Register failed: %v", err)
		}
	}

	// Test List
	list := r.List()
	if len(list) != 3 {
		t.Errorf("List returned %d tools, want 3", len(list))
	}

	// Test ListEnabled
	enabled := r.ListEnabled()
	if len(enabled) != 2 {
		t.Errorf("ListEnabled returned %d tools, want 2", len(enabled))
	}
}

func TestEnableDisable(t *testing.T) {
	r := NewRegistry()

	tool := &Tool{Name: "test", Command: "cmd", Enabled: false}
	if err := r.Register(tool); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Enable
	if err := r.Enable("test"); err != nil {
		t.Errorf("Enable failed: %v", err)
	}
	got, _ := r.Get("test")
	if !got.Enabled {
		t.Error("tool should be enabled")
	}

	// Disable
	if err := r.Disable("test"); err != nil {
		t.Errorf("Disable failed: %v", err)
	}
	got, _ = r.Get("test")
	if got.Enabled {
		t.Error("tool should be disabled")
	}

	// Test not found
	if err := r.Enable("nonexistent"); err == nil {
		t.Error("Enable nonexistent tool should fail")
	}
	if err := r.Disable("nonexistent"); err == nil {
		t.Error("Disable nonexistent tool should fail")
	}
}

func TestExec(t *testing.T) {
	r := NewRegistry()

	// Register echo tool
	tool := &Tool{Name: "echo", Command: "echo", Enabled: true}
	if err := r.Register(tool); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Execute
	ctx := context.Background()
	result, err := r.Exec(ctx, "echo", "hello", "world")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if result.Output != "hello world\n" {
		t.Errorf("Output = %q, want %q", result.Output, "hello world\n")
	}

	// Test not found
	_, err = r.Exec(ctx, "nonexistent")
	if err == nil {
		t.Error("Exec nonexistent tool should fail")
	}

	// Test disabled tool
	tool2 := &Tool{Name: "disabled", Command: "echo", Enabled: false}
	_ = r.Register(tool2)
	_, err = r.Exec(ctx, "disabled")
	if err == nil {
		t.Error("Exec disabled tool should fail")
	}
}

func TestIsInstalled(t *testing.T) {
	// Test with common command
	tool := &Tool{Name: "echo", Command: "echo hello"}
	if !tool.IsInstalled() {
		t.Error("echo should be installed")
	}

	// Test with nonexistent command
	tool2 := &Tool{Name: "fake", Command: "nonexistent-cmd-12345"}
	if tool2.IsInstalled() {
		t.Error("nonexistent command should not be installed")
	}
}

func TestStatus(t *testing.T) {
	tests := []struct {
		name string
		tool *Tool
		want string
	}{
		{
			name: "disabled",
			tool: &Tool{Name: "test", Command: "echo", Enabled: false},
			want: "disabled",
		},
		{
			name: "not installed",
			tool: &Tool{Name: "test", Command: "nonexistent-12345", Enabled: true},
			want: "not installed",
		},
		{
			name: "ready",
			tool: &Tool{Name: "test", Command: "echo", Enabled: true},
			want: "ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tool.Status()
			if got != tt.want {
				t.Errorf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Reset default registry for clean test
	DefaultRegistry = NewRegistry()

	tool := &Tool{Name: "global", Command: "echo", Enabled: true}
	if err := Register(tool); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, ok := Get("global")
	if !ok {
		t.Fatal("Get failed")
	}
	if got.Name != "global" {
		t.Errorf("Name = %q, want %q", got.Name, "global")
	}

	list := List()
	if len(list) != 1 {
		t.Errorf("List returned %d tools, want 1", len(list))
	}

	enabled := ListEnabled()
	if len(enabled) != 1 {
		t.Errorf("ListEnabled returned %d tools, want 1", len(enabled))
	}

	if err := Disable("global"); err != nil {
		t.Errorf("Disable failed: %v", err)
	}

	enabled = ListEnabled()
	if len(enabled) != 0 {
		t.Errorf("ListEnabled returned %d tools, want 0", len(enabled))
	}

	if err := Enable("global"); err != nil {
		t.Errorf("Enable failed: %v", err)
	}

	ctx := context.Background()
	result, err := Exec(ctx, "global", "test")
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if result.Output != "test\n" {
		t.Errorf("Output = %q, want %q", result.Output, "test\n")
	}
}
