package cmd

import (
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/channel"
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

func TestFindTechLeads(t *testing.T) {
	tests := []struct {
		name    string
		want    []string
		members []string
	}{
		{
			name:    "no members",
			members: nil,
			want:    nil,
		},
		{
			name:    "no tech-leads",
			members: []string{"engineer-01", "qa-01"},
			want:    nil,
		},
		{
			name:    "single tech-lead",
			members: []string{"engineer-01", "tech-lead-01", "qa-01"},
			want:    []string{"tech-lead-01"},
		},
		{
			name:    "multiple tech-leads",
			members: []string{"tech-lead-01", "engineer-01", "tech-lead-02"},
			want:    []string{"tech-lead-01", "tech-lead-02"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store := channel.NewSQLiteStore(tmpDir)
			if err := store.Open(); err != nil {
				t.Fatalf("failed to open store: %v", err)
			}
			defer func() { _ = store.Close() }()

			// Check if engineering channel exists (may be created by default)
			ch, _ := store.GetChannel("engineering")
			if ch == nil {
				// Create engineering channel if it doesn't exist
				_, err := store.CreateChannel("engineering", channel.ChannelTypeGroup, "Engineering team")
				if err != nil {
					t.Fatalf("failed to create channel: %v", err)
				}
			}

			// Add members to channel
			for _, member := range tt.members {
				if addErr := store.AddMember("engineering", member); addErr != nil {
					t.Fatalf("failed to add member: %v", addErr)
				}
			}

			got := findTechLeads(store)

			if len(got) != len(tt.want) {
				t.Errorf("findTechLeads() = %v, want %v", got, tt.want)
				return
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("findTechLeads()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
