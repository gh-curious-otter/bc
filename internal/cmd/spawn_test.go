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

// --- Spawn command tests ---

func TestSpawnCmd_NoArgs(t *testing.T) {
	_, cleanup := setupIntegrationWorkspace(t)
	defer cleanup()

	_, _, err := executeIntegrationCmd("spawn")
	if err == nil {
		t.Error("spawn without args should fail")
	}
}

func TestSpawnCmd_Flags(t *testing.T) {
	// Verify expected flags exist
	toolFlag := spawnCmd.Flags().Lookup("tool")
	if toolFlag == nil {
		t.Fatal("expected --tool flag")
	}
	if toolFlag.DefValue != "" {
		t.Errorf("tool default should be empty, got: %s", toolFlag.DefValue)
	}

	roleFlag := spawnCmd.Flags().Lookup("role")
	if roleFlag == nil {
		t.Fatal("expected --role flag")
	}
	if roleFlag.DefValue != "worker" {
		t.Errorf("role default should be 'worker', got: %s", roleFlag.DefValue)
	}
}

func TestSpawnCmd_DeprecationMessage(t *testing.T) {
	if spawnCmd.Deprecated == "" {
		t.Error("spawn should have deprecation message")
	}
}
