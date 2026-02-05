// Package workspace provides workspace/project management.
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rpuneet/bc/config"
)

// Config represents workspace configuration.
type Config struct {
	Version  int    `json:"version"`
	Name     string `json:"name"`
	RootDir  string `json:"root_dir"`
	StateDir string `json:"state_dir"`

	// Agent settings
	MaxWorkers   int    `json:"max_workers"`
	AgentCommand string `json:"agent_command,omitempty"` // Custom command (overrides Tool)
	Tool         string `json:"tool,omitempty"`          // Tool type: claude, cursor, codex, server
}

// Workspace represents an active workspace.
type Workspace struct {
	Config  Config
	RootDir string
}

// DefaultConfig returns default workspace configuration.
func DefaultConfig(rootDir string) Config {
	return Config{
		Version:    1,
		Name:       filepath.Base(rootDir),
		RootDir:    rootDir,
		StateDir:   filepath.Join(rootDir, config.Workspace.StateDir),
		MaxWorkers: int(config.Workspace.MaxWorkers),
	}
}

// Init initializes a new workspace in the given directory.
func Init(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	// Create state directory
	stateDir := filepath.Join(absRoot, ".bc")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create default config
	config := DefaultConfig(absRoot)

	// Save config
	configPath := filepath.Join(stateDir, "config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return nil, err
	}

	return &Workspace{
		Config:  config,
		RootDir: absRoot,
	}, nil
}

// Load loads a workspace from a directory.
func Load(rootDir string) (*Workspace, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(absRoot, ".bc")
	configPath := filepath.Join(stateDir, "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not a bc workspace (no .bc/config.json found)")
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid config.json: %w", err)
	}

	// Update paths if directory was moved
	config.RootDir = absRoot
	config.StateDir = stateDir

	return &Workspace{
		Config:  config,
		RootDir: absRoot,
	}, nil
}

// Find searches for a workspace starting from dir and going up.
func Find(dir string) (*Workspace, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	current := absDir
	for {
		// Check for .bc directory
		stateDir := filepath.Join(current, ".bc")
		if _, err := os.Stat(stateDir); err == nil {
			return Load(current)
		}

		// Go up one directory
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			return nil, fmt.Errorf("no workspace found (searched from %s to root)", absDir)
		}
		current = parent
	}
}

// Save saves the workspace configuration.
func (w *Workspace) Save() error {
	configPath := filepath.Join(w.Config.StateDir, "config.json")
	data, err := json.MarshalIndent(w.Config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// StateDir returns the state directory path.
func (w *Workspace) StateDir() string {
	return w.Config.StateDir
}

// AgentsDir returns the agents state directory.
func (w *Workspace) AgentsDir() string {
	return filepath.Join(w.Config.StateDir, "agents")
}

// LogsDir returns the logs directory.
func (w *Workspace) LogsDir() string {
	return filepath.Join(w.Config.StateDir, "logs")
}

// EnsureDirs creates all required directories.
func (w *Workspace) EnsureDirs() error {
	dirs := []string{
		w.Config.StateDir,
		w.AgentsDir(),
		w.LogsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// IsWorkspace checks if a directory is a workspace.
func IsWorkspace(dir string) bool {
	stateDir := filepath.Join(dir, config.Workspace.StateDir)
	_, err := os.Stat(stateDir)
	return err == nil
}
