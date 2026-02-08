package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/github"
)

func TestGetItemType(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		issue    github.Issue
	}{
		{
			name: "task label",
			issue: github.Issue{
				Title:  "Fix bug",
				Labels: []string{"task"},
			},
			expected: "task",
		},
		{
			name: "epic label",
			issue: github.Issue{
				Title:  "New feature",
				Labels: []string{"epic"},
			},
			expected: "epic",
		},
		{
			name: "EPIC prefix in title",
			issue: github.Issue{
				Title:  "[EPIC] Big feature",
				Labels: []string{},
			},
			expected: "epic",
		},
		{
			name: "Epic prefix in title",
			issue: github.Issue{
				Title:  "[Epic] Another feature",
				Labels: []string{},
			},
			expected: "epic",
		},
		{
			name: "no task or epic label",
			issue: github.Issue{
				Title:  "Random issue",
				Labels: []string{"bug", "priority-high"},
			},
			expected: "",
		},
		{
			name: "task label with others",
			issue: github.Issue{
				Title:  "Do something",
				Labels: []string{"priority-high", "task", "v2"},
			},
			expected: "task",
		},
		{
			name: "epic takes precedence",
			issue: github.Issue{
				Title:  "Both labels",
				Labels: []string{"task", "epic"},
			},
			expected: "epic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getItemType(tt.issue)
			if result != tt.expected {
				t.Errorf("getItemType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestQueueCmdExists(t *testing.T) {
	if queueCmd == nil {
		t.Fatal("queueCmd should not be nil")
	}
	if queueCmd.Use != "queue" {
		t.Errorf("queueCmd.Use = %q, want %q", queueCmd.Use, "queue")
	}
}

func TestQueueListCmdExists(t *testing.T) {
	if queueListCmd == nil {
		t.Fatal("queueListCmd should not be nil")
	}
	if queueListCmd.Use != "list" {
		t.Errorf("queueListCmd.Use = %q, want %q", queueListCmd.Use, "list")
	}
}

func TestQueueAddCmdExists(t *testing.T) {
	if queueAddCmd == nil {
		t.Fatal("queueAddCmd should not be nil")
	}
	if queueAddCmd.Use != "add <title>" {
		t.Errorf("queueAddCmd.Use = %q, want %q", queueAddCmd.Use, "add <title>")
	}
}

func TestQueueAddCmdFlags(t *testing.T) {
	// Check description flag
	descFlag := queueAddCmd.Flags().Lookup("description")
	if descFlag == nil {
		t.Fatal("expected 'description' flag to exist")
	}
	if descFlag.Shorthand != "d" {
		t.Errorf("description shorthand = %q, want %q", descFlag.Shorthand, "d")
	}

	// Check epic flag
	epicFlag := queueAddCmd.Flags().Lookup("epic")
	if epicFlag == nil {
		t.Fatal("expected 'epic' flag to exist")
	}

	// Check label flag
	labelFlag := queueAddCmd.Flags().Lookup("label")
	if labelFlag == nil {
		t.Fatal("expected 'label' flag to exist")
	}
}

func TestQueueItemStruct(t *testing.T) {
	item := QueueItem{
		Number: 123,
		Title:  "Test item",
		Type:   "task",
		State:  "OPEN",
		Labels: []string{"task", "priority-high"},
		URL:    "https://github.com/owner/repo/issues/123",
	}

	if item.Number != 123 {
		t.Errorf("Number = %d, want 123", item.Number)
	}
	if item.Title != "Test item" {
		t.Errorf("Title = %q, want %q", item.Title, "Test item")
	}
	if item.Type != "task" {
		t.Errorf("Type = %q, want %q", item.Type, "task")
	}
	if item.State != "OPEN" {
		t.Errorf("State = %q, want %q", item.State, "OPEN")
	}
	if len(item.Labels) != 2 {
		t.Errorf("Labels length = %d, want 2", len(item.Labels))
	}
	if item.URL != "https://github.com/owner/repo/issues/123" {
		t.Errorf("URL = %q, want %q", item.URL, "https://github.com/owner/repo/issues/123")
	}
}

func TestQueueList_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err = executeCmd("queue")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestQueueListCmd_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err = executeCmd("queue", "list")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestQueueAdd_NoWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err = executeCmd("queue", "add", "Test task")
	if err == nil {
		t.Fatal("expected error when not in workspace")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestQueueAdd_RequiresTitle(t *testing.T) {
	setupTestWorkspace(t)

	_, err := executeCmd("queue", "add")
	if err == nil {
		t.Fatal("expected error when no title provided")
	}
	// Cobra should return an error about missing argument
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected argument error, got: %v", err)
	}
}

func TestGetItemType_EmptyLabels(t *testing.T) {
	issue := github.Issue{
		Title:  "Regular issue",
		Labels: nil,
	}
	result := getItemType(issue)
	if result != "" {
		t.Errorf("getItemType() with nil labels = %q, want empty", result)
	}
}

func TestGetItemType_CaseSensitive(t *testing.T) {
	// Labels should be case-sensitive (lowercase only matches)
	tests := []struct {
		name     string
		expected string
		labels   []string
	}{
		{"lowercase task", "task", []string{"task"}},
		{"lowercase epic", "epic", []string{"epic"}},
		{"uppercase TASK", "", []string{"TASK"}},
		{"uppercase EPIC", "", []string{"EPIC"}},
		{"mixed case Task", "", []string{"Task"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := github.Issue{
				Title:  "Test",
				Labels: tt.labels,
			}
			result := getItemType(issue)
			if result != tt.expected {
				t.Errorf("getItemType() = %q, want %q", result, tt.expected)
			}
		})
	}
}
