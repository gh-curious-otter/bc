package provider

import (
	"context"
	"strings"
)

// ClaudeProvider implements the Provider interface for Claude Code.
// Claude Code is the Anthropic CLI for Claude.
type ClaudeProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewClaudeProvider creates a new Claude provider.
func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{
		name:        "claude",
		description: "Anthropic Claude Code CLI",
		command:     "claude --dangerously-skip-permissions",
		binary:      "claude",
	}
}

// Name returns the provider's unique identifier.
func (p *ClaudeProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *ClaudeProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *ClaudeProvider) Command() string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *ClaudeProvider) IsInstalled(ctx context.Context) bool {
	return checkBinaryExists(ctx, p.binary)
}

// Version returns the installed version.
func (p *ClaudeProvider) Version(ctx context.Context) string {
	return getBinaryVersion(ctx, p.binary, "--version")
}

// DetectState analyzes output to determine agent state.
// Claude uses specific spinner and prompt symbols.
func (p *ClaudeProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])

		// Working indicators - Claude's spinner symbols
		if strings.HasPrefix(line, "✻") ||
			strings.HasPrefix(line, "✳") ||
			strings.HasPrefix(line, "✽") ||
			strings.HasPrefix(line, "·") {
			return StateWorking
		}

		// Tool call indicator
		if strings.HasPrefix(line, "⏺") {
			return StateWorking
		}

		// Idle/prompt indicator
		if strings.HasPrefix(line, "❯") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure ClaudeProvider implements Provider interface.
var _ Provider = (*ClaudeProvider)(nil)
