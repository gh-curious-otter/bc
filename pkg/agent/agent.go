// Package agent provides agent lifecycle management.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/memory"
	"github.com/rpuneet/bc/pkg/tmux"
	"github.com/rpuneet/bc/pkg/workspace"
)

// containsShellMetachars checks if a string contains shell metacharacters that could be exploited
func containsShellMetachars(s string) bool {
	// Check for common shell injection characters
	dangerous := []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "[", "]", "<", ">", "\n", "\r"}
	for _, char := range dangerous {
		if strings.Contains(s, char) {
			return true
		}
	}
	return false
}

// Role defines the type of agent.
type Role string

const (
	// RoleRoot is the only hardcoded role - a singleton root agent.
	// All other roles are defined in workspace .bc/roles/*.md files.
	RoleRoot Role = "root"
)

// Capability defines what actions a role can perform.
type Capability string

const (
	CapCreateAgents   Capability = "create_agents"   // Can spawn child agents
	CapAssignWork     Capability = "assign_work"     // Can assign work to others
	CapCreateEpics    Capability = "create_epics"    // Can create high-level epics
	CapImplementTasks Capability = "implement_tasks" // Can write code/implement
	CapReviewWork     Capability = "review_work"     // Can review others' work
	CapTestWork       Capability = "test_work"       // Can test and validate implementations
)

// RoleCapabilities and RoleHierarchy are empty here.
// All role definitions (capabilities, hierarchy, metadata) are loaded from
// workspace .bc/roles/*.md files via RoleManager.
// Only the root role has hardcoded capabilities.
var RoleCapabilities = map[Role][]Capability{
	RoleRoot: {CapCreateAgents, CapAssignWork, CapCreateEpics, CapReviewWork}, // Root can do everything
}

var RoleHierarchy = map[Role][]Role{
	// Root can create any role defined in workspace (checked at runtime)
	RoleRoot: {}, // Empty - all roles allowed, checked dynamically
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

// RoleLevel returns the hierarchy level for built-in roles.
// Custom roles loaded from .bc/roles/*.md return level 1 by default.
func RoleLevel(role Role) int {
	switch role {
	case RoleRoot:
		return -1 // Root is at the top
	default:
		return 1 // All custom roles are at level 1
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

// validTransitions defines allowed state transitions. Internal transitions
// (e.g. spawn setting starting→idle, stop setting →stopped) bypass this
// validation and set state directly. This map governs transitions through
// UpdateAgentState, which is called by bc report.
var validTransitions = map[State][]State{
	StateStarting: {StateIdle, StateError, StateStopped},
	StateIdle:     {StateIdle, StateWorking, StateDone, StateStuck, StateError, StateStopped},
	StateWorking:  {StateWorking, StateIdle, StateDone, StateStuck, StateError, StateStopped},
	StateDone:     {StateIdle, StateWorking, StateStopped},
	StateStuck:    {StateStuck, StateIdle, StateWorking, StateError, StateStopped},
	StateError:    {StateIdle, StateWorking, StateStopped},
	StateStopped:  {StateIdle, StateStarting},
}

// ValidateTransition checks whether a state transition from → to is allowed.
// Returns an error if the transition is invalid.
func ValidateTransition(from, to State) error {
	allowed, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("unknown current state: %s", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid state transition: %s → %s", from, to)
}

// AgentMemory holds role-specific content loaded from prompts/<role>.md.
type AgentMemory struct {
	// LoadedAt is when the memory was loaded.
	LoadedAt time.Time `json:"loaded_at,omitempty"`
	// RolePrompt is the full content of the role's prompt file.
	RolePrompt string `json:"role_prompt,omitempty"`
}

// Agent represents a running AI agent.
type Agent struct {
	UpdatedAt   time.Time    `json:"updated_at"`
	StartedAt   time.Time    `json:"started_at"`
	Memory      *AgentMemory `json:"memory,omitempty"`
	Workspace   string       `json:"workspace"`
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Task        string       `json:"task,omitempty"`
	Session     string       `json:"session"`
	Tool        string       `json:"tool,omitempty"`
	ParentID    string       `json:"parent_id,omitempty"`
	HookedWork  string       `json:"hooked_work,omitempty"`
	WorktreeDir string       `json:"worktree_dir,omitempty"`
	MemoryDir   string       `json:"memory_dir,omitempty"`
	Team        string       `json:"team,omitempty"`
	Role        Role         `json:"role"`
	State       State        `json:"state"`
	Children    []string     `json:"children,omitempty"`
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

// LoadRoleMemory loads role-specific prompt content from .bc/roles/<role>.md.
// For the root role, loads from .bc/prompts/root.md for backward compatibility.
// Returns nil AgentMemory if the file doesn't exist.
func LoadRoleMemory(workspacePath string, role Role) *AgentMemory {
	// For root role, try backward compatible location first
	if role == RoleRoot {
		rootPromptPath := filepath.Join(workspacePath, "prompts", "root.md")
		//nolint:gosec // path constructed from trusted workspace root
		if data, err := os.ReadFile(rootPromptPath); err == nil {
			return &AgentMemory{
				RolePrompt: string(data),
				LoadedAt:   time.Now(),
			}
		}
	}

	// Load role from .bc/roles/<role>.md using RoleManager
	stateDir := filepath.Join(workspacePath, ".bc")
	rm := workspace.NewRoleManager(stateDir)
	roleObj, err := rm.LoadRole(string(role))
	if err != nil {
		log.Debug("failed to load role prompt", "role", role, "error", err)
		return nil
	}

	if roleObj.Prompt == "" {
		return nil
	}

	return &AgentMemory{
		RolePrompt: roleObj.Prompt,
		LoadedAt:   time.Now(),
	}
}

// DefaultBootstrapDelay is the default time to wait before sending bootstrap
// prompts after starting an agent. Different AI tools have different startup
// times, so this can be configured per-manager.
const DefaultBootstrapDelay = 3 * time.Second

// Manager handles agent lifecycle.
type Manager struct {
	agents map[string]*Agent
	tmux   *tmux.Manager

	stateDir string

	// Agent command (e.g., "claude" or "claude --dangerously-skip-permissions")
	agentCmd string

	// Workspace path for env vars
	workspacePath string

	// BootstrapDelay is the time to wait before sending bootstrap prompts.
	// If zero, DefaultBootstrapDelay is used.
	BootstrapDelay time.Duration

	mu sync.RWMutex
}

// NewManager creates a new agent manager with workspace-scoped tmux sessions.
func NewManager(stateDir string) *Manager {
	return &Manager{
		agents:   make(map[string]*Agent),
		tmux:     tmux.NewManager(config.Tmux.SessionPrefix),
		stateDir: stateDir,
		agentCmd: config.AgentLegacy.Command,
	}
}

// NewWorkspaceManager creates an agent manager scoped to a workspace.
// Tmux session names will be unique per workspace to avoid collisions.
func NewWorkspaceManager(stateDir, workspacePath string) *Manager {
	return &Manager{
		agents:        make(map[string]*Agent),
		tmux:          tmux.NewWorkspaceManager(config.Tmux.SessionPrefix, workspacePath),
		stateDir:      stateDir,
		agentCmd:      config.AgentLegacy.Command,
		workspacePath: workspacePath,
	}
}

// SetAgentCommand sets the command to run for agents.
func (m *Manager) SetAgentCommand(cmd string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentCmd = cmd
}

// SetAgentByName sets the agent command by looking up the agent name in config.
func (m *Manager) SetAgentByName(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, a := range config.Agents {
		if a.Name == name {
			m.agentCmd = a.Command
			return true
		}
	}
	return false
}

// SetBootstrapDelay sets the delay before sending bootstrap prompts.
func (m *Manager) SetBootstrapDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.BootstrapDelay = d
}

// getBootstrapDelay returns the configured bootstrap delay or the default.
func (m *Manager) getBootstrapDelay() time.Duration {
	if m.BootstrapDelay > 0 {
		return m.BootstrapDelay
	}
	return DefaultBootstrapDelay
}

// GetAgentCommand returns the command for a tool name from config.
// Returns the command and true if found, or empty string and false if not.
func GetAgentCommand(toolName string) (string, bool) {
	for _, a := range config.Agents {
		if a.Name == toolName {
			return a.Command, true
		}
	}
	return "", false
}

// ListAvailableTools returns a list of configured tool names.
func ListAvailableTools() []string {
	tools := make([]string, 0, len(config.Agents))
	for _, a := range config.Agents {
		tools = append(tools, a.Name)
	}
	return tools
}

// SpawnAgent creates and starts a new agent.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgent(name string, role Role, workspace string) (*Agent, error) {
	return m.SpawnAgentWithOptions(name, role, workspace, "", "")
}

// SpawnAgentWithTool creates and starts a new agent with a specific tool.
// If tool is empty, uses the manager's default agent command.
func (m *Manager) SpawnAgentWithTool(name string, role Role, workspace string, tool string) (*Agent, error) {
	return m.SpawnAgentWithOptions(name, role, workspace, "", tool)
}

// SpawnAgentWithParent creates and starts a new agent with a parent relationship.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgentWithParent(name string, role Role, workspace string, parentID string) (*Agent, error) {
	return m.SpawnAgentWithOptions(name, role, workspace, parentID, "")
}

// SpawnAgentWithOptions creates and starts a new agent with all options.
// If tool is empty, uses the manager's default agent command.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgentWithOptions(name string, role Role, workspace string, parentID string, tool string) (*Agent, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debug("spawning agent", "name", name, "role", role, "workspace", workspace, "parentID", parentID, "tool", tool)

	// Enforce root singleton constraint
	if role == RoleRoot {
		if err := m.enforceRootSingleton(workspace); err != nil {
			return nil, err
		}
	}

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
			// Create worktree if missing (agents created before worktree feature)
			if existing.WorktreeDir == "" {
				if wtDir, err := createWorktree(workspace, name); err == nil {
					existing.WorktreeDir = wtDir
				}
			}
			existing.UpdatedAt = time.Now()
			if err := m.saveState(); err != nil {
				log.Warn("failed to save agent state", "error", err)
			}
			return existing, nil
		}
		// Session is dead but agent is in an active state — only respawn
		// if agent is in a terminal state (stopped/error). Otherwise preserve
		// the record so we don't overwrite working/stuck state.
		switch existing.State {
		case StateStopped, StateError:
			// Terminal state — clean up and respawn below
			m.removeFromParent(name)
			delete(m.agents, name)
		default:
			// Active state (working, idle, stuck, etc.) with dead session —
			// restart the tmux session but preserve agent state.
			agentCmd := m.agentCmd
			if existing.Tool != "" {
				if cmd, ok := GetAgentCommand(existing.Tool); ok {
					agentCmd = cmd
				}
			}

			// Ensure worktree exists for respawn
			sessionDir := workspace
			if existing.WorktreeDir != "" {
				sessionDir = existing.WorktreeDir
			} else {
				if wtDir, err := createWorktree(workspace, name); err == nil {
					sessionDir = wtDir
					existing.WorktreeDir = wtDir
				}
			}

			// Install git wrapper for worktree enforcement
			if wrapErr := ensureGitWrapper(workspace); wrapErr != nil {
				log.Warn("failed to install git wrapper", "error", wrapErr)
			}

			// Build PATH safely: only use trusted paths, not user-controlled ones
			bcBinPath := filepath.Join(workspace, ".bc", "bin")
			systemPath := os.Getenv("PATH")
			// Validate PATH doesn't contain shell metacharacters
			if containsShellMetachars(systemPath) {
				log.Warn("PATH contains suspicious characters, using minimal PATH")
				systemPath = "/usr/local/bin:/usr/bin:/bin"
			}

			env := map[string]string{
				"BC_AGENT_ID":       name,
				"BC_AGENT_ROLE":     string(existing.Role),
				"BC_WORKSPACE":      workspace,
				"BC_AGENT_WORKTREE": sessionDir,
				"PATH":              bcBinPath + ":" + systemPath,
			}
			if existing.Tool != "" {
				env["BC_AGENT_TOOL"] = existing.Tool
			}
			if existing.ParentID != "" {
				env["BC_PARENT_ID"] = existing.ParentID
			}
			if err := m.tmux.CreateSessionWithEnv(name, sessionDir, agentCmd, env); err != nil {
				return nil, fmt.Errorf("failed to recreate tmux session: %w", err)
			}
			existing.UpdatedAt = time.Now()
			if err := m.saveState(); err != nil {
				log.Warn("failed to save agent state", "error", err)
			}
			return existing, nil
		}
	}

	// If a tmux session exists from a previous crash, kill it first
	if m.tmux.HasSession(name) {
		if err := m.tmux.KillSession(name); err != nil {
			log.Warn("failed to kill existing session", "session", name, "error", err)
		}
	}

	// Determine the command to use
	agentCmd := m.agentCmd
	if tool != "" {
		if cmd, ok := GetAgentCommand(tool); ok {
			agentCmd = cmd
		} else {
			return nil, fmt.Errorf("unknown tool %q, available tools: %v", tool, ListAvailableTools())
		}
	}

	// Create agent
	agent := &Agent{
		ID:        name,
		Name:      name,
		Role:      role,
		State:     StateStarting,
		Workspace: workspace,
		Session:   name,
		Tool:      tool,
		ParentID:  parentID,
		Children:  []string{},
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create per-agent git worktree so agents don't clobber each other
	worktreeDir, err := createWorktree(workspace, name)
	if err != nil {
		log.Warn("failed to create worktree, falling back to shared workspace", "agent", name, "error", err)
		worktreeDir = workspace
	}
	agent.WorktreeDir = worktreeDir

	// Create memory directory for agent
	memoryDir, err := createMemoryDir(workspace, name)
	if err != nil {
		log.Warn("failed to create memory dir", "agent", name, "error", err)
	} else {
		agent.MemoryDir = memoryDir
	}

	// Install git wrapper for worktree enforcement
	if err := ensureGitWrapper(workspace); err != nil {
		log.Warn("failed to install git wrapper", "error", err)
	}

	// Build env vars so the spawned process sees them immediately
	// Build PATH safely: only use trusted paths, not user-controlled ones
	bcBinPath := filepath.Join(workspace, ".bc", "bin")
	systemPath := os.Getenv("PATH")
	// Validate PATH doesn't contain shell metacharacters
	if containsShellMetachars(systemPath) {
		log.Warn("PATH contains suspicious characters, using minimal PATH")
		systemPath = "/usr/local/bin:/usr/bin:/bin"
	}

	env := map[string]string{
		"BC_AGENT_ID":       name,
		"BC_AGENT_ROLE":     string(role),
		"BC_WORKSPACE":      workspace,
		"BC_AGENT_WORKTREE": worktreeDir,
		"PATH":              bcBinPath + ":" + systemPath,
	}
	if tool != "" {
		env["BC_AGENT_TOOL"] = tool
	}
	if parentID != "" {
		env["BC_PARENT_ID"] = parentID
	}
	if agent.MemoryDir != "" {
		env["BC_AGENT_MEMORY"] = agent.MemoryDir
	}

	// Create tmux session in the agent's worktree directory
	if err := m.tmux.CreateSessionWithEnv(name, worktreeDir, agentCmd, env); err != nil {
		return nil, fmt.Errorf("failed to create tmux session: %w", err)
	}

	// Load role memory from prompts/<role>.md
	agent.Memory = LoadRoleMemory(workspace, role)

	// Update state
	agent.State = StateIdle
	agent.UpdatedAt = time.Now()
	m.agents[name] = agent

	// Build bootstrap prompt with role prompt and agent memories
	var promptParts []string

	// Add role prompt if available
	if agent.Memory != nil && agent.Memory.RolePrompt != "" {
		promptParts = append(promptParts, agent.Memory.RolePrompt)
	}

	// Load and inject agent memories from .bc/memory/<agent-name>/
	if agent.MemoryDir != "" {
		memStore := memory.NewStore(workspace, name)
		if memStore.Exists() {
			memCtx, memErr := memStore.GetMemoryContext(memory.DefaultMemoryLimit)
			if memErr != nil {
				log.Warn("failed to load agent memories", "agent", name, "error", memErr)
			} else if memCtx != "" {
				promptParts = append(promptParts, memCtx)
				log.Debug("injected agent memories", "agent", name)
			}
		}
	}

	// Send bootstrap prompt if we have content
	if len(promptParts) > 0 {
		// Wait for agent to initialize (Gemini/Claude needs time to start REPL)
		time.Sleep(m.getBootstrapDelay())

		prompt := strings.Join(promptParts, "\n\n---\n\n")
		prompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nAgent ID: %s\n", workspace, name)
		if err := m.tmux.SendKeys(name, prompt); err != nil {
			log.Warn("failed to send bootstrap prompt", "agent", name, "error", err)
		}
	}

	// Update parent's children list
	if parentID != "" {
		if parent, exists := m.agents[parentID]; exists {
			parent.Children = append(parent.Children, name)
			parent.UpdatedAt = time.Now()
		}
	}

	// Save state
	if err := m.saveState(); err != nil {
		log.Warn("failed to save agent state", "error", err)
	}

	return agent, nil
}

// createWorktree creates a per-agent git worktree so agents don't clobber each other.
// Returns the worktree directory path.
func createWorktree(workspace, agentName string) (string, error) {
	worktreeDir := filepath.Join(workspace, ".bc", "worktrees", agentName)

	// If worktree already exists, reuse it
	if _, err := os.Stat(worktreeDir); err == nil {
		log.Debug("reusing existing worktree", "agent", agentName, "dir", worktreeDir)
		return worktreeDir, nil
	}

	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(worktreeDir), 0750); err != nil {
		return "", fmt.Errorf("failed to create worktrees dir: %w", err)
	}

	// Create detached worktree at HEAD (current main)
	cmd := exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "add", "--detach", worktreeDir, "HEAD") //nolint:gosec // args are trusted internal paths
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If failed because it's already registered but missing, prune and retry
		if strings.Contains(string(output), "already registered") {
			log.Info("pruning stale worktrees to recover", "agent", agentName)
			_ = exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "prune").Run()

			// Retry
			cmd = exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "add", "--detach", worktreeDir, "HEAD") //nolint:gosec // args are trusted internal paths
			output, err = cmd.CombinedOutput()
		}

		if err != nil {
			return "", fmt.Errorf("failed to create worktree for %s: %w (%s)", agentName, err, string(output))
		}
	}

	log.Debug("created worktree", "agent", agentName, "dir", worktreeDir)
	return worktreeDir, nil
}

// gitWrapperScript is the shell script that shadows git to warn on write
// operations outside the agent's worktree. It always execs real git — never
// blocks — and is a no-op when BC_AGENT_WORKTREE is unset (tests, humans).
const gitWrapperScript = `#!/bin/bash
# bc worktree enforcement — warns on git write ops outside agent worktree
REAL_GIT="/usr/bin/git"

# No-op when BC_AGENT_WORKTREE is unset (tests, human usage)
if [ -z "$BC_AGENT_WORKTREE" ]; then
    exec "$REAL_GIT" "$@"
fi

# Check if CWD is inside the agent's worktree
case "$PWD" in
    "$BC_AGENT_WORKTREE"*) ;; # Inside worktree — OK
    *)
        # Warn only on write operations, not reads
        case "$1" in
            checkout|commit|push|reset|clean|merge|rebase|stash|add|rm|mv|init)
                echo "WARNING: git $1 outside worktree ($PWD != $BC_AGENT_WORKTREE)" >&2
                ;;
        esac
        ;;
esac

exec "$REAL_GIT" "$@"
`

// ensureGitWrapper creates the .bc/bin/git wrapper script if it does not
// already exist. The wrapper shadows /usr/bin/git on PATH and warns when
// agents run git write operations outside their worktree.
func ensureGitWrapper(workspace string) error {
	binDir := filepath.Join(workspace, ".bc", "bin")
	wrapperPath := filepath.Join(binDir, "git")

	// Idempotent — skip if already exists
	if _, err := os.Stat(wrapperPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(binDir, 0750); err != nil {
		return fmt.Errorf("failed to create .bc/bin: %w", err)
	}

	if err := os.WriteFile(wrapperPath, []byte(gitWrapperScript), 0700); err != nil { //nolint:gosec // executable script needs 0700
		return fmt.Errorf("failed to write git wrapper: %w", err)
	}

	return nil
}

// createMemoryDir creates the per-agent memory directory structure.
// Memory is stored in .bc/memory/<agent-name>/ with:
// - experiences.jsonl for task outcomes
// - learnings.md for agent insights
func createMemoryDir(workspace, agentName string) (string, error) {
	memoryDir := filepath.Join(workspace, ".bc", "memory", agentName)

	// If memory dir already exists, reuse it
	if _, err := os.Stat(memoryDir); err == nil {
		log.Debug("reusing existing memory dir", "agent", agentName, "dir", memoryDir)
		return memoryDir, nil
	}

	// Create memory directory
	if err := os.MkdirAll(memoryDir, 0750); err != nil {
		return "", fmt.Errorf("failed to create memory dir: %w", err)
	}

	// Initialize experiences.jsonl (empty JSONL file)
	experiencesPath := filepath.Join(memoryDir, "experiences.jsonl")
	if err := os.WriteFile(experiencesPath, []byte{}, 0600); err != nil {
		return "", fmt.Errorf("failed to create experiences.jsonl: %w", err)
	}

	// Initialize learnings.md with header
	learningsPath := filepath.Join(memoryDir, "learnings.md")
	learningsContent := fmt.Sprintf("# %s Learnings\n\nAgent insights and lessons learned.\n", agentName)
	if err := os.WriteFile(learningsPath, []byte(learningsContent), 0600); err != nil {
		return "", fmt.Errorf("failed to create learnings.md: %w", err)
	}

	log.Debug("created memory dir", "agent", agentName, "dir", memoryDir)
	return memoryDir, nil
}

// removeWorktree removes a per-agent git worktree.
func removeWorktree(workspace, worktreeDir string) {
	if worktreeDir == "" {
		return
	}
	cmd := exec.CommandContext(context.Background(), "git", "-C", workspace, "worktree", "remove", "--force", worktreeDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Warn("failed to remove worktree", "dir", worktreeDir, "error", err, "output", string(output))
	} else {
		log.Debug("removed worktree", "dir", worktreeDir)
	}
}

// SpawnChildAgent creates a child agent under a parent agent.
// Validates that the parent has permission to create the child role.
func (m *Manager) SpawnChildAgent(parentID, childName string, childRole Role, workspace string) (*Agent, error) {
	return m.SpawnAgentWithOptions(childName, childRole, workspace, parentID, "")
}

// SpawnChildAgentWithTool creates a child agent under a parent agent with a specific tool.
// Validates that the parent has permission to create the child role.
func (m *Manager) SpawnChildAgentWithTool(parentID, childName string, childRole Role, workspace, tool string) (*Agent, error) {
	return m.SpawnAgentWithOptions(childName, childRole, workspace, parentID, tool)
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

	log.Debug("stopping agent", "name", name)

	agent, exists := m.agents[name]
	if !exists {
		log.Warn("agent not found", "name", name)
		return fmt.Errorf("agent %s not found", name)
	}

	// Kill tmux session (ignore error - session might already be dead)
	_ = m.tmux.KillSession(name)

	// Clean up per-agent git worktree
	if agent.WorktreeDir != "" && agent.WorktreeDir != agent.Workspace {
		removeWorktree(agent.Workspace, agent.WorktreeDir)
		agent.WorktreeDir = ""
	}

	agent.State = StateStopped
	agent.UpdatedAt = time.Now()

	// Remove from parent's children list
	m.removeFromParent(name)

	if err := m.saveState(); err != nil {
		log.Warn("failed to save agent state", "error", err)
	}

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

	// Stop all children first (depth-first, continue on errors)
	for _, childID := range agent.Children {
		_ = m.stopAgentTreeLocked(childID)
	}

	// Kill this agent's tmux session (ignore error - session might already be dead)
	_ = m.tmux.KillSession(name)

	agent.State = StateStopped
	agent.UpdatedAt = time.Now()
	agent.Children = []string{} // Clear children since they're stopped

	return nil
}

// DeleteOptions configures agent deletion behavior.
type DeleteOptions struct {
	// PurgeMemory removes the memory directory. Default (false) preserves it.
	PurgeMemory bool
}

// DeleteAgent permanently removes an agent from the workspace.
// This stops the agent, removes its worktree, memory directory, and state.
func (m *Manager) DeleteAgent(name string) error {
	return m.DeleteAgentWithOptions(name, DeleteOptions{PurgeMemory: true})
}

// DeleteAgentWithOptions permanently removes an agent with configurable options.
func (m *Manager) DeleteAgentWithOptions(name string, opts DeleteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debug("deleting agent", "name", name, "purgeMemory", opts.PurgeMemory)

	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	// Kill tmux session (ignore error - session might already be dead)
	_ = m.tmux.KillSession(name)

	// Clean up per-agent git worktree
	if agent.WorktreeDir != "" && agent.WorktreeDir != agent.Workspace {
		removeWorktree(agent.Workspace, agent.WorktreeDir)
	}

	// Clean up per-agent memory directory (only if purge requested)
	if opts.PurgeMemory && agent.MemoryDir != "" {
		if err := os.RemoveAll(agent.MemoryDir); err != nil {
			log.Warn("failed to remove memory dir", "dir", agent.MemoryDir, "error", err)
		} else {
			log.Debug("removed memory dir", "dir", agent.MemoryDir)
		}
	}

	// Remove from parent's children list
	m.removeFromParent(name)

	// Delete from state
	delete(m.agents, name)

	if err := m.saveState(); err != nil {
		log.Warn("failed to save agent state", "error", err)
	}

	return nil
}

// RenameAgent renames an agent from oldName to newName.
func (m *Manager) RenameAgent(oldName, newName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debug("renaming agent", "oldName", oldName, "newName", newName)

	agent, exists := m.agents[oldName]
	if !exists {
		return fmt.Errorf("agent %s not found", oldName)
	}

	// Check if new name already exists
	if _, newExists := m.agents[newName]; newExists {
		return fmt.Errorf("agent %s already exists", newName)
	}

	// Update agent name
	agent.Name = newName
	agent.UpdatedAt = time.Now()

	// Update in agents map
	delete(m.agents, oldName)
	m.agents[newName] = agent

	// Update parent's children list if applicable
	if agent.ParentID != "" {
		parent, parentExists := m.agents[agent.ParentID]
		if parentExists {
			for i, child := range parent.Children {
				if child == oldName {
					parent.Children[i] = newName
					break
				}
			}
		}
	}

	// Update children's parent reference (parent ID is unchanged, just log)
	log.Debug("agent renamed", "oldName", oldName, "newName", newName)

	if err := m.saveState(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// StopAll stops all agents.
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, agent := range m.agents {
		_ = m.tmux.KillSession(name) //nolint:errcheck // best-effort cleanup
		agent.State = StateStopped
		agent.UpdatedAt = time.Now()
	}

	if err := m.saveState(); err != nil {
		log.Warn("failed to save agent state", "error", err)
	}
	return nil
}

// GetAgent returns a copy of an agent by name.
// Returns nil if the agent doesn't exist.
func (m *Manager) GetAgent(name string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, exists := m.agents[name]
	if !exists {
		return nil
	}
	// Return a copy to avoid data races
	copy := *a
	copy.Children = append([]string{}, a.Children...)
	return &copy
}

// ListAgents returns copies of all agents sorted by role hierarchy then by name.
// Order: ProductManager/Coordinator → Manager → Engineer/Worker
func (m *Manager) ListAgents() []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return copies to avoid data races
	agents := make([]*Agent, 0, len(m.agents))
	for _, a := range m.agents {
		copy := *a
		copy.Children = append([]string{}, a.Children...)
		agents = append(agents, &copy)
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

// ListChildren returns copies of all direct children of an agent.
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
			// Return copy to avoid data races
			copy := *child
			copy.Children = append([]string{}, child.Children...)
			children = append(children, &copy)
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

// collectDescendants recursively collects copies of all descendants.
func (m *Manager) collectDescendants(parentID string, result *[]*Agent) {
	parent, exists := m.agents[parentID]
	if !exists {
		return
	}

	for _, childID := range parent.Children {
		if child, exists := m.agents[childID]; exists {
			// Return copy to avoid data races
			copy := *child
			copy.Children = append([]string{}, child.Children...)
			*result = append(*result, &copy)
			m.collectDescendants(childID, result)
		}
	}
}

// GetParent returns a copy of the parent agent, or nil if no parent.
func (m *Manager) GetParent(agentID string) *Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	agent, exists := m.agents[agentID]
	if !exists || agent.ParentID == "" {
		return nil
	}

	parent, exists := m.agents[agent.ParentID]
	if !exists {
		return nil
	}
	// Return copy to avoid data races
	copy := *parent
	copy.Children = append([]string{}, parent.Children...)
	return &copy
}

// ListByRole returns copies of all agents with a specific role.
func (m *Manager) ListByRole(role Role) []*Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var agents []*Agent
	for _, a := range m.agents {
		if a.Role == role {
			// Return copy to avoid data races
			copy := *a
			copy.Children = append([]string{}, a.Children...)
			agents = append(agents, &copy)
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

			// Sync state with task symbols:
			// - Spinner symbols (✻ ✳ ✽ ·) → working
			// - Prompt symbol (❯) or pause (⏺) → idle (waiting for input)
			if strings.HasPrefix(live, "✻") ||
				strings.HasPrefix(live, "✳") ||
				strings.HasPrefix(live, "✽") ||
				strings.HasPrefix(live, "·") {
				if a.State != StateWorking {
					a.State = StateWorking
					a.UpdatedAt = time.Now()
				}
			} else if strings.HasPrefix(live, "❯") ||
				strings.HasPrefix(live, "⏺") {
				if a.State == StateWorking {
					a.State = StateIdle
					a.UpdatedAt = time.Now()
				}
			}
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
// Returns an error if the transition is invalid per the state machine.
func (m *Manager) UpdateAgentState(name string, state State, task string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	if err := ValidateTransition(agent.State, state); err != nil {
		return fmt.Errorf("agent %s: %w", name, err)
	}

	agent.State = state
	agent.Task = task
	agent.UpdatedAt = time.Now()

	if err := m.saveState(); err != nil {
		log.Warn("failed to save agent state", "error", err)
	}
	return nil
}

// SetAgentTeam assigns an agent to a team.
func (m *Manager) SetAgentTeam(name, team string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	agent.Team = team
	agent.UpdatedAt = time.Now()

	if err := m.saveState(); err != nil {
		log.Warn("failed to save agent state", "error", err)
	}
	return nil
}

// SendToAgent sends a message/command to an agent's session.
// Sends Enter after the message to submit it.
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

	if err := os.MkdirAll(m.stateDir, 0750); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.agents, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(m.stateDir, "agents.json"), data, 0600)
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

// enforceRootSingleton checks if a root agent can be spawned.
// Returns ErrRootExists if a root already exists and is running.
// Allows respawn if root is stopped or in error state.
func (m *Manager) enforceRootSingleton(workspace string) error {
	bcDir := filepath.Join(workspace, ".bc")
	rootStore := NewRootStateStore(bcDir)

	// Check if root state exists
	rootState, err := rootStore.Load()
	if err != nil {
		if err == ErrRootNotFound {
			// No root exists - allow creation
			return nil
		}
		return fmt.Errorf("failed to check root state: %w", err)
	}

	// Root exists - check if it's in a terminal state that allows respawn
	switch rootState.State {
	case StateStopped, StateError:
		// Terminal state - allow respawn by deleting old state
		log.Debug("root in terminal state, allowing respawn", "state", rootState.State)
		if delErr := rootStore.Delete(); delErr != nil {
			log.Warn("failed to delete old root state", "error", delErr)
		}
		return nil
	default:
		// Root is active (idle, working, stuck, etc.) - deny new spawn
		return fmt.Errorf("%w: existing root is in state %q", ErrRootExists, rootState.State)
	}
}
