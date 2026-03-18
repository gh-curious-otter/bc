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

// Binary returns the executable name for LookPath/version checks.
func (p *ClaudeProvider) Binary() string {
	return p.binary
}

// InstallHint returns a human-readable install instruction.
func (p *ClaudeProvider) InstallHint() string {
	return "npx -y @anthropic-ai/claude-code"
}

// BuildCommand returns the full command for a given runtime context.
// Uses -w bc-<workspace>-<agent> for unique worktree names across workspaces
// to avoid branch collisions with other Claude Code sessions.
func (p *ClaudeProvider) BuildCommand(opts CommandOpts) string {
	cmd := p.command
	if opts.AgentName != "" {
		worktreeName := "bc-" + opts.AgentName
		if opts.WorkspaceName != "" {
			worktreeName = "bc-" + opts.WorkspaceName + "-" + opts.AgentName
		}
		cmd = "claude -w " + worktreeName + " " + strings.TrimPrefix(cmd, "claude")
	}
	if opts.Resume {
		// Try --continue to resume previous session; fall back to fresh start
		// if no prior session exists (avoids "No conversation found" crash)
		cmd += " --continue || " + cmd
	}
	return cmd
}

// AdjustSessionCommand injects --tmux for headless session execution (tmux or Docker).
func (p *ClaudeProvider) AdjustSessionCommand(command string) string {
	if !strings.Contains(command, "--tmux") {
		return strings.Replace(command, "claude", "claude --tmux", 1)
	}
	return command
}

// AdjustContainerCommand injects --tmux for Docker execution.
// Delegates to AdjustSessionCommand since the adjustment is the same.
func (p *ClaudeProvider) AdjustContainerCommand(command string) string {
	return p.AdjustSessionCommand(command)
}

// DockerImage returns empty to use default convention.
func (p *ClaudeProvider) DockerImage() string { return "" }

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

// Ensure ClaudeProvider implements Provider, ContainerCustomizer, and SessionCustomizer interfaces.
var _ Provider = (*ClaudeProvider)(nil)
var _ ContainerCustomizer = (*ClaudeProvider)(nil)
var _ SessionCustomizer = (*ClaudeProvider)(nil)
