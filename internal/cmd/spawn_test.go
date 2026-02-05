package cmd

import (
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

func TestParseRole_ValidRoles(t *testing.T) {
	tests := []struct {
		input string
		want  agent.Role
	}{
		{"worker", agent.RoleWorker},
		{"engineer", agent.RoleEngineer},
		{"manager", agent.RoleManager},
		{"product-manager", agent.RoleProductManager},
		{"coordinator", agent.RoleCoordinator},
		{"qa", agent.RoleQA},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRole(tt.input)
			if err != nil {
				t.Fatalf("parseRole(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRole_Aliases(t *testing.T) {
	tests := []struct {
		input string
		want  agent.Role
	}{
		{"pm", agent.RoleProductManager},
		{"coord", agent.RoleCoordinator},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRole(tt.input)
			if err != nil {
				t.Fatalf("parseRole(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRole_CaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  agent.Role
	}{
		{"Worker", agent.RoleWorker},
		{"ENGINEER", agent.RoleEngineer},
		{"Manager", agent.RoleManager},
		{"Product-Manager", agent.RoleProductManager},
		{"PM", agent.RoleProductManager},
		{"COORD", agent.RoleCoordinator},
		{"QA", agent.RoleQA},
		{"Qa", agent.RoleQA},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseRole(tt.input)
			if err != nil {
				t.Fatalf("parseRole(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseRole(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseRole_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"admin",
		"supervisor",
		"developer",
		"tester",
		"unknown",
		"work",
		"eng",
	}

	for _, input := range invalid {
		t.Run(input, func(t *testing.T) {
			_, err := parseRole(input)
			if err == nil {
				t.Fatalf("parseRole(%q) should have returned error", input)
			}
			// Error message should mention "unknown role"
			if got := err.Error(); !contains(got, "unknown role") {
				t.Errorf("error should mention 'unknown role', got: %s", got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
