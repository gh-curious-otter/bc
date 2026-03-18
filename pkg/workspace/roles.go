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
	Permissions  []string `yaml:"permissions,omitempty"`  // RBAC permissions (#1191)
	ParentRoles  []string `yaml:"parent_roles,omitempty"`
	MCPServers   []string `yaml:"mcp_servers,omitempty"` // MCP server associations (#1924)
	IsSingleton  bool     `yaml:"is_singleton,omitempty"`
	Level        int      `yaml:"level,omitempty"`       // Role hierarchy level (-1=root, 0=manager, 1=engineer)
	Plugins []string `yaml:"plugins,omitempty"` // Claude Code plugins to install on agent start (#1959)
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
level: -1
capabilities:
  - create_agents
  - assign_work
  - create_epics
  - review_work
permissions:
  - can_create_agents
  - can_stop_agents
  - can_delete_agents
  - can_restart_agents
  - can_send_commands
  - can_view_logs
  - can_modify_config
  - can_modify_roles
  - can_create_channels
  - can_delete_channels
  - can_send_messages
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

// DefaultRoles contains the built-in role definitions for the bc agent team.
// These are written to .bc/roles/ if the files don't already exist.
var DefaultRoles = map[string]string{
	"go-reviewer": `---
name: go-reviewer
description: Go code quality reviewer
level: 0
capabilities:
  - review_work
mcp_servers:
  - github
---

# Go Reviewer

You are a Go code quality reviewer for this bc workspace.

## Responsibilities
- Review Go CLI pull requests for correctness, security, and test coverage
- Enforce Go idioms, linting standards, and project conventions
- Check for security issues (OWASP, injection, credentials exposure)
- Ensure error handling is explicit and complete
- Validate that tests cover edge cases and use table-driven patterns

## Guidelines
1. Use the github MCP to fetch PR diffs and leave inline review comments
2. Block merges on security issues or broken tests; suggest rather than block on style
3. Reference .golangci.yml rules when flagging lint violations
4. Be concise — one clear comment beats three vague ones
`,
	"web-reviewer": `---
name: web-reviewer
description: Web/TypeScript UI reviewer
level: 0
capabilities:
  - review_work
mcp_servers:
  - github
---

# Web Reviewer

You are a React/TypeScript code quality reviewer for this bc workspace.

## Responsibilities
- Review TUI (Ink/React) pull requests for correctness and component patterns
- Enforce TypeScript type safety and accessibility best practices
- Check for performance issues in React hooks and re-render paths
- Validate test coverage for exported helpers and type interfaces

## Guidelines
1. Use the github MCP to fetch PR diffs and leave inline review comments
2. Note: hooks cannot be tested without a DOM in Ink — test exported helpers instead
3. Block on broken builds or type errors; suggest on style
4. Keep feedback actionable — link to specific lines
`,
	"feature-dev": `---
name: feature-dev
description: Feature developer — implements tasks in isolated worktrees
level: 1
capabilities:
  - implement_tasks
  - run_tests
  - fix_bugs
mcp_servers:
  - github
  - filesystem
---

# Feature Developer

You are a feature developer for the bc project. You work in an isolated git worktree
and commit your changes to a feature branch for review.

## Responsibilities
- Implement assigned issues and feature tasks
- Write tests for all new code (table-driven where applicable)
- Fix bugs and regressions in your area of ownership
- Create pull requests targeting main for review

## Workflow
1. Read the assigned issue thoroughly before writing any code
2. Create a feature branch: feat/<issue>-<slug> from main
3. Implement the feature with tests; run make check before pushing
4. Open a PR and request review from the appropriate reviewer agent

## Guidelines
- Follow CLAUDE.md conventions: gofmt -s, goimports, short receivers
- Never ignore errors — use explicit handling or //nolint:errcheck with justification
- Prefer editing existing files over creating new ones
- Commit messages: conventional commits format (feat:, fix:, docs:, etc.)
`,
	"designer": `---
name: designer
description: Design system and Web UI specialist
level: 1
capabilities:
  - implement_tasks
mcp_servers:
  - github
---

# Designer

You are the design system and Web UI specialist for the bc project.

## Responsibilities
- Maintain the design token system (colors, spacing, typography)
- Build and refine React/Ink TUI components
- Create component specs and accessibility guidelines
- Implement CSS/Tailwind changes for the web dashboard

## Guidelines
1. Consistency over novelty — extend the existing design system
2. Every new component needs a usage example and props documentation
3. Accessibility is non-negotiable: keyboard navigation, color contrast, screen readers
4. Test components in both dark and light themes
`,
	"product-manager": `---
name: product-manager
description: Product coordination and epic management
level: 0
capabilities:
  - create_epics
  - assign_work
  - review_work
---

# Product Manager

You are the product manager for the bc project, responsible for coordinating
the agent team and ensuring the product vision is reflected in delivered work.

## Responsibilities
- Break down high-level goals into actionable epics and issues
- Assign work to the appropriate agent based on role and current load
- Review completed work against acceptance criteria before merge
- Maintain the product roadmap and prioritization

## Guidelines
1. Use GitHub issues to track all work — no verbal assignments
2. Each epic must have clear acceptance criteria before agents start work
3. Communicate blockers to the root agent immediately
4. Keep the team focused: one epic per engineer at a time
`,
	"docs": `---
name: docs
description: Documentation writer
level: 1
capabilities:
  - implement_tasks
mcp_servers:
  - github
---

# Documentation Writer

You are the documentation specialist for the bc project.

## Responsibilities
- Write and maintain README, CONTRIBUTING, and API reference docs
- Keep CLAUDE.md up-to-date with architecture and convention changes
- Build and maintain the mkdocs documentation site
- Document CLI commands, flags, and usage examples

## Guidelines
1. Docs live alongside the code — update docs in the same PR as the feature
2. Use concrete examples — show, don't just tell
3. Keep the CLI help text in sync with the markdown docs
4. Plain language over jargon; assume the reader is new to bc
`,
}

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

// EnsureDefaultRoles creates built-in role files that don't already exist.
// Returns the names of any roles that were created.
func (rm *RoleManager) EnsureDefaultRoles() ([]string, error) {
	if err := rm.EnsureRolesDir(); err != nil {
		return nil, err
	}

	var created []string
	for name, content := range DefaultRoles {
		rolePath := filepath.Join(rm.rolesDir, name+".md")
		if _, err := os.Stat(rolePath); err == nil {
			continue // already exists
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check %s.md: %w", name, err)
		}
		if err := os.WriteFile(rolePath, []byte(content), 0600); err != nil {
			return nil, fmt.Errorf("failed to create %s.md: %w", name, err)
		}
		created = append(created, name)
	}
	return created, nil
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

// HasPermission checks if a role has a specific permission.
// Returns true if the permission is explicitly listed or if the role
// inherits the permission from its default level.
func (r *Role) HasPermission(permission string) bool {
	// Check explicit permissions first
	for _, p := range r.Metadata.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetEffectivePermissions returns all permissions for a role,
// including inherited defaults based on role level.
func (r *Role) GetEffectivePermissions() []string {
	// If explicit permissions are set, use those
	if len(r.Metadata.Permissions) > 0 {
		return r.Metadata.Permissions
	}

	// Otherwise, return defaults based on role level
	level := r.Metadata.Level
	switch {
	case level <= -1:
		// Root level - all permissions
		return []string{
			"can_create_agents", "can_stop_agents", "can_delete_agents", "can_restart_agents",
			"can_send_commands", "can_view_logs",
			"can_modify_config", "can_modify_roles",
			"can_create_channels", "can_delete_channels", "can_send_messages",
		}
	case level == 0:
		// Manager level
		return []string{
			"can_create_agents", "can_stop_agents", "can_restart_agents",
			"can_send_commands", "can_view_logs",
			"can_create_channels", "can_send_messages",
		}
	default:
		// Engineer/worker level
		return []string{
			"can_view_logs", "can_send_commands", "can_send_messages",
		}
	}
}

// SetPermissions updates the permissions for a role.
func (rm *RoleManager) SetPermissions(roleName string, permissions []string) error {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	role.Metadata.Permissions = permissions

	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}

// AddPermission adds a permission to a role if not already present.
func (rm *RoleManager) AddPermission(roleName, permission string) error {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	// Check if already has permission
	for _, p := range role.Metadata.Permissions {
		if p == permission {
			return nil // Already has permission
		}
	}

	role.Metadata.Permissions = append(role.Metadata.Permissions, permission)

	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}

// RemovePermission removes a permission from a role.
func (rm *RoleManager) RemovePermission(roleName, permission string) error {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	// Filter out the permission
	filtered := make([]string, 0, len(role.Metadata.Permissions))
	for _, p := range role.Metadata.Permissions {
		if p != permission {
			filtered = append(filtered, p)
		}
	}
	role.Metadata.Permissions = filtered

	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}

// GetMCPServers returns the MCP server associations for a role.
func (rm *RoleManager) GetMCPServers(roleName string) ([]string, error) {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to load role: %w", err)
	}
	return role.Metadata.MCPServers, nil
}

// SetMCPServers replaces the MCP server list for a role.
func (rm *RoleManager) SetMCPServers(roleName string, servers []string) error {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	role.Metadata.MCPServers = servers

	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}

// AddMCPServer adds an MCP server association to a role if not already present.
func (rm *RoleManager) AddMCPServer(roleName, server string) error {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	for _, s := range role.Metadata.MCPServers {
		if s == server {
			return nil // Already associated
		}
	}

	role.Metadata.MCPServers = append(role.Metadata.MCPServers, server)

	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}

// RemoveMCPServer removes an MCP server association from a role.
func (rm *RoleManager) RemoveMCPServer(roleName, server string) error {
	role, err := rm.LoadRole(roleName)
	if err != nil {
		return fmt.Errorf("failed to load role: %w", err)
	}

	filtered := make([]string, 0, len(role.Metadata.MCPServers))
	for _, s := range role.Metadata.MCPServers {
		if s != server {
			filtered = append(filtered, s)
		}
	}
	role.Metadata.MCPServers = filtered

	if err := rm.WriteRole(role); err != nil {
		return fmt.Errorf("failed to save role: %w", err)
	}

	return nil
}
