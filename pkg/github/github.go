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

// ReviewEvent is the type of PR review: approve, request-changes, or comment.
type ReviewEvent string

const (
	ReviewApprove        ReviewEvent = "approve"
	ReviewRequestChanges ReviewEvent = "request-changes"
	ReviewComment        ReviewEvent = "comment"
)

// SubmitReview submits a PR review (approve, request-changes, or comment).
// Body is optional for approve; required or recommended for request-changes and comment.
// Returns ErrNoGitRemote if no remote is configured.
func SubmitReview(workspacePath string, prNumber int, event ReviewEvent, body string) error {
	if !HasGitRemote(workspacePath) {
		return ErrNoGitRemote
	}
	args := []string{"pr", "review", fmt.Sprintf("%d", prNumber)}
	switch event {
	case ReviewApprove:
		args = append(args, "--approve")
	case ReviewRequestChanges:
		args = append(args, "--request-changes")
	case ReviewComment:
		args = append(args, "--comment")
	default:
		return fmt.Errorf("invalid review event: %q (use approve, request-changes, or comment)", event)
	}
	if body != "" {
		args = append(args, "--body", body)
	}
	cmd := exec.CommandContext(context.Background(), "gh", args...) //nolint:gosec // gh command with trusted args
	cmd.Dir = workspacePath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh pr review: %w: %s", err, string(out))
	}
	return nil
}

// AddPRComment adds a single comment to a pull request.
// Returns ErrNoGitRemote if no remote is configured.
func AddPRComment(workspacePath string, prNumber int, body string) error {
	if !HasGitRemote(workspacePath) {
		return ErrNoGitRemote
	}
	if body == "" {
		return fmt.Errorf("comment body is required")
	}
	args := []string{"pr", "comment", fmt.Sprintf("%d", prNumber), "--body", body}
	cmd := exec.CommandContext(context.Background(), "gh", args...) //nolint:gosec // gh command with trusted args
	cmd.Dir = workspacePath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh pr comment: %w: %s", err, string(out))
	}
	return nil
}

// MergeMethod is how to merge: merge, squash, or rebase.
type MergeMethod string

const (
	MergeMerge  MergeMethod = "merge"
	MergeSquash MergeMethod = "squash"
	MergeRebase MergeMethod = "rebase"
)

// MergePR merges a pull request with the given method (merge, squash, or rebase).
// Returns ErrNoGitRemote if no remote is configured.
func MergePR(workspacePath string, prNumber int, method MergeMethod) error {
	if !HasGitRemote(workspacePath) {
		return ErrNoGitRemote
	}
	args := []string{"pr", "merge", fmt.Sprintf("%d", prNumber)}
	switch method {
	case MergeMerge:
		args = append(args, "--merge")
	case MergeSquash:
		args = append(args, "--squash")
	case MergeRebase:
		args = append(args, "--rebase")
	default:
		return fmt.Errorf("invalid merge method: %q (use merge, squash, or rebase)", method)
	}
	cmd := exec.CommandContext(context.Background(), "gh", args...) //nolint:gosec // gh command with trusted args
	cmd.Dir = workspacePath
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh pr merge: %w: %s", err, string(out))
	}
	return nil
}
