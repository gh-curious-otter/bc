package cmd

import (
	"testing"

	"github.com/rpuneet/bc/pkg/agent"
)

func TestParseRole_ValidRoles(t *testing.T) {
	// All roles are custom now - parseRole accepts any valid role name
	tests := []struct {
		input string
		want  agent.Role
	}{
		{"worker", agent.Role("worker")},
		{"engineer", agent.Role("engineer")},
		{"manager", agent.Role("manager")},
		{"product-manager", agent.Role("product-manager")},
		{"coordinator", agent.Role("coordinator")}, // No special handling
		{"qa", agent.Role("qa")},
		{"custom-role", agent.Role("custom-role")}, // Any valid name accepted
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

func TestParseRole_NoAliases(t *testing.T) {
	// Legacy aliases are no longer supported - roles are custom now
	// Input is returned as-is (lowercased)
	tests := []struct {
		input string
		want  agent.Role
	}{
		{"pm", agent.Role("pm")},       // No expansion to product-manager
		{"coord", agent.Role("coord")}, // No expansion to root
		{"tl", agent.Role("tl")},       // Any short name is valid
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
	// Input is lowercased
	tests := []struct {
		input string
		want  agent.Role
	}{
		{"Worker", agent.Role("worker")},
		{"ENGINEER", agent.Role("engineer")},
		{"Manager", agent.Role("manager")},
		{"Product-Manager", agent.Role("product-manager")},
		{"PM", agent.Role("pm")},       // No alias expansion, just lowercase
		{"COORD", agent.Role("coord")}, // No alias expansion, just lowercase
		{"QA", agent.Role("qa")},
		{"Qa", agent.Role("qa")},
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
	// Only truly invalid role names should error (format validation)
	// Any alphanumeric name with hyphens is valid (roles are custom)
	invalid := []struct {
		input string
		desc  string
	}{
		{"role@invalid", "contains @ symbol"},
		{"role with spaces", "contains spaces"},
		{"role!name", "contains exclamation"},
	}

	for _, tt := range invalid {
		t.Run(tt.desc, func(t *testing.T) {
			_, err := parseRole(tt.input)
			if err == nil {
				t.Fatalf("parseRole(%q) should have returned error", tt.input)
			}
		})
	}
}

func TestParseRole_EmptyDefaultsToRoot(t *testing.T) {
	// Empty role defaults to root
	got, err := parseRole("")
	if err != nil {
		t.Fatalf("parseRole(\"\") returned error: %v", err)
	}
	if got != agent.RoleRoot {
		t.Errorf("parseRole(\"\") = %q, want %q", got, agent.RoleRoot)
	}
}

func TestParseRole_ValidCustomRoles(t *testing.T) {
	// Any valid alphanumeric name is accepted (roles are custom)
	custom := []string{
		"admin",
		"supervisor",
		"developer",
		"tester",
		"my-custom-role",
		"role123",
	}

	for _, input := range custom {
		t.Run(input, func(t *testing.T) {
			got, err := parseRole(input)
			if err != nil {
				t.Fatalf("parseRole(%q) should succeed for custom role, got error: %v", input, err)
			}
			if got != agent.Role(input) {
				t.Errorf("parseRole(%q) = %q, want %q", input, got, input)
			}
		})
	}
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
