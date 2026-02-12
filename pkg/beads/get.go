package beads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
)

// ErrIssueNotFound indicates the requested issue was not found.
var ErrIssueNotFound = errors.New("issue not found")

// GetIssue returns details for a single beads issue by ID.
// Returns ErrNoBeadsDir if the workspace has no .beads directory.
// Returns ErrIssueNotFound if the issue does not exist.
func GetIssue(workspacePath, issueID string) (*Issue, error) {
	if !HasBeads(workspacePath) {
		return nil, ErrNoBeadsDir
	}

	cmd := exec.CommandContext(context.Background(), "bd", "show", issueID, "--json")
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("bd show failed: %w", err)
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, fmt.Errorf("failed to parse issue: %w", err)
	}
	if len(issues) == 0 {
		return nil, ErrIssueNotFound
	}

	issues[0].Source = "beads"
	return &issues[0], nil
}
