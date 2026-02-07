// Package agent provides agent lifecycle management.
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RootFileName is the filename for root agent state.
const RootFileName = "root.json"

// ErrRootExists is returned when attempting to create a root when one already exists.
var ErrRootExists = errors.New("root agent already exists")

// ErrRootNotFound is returned when the root agent state file doesn't exist.
var ErrRootNotFound = errors.New("root agent not found")

// RootAgentState extends AgentState with root-specific fields.
// The root agent is special: it's the singleton entry point created by `bc up`.
type RootAgentState struct {
	AgentState
	Children    []string `json:"children,omitempty"`
	IsSingleton bool     `json:"is_singleton"`
}

// RootStateStore manages the root agent state file at .bc/agents/root.json
type RootStateStore struct {
	agentsDir string
	mu        sync.RWMutex
}

// NewRootStateStore creates a new root state store for the given .bc directory.
func NewRootStateStore(bcDir string) *RootStateStore {
	return &RootStateStore{
		agentsDir: filepath.Join(bcDir, "agents"),
	}
}

// rootFilePath returns the path to root.json.
func (s *RootStateStore) rootFilePath() string {
	return filepath.Join(s.agentsDir, RootFileName)
}

// tempFilePath returns a temporary file path for atomic writes.
func (s *RootStateStore) tempFilePath() string {
	return filepath.Join(s.agentsDir, "."+RootFileName+".tmp")
}

// ensureDir creates the agents directory if it doesn't exist.
func (s *RootStateStore) ensureDir() error {
	return os.MkdirAll(s.agentsDir, 0750)
}

// Exists checks if a root agent state file exists.
func (s *RootStateStore) Exists() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, err := os.Stat(s.rootFilePath())
	return err == nil
}

// Load reads the root agent state from disk.
// Returns ErrRootNotFound if the file doesn't exist.
func (s *RootStateStore) Load() (*RootAgentState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.rootFilePath()
	data, err := os.ReadFile(path) //nolint:gosec // path constructed from known agents dir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrRootNotFound
		}
		return nil, fmt.Errorf("failed to read root state: %w", err)
	}

	var state RootAgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal root state: %w", err)
	}

	return &state, nil
}

// Save persists the root agent state to disk atomically.
// It writes to a temp file first, then renames to ensure atomic updates.
func (s *RootStateStore) Save(state *RootAgentState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDir(); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Ensure singleton flag is set
	state.IsSingleton = true

	// Update timestamp
	state.UpdatedAt = time.Now()

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal root state: %w", err)
	}

	// Write to temp file first
	tempPath := s.tempFilePath()
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Atomic rename
	targetPath := s.rootFilePath()
	if err := os.Rename(tempPath, targetPath); err != nil {
		// Clean up temp file on failure
		_ = os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// Delete removes the root agent state file.
func (s *RootStateStore) Delete() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.rootFilePath()
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete root state: %w", err)
	}
	return nil
}

// EnsureSingleton validates that only one root agent exists.
// If a root already exists, returns ErrRootExists.
// This should be called before creating a new root agent.
func (s *RootStateStore) EnsureSingleton() error {
	if s.Exists() {
		return ErrRootExists
	}
	return nil
}

// Create creates a new root agent state if one doesn't exist.
// Returns ErrRootExists if a root already exists.
func (s *RootStateStore) Create(name string, role Role, tool string) (*RootAgentState, error) {
	if err := s.EnsureSingleton(); err != nil {
		return nil, err
	}

	state := &RootAgentState{
		AgentState: AgentState{
			Name:      name,
			Role:      role,
			Tool:      tool,
			State:     StateIdle,
			StartedAt: time.Now(),
		},
		IsSingleton: true,
		Children:    []string{},
	}

	if err := s.Save(state); err != nil {
		return nil, err
	}

	return state, nil
}

// GetOrCreate returns the existing root state or creates a new one.
// This is the primary method for `bc up` to use.
func (s *RootStateStore) GetOrCreate(name string, role Role, tool string) (*RootAgentState, bool, error) {
	// Try to load existing
	state, err := s.Load()
	if err == nil {
		return state, false, nil // existing root, not created
	}

	if !errors.Is(err, ErrRootNotFound) {
		return nil, false, err // unexpected error
	}

	// Create new root
	state, err = s.Create(name, role, tool)
	if err != nil {
		return nil, false, err
	}

	return state, true, nil // new root created
}

// AddChild adds a child agent name to the root's children list.
func (s *RootStateStore) AddChild(childName string) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	// Check if already a child
	for _, c := range state.Children {
		if c == childName {
			return nil // already exists
		}
	}

	state.Children = append(state.Children, childName)
	return s.Save(state)
}

// RemoveChild removes a child agent name from the root's children list.
func (s *RootStateStore) RemoveChild(childName string) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	filtered := make([]string, 0, len(state.Children))
	for _, c := range state.Children {
		if c != childName {
			filtered = append(filtered, c)
		}
	}

	state.Children = filtered
	return s.Save(state)
}

// UpdateState updates the root agent's state field.
func (s *RootStateStore) UpdateState(newState State) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	state.State = newState
	return s.Save(state)
}

// UpdateSession updates the root agent's session field.
func (s *RootStateStore) UpdateSession(session string) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	state.Session = session
	return s.Save(state)
}

// TmuxChecker interface for checking tmux session status.
// This allows for easier testing without real tmux.
type TmuxChecker interface {
	HasSession(name string) bool
}

// RootRecoveryResult describes the outcome of a root recovery check.
type RootRecoveryResult struct {
	State        *RootAgentState
	NeedsCreate  bool // No root state exists
	NeedsRecover bool // Root state exists but session dead
	IsRunning    bool // Root is running normally
}

// CheckRecovery checks if root needs to be created or recovered.
// This is the first step in `bc up` to determine what action to take.
func (s *RootStateStore) CheckRecovery(tmux TmuxChecker) (*RootRecoveryResult, error) {
	state, err := s.Load()
	if errors.Is(err, ErrRootNotFound) {
		return &RootRecoveryResult{NeedsCreate: true}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load root state: %w", err)
	}

	// Check if tmux session is alive
	if state.Session != "" && tmux.HasSession(state.Session) {
		return &RootRecoveryResult{
			State:     state,
			IsRunning: true,
		}, nil
	}

	// Session dead or missing - needs recovery
	return &RootRecoveryResult{
		State:        state,
		NeedsRecover: true,
	}, nil
}

// MarkRecovered updates root state after successful recovery.
func (s *RootStateStore) MarkRecovered(session string) error {
	state, err := s.Load()
	if err != nil {
		return err
	}

	state.Session = session
	state.State = StateIdle
	state.UpdatedAt = time.Now()

	return s.Save(state)
}

// GetChildren returns the list of child agent names.
func (s *RootStateStore) GetChildren() ([]string, error) {
	state, err := s.Load()
	if err != nil {
		return nil, err
	}
	return state.Children, nil
}
