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
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	ParentRoles []string `yaml:"parent_roles,omitempty"` // Roles to inherit from (BFS merge)
	MCPServers  []string `yaml:"mcp_servers,omitempty"`  // MCP servers available to this role
	Secrets     []string `yaml:"secrets,omitempty"`      // Secret names needed by MCP env vars
	Plugins     []string `yaml:"plugins,omitempty"`      // Claude Code plugins to install on agent start

	// Lifecycle prompts — sent to the agent at each lifecycle stage.
	// These are in addition to the main prompt body (which becomes CLAUDE.md).
	PromptCreate string `yaml:"prompt_create,omitempty"` // Sent when agent is created
	PromptStart  string `yaml:"prompt_start,omitempty"`  // Sent on each start/resume
	PromptStop   string `yaml:"prompt_stop,omitempty"`   // Sent before stopping
	PromptDelete string `yaml:"prompt_delete,omitempty"` // Sent before deletion

	// Claude Code workspace files generated from role
	Settings map[string]any    `yaml:"settings,omitempty"` // Written to .claude/settings.json
	Commands map[string]string `yaml:"commands,omitempty"` // Written to .claude/commands/<name>.md
	Skills   map[string]string `yaml:"skills,omitempty"`   // Written to .claude/skills/<name>.md
	Agents   map[string]string `yaml:"agents,omitempty"`   // Written to .claude/agents/<name>.md
	Rules    map[string]string `yaml:"rules,omitempty"`    // Written to .claude/rules/<name>.md
	Review   string            `yaml:"review,omitempty"`   // Written to REVIEW.md
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

// DefaultBaseRole is the foundational role all other roles inherit from.
// It provides the bc MCP server so every agent can communicate with the workspace.
const DefaultBaseRole = `---
name: base
description: Base role — provides bc MCP server, workspace communication, and shared commands to all agents
mcp_servers:
  - bc
prompt_create: |
  You have been created as a new agent in a bc workspace.
  Use the report_status MCP tool to set your initial task.
  Check #all and #engineering channels for context.
prompt_start: |
  You are online. Use report_status to update your current task.
  Check channels for any messages sent while you were offline.
prompt_stop: |
  You are being stopped. Save any important state.
  Post a status update to #engineering if you have work in progress.
commands:
  status: |
    Check all channels for recent messages addressed to you.
    Report your current task using the report_status MCP tool.
    Query workspace costs using query_costs.
  channels: |
    Read recent messages from all channels: all, engineering, general, merge, ops.
    Summarize any messages that are relevant to your current work.
  announce: |
    Send an announcement to the #all channel using send_message.
    Include your agent name as the sender.
rules:
  workspace-communication: |
    All workspace operations MUST use bc MCP tools. Never use bc CLI commands directly.
    Available MCP tools: send_message, report_status, query_costs.
    Always include your agent name as sender when sending messages.
  channel-etiquette: |
    Use the right channel for the right purpose:
    - #all for broadcast announcements only
    - #engineering for technical coordination
    - #general for general discussion
    - #merge for PR review requests
    - #ops for system health and cost reports
    Do not spam #all with routine updates.
---

# bc Agent

You are an agent in a **bc** workspace — a CLI-first AI agent orchestration system.

## MCP Tools

All workspace operations use bc MCP tools (never CLI commands):

| Tool | Purpose | Parameters |
|------|---------|------------|
| **send_message** | Send messages to channels | {channel, message, sender} |
| **report_status** | Update your current task | {agent, task} |
| **query_costs** | Check workspace costs | {agent?} |

## Channels

- **#all** — Broadcast announcements
- **#engineering** — Engineering coordination
- **#general** — General discussion
- **#merge** — PR review pipeline
- **#ops** — System health and costs

## Guidelines

- Report your status when starting or finishing work
- Post to the appropriate channel, not #all, for routine updates
- Use #merge when a PR is ready for review
- Check channels for messages before starting new work
`

// DefaultRootRole returns the default content for root.md.
const DefaultRootRole = `---
name: root
description: Root orchestrator — singleton workspace owner
parent_roles:
  - base
mcp_servers:
  - github
secrets:
  - GITHUB_PERSONAL_ACCESS_TOKEN
prompt_start: |
  You are back online. Check #all channel for any messages you missed.
  Report your status using the report_status MCP tool.
---

# Root Agent

You are the root agent for this bc workspace.

## Additional MCP Tools
- **create_agent**: Create new agents {name, role, tool}

## Responsibilities
- Oversee all workspace operations
- Create and coordinate agents
- Handle merge queue for the main branch
- Monitor workspace health and costs
`

// DefaultRoles contains the built-in role definitions for the bc agent team.
// These are written to .bc/roles/ if the files don't already exist.
var DefaultRoles = map[string]string{
	"feature-dev": `---
name: feature-dev
description: Feature developer — implements tasks in isolated worktrees
parent_roles:
  - base
mcp_servers:
  - github
secrets:
  - GITHUB_PERSONAL_ACCESS_TOKEN
prompt_start: |
  Check #engineering channel for any new assignments or updates.
---

# Feature Developer

You implement features, fix bugs, and write tests in an isolated git worktree.

## Workflow
1. Read the assigned issue, create a feature branch (feat/<issue>-<slug>)
2. Implement with tests, run make check
3. Open a PR and post to #merge when ready for review
`,
	"go-reviewer": `---
name: go-reviewer
description: Go code quality reviewer
parent_roles:
  - feature-dev
---

# Go Reviewer

Review Go pull requests for correctness, security, and test coverage.
Use the github MCP to fetch PR diffs and leave inline review comments.
Block merges on security issues or broken tests; suggest rather than block on style.
`,
	"web-reviewer": `---
name: web-reviewer
description: Web/TypeScript UI reviewer
parent_roles:
  - feature-dev
---

# Web Reviewer

Review React/TypeScript pull requests for correctness and component patterns.
Use the github MCP to fetch PR diffs and leave inline review comments.
Hooks cannot be tested without a DOM in Ink — test exported helpers instead.
`,
	"designer": `---
name: designer
description: Design system and Web UI specialist
parent_roles:
  - feature-dev
---

# Designer

Maintain the design token system, build React/Ink TUI components, and implement
CSS/Tailwind changes for the web dashboard. Accessibility is non-negotiable.
`,
	"product-manager": `---
name: product-manager
description: Product coordination and epic management
parent_roles:
  - base
mcp_servers:
  - github
secrets:
  - GITHUB_PERSONAL_ACCESS_TOKEN
---

# Product Manager

Break down goals into epics and issues, assign work to agents, and review
completed work against acceptance criteria. Use GitHub issues for all tracking.
`,
	"docs": `---
name: docs
description: Documentation writer
parent_roles:
  - feature-dev
---

# Documentation Writer

Write and maintain README, CONTRIBUTING, API docs, and CLAUDE.md.
Update docs in the same PR as the feature. Use concrete examples.
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

// EnsureDefaultRoot creates the default root.md and base.md if they don't exist.
// Returns true if root.md was created, false if it already existed.
func (rm *RoleManager) EnsureDefaultRoot() (bool, error) {
	if err := rm.EnsureRolesDir(); err != nil {
		return false, err
	}

	// Always ensure base.md exists (root and all roles inherit from it)
	basePath := filepath.Join(rm.rolesDir, "base.md")
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		if writeErr := os.WriteFile(basePath, []byte(DefaultBaseRole), 0600); writeErr != nil {
			return false, fmt.Errorf("failed to create base.md: %w", writeErr)
		}
	}

	rootPath := filepath.Join(rm.rolesDir, "root.md")
	if _, err := os.Stat(rootPath); err == nil {
		return false, nil // Already exists
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("failed to check root.md: %w", err)
	}

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

// ResolvedRole contains the fully resolved role after BFS inheritance merge.
type ResolvedRole struct {
	Name        string   // Role name
	Prompt      string   // Main prompt body (becomes CLAUDE.md)
	MCPServers  []string // Merged MCP servers (child first, parents add missing)
	Secrets     []string // Merged secrets (child first, parents add missing)
	Plugins     []string // Merged plugins (child first, parents add missing)
	PromptCreate string  // Lifecycle prompt for create
	PromptStart  string  // Lifecycle prompt for start
	PromptStop   string  // Lifecycle prompt for stop
	PromptDelete string  // Lifecycle prompt for delete

	// Claude Code workspace files (merged from role hierarchy)
	Settings map[string]any    // Merged settings (child keys win)
	Commands map[string]string // Merged commands (child wins per name)
	Skills   map[string]string // Merged skills (child wins per name)
	Agents   map[string]string // Merged agents (child wins per name)
	Rules    map[string]string // Merged rules (child wins per name)
	Review   string            // Review content (child's if set, else first parent's)
}

// ResolveRole performs BFS inheritance merge starting from the given role.
// Child values take priority — parent values are only added if not already present.
// MCP servers and secrets are unioned; lifecycle prompts use child's if set, else first parent's.
func (rm *RoleManager) ResolveRole(name string) (*ResolvedRole, error) {
	role, err := rm.LoadRole(name)
	if err != nil {
		return nil, fmt.Errorf("failed to load role %q: %w", name, err)
	}

	resolved := &ResolvedRole{
		Name:         role.Metadata.Name,
		Prompt:       role.Prompt,
		MCPServers:   append([]string{}, role.Metadata.MCPServers...),
		Secrets:      append([]string{}, role.Metadata.Secrets...),
		Plugins:      append([]string{}, role.Metadata.Plugins...),
		PromptCreate: role.Metadata.PromptCreate,
		PromptStart:  role.Metadata.PromptStart,
		PromptStop:   role.Metadata.PromptStop,
		PromptDelete: role.Metadata.PromptDelete,
		Settings:     mergeAnyMaps(nil, role.Metadata.Settings),
		Commands:     mergeMaps(nil, role.Metadata.Commands),
		Skills:       mergeMaps(nil, role.Metadata.Skills),
		Agents:       mergeMaps(nil, role.Metadata.Agents),
		Rules:        mergeMaps(nil, role.Metadata.Rules),
		Review:       role.Metadata.Review,
	}

	// BFS through parent roles
	visited := map[string]bool{name: true}
	queue := append([]string{}, role.Metadata.ParentRoles...)

	for len(queue) > 0 {
		parentName := queue[0]
		queue = queue[1:]

		if visited[parentName] {
			continue
		}
		visited[parentName] = true

		parent, loadErr := rm.LoadRole(parentName)
		if loadErr != nil {
			continue // Skip missing parents gracefully
		}

		// Merge MCP servers — add only if not already present
		resolved.MCPServers = mergeUnique(resolved.MCPServers, parent.Metadata.MCPServers)
		// Merge secrets — add only if not already present
		resolved.Secrets = mergeUnique(resolved.Secrets, parent.Metadata.Secrets)
		// Merge plugins — add only if not already present
		resolved.Plugins = mergeUnique(resolved.Plugins, parent.Metadata.Plugins)

		// Lifecycle prompts — use parent's only if child doesn't have one
		if resolved.PromptCreate == "" {
			resolved.PromptCreate = parent.Metadata.PromptCreate
		}
		if resolved.PromptStart == "" {
			resolved.PromptStart = parent.Metadata.PromptStart
		}
		if resolved.PromptStop == "" {
			resolved.PromptStop = parent.Metadata.PromptStop
		}
		if resolved.PromptDelete == "" {
			resolved.PromptDelete = parent.Metadata.PromptDelete
		}

		// Merge map fields — child keys take priority
		resolved.Settings = mergeAnyMaps(resolved.Settings, parent.Metadata.Settings)
		resolved.Commands = mergeMaps(resolved.Commands, parent.Metadata.Commands)
		resolved.Skills = mergeMaps(resolved.Skills, parent.Metadata.Skills)
		resolved.Agents = mergeMaps(resolved.Agents, parent.Metadata.Agents)
		resolved.Rules = mergeMaps(resolved.Rules, parent.Metadata.Rules)

		// Review — use parent's only if child doesn't have one
		if resolved.Review == "" {
			resolved.Review = parent.Metadata.Review
		}

		// Enqueue grandparents
		queue = append(queue, parent.Metadata.ParentRoles...)
	}

	return resolved, nil
}

// mergeUnique appends items from src to dst only if they don't already exist in dst.
func mergeUnique(dst, src []string) []string {
	seen := make(map[string]bool, len(dst))
	for _, v := range dst {
		seen[v] = true
	}
	for _, v := range src {
		if !seen[v] {
			dst = append(dst, v)
			seen[v] = true
		}
	}
	return dst
}

// mergeMaps copies src entries to dst only if the key doesn't already exist in dst.
// If dst is nil, a new map is created. Returns nil if both are empty/nil.
func mergeMaps(dst, src map[string]string) map[string]string {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]string, len(src))
	}
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
	return dst
}

// mergeAnyMaps copies src entries to dst only if the key doesn't already exist in dst.
// If dst is nil, a new map is created. Returns nil if both are empty/nil.
func mergeAnyMaps(dst, src map[string]any) map[string]any {
	if len(src) == 0 {
		return dst
	}
	if dst == nil {
		dst = make(map[string]any, len(src))
	}
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			dst[k] = v
		}
	}
	return dst
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
