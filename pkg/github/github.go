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

// ListIssuesOpts holds filter options for listing issues.
//
//nolint:govet // fieldalignment: opts struct, readability over 8-byte packing
type ListIssuesOpts struct {
	State     string   // open, closed, all
	Repo      string   // owner/repo (optional; uses workspace repo if empty)
	Author    string   // filter by author
	Assignee  string   // filter by assignee
	Labels    []string // filter by labels
	Limit     int      // max items (0 = default 50)
	Workspace string   // directory for repo context when Repo is empty
}

// ListPROpts holds filter options for listing pull requests.
//
//nolint:govet // fieldalignment: opts struct, readability over 8-byte packing
type ListPROpts struct {
	State     string   // open, closed, merged, all
	Repo      string   // owner/repo (optional)
	Author    string   // filter by author
	Assignee  string   // filter by assignee
	Labels    []string // filter by labels
	Limit     int      // max items (0 = default 50)
	Workspace string   // directory for repo context when Repo is empty
}

func defaultLimit(n int) int {
	if n <= 0 {
		return 50
	}
	return n
}

// ListIssues returns GitHub issues for the workspace's repo.
// Returns ErrNoGitRemote if no remote is configured.
func ListIssues(workspacePath string) ([]Issue, error) {
	return ListIssuesWithOpts(context.Background(), ListIssuesOpts{Workspace: workspacePath})
}

// ListIssuesWithOpts returns GitHub issues with the given filters.
// When Repo is empty, Workspace must be set and must have a git remote; otherwise Repo is used with gh -R.
func ListIssuesWithOpts(ctx context.Context, opts ListIssuesOpts) ([]Issue, error) {
	if opts.Repo == "" {
		if opts.Workspace == "" || !HasGitRemote(opts.Workspace) {
			return nil, ErrNoGitRemote
		}
	}

	args := []string{"issue", "list",
		"--json", "number,title,state,labels",
		"--limit", fmt.Sprintf("%d", defaultLimit(opts.Limit)),
	}
	if opts.State != "" {
		args = append(args, "--state", opts.State)
	}
	if opts.Repo != "" {
		args = append(args, "--repo", opts.Repo)
	}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	if opts.Assignee != "" {
		args = append(args, "--assignee", opts.Assignee)
	}
	for _, l := range opts.Labels {
		args = append(args, "--label", l)
	}

	cmd := exec.CommandContext(ctx, "gh", args...) //nolint:gosec // gh with trusted args
	if opts.Repo == "" {
		cmd.Dir = opts.Workspace
	}
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
	return ListPRsWithOpts(context.Background(), ListPROpts{Workspace: workspacePath})
}

// ListPRsWithOpts returns GitHub pull requests with the given filters.
// When Repo is empty, Workspace must be set and must have a git remote; otherwise Repo is used with gh -R.
func ListPRsWithOpts(ctx context.Context, opts ListPROpts) ([]PR, error) {
	if opts.Repo == "" {
		if opts.Workspace == "" || !HasGitRemote(opts.Workspace) {
			return nil, ErrNoGitRemote
		}
	}

	args := []string{"pr", "list",
		"--json", "number,title,state,reviewDecision,isDraft",
		"--limit", fmt.Sprintf("%d", defaultLimit(opts.Limit)),
	}
	if opts.State != "" {
		args = append(args, "--state", opts.State)
	}
	if opts.Repo != "" {
		args = append(args, "--repo", opts.Repo)
	}
	if opts.Author != "" {
		args = append(args, "--author", opts.Author)
	}
	if opts.Assignee != "" {
		args = append(args, "--assignee", opts.Assignee)
	}
	for _, l := range opts.Labels {
		args = append(args, "--label", l)
	}

	cmd := exec.CommandContext(ctx, "gh", args...) //nolint:gosec // gh with trusted args
	if opts.Repo == "" {
		cmd.Dir = opts.Workspace
	}
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
