// Package github provides integration with GitHub via the gh CLI.
//
// This package wraps the gh CLI to query issues and pull requests
// for repositories associated with bc workspaces.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// Issue represents a GitHub issue.
type Issue struct {
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Source string   `json:"source"` // "github"
	Labels []string `json:"labels,omitempty"`
	Number int      `json:"number"`
}

// PR represents a GitHub pull request.
type PR struct {
	Title          string `json:"title"`
	State          string `json:"state"`
	ReviewDecision string `json:"reviewDecision,omitempty"`
	Number         int    `json:"number"`
	IsDraft        bool   `json:"isDraft,omitempty"`
}

// ghIssue is the raw JSON shape from gh issue list.
type ghIssue struct {
	Title  string `json:"title"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Number int `json:"number"`
}

// ghPR is the raw JSON shape from gh pr list.
type ghPR struct {
	Title          string `json:"title"`
	State          string `json:"state"`
	ReviewDecision string `json:"reviewDecision"`
	Number         int    `json:"number"`
	IsDraft        bool   `json:"isDraft"`
}

// HasGitRemote checks if the workspace has a git remote configured.
func HasGitRemote(workspacePath string) bool {
	cmd := exec.CommandContext(context.Background(), "git", "remote", "get-url", "origin")
	cmd.Dir = workspacePath
	return cmd.Run() == nil
}

// ErrNoGitRemote indicates no git remote is configured.
var ErrNoGitRemote = fmt.Errorf("no git remote configured")

// ListIssues returns GitHub issues for the workspace's repo.
// Returns ErrNoGitRemote if no remote is configured.
func ListIssues(workspacePath string) ([]Issue, error) {
	if !HasGitRemote(workspacePath) {
		return nil, ErrNoGitRemote
	}

	cmd := exec.CommandContext(context.Background(), "gh", "issue", "list",
		"--json", "number,title,state,labels",
		"--limit", "50",
	)
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub issues: %w", err)
	}

	var raw []ghIssue
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub issues: %w", err)
	}

	issues := make([]Issue, 0, len(raw))
	for _, r := range raw {
		var labels []string
		for _, l := range r.Labels {
			labels = append(labels, l.Name)
		}
		issues = append(issues, Issue{
			Number: r.Number,
			Title:  r.Title,
			State:  r.State,
			Labels: labels,
			Source: "github",
		})
	}

	return issues, nil
}

// ListPRs returns GitHub pull requests for the workspace's repo.
// Returns ErrNoGitRemote if no remote is configured.
func ListPRs(workspacePath string) ([]PR, error) {
	if !HasGitRemote(workspacePath) {
		return nil, ErrNoGitRemote
	}

	cmd := exec.CommandContext(context.Background(), "gh", "pr", "list",
		"--json", "number,title,state,reviewDecision,isDraft",
		"--limit", "50",
	)
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub PRs: %w", err)
	}

	var raw []ghPR
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub PRs: %w", err)
	}

	prs := make([]PR, 0, len(raw))
	for _, r := range raw {
		prs = append(prs, PR(r))
	}

	return prs, nil
}

// ListPRsInRepo returns GitHub pull requests for the given repo (owner/repo).
// Runs from current directory; uses gh pr list --repo.
func ListPRsInRepo(repo string) ([]PR, error) {
	cmd := exec.CommandContext(context.Background(), "gh", "pr", "list",
		"--repo", repo,
		"--json", "number,title,state,reviewDecision,isDraft",
		"--limit", "50",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub PRs: %w", err)
	}
	var raw []ghPR
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub PRs: %w", err)
	}
	prs := make([]PR, 0, len(raw))
	for _, r := range raw {
		prs = append(prs, PR(r))
	}
	return prs, nil
}

// ListIssuesInRepo returns GitHub issues for the given repo (owner/repo).
func ListIssuesInRepo(repo string) ([]Issue, error) {
	cmd := exec.CommandContext(context.Background(), "gh", "issue", "list",
		"--repo", repo,
		"--json", "number,title,state,labels",
		"--limit", "50",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub issues: %w", err)
	}
	var raw []ghIssue
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub issues: %w", err)
	}
	issues := make([]Issue, 0, len(raw))
	for _, r := range raw {
		var labels []string
		for _, l := range r.Labels {
			labels = append(labels, l.Name)
		}
		issues = append(issues, Issue{
			Number: r.Number,
			Title:  r.Title,
			State:  r.State,
			Labels: labels,
			Source: "github",
		})
	}
	return issues, nil
}

// CreateIssue creates a GitHub issue.
func CreateIssue(workspacePath, title, body string) error {
	args := []string{"issue", "create", "--title", title}
	if body != "" {
		args = append(args, "--body", body)
	}
	cmd := exec.CommandContext(context.Background(), "gh", args...) //nolint:gosec // gh command with trusted args
	cmd.Dir = workspacePath
	return cmd.Run()
}
