// Package process provides process management for bc.
package process

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// Process represents a managed process.
type Process struct {
	StartedAt time.Time `json:"started_at"`
	Name      string    `json:"name"`
	Command   string    `json:"command"`
	Owner     string    `json:"owner,omitempty"` // Agent that started the process
	WorkDir   string    `json:"work_dir,omitempty"`
	PID       int       `json:"pid"`
	Port      int       `json:"port,omitempty"`
	Running   bool      `json:"running"`
}

// Registry manages running processes.
type Registry struct {
	processes    map[string]*Process
	processesDir string
	mu           sync.RWMutex
}

// NewRegistry creates a new process registry.
func NewRegistry(rootDir string) *Registry {
	return &Registry{
		processes:    make(map[string]*Process),
		processesDir: filepath.Join(rootDir, ".bc", "processes"),
	}
}

// Init creates the processes directory and loads existing state.
func (r *Registry) Init() error {
	if err := os.MkdirAll(r.processesDir, 0750); err != nil {
		return fmt.Errorf("failed to create processes directory: %w", err)
	}
	return r.load()
}

// Register adds a process to the registry.
func (r *Registry) Register(p *Process) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.processes[p.Name]; exists {
		return fmt.Errorf("process %q already registered", p.Name)
	}

	if p.StartedAt.IsZero() {
		p.StartedAt = time.Now().UTC()
	}
	p.Running = true

	r.processes[p.Name] = p
	return r.persist()
}

// Unregister removes a process from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.processes[name]; !exists {
		return fmt.Errorf("process %q not found", name)
	}

	delete(r.processes, name)
	return r.persist()
}

// Get retrieves a process by name.
func (r *Registry) Get(name string) *Process {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.processes[name]
}

// List returns all registered processes.
func (r *Registry) List() []*Process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Process, 0, len(r.processes))
	for _, p := range r.processes {
		result = append(result, p)
	}

	// Sort by name for consistent ordering
	slices.SortFunc(result, func(a, b *Process) int {
		return strings.Compare(a.Name, b.Name)
	})

	return result
}

// ListByOwner returns all processes owned by a specific agent.
func (r *Registry) ListByOwner(owner string) []*Process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Process
	for _, p := range r.processes {
		if p.Owner == owner {
			result = append(result, p)
		}
	}
	return result
}

// MarkStopped marks a process as stopped without removing it.
func (r *Registry) MarkStopped(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.processes[name]
	if !exists {
		return fmt.Errorf("process %q not found", name)
	}

	p.Running = false
	p.PID = 0
	return r.persist()
}

// UpdatePID updates the PID of a running process.
func (r *Registry) UpdatePID(name string, pid int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, exists := r.processes[name]
	if !exists {
		return fmt.Errorf("process %q not found", name)
	}

	p.PID = pid
	return r.persist()
}

// IsPortInUse checks if a port is used by any registered process.
func (r *Registry) IsPortInUse(port int) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.processes {
		if p.Running && p.Port == port {
			return true
		}
	}
	return false
}

// GetByPort returns the process using a specific port.
func (r *Registry) GetByPort(port int) *Process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.processes {
		if p.Running && p.Port == port {
			return p
		}
	}
	return nil
}

// Clear removes all processes from the registry.
func (r *Registry) Clear() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.processes = make(map[string]*Process)
	return r.persist()
}

// persist saves the registry state to disk.
func (r *Registry) persist() error {
	data, err := json.MarshalIndent(r.processes, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal processes: %w", err)
	}

	path := filepath.Join(r.processesDir, "registry.json")
	if err := os.WriteFile(path, data, 0600); err != nil { //nolint:gosec // path constructed from trusted dir
		return fmt.Errorf("failed to write registry: %w", err)
	}

	return nil
}

// load reads the registry state from disk.
func (r *Registry) load() error {
	path := filepath.Join(r.processesDir, "registry.json")
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from trusted dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing registry
		}
		return fmt.Errorf("failed to read registry: %w", err)
	}

	if err := json.Unmarshal(data, &r.processes); err != nil {
		return fmt.Errorf("failed to parse registry: %w", err)
	}

	return nil
}

// ProcessesDir returns the path to the processes directory.
func (r *Registry) ProcessesDir() string {
	return r.processesDir
}
