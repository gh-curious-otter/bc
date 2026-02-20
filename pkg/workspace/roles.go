// Package workspace provides workspace/project management.
package workspace

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RoleMetadata contains the parsed frontmatter from a role file.
type RoleMetadata struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description,omitempty"`
	Capabilities []string `yaml:"capabilities,omitempty"`
	ParentRoles  []string `yaml:"parent_roles,omitempty"`
	IsSingleton  bool     `yaml:"is_singleton,omitempty"`
}

// Role represents a parsed role file with metadata and prompt content.
type Role struct {
	FilePath string       // Path to the role file
	Prompt   string       // Markdown body after frontmatter
	Metadata RoleMetadata // Parsed YAML frontmatter
}

// Description returns a brief description for the role.
// Uses Metadata.Description if set, otherwise extracts from the first heading in Prompt.
func (r *Role) Description() string {
	if r.Metadata.Description != "" {
		return r.Metadata.Description
	}

	// Extract from first heading in prompt
	lines := strings.Split(r.Prompt, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	return ""
}

// RoleManager handles role file operations for a workspace.
type RoleManager struct {
	roles    map[string]*Role
	rolesDir string
}

// DefaultRootRole returns the default content for root.md.
const DefaultRootRole = `---
name: root
is_singleton: true
capabilities:
  - create_agents
  - assign_work
  - create_epics
  - review_work
---

# Root Agent

You are the root agent for this bc workspace.

## Responsibilities
- Oversee all workspace operations
- Coordinate top-level agents
- Handle merge queue for the main branch
- Ensure workspace integrity

## Guidelines
1. You are the singleton root - only one instance exists
2. Delegate work to child agents (managers, engineers)
3. Review and merge completed work
4. Monitor workspace health
`

// NewRoleManager creates a new role manager for the given workspace state directory.
func NewRoleManager(stateDir string) *RoleManager {
	return &RoleManager{
		rolesDir: filepath.Join(stateDir, "roles"),
		roles:    make(map[string]*Role),
	}
}

// RolesDir returns the roles directory path.
func (rm *RoleManager) RolesDir() string {
	return rm.rolesDir
}

// EnsureRolesDir creates the roles directory if it doesn't exist.
func (rm *RoleManager) EnsureRolesDir() error {
	return os.MkdirAll(rm.rolesDir, 0750)
}

// EnsureDefaultRoot creates the default root.md if it doesn't exist.
// Returns true if the file was created, false if it already existed.
func (rm *RoleManager) EnsureDefaultRoot() (bool, error) {
	rootPath := filepath.Join(rm.rolesDir, "root.md")

	// Check if file exists
	if _, err := os.Stat(rootPath); err == nil {
		return false, nil // Already exists
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check root.md: %w", err)
	}

	// Create roles directory if needed
	if err := rm.EnsureRolesDir(); err != nil {
		return false, err
	}

	// Write default root.md
	if err := os.WriteFile(rootPath, []byte(DefaultRootRole), 0600); err != nil {
		return false, fmt.Errorf("failed to create root.md: %w", err)
	}

	return true, nil
}

// LoadRole loads and parses a single role file.
func (rm *RoleManager) LoadRole(name string) (*Role, error) {
	// Check cache first
	if role, ok := rm.roles[name]; ok {
		return role, nil
	}

	filePath := filepath.Join(rm.rolesDir, name+".md")
	return rm.loadRoleFromPath(filePath)
}

// loadRoleFromPath loads a role from a specific file path.
func (rm *RoleManager) loadRoleFromPath(filePath string) (*Role, error) {
	data, err := os.ReadFile(filePath) //nolint:gosec // path constructed from known roles dir
	if err != nil {
		return nil, fmt.Errorf("failed to read role file: %w", err)
	}

	role, err := ParseRoleFile(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse role file %s: %w", filePath, err)
	}

	role.FilePath = filePath

	// If role name is empty (no frontmatter or missing name field),
	// derive it from the filename as a fallback
	if role.Metadata.Name == "" {
		role.Metadata.Name = strings.TrimSuffix(filepath.Base(filePath), ".md")
	}

	// Cache the role
	rm.roles[role.Metadata.Name] = role

	return role, nil
}

// LoadAllRoles loads all role files from the roles directory.
func (rm *RoleManager) LoadAllRoles() (map[string]*Role, error) {
	// Ensure default root exists
	if _, err := rm.EnsureDefaultRoot(); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(rm.rolesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return rm.roles, nil // Empty roles directory
		}
		return nil, fmt.Errorf("failed to read roles directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		filePath := filepath.Join(rm.rolesDir, name)
		if _, err := rm.loadRoleFromPath(filePath); err != nil {
			return nil, err
		}
	}

	return rm.roles, nil
}

// GetRole returns a cached role by name.
func (rm *RoleManager) GetRole(name string) (*Role, bool) {
	role, ok := rm.roles[name]
	return role, ok
}

// HasRole checks if a role exists (either cached or on disk).
func (rm *RoleManager) HasRole(name string) bool {
	// Check cache
	if _, ok := rm.roles[name]; ok {
		return true
	}

	// Check disk
	filePath := filepath.Join(rm.rolesDir, name+".md")
	_, err := os.Stat(filePath)
	return err == nil
}

// ParseRoleFile parses a role file with YAML frontmatter and markdown body.
// The frontmatter is delimited by --- on its own lines.
func ParseRoleFile(data []byte) (*Role, error) {
	content := string(data)

	// Check for frontmatter delimiter
	if !strings.HasPrefix(content, "---\n") && !strings.HasPrefix(content, "---\r\n") {
		// No frontmatter - treat entire content as prompt
		return &Role{
			Metadata: RoleMetadata{},
			Prompt:   strings.TrimSpace(content),
		}, nil
	}

	// Find the closing delimiter
	// Skip the opening "---\n"
	rest := content[4:]
	endIdx := strings.Index(rest, "\n---")
	if endIdx == -1 {
		// No closing delimiter - treat as no frontmatter
		return &Role{
			Metadata: RoleMetadata{},
			Prompt:   strings.TrimSpace(content),
		}, nil
	}

	// Extract frontmatter and body
	frontmatter := rest[:endIdx]
	body := rest[endIdx+4:] // Skip "\n---"

	// Skip any trailing newline after closing delimiter
	if strings.HasPrefix(body, "\n") {
		body = body[1:]
	} else if strings.HasPrefix(body, "\r\n") {
		body = body[2:]
	}

	// Parse YAML frontmatter
	var metadata RoleMetadata
	decoder := yaml.NewDecoder(bytes.NewReader([]byte(frontmatter)))
	if err := decoder.Decode(&metadata); err != nil {
		return nil, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	return &Role{
		Metadata: metadata,
		Prompt:   strings.TrimSpace(body),
	}, nil
}

// WriteRole writes a role to the roles directory.
func (rm *RoleManager) WriteRole(role *Role) error {
	if err := rm.EnsureRolesDir(); err != nil {
		return err
	}

	name := role.Metadata.Name
	if name == "" {
		return fmt.Errorf("role name is required")
	}

	filePath := filepath.Join(rm.rolesDir, name+".md")

	// Generate file content
	content, err := FormatRoleFile(role)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write role file: %w", err)
	}

	// Update cache
	role.FilePath = filePath
	rm.roles[name] = role

	return nil
}

// FormatRoleFile formats a role as a markdown file with YAML frontmatter.
func FormatRoleFile(role *Role) (string, error) {
	var buf bytes.Buffer

	// Write frontmatter
	buf.WriteString("---\n")

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(role.Metadata); err != nil {
		return "", fmt.Errorf("failed to encode metadata: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("failed to close encoder: %w", err)
	}

	buf.WriteString("---\n\n")

	// Write prompt body
	buf.WriteString(role.Prompt)
	if !strings.HasSuffix(role.Prompt, "\n") {
		buf.WriteString("\n")
	}

	return buf.String(), nil
}
