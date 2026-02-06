// Package github provides integration with GitHub via the gh CLI.
//
// This package wraps the gh CLI to query issues and pull requests
// for repositories associated with bc workspaces.
package github

import (
	"encoding/json"
	"os/exec"
)

// Issue represents a GitHub issue.
type Issue struct {
	Number int      `json:"number"`
	Title  string   `json:"title"`
	State  string   `json:"state"`
	Labels []string `json:"labels,omitempty"`
	Source string   `json:"source"` // "github"
}

// PR represents a GitHub pull request.
type PR struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	State          string `json:"state"`
	ReviewDecision string `json:"reviewDecision,omitempty"`
	IsDraft        bool   `json:"isDraft,omitempty"`
}

// ghIssue is the raw JSON shape from gh issue list.
type ghIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

// ghPR is the raw JSON shape from gh pr list.
type ghPR struct {
	Number         int    `json:"number"`
	Title          string `json:"title"`
	State          string `json:"state"`
	ReviewDecision string `json:"reviewDecision"`
	IsDraft        bool   `json:"isDraft"`
}

// HasGitRemote checks if the workspace has a git remote configured.
func HasGitRemote(workspacePath string) bool {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = workspacePath
	return cmd.Run() == nil
}

// ListIssues returns GitHub issues for the workspace's repo.
// Falls back to empty list if gh is not available or no remote exists.
func ListIssues(workspacePath string) []Issue {
	if !HasGitRemote(workspacePath) {
		return nil
	}

	cmd := exec.Command("gh", "issue", "list",
		"--json", "number,title,state,labels",
		"--limit", "50",
	)
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var raw []ghIssue
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil
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

	return issues
}

// ListPRs returns GitHub pull requests for the workspace's repo.
func ListPRs(workspacePath string) []PR {
	if !HasGitRemote(workspacePath) {
		return nil
	}

	cmd := exec.Command("gh", "pr", "list",
		"--json", "number,title,state,reviewDecision,isDraft",
		"--limit", "50",
	)
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var raw []ghPR
	if err := json.Unmarshal(output, &raw); err != nil {
		return nil
	}

	prs := make([]PR, 0, len(raw))
	for _, r := range raw {
		prs = append(prs, PR{
			Number:         r.Number,
			Title:          r.Title,
			State:          r.State,
			ReviewDecision: r.ReviewDecision,
			IsDraft:        r.IsDraft,
		})
	}

	return prs
}

// CreateIssue creates a GitHub issue.
func CreateIssue(workspacePath, title, body string) error {
	args := []string{"issue", "create", "--title", title}
	if body != "" {
		args = append(args, "--body", body)
	}
	cmd := exec.Command("gh", args...) //nolint:gosec // gh command with trusted args
	cmd.Dir = workspacePath
	return cmd.Run()
}
