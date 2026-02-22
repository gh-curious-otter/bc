package provider

import (
	"context"
	"strings"
)

// OpenCodeProvider implements the Provider interface for OpenCode/Crush.
// OpenCode was originally github.com/opencode-ai/opencode, now archived.
// Crush (github.com/charmbracelet/crush) is the successor.
//
// Issue #1451: OpenCode Provider Integration
type OpenCodeProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewOpenCodeProvider creates a new OpenCode provider.
func NewOpenCodeProvider() *OpenCodeProvider {
	return &OpenCodeProvider{
		name:        "opencode",
		description: "OpenCode/Crush AI coding assistant",
		command:     "crush",
		binary:      "crush",
	}
}

// Name returns the provider's unique identifier.
func (p *OpenCodeProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *OpenCodeProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *OpenCodeProvider) Command() string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *OpenCodeProvider) IsInstalled(ctx context.Context) bool {
	// Check for crush (successor) first
	if checkBinaryExists(ctx, "crush") {
		return true
	}
	// Fall back to opencode (legacy)
	return checkBinaryExists(ctx, "opencode")
}

// Version returns the installed version.
func (p *OpenCodeProvider) Version(ctx context.Context) string {
	// Try crush first
	if checkBinaryExists(ctx, "crush") {
		return getBinaryVersion(ctx, "crush", "--version")
	}
	// Fall back to opencode
	if checkBinaryExists(ctx, "opencode") {
		return getBinaryVersion(ctx, "opencode", "--version")
	}
	return ""
}

// DetectState analyzes output to determine agent state.
// Crush/OpenCode uses similar spinner patterns to Claude.
func (p *OpenCodeProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])

		// Working indicators - spinner symbols
		if strings.HasPrefix(line, "⠋") ||
			strings.HasPrefix(line, "⠙") ||
			strings.HasPrefix(line, "⠹") ||
			strings.HasPrefix(line, "⠸") ||
			strings.HasPrefix(line, "⠼") ||
			strings.HasPrefix(line, "⠴") ||
			strings.HasPrefix(line, "⠦") ||
			strings.HasPrefix(line, "⠧") ||
			strings.HasPrefix(line, "⠇") ||
			strings.HasPrefix(line, "⠏") ||
			strings.Contains(line, "thinking") ||
			strings.Contains(line, "processing") {
			return StateWorking
		}

		// Done indicators
		if strings.Contains(line, "✓") ||
			strings.Contains(line, "done") ||
			strings.Contains(line, "complete") ||
			strings.Contains(line, "finished") {
			return StateDone
		}

		// Error indicators
		if strings.Contains(line, "error") ||
			strings.Contains(line, "failed") ||
			strings.Contains(line, "✗") {
			return StateError
		}

		// Idle/prompt indicators
		if strings.HasPrefix(line, ">") ||
			strings.HasPrefix(line, "❯") ||
			strings.HasPrefix(line, "$") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure OpenCodeProvider implements Provider interface.
var _ Provider = (*OpenCodeProvider)(nil)
