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

// ErrNotAuthenticated indicates the user is not logged in to GitHub (gh).
// Callers should prompt to run bc github auth login.
var ErrNotAuthenticated = fmt.Errorf("not logged in to GitHub: run 'bc github auth login'")

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
		if isNotAuthenticated() {
			return nil, ErrNotAuthenticated
		}
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
		if isNotAuthenticated() {
			return nil, ErrNotAuthenticated
		}
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

// --- GitHub auth (login, status, token storage) ---

// AuthAccount represents one logged-in GitHub account for a host.
type AuthAccount struct {
	Host        string `json:"host"`
	Login       string `json:"login"`
	State       string `json:"state"`       // "success" when logged in
	TokenSource string `json:"tokenSource"` // e.g. "keyring", "oauth_token"
	Scopes      string `json:"scopes"`
	GitProtocol string `json:"gitProtocol"`
	Active      bool   `json:"active"`
}

// AuthStatusResult is the result of AuthStatus.
//
//nolint:govet // fieldalignment: struct layout kept for JSON field order
type AuthStatusResult struct {
	Accounts []AuthAccount `json:"accounts,omitempty"`
	LoggedIn bool          `json:"loggedIn"`
	RawOut   string        `json:"-"` // raw stdout from gh auth status
}

// authStatusJSON is the shape of `gh auth status --json hosts`.
type authStatusJSON struct {
	Hosts map[string][]AuthAccount `json:"hosts"`
}

// isNotAuthenticated returns true if gh reports no logged-in user.
// Used to convert gh command failures into ErrNotAuthenticated.
func isNotAuthenticated() bool {
	result, err := AuthStatus()
	return err == nil && !result.LoggedIn
}

// AuthStatus runs `gh auth status` and returns whether the user is logged in
// and account details. It does not require a workspace (gh auth is global).
func AuthStatus() (AuthStatusResult, error) {
	cmd := exec.CommandContext(context.Background(), "gh", "auth", "status", "--json", "hosts")
	output, err := cmd.Output()
	rawOut := string(output)
	if err != nil {
		return AuthStatusResult{LoggedIn: false, RawOut: rawOut}, fmt.Errorf("gh auth status: %w", err)
	}

	var parsed authStatusJSON
	if err := json.Unmarshal(output, &parsed); err != nil {
		return AuthStatusResult{RawOut: rawOut}, fmt.Errorf("parse gh auth status: %w", err)
	}

	var accounts []AuthAccount
	for _, accts := range parsed.Hosts {
		accounts = append(accounts, accts...)
	}
	loggedIn := len(accounts) > 0
	for _, a := range accounts {
		if a.State != "success" {
			loggedIn = false
			break
		}
	}

	return AuthStatusResult{
		LoggedIn: loggedIn,
		Accounts: accounts,
		RawOut:   rawOut,
	}, nil
}

// AuthLogin runs `gh auth login` interactively. The user must complete
// login in the terminal (browser or token). For scripted token use,
// run `gh auth login --with-token` with token on stdin.
func AuthLogin() error {
	cmd := exec.CommandContext(context.Background(), "gh", "auth", "login")
	cmd.Stdin = nil  // use terminal stdin
	cmd.Stdout = nil // use terminal stdout
	cmd.Stderr = nil // use terminal stderr
	return cmd.Run()
}

// TokenStorageInfo returns a short description of where gh stores the token.
// gh stores tokens in the system keyring (e.g. macOS Keychain) or in
// ~/.config/gh/hosts.yml when not using keyring. This is informational only.
func TokenStorageInfo() string {
	return "gh stores tokens in the system credential helper (e.g. keyring) or in ~/.config/gh/hosts.yml. Use 'gh auth status' to see token source."
}
