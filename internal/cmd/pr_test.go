package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/github"
	"github.com/rpuneet/bc/pkg/workspace"
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

// --- prNumberFromArgs ---

func TestPrNumberFromArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		prFlag  int
		wantNum int
		wantErr bool
	}{
		{"from args", []string{"5"}, 0, 5, false},
		{"from flag", nil, 7, 7, false},
		{"args override flag", []string{"12"}, 3, 12, false},
		{"missing", nil, 0, 0, true},
		{"invalid", []string{"x"}, 0, 0, true},
		{"zero", []string{"0"}, 0, 0, true},
		{"negative", []string{"-1"}, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prNumber = tt.prFlag
			defer func() { prNumber = 0 }()
			got, err := prNumberFromArgs(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("prNumberFromArgs() err = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.wantNum {
				t.Errorf("prNumberFromArgs() = %d, want %d", got, tt.wantNum)
			}
		})
	}
}

// --- pr review / comment / merge with workspace and mock gh ---

func TestPRReviewCommentMergeInWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	ws, err := workspace.Init(tmpDir)
	if err != nil {
		t.Fatalf("workspace init: %v", err)
	}
	gitDir := ws.RootDir
	setupGitRepoWithRemote(t, gitDir)

	mockGh := createMockGhForCmd(t)
	pathEnv := filepath.Dir(mockGh) + string(filepath.ListSeparator) + os.Getenv("PATH")
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()
	if err := os.Chdir(gitDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldDir) }()

	t.Run("review_approve", func(t *testing.T) {
		prReviewApprove = true
		prReviewRequestChanges = false
		prReviewComment = false
		prReviewBody = ""
		defer func() {
			prReviewApprove = false
		}()
		_ = os.Setenv("PATH", pathEnv)
		defer func() { _ = os.Unsetenv("PATH") }()
		err := runPRReview(nil, []string{"1"})
		if err != nil {
			t.Errorf("runPRReview: %v", err)
		}
	})

	t.Run("comment", func(t *testing.T) {
		prCommentBody = "test comment"
		defer func() { prCommentBody = "" }()
		_ = os.Setenv("PATH", pathEnv)
		defer func() { _ = os.Unsetenv("PATH") }()
		err := runPRComment(nil, []string{"2"})
		if err != nil {
			t.Errorf("runPRComment: %v", err)
		}
	})

	t.Run("merge", func(t *testing.T) {
		prMergeMethod = "squash"
		defer func() { prMergeMethod = "merge" }()
		_ = os.Setenv("PATH", pathEnv)
		defer func() { _ = os.Unsetenv("PATH") }()
		err := runPRMerge(nil, []string{"3"})
		if err != nil {
			t.Errorf("runPRMerge: %v", err)
		}
	})
}

// setupGitRepoWithRemote runs git init and git remote add origin in dir (so HasGitRemote returns true).
func setupGitRepoWithRemote(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	initCmd := exec.CommandContext(ctx, "git", "init")
	initCmd.Dir = dir
	if err := initCmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}
	remoteCmd := exec.CommandContext(ctx, "git", "remote", "add", "origin", "https://example.com/repo.git")
	remoteCmd.Dir = dir
	if err := remoteCmd.Run(); err != nil {
		t.Fatalf("git remote add: %v", err)
	}
}

// createMockGhForCmd creates a mock gh script that exits 0 (for use in cmd tests).
func createMockGhForCmd(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mockPath := filepath.Join(dir, "gh")
	script := "#!/bin/sh\nexit 0\n"
	if err := os.WriteFile(mockPath, []byte(script), 0700); err != nil { //nolint:gosec
		t.Fatalf("create mock gh: %v", err)
	}
	return mockPath
}
