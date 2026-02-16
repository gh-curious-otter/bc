package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRoleFile_WithFrontmatter(t *testing.T) {
	content := `---
name: engineer
capabilities:
  - implement_tasks
  - write_tests
parent_roles:
  - manager
---

# Engineer Role

You are an engineer agent.
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile failed: %v", err)
	}

	if role.Metadata.Name != "engineer" {
		t.Errorf("Name = %q, want %q", role.Metadata.Name, "engineer")
	}

	if len(role.Metadata.Capabilities) != 2 {
		t.Errorf("Capabilities len = %d, want 2", len(role.Metadata.Capabilities))
	}

	if role.Metadata.Capabilities[0] != "implement_tasks" {
		t.Errorf("Capabilities[0] = %q, want %q", role.Metadata.Capabilities[0], "implement_tasks")
	}

	if len(role.Metadata.ParentRoles) != 1 || role.Metadata.ParentRoles[0] != "manager" {
		t.Errorf("ParentRoles = %v, want [manager]", role.Metadata.ParentRoles)
	}

	expectedPrompt := "# Engineer Role\n\nYou are an engineer agent."
	if role.Prompt != expectedPrompt {
		t.Errorf("Prompt = %q, want %q", role.Prompt, expectedPrompt)
	}
}

func TestParseRoleFile_WithSingleton(t *testing.T) {
	content := `---
name: root
is_singleton: true
---

# Root Agent
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile failed: %v", err)
	}

	if !role.Metadata.IsSingleton {
		t.Error("IsSingleton should be true")
	}
}

func TestParseRoleFile_NoFrontmatter(t *testing.T) {
	content := `# Simple Role

Just a markdown file without frontmatter.
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile failed: %v", err)
	}

	if role.Metadata.Name != "" {
		t.Errorf("Name should be empty, got %q", role.Metadata.Name)
	}

	if role.Prompt == "" {
		t.Error("Prompt should not be empty")
	}
}

func TestParseRoleFile_InvalidYAML(t *testing.T) {
	content := `---
name: [invalid yaml
---

Body
`

	_, err := ParseRoleFile([]byte(content))
	if err == nil {
		t.Error("ParseRoleFile should fail on invalid YAML")
	}
}

func TestParseRoleFile_UnclosedFrontmatter(t *testing.T) {
	content := `---
name: test

No closing delimiter, treat as plain markdown.
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile should handle unclosed frontmatter: %v", err)
	}

	// Should treat entire content as prompt
	if role.Metadata.Name != "" {
		t.Errorf("Name should be empty for unclosed frontmatter, got %q", role.Metadata.Name)
	}
}

func TestRoleManager_EnsureDefaultRoot(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")

	rm := NewRoleManager(stateDir)

	// First call should create
	created, err := rm.EnsureDefaultRoot()
	if err != nil {
		t.Fatalf("EnsureDefaultRoot failed: %v", err)
	}
	if !created {
		t.Error("First call should report created=true")
	}

	// Verify file exists
	rootPath := filepath.Join(rm.RolesDir(), "root.md")
	if _, statErr := os.Stat(rootPath); statErr != nil {
		t.Errorf("root.md should exist: %v", statErr)
	}

	// Second call should not create
	created, err = rm.EnsureDefaultRoot()
	if err != nil {
		t.Fatalf("Second EnsureDefaultRoot failed: %v", err)
	}
	if created {
		t.Error("Second call should report created=false")
	}
}

func TestRoleManager_LoadRole(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	rolesDir := filepath.Join(stateDir, "roles")

	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create a test role
	roleContent := `---
name: qa
capabilities:
  - run_tests
  - validate_fixes
---

# QA Role

You are a QA agent.
`
	if err := os.WriteFile(filepath.Join(rolesDir, "qa.md"), []byte(roleContent), 0600); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)

	role, err := rm.LoadRole("qa")
	if err != nil {
		t.Fatalf("LoadRole failed: %v", err)
	}

	if role.Metadata.Name != "qa" {
		t.Errorf("Name = %q, want %q", role.Metadata.Name, "qa")
	}

	if len(role.Metadata.Capabilities) != 2 {
		t.Errorf("Capabilities len = %d, want 2", len(role.Metadata.Capabilities))
	}

	// Should be cached
	role2, ok := rm.GetRole("qa")
	if !ok {
		t.Error("Role should be cached")
	}
	if role2 != role {
		t.Error("Cached role should be same pointer")
	}
}

func TestRoleManager_LoadAllRoles(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	rolesDir := filepath.Join(stateDir, "roles")

	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create multiple roles
	roles := map[string]string{
		"engineer.md": `---
name: engineer
---

Engineer prompt.
`,
		"manager.md": `---
name: manager
---

Manager prompt.
`,
	}

	for name, content := range roles {
		if err := os.WriteFile(filepath.Join(rolesDir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	rm := NewRoleManager(stateDir)

	loadedRoles, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	// Should have engineer, manager, and auto-generated root
	if len(loadedRoles) != 3 {
		t.Errorf("Should have 3 roles (engineer, manager, root), got %d", len(loadedRoles))
	}

	if _, ok := loadedRoles["engineer"]; !ok {
		t.Error("Should have engineer role")
	}
	if _, ok := loadedRoles["manager"]; !ok {
		t.Error("Should have manager role")
	}
	if _, ok := loadedRoles["root"]; !ok {
		t.Error("Should have root role (auto-generated)")
	}
}

func TestRoleManager_HasRole(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	rolesDir := filepath.Join(stateDir, "roles")

	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create one role file
	if err := os.WriteFile(filepath.Join(rolesDir, "test.md"), []byte("---\nname: test\n---\n"), 0600); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)

	if !rm.HasRole("test") {
		t.Error("HasRole should return true for existing role")
	}

	if rm.HasRole("nonexistent") {
		t.Error("HasRole should return false for nonexistent role")
	}
}

func TestRoleManager_WriteRole(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")

	rm := NewRoleManager(stateDir)

	role := &Role{
		Metadata: RoleMetadata{
			Name:         "custom",
			Capabilities: []string{"custom_capability"},
			ParentRoles:  []string{"root"},
		},
		Prompt: "# Custom Role\n\nCustom prompt content.",
	}

	if err := rm.WriteRole(role); err != nil {
		t.Fatalf("WriteRole failed: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(rm.RolesDir(), "custom.md")
	if _, err := os.Stat(filePath); err != nil {
		t.Errorf("Role file should exist: %v", err)
	}

	// Reload and verify
	rm2 := NewRoleManager(stateDir)
	loaded, err := rm2.LoadRole("custom")
	if err != nil {
		t.Fatalf("Failed to reload role: %v", err)
	}

	if loaded.Metadata.Name != "custom" {
		t.Errorf("Name = %q, want %q", loaded.Metadata.Name, "custom")
	}

	if len(loaded.Metadata.Capabilities) != 1 {
		t.Errorf("Capabilities len = %d, want 1", len(loaded.Metadata.Capabilities))
	}
}

func TestRoleManager_WriteRole_NoName(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")

	rm := NewRoleManager(stateDir)

	role := &Role{
		Prompt: "No name provided",
	}

	err := rm.WriteRole(role)
	if err == nil {
		t.Error("WriteRole should fail without name")
	}
}

func TestRole_Description(t *testing.T) {
	// Test metadata description takes precedence
	t.Run("uses metadata description", func(t *testing.T) {
		role := Role{
			Metadata: RoleMetadata{Description: "Custom description"},
			Prompt:   "# Heading\n\nContent",
		}
		if got := role.Description(); got != "Custom description" {
			t.Errorf("Description() = %q, want %q", got, "Custom description")
		}
	})

	// Test extracts from first heading
	t.Run("extracts from heading", func(t *testing.T) {
		role := Role{Prompt: "# Engineer Agent\n\nYou are an engineer."}
		if got := role.Description(); got != "Engineer Agent" {
			t.Errorf("Description() = %q, want %q", got, "Engineer Agent")
		}
	})

	// Test handles no heading gracefully
	t.Run("handles no heading", func(t *testing.T) {
		role := Role{Prompt: "Just some content"}
		if got := role.Description(); got != "" {
			t.Errorf("Description() = %q, want empty string", got)
		}
	})

	// Test metadata takes precedence over prompt heading
	t.Run("metadata takes precedence", func(t *testing.T) {
		role := Role{
			Metadata: RoleMetadata{Description: "Metadata wins"},
			Prompt:   "# Prompt heading",
		}
		if got := role.Description(); got != "Metadata wins" {
			t.Errorf("Description() = %q, want %q", got, "Metadata wins")
		}
	})
}

func TestFormatRoleFile(t *testing.T) {
	role := &Role{
		Metadata: RoleMetadata{
			Name:         "test",
			Capabilities: []string{"cap1", "cap2"},
			IsSingleton:  true,
		},
		Prompt: "# Test Role\n\nTest content.",
	}

	content, err := FormatRoleFile(role)
	if err != nil {
		t.Fatalf("FormatRoleFile failed: %v", err)
	}

	// Should be parseable
	parsed, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("Failed to parse formatted content: %v", err)
	}

	if parsed.Metadata.Name != role.Metadata.Name {
		t.Errorf("Round-trip Name = %q, want %q", parsed.Metadata.Name, role.Metadata.Name)
	}

	if !parsed.Metadata.IsSingleton {
		t.Error("Round-trip IsSingleton should be true")
	}

	if len(parsed.Metadata.Capabilities) != 2 {
		t.Errorf("Round-trip Capabilities len = %d, want 2", len(parsed.Metadata.Capabilities))
	}
}

func TestDefaultRootRole_Parseable(t *testing.T) {
	role, err := ParseRoleFile([]byte(DefaultRootRole))
	if err != nil {
		t.Fatalf("DefaultRootRole should be parseable: %v", err)
	}

	if role.Metadata.Name != "root" {
		t.Errorf("Name = %q, want %q", role.Metadata.Name, "root")
	}

	if !role.Metadata.IsSingleton {
		t.Error("IsSingleton should be true")
	}

	if role.Prompt == "" {
		t.Error("Prompt should not be empty")
	}
}
