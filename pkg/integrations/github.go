// Package integrations provides external service integrations.
package integrations

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rpuneet/bc/pkg/workspace"
)

// CommandFunc is the function signature for creating exec.Cmd.
// Used for testing to inject mock commands.
type CommandFunc func(ctx context.Context, name string, args ...string) *exec.Cmd

// GitHubIntegration wraps GitHub CLI (gh) operations.
type GitHubIntegration struct {
	execCommand CommandFunc
	command     string
}

// NewGitHubIntegration creates a new GitHub integration from workspace config.
func NewGitHubIntegration(ws *workspace.Workspace) (*GitHubIntegration, error) {
	if ws.V2Config == nil {
		return nil, fmt.Errorf("v2 configuration required for GitHub integration")
	}
	ghConfig := ws.V2Config.Tools.GitHub
	if ghConfig == nil || ghConfig.Command == "" {
		return nil, fmt.Errorf("github tool not configured in config.toml")
	}
	if !ghConfig.Enabled {
		return nil, fmt.Errorf("github tool is disabled in config.toml")
	}
	return &GitHubIntegration{
		command:     ghConfig.Command,
		execCommand: exec.CommandContext,
	}, nil
}

// CheckAuth verifies gh authentication status.
// Returns error if gh is not authenticated.
func (g *GitHubIntegration) CheckAuth(ctx context.Context) error {
	cmd := g.execCommand(ctx, g.command, "auth", "status")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh not authenticated - run 'gh auth login': %w", err)
	}
	return nil
}

// CreateIssue creates a new GitHub issue.
// Returns the issue URL on success.
func (g *GitHubIntegration) CreateIssue(ctx context.Context, title, body string, labels []string) (string, error) {
	args := make([]string, 0, 4+2*len(labels))
	args = append(args, "issue", "create", "-t", title, "-b", body)
	for _, label := range labels {
		args = append(args, "-l", label)
	}

	cmd := g.execCommand(ctx, g.command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create issue: %w - %s", err, strings.TrimSpace(string(output)))
	}

	return strings.TrimSpace(string(output)), nil
}

// FindIssue searches for an existing issue by query.
// Returns the issue number if found, empty string if not found.
func (g *GitHubIntegration) FindIssue(ctx context.Context, searchQuery string) (string, error) {
	cmd := g.execCommand(ctx, g.command, "issue", "list",
		"--search", searchQuery,
		"--json", "number",
		"--jq", ".[0].number")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Issue not found is not an error, just return empty
		return "", nil
	}

	result := strings.TrimSpace(string(output))
	if result == "null" || result == "" {
		return "", nil
	}

	return result, nil
}

// IssueExists checks if an issue with the given query exists.
func (g *GitHubIntegration) IssueExists(ctx context.Context, searchQuery string) (bool, error) {
	num, err := g.FindIssue(ctx, searchQuery)
	if err != nil {
		return false, err
	}
	return num != "", nil
}
