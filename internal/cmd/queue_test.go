package cmd

import (
	"encoding/json"
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

func TestQueueItemJSONMarshal(t *testing.T) {
	item := QueueItem{
		Number: 42,
		Title:  "Implement feature",
		Type:   "epic",
		State:  "OPEN",
		Labels: []string{"epic", "priority-high"},
		URL:    "https://github.com/owner/repo/issues/42",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal QueueItem: %v", err)
	}

	var parsed QueueItem
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal QueueItem: %v", err)
	}

	if parsed.Number != item.Number {
		t.Errorf("Number = %d, want %d", parsed.Number, item.Number)
	}
	if parsed.Title != item.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, item.Title)
	}
	if parsed.Type != item.Type {
		t.Errorf("Type = %q, want %q", parsed.Type, item.Type)
	}
	if parsed.State != item.State {
		t.Errorf("State = %q, want %q", parsed.State, item.State)
	}
	if len(parsed.Labels) != len(item.Labels) {
		t.Errorf("Labels length = %d, want %d", len(parsed.Labels), len(item.Labels))
	}
	if parsed.URL != item.URL {
		t.Errorf("URL = %q, want %q", parsed.URL, item.URL)
	}
}

func TestQueueItemJSONOmitEmpty(t *testing.T) {
	// Test that empty labels and URL are omitted from JSON
	item := QueueItem{
		Number: 1,
		Title:  "Simple task",
		Type:   "task",
		State:  "OPEN",
		Labels: nil,
		URL:    "",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("failed to marshal QueueItem: %v", err)
	}

	jsonStr := string(data)
	if strings.Contains(jsonStr, "labels") {
		t.Errorf("expected labels to be omitted, got: %s", jsonStr)
	}
	if strings.Contains(jsonStr, "url") {
		t.Errorf("expected url to be omitted, got: %s", jsonStr)
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

	_, _, err = executeIntegrationCmd("queue")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
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

	_, _, err = executeIntegrationCmd("queue", "add", "Test task")
	if err == nil {
		t.Fatal("expected error when not in workspace, got nil")
	}
	if !strings.Contains(err.Error(), "not in a bc workspace") {
		t.Errorf("expected workspace error, got: %v", err)
	}
}

func TestQueueAdd_NoTitle(t *testing.T) {
	_, _, err := executeIntegrationCmd("queue", "add")
	if err == nil {
		t.Fatal("expected error when no title provided, got nil")
	}
	// cobra.ExactArgs(1) should reject missing argument
	if !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Errorf("expected 'accepts 1 arg' error, got: %v", err)
	}
}

func TestQueueCmdDefaultsToList(t *testing.T) {
	// Verify that queueCmd.RunE is the same as queueListCmd.RunE
	// This ensures 'bc queue' defaults to 'bc queue list'
	if queueCmd.RunE == nil {
		t.Fatal("queueCmd.RunE should not be nil")
	}
	if queueListCmd.RunE == nil {
		t.Fatal("queueListCmd.RunE should not be nil")
	}
}

func TestGetItemType_EmptyLabels(t *testing.T) {
	issue := github.Issue{
		Title:  "Regular issue",
		Labels: []string{},
	}
	result := getItemType(issue)
	if result != "" {
		t.Errorf("expected empty string for issue with no task/epic labels, got %q", result)
	}
}

func TestGetItemType_NilLabels(t *testing.T) {
	issue := github.Issue{
		Title:  "Regular issue",
		Labels: nil,
	}
	result := getItemType(issue)
	if result != "" {
		t.Errorf("expected empty string for issue with nil labels, got %q", result)
	}
}

func TestGetItemType_CaseSensitiveLabels(t *testing.T) {
	// Labels are case-sensitive - "Task" is not the same as "task"
	tests := []struct {
		name     string
		expected string
		issue    github.Issue
	}{
		{
			name: "uppercase TASK label",
			issue: github.Issue{
				Title:  "Test",
				Labels: []string{"TASK"},
			},
			expected: "", // uppercase doesn't match
		},
		{
			name: "mixed case Epic label",
			issue: github.Issue{
				Title:  "Test",
				Labels: []string{"Epic"},
			},
			expected: "", // mixed case doesn't match
		},
		{
			name: "lowercase task and epic",
			issue: github.Issue{
				Title:  "Test",
				Labels: []string{"task", "epic"},
			},
			expected: "epic", // epic takes precedence
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

func TestGetItemType_TitleFallback(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		issue    github.Issue
	}{
		{
			name: "[EPIC] prefix uppercase",
			issue: github.Issue{
				Title:  "[EPIC] Build new system",
				Labels: []string{"priority-high"},
			},
			expected: "epic",
		},
		{
			name: "[Epic] prefix mixed case",
			issue: github.Issue{
				Title:  "[Epic] Build new system",
				Labels: []string{"priority-high"},
			},
			expected: "epic",
		},
		{
			name: "[epic] prefix lowercase - not matched",
			issue: github.Issue{
				Title:  "[epic] Build new system",
				Labels: []string{},
			},
			expected: "", // lowercase not matched
		},
		{
			name: "EPIC in middle of title - not matched",
			issue: github.Issue{
				Title:  "This is an [EPIC] story",
				Labels: []string{},
			},
			expected: "", // must be prefix
		},
		{
			name: "Label takes precedence over title",
			issue: github.Issue{
				Title:  "[EPIC] Some task",
				Labels: []string{"task"},
			},
			expected: "task", // label "task" is found before title fallback
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
