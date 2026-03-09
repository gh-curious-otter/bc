package provider

import (
	"context"
	"strings"
)

// OpenClawProvider implements the Provider interface for OpenClaw CLI.
// OpenClaw is an open-source AI coding assistant.
//
// Issue #1478: OpenClaw CLI Provider Integration
type OpenClawProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewOpenClawProvider creates a new OpenClaw provider.
func NewOpenClawProvider() *OpenClawProvider {
	return &OpenClawProvider{
		name:        "openclaw",
		description: "OpenClaw AI Coding Assistant",
		command:     "openclaw --auto",
		binary:      "openclaw",
	}
}

// Name returns the provider's unique identifier.
func (p *OpenClawProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *OpenClawProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *OpenClawProvider) Command() string {
	return p.command
}

// Binary returns the executable name for LookPath/version checks.
func (p *OpenClawProvider) Binary() string {
	return p.binary
}

// InstallHint returns a human-readable install instruction.
func (p *OpenClawProvider) InstallHint() string {
	return "pip install openclaw"
}

// BuildCommand returns the full command for a given runtime context.
func (p *OpenClawProvider) BuildCommand(_ CommandOpts) string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *OpenClawProvider) IsInstalled(ctx context.Context) bool {
	return checkBinaryExists(ctx, p.binary)
}

// Version returns the installed version.
func (p *OpenClawProvider) Version(ctx context.Context) string {
	return getBinaryVersion(ctx, p.binary, "--version")
}

// DetectState analyzes output to determine agent state.
// OpenClaw uses specific output patterns for state detection.
func (p *OpenClawProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])
		lineLower := strings.ToLower(line)

		// Working indicators - OpenClaw spinner and activity patterns
		if strings.HasPrefix(line, "⠋") ||
			strings.HasPrefix(line, "⠙") ||
			strings.HasPrefix(line, "⠹") ||
			strings.HasPrefix(line, "⠸") ||
			strings.HasPrefix(line, "⠼") ||
			strings.HasPrefix(line, "⠴") ||
			strings.HasPrefix(line, "🔍") ||
			strings.HasPrefix(line, "🔧") ||
			strings.Contains(lineLower, "analyzing") ||
			strings.Contains(lineLower, "thinking") ||
			strings.Contains(lineLower, "processing") ||
			strings.Contains(lineLower, "working") ||
			strings.Contains(lineLower, "searching") {
			return StateWorking
		}

		// Done indicators
		if strings.Contains(line, "✓") ||
			strings.Contains(line, "✔") ||
			strings.Contains(line, "🎉") ||
			strings.Contains(lineLower, "complete") ||
			strings.Contains(lineLower, "finished") ||
			strings.Contains(lineLower, "done") ||
			strings.Contains(lineLower, "success") {
			return StateDone
		}

		// Stuck indicators - check before error since some stuck messages contain "error"
		if strings.Contains(lineLower, "timeout") ||
			strings.Contains(lineLower, "rate limit") ||
			strings.Contains(lineLower, "quota") ||
			strings.Contains(lineLower, "connection refused") ||
			strings.Contains(lineLower, "network error") {
			return StateStuck
		}

		// Error indicators
		if strings.Contains(lineLower, "error") ||
			strings.Contains(lineLower, "failed") ||
			strings.Contains(lineLower, "exception") ||
			strings.Contains(line, "✗") ||
			strings.Contains(line, "✖") ||
			strings.Contains(line, "❌") {
			return StateError
		}

		// Idle/prompt indicators
		if strings.HasPrefix(line, ">") ||
			strings.HasPrefix(line, "$") ||
			strings.HasPrefix(line, "openclaw>") ||
			strings.HasPrefix(line, "claw>") ||
			strings.Contains(lineLower, "ready") ||
			strings.Contains(lineLower, "awaiting input") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure OpenClawProvider implements Provider interface.
var _ Provider = (*OpenClawProvider)(nil)
