// Package beads provides integration with the beads (bd) issue tracker.
//
// Beads is a distributed, git-backed graph issue tracker for AI agents.
// Issues are stored as JSONL in .beads/ directories. This package wraps
// the bd CLI to query and manage issues.
package beads

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNoBeadsDir indicates the workspace has no .beads directory configured.
var ErrNoBeadsDir = errors.New("no .beads directory found")

// Issue represents a beads issue.
type Issue struct {
	Dependencies []string `json:"dependencies,omitempty"`
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Status       string   `json:"status"`
	Priority     any      `json:"priority,omitempty"`
	Assignee     string   `json:"assignee,omitempty"`
	Type         string   `json:"issue_type,omitempty"`
	Source       string   `json:"source"` // "beads"
}

// HasBeads checks if the workspace has a .beads directory.
func HasBeads(workspacePath string) bool {
	_, err := os.Stat(filepath.Join(workspacePath, ".beads"))
	return err == nil
}

// ListIssues returns beads issues for a workspace (excluding epics).
// Returns ErrNoBeadsDir if the workspace has no .beads directory.
func ListIssues(workspacePath string) ([]Issue, error) {
	all, err := ListAllIssues(workspacePath)
	if err != nil {
		return nil, err
	}
	var filtered []Issue
	for _, issue := range all {
		if issue.Type != "epic" {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

// ListAllIssues returns all beads issues for a workspace (including epics).
// Returns ErrNoBeadsDir if the workspace has no .beads directory.
func ListAllIssues(workspacePath string) ([]Issue, error) {
	if !HasBeads(workspacePath) {
		return nil, ErrNoBeadsDir
	}

	// Try running bd list --all --json
	cmd := exec.Command("bd", "list", "--all", "--json") //nolint:noctx // no caller context available yet
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd list failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		// Try line-by-line JSONL parsing
		return parseJSONL(output), nil
	}

	// Tag source
	for i := range issues {
		issues[i].Source = "beads"
	}

	return issues, nil
}

// parseJSONL parses newline-delimited JSON.
func parseJSONL(data []byte) []Issue {
	var issues []Issue
	dec := json.NewDecoder(strings.NewReader(string(data)))
	for dec.More() {
		var issue Issue
		if err := dec.Decode(&issue); err != nil {
			break
		}
		issue.Source = "beads"
		issues = append(issues, issue)
	}
	return issues
}

// AddIssue creates a new beads issue.
func AddIssue(workspacePath, title, description string) error {
	args := []string{"add", title}
	if description != "" {
		args = append(args, "-d", description)
	}
	cmd := exec.Command("bd", args...) //nolint:gosec // bd command with trusted args
	cmd.Dir = workspacePath
	return cmd.Run()
}

// ReadyIssues returns issues that are unblocked and ready for work.
func ReadyIssues(workspacePath string) []Issue {
	if !HasBeads(workspacePath) {
		return nil
	}

	cmd := exec.Command("bd", "ready", "--json")
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil
	}

	// Tag source and filter out epics
	var filtered []Issue
	for i := range issues {
		issues[i].Source = "beads"
		if issues[i].Type != "epic" {
			filtered = append(filtered, issues[i])
		}
	}

	return filtered
}

// AssignIssue assigns an issue to an agent.
func AssignIssue(workspacePath, issueID, agentName string) error {
	cmd := exec.Command("bd", "update", issueID, "--assign", agentName)
	cmd.Dir = workspacePath
	return cmd.Run()
}

// CloseIssue closes an issue.
func CloseIssue(workspacePath, issueID string) error {
	cmd := exec.Command("bd", "close", issueID)
	cmd.Dir = workspacePath
	return cmd.Run()
}
