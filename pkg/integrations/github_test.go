package integrations

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/rpuneet/bc/pkg/workspace"
)

// mockCmd creates a mock command that returns the given output and exit code.
func mockCmd(stdout string, exitCode int) CommandFunc {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", stdout}
		cmd := exec.CommandContext(ctx, os.Args[0], cs...) //nolint:gosec // test helper
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"MOCK_EXIT_CODE="+string(rune('0'+exitCode)),
			"MOCK_STDOUT="+stdout,
		)
		return cmd
	}
}

// TestHelperProcess is used by mockCmd to simulate command execution.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	stdout := os.Getenv("MOCK_STDOUT")
	exitCode := int(os.Getenv("MOCK_EXIT_CODE")[0] - '0')
	os.Stdout.WriteString(stdout) //nolint:errcheck
	os.Exit(exitCode)
}

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

func TestCheckAuthSuccess(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("", 0),
	}

	ctx := context.Background()
	err := gh.CheckAuth(ctx)
	if err != nil {
		t.Errorf("CheckAuth should succeed: %v", err)
	}
}

func TestCheckAuthFailure(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("not logged in", 1),
	}

	ctx := context.Background()
	err := gh.CheckAuth(ctx)
	if err == nil {
		t.Error("CheckAuth should fail when not authenticated")
	}
}

func TestCreateIssueSuccess(t *testing.T) {
	expectedURL := "https://github.com/test/repo/issues/123"
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd(expectedURL, 0),
	}

	ctx := context.Background()
	url, err := gh.CreateIssue(ctx, "Test Issue", "Test body", []string{"bug"})
	if err != nil {
		t.Fatalf("CreateIssue failed: %v", err)
	}
	if url != expectedURL {
		t.Errorf("url: got %q, want %q", url, expectedURL)
	}
}

func TestCreateIssueFailure(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("error: not found", 1),
	}

	ctx := context.Background()
	_, err := gh.CreateIssue(ctx, "Test Issue", "Test body", nil)
	if err == nil {
		t.Error("CreateIssue should fail")
	}
}

func TestFindIssueFound(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("42", 0),
	}

	ctx := context.Background()
	issueNum, err := gh.FindIssue(ctx, "test query")
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}
	if issueNum != "42" {
		t.Errorf("issueNum: got %q, want %q", issueNum, "42")
	}
}

func TestFindIssueNotFound(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("null", 0),
	}

	ctx := context.Background()
	issueNum, err := gh.FindIssue(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}
	if issueNum != "" {
		t.Errorf("issueNum should be empty for not found, got %q", issueNum)
	}
}

func TestFindIssueEmpty(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("", 0),
	}

	ctx := context.Background()
	issueNum, err := gh.FindIssue(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("FindIssue failed: %v", err)
	}
	if issueNum != "" {
		t.Errorf("issueNum should be empty, got %q", issueNum)
	}
}

func TestIssueExistsTrue(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("42", 0),
	}

	ctx := context.Background()
	exists, err := gh.IssueExists(ctx, "test query")
	if err != nil {
		t.Fatalf("IssueExists failed: %v", err)
	}
	if !exists {
		t.Error("IssueExists should return true when issue is found")
	}
}

func TestIssueExistsFalse(t *testing.T) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: mockCmd("null", 0),
	}

	ctx := context.Background()
	exists, err := gh.IssueExists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("IssueExists failed: %v", err)
	}
	if exists {
		t.Error("IssueExists should return false when issue is not found")
	}
}
