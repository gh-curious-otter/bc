package integrations

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/rpuneet/bc/pkg/workspace"
)

// --- Benchmark helpers ---

// fastMockCmd creates a mock command for benchmarking that returns quickly.
// Uses the test helper process pattern from github_test.go.
func fastMockCmd(stdout string, exitCode int) CommandFunc {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", stdout}
		cmd := exec.CommandContext(ctx, os.Args[0], cs...) //nolint:gosec // benchmark helper
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"MOCK_EXIT_CODE="+string(rune('0'+exitCode)),
			"MOCK_STDOUT="+stdout,
		)
		return cmd
	}
}

// newBenchWorkspace creates a workspace with GitHub tool enabled for benchmarking.
func newBenchWorkspace() *workspace.Workspace {
	return &workspace.Workspace{
		V2Config: &workspace.V2Config{
			Tools: workspace.ToolsConfig{
				GitHub: &workspace.ToolConfig{
					Command: "gh",
					Enabled: true,
				},
			},
		},
	}
}

// --- NewGitHubIntegration benchmarks ---

func BenchmarkNewGitHubIntegration(b *testing.B) {
	ws := newBenchWorkspace()

	b.ResetTimer()
	for range b.N {
		_, _ = NewGitHubIntegration(ws)
	}
}

func BenchmarkNewGitHubIntegration_Disabled(b *testing.B) {
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

	b.ResetTimer()
	for range b.N {
		_, _ = NewGitHubIntegration(ws)
	}
}

func BenchmarkNewGitHubIntegration_NoV2Config(b *testing.B) {
	ws := &workspace.Workspace{}

	b.ResetTimer()
	for range b.N {
		_, _ = NewGitHubIntegration(ws)
	}
}

// --- CheckAuth benchmarks ---

func BenchmarkCheckAuth_Success(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_ = gh.CheckAuth(ctx)
	}
}

func BenchmarkCheckAuth_Failure(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("not logged in", 1),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_ = gh.CheckAuth(ctx)
	}
}

// --- CreateIssue benchmarks ---

func BenchmarkCreateIssue_NoLabels(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("https://github.com/test/repo/issues/123", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.CreateIssue(ctx, "Test Issue", "Test body content", nil)
	}
}

func BenchmarkCreateIssue_WithLabels(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("https://github.com/test/repo/issues/123", 0),
	}
	ctx := context.Background()
	labels := []string{"bug", "priority:high", "team:backend"}

	b.ResetTimer()
	for range b.N {
		_, _ = gh.CreateIssue(ctx, "Test Issue", "Test body content", labels)
	}
}

func BenchmarkCreateIssue_ManyLabels(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("https://github.com/test/repo/issues/123", 0),
	}
	ctx := context.Background()
	labels := []string{
		"bug", "priority:high", "team:backend", "needs-review",
		"documentation", "security", "performance", "breaking-change",
	}

	b.ResetTimer()
	for range b.N {
		_, _ = gh.CreateIssue(ctx, "Test Issue", "Test body content", labels)
	}
}

func BenchmarkCreateIssue_Failure(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("error: not found", 1),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.CreateIssue(ctx, "Test Issue", "Test body", nil)
	}
}

// --- FindIssue benchmarks ---

func BenchmarkFindIssue_Found(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("42", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.FindIssue(ctx, "test query in:title")
	}
}

func BenchmarkFindIssue_NotFound(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("null", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.FindIssue(ctx, "nonexistent query")
	}
}

func BenchmarkFindIssue_Empty(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.FindIssue(ctx, "query")
	}
}

// --- IssueExists benchmarks ---

func BenchmarkIssueExists_True(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("42", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.IssueExists(ctx, "test query")
	}
}

func BenchmarkIssueExists_False(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("null", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	for range b.N {
		_, _ = gh.IssueExists(ctx, "nonexistent")
	}
}

// --- Parallel benchmarks ---

func BenchmarkCheckAuth_Parallel(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = gh.CheckAuth(ctx)
		}
	})
}

func BenchmarkFindIssue_Parallel(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("42", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = gh.FindIssue(ctx, "test query")
		}
	})
}

func BenchmarkIssueExists_Parallel(b *testing.B) {
	gh := &GitHubIntegration{
		command:     "gh",
		execCommand: fastMockCmd("42", 0),
	}
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = gh.IssueExists(ctx, "test query")
		}
	})
}
