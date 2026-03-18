package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseRoleFile_WithFrontmatter(t *testing.T) {
	content := `---
name: engineer
parent_roles:
  - manager
mcp_servers:
  - bc
  - github
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

	if len(role.Metadata.ParentRoles) != 1 || role.Metadata.ParentRoles[0] != "manager" {
		t.Errorf("ParentRoles = %v, want [manager]", role.Metadata.ParentRoles)
	}

	if len(role.Metadata.MCPServers) != 2 {
		t.Errorf("MCPServers len = %d, want 2", len(role.Metadata.MCPServers))
	}

	expectedPrompt := "# Engineer Role\n\nYou are an engineer agent."
	if role.Prompt != expectedPrompt {
		t.Errorf("Prompt = %q, want %q", role.Prompt, expectedPrompt)
	}
}

func TestParseRoleFile_WithSecrets(t *testing.T) {
	content := `---
name: root
secrets:
  - GITHUB_PERSONAL_ACCESS_TOKEN
---

# Root Agent
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile failed: %v", err)
	}

	if len(role.Metadata.Secrets) != 1 || role.Metadata.Secrets[0] != "GITHUB_PERSONAL_ACCESS_TOKEN" {
		t.Errorf("Secrets = %v, want [GITHUB_PERSONAL_ACCESS_TOKEN]", role.Metadata.Secrets)
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
mcp_servers:
  - bc
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

	if len(role.Metadata.MCPServers) != 1 {
		t.Errorf("MCPServers len = %d, want 1", len(role.Metadata.MCPServers))
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

	// Should have engineer, manager, auto-generated root, and auto-generated base
	if len(loadedRoles) != 4 {
		t.Errorf("Should have 4 roles (engineer, manager, root, base), got %d", len(loadedRoles))
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
			Name:        "custom",
			MCPServers:  []string{"bc"},
			ParentRoles: []string{"root"},
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

	if len(loaded.Metadata.MCPServers) != 1 {
		t.Errorf("MCPServers len = %d, want 1", len(loaded.Metadata.MCPServers))
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
			Name:       "test",
			MCPServers: []string{"bc", "github"},
			Secrets:    []string{"TOKEN"},
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

	if len(parsed.Metadata.MCPServers) != 2 {
		t.Errorf("Round-trip MCPServers len = %d, want 2", len(parsed.Metadata.MCPServers))
	}

	if len(parsed.Metadata.Secrets) != 1 {
		t.Errorf("Round-trip Secrets len = %d, want 1", len(parsed.Metadata.Secrets))
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

	if role.Prompt == "" {
		t.Error("Prompt should not be empty")
	}

	// Verify root has secrets defined
	if len(role.Metadata.Secrets) == 0 {
		t.Error("Root role should have secrets defined")
	}

	// Verify root has MCP servers defined
	if len(role.Metadata.MCPServers) == 0 {
		t.Error("Root role should have MCP servers defined")
	}
}

func TestLoadAllRoles(t *testing.T) {
	stateDir := t.TempDir()
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatalf("failed to create roles dir: %v", err)
	}

	// Create multiple role files
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
		path := filepath.Join(rolesDir, name)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create role %s: %v", name, err)
		}
	}

	rm := NewRoleManager(stateDir)
	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	// Should have loaded engineer, manager, plus default root
	if len(loaded) < 2 {
		t.Errorf("expected at least 2 roles, got %d", len(loaded))
	}

	if _, ok := loaded["engineer"]; !ok {
		t.Error("expected engineer role to be loaded")
	}
	if _, ok := loaded["manager"]; !ok {
		t.Error("expected manager role to be loaded")
	}
}

func TestLoadAllRolesSkipsNonMd(t *testing.T) {
	stateDir := t.TempDir()
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatalf("failed to create roles dir: %v", err)
	}

	// Create a .md file and a non-.md file
	mdPath := filepath.Join(rolesDir, "test.md")
	if err := os.WriteFile(mdPath, []byte("---\nname: test\n---\nTest."), 0600); err != nil {
		t.Fatal(err)
	}

	txtPath := filepath.Join(rolesDir, "readme.txt")
	if err := os.WriteFile(txtPath, []byte("readme"), 0600); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)
	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	// Only .md files should be loaded
	if _, ok := loaded["test"]; !ok {
		t.Error("expected test role to be loaded")
	}
}

func TestLoadAllRolesSkipsDirectories(t *testing.T) {
	stateDir := t.TempDir()
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatalf("failed to create roles dir: %v", err)
	}

	// Create a subdirectory that looks like a .md file
	subDir := filepath.Join(rolesDir, "subdir.md")
	if err := os.Mkdir(subDir, 0750); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)
	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	// Should not crash on directory with .md extension
	// Just verify it loaded something (at least root)
	if len(loaded) == 0 {
		t.Error("expected at least root role to be loaded")
	}
}

func TestLoadAllRolesEmptyDir(t *testing.T) {
	stateDir := t.TempDir()
	// Don't create roles dir - LoadAllRoles should handle missing dir

	rm := NewRoleManager(stateDir)
	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	// Should still have root role (created by EnsureDefaultRoot)
	if _, ok := loaded["root"]; !ok {
		t.Error("expected root role to exist")
	}
}

func TestRoleManager_LoadRoleNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	if err := os.MkdirAll(filepath.Join(stateDir, "roles"), 0750); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)

	_, err := rm.LoadRole("nonexistent")
	if err == nil {
		t.Error("LoadRole should fail for nonexistent role")
	}
}

func TestRoleManager_LoadRoleCached(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(rolesDir, "cached.md"), []byte("---\nname: cached\n---\nPrompt."), 0600); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)

	// First load from disk
	role1, err := rm.LoadRole("cached")
	if err != nil {
		t.Fatal(err)
	}

	// Second load should return cached (same pointer)
	role2, err := rm.LoadRole("cached")
	if err != nil {
		t.Fatal(err)
	}

	if role1 != role2 {
		t.Error("second LoadRole should return cached pointer")
	}
}

func TestRoleManager_LoadRoleFromPathNameFallback(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create role with no name in frontmatter
	content := "---\nmcp_servers:\n  - bc\n---\n\n# No Name Role\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "fallback.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)
	role, err := rm.LoadRole("fallback")
	if err != nil {
		t.Fatal(err)
	}

	// Name should be derived from filename
	if role.Metadata.Name != "fallback" {
		t.Errorf("expected name 'fallback' (from filename), got %q", role.Metadata.Name)
	}
}

func TestParseRoleFile_CRLFLineEndings(t *testing.T) {
	content := "---\r\nname: crlf\r\n---\r\n\r\n# CRLF Role\r\n"

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile with CRLF failed: %v", err)
	}

	if role.Metadata.Name != "crlf" {
		t.Errorf("Name = %q, want %q", role.Metadata.Name, "crlf")
	}
}

func TestFormatRoleFile_PromptEndsWithNewline(t *testing.T) {
	role := &Role{
		Metadata: RoleMetadata{Name: "test"},
		Prompt:   "Content ending with newline.\n",
	}

	content, err := FormatRoleFile(role)
	if err != nil {
		t.Fatal(err)
	}

	// Should not double up the trailing newline
	if strings.HasSuffix(content, "\n\n\n") {
		t.Error("FormatRoleFile should not add extra trailing newlines")
	}

	// Should be round-trippable
	parsed, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Metadata.Name != "test" {
		t.Errorf("round-trip name = %q, want test", parsed.Metadata.Name)
	}
}

func TestRoleManager_HasRoleDisk(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".bc")
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	// Create role on disk but don't load it into cache
	if err := os.WriteFile(filepath.Join(rolesDir, "diskonly.md"), []byte("---\nname: diskonly\n---\nPrompt."), 0600); err != nil {
		t.Fatal(err)
	}

	rm := NewRoleManager(stateDir)

	// Should find on disk even though not cached
	if !rm.HasRole("diskonly") {
		t.Error("HasRole should return true for role on disk")
	}

	// Should return false for truly nonexistent
	if rm.HasRole("nope") {
		t.Error("HasRole should return false for nonexistent role")
	}
}

func TestParseRoleFile_WithPlugins(t *testing.T) {
	content := `---
name: feature-dev
plugins:
  - feature-dev
  - github
  - typescript-lsp
---

# Feature Developer

You are a feature developer agent.
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile failed: %v", err)
	}

	if role.Metadata.Name != "feature-dev" {
		t.Errorf("Name = %q, want %q", role.Metadata.Name, "feature-dev")
	}

	wantPlugins := []string{"feature-dev", "github", "typescript-lsp"}
	if len(role.Metadata.Plugins) != len(wantPlugins) {
		t.Errorf("Plugins len = %d, want %d", len(role.Metadata.Plugins), len(wantPlugins))
	}
	for i, p := range wantPlugins {
		if i < len(role.Metadata.Plugins) && role.Metadata.Plugins[i] != p {
			t.Errorf("Plugins[%d] = %q, want %q", i, role.Metadata.Plugins[i], p)
		}
	}
}

func TestEnsureDefaultRoles_CreatesFiles(t *testing.T) {
	dir := t.TempDir()
	rm := NewRoleManager(dir)

	created, err := rm.EnsureDefaultRoles()
	if err != nil {
		t.Fatalf("EnsureDefaultRoles: %v", err)
	}

	if len(created) != len(DefaultRoles) {
		t.Errorf("created %d roles, want %d", len(created), len(DefaultRoles))
	}

	for name := range DefaultRoles {
		rolePath := filepath.Join(rm.RolesDir(), name+".md")
		if _, statErr := os.Stat(rolePath); statErr != nil {
			t.Errorf("expected role file %s.md to exist", name)
		}
	}
}

func TestEnsureDefaultRoles_Idempotent(t *testing.T) {
	dir := t.TempDir()
	rm := NewRoleManager(dir)

	_, err := rm.EnsureDefaultRoles()
	if err != nil {
		t.Fatalf("first EnsureDefaultRoles: %v", err)
	}

	created, err := rm.EnsureDefaultRoles()
	if err != nil {
		t.Fatalf("second EnsureDefaultRoles: %v", err)
	}
	if len(created) != 0 {
		t.Errorf("second call created %v, want none", created)
	}
}

func TestEnsureDefaultRoles_ParsesCleanly(t *testing.T) {
	dir := t.TempDir()
	rm := NewRoleManager(dir)

	if _, err := rm.EnsureDefaultRoles(); err != nil {
		t.Fatalf("EnsureDefaultRoles: %v", err)
	}

	for name := range DefaultRoles {
		role, err := rm.LoadRole(name)
		if err != nil {
			t.Errorf("LoadRole(%q): %v", name, err)
			continue
		}
		if role.Metadata.Name != name {
			t.Errorf("role %q: Name = %q, want %q", name, role.Metadata.Name, name)
		}
	}
}

func TestParseRoleFile_WithLifecyclePrompts(t *testing.T) {
	content := `---
name: test
prompt_create: "Welcome, new agent."
prompt_start: "Check channels for updates."
prompt_stop: "Save your work."
prompt_delete: "Goodbye."
---

# Test Role
`

	role, err := ParseRoleFile([]byte(content))
	if err != nil {
		t.Fatalf("ParseRoleFile failed: %v", err)
	}

	if role.Metadata.PromptCreate != "Welcome, new agent." {
		t.Errorf("PromptCreate = %q, want %q", role.Metadata.PromptCreate, "Welcome, new agent.")
	}
	if role.Metadata.PromptStart != "Check channels for updates." {
		t.Errorf("PromptStart = %q, want %q", role.Metadata.PromptStart, "Check channels for updates.")
	}
	if role.Metadata.PromptStop != "Save your work." {
		t.Errorf("PromptStop = %q, want %q", role.Metadata.PromptStop, "Save your work.")
	}
	if role.Metadata.PromptDelete != "Goodbye." {
		t.Errorf("PromptDelete = %q, want %q", role.Metadata.PromptDelete, "Goodbye.")
	}
}
