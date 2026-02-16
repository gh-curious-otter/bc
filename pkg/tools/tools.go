// Package tools provides a unified interface for external tool integrations.
// Tools like GitHub, GitLab, Claude, etc. can be registered and executed
// through a consistent API.
package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// Tool represents a configured external tool integration.
type Tool struct {
	Name      string `toml:"name"`
	Command   string `toml:"command"`
	Scope     string `toml:"scope,omitempty"`      // e.g., "issues,pulls,api" for GitHub
	TokenEnv  string `toml:"token_env,omitempty"`  // environment variable for auth token
	URL       string `toml:"url,omitempty"`        // API URL for custom instances
	RateLimit int    `toml:"rate_limit,omitempty"` // requests per hour
	Enabled   bool   `toml:"enabled"`
}

// Registry holds all registered tools.
type Registry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry.
func (r *Registry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool.Command == "" {
		return fmt.Errorf("tool command is required")
	}

	r.tools[tool.Name] = tool
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tools.
func (r *Registry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// ListEnabled returns only enabled tools.
func (r *Registry) ListEnabled() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*Tool, 0)
	for _, t := range r.tools {
		if t.Enabled {
			tools = append(tools, t)
		}
	}
	return tools
}

// Enable enables a tool by name.
func (r *Registry) Enable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tool, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("tool not found: %s", name)
	}
	tool.Enabled = true
	return nil
}

// Disable disables a tool by name.
func (r *Registry) Disable(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	tool, ok := r.tools[name]
	if !ok {
		return fmt.Errorf("tool not found: %s", name)
	}
	tool.Enabled = false
	return nil
}

// ExecResult contains the result of a tool execution.
type ExecResult struct {
	Error    error
	Output   string
	ExitCode int
}

// Exec executes a tool command with the given arguments.
func (r *Registry) Exec(ctx context.Context, name string, args ...string) (*ExecResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	if !tool.Enabled {
		return nil, fmt.Errorf("tool is disabled: %s", name)
	}

	// Parse the tool command
	parts := strings.Fields(tool.Command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid tool command: %s", tool.Command)
	}

	// Build the full command
	cmdArgs := append(parts[1:], args...)
	// G204: Tool execution requires dynamic command construction - commands come from trusted config
	cmd := exec.CommandContext(ctx, parts[0], cmdArgs...) //nolint:gosec

	// Execute the command
	output, err := cmd.CombinedOutput()

	result := &ExecResult{
		Output:   string(output),
		ExitCode: 0,
	}

	if err != nil {
		result.Error = err
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
	}

	return result, nil
}

// IsInstalled checks if the tool's command is available in PATH.
func (t *Tool) IsInstalled() bool {
	parts := strings.Fields(t.Command)
	if len(parts) == 0 {
		return false
	}
	_, err := exec.LookPath(parts[0])
	return err == nil
}

// Status returns the tool's status.
func (t *Tool) Status() string {
	if !t.Enabled {
		return "disabled"
	}
	if !t.IsInstalled() {
		return "not installed"
	}
	return "ready"
}

// DefaultRegistry is the global tool registry.
var DefaultRegistry = NewRegistry()

// Register adds a tool to the default registry.
func Register(tool *Tool) error {
	return DefaultRegistry.Register(tool)
}

// Get retrieves a tool from the default registry.
func Get(name string) (*Tool, bool) {
	return DefaultRegistry.Get(name)
}

// List returns all tools from the default registry.
func List() []*Tool {
	return DefaultRegistry.List()
}

// ListEnabled returns enabled tools from the default registry.
func ListEnabled() []*Tool {
	return DefaultRegistry.ListEnabled()
}

// Exec executes a tool command using the default registry.
func Exec(ctx context.Context, name string, args ...string) (*ExecResult, error) {
	return DefaultRegistry.Exec(ctx, name, args...)
}

// Enable enables a tool in the default registry.
func Enable(name string) error {
	return DefaultRegistry.Enable(name)
}

// Disable disables a tool in the default registry.
func Disable(name string) error {
	return DefaultRegistry.Disable(name)
}
