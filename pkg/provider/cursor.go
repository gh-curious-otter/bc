package provider

import (
	"context"
	"strings"
)

// CursorProvider implements the Provider interface for Cursor Agent.
// Cursor is an AI-powered code editor that can run in terminal mode.
// https://cursor.sh
//
// Issue #1452: Cursor Agent Integration
type CursorProvider struct {
	name        string
	description string
	command     string
	binary      string
}

// NewCursorProvider creates a new Cursor provider.
func NewCursorProvider() *CursorProvider {
	return &CursorProvider{
		name:        "cursor",
		description: "Cursor AI-powered code editor",
		command:     "cursor-agent --force --print",
		binary:      "cursor-agent",
	}
}

// Name returns the provider's unique identifier.
func (p *CursorProvider) Name() string {
	return p.name
}

// Description returns a human-readable description.
func (p *CursorProvider) Description() string {
	return p.description
}

// Command returns the shell command to start this provider.
func (p *CursorProvider) Command() string {
	return p.command
}

// IsInstalled checks if the provider binary is available.
func (p *CursorProvider) IsInstalled(ctx context.Context) bool {
	// Check for cursor-agent (terminal mode) first
	if checkBinaryExists(ctx, "cursor-agent") {
		return true
	}
	// Fall back to cursor CLI
	return checkBinaryExists(ctx, "cursor")
}

// Version returns the installed version.
func (p *CursorProvider) Version(ctx context.Context) string {
	// Try cursor-agent first
	if checkBinaryExists(ctx, "cursor-agent") {
		return getBinaryVersion(ctx, "cursor-agent", "--version")
	}
	// Fall back to cursor
	if checkBinaryExists(ctx, "cursor") {
		return getBinaryVersion(ctx, "cursor", "--version")
	}
	return ""
}

// DetectState analyzes output to determine agent state.
// Cursor Agent uses terminal output patterns similar to other AI agents.
func (p *CursorProvider) DetectState(output string) State {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return StateUnknown
	}

	// Check last few lines for state indicators
	for i := len(lines) - 1; i >= 0 && i >= len(lines)-5; i-- {
		line := strings.TrimSpace(lines[i])
		lineLower := strings.ToLower(line)

		// Working indicators - Cursor's spinner and status patterns
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
			strings.HasPrefix(line, "◒") ||
			strings.Contains(lineLower, "thinking") ||
			strings.Contains(lineLower, "generating") ||
			strings.Contains(lineLower, "analyzing") ||
			strings.Contains(lineLower, "processing") ||
			strings.Contains(lineLower, "editing") ||
			strings.Contains(lineLower, "applying") {
			return StateWorking
		}

		// Done indicators
		if strings.Contains(line, "✓") ||
			strings.Contains(line, "✔") ||
			strings.Contains(lineLower, "done") ||
			strings.Contains(lineLower, "complete") ||
			strings.Contains(lineLower, "finished") ||
			strings.Contains(lineLower, "applied") {
			return StateDone
		}

		// Error indicators
		if strings.Contains(lineLower, "error") ||
			strings.Contains(lineLower, "failed") ||
			strings.Contains(line, "✗") ||
			strings.Contains(line, "✖") {
			return StateError
		}

		// Stuck indicators
		if strings.Contains(lineLower, "stuck") ||
			strings.Contains(lineLower, "timeout") ||
			strings.Contains(lineLower, "timed out") {
			return StateStuck
		}

		// Idle/prompt indicators
		if strings.HasPrefix(line, ">") ||
			strings.HasPrefix(line, "❯") ||
			strings.HasPrefix(line, "$") ||
			strings.HasPrefix(line, "cursor>") ||
			strings.Contains(lineLower, "ready") ||
			strings.Contains(lineLower, "waiting for input") {
			return StateIdle
		}
	}

	return StateUnknown
}

// Ensure CursorProvider implements Provider interface.
var _ Provider = (*CursorProvider)(nil)
