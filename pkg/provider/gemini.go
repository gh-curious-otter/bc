package provider

import (
	"context"
	"strings"
)

// GeminiProvider implements the Provider interface for Google Gemini CLI.
// Gemini CLI is Google's AI coding assistant for the terminal.
// See: https://github.com/google-gemini/gemini-cli
//
// Issue #1476: Gemini Provider Integration
type GeminiProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewGeminiProvider creates a new Gemini provider.
func NewGeminiProvider() *GeminiProvider {
	return &GeminiProvider{
		name:        "gemini",
		description: "Google Gemini CLI",
		command:     "gemini --yolo",
		binary:      "gemini",
	}
}

// Name returns the provider's unique identifier.
func (p *GeminiProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *GeminiProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *GeminiProvider) Command() string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *GeminiProvider) IsInstalled(ctx context.Context) bool {
	return checkBinaryExists(ctx, p.binary)
}

// Version returns the installed version.
func (p *GeminiProvider) Version(ctx context.Context) string {
	return getBinaryVersion(ctx, p.binary, "--version")
}

// DetectState analyzes output to determine agent state.
// Gemini CLI uses various indicators for different states.
func (p *GeminiProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])
		lineLower := strings.ToLower(line)

		// Working indicators - Gemini's thinking/processing patterns
		if strings.Contains(lineLower, "thinking") ||
			strings.Contains(lineLower, "processing") ||
			strings.Contains(lineLower, "generating") ||
			strings.Contains(lineLower, "analyzing") ||
			strings.Contains(lineLower, "reading") ||
			strings.Contains(lineLower, "writing") ||
			strings.Contains(lineLower, "searching") {
			return StateWorking
		}

		// Spinner indicators (common unicode spinners)
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
			strings.HasPrefix(line, "◐") ||
			strings.HasPrefix(line, "◓") ||
			strings.HasPrefix(line, "◑") ||
			strings.HasPrefix(line, "◒") {
			return StateWorking
		}

		// Done indicators
		if strings.Contains(line, "✓") ||
			strings.Contains(lineLower, "done") ||
			strings.Contains(lineLower, "complete") ||
			strings.Contains(lineLower, "finished") ||
			strings.Contains(lineLower, "success") {
			return StateDone
		}

		// Error indicators
		if strings.Contains(lineLower, "error") ||
			strings.Contains(lineLower, "failed") ||
			strings.Contains(lineLower, "exception") ||
			strings.Contains(line, "✗") ||
			strings.Contains(line, "✘") {
			return StateError
		}

		// Stuck indicators
		if strings.Contains(lineLower, "stuck") ||
			strings.Contains(lineLower, "timeout") ||
			strings.Contains(lineLower, "timed out") ||
			strings.Contains(lineLower, "rate limit") {
			return StateStuck
		}

		// Idle/prompt indicators - Gemini uses ">" for prompts
		if strings.HasPrefix(line, ">") ||
			strings.HasPrefix(line, "❯") ||
			strings.HasPrefix(line, "gemini>") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure GeminiProvider implements Provider interface.
var _ Provider = (*GeminiProvider)(nil)
