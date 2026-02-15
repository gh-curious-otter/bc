package integrations

import (
	"context"
	"testing"

	"github.com/rpuneet/bc/pkg/workspace"
)

func TestNewGitHubIntegrationDisabled(t *testing.T) {
	// Test when GitHub tool is disabled
	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: false,
				},
			},
		},
	}

	_, err := NewGitHubIntegration(ws)
	if err == nil {
		t.Error("expected error when GitHub is disabled")
	}
}

func TestNewGitHubIntegrationMissing(t *testing.T) {
	// Test when GitHub tool is not configured
	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{},
		},
	}

	_, err := NewGitHubIntegration(ws)
	if err == nil {
		t.Error("expected error when GitHub is not configured")
	}
}

func TestNewGitHubIntegrationEnabled(t *testing.T) {
	// Test when GitHub tool is properly configured
	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: true,
				},
			},
		},
	}

	gh, err := NewGitHubIntegration(ws)
	if err != nil {
		t.Fatalf("NewGitHubIntegration failed: %v", err)
	}
	if gh.command != "gh" {
		t.Errorf("command: got %q, want %q", gh.command, "gh")
	}
}

func TestCheckAuth(t *testing.T) {
	// Note: This test requires 'gh' to be installed and authenticated
	// It will be skipped in CI if gh is not available
	t.Skip("Integration test - requires gh to be installed and authenticated")

	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: true,
				},
			},
		},
	}

	gh, err := NewGitHubIntegration(ws)
	if err != nil {
		t.Fatalf("NewGitHubIntegration failed: %v", err)
	}

	ctx := context.Background()
	// If gh is available and authenticated, this should work
	err = gh.CheckAuth(ctx)
	if err != nil {
		t.Logf("gh auth check failed (expected in some environments): %v", err)
	}
}

func TestCreateIssue(t *testing.T) {
	// This is an integration test that requires gh and GitHub access
	// Skipped by default as it modifies state
	t.Skip("Integration test - requires gh and GitHub access")

	ctx := context.Background()
	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: true,
				},
			},
		},
	}

	gh, err := NewGitHubIntegration(ws)
	if err != nil {
		t.Fatalf("NewGitHubIntegration failed: %v", err)
	}

	title := "[test] automated issue creation"
	body := "This is a test issue created by bc integration test"
	labels := []string{"test", "automated"}

	url, err := gh.CreateIssue(ctx, title, body, labels)
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}

	if url == "" {
		t.Error("expected non-empty issue URL")
	}

	t.Logf("Created issue: %s", url)
}

func TestFindIssue(t *testing.T) {
	// This is an integration test that requires gh and GitHub access
	t.Skip("Integration test - requires gh and GitHub access")

	ctx := context.Background()
	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: true,
				},
			},
		},
	}

	gh, err := NewGitHubIntegration(ws)
	if err != nil {
		t.Fatalf("NewGitHubIntegration failed: %v", err)
	}

	// Search for a test issue (won't find anything, but tests the query mechanism)
	issueNum, err := gh.FindIssue(ctx, "nonexistent-test-issue-12345")
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}

	// Should return empty string if not found
	if issueNum != "" {
		t.Logf("Found issue: %s", issueNum)
	}
}

func TestIssueExists(t *testing.T) {
	// This is an integration test that requires gh and GitHub access
	t.Skip("Integration test - requires gh and GitHub access")

	ctx := context.Background()
	ws := &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: true,
				},
			},
		},
	}

	gh, err := NewGitHubIntegration(ws)
	if err != nil {
		t.Fatalf("NewGitHubIntegration failed: %v", err)
	}

	exists, err := gh.IssueExists(ctx, "nonexistent-test-issue-12345")
	if err != nil {
		t.Fatalf("IssueExists failed: %v", err)
	}

	if exists {
		t.Error("expected nonexistent issue to return false")
	}
}
