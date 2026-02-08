package cmd

import (
	"os"
	"strings"
	"testing"
)

func TestGithubPRListHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"github", "pr", "list", "--help"})
	var buf strings.Builder
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "pull requests") {
		t.Errorf("help should describe pull requests; got: %s", out)
	}
	if !strings.Contains(out, "state") {
		t.Errorf("help should mention state flag; got: %s", out)
	}
	if !strings.Contains(out, "repo") {
		t.Errorf("help should mention repo flag; got: %s", out)
	}
}

func TestGithubIssueListHelp(t *testing.T) {
	rootCmd.SetArgs([]string{"github", "issue", "list", "--help"})
	var buf strings.Builder
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "issues") {
		t.Errorf("help should describe issues; got: %s", out)
	}
	if !strings.Contains(out, "assignee") {
		t.Errorf("help should mention assignee flag; got: %s", out)
	}
}

func TestGithubPRListNoWorkspaceNoRepo(t *testing.T) {
	// Skip when already inside a bc workspace (e.g. running from worktree)
	if ws, _ := getWorkspace(); ws != nil {
		t.Skip("running inside a bc workspace; skip no-workspace test")
	}
	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prev) }()

	rootCmd.SetArgs([]string{"github", "pr", "list"})
	var outBuf, errBuf strings.Builder
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when not in workspace and --repo not set")
		return
	}
	msg := err.Error() + errBuf.String()
	if !strings.Contains(msg, "workspace") {
		t.Errorf("error should mention workspace; err=%v stderr=%s", err, errBuf.String())
	}
}

func TestGithubIssueListNoWorkspaceNoRepo(t *testing.T) {
	if ws, _ := getWorkspace(); ws != nil {
		t.Skip("running inside a bc workspace; skip no-workspace test")
	}
	dir := t.TempDir()
	prev, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(prev) }()

	rootCmd.SetArgs([]string{"github", "issue", "list"})
	var outBuf, errBuf strings.Builder
	rootCmd.SetOut(&outBuf)
	rootCmd.SetErr(&errBuf)
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error when not in workspace and --repo not set")
		return
	}
	msg := err.Error() + errBuf.String()
	if !strings.Contains(msg, "workspace") {
		t.Errorf("error should mention workspace; err=%v stderr=%s", err, errBuf.String())
	}
}

// List-with-mock tests live in pkg/github (ListPRsWithOpts, ListIssuesWithOpts with mock gh).
// Cmd tests cover help and no-workspace error only to avoid cobra/process state issues.
