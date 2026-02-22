package provider

import (
	"context"
	"strings"
)

// CodexProvider implements the Provider interface for OpenAI Codex CLI.
// Codex is OpenAI's code generation model.
//
// Issue #1479: Codex CLI Provider Integration
type CodexProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewCodexProvider creates a new Codex provider.
func NewCodexProvider() *CodexProvider {
	return &CodexProvider{
		name:        "codex",
		description: "OpenAI Codex CLI",
		command:     "codex --full-auto",
		binary:      "codex",
	}
}

// Name returns the provider's unique identifier.
func (p *CodexProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *CodexProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *CodexProvider) Command() string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *CodexProvider) IsInstalled(ctx context.Context) bool {
	return checkBinaryExists(ctx, p.binary)
}

// Version returns the installed version.
func (p *CodexProvider) Version(ctx context.Context) string {
	return getBinaryVersion(ctx, p.binary, "--version")
}

// DetectState analyzes output to determine agent state.
// Codex uses specific output patterns for state detection.
func (p *CodexProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])
		lineLower := strings.ToLower(line)

		// Working indicators - Codex spinner patterns
		if strings.HasPrefix(line, "⠋") ||
			strings.HasPrefix(line, "⠙") ||
			strings.HasPrefix(line, "⠹") ||
			strings.HasPrefix(line, "⠸") ||
			strings.HasPrefix(line, "⠼") ||
			strings.HasPrefix(line, "⠴") ||
			strings.Contains(lineLower, "generating") ||
			strings.Contains(lineLower, "thinking") ||
			strings.Contains(lineLower, "processing") ||
			strings.Contains(lineLower, "executing") {
			return StateWorking
		}

		// Done indicators
		if strings.Contains(line, "✓") ||
			strings.Contains(line, "✔") ||
			strings.Contains(lineLower, "complete") ||
			strings.Contains(lineLower, "finished") ||
			strings.Contains(lineLower, "done") ||
			strings.Contains(lineLower, "success") {
			return StateDone
		}

		// Error indicators
		if strings.Contains(lineLower, "error") ||
			strings.Contains(lineLower, "failed") ||
			strings.Contains(lineLower, "exception") ||
			strings.Contains(line, "✗") ||
			strings.Contains(line, "✖") {
			return StateError
		}

		// Stuck indicators
		if strings.Contains(lineLower, "timeout") ||
			strings.Contains(lineLower, "rate limit") ||
			strings.Contains(lineLower, "quota exceeded") {
			return StateStuck
		}

		// Idle/prompt indicators
		if strings.HasPrefix(line, ">") ||
			strings.HasPrefix(line, "$") ||
			strings.HasPrefix(line, "codex>") ||
			strings.Contains(lineLower, "ready") ||
			strings.Contains(lineLower, "awaiting") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure CodexProvider implements Provider interface.
var _ Provider = (*CodexProvider)(nil)
