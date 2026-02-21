package cmd

import "testing"

func TestIsValidIssueType(t *testing.T) {
	tests := []struct {
		name     string
		typeArg  string
		expected bool
	}{
		{"bug is valid", "bug", true},
		{"enhancement is valid", "enhancement", true},
		{"test-failure is valid", "test-failure", true},
		{"feature is valid", "feature", true},
		{"documentation is valid", "documentation", true},
		{"epic is valid", "epic", true},
		{"task is valid", "task", true},
		{"chore is valid", "chore", true},
		{"invalid type", "invalid", false},
		{"empty type", "", false},
		{"typo type", "bugg", false},
		{"case sensitive", "Bug", false},
		{"special chars", "bug!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidIssueType(tt.typeArg)
			if result != tt.expected {
				t.Errorf("isValidIssueType(%q) = %v, want %v", tt.typeArg, result, tt.expected)
			}
		})
	}
}

func TestIsValidSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		expected bool
	}{
		{"critical is valid", "critical", true},
		{"high is valid", "high", true},
		{"medium is valid", "medium", true},
		{"low is valid", "low", true},
		{"invalid severity", "urgent", false},
		{"empty severity", "", false},
		{"typo severity", "hig", false},
		{"case sensitive", "High", false},
		{"numeric severity", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSeverity(tt.severity)
			if result != tt.expected {
				t.Errorf("isValidSeverity(%q) = %v, want %v", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestValidIssueTypesSlice(t *testing.T) {
	// Ensure validIssueTypes contains expected values
	expected := map[string]bool{
		"bug":           true,
		"enhancement":   true,
		"test-failure":  true,
		"feature":       true,
		"documentation": true,
		"epic":          true,
		"task":          true,
		"chore":         true,
	}

	for _, validType := range validIssueTypes {
		if !expected[validType] {
			t.Errorf("unexpected type in validIssueTypes: %q", validType)
		}
		delete(expected, validType)
	}

	if len(expected) > 0 {
		t.Errorf("missing types in validIssueTypes: %v", expected)
	}
}

func TestValidSeveritiesSlice(t *testing.T) {
	// Ensure validSeverities contains expected values
	expected := map[string]bool{
		"critical": true,
		"high":     true,
		"medium":   true,
		"low":      true,
	}

	for _, sev := range validSeverities {
		if !expected[sev] {
			t.Errorf("unexpected severity in validSeverities: %q", sev)
		}
		delete(expected, sev)
	}

	if len(expected) > 0 {
		t.Errorf("missing severities in validSeverities: %v", expected)
	}
}

func TestCreateIssueFromTestFailure(t *testing.T) {
	testName := "TestSomething"
	output := "failed: expected true, got false"
	reproduction := "go test -run TestSomething"

	issue, err := CreateIssueFromTestFailure(testName, output, reproduction)
	if err != nil {
		t.Fatalf("CreateIssueFromTestFailure() error = %v", err)
	}

	if issue == nil {
		t.Fatal("CreateIssueFromTestFailure() returned nil")
	}

	expectedTitle := "Test failure: TestSomething"
	if issue.Title != expectedTitle {
		t.Errorf("Title = %q, want %q", issue.Title, expectedTitle)
	}

	if issue.Type != "bug" {
		t.Errorf("Type = %q, want %q", issue.Type, "bug")
	}

	if issue.Severity != "high" {
		t.Errorf("Severity = %q, want %q", issue.Severity, "high")
	}

	if issue.TestFailure != testName {
		t.Errorf("TestFailure = %q, want %q", issue.TestFailure, testName)
	}

	// Check labels
	expectedLabels := []string{"bug", "test-failure", "automated"}
	if len(issue.Labels) != len(expectedLabels) {
		t.Errorf("Labels len = %d, want %d", len(issue.Labels), len(expectedLabels))
	}
}
