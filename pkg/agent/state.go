// Package agent provides agent lifecycle management.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AgentState represents the persistent state of an agent.
// This is stored as a per-agent JSON file in .bc/agents/<name>.json
type AgentState struct {
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Tool      string    `json:"tool,omitempty"`
	Team      string    `json:"team,omitempty"`
	Parent    string    `json:"parent,omitempty"`
	Worktree  string    `json:"worktree,omitempty"`
	Session   string    `json:"session,omitempty"`
	Role      Role      `json:"role"`
	State     State     `json:"state"`
}

// StateStore manages per-agent state files in .bc/agents/
type StateStore struct {
	agentsDir string
	mu        sync.RWMutex
}

// NewStateStore creates a new state store for the given .bc directory.
func NewStateStore(bcDir string) *StateStore {
	return &StateStore{
		agentsDir: filepath.Join(bcDir, "agents"),
	}
}

// agentFilePath returns the path for an agent's state file.
func (s *StateStore) agentFilePath(name string) string {
	return filepath.Join(s.agentsDir, name+".json")
}

// tempFilePath returns a temporary file path for atomic writes.
func (s *StateStore) tempFilePath(name string) string {
	return filepath.Join(s.agentsDir, "."+name+".json.tmp")
}

// EnsureDir creates the agents directory if it doesn't exist.
func (s *StateStore) EnsureDir() error {
	return os.MkdirAll(s.agentsDir, 0750)
}

// Save persists an agent's state to disk atomically.
// It writes to a temp file first, then renames to ensure atomic updates.
func (s *StateStore) Save(state *AgentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.saveLocked(state)
}

// saveLocked persists an agent's state to disk without acquiring the lock.
// Caller must hold s.mu.Lock().
func (s *StateStore) saveLocked(state *AgentState) error {
	if err := s.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Update timestamp
	state.UpdatedAt = time.Now()

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agent state: %w", err)
	}

	// Write to temp file first
	tempPath := s.tempFilePath(state.Name)
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	targetPath := s.agentFilePath(state.Name)
	if err := os.Rename(tempPath, targetPath); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Load reads an agent's state from disk.
// Returns nil, nil if the agent file doesn't exist.
func (s *StateStore) Load(name string) (*AgentState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.agentFilePath(name)
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from known agents dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read agent state: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent state: %w", err)
	}

	return &state, nil
}

// Delete removes an agent's state file.
func (s *StateStore) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.agentFilePath(name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete agent state: %w", err)
	}
	return nil
}

// List returns the names of all agents with state files.
func (s *StateStore) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only include .json files, skip temp files
		if filepath.Ext(name) == ".json" && name[0] != '.' {
			names = append(names, name[:len(name)-5]) // strip .json
		}
	}
	return names, nil
}

// LoadAll reads all agent states from disk atomically.
// Holds the read lock for the entire operation to prevent race conditions
// where agents could be added, deleted, or modified between list and load.
func (s *StateStore) LoadAll() ([]*AgentState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.agentsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}

	var states []*AgentState
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Only include .json files, skip temp files
		if filepath.Ext(name) != ".json" || name[0] == '.' {
			continue
		}

		agentName := name[:len(name)-5] // strip .json
		state, err := s.loadLocked(agentName)
		if err != nil {
			return nil, fmt.Errorf("failed to load agent %s: %w", agentName, err)
		}
		if state != nil {
			states = append(states, state)
		}
	}
	return states, nil
}

// loadLocked reads an agent's state from disk without acquiring the lock.
// Caller must hold s.mu.RLock() or s.mu.Lock().
func (s *StateStore) loadLocked(name string) (*AgentState, error) {
	path := s.agentFilePath(name)
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from known agents dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read agent state: %w", err)
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal agent state: %w", err)
	}

	return &state, nil
}

// Exists checks if an agent state file exists.
func (s *StateStore) Exists(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.agentFilePath(name))
	return err == nil
}

// UpdateState updates an agent's state field atomically.
// Holds the write lock for the entire load-modify-save operation to prevent
// race conditions where changes could be lost.
func (s *StateStore) UpdateState(name string, newState State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.loadLocked(name)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("agent %s not found", name)
	}

	state.State = newState
	return s.saveLocked(state)
}

// ToAgentState converts an Agent to an AgentState for persistence.
func ToAgentState(a *Agent) *AgentState {
	return &AgentState{
		Name:      a.Name,
		Role:      a.Role,
		Tool:      a.Tool,
		Parent:    a.ParentID,
		State:     a.State,
		Worktree:  a.WorktreeDir,
		Session:   a.Session,
		StartedAt: a.StartedAt,
		UpdatedAt: a.UpdatedAt,
	}
}

// ToAgent converts an AgentState back to an Agent.
func (s *AgentState) ToAgent(workspace string) *Agent {
	return &Agent{
		Name:        s.Name,
		ID:          s.Name, // Use name as ID for v2
		Role:        s.Role,
		Tool:        s.Tool,
		ParentID:    s.Parent,
		State:       s.State,
		WorktreeDir: s.Worktree,
		Session:     s.Session,
		Workspace:   workspace,
		StartedAt:   s.StartedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}
