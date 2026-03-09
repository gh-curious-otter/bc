package provider

import (
	"context"
	"strings"
)

// AiderProvider implements the Provider interface for Aider.
// Aider is a terminal-based AI pair programming tool.
//
// Issue #1477: Aider Provider Integration
type AiderProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewAiderProvider creates a new Aider provider.
func NewAiderProvider() *AiderProvider {
	return &AiderProvider{
		name:        "aider",
		description: "Aider AI Pair Programming",
		command:     "aider --yes",
		binary:      "aider",
	}
}

// Name returns the provider's unique identifier.
func (p *AiderProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *AiderProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *AiderProvider) Command() string {
	return p.command
}

// Binary returns the executable name for LookPath/version checks.
func (p *AiderProvider) Binary() string {
	return p.binary
}

// InstallHint returns a human-readable install instruction.
func (p *AiderProvider) InstallHint() string {
	return "pip install aider-chat"
}

// BuildCommand returns the full command for a given runtime context.
func (p *AiderProvider) BuildCommand(_ CommandOpts) string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *AiderProvider) IsInstalled(ctx context.Context) bool {
	return checkBinaryExists(ctx, p.binary)
}

// Version returns the installed version.
func (p *AiderProvider) Version(ctx context.Context) string {
	return getBinaryVersion(ctx, p.binary, "--version")
}

// DetectState analyzes output to determine agent state.
// Aider uses specific output patterns for state detection.
func (p *AiderProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])
		lineLower := strings.ToLower(line)

		// Working indicators - Aider shows activity patterns
		if strings.Contains(lineLower, "thinking") ||
			strings.Contains(lineLower, "sending") ||
			strings.Contains(lineLower, "editing") ||
			strings.Contains(lineLower, "running") ||
			strings.Contains(lineLower, "applying") ||
			strings.Contains(lineLower, "streaming") ||
			strings.Contains(lineLower, "analyzing") ||
			strings.HasPrefix(line, "⠋") ||
			strings.HasPrefix(line, "⠙") ||
			strings.HasPrefix(line, "⠹") ||
			strings.HasPrefix(line, "⠸") {
			return StateWorking
		}

		// Done indicators - Aider completion patterns
		if strings.Contains(lineLower, "applied edit") ||
			strings.Contains(lineLower, "wrote") ||
			strings.Contains(lineLower, "committed") ||
			strings.Contains(lineLower, "added to git") ||
			strings.Contains(line, "✓") ||
			strings.Contains(line, "✔") {
			return StateDone
		}

		// Stuck indicators - check before error
		if strings.Contains(lineLower, "rate limit") ||
			strings.Contains(lineLower, "quota") ||
			strings.Contains(lineLower, "timeout") ||
			strings.Contains(lineLower, "connection error") ||
			strings.Contains(lineLower, "api key") ||
			strings.Contains(lineLower, "authentication") {
			return StateStuck
		}

		// Error indicators
		if strings.Contains(lineLower, "error") ||
			strings.Contains(lineLower, "failed") ||
			strings.Contains(lineLower, "exception") ||
			strings.Contains(lineLower, "traceback") ||
			strings.Contains(line, "✗") ||
			strings.Contains(line, "❌") {
			return StateError
		}

		// Idle/prompt indicators - Aider prompt patterns
		if strings.HasPrefix(line, ">") ||
			strings.HasPrefix(line, "aider>") ||
			strings.Contains(lineLower, "enter to send") ||
			strings.Contains(lineLower, "waiting for input") ||
			strings.HasSuffix(lineLower, "? ") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure AiderProvider implements Provider interface.
var _ Provider = (*AiderProvider)(nil)
