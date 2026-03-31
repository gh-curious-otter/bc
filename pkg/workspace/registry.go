package workspace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// RegistryEntry represents a registered workspace.
type RegistryEntry struct {
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Alias        string    `json:"alias,omitempty"` // Short alias for quick access (#1218)
}

// Registry manages the global list of workspaces at ~/.bc/workspaces.json.
// Issue #1218: Multi-workspace orchestration support.
type Registry struct {
	path       string
	Active     string          `json:"active,omitempty"` // Path or alias of active workspace
	Workspaces []RegistryEntry `json:"workspaces"`
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
	_ = r.RegisterWithAlias(path, name, "") //nolint:errcheck // legacy function, alias conflict not possible with empty alias
}

// RegisterWithAlias adds or updates a workspace with an optional alias.
// Issue #1218: Multi-workspace orchestration.
func (r *Registry) RegisterWithAlias(path, name, alias string) error {
	now := time.Now()

	// Check alias conflict if setting one
	if alias != "" {
		existing := r.FindByAlias(alias)
		if existing != nil && existing.Path != path {
			return &AliasConflictError{Alias: alias, ExistingPath: existing.Path}
		}
	}

	for i, w := range r.Workspaces {
		if w.Path == path {
			r.Workspaces[i].Name = name
			r.Workspaces[i].LastAccessed = now
			if alias != "" {
				r.Workspaces[i].Alias = alias
			}
			return nil
		}
	}

	r.Workspaces = append(r.Workspaces, RegistryEntry{
		Path:         path,
		Name:         name,
		Alias:        alias,
		CreatedAt:    now,
		LastAccessed: now,
	})
	return nil
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
// Checks for .bc/ dir in project root OR state dir in ~/.bc/workspaces/<id>/.
func (r *Registry) Prune() int {
	pruned := 0
	valid := make([]RegistryEntry, 0, len(r.Workspaces))
	for _, w := range r.Workspaces {
		// Check legacy .bc/ marker
		if _, err := os.Stat(filepath.Join(w.Path, ".bc")); err == nil {
			valid = append(valid, w)
			continue
		}
		// Check global state dir exists
		if stateDir, err := GlobalStateDir(w.Path); err == nil {
			if _, statErr := os.Stat(stateDir); statErr == nil {
				valid = append(valid, w)
				continue
			}
		}
		pruned++
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

// FindByAlias returns the entry for a given alias, or nil if not found.
// Issue #1218: Multi-workspace orchestration.
func (r *Registry) FindByAlias(alias string) *RegistryEntry {
	for i, w := range r.Workspaces {
		if w.Alias == alias {
			return &r.Workspaces[i]
		}
	}
	return nil
}

// FindByNameOrAlias returns the entry matching name, alias, or path.
// Tries alias first, then name, then path.
func (r *Registry) FindByNameOrAlias(identifier string) *RegistryEntry {
	// Check alias first (most specific)
	if entry := r.FindByAlias(identifier); entry != nil {
		return entry
	}
	// Check name
	for i, w := range r.Workspaces {
		if w.Name == identifier {
			return &r.Workspaces[i]
		}
	}
	// Check path last
	return r.Find(identifier)
}

// SetAlias sets or clears the alias for a workspace.
func (r *Registry) SetAlias(path, alias string) error {
	// Check if alias is already in use by another workspace
	if alias != "" {
		existing := r.FindByAlias(alias)
		if existing != nil && existing.Path != path {
			return &AliasConflictError{Alias: alias, ExistingPath: existing.Path}
		}
	}

	for i, w := range r.Workspaces {
		if w.Path == path {
			r.Workspaces[i].Alias = alias
			return nil
		}
	}
	return &WorkspaceNotFoundError{Identifier: path}
}

// GetActive returns the active workspace entry, or nil if none set.
func (r *Registry) GetActive() *RegistryEntry {
	if r.Active == "" {
		return nil
	}
	return r.FindByNameOrAlias(r.Active)
}

// SetActive sets the active workspace by path, alias, or name.
func (r *Registry) SetActive(identifier string) error {
	if identifier == "" {
		r.Active = ""
		return nil
	}
	entry := r.FindByNameOrAlias(identifier)
	if entry == nil {
		return &WorkspaceNotFoundError{Identifier: identifier}
	}
	// Store the alias if available, otherwise the path
	if entry.Alias != "" {
		r.Active = entry.Alias
	} else {
		r.Active = entry.Path
	}
	return nil
}

// AliasConflictError indicates the alias is already in use.
type AliasConflictError struct {
	Alias        string
	ExistingPath string
}

func (e *AliasConflictError) Error() string {
	return "alias '" + e.Alias + "' is already in use by workspace at " + e.ExistingPath
}

// WorkspaceNotFoundError indicates the workspace was not found.
type WorkspaceNotFoundError struct {
	Identifier string
}

func (e *WorkspaceNotFoundError) Error() string {
	return "workspace not found: " + e.Identifier
}
