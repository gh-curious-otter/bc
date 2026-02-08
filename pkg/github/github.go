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

// IssueDetail is the result of viewing a single issue.
type IssueDetail struct {
	Title   string `json:"title"`
	State   string `json:"state"`
	Body    string `json:"body"`
	URL     string `json:"url"`
	Author  string `json:"author,omitempty"`
	Created string `json:"createdAt,omitempty"`
	Number  int    `json:"number"`
}

type ghIssueView struct {
	Title     string `json:"title"`
	State     string `json:"state"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	CreatedAt string `json:"createdAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Number int `json:"number"`
}

// ViewIssue returns details for one issue. Uses gh auth from #288.
func ViewIssue(workspacePath string, number int) (*IssueDetail, error) {
	if !HasGitRemote(workspacePath) {
		return nil, ErrNoGitRemote
	}
	cmd := exec.CommandContext(context.Background(), "gh", "issue", "view", fmt.Sprintf("%d", number), //nolint:gosec // number is int, not user input
		"--json", "number,title,state,body,url,author,createdAt",
	)
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to view issue: %w", err)
	}
	var raw ghIssueView
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}
	return &IssueDetail{
		Number:  raw.Number,
		Title:   raw.Title,
		State:   raw.State,
		Body:    raw.Body,
		URL:     raw.URL,
		Author:  raw.Author.Login,
		Created: raw.CreatedAt,
	}, nil
}

// IssueComment adds a comment to an issue. Uses gh auth.
func IssueComment(workspacePath string, number int, body string) error {
	if !HasGitRemote(workspacePath) {
		return ErrNoGitRemote
	}
	cmd := exec.CommandContext(context.Background(), "gh", "issue", "comment", fmt.Sprintf("%d", number), "--body", body) //nolint:gosec
	cmd.Dir = workspacePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}
	return nil
}

// AddReaction adds a reaction to an issue (e.g. "+1", "heart", "rocket").
// Uses gh API; requires gh auth. See GitHub API reaction content values.
func AddReaction(workspacePath string, issueNumber int, content string) error {
	if !HasGitRemote(workspacePath) {
		return ErrNoGitRemote
	}
	// Get repo owner/name
	getRepo := exec.CommandContext(context.Background(), "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	getRepo.Dir = workspacePath
	repoOut, err := getRepo.Output()
	if err != nil {
		return fmt.Errorf("failed to get repo: %w", err)
	}
	repo := string(repoOut)
	if repo == "" {
		return fmt.Errorf("could not determine repo")
	}
	// Trim newline
	if len(repo) > 0 && repo[len(repo)-1] == '\n' {
		repo = repo[:len(repo)-1]
	}
	ep := fmt.Sprintf("repos/%s/issues/%d/reactions", repo, issueNumber)
	cmd := exec.CommandContext(context.Background(), "gh", "api", ep, "-f", "content="+content) //nolint:gosec
	cmd.Dir = workspacePath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	return nil
}
