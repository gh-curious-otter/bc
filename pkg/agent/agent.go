// Package agent provides agent lifecycle management.
package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/tmux"
)

// Role defines the type of agent.
type Role string

const (
	// Legacy roles (for backward compatibility)
	RoleCoordinator Role = "coordinator"
	RoleWorker      Role = "worker"

	// Hierarchical roles
	RoleProductManager Role = "product-manager" // Owns product vision, creates epics
	RoleManager        Role = "manager"         // Breaks down epics, manages engineers
	RoleEngineer       Role = "engineer"        // Implements tasks (like worker)
)

// Capability defines what actions a role can perform.
type Capability string

const (
	CapCreateAgents   Capability = "create_agents"   // Can spawn child agents
	CapAssignWork     Capability = "assign_work"     // Can assign work to others
	CapCreateEpics    Capability = "create_epics"    // Can create high-level epics
	CapImplementTasks Capability = "implement_tasks" // Can write code/implement
	CapReviewWork     Capability = "review_work"     // Can review others' work
)

// RoleCapabilities defines what each role can do.
var RoleCapabilities = map[Role][]Capability{
	RoleProductManager: {CapCreateAgents, CapAssignWork, CapCreateEpics, CapReviewWork},
	RoleManager:        {CapCreateAgents, CapAssignWork, CapReviewWork},
	RoleEngineer:       {CapImplementTasks},
	// Legacy mappings
	RoleCoordinator: {CapCreateAgents, CapAssignWork, CapReviewWork},
	RoleWorker:      {CapImplementTasks},
}

// RoleHierarchy defines which roles can create which child roles.
var RoleHierarchy = map[Role][]Role{
	RoleProductManager: {RoleManager},
	RoleManager:        {RoleEngineer},
	RoleEngineer:       {}, // Cannot create children
	// Legacy mappings
	RoleCoordinator: {RoleWorker, RoleManager, RoleEngineer},
	RoleWorker:      {},
}

// CanCreateRole checks if a parent role can create a child role.
func CanCreateRole(parent, child Role) bool {
	allowed, ok := RoleHierarchy[parent]
	if !ok {
		return false
	}
	for _, r := range allowed {
		if r == child {
			return true
		}
	}
	return false
}

// HasCapability checks if a role has a specific capability.
func HasCapability(role Role, cap Capability) bool {
	caps, ok := RoleCapabilities[role]
	if !ok {
		return false
	}
	for _, c := range caps {
		if c == cap {
			return true
		}
	}
	return false
}

// RoleLevel returns the hierarchy level (0 = top, higher = lower in hierarchy).
func RoleLevel(role Role) int {
	switch role {
	case RoleProductManager, RoleCoordinator:
		return 0
	case RoleManager:
		return 1
	case RoleEngineer, RoleWorker:
		return 2
	default:
		return 99
	}
}

// State represents the current state of an agent.
type State string

const (
	StateIdle     State = "idle"
	StateStarting State = "starting"
	StateWorking  State = "working"
	StateDone     State = "done"
	StateStuck    State = "stuck"
	StateError    State = "error"
	StateStopped  State = "stopped"
)

// Agent represents a running AI agent.
type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      Role      `json:"role"`
	State     State     `json:"state"`
	Workspace string    `json:"workspace"`
	Task      string    `json:"task,omitempty"`
	StartedAt time.Time `json:"started_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Session info
	Session string `json:"session"`

	// Hierarchy info
	ParentID string   `json:"parent_id,omitempty"` // ID of parent agent (who created this agent)
	Children []string `json:"children,omitempty"`  // IDs of child agents

	// For workers/engineers
	HookedWork string `json:"hooked_work,omitempty"`
}

// HasCapability checks if this agent has a specific capability.
func (a *Agent) HasCapability(cap Capability) bool {
	return HasCapability(a.Role, cap)
}

// CanCreate checks if this agent can create an agent with the given role.
func (a *Agent) CanCreate(childRole Role) bool {
	return CanCreateRole(a.Role, childRole)
}

// IsLeaf returns true if this agent has no children.
func (a *Agent) IsLeaf() bool {
	return len(a.Children) == 0
}

// Level returns the hierarchy level of this agent.
func (a *Agent) Level() int {
	return RoleLevel(a.Role)
}

// Manager handles agent lifecycle.
type Manager struct {
	mu       sync.RWMutex
	agents   map[string]*Agent
	tmux     *tmux.Manager
	stateDir string

	// Agent command (e.g., "claude" or "claude --dangerously-skip-permissions")
	agentCmd string

	// Workspace path for env vars
	workspacePath string
}

// NewManager creates a new agent manager with workspace-scoped tmux sessions.
func NewManager(stateDir string) *Manager {
	return &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager(config.Tmux.SessionPrefix),
		stateDir: stateDir,
		agentCmd: config.Agent.Command,
	}
}

// NewWorkspaceManager creates an agent manager scoped to a workspace.
// Tmux session names will be unique per workspace to avoid collisions.
func NewWorkspaceManager(stateDir, workspacePath string) *Manager {
	return &Manager{
		agents:        make(map[string]*Agent),
		tmux:          tmux.NewWorkspaceManager(config.Tmux.SessionPrefix, workspacePath),
		stateDir:      stateDir,
		agentCmd:      config.Agent.Command,
		workspacePath: workspacePath,
	}
}

// SetAgentCommand sets the command to run for agents.
func (m *Manager) SetAgentCommand(cmd string) {
	m.agentCmd = cmd
}

// SetAgentByName sets the agent command by looking up the agent name in config.
func (m *Manager) SetAgentByName(name string) bool {
	for _, a := range config.Agents {
		if a.Name == name {
			m.agentCmd = a.Command
			return true
		}
	}
	return false
}

// SpawnAgent creates and starts a new agent.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgent(name string, role Role, workspace string) (*Agent, error) {
	return m.SpawnAgentWithParent(name, role, workspace, "")
}

// SpawnAgentWithParent creates and starts a new agent with a parent relationship.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgentWithParent(name string, role Role, workspace string, parentID string) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate parent relationship if specified
	if parentID != "" {
		parent, exists := m.agents[parentID]
		if !exists {
			return nil, fmt.Errorf("parent agent %s not found", parentID)
		}
		if !CanCreateRole(parent.Role, role) {
			return nil, fmt.Errorf("agent %s (role %s) cannot create child with role %s", parentID, parent.Role, role)
		}
	}

	// Check if already exists in our state
	if existing, exists := m.agents[name]; exists {
		// If its tmux session is still alive, reuse it
		if m.tmux.HasSession(name) {
			existing.UpdatedAt = time.Now()
			m.saveState()
			return existing, nil
		}
		// Session is dead — clean up stale entry and respawn
		m.removeFromParent(name)
		delete(m.agents, name)
	}

	// If a tmux session exists from a previous crash, kill it first
	if m.tmux.HasSession(name) {
		_ = m.tmux.KillSession(name)
	}

	// Create agent
	agent := &Agent{
		ID:        name,
		Name:      name,
		Role:      role,
		State:     StateStarting,
		Workspace: workspace,
		Session:   name,
		ParentID:  parentID,
		Children:  []string{},
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Build env vars so the spawned process sees them immediately
	env := map[string]string{
		"BC_AGENT_ID":   name,
		"BC_AGENT_ROLE": string(role),
		"BC_WORKSPACE":  workspace,
	}
	if parentID != "" {
		env["BC_PARENT_ID"] = parentID
	}

	// Create tmux session with env vars baked into the command
	if err := m.tmux.CreateSessionWithEnv(name, workspace, m.agentCmd, env); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Update state
	agent.State = StateIdle
	agent.UpdatedAt = time.Now()
	m.agents[name] = agent

	// Update parent's children list
	if parentID != "" {
		if parent, exists := m.agents[parentID]; exists {
			parent.Children = append(parent.Children, name)
			parent.UpdatedAt = time.Now()
		}
	}

	// Save state
	m.saveState()

	return agent, nil
}

// SpawnChildAgent creates a child agent under a parent agent.
// Validates that the parent has permission to create the child role.
func (m *Manager) SpawnChildAgent(parentID, childName string, childRole Role, workspace string) (*Agent, error) {
	return m.SpawnAgentWithParent(childName, childRole, workspace, parentID)
}

// removeFromParent removes an agent from its parent's children list.
// Must be called while holding the lock.
func (m *Manager) removeFromParent(name string) {
	agent, exists := m.agents[name]
	if !exists || agent.ParentID == "" {
		return
	}

	parent, exists := m.agents[agent.ParentID]
	if !exists {
		return
	}

	// Remove from parent's children
	newChildren := make([]string, 0, len(parent.Children))
	for _, childID := range parent.Children {
		if childID != name {
			newChildren = append(newChildren, childID)
		}
	}
	parent.Children = newChildren
	parent.UpdatedAt = time.Now()
}

// StopAgent stops an agent.
func (m *Manager) StopAgent(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	// Kill tmux session
	if err := m.tmux.KillSession(name); err != nil {
		// Session might already be dead — that's fine
	}

	agent.State = StateStopped
	agent.UpdatedAt = time.Now()

	// Remove from parent's children list
	m.removeFromParent(name)

	m.saveState()

	return nil
}

// StopAgentTree stops an agent and all its children recursively.
func (m *Manager) StopAgentTree(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.stopAgentTreeLocked(name)
}

// stopAgentTreeLocked stops an agent tree while holding the lock.
func (m *Manager) stopAgentTreeLocked(name string) error {
	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	// Stop all children first (depth-first)
	for _, childID := range agent.Children {
		if err := m.stopAgentTreeLocked(childID); err != nil {
			// Continue stopping other children even if one fails
		}
	}

	// Kill this agent's tmux session
	if err := m.tmux.KillSession(name); err != nil {
		// Session might already be dead
	}

	agent.State = StateStopped
	agent.UpdatedAt = time.Now()
	agent.Children = []string{} // Clear children since they're stopped

	return nil
}

// StopAll stops all agents.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, agent := range m.agents {
		m.tmux.KillSession(name)
		agent.State = StateStopped
		agent.UpdatedAt = time.Now()
	}

	m.saveState()
	return nil
}

// GetAgent returns an agent by name.
func (m *Manager) GetAgent(name string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[name]
}

// ListAgents returns all agents sorted by role hierarchy then by name.
// Order: ProductManager/Coordinator → Manager → Engineer/Worker
func (m *Manager) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agents := make([]*Agent, 0, len(m.agents))
	for _, a := range m.agents {
		agents = append(agents, a)
	}

	sort.Slice(agents, func(i, j int) bool {
		// Sort by hierarchy level first
		levelI := RoleLevel(agents[i].Role)
		levelJ := RoleLevel(agents[j].Role)
		if levelI != levelJ {
			return levelI < levelJ
		}
		// Then by name
		return agents[i].Name < agents[j].Name
	})

	return agents
}

// ListChildren returns all direct children of an agent.
func (m *Manager) ListChildren(parentID string) []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	parent, exists := m.agents[parentID]
	if !exists {
		return nil
	}

	children := make([]*Agent, 0, len(parent.Children))
	for _, childID := range parent.Children {
		if child, exists := m.agents[childID]; exists {
			children = append(children, child)
		}
	}

	return children
}

// ListDescendants returns all descendants of an agent (children, grandchildren, etc.).
func (m *Manager) ListDescendants(parentID string) []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var descendants []*Agent
	m.collectDescendants(parentID, &descendants)
	return descendants
}

// collectDescendants recursively collects all descendants.
func (m *Manager) collectDescendants(parentID string, result *[]*Agent) {
	parent, exists := m.agents[parentID]
	if !exists {
		return
	}

	for _, childID := range parent.Children {
		if child, exists := m.agents[childID]; exists {
			*result = append(*result, child)
			m.collectDescendants(childID, result)
		}
	}
}

// GetParent returns the parent agent, or nil if no parent.
func (m *Manager) GetParent(agentID string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists || agent.ParentID == "" {
		return nil
	}

	return m.agents[agent.ParentID]
}

// ListByRole returns all agents with a specific role.
func (m *Manager) ListByRole(role Role) []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []*Agent
	for _, a := range m.agents {
		if a.Role == role {
			agents = append(agents, a)
		}
	}

	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Name < agents[j].Name
	})

	return agents
}

// RefreshState updates agent states from tmux.
// Also captures a live task summary from each agent's tmux pane.
func (m *Manager) RefreshState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sessions, err := m.tmux.ListSessions()
	if err != nil {
		return err
	}

	// Build map of active sessions
	active := make(map[string]bool)
	for _, s := range sessions {
		active[s.Name] = true
	}

	// Update agent states and capture live tasks
	for name, a := range m.agents {
		if !active[name] && a.State != StateStopped {
			a.State = StateStopped
			a.UpdatedAt = time.Now()
			continue
		}
		if !active[name] {
			continue
		}

		// Capture live task from tmux pane
		if live := m.captureLiveTask(name); live != "" {
			a.Task = live
		}
	}

	return nil
}

// captureLiveTask extracts a one-line activity summary from an agent's tmux pane.
func (m *Manager) captureLiveTask(name string) string {
	output, err := m.tmux.Capture(name, 15)
	if err != nil {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || line == "❯" {
			continue
		}

		// Skip status bar
		if strings.Contains(line, "bypass permissions") ||
			strings.Contains(line, "shift+Tab to cycle") ||
			strings.Contains(line, "Update available") {
			continue
		}

		// Spinner lines: active work
		if strings.HasPrefix(line, "✻") ||
			strings.HasPrefix(line, "✳") ||
			strings.HasPrefix(line, "✽") ||
			strings.HasPrefix(line, "·") {
			if idx := strings.LastIndex(line, "("); idx > 20 {
				line = strings.TrimSpace(line[:idx])
			}
			return line
		}

		// Tool call lines
		if strings.HasPrefix(line, "⏺") {
			return line
		}

		// Prompt line — agent waiting for input
		if strings.HasPrefix(line, "❯") && len(line) > 2 {
			return line
		}
	}

	return ""
}

// AgentCount returns the number of agents.
func (m *Manager) AgentCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}

// RunningCount returns the number of non-stopped agents.
func (m *Manager) RunningCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, a := range m.agents {
		if a.State != StateStopped {
			count++
		}
	}
	return count
}

// UpdateAgentState updates an agent's state and task.
func (m *Manager) UpdateAgentState(name string, state State, task string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	agent.State = state
	agent.Task = task
	agent.UpdatedAt = time.Now()

	m.saveState()
	return nil
}

// SendToAgent sends a message/command to an agent's session.
func (m *Manager) SendToAgent(name, message string) error {
	return m.tmux.SendKeys(name, message)
}

// CaptureOutput captures recent output from an agent's session.
func (m *Manager) CaptureOutput(name string, lines int) (string, error) {
	return m.tmux.Capture(name, lines)
}

// AttachToAgent returns the command to attach to an agent's session.
func (m *Manager) AttachToAgent(name string) error {
	cmd := m.tmux.AttachCmd(name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// saveState persists agent state to disk.
func (m *Manager) saveState() error {
	if m.stateDir == "" {
		return nil
	}

	if err := os.MkdirAll(m.stateDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.agents, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(m.stateDir, "agents.json"), data, 0644)
}

// LoadState loads agent state from disk.
func (m *Manager) LoadState() error {
	if m.stateDir == "" {
		return nil
	}

	data, err := os.ReadFile(filepath.Join(m.stateDir, "agents.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return json.Unmarshal(data, &m.agents)
}

// Tmux returns the underlying tmux manager.
func (m *Manager) Tmux() *tmux.Manager {
	return m.tmux
}
