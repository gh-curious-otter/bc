package container

import (
	"testing"

	"github.com/rpuneet/bc/pkg/workspace"
)

func TestConfigFromWorkspace_Defaults(t *testing.T) {
	cfg := ConfigFromWorkspace(workspace.DockerRuntimeConfig{})

	if cfg.Image != "bc-agent-claude:latest" {
		t.Errorf("Image = %q, want bc-agent-claude:latest", cfg.Image)
	}
	if cfg.CPUs != 2.0 {
		t.Errorf("CPUs = %f, want 2.0", cfg.CPUs)
	}
	if cfg.MemoryMB != 2048 {
		t.Errorf("MemoryMB = %d, want 2048", cfg.MemoryMB)
	}
	if cfg.Network != "bridge" {
		t.Errorf("Network = %q, want bridge", cfg.Network)
	}
}

func TestConfigFromWorkspace_CustomValues(t *testing.T) {
	cfg := ConfigFromWorkspace(workspace.DockerRuntimeConfig{
		Image:       "custom-image:v1",
		Network:     "host",
		CPUs:        4.0,
		MemoryMB:    4096,
		ExtraMounts: []string{"/data:/data:ro"},
	})

	if cfg.Image != "custom-image:v1" {
		t.Errorf("Image = %q, want custom-image:v1", cfg.Image)
	}
	if cfg.CPUs != 4.0 {
		t.Errorf("CPUs = %f, want 4.0", cfg.CPUs)
	}
	if cfg.MemoryMB != 4096 {
		t.Errorf("MemoryMB = %d, want 4096", cfg.MemoryMB)
	}
	if cfg.Network != "host" {
		t.Errorf("Network = %q, want host", cfg.Network)
	}
	if len(cfg.ExtraMounts) != 1 || cfg.ExtraMounts[0] != "/data:/data:ro" {
		t.Errorf("ExtraMounts = %v, want [\"/data:/data:ro\"]", cfg.ExtraMounts)
	}
}

func TestConfigFromWorkspace_PartialOverride(t *testing.T) {
	// Only set image, rest should default
	cfg := ConfigFromWorkspace(workspace.DockerRuntimeConfig{
		Image: "my-agent:latest",
	})

	if cfg.Image != "my-agent:latest" {
		t.Errorf("Image = %q, want my-agent:latest", cfg.Image)
	}
	if cfg.CPUs != 2.0 {
		t.Errorf("CPUs should default to 2.0, got %f", cfg.CPUs)
	}
	if cfg.MemoryMB != 2048 {
		t.Errorf("MemoryMB should default to 2048, got %d", cfg.MemoryMB)
	}
	if cfg.Network != "bridge" {
		t.Errorf("Network should default to bridge, got %q", cfg.Network)
	}
}

func TestContainerName(t *testing.T) {
	b := &Backend{
		prefix:        "bc-",
		workspaceHash: "a1b2c3",
	}

	got := b.containerName("alice")
	want := "bc-a1b2c3-alice"
	if got != want {
		t.Errorf("containerName = %q, want %q", got, want)
	}
}

func TestContainerName_SpecialChars(t *testing.T) {
	b := &Backend{
		prefix:        "bc-",
		workspaceHash: "ff00ff",
	}

	// Agent names with hyphens and underscores
	got := b.containerName("eng-01")
	want := "bc-ff00ff-eng-01"
	if got != want {
		t.Errorf("containerName = %q, want %q", got, want)
	}

	got = b.containerName("test_agent")
	want = "bc-ff00ff-test_agent"
	if got != want {
		t.Errorf("containerName = %q, want %q", got, want)
	}
}

func TestSessionName(t *testing.T) {
	b := &Backend{
		prefix:        "bc-",
		workspaceHash: "abc123",
	}

	got := b.SessionName("worker")
	want := "bc-abc123-worker"
	if got != want {
		t.Errorf("SessionName = %q, want %q", got, want)
	}
}

func TestImageForTool_Default(t *testing.T) {
	b := &Backend{
		cfg: Config{Image: "bc-agent-claude:latest"},
	}

	// Empty tool name returns default image
	got := b.imageForTool("")
	if got != "bc-agent-claude:latest" {
		t.Errorf("imageForTool(\"\") = %q, want bc-agent-claude:latest", got)
	}
}

func TestImageForTool_Convention(t *testing.T) {
	b := &Backend{
		cfg: Config{Image: "bc-agent-claude:latest"},
	}

	// Unknown tool without registry falls back to convention
	got := b.imageForTool("gemini")
	want := "bc-agent-gemini:latest"
	if got != want {
		t.Errorf("imageForTool(\"gemini\") = %q, want %q", got, want)
	}
}

func TestImageForTool_FallbackToConfig(t *testing.T) {
	b := &Backend{
		cfg: Config{Image: "custom-default:v2"},
	}

	// Tool that doesn't match convention pattern
	got := b.imageForTool("unknown-tool")
	// Should return convention-based name
	want := "bc-agent-unknown-tool:latest"
	if got != want {
		t.Errorf("imageForTool(\"unknown-tool\") = %q, want %q", got, want)
	}
}
