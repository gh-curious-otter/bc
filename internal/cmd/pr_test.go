package cmd

import (
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/github"
)

func TestFormatReviewRequest(t *testing.T) {
	tests := []struct {
		wantParts []string
		techLeads []string
		name      string
		pr        github.PR
	}{
		{
			name: "basic PR without tech-leads",
			pr: github.PR{
				Number: 123,
				Title:  "Fix bug in auth",
			},
			techLeads: nil,
			wantParts: []string{"PR #123", "Fix bug in auth"},
		},
		{
			name: "PR with single tech-lead",
			pr: github.PR{
				Number: 456,
				Title:  "Add new feature",
			},
			techLeads: []string{"tech-lead-01"},
			wantParts: []string{"@tech-lead-01", "PR #456", "Add new feature"},
		},
		{
			name: "PR with multiple tech-leads",
			pr: github.PR{
				Number: 789,
				Title:  "Refactor module",
			},
			techLeads: []string{"tech-lead-01", "tech-lead-02"},
			wantParts: []string{"@tech-lead-01", "@tech-lead-02", "PR #789", "Refactor module"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatReviewRequest(tt.pr, tt.techLeads)
			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("formatReviewRequest() = %q, missing %q", got, part)
				}
			}
		})
	}
}
