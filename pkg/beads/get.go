package beads

import (
	"encoding/json"
	"os/exec"
)

// GetIssue returns details for a single beads issue by ID.
// Returns nil if the issue is not found or bd is unavailable.
func GetIssue(workspacePath, issueID string) *Issue {
	if !HasBeads(workspacePath) {
		return nil
	}

	cmd := exec.Command("bd", "show", issueID, "--json")
	cmd.Dir = workspacePath
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil
	}
	if len(issues) == 0 {
		return nil
	}

	issues[0].Source = "beads"
	return &issues[0]
}
