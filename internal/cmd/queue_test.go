package cmd

import (
	"encoding/json"
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
		Title:  "Test JSON",
		Type:   "epic",
		State:  "CLOSED",
		Labels: []string{"epic", "v2"},
		URL:    "https://github.com/owner/repo/issues/42",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify key fields are present in JSON
	jsonStr := string(data)
	if !strings.Contains(jsonStr, `"number":42`) {
		t.Errorf("JSON should contain number field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"title":"Test JSON"`) {
		t.Errorf("JSON should contain title field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"type":"epic"`) {
		t.Errorf("JSON should contain type field: %s", jsonStr)
	}
	if !strings.Contains(jsonStr, `"state":"CLOSED"`) {
		t.Errorf("JSON should contain state field: %s", jsonStr)
	}
}

func TestQueueItemJSONUnmarshal(t *testing.T) {
	jsonData := `{"number":99,"title":"Unmarshaled","type":"task","state":"OPEN","labels":["task"]}`

	var item QueueItem
	err := json.Unmarshal([]byte(jsonData), &item)
	if err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if item.Number != 99 {
		t.Errorf("Number = %d, want 99", item.Number)
	}
	if item.Title != "Unmarshaled" {
		t.Errorf("Title = %q, want %q", item.Title, "Unmarshaled")
	}
	if item.Type != "task" {
		t.Errorf("Type = %q, want %q", item.Type, "task")
	}
	if item.State != "OPEN" {
		t.Errorf("State = %q, want %q", item.State, "OPEN")
	}
}

func TestQueueItemEmptyLabels(t *testing.T) {
	item := QueueItem{
		Number: 1,
		Title:  "No labels",
		Type:   "task",
		State:  "OPEN",
		Labels: nil,
	}

	if item.Labels != nil {
		t.Errorf("Labels should be nil, got %v", item.Labels)
	}

	// JSON should omit empty labels
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"labels":null`) {
		t.Errorf("JSON should omit nil labels: %s", string(data))
	}
}

func TestQueueItemEmptyURL(t *testing.T) {
	item := QueueItem{
		Number: 1,
		Title:  "No URL",
		Type:   "task",
		State:  "OPEN",
		URL:    "",
	}

	// JSON should omit empty URL due to omitempty tag
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	if strings.Contains(string(data), `"url":""`) {
		t.Errorf("JSON should omit empty URL: %s", string(data))
	}
}

func TestGetItemTypeEmptyLabels(t *testing.T) {
	issue := github.Issue{
		Title:  "No labels at all",
		Labels: nil,
	}
	result := getItemType(issue)
	if result != "" {
		t.Errorf("getItemType() = %q, want empty string", result)
	}
}

func TestGetItemTypeEpicPrefixCaseSensitivity(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"[EPIC] Uppercase prefix", "epic"},
		{"[Epic] Title case prefix", "epic"},
		{"[epic] Lowercase prefix", ""}, // Not matched - case sensitive
		{"EPIC without brackets", ""},   // Not matched - needs brackets
		{"[EPIC]NoSpace", "epic"},       // Matched even without space
		{" [EPIC] Leading space", ""},   // Not matched - must be at start
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			issue := github.Issue{
				Title:  tt.title,
				Labels: []string{},
			}
			result := getItemType(issue)
			if result != tt.expected {
				t.Errorf("getItemType(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestQueueCmdHasSubcommands(t *testing.T) {
	subcommands := queueCmd.Commands()
	if len(subcommands) < 2 {
		t.Errorf("queueCmd should have at least 2 subcommands, got %d", len(subcommands))
	}

	// Check for list and add subcommands
	hasAdd := false
	hasList := false
	for _, cmd := range subcommands {
		if cmd.Use == "list" {
			hasList = true
		}
		if cmd.Use == "add <title>" {
			hasAdd = true
		}
	}

	if !hasList {
		t.Error("queueCmd should have 'list' subcommand")
	}
	if !hasAdd {
		t.Error("queueCmd should have 'add' subcommand")
	}
}
