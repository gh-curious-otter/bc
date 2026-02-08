package github

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// --- HasGitRemote ---

func TestHasGitRemoteNoRepo(t *testing.T) {
	dir := t.TempDir()
	if HasGitRemote(dir) {
		t.Error("HasGitRemote should return false for non-git directory")
	}
}

func TestHasGitRemoteNoOrigin(t *testing.T) {
	dir := t.TempDir()
	// Init a git repo but don't add a remote
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	if HasGitRemote(dir) {
		t.Error("HasGitRemote should return false for repo without origin")
	}
}

func TestHasGitRemoteWithOrigin(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	addRemote := exec.CommandContext(context.Background(), "git", "remote", "add", "origin", "https://example.com/repo.git")
	addRemote.Dir = dir
	if err := addRemote.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	if !HasGitRemote(dir) {
		t.Error("HasGitRemote should return true for repo with origin")
	}
}

// --- ListIssues ---

func TestListIssuesNoRemote(t *testing.T) {
	dir := t.TempDir()
	issues, err := ListIssues(dir)
	if err != ErrNoGitRemote {
		t.Errorf("ListIssues without remote should return ErrNoGitRemote, got %v", err)
	}
	if issues != nil {
		t.Errorf("ListIssues without remote should return nil issues, got %d issues", len(issues))
	}
}

func TestListIssuesWithMockGh(t *testing.T) {
	dir := setupGitRepo(t)
	// Use raw JSON to avoid struct tag mismatch with ghIssue's anonymous labels struct
	data := []byte(`[
		{"number":1,"title":"Bug report","state":"OPEN","labels":[{"name":"bug"},{"name":"p1"}]},
		{"number":2,"title":"Feature request","state":"OPEN","labels":null},
		{"number":3,"title":"Closed issue","state":"CLOSED","labels":[{"name":"done"}]}
	]`)

	mockGh := createMockCLI(t, "gh", string(data))
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	issues, err := ListIssues(dir)
	if err != nil {
		t.Fatalf("ListIssues returned error: %v", err)
	}
	if len(issues) != 3 {
		t.Fatalf("ListIssues returned %d issues, want 3", len(issues))
	}

	// Check first issue
	if issues[0].Number != 1 {
		t.Errorf("issues[0].Number = %d, want 1", issues[0].Number)
	}
	if issues[0].Title != "Bug report" {
		t.Errorf("issues[0].Title = %q, want %q", issues[0].Title, "Bug report")
	}
	if issues[0].Source != "github" {
		t.Errorf("issues[0].Source = %q, want %q", issues[0].Source, "github")
	}
	// Labels should be flattened
	if len(issues[0].Labels) != 2 || issues[0].Labels[0] != "bug" || issues[0].Labels[1] != "p1" {
		t.Errorf("issues[0].Labels = %v, want [bug, p1]", issues[0].Labels)
	}

	// Issue without labels
	if issues[1].Labels != nil {
		t.Errorf("issues[1].Labels = %v, want nil", issues[1].Labels)
	}
}

func TestListIssuesEmptyResult(t *testing.T) {
	dir := setupGitRepo(t)
	mockGh := createMockCLI(t, "gh", "[]")
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	issues, err := ListIssues(dir)
	if err != nil {
		t.Errorf("ListIssues returned error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("ListIssues with empty array returned %d issues, want 0", len(issues))
	}
}

func TestListIssuesMalformedOutput(t *testing.T) {
	dir := setupGitRepo(t)
	mockGh := createMockCLI(t, "gh", "not json")
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	issues, err := ListIssues(dir)
	if err == nil {
		t.Error("ListIssues with malformed output should return error")
	}
	if issues != nil {
		t.Errorf("ListIssues with malformed output should return nil, got %d", len(issues))
	}
}

// --- ListPRs ---

func TestListPRsNoRemote(t *testing.T) {
	dir := t.TempDir()
	prs, err := ListPRs(dir)
	if err != ErrNoGitRemote {
		t.Errorf("ListPRs without remote should return ErrNoGitRemote, got %v", err)
	}
	if prs != nil {
		t.Errorf("ListPRs without remote should return nil, got %d PRs", len(prs))
	}
}

func TestListPRsWithMockGh(t *testing.T) {
	dir := setupGitRepo(t)
	data := `[
		{"number":10,"title":"Add feature","state":"OPEN","reviewDecision":"APPROVED","isDraft":false},
		{"number":11,"title":"WIP fix","state":"OPEN","reviewDecision":"","isDraft":true}
	]`

	mockGh := createMockCLI(t, "gh", data)
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	prs, err := ListPRs(dir)
	if err != nil {
		t.Fatalf("ListPRs returned error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("ListPRs returned %d PRs, want 2", len(prs))
	}

	if prs[0].Number != 10 {
		t.Errorf("prs[0].Number = %d, want 10", prs[0].Number)
	}
	if prs[0].ReviewDecision != "APPROVED" {
		t.Errorf("prs[0].ReviewDecision = %q, want %q", prs[0].ReviewDecision, "APPROVED")
	}
	if prs[0].IsDraft {
		t.Error("prs[0].IsDraft should be false")
	}
	if !prs[1].IsDraft {
		t.Error("prs[1].IsDraft should be true")
	}
}

func TestListPRsEmptyResult(t *testing.T) {
	dir := setupGitRepo(t)
	mockGh := createMockCLI(t, "gh", "[]")
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	prs, err := ListPRs(dir)
	if err != nil {
		t.Errorf("ListPRs returned error: %v", err)
	}
	if len(prs) != 0 {
		t.Errorf("ListPRs with empty array returned %d PRs, want 0", len(prs))
	}
}

// --- CreateIssue ---

func TestCreateIssueNoGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	err := CreateIssue(t.TempDir(), "title", "body")
	if err == nil {
		t.Error("CreateIssue should fail when gh is not available")
	}
}

func TestCreateIssueWithMockGh(t *testing.T) {
	mockGh := createMockCLI(t, "gh", "")
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	err := CreateIssue(t.TempDir(), "Test issue", "body text")
	if err != nil {
		t.Errorf("CreateIssue with mock gh: %v", err)
	}
}

func TestCreateIssueNoBody(t *testing.T) {
	mockGh := createMockCLI(t, "gh", "")
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	err := CreateIssue(t.TempDir(), "Title only", "")
	if err != nil {
		t.Errorf("CreateIssue without body: %v", err)
	}
}

// --- JSON round-trip ---

func TestIssueJSON(t *testing.T) {
	original := Issue{
		Number: 42,
		Title:  "Test",
		State:  "OPEN",
		Labels: []string{"bug", "p1"},
		Source: "github",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded Issue
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Number != original.Number {
		t.Errorf("Number = %d, want %d", decoded.Number, original.Number)
	}
	if decoded.Source != "github" {
		t.Errorf("Source = %q, want %q", decoded.Source, "github")
	}
	if len(decoded.Labels) != 2 {
		t.Errorf("Labels len = %d, want 2", len(decoded.Labels))
	}
}

func TestPRJSON(t *testing.T) {
	original := PR{
		Number:         7,
		Title:          "PR test",
		State:          "MERGED",
		ReviewDecision: "APPROVED",
		IsDraft:        false,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded PR
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded.Number != 7 {
		t.Errorf("Number = %d, want 7", decoded.Number)
	}
	if decoded.State != "MERGED" {
		t.Errorf("State = %q, want %q", decoded.State, "MERGED")
	}
}

// --- helpers ---

// setupGitRepo creates a temp git repo with an origin remote.
func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	cmd := exec.CommandContext(context.Background(), "git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	addRemote := exec.CommandContext(context.Background(), "git", "remote", "add", "origin", "https://example.com/repo.git")
	addRemote.Dir = dir
	if err := addRemote.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	return dir
}

// createMockCLI creates a mock CLI script that outputs the given string.
func createMockCLI(t *testing.T, name, output string) string {
	t.Helper()
	dir := t.TempDir()
	mockPath := filepath.Join(dir, name)

	script := "#!/bin/sh\nprintf '%s' " + shellQuote(output) + "\n"
	if err := os.WriteFile(mockPath, []byte(script), 0700); err != nil { //nolint:gosec // executable script
		t.Fatalf("failed to create mock %s: %v", name, err)
	}
	return mockPath
}

func shellQuote(s string) string {
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

// --- AuthStatus ---

func TestAuthStatusNoGh(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := AuthStatus()
	if err == nil {
		t.Error("AuthStatus should fail when gh is not available")
	}
}

func TestAuthStatusWithMockGh(t *testing.T) {
	data := `{"hosts":{"github.com":[{"host":"github.com","login":"testuser","state":"success","active":true,"tokenSource":"keyring","scopes":"repo, read:org","gitProtocol":"ssh"}]}}`
	mockGh := createMockCLI(t, "gh", data)
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	result, err := AuthStatus()
	if err != nil {
		t.Fatalf("AuthStatus: %v", err)
	}
	if !result.LoggedIn {
		t.Error("AuthStatus should report logged in when state is success")
	}
	if len(result.Accounts) != 1 {
		t.Fatalf("Accounts len = %d, want 1", len(result.Accounts))
	}
	if result.Accounts[0].Login != "testuser" {
		t.Errorf("Accounts[0].Login = %q, want testuser", result.Accounts[0].Login)
	}
	if result.Accounts[0].TokenSource != "keyring" {
		t.Errorf("Accounts[0].TokenSource = %q, want keyring", result.Accounts[0].TokenSource)
	}
}

func TestAuthStatusNotLoggedIn(t *testing.T) {
	data := `{"hosts":{}}`
	mockGh := createMockCLI(t, "gh", data)
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	result, err := AuthStatus()
	if err != nil {
		t.Fatalf("AuthStatus: %v", err)
	}
	if result.LoggedIn {
		t.Error("AuthStatus should report not logged in when no hosts")
	}
	if len(result.Accounts) != 0 {
		t.Errorf("Accounts len = %d, want 0", len(result.Accounts))
	}
}

func TestAuthStatusMalformedJSON(t *testing.T) {
	mockGh := createMockCLI(t, "gh", "not json")
	t.Setenv("PATH", filepath.Dir(mockGh)+":"+os.Getenv("PATH"))

	_, err := AuthStatus()
	if err == nil {
		t.Error("AuthStatus should fail on malformed JSON")
	}
}

func TestTokenStorageInfo(t *testing.T) {
	info := TokenStorageInfo()
	if info == "" {
		t.Error("TokenStorageInfo should return non-empty string")
	}
	if !strings.Contains(info, "gh") {
		t.Errorf("TokenStorageInfo should mention gh: %q", info)
	}
}
