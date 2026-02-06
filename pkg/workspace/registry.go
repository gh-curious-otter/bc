package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// RegistryEntry represents a registered workspace.
type RegistryEntry struct {
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// Registry manages the global list of workspaces at ~/.bc/workspaces.json.
type Registry struct {
	Workspaces []RegistryEntry `json:"workspaces"`
	path       string
}

// GlobalDir returns the path to ~/.bc/.
func GlobalDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bc")
}

// RegistryPath returns the path to ~/.bc/workspaces.json.
func RegistryPath() string {
	return filepath.Join(GlobalDir(), "workspaces.json")
}

// LoadRegistry loads the global workspace registry.
// Returns an empty registry if the file doesn't exist.
func LoadRegistry() (*Registry, error) {
	r := &Registry{path: RegistryPath()}

	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, r); err != nil {
		return nil, err
	}

	return r, nil
}

// Save persists the registry to disk.
func (r *Registry) Save() error {
	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.path, data, 0600)
}

// Register adds or updates a workspace in the registry.
func (r *Registry) Register(path, name string) {
	now := time.Now()

	for i, w := range r.Workspaces {
		if w.Path == path {
			r.Workspaces[i].Name = name
			r.Workspaces[i].LastAccessed = now
			return
		}
	}

	r.Workspaces = append(r.Workspaces, RegistryEntry{
		Path:         path,
		Name:         name,
		CreatedAt:    now,
		LastAccessed: now,
	})
}

// Unregister removes a workspace from the registry.
func (r *Registry) Unregister(path string) {
	for i, w := range r.Workspaces {
		if w.Path == path {
			r.Workspaces = append(r.Workspaces[:i], r.Workspaces[i+1:]...)
			return
		}
	}
}

// Touch updates the last-accessed time for a workspace.
func (r *Registry) Touch(path string) {
	for i, w := range r.Workspaces {
		if w.Path == path {
			r.Workspaces[i].LastAccessed = time.Now()
			return
		}
	}
}

// Prune removes entries where the workspace no longer exists on disk.
func (r *Registry) Prune() int {
	pruned := 0
	valid := make([]RegistryEntry, 0, len(r.Workspaces))
	for _, w := range r.Workspaces {
		if IsWorkspace(w.Path) {
			valid = append(valid, w)
		} else {
			pruned++
		}
	}
	r.Workspaces = valid
	return pruned
}

// List returns all registered workspaces.
func (r *Registry) List() []RegistryEntry {
	return r.Workspaces
}

// Find returns the entry for a given path, or nil if not found.
func (r *Registry) Find(path string) *RegistryEntry {
	for i, w := range r.Workspaces {
		if w.Path == path {
			return &r.Workspaces[i]
		}
	}
	return nil
}
