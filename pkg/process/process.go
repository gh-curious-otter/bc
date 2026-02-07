// Package process provides managed process lifecycle for bc workspaces.
package process

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// State represents the state of a managed process.
type State string

const (
	// StateRunning indicates the process is running.
	StateRunning State = "running"

	// StateStopped indicates the process has stopped.
	StateStopped State = "stopped"

	// StateFailed indicates the process exited with an error.
	StateFailed State = "failed"
)

// Process represents a managed background process.
type Process struct {
	// cmd is the underlying exec.Cmd (not persisted).
	cmd *exec.Cmd

	// StartedAt is when the process was started.
	StartedAt time.Time `json:"started_at"`

	// StoppedAt is when the process stopped (if stopped).
	StoppedAt time.Time `json:"stopped_at,omitempty"`

	// Name is the unique identifier for this process.
	Name string `json:"name"`

	// Command is the command being executed.
	Command string `json:"command"`

	// WorkDir is the working directory for the process.
	WorkDir string `json:"work_dir,omitempty"`

	// Owner is the agent that started this process.
	Owner string `json:"owner,omitempty"`

	// State is the current process state.
	State State `json:"state"`

	// Args are the command arguments.
	Args []string `json:"args,omitempty"`

	// PID is the process ID (0 if not running).
	PID int `json:"pid,omitempty"`

	// ExitCode is the exit code (only valid if stopped).
	ExitCode int `json:"exit_code,omitempty"`
}

// Manager handles process lifecycle operations.
type Manager struct {
	processes map[string]*Process
	stateDir  string
	mu        sync.RWMutex
}

// NewManager creates a new process manager.
func NewManager(stateDir string) *Manager {
	return &Manager{
		processes: make(map[string]*Process),
		stateDir:  filepath.Join(stateDir, ".bc", "processes"),
	}
}

// Start starts a new managed process.
func (m *Manager) Start(name, command string, args []string, workDir, owner string) (*Process, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if process already exists and is running
	if existing, ok := m.processes[name]; ok {
		if existing.State == StateRunning && existing.isAlive() {
			return nil, fmt.Errorf("process %q is already running (PID %d)", name, existing.PID)
		}
	}

	// Create the process
	proc := &Process{
		Name:      name,
		Command:   command,
		Args:      args,
		WorkDir:   workDir,
		Owner:     owner,
		StartedAt: time.Now(),
		State:     StateRunning,
	}

	// Prepare the command
	cmd := exec.CommandContext(context.Background(), command, args...) //nolint:gosec // command is user-provided
	if workDir != "" {
		cmd.Dir = workDir
	}

	// Start the process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	proc.cmd = cmd
	proc.PID = cmd.Process.Pid

	// Monitor the process in background
	go m.monitor(name, cmd)

	m.processes[name] = proc

	// Save state
	if err := m.saveState(); err != nil {
		// Log but don't fail
		fmt.Fprintf(os.Stderr, "warning: failed to save process state: %v\n", err)
	}

	return proc, nil
}

// Stop stops a running process.
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proc, ok := m.processes[name]
	if !ok {
		return fmt.Errorf("process %q not found", name)
	}

	if proc.State != StateRunning {
		return fmt.Errorf("process %q is not running (state: %s)", name, proc.State)
	}

	// Try graceful shutdown first (SIGTERM)
	if proc.cmd != nil && proc.cmd.Process != nil {
		if err := proc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			// If SIGTERM fails, try SIGKILL
			if killErr := proc.cmd.Process.Kill(); killErr != nil {
				return fmt.Errorf("failed to stop process: %w", killErr)
			}
		}
	}

	proc.State = StateStopped
	proc.StoppedAt = time.Now()

	return m.saveState()
}

// Get returns a process by name.
func (m *Manager) Get(name string) (*Process, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	proc, ok := m.processes[name]
	return proc, ok
}

// List returns all processes.
func (m *Manager) List() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Process, 0, len(m.processes))
	for _, proc := range m.processes {
		result = append(result, proc)
	}
	return result
}

// ListRunning returns only running processes.
func (m *Manager) ListRunning() []*Process {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*Process
	for _, proc := range m.processes {
		if proc.State == StateRunning {
			result = append(result, proc)
		}
	}
	return result
}

// RefreshState updates process states by checking if they're still alive.
func (m *Manager) RefreshState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, proc := range m.processes {
		if proc.State == StateRunning && !proc.isAlive() {
			proc.State = StateStopped
			proc.StoppedAt = time.Now()
		}
	}

	return m.saveState()
}

// isAlive checks if a process is still running.
func (p *Process) isAlive() bool {
	if p.PID == 0 {
		return false
	}

	process, err := os.FindProcess(p.PID)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// monitor watches a process and updates state when it exits.
func (m *Manager) monitor(name string, cmd *exec.Cmd) {
	err := cmd.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()

	proc, ok := m.processes[name]
	if !ok {
		return
	}

	proc.StoppedAt = time.Now()
	if err != nil {
		proc.State = StateFailed
		if exitErr, ok := err.(*exec.ExitError); ok {
			proc.ExitCode = exitErr.ExitCode()
		}
	} else {
		proc.State = StateStopped
		proc.ExitCode = 0
	}

	_ = m.saveState()
}

// saveState persists process state to disk.
func (m *Manager) saveState() error {
	if err := os.MkdirAll(m.stateDir, 0750); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	statePath := filepath.Join(m.stateDir, "processes.json")

	// Convert to slice for JSON
	procs := make([]*Process, 0, len(m.processes))
	for _, p := range m.processes {
		procs = append(procs, p)
	}

	data, err := json.MarshalIndent(procs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}

// LoadState loads process state from disk.
func (m *Manager) LoadState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	statePath := filepath.Join(m.stateDir, "processes.json")

	data, err := os.ReadFile(statePath) //nolint:gosec // path is constructed from trusted stateDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state: %w", err)
	}

	var procs []*Process
	if err := json.Unmarshal(data, &procs); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	m.processes = make(map[string]*Process)
	for _, p := range procs {
		m.processes[p.Name] = p
	}

	return nil
}

// RunningCount returns the number of running processes.
func (m *Manager) RunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, proc := range m.processes {
		if proc.State == StateRunning {
			count++
		}
	}
	return count
}
