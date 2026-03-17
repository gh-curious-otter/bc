// Package agent provides agent lifecycle management for bc.
//
// An agent is an AI assistant running in an isolated tmux session with its own
// git worktree. Agents have roles (engineer, manager, etc.) that determine
// their capabilities and permissions.
//
// # Basic Usage
//
// Create an agent manager:
//
//	mgr := agent.NewWorkspaceManager(".bc/agents", "/path/to/workspace")
//	if err := mgr.LoadState(); err != nil {
//	    log.Warn("failed to load state", "error", err)
//	}
//
// List agents:
//
//	for _, a := range mgr.ListAgents() {
//	    fmt.Printf("%s: %s (%s)\n", a.Name, a.Role, a.State)
//	}
//
// Start an agent:
//
//	ag, err := mgr.Start(ctx, "eng-01", agent.Role("engineer"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Roles and Capabilities
//
// Agents have roles that define their capabilities:
//
//	if agent.HasCapability(agent.Role("engineer"), agent.CapImplementTasks) {
//	    // Engineer can implement tasks
//	}
//
// Check if a role can create another:
//
//	if agent.CanCreateRole(agent.Role("manager"), agent.Role("engineer")) {
//	    // Manager can spawn engineers
//	}
//
// # States
//
// Agents transition through states: Idle -> Working -> Done/Error.
// State transitions are validated:
//
//	if err := agent.ValidateTransition(agent.StateIdle, agent.StateWorking); err != nil {
//	    log.Error("invalid transition", "error", err)
//	}
package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rpuneet/bc/config"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/provider"
	"github.com/rpuneet/bc/pkg/runtime"
	"github.com/rpuneet/bc/pkg/tmux"
	"github.com/rpuneet/bc/pkg/workspace"
)

// IsValidAgentName validates that agent names contain only alphanumeric characters, hyphens, and underscores.
// This ensures agent names are safe for use in file paths, shell environments, and tmux sessions.
func IsValidAgentName(name string) bool {
	if name == "" {
		return false
	}
	for _, c := range name {
		isLower := c >= 'a' && c <= 'z'
		isUpper := c >= 'A' && c <= 'Z'
		isDigit := c >= '0' && c <= '9'
		isAllowed := isLower || isUpper || isDigit || c == '-' || c == '_'
		if !isAllowed {
			return false
		}
	}
	return true
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

// Permission defines RBAC permissions for agent operations.
// Issue #1191: RBAC permissions for agent capabilities
type Permission string

const (
	// Agent lifecycle permissions
	PermCreateAgents  Permission = "can_create_agents"  // Can spawn new agents
	PermStopAgents    Permission = "can_stop_agents"    // Can stop running agents
	PermDeleteAgents  Permission = "can_delete_agents"  // Can permanently delete agents
	PermRestartAgents Permission = "can_restart_agents" // Can restart stopped agents

	// Communication permissions
	PermSendCommands Permission = "can_send_commands" // Can send commands to agents
	PermViewLogs     Permission = "can_view_logs"     // Can view agent logs/output

	// Configuration permissions
	PermModifyConfig Permission = "can_modify_config" // Can modify workspace config
	PermModifyRoles  Permission = "can_modify_roles"  // Can edit role definitions

	// Channel permissions
	PermCreateChannels Permission = "can_create_channels" // Can create new channels
	PermDeleteChannels Permission = "can_delete_channels" // Can delete channels
	PermSendMessages   Permission = "can_send_messages"   // Can send messages to channels
)

// AllPermissions lists all available permissions.
var AllPermissions = []Permission{
	PermCreateAgents,
	PermStopAgents,
	PermDeleteAgents,
	PermRestartAgents,
	PermSendCommands,
	PermViewLogs,
	PermModifyConfig,
	PermModifyRoles,
	PermCreateChannels,
	PermDeleteChannels,
	PermSendMessages,
}

// DefaultPermissions returns default permissions for a role level.
// Higher level roles (root, manager) have more permissions by default.
func DefaultPermissions(roleLevel int) []Permission {
	switch {
	case roleLevel <= -1:
		// Root level - all permissions
		return AllPermissions
	case roleLevel == 0:
		// Manager level
		return []Permission{
			PermCreateAgents,
			PermStopAgents,
			PermRestartAgents,
			PermSendCommands,
			PermViewLogs,
			PermCreateChannels,
			PermSendMessages,
		}
	default:
		// Engineer/worker level
		return []Permission{
			PermViewLogs,
			PermSendCommands,
			PermSendMessages,
		}
	}
}

// CheckPermission verifies an agent has the required permission.
// Returns nil if permitted, error otherwise.
func CheckPermission(permissions []string, required Permission) error {
	requiredStr := string(required)
	for _, p := range permissions {
		if p == requiredStr {
			return nil
		}
	}
	return fmt.Errorf("permission denied: %s required", required)
}

// HasPermissionStr checks if a permission string is in the list.
func HasPermissionStr(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == required {
			return true
		}
	}
	return false
}

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
	UpdatedAt      time.Time    `json:"updated_at"`
	StartedAt      time.Time    `json:"started_at"`
	RolePrompt     *AgentMemory `json:"memory,omitempty"`
	Workspace      string       `json:"workspace"`
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Task           string       `json:"task,omitempty"`
	Session        string       `json:"session"`
	Tool           string       `json:"tool,omitempty"`
	ParentID       string       `json:"parent_id,omitempty"`
	HookedWork     string       `json:"hooked_work,omitempty"`
	WorktreeDir    string       `json:"worktree_dir,omitempty"`
	LogFile        string       `json:"log_file,omitempty"`
	Team           string       `json:"team,omitempty"`
	RecoveredFrom  string       `json:"recovered_from,omitempty"`
	EnvFile        string       `json:"env_file,omitempty"`
	RuntimeBackend string       `json:"runtime_backend,omitempty"`
	LastCrashTime  *time.Time   `json:"last_crash_time,omitempty"`
	Role           Role         `json:"role"`
	State          State        `json:"state"`
	Children       []string     `json:"children,omitempty"`
	CrashCount     int          `json:"crash_count,omitempty"`
	IsRoot         bool         `json:"is_root,omitempty"`
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
	agents           map[string]*Agent
	store            *SQLiteStore               // SQLite-backed agent persistence
	backends         map[string]runtime.Backend // keyed by "tmux", "docker"
	defaultBackend   string                     // "tmux" or "docker"
	providerRegistry *provider.Registry

	stateDir string

	// Agent command (e.g., "claude" or "claude --dangerously-skip-permissions")
	agentCmd string

	// defaultTool is the provider name for the default agentCmd (for BuildCommand)
	defaultTool string

	// Workspace path for env vars
	workspacePath string

	// BootstrapDelay is the time to wait before sending bootstrap prompts.
	// If zero, DefaultBootstrapDelay is used.
	BootstrapDelay time.Duration

	mu sync.RWMutex
}

// runtime returns the default runtime backend.
func (m *Manager) runtime() runtime.Backend {
	return m.backends[m.defaultBackend]
}

// runtimeForAgent returns the appropriate runtime backend for an agent,
// based on the agent's stored RuntimeBackend. Falls back to the default.
func (m *Manager) runtimeForAgent(name string) runtime.Backend {
	if a, ok := m.agents[name]; ok && a.RuntimeBackend != "" {
		if be, ok := m.backends[a.RuntimeBackend]; ok {
			return be
		}
	}
	return m.runtime()
}

// NewManager creates a new agent manager with workspace-scoped tmux sessions.
func NewManager(stateDir string) *Manager {
	cmd, tool := defaultAgentCmd()
	tmuxBe := runtime.NewTmuxBackend(tmux.NewManager(config.Tmux.SessionPrefix))
	return &Manager{
		agents:           make(map[string]*Agent),
		backends:         map[string]runtime.Backend{"tmux": tmuxBe},
		defaultBackend:   "tmux",
		providerRegistry: provider.DefaultRegistry,
		stateDir:         stateDir,
		agentCmd:         cmd,
		defaultTool:      tool,
	}
}

// NewWorkspaceManager creates an agent manager scoped to a workspace.
// Session names will be unique per workspace to avoid collisions.
func NewWorkspaceManager(stateDir, workspacePath string) *Manager {
	cmd, tool := defaultAgentCmd()
	tmuxBe := runtime.NewTmuxBackend(tmux.NewWorkspaceManager(config.Tmux.SessionPrefix, workspacePath))
	return &Manager{
		agents:           make(map[string]*Agent),
		backends:         map[string]runtime.Backend{"tmux": tmuxBe},
		defaultBackend:   "tmux",
		providerRegistry: provider.DefaultRegistry,
		stateDir:         stateDir,
		agentCmd:         cmd,
		defaultTool:      tool,
		workspacePath:    workspacePath,
	}
}

// NewWorkspaceManagerWithRuntime creates an agent manager with a specific runtime backend.
// rtName should be "docker" or "tmux".
func NewWorkspaceManagerWithRuntime(stateDir, workspacePath string, rt runtime.Backend, rtName string) *Manager {
	cmd, tool := defaultAgentCmd()
	bes := map[string]runtime.Backend{rtName: rt}
	// Always register a tmux backend so agents with RuntimeBackend="tmux" work
	if rtName != "tmux" {
		bes["tmux"] = runtime.NewTmuxBackend(tmux.NewWorkspaceManager(config.Tmux.SessionPrefix, workspacePath))
	}
	return &Manager{
		agents:           make(map[string]*Agent),
		backends:         bes,
		defaultBackend:   rtName,
		providerRegistry: provider.DefaultRegistry,
		stateDir:         stateDir,
		agentCmd:         cmd,
		defaultTool:      tool,
		workspacePath:    workspacePath,
	}
}

// defaultAgentCmd returns the command and tool name for the default provider.
func defaultAgentCmd() (string, string) {
	name := config.Providers.Default
	if name == "" {
		return "", ""
	}
	p, ok := provider.DefaultRegistry.Get(name)
	if !ok {
		return "", ""
	}
	return p.Command(), name
}

// getAgentCommand looks up the command for a tool from the manager's provider registry.
func (m *Manager) getAgentCommand(toolName, agentName string, resume bool) (string, bool) {
	if m.providerRegistry != nil {
		if p, ok := m.providerRegistry.Get(toolName); ok {
			wsName := filepath.Base(m.workspacePath)
			return p.BuildCommand(provider.CommandOpts{
				AgentName:     agentName,
				WorkspaceName: wsName,
				Resume:        resume,
			}), true
		}
	}
	return "", false
}

// listAvailableTools returns tool names from the manager's provider registry.
func (m *Manager) listAvailableTools() []string {
	if m.providerRegistry == nil {
		return nil
	}
	providers := m.providerRegistry.List()
	tools := make([]string, 0, len(providers))
	for _, p := range providers {
		tools = append(tools, p.Name())
	}
	return tools
}

// SetAgentCommand sets the command to run for agents.
func (m *Manager) SetAgentCommand(cmd string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agentCmd = cmd
}

// SetAgentByName sets the agent command by looking up the provider name in the registry.
func (m *Manager) SetAgentByName(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.providerRegistry.Get(name)
	if !ok {
		return false
	}
	m.agentCmd = p.Command()
	m.defaultTool = name
	return true
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

// GetAgentCommand returns the command for a tool name from the provider registry.
// Returns the command and true if found, or empty string and false if not.
func GetAgentCommand(toolName string) (string, bool) {
	p, ok := provider.DefaultRegistry.Get(toolName)
	if !ok {
		return "", false
	}
	return p.Command(), true
}

// GetAgentCommandFromConfig returns the command for a tool name,
// checking workspace ProvidersConfig first, then falling back to global config.
// This enables per-workspace tool customization.
func GetAgentCommandFromConfig(toolName string, wsCfg *workspace.Config) (string, bool) {
	// Check workspace ProvidersConfig first
	if wsCfg != nil {
		if p := wsCfg.GetProvider(toolName); p != nil && p.Command != "" {
			return p.Command, true
		}
	}
	// Fall back to global config
	return GetAgentCommand(toolName)
}

// ListAvailableTools returns a list of configured tool names from the provider registry.
func ListAvailableTools() []string {
	providers := provider.DefaultRegistry.List()
	tools := make([]string, 0, len(providers))
	for _, p := range providers {
		tools = append(tools, p.Name())
	}
	return tools
}

// SpawnOptions holds all parameters for creating an agent.
type SpawnOptions struct {
	Name      string
	Role      Role
	Workspace string
	ParentID  string
	Tool      string
	EnvFile   string
	Runtime   string // override runtime backend ("tmux" or "docker"); empty uses manager default
}

// SpawnAgent creates and starts a new agent.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgent(name string, role Role, workspace string) (*Agent, error) {
	return m.SpawnAgentWithOptions(SpawnOptions{Name: name, Role: role, Workspace: workspace})
}

// SpawnAgentWithTool creates and starts a new agent with a specific tool.
// If tool is empty, uses the manager's default agent command.
func (m *Manager) SpawnAgentWithTool(name string, role Role, workspace string, tool string) (*Agent, error) {
	return m.SpawnAgentWithOptions(SpawnOptions{Name: name, Role: role, Workspace: workspace, Tool: tool})
}

// SpawnAgentWithParent creates and starts a new agent with a parent relationship.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgentWithParent(name string, role Role, workspace string, parentID string) (*Agent, error) {
	return m.SpawnAgentWithOptions(SpawnOptions{Name: name, Role: role, Workspace: workspace, ParentID: parentID})
}

// SpawnAgentWithOptions creates and starts a new agent with all options.
// If tool is empty, uses the manager's default agent command.
// Idempotent: if the agent already exists and its tmux session is alive, reuse it.
func (m *Manager) SpawnAgentWithOptions(opts SpawnOptions) (*Agent, error) {
	name := opts.Name
	role := opts.Role
	wsPath := opts.Workspace
	parentID := opts.ParentID
	tool := opts.Tool

	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debug("spawning agent", "name", name, "role", role, "workspace", wsPath, "parentID", parentID, "tool", tool)

	// Validate agent name format
	if !IsValidAgentName(name) {
		return nil, fmt.Errorf("agent name %q contains invalid characters (use letters, numbers, dash, underscore)", name)
	}

	// Validate role is not empty or null-like
	if role == "" || role == "null" || role == "<nil>" {
		return nil, fmt.Errorf("role is required and cannot be empty or null")
	}

	// Enforce root singleton constraint
	if role == RoleRoot {
		if err := m.enforceRootSingleton(wsPath); err != nil {
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
		if m.runtimeForAgent(name).HasSession(context.TODO(), name) {
			existing.UpdatedAt = time.Now()
			if err := m.saveState(); err != nil {
				log.Warn("failed to save agent state", "error", err)
			}
			return existing, nil
		}
		// Agent exists but session is dead — always resume with --continue
		// to pick up the previous Claude conversation.
		// Update runtime backend if overridden
		if opts.Runtime != "" {
			existing.RuntimeBackend = opts.Runtime
		}
		toolName := existing.Tool
		if toolName == "" {
			toolName = m.defaultTool
		}
		agentCmd := m.agentCmd
		if toolName != "" {
			if cmd, ok := m.getAgentCommand(toolName, name, true); ok {
				agentCmd = cmd
			}
		}

		// Apply provider session customization for container backends only.
		// Claude's --tmux flag is only needed inside Docker, not native tmux.
		if existing.RuntimeBackend != "tmux" {
			if toolName != "" && m.providerRegistry != nil {
				if p, ok := m.providerRegistry.Get(toolName); ok {
					if sc, ok := p.(provider.SessionCustomizer); ok {
						agentCmd = sc.AdjustSessionCommand(agentCmd)
					}
				}
			}
		}

		env := map[string]string{
			"BC_AGENT_ID":   name,
			"BC_AGENT_ROLE": string(existing.Role),
			"BC_WORKSPACE":  wsPath,
		}
		if toolName != "" {
			env["BC_AGENT_TOOL"] = toolName
		}
		if existing.ParentID != "" {
			env["BC_PARENT_ID"] = existing.ParentID
		}
		injectEnv(env, wsPath, toolName, existing.EnvFile)
		if err := m.runtimeForAgent(name).CreateSessionWithEnv(context.TODO(), name, wsPath, agentCmd, env); err != nil {
			return nil, fmt.Errorf("failed to recreate tmux session: %w", err)
		}

		// Resume log streaming if log file was set
		if existing.LogFile != "" {
			truncateLogFile(existing.LogFile, config.Logs.MaxBytes)
			if pipeErr := m.runtimeForAgent(name).PipePane(context.TODO(), name, existing.LogFile); pipeErr != nil {
				log.Warn("failed to resume pipe-pane", "agent", name, "error", pipeErr)
			}
		} else {
			// Set up new log pipe for agents that didn't have one
			existing.LogFile = m.setupLogPipe(name, wsPath)
		}

		// Only inject bootstrap prompt for non-terminal states (active respawn)
		// For stopped/error agents, --continue handles resumption
		if existing.State != StateStopped && existing.State != StateError {
			go m.sendRespawnBootstrap(name, existing, wsPath)
		} else {
			// Terminal state — reset to starting
			existing.State = StateStarting
		}

		existing.UpdatedAt = time.Now()
		if err := m.saveState(); err != nil {
			log.Warn("failed to save agent state", "error", err)
		}
		return existing, nil
	}

	// If a session exists from a previous crash, kill it in all backends
	for beName, be := range m.backends {
		if be.HasSession(context.TODO(), name) {
			log.Debug("killing stale session", "session", name, "backend", beName)
			if err := be.KillSession(context.TODO(), name); err != nil {
				log.Warn("failed to kill existing session", "session", name, "backend", beName, "error", err)
			}
		}
	}

	// Determine the command to use (fresh create — no resume)
	agentCmd := m.agentCmd
	if tool != "" {
		if cmd, ok := m.getAgentCommand(tool, name, false); ok {
			agentCmd = cmd
		} else {
			return nil, fmt.Errorf("unknown tool %q, available tools: %v", tool, m.listAvailableTools())
		}
	} else if m.defaultTool != "" {
		if cmd, ok := m.getAgentCommand(m.defaultTool, name, false); ok {
			agentCmd = cmd
		}
	}

	// Determine runtime backend for this agent
	agentRuntime := m.defaultBackend
	if opts.Runtime != "" {
		agentRuntime = opts.Runtime
	}

	// Apply provider session customization for container backends only.
	// Claude's --tmux flag is needed inside Docker (tmux runs inside the container)
	// but NOT for native tmux sessions (claude detects tmux automatically).
	if agentRuntime != "tmux" {
		sessionTool := tool
		if sessionTool == "" {
			sessionTool = m.defaultTool
		}
		if sessionTool != "" && m.providerRegistry != nil {
			if p, ok := m.providerRegistry.Get(sessionTool); ok {
				if sc, ok := p.(provider.SessionCustomizer); ok {
					agentCmd = sc.AdjustSessionCommand(agentCmd)
				}
			}
		}
	}

	// Validate tool binary exists before spawning.
	// Use provider registry for known tools (richer validation + version logging),
	// fall back to PATH check for custom/unknown tools.
	providerValidated := false
	if tool != "" && m.providerRegistry != nil {
		if p, ok := m.providerRegistry.Get(tool); ok {
			ctx := context.TODO()
			if !p.IsInstalled(ctx) {
				return nil, fmt.Errorf("tool %q is not installed. Install %s or configure a different tool in config.toml", tool, p.Name())
			}
			if v := p.Version(ctx); v != "" {
				log.Debug("provider validated", "tool", tool, "version", v)
			}
			providerValidated = true
		}
	}

	if !providerValidated && agentCmd != "" {
		parts := strings.Fields(agentCmd)
		if len(parts) > 0 {
			if _, err := exec.LookPath(parts[0]); err != nil {
				return nil, fmt.Errorf("tool %q command %q not found in PATH. Install it or configure a different tool in config.toml", tool, parts[0])
			}
		}
	}
	log.Debug("agent runtime selected", "agent", name, "runtime", agentRuntime, "default", m.defaultBackend, "override", opts.Runtime)

	// Create agent
	agent := &Agent{
		ID:             name,
		Name:           name,
		Role:           role,
		State:          StateStarting,
		Workspace:      wsPath,
		Session:        name,
		Tool:           tool,
		ParentID:       parentID,
		EnvFile:        opts.EnvFile,
		RuntimeBackend: agentRuntime,
		Children:       []string{},
		IsRoot:         role == RoleRoot,
		StartedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Register agent early so runtimeForAgent can resolve the correct backend
	m.agents[name] = agent

	// Build env vars so the spawned process sees them immediately
	env := map[string]string{
		"BC_AGENT_ID":   name,
		"BC_AGENT_ROLE": string(role),
		"BC_WORKSPACE":  wsPath,
	}
	effectiveTool := tool
	if effectiveTool == "" {
		effectiveTool = m.defaultTool
	}
	if effectiveTool != "" {
		env["BC_AGENT_TOOL"] = effectiveTool
	}
	if parentID != "" {
		env["BC_PARENT_ID"] = parentID
	}
	injectEnv(env, wsPath, effectiveTool, opts.EnvFile)

	// Create session in the workspace directory using the agent's runtime backend
	if err := m.runtimeForAgent(name).CreateSessionWithEnv(context.TODO(), name, wsPath, agentCmd, env); err != nil {
		delete(m.agents, name) // clean up early registration
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Start log streaming via pipe-pane
	agent.LogFile = m.setupLogPipe(name, wsPath)

	// Load role prompt from prompts/<role>.md
	agent.RolePrompt = LoadRoleMemory(wsPath, role)

	// Update state
	agent.State = StateIdle
	agent.UpdatedAt = time.Now()

	// Build bootstrap prompt with role prompt
	var promptParts []string

	// Add role prompt if available
	if agent.RolePrompt != nil && agent.RolePrompt.RolePrompt != "" {
		promptParts = append(promptParts, agent.RolePrompt.RolePrompt)
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

	// Send bootstrap prompt asynchronously (like respawn path at line 656)
	// to avoid holding m.mu during the blocking sleep.
	if len(promptParts) > 0 {
		bootstrapName := name
		bootstrapWorkspace := wsPath
		bootstrapParts := make([]string, len(promptParts))
		copy(bootstrapParts, promptParts)
		bootstrapDelay := m.getBootstrapDelay()

		go func() {
			time.Sleep(bootstrapDelay)
			prompt := strings.Join(bootstrapParts, "\n\n---\n\n")
			prompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nAgent ID: %s\n", bootstrapWorkspace, bootstrapName)
			if err := m.runtimeForAgent(name).SendKeys(context.TODO(), bootstrapName, prompt); err != nil {
				log.Warn("failed to send bootstrap prompt", "agent", bootstrapName, "error", err)
			}
		}()
	}

	return agent, nil
}

// sendRespawnBootstrap sends the role prompt to a respawned agent.
// Runs in a goroutine since it needs to wait for the agent to initialize.
func (m *Manager) sendRespawnBootstrap(name string, agent *Agent, workspace string) {
	time.Sleep(m.getBootstrapDelay())

	var promptParts []string

	// Load role prompt from role files
	if agent.RolePrompt != nil && agent.RolePrompt.RolePrompt != "" {
		promptParts = append(promptParts, agent.RolePrompt.RolePrompt)
	}

	// Fall back to loading from role manager
	if len(promptParts) == 0 {
		if mem := LoadRoleMemory(workspace, agent.Role); mem != nil && mem.RolePrompt != "" {
			promptParts = append(promptParts, mem.RolePrompt)
		}
	}

	if len(promptParts) > 0 {
		prompt := strings.Join(promptParts, "\n\n---\n\n")
		prompt += fmt.Sprintf("\n\n---\n\nWorkspace: %s\nAgent ID: %s\n(Respawned session — continuing previous work)\n", workspace, name)
		if err := m.runtimeForAgent(name).SendKeys(context.TODO(), name, prompt); err != nil {
			log.Warn("failed to send respawn bootstrap", "agent", name, "error", err)
		}
	}
}

// setupLogPipe creates the logs directory and starts pipe-pane for the agent.
// Returns the log file path.
func (m *Manager) setupLogPipe(name, workspace string) string {
	logsDir := filepath.Join(workspace, ".bc", "logs")
	if err := os.MkdirAll(logsDir, 0750); err != nil {
		log.Warn("failed to create logs dir", "error", err)
		return ""
	}

	logPath := filepath.Join(logsDir, name+".log")

	// Truncate if over max size
	truncateLogFile(logPath, config.Logs.MaxBytes)

	if err := m.runtimeForAgent(name).PipePane(context.TODO(), name, logPath); err != nil {
		log.Warn("failed to start pipe-pane", "agent", name, "error", err)
		return ""
	}

	log.Debug("started log streaming", "agent", name, "path", logPath)
	return logPath
}

// truncateLogFile truncates a log file if it exceeds maxBytes.
// Keeps the last half of the file to preserve recent output.
func truncateLogFile(path string, maxBytes int64) {
	if maxBytes <= 0 {
		return
	}

	info, err := os.Stat(path)
	if err != nil || info.Size() <= maxBytes {
		return
	}

	data, err := os.ReadFile(path) //nolint:gosec // path constructed from trusted workspace root
	if err != nil {
		log.Warn("failed to read log for truncation", "path", path, "error", err)
		return
	}

	// Keep last half
	half := len(data) / 2
	// Find next newline to avoid cutting mid-line
	for half < len(data) && data[half] != '\n' {
		half++
	}
	if half < len(data) {
		half++ // skip the newline
	}

	if err := os.WriteFile(path, data[half:], 0600); err != nil { //nolint:gosec // path constructed from trusted workspace root
		log.Warn("failed to truncate log", "path", path, "error", err)
	}
}

// SpawnChildAgent creates a child agent under a parent agent.
// Validates that the parent has permission to create the child role.
func (m *Manager) SpawnChildAgent(parentID, childName string, childRole Role, workspace string) (*Agent, error) {
	return m.SpawnAgentWithOptions(SpawnOptions{Name: childName, Role: childRole, Workspace: workspace, ParentID: parentID})
}

// SpawnChildAgentWithTool creates a child agent under a parent agent with a specific tool.
// Validates that the parent has permission to create the child role.
func (m *Manager) SpawnChildAgentWithTool(parentID, childName string, childRole Role, workspace, tool string) (*Agent, error) {
	return m.SpawnAgentWithOptions(SpawnOptions{Name: childName, Role: childRole, Workspace: workspace, ParentID: parentID, Tool: tool})
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
	_ = m.runtimeForAgent(name).KillSession(context.TODO(), name)

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
	_ = m.runtimeForAgent(name).KillSession(context.TODO(), name)

	agent.State = StateStopped
	agent.UpdatedAt = time.Now()
	agent.Children = []string{} // Clear children since they're stopped

	return nil
}

// DeleteOptions configures agent deletion behavior.
type DeleteOptions struct {
	// Placeholder for future options.
	Force bool
}

// DeleteAgent permanently removes an agent from the workspace.
func (m *Manager) DeleteAgent(name string) error {
	return m.DeleteAgentWithOptions(name, DeleteOptions{})
}

// DeleteAgentWithOptions permanently removes an agent with configurable options.
func (m *Manager) DeleteAgentWithOptions(name string, opts DeleteOptions) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debug("deleting agent", "name", name)

	_, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}

	// Kill tmux session (ignore error - session might already be dead)
	_ = m.runtimeForAgent(name).KillSession(context.TODO(), name)

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
		_ = m.runtimeForAgent(name).KillSession(context.TODO(), name) //nolint:errcheck // best-effort cleanup
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

	// List sessions from all backends to support mixed runtimes
	active := make(map[string]bool)
	for _, be := range m.backends {
		sessions, err := be.ListSessions(context.TODO())
		if err != nil {
			continue // backend may be unavailable
		}
		for _, s := range sessions {
			active[s.Name] = true
		}
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

			// Use provider-based state detection if available for richer state inference
			newState := m.detectAgentState(a.Tool, live)

			// IMPORTANT: Preserve error, stuck, and done states - these are explicitly
			// reported by agents and should not be overwritten by activity detection.
			// Only toggle between working and idle for agents in "normal" states.
			switch newState {
			case StateWorking:
				if a.State == StateIdle || a.State == StateStarting {
					a.State = StateWorking
					a.UpdatedAt = time.Now()
				}
			case StateIdle:
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
// detectAgentState determines agent state from output, using provider-based
// detection when available or falling back to symbol-based heuristics.
func (m *Manager) detectAgentState(tool, output string) State {
	// Try provider-based detection if tool is known
	if tool != "" && m.providerRegistry != nil {
		if p, ok := m.providerRegistry.Get(tool); ok {
			ps := p.DetectState(output)
			switch ps {
			case provider.StateWorking:
				return StateWorking
			case provider.StateIdle:
				return StateIdle
			case provider.StateDone:
				return StateDone
			case provider.StateError:
				return StateError
			case provider.StateStuck:
				return StateStuck
			}
			// provider.StateUnknown — fall through to symbol detection
		}
	}

	// Fall back to symbol-based detection (works for all tools)
	if strings.HasPrefix(output, "✻") ||
		strings.HasPrefix(output, "✳") ||
		strings.HasPrefix(output, "✽") ||
		strings.HasPrefix(output, "·") {
		return StateWorking
	}
	if strings.HasPrefix(output, "❯") ||
		strings.HasPrefix(output, "⏺") {
		return StateIdle
	}

	return ""
}

func (m *Manager) captureLiveTask(name string) string {
	output, err := m.runtimeForAgent(name).Capture(context.TODO(), name, 15)
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
	m.mu.RLock()
	be := m.runtimeForAgent(name)
	m.mu.RUnlock()
	return be.SendKeys(context.TODO(), name, message)
}

// CaptureOutput captures recent output from an agent's session.
// Reads from the agent's log file first (includes full history with ANSI).
// Falls back to tmux capture-pane if log file is not available.
func (m *Manager) CaptureOutput(name string, lines int) (string, error) {
	m.mu.RLock()
	agent := m.agents[name]
	m.mu.RUnlock()

	// Try log file first
	if agent != nil && agent.LogFile != "" {
		output, err := tailFile(agent.LogFile, lines)
		if err == nil && output != "" {
			return output, nil
		}
		log.Debug("log file read failed, falling back to capture-pane", "agent", name, "error", err)
	}

	// Fall back to tmux capture-pane
	return m.runtimeForAgent(name).Capture(context.TODO(), name, lines)
}

// tailFile reads the last N lines from a file.
func tailFile(path string, lines int) (string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path from trusted agent state
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "", nil
	}

	// Find last N lines by scanning backward
	count := 0
	pos := len(data) - 1
	// Skip trailing newline
	if pos >= 0 && data[pos] == '\n' {
		pos--
	}
	for pos >= 0 {
		if data[pos] == '\n' {
			count++
			if count >= lines {
				pos++
				break
			}
		}
		pos--
	}
	if pos < 0 {
		pos = 0
	}

	return string(data[pos:]), nil
}

// FollowOutput streams new log lines in real-time, like tail -f.
// It prints the last N lines first, then polls for new content every 200ms.
// Blocks until the context is canceled.
// Falls back to a one-shot CaptureOutput if no log file exists.
func (m *Manager) FollowOutput(ctx context.Context, name string, lines int, w io.Writer) error {
	m.mu.RLock()
	a := m.agents[name]
	m.mu.RUnlock()

	if a == nil {
		return fmt.Errorf("agent %q not found", name)
	}

	// No log file — fall back to one-shot capture
	if a.LogFile == "" {
		output, err := m.CaptureOutput(name, lines)
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, output)
		return err
	}

	f, err := os.Open(a.LogFile) //nolint:gosec // path from trusted agent state
	if err != nil {
		// Log file doesn't exist yet — fall back to one-shot
		output, captureErr := m.CaptureOutput(name, lines)
		if captureErr != nil {
			return captureErr
		}
		_, captureErr = io.WriteString(w, output)
		return captureErr
	}
	defer func() { _ = f.Close() }()

	// Print last N lines to start
	initial, tailErr := tailFile(a.LogFile, lines)
	if tailErr == nil && initial != "" {
		if _, writeErr := io.WriteString(w, initial); writeErr != nil {
			return writeErr
		}
	}

	// Seek to end for follow mode
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("seek failed: %w", err)
	}

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			n, readErr := f.ReadAt(buf, offset)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					return writeErr
				}
				offset += int64(n)
			}
			if readErr != nil && readErr != io.EOF {
				return fmt.Errorf("read failed: %w", readErr)
			}
		}
	}
}

// AttachToAgent returns the command to attach to an agent's session.
func (m *Manager) AttachToAgent(name string) error {
	m.mu.RLock()
	be := m.runtimeForAgent(name)
	m.mu.RUnlock()
	cmd := be.AttachCmd(context.TODO(), name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// saveState persists agent state to SQLite.
// SQLite with WAL mode handles concurrency natively — no file locks needed.
// Must be called while holding m.mu.
func (m *Manager) saveState() error {
	if m.store == nil {
		return nil
	}
	return m.store.SaveAll(m.agents)
}

// LoadState loads agent state from SQLite.
// On first run after upgrade, migrates JSON files to SQLite automatically.
func (m *Manager) LoadState() error {
	if m.stateDir == "" {
		return nil
	}

	// Open SQLite store (state.db lives alongside agents dir)
	dbPath := filepath.Join(m.stateDir, "state.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("open agent store: %w", err)
	}
	m.store = store

	// Auto-migrate JSON files if they exist
	if needsMigration(m.stateDir) {
		log.Info("migrating agent state from JSON to SQLite")
		if migErr := migrateJSONToSQLite(store, m.stateDir, m.workspacePath); migErr != nil {
			log.Warn("migration had errors", "error", migErr)
		}
	}

	// Load all agents from SQLite
	agents, err := store.LoadAll()
	if err != nil {
		return fmt.Errorf("load agents: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents = agents
	return nil
}

// Runtime returns the default runtime backend for session management.
func (m *Manager) Runtime() runtime.Backend {
	return m.runtime()
}

// RuntimeForAgent returns the runtime backend for a specific agent.
func (m *Manager) RuntimeForAgent(name string) runtime.Backend {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.runtimeForAgent(name)
}

// Tmux returns the underlying tmux manager if the default backend is tmux.
// Deprecated: Use Runtime() instead. This is kept for backward compatibility.
func (m *Manager) Tmux() *tmux.Manager {
	if tb, ok := m.runtime().(*runtime.TmuxBackend); ok {
		return tb.TmuxManager()
	}
	return nil
}

// Close closes the SQLite store. Call when done with the manager.
func (m *Manager) Close() error {
	if m.store != nil {
		return m.store.Close()
	}
	return nil
}

// enforceRootSingleton checks if a root agent can be spawned.
// Returns an error if a root already exists and is running.
// Allows respawn if root is stopped or in error state.
func (m *Manager) enforceRootSingleton(_ string) error {
	// Check in-memory state for existing root
	for _, a := range m.agents {
		if a.IsRoot {
			switch a.State {
			case StateStopped, StateError:
				// Terminal state - allow respawn
				log.Debug("root in terminal state, allowing respawn", "state", a.State)
				return nil
			default:
				// Root is active - deny new spawn
				return fmt.Errorf("root agent already exists and is in state %q", a.State)
			}
		}
	}
	return nil
}

// injectEnv merges layered environment variables into the env map.
// Priority (highest wins): agent env file > provider env > workspace env.
func injectEnv(env map[string]string, workspacePath, toolName, envFile string) {
	// 1. Workspace [env] (lowest priority)
	ws, err := workspace.Load(workspacePath)
	if err == nil && ws.Config != nil {
		for k, v := range ws.Config.Env {
			env[k] = v
		}
		// 2. Provider-specific env
		if toolName != "" {
			if p := ws.Config.GetProvider(toolName); p != nil {
				for k, v := range p.Env {
					env[k] = v
				}
			}
		}
	}
	// 3. Agent env file (highest priority)
	if envFile != "" {
		parseEnvFile(env, envFile)
	}
}

// parseEnvFile reads KEY=VALUE lines from a file and merges them into env.
// Lines starting with # and blank lines are skipped.
func parseEnvFile(env map[string]string, path string) {
	data, err := os.ReadFile(path) //nolint:gosec // path provided by caller
	if err != nil {
		log.Warn("failed to read env file", "path", path, "error", err)
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		env[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
}
