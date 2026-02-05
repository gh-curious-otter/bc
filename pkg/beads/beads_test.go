package beads

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// --- HasBeads ---

func TestHasBeads(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(dir string)
		wantBeads bool
	}{
		{
			name:      "no .beads directory",
			setup:     func(dir string) {},
			wantBeads: false,
		},
		{
			name: "with .beads directory",
			setup: func(dir string) {
				os.MkdirAll(filepath.Join(dir, ".beads"), 0755)
			},
			wantBeads: true,
		},
		{
			name: ".beads is a file not directory",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, ".beads"), []byte("not a dir"), 0644)
			},
			wantBeads: true, // os.Stat succeeds for files too
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)

			got := HasBeads(dir)
			if got != tt.wantBeads {
				t.Errorf("HasBeads() = %v, want %v", got, tt.wantBeads)
			}
		})
	}
}

func TestHasBeadsNonexistentPath(t *testing.T) {
	got := HasBeads("/nonexistent/path/that/does/not/exist")
	if got {
		t.Error("HasBeads on nonexistent path should return false")
	}
}

// --- parseJSONL ---

func TestParseJSONL(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantFirst string // title of first issue
	}{
		{
			name:      "single issue",
			input:     `{"id":"abc","title":"Fix bug","status":"open"}`,
			wantCount: 1,
			wantFirst: "Fix bug",
		},
		{
			name:      "multiple issues",
			input:     "{\"id\":\"a\",\"title\":\"First\",\"status\":\"open\"}\n{\"id\":\"b\",\"title\":\"Second\",\"status\":\"closed\"}\n",
			wantCount: 2,
			wantFirst: "First",
		},
		{
			name:      "empty input",
			input:     "",
			wantCount: 0,
		},
		{
			name:      "whitespace only",
			input:     "   \n  \n  ",
			wantCount: 0,
		},
		{
			name:      "malformed first line stops parsing",
			input:     "not json\n{\"id\":\"a\",\"title\":\"Good\",\"status\":\"open\"}\n",
			wantCount: 0,
		},
		{
			name:      "valid then malformed",
			input:     "{\"id\":\"a\",\"title\":\"Good\",\"status\":\"open\"}\nnot json\n",
			wantCount: 1,
			wantFirst: "Good",
		},
		{
			name:      "issue with all fields",
			input:     `{"id":"x","title":"Full","description":"desc","status":"open","priority":1,"assignee":"alice","issue_type":"bug","dependencies":["y"]}`,
			wantCount: 1,
			wantFirst: "Full",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := parseJSONL([]byte(tt.input))
			if len(issues) != tt.wantCount {
				t.Fatalf("parseJSONL returned %d issues, want %d", len(issues), tt.wantCount)
			}
			if tt.wantCount > 0 {
				if issues[0].Title != tt.wantFirst {
					t.Errorf("first issue title = %q, want %q", issues[0].Title, tt.wantFirst)
				}
				// All parsed issues should have source tagged
				for i, issue := range issues {
					if issue.Source != "beads" {
						t.Errorf("issues[%d].Source = %q, want %q", i, issue.Source, "beads")
					}
				}
			}
		})
	}
}

func TestParseJSONLPreservesFields(t *testing.T) {
	input := `{"id":"bc-123","title":"Test issue","description":"A test","status":"open","priority":"high","assignee":"bob","issue_type":"task","dependencies":["bc-100","bc-101"]}`
	issues := parseJSONL([]byte(input))

	if len(issues) != 1 {
		t.Fatalf("got %d issues, want 1", len(issues))
	}

	issue := issues[0]
	if issue.ID != "bc-123" {
		t.Errorf("ID = %q, want %q", issue.ID, "bc-123")
	}
	if issue.Title != "Test issue" {
		t.Errorf("Title = %q, want %q", issue.Title, "Test issue")
	}
	if issue.Description != "A test" {
		t.Errorf("Description = %q, want %q", issue.Description, "A test")
	}
	if issue.Status != "open" {
		t.Errorf("Status = %q, want %q", issue.Status, "open")
	}
	if issue.Assignee != "bob" {
		t.Errorf("Assignee = %q, want %q", issue.Assignee, "bob")
	}
	if issue.Type != "task" {
		t.Errorf("Type = %q, want %q", issue.Type, "task")
	}
	if len(issue.Dependencies) != 2 {
		t.Errorf("Dependencies len = %d, want 2", len(issue.Dependencies))
	}
	if issue.Source != "beads" {
		t.Errorf("Source = %q, want %q", issue.Source, "beads")
	}
}

// --- ListAllIssues ---

func TestListAllIssuesNoBeadsDir(t *testing.T) {
	dir := t.TempDir()
	// No .beads directory — should return ErrNoBeadsDir
	issues, err := ListAllIssues(dir)
	if !errors.Is(err, ErrNoBeadsDir) {
		t.Errorf("ListAllIssues without .beads should return ErrNoBeadsDir, got %v", err)
	}
	if issues != nil {
		t.Errorf("ListAllIssues without .beads should return nil issues, got %d", len(issues))
	}
}

func TestListIssuesNoBeadsDir(t *testing.T) {
	dir := t.TempDir()
	issues, err := ListIssues(dir)
	if !errors.Is(err, ErrNoBeadsDir) {
		t.Errorf("ListIssues without .beads should return ErrNoBeadsDir, got %v", err)
	}
	if issues != nil {
		t.Errorf("ListIssues without .beads should return nil issues, got %d", len(issues))
	}
}

func TestReadyIssuesNoBeadsDir(t *testing.T) {
	dir := t.TempDir()
	issues := ReadyIssues(dir)
	if issues != nil {
		t.Errorf("ReadyIssues without .beads should return nil, got %d issues", len(issues))
	}
}

// --- ListIssues filtering ---

func TestListIssuesFiltersEpics(t *testing.T) {
	// Create a workspace with .beads dir and a mock bd script
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	// Create mock bd that outputs JSON with mixed types
	issues := []Issue{
		{ID: "a", Title: "Epic task", Status: "open", Type: "epic"},
		{ID: "b", Title: "Bug fix", Status: "open", Type: "bug"},
		{ID: "c", Title: "Regular task", Status: "open", Type: "task"},
	}
	data, _ := json.Marshal(issues)

	mockBd := createMockBd(t, string(data))
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListIssues(dir)
	if err != nil {
		t.Fatalf("ListIssues unexpected error: %v", err)
	}

	// Should filter out the epic
	if len(result) != 2 {
		t.Fatalf("ListIssues returned %d issues, want 2 (epic filtered)", len(result))
	}
	for _, issue := range result {
		if issue.Type == "epic" {
			t.Error("ListIssues should filter out epics")
		}
	}
}

func TestListAllIssuesIncludesEpics(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	issues := []Issue{
		{ID: "a", Title: "Epic task", Status: "open", Type: "epic"},
		{ID: "b", Title: "Bug fix", Status: "open", Type: "bug"},
	}
	data, _ := json.Marshal(issues)

	mockBd := createMockBd(t, string(data))
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListAllIssues(dir)
	if err != nil {
		t.Fatalf("ListAllIssues unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("ListAllIssues returned %d issues, want 2", len(result))
	}

	// All should have source tagged
	for _, issue := range result {
		if issue.Source != "beads" {
			t.Errorf("issue %q Source = %q, want %q", issue.ID, issue.Source, "beads")
		}
	}
}

func TestListAllIssuesJSONLFallback(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	// Output JSONL instead of a JSON array — the function should fall back to parseJSONL
	jsonl := "{\"id\":\"a\",\"title\":\"First\",\"status\":\"open\"}\n{\"id\":\"b\",\"title\":\"Second\",\"status\":\"open\"}\n"

	mockBd := createMockBd(t, jsonl)
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListAllIssues(dir)
	if err != nil {
		t.Fatalf("ListAllIssues JSONL unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("ListAllIssues with JSONL returned %d issues, want 2", len(result))
	}
	if result[0].Title != "First" {
		t.Errorf("first issue title = %q, want %q", result[0].Title, "First")
	}
}

func TestListAllIssuesEmptyOutput(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBd(t, "[]")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListAllIssues(dir)
	if err != nil {
		t.Fatalf("ListAllIssues empty unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("ListAllIssues with empty array returned %d issues, want 0", len(result))
	}
}

func TestListAllIssuesMalformedOutput(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	// Totally broken output — not valid JSON or JSONL
	mockBd := createMockBd(t, "this is not json at all")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListAllIssues(dir)
	if err != nil {
		t.Fatalf("ListAllIssues malformed unexpected error: %v", err)
	}

	// Should gracefully return empty (parseJSONL will fail on non-JSON)
	if len(result) != 0 {
		t.Errorf("ListAllIssues with malformed output returned %d issues, want 0", len(result))
	}
}

// --- ReadyIssues ---

func TestReadyIssuesFiltersEpics(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	issues := []Issue{
		{ID: "a", Title: "Epic", Status: "open", Type: "epic"},
		{ID: "b", Title: "Ready bug", Status: "open", Type: "bug"},
	}
	data, _ := json.Marshal(issues)

	// ReadyIssues calls "bd ready --json", so mock needs to respond to that
	mockBd := createMockBd(t, string(data))
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := ReadyIssues(dir)

	if len(result) != 1 {
		t.Fatalf("ReadyIssues returned %d issues, want 1 (epic filtered)", len(result))
	}
	if result[0].Type == "epic" {
		t.Error("ReadyIssues should filter out epics")
	}
	if result[0].Source != "beads" {
		t.Errorf("Source = %q, want %q", result[0].Source, "beads")
	}
}

func TestReadyIssuesEmptyResult(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBd(t, "[]")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := ReadyIssues(dir)
	if result != nil {
		t.Errorf("ReadyIssues with empty array returned %v, want nil", result)
	}
}

// --- AddIssue ---

func TestAddIssueBdNotAvailable(t *testing.T) {
	// Ensure bd is not in PATH
	t.Setenv("PATH", t.TempDir())

	err := AddIssue(t.TempDir(), "Test issue", "description")
	if err == nil {
		t.Error("AddIssue should fail when bd is not available")
	}
}

func TestAddIssueWithMockBd(t *testing.T) {
	mockBd := createMockBd(t, "")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	err := AddIssue(t.TempDir(), "Test issue", "a description")
	if err != nil {
		t.Errorf("AddIssue with mock bd: %v", err)
	}
}

func TestAddIssueNoDescription(t *testing.T) {
	mockBd := createMockBd(t, "")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	err := AddIssue(t.TempDir(), "Test issue", "")
	if err != nil {
		t.Errorf("AddIssue without description: %v", err)
	}
}

// --- AssignIssue ---

func TestAssignIssueBdNotAvailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := AssignIssue(t.TempDir(), "bc-123", "agent-01")
	if err == nil {
		t.Error("AssignIssue should fail when bd is not available")
	}
}

func TestAssignIssueWithMockBd(t *testing.T) {
	mockBd := createMockBd(t, "")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	err := AssignIssue(t.TempDir(), "bc-123", "agent-01")
	if err != nil {
		t.Errorf("AssignIssue with mock bd: %v", err)
	}
}

// --- CloseIssue ---

func TestCloseIssueBdNotAvailable(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := CloseIssue(t.TempDir(), "bc-123")
	if err == nil {
		t.Error("CloseIssue should fail when bd is not available")
	}
}

func TestCloseIssueWithMockBd(t *testing.T) {
	mockBd := createMockBd(t, "")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	err := CloseIssue(t.TempDir(), "bc-123")
	if err != nil {
		t.Errorf("CloseIssue with mock bd: %v", err)
	}
}

// --- Issue struct JSON ---

func TestIssueJSONRoundTrip(t *testing.T) {
	original := Issue{
		ID:           "bc-42",
		Title:        "Test JSON",
		Description:  "Round trip test",
		Status:       "open",
		Priority:     "high",
		Assignee:     "alice",
		Type:         "bug",
		Dependencies: []string{"bc-1", "bc-2"},
		Source:       "beads",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded Issue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title = %q, want %q", decoded.Title, original.Title)
	}
	if decoded.Description != original.Description {
		t.Errorf("Description = %q, want %q", decoded.Description, original.Description)
	}
	if decoded.Status != original.Status {
		t.Errorf("Status = %q, want %q", decoded.Status, original.Status)
	}
	if decoded.Assignee != original.Assignee {
		t.Errorf("Assignee = %q, want %q", decoded.Assignee, original.Assignee)
	}
	if decoded.Type != original.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Source != original.Source {
		t.Errorf("Source = %q, want %q", decoded.Source, original.Source)
	}
	if len(decoded.Dependencies) != 2 {
		t.Errorf("Dependencies len = %d, want 2", len(decoded.Dependencies))
	}
}

func TestIssueJSONOmitsEmpty(t *testing.T) {
	issue := Issue{
		ID:     "bc-1",
		Title:  "Minimal",
		Status: "open",
		Source: "beads",
	}

	data, err := json.Marshal(issue)
	if err != nil {
		t.Fatal(err)
	}

	// Optional fields with omitempty should not be present
	var raw map[string]any
	json.Unmarshal(data, &raw)

	for _, field := range []string{"description", "priority", "assignee", "issue_type", "dependencies"} {
		if _, exists := raw[field]; exists {
			t.Errorf("expected %q to be omitted from JSON, but it was present", field)
		}
	}
}

// --- GetIssue ---

func TestGetIssueNoBeadsDir(t *testing.T) {
	dir := t.TempDir()
	// No .beads directory — should return nil
	result := GetIssue(dir, "bc-123")
	if result != nil {
		t.Errorf("GetIssue without .beads should return nil, got %+v", result)
	}
}

func TestGetIssueBdFails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBdFailing(t)
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := GetIssue(dir, "bc-123")
	if result != nil {
		t.Errorf("GetIssue when bd fails should return nil, got %+v", result)
	}
}

func TestGetIssueMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBd(t, "this is not json")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := GetIssue(dir, "bc-123")
	if result != nil {
		t.Errorf("GetIssue with malformed JSON should return nil, got %+v", result)
	}
}

func TestGetIssueEmptyArray(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBd(t, "[]")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := GetIssue(dir, "bc-123")
	if result != nil {
		t.Errorf("GetIssue with empty array should return nil, got %+v", result)
	}
}

func TestGetIssueSuccess(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	issues := []Issue{
		{ID: "bc-42", Title: "Test bug", Status: "open", Type: "bug", Assignee: "alice"},
	}
	data, _ := json.Marshal(issues)

	mockBd := createMockBd(t, string(data))
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := GetIssue(dir, "bc-42")
	if result == nil {
		t.Fatal("GetIssue should return an issue, got nil")
	}
	if result.ID != "bc-42" {
		t.Errorf("ID = %q, want %q", result.ID, "bc-42")
	}
	if result.Title != "Test bug" {
		t.Errorf("Title = %q, want %q", result.Title, "Test bug")
	}
	if result.Status != "open" {
		t.Errorf("Status = %q, want %q", result.Status, "open")
	}
	if result.Source != "beads" {
		t.Errorf("Source = %q, want %q", result.Source, "beads")
	}
}

func TestGetIssueReturnsFirst(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	issues := []Issue{
		{ID: "bc-1", Title: "First", Status: "open"},
		{ID: "bc-2", Title: "Second", Status: "open"},
	}
	data, _ := json.Marshal(issues)

	mockBd := createMockBd(t, string(data))
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := GetIssue(dir, "bc-1")
	if result == nil {
		t.Fatal("GetIssue should return an issue, got nil")
	}
	if result.ID != "bc-1" {
		t.Errorf("ID = %q, want %q", result.ID, "bc-1")
	}
	if result.Title != "First" {
		t.Errorf("Title = %q, want %q", result.Title, "First")
	}
}

func TestGetIssueEmptyID(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	// Mock bd that outputs empty array for any query
	mockBd := createMockBd(t, "[]")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := GetIssue(dir, "")
	if result != nil {
		t.Errorf("GetIssue with empty ID should return nil, got %+v", result)
	}
}

// --- ListAllIssues bd failure ---

func TestListAllIssuesBdFails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBdFailing(t)
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListAllIssues(dir)
	if err == nil {
		t.Error("ListAllIssues when bd fails should return an error")
	}
	if result != nil {
		t.Errorf("ListAllIssues when bd fails should return nil issues, got %d", len(result))
	}
}

func TestListAllIssuesBdNotFound(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	// Set PATH to empty dir so bd is not found
	t.Setenv("PATH", t.TempDir())

	result, err := ListAllIssues(dir)
	if err == nil {
		t.Fatal("ListAllIssues when bd not in PATH should return an error")
	}
	var execErr *exec.Error
	if !errors.As(err, &execErr) {
		t.Errorf("expected error wrapping exec.Error, got %T: %v", err, err)
	}
	if result != nil {
		t.Errorf("expected nil issues, got %d", len(result))
	}
}

func TestListIssuesPropagatesError(t *testing.T) {
	dir := t.TempDir()
	// No .beads dir — ListIssues should propagate ErrNoBeadsDir from ListAllIssues
	_, err := ListIssues(dir)
	if !errors.Is(err, ErrNoBeadsDir) {
		t.Errorf("ListIssues should propagate ErrNoBeadsDir, got %v", err)
	}
}

func TestListIssuesBdFails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBdFailing(t)
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result, err := ListIssues(dir)
	if err == nil {
		t.Error("ListIssues when bd fails should return an error")
	}
	if result != nil {
		t.Errorf("ListIssues when bd fails should return nil issues, got %d", len(result))
	}
}

// --- ReadyIssues coverage gaps ---

func TestReadyIssuesBdFails(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBdFailing(t)
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := ReadyIssues(dir)
	if result != nil {
		t.Errorf("ReadyIssues when bd fails should return nil, got %d issues", len(result))
	}
}

func TestReadyIssuesMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	mockBd := createMockBd(t, "not valid json")
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := ReadyIssues(dir)
	if result != nil {
		t.Errorf("ReadyIssues with malformed JSON should return nil, got %d issues", len(result))
	}
}

func TestReadyIssuesAllEpics(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".beads"), 0755)

	issues := []Issue{
		{ID: "a", Title: "Epic 1", Status: "open", Type: "epic"},
		{ID: "b", Title: "Epic 2", Status: "open", Type: "epic"},
	}
	data, _ := json.Marshal(issues)

	mockBd := createMockBd(t, string(data))
	t.Setenv("PATH", filepath.Dir(mockBd)+":"+os.Getenv("PATH"))

	result := ReadyIssues(dir)
	if result != nil {
		t.Errorf("ReadyIssues with only epics should return nil, got %d issues", len(result))
	}
}

// --- helpers ---

// createMockBdFailing creates a mock "bd" script that exits with status 1.
// Returns the path to the mock script.
func createMockBdFailing(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mockPath := filepath.Join(dir, "bd")

	script := "#!/bin/sh\nexit 1\n"
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock bd: %v", err)
	}

	return mockPath
}

// createMockBd creates a mock "bd" script that outputs the given string to stdout.
// Returns the path to the mock script.
func createMockBd(t *testing.T, output string) string {
	t.Helper()
	dir := t.TempDir()
	mockPath := filepath.Join(dir, "bd")

	// Write a shell script that just echoes the output
	script := "#!/bin/sh\nprintf '%s' " + shellQuote(output) + "\n"
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock bd: %v", err)
	}

	return mockPath
}

// shellQuote wraps a string in single quotes for safe shell embedding.
func shellQuote(s string) string {
	// Replace single quotes with '\'' (end quote, escaped quote, start quote)
	quoted := "'"
	for _, c := range s {
		if c == '\'' {
			quoted += "'\\''"
		} else {
			quoted += string(c)
		}
	}
	quoted += "'"
	return quoted
}
