package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestRoleManager creates a RoleManager backed by a SQLite store in a temp dir.
func newTestRoleManager(t *testing.T) *RoleManager {
	t.Helper()
	stateDir := t.TempDir()
	dbPath := filepath.Join(stateDir, "bc.db")
	store, err := NewRoleStore(dbPath)
	if err != nil {
		t.Fatalf("NewRoleStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return NewRoleManagerWithStore(stateDir, store)
}

// newTestRoleManagerWithDir creates a RoleManager backed by a SQLite store,
// also returning the state dir so callers can set up migration fixtures.
func newTestRoleManagerWithDir(t *testing.T) (*RoleManager, string) {
	t.Helper()
	stateDir := t.TempDir()
	dbPath := filepath.Join(stateDir, "bc.db")
	store, err := NewRoleStore(dbPath)
	if err != nil {
		t.Fatalf("NewRoleStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return NewRoleManagerWithStore(stateDir, store), stateDir
}

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
	rm := newTestRoleManager(t)

	// First call should create
	created, err := rm.EnsureDefaultRoot()
	if err != nil {
		t.Fatalf("EnsureDefaultRoot failed: %v", err)
	}
	if !created {
		t.Error("First call should report created=true")
	}

	// Verify role exists in store
	if !rm.store.Has("root") {
		t.Error("root should exist in store")
	}
	if !rm.store.Has("base") {
		t.Error("base should exist in store")
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
	rm := newTestRoleManager(t)

	// Save a test role to the store
	role := &Role{
		Metadata: RoleMetadata{
			Name:       "qa",
			MCPServers: []string{"bc"},
		},
		Prompt: "# QA Role\n\nYou are a QA agent.",
	}
	if err := rm.store.Save(role); err != nil {
		t.Fatal(err)
	}

	loaded, err := rm.LoadRole("qa")
	if err != nil {
		t.Fatalf("LoadRole failed: %v", err)
	}

	if loaded.Metadata.Name != "qa" {
		t.Errorf("Name = %q, want %q", loaded.Metadata.Name, "qa")
	}

	if len(loaded.Metadata.MCPServers) != 1 {
		t.Errorf("MCPServers len = %d, want 1", len(loaded.Metadata.MCPServers))
	}

	// Should be cached
	role2, ok := rm.GetRole("qa")
	if !ok {
		t.Error("Role should be cached")
	}
	if role2 != loaded {
		t.Error("Cached role should be same pointer")
	}
}

func TestRoleManager_LoadAllRoles(t *testing.T) {
	rm := newTestRoleManager(t)

	// Save roles to store
	roles := []*Role{
		{Metadata: RoleMetadata{Name: "engineer"}, Prompt: "Engineer prompt."},
		{Metadata: RoleMetadata{Name: "manager"}, Prompt: "Manager prompt."},
	}
	for _, r := range roles {
		if err := rm.store.Save(r); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	if len(loaded) != 2 {
		t.Errorf("Should have 2 roles, got %d", len(loaded))
	}

	if _, ok := loaded["engineer"]; !ok {
		t.Error("Should have engineer role")
	}
	if _, ok := loaded["manager"]; !ok {
		t.Error("Should have manager role")
	}
}

func TestRoleManager_HasRole(t *testing.T) {
	rm := newTestRoleManager(t)

	// Save a role to the store
	role := &Role{
		Metadata: RoleMetadata{Name: "test"},
		Prompt:   "Test.",
	}
	if err := rm.store.Save(role); err != nil {
		t.Fatal(err)
	}

	if !rm.HasRole("test") {
		t.Error("HasRole should return true for existing role")
	}

	if rm.HasRole("nonexistent") {
		t.Error("HasRole should return false for nonexistent role")
	}
}

func TestRoleManager_WriteRole(t *testing.T) {
	rm := newTestRoleManager(t)

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

	// Verify exists in store
	if !rm.store.Has("custom") {
		t.Error("Role should exist in store after WriteRole")
	}

	// Load from store and verify
	loaded, err := rm.store.Load("custom")
	if err != nil {
		t.Fatalf("Failed to load role from store: %v", err)
	}

	if loaded.Metadata.Name != "custom" {
		t.Errorf("Name = %q, want %q", loaded.Metadata.Name, "custom")
	}

	if len(loaded.Metadata.MCPServers) != 1 {
		t.Errorf("MCPServers len = %d, want 1", len(loaded.Metadata.MCPServers))
	}
}

func TestRoleManager_WriteRole_NoName(t *testing.T) {
	rm := newTestRoleManager(t)

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
	rm := newTestRoleManager(t)

	// Save roles to store
	roles := []*Role{
		{Metadata: RoleMetadata{Name: "engineer"}, Prompt: "Engineer prompt."},
		{Metadata: RoleMetadata{Name: "manager"}, Prompt: "Manager prompt."},
	}
	for _, r := range roles {
		if err := rm.store.Save(r); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

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

func TestLoadAllRolesEmpty(t *testing.T) {
	rm := newTestRoleManager(t)

	loaded, err := rm.LoadAllRoles()
	if err != nil {
		t.Fatalf("LoadAllRoles failed: %v", err)
	}

	// Empty store should return empty map
	if len(loaded) != 0 {
		t.Errorf("expected 0 roles, got %d", len(loaded))
	}
}

func TestRoleManager_LoadRoleNotFound(t *testing.T) {
	rm := newTestRoleManager(t)

	_, err := rm.LoadRole("nonexistent")
	if err == nil {
		t.Error("LoadRole should fail for nonexistent role")
	}
}

func TestRoleManager_LoadRoleCached(t *testing.T) {
	rm := newTestRoleManager(t)

	// Save a role to store
	role := &Role{
		Metadata: RoleMetadata{Name: "cached"},
		Prompt:   "Prompt.",
	}
	if err := rm.store.Save(role); err != nil {
		t.Fatal(err)
	}

	// First load from store
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

func TestRoleManager_HasRoleFromStore(t *testing.T) {
	rm := newTestRoleManager(t)

	// Save role to store but don't cache it in the manager
	role := &Role{
		Metadata: RoleMetadata{Name: "storeonly"},
		Prompt:   "Prompt.",
	}
	if err := rm.store.Save(role); err != nil {
		t.Fatal(err)
	}

	// Should find in store even though not cached
	if !rm.HasRole("storeonly") {
		t.Error("HasRole should return true for role in store")
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

func TestEnsureDefaultRoles_CreatesInStore(t *testing.T) {
	rm := newTestRoleManager(t)

	created, err := rm.EnsureDefaultRoles()
	if err != nil {
		t.Fatalf("EnsureDefaultRoles: %v", err)
	}

	if len(created) != len(DefaultRoles) {
		t.Errorf("created %d roles, want %d", len(created), len(DefaultRoles))
	}

	for name := range DefaultRoles {
		if !rm.store.Has(name) {
			t.Errorf("expected role %q to exist in store", name)
		}
	}
}

func TestEnsureDefaultRoles_Idempotent(t *testing.T) {
	rm := newTestRoleManager(t)

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
	rm := newTestRoleManager(t)

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

func TestRoleManager_DeleteRole(t *testing.T) {
	rm := newTestRoleManager(t)

	role := &Role{
		Metadata: RoleMetadata{Name: "deleteme"},
		Prompt:   "Delete me.",
	}
	if err := rm.WriteRole(role); err != nil {
		t.Fatal(err)
	}

	if err := rm.DeleteRole("deleteme"); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	if rm.HasRole("deleteme") {
		t.Error("HasRole should return false after DeleteRole")
	}
}

func TestRoleManager_DeleteRole_NotFound(t *testing.T) {
	rm := newTestRoleManager(t)

	err := rm.DeleteRole("nonexistent")
	if err == nil {
		t.Error("DeleteRole should fail for nonexistent role")
	}
}

func TestRoleManager_MigrateFromFiles(t *testing.T) {
	rm, stateDir := newTestRoleManagerWithDir(t)

	// Create legacy roles dir with .md files
	rolesDir := filepath.Join(stateDir, "roles")
	if err := os.MkdirAll(rolesDir, 0750); err != nil {
		t.Fatal(err)
	}

	content := "---\nname: legacy\ndescription: From filesystem\n---\nLegacy prompt.\n"
	if err := os.WriteFile(filepath.Join(rolesDir, "legacy.md"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	migrated, err := rm.store.MigrateFromFiles(rolesDir)
	if err != nil {
		t.Fatalf("MigrateFromFiles: %v", err)
	}
	if migrated != 1 {
		t.Errorf("migrated = %d, want 1", migrated)
	}

	loaded, err := rm.LoadRole("legacy")
	if err != nil {
		t.Fatalf("LoadRole(legacy): %v", err)
	}
	if loaded.Metadata.Description != "From filesystem" {
		t.Errorf("Description = %q, want 'From filesystem'", loaded.Metadata.Description)
	}
}
