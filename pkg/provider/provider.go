// Package provider implements AI agent provider integrations.
// Issue #1451: OpenCode support
// Issue #1452: Cursor Agent support
// Epic #1429: Multi-Agent Integration
package provider

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Provider represents an AI agent provider that can run in a bc workspace.
type Provider interface {
	// Name returns the provider's unique identifier (e.g., "opencode", "cursor")
	Name() string

	// Description returns a human-readable description
	Description() string

	// Command returns the shell command to start this provider
	Command() string

	// IsInstalled checks if the provider binary is available on the system
	IsInstalled(ctx context.Context) bool

	// Version returns the installed version, or empty string if not installed
	Version(ctx context.Context) string

	// DetectState analyzes output to determine agent state (working, idle, done, etc.)
	DetectState(output string) State
}

// State represents the detected state of a provider's agent.
type State string

const (
	StateUnknown State = "unknown"
	StateIdle    State = "idle"
	StateWorking State = "working"
	StateDone    State = "done"
	StateError   State = "error"
	StateStuck   State = "stuck"
)

// Registry holds all registered providers.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.providers[p.Name()] = p
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// List returns all registered providers.
func (r *Registry) List() []Provider {
	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ListInstalled returns all installed providers.
func (r *Registry) ListInstalled(ctx context.Context) []Provider {
	var installed []Provider
	for _, p := range r.providers {
		if p.IsInstalled(ctx) {
			installed = append(installed, p)
		}
	}
	return installed
}

// DefaultRegistry is the global provider registry with all built-in providers.
var DefaultRegistry = NewRegistry()

func init() {
	// Register built-in providers
	DefaultRegistry.Register(NewOpenCodeProvider())
	DefaultRegistry.Register(NewClaudeProvider())
	DefaultRegistry.Register(NewCodexProvider())
}

// checkBinaryExists checks if a binary exists in PATH.
func checkBinaryExists(_ context.Context, name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// getBinaryVersion runs a command and returns the first line of output.
func getBinaryVersion(ctx context.Context, name string, args ...string) string {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // args are trusted provider names
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return ""
}

// GetProvider returns a provider by name from the default registry.
func GetProvider(name string) (Provider, error) {
	p, ok := DefaultRegistry.Get(name)
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return p, nil
}

// ListProviders returns all registered providers.
func ListProviders() []Provider {
	return DefaultRegistry.List()
}

// ListInstalledProviders returns all installed providers.
func ListInstalledProviders(ctx context.Context) []Provider {
	return DefaultRegistry.ListInstalled(ctx)
}
