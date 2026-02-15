package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/names"
	"github.com/rpuneet/bc/pkg/team"
)

// agentCmd is the parent command for all agent operations
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage bc agents",
	Long: `Manage bc agent lifecycle: create, list, attach, peek, stop, send.

Examples:
  bc agent list                          # List all agents
  bc agent create eng-01 --role engineer # Create new agent
  bc agent attach eng-01                 # Attach to agent session
  bc agent peek eng-01                   # View recent output
  bc agent send eng-01 "run tests"       # Send message to agent
  bc agent stop eng-01                   # Stop agent
  bc agent broadcast "check status"      # Send to all agents
  bc agent send-to-role engineer "test"  # Send to all engineers
  bc agent send-pattern "eng-*" "hello"  # Send to matching agents`,
}

// agentCreateCmd creates a new agent (replaces bc spawn)
var agentCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new agent",
	Long: `Create and start a new agent.

If no name is provided, a random memorable name is generated (e.g., swift-falcon).

Examples:
  bc agent create --role engineer              # Create with random name
  bc agent create worker-01                    # Create with explicit name
  bc agent create eng-01 --role engineer       # Create engineer
  bc agent create qa-01 --role qa --tool cursor # Create QA with Cursor`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentCreate,
}

// agentListCmd lists all agents (enhanced bc status)
var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agents",
	Long: `List all agents with their status, role, and current task.

Examples:
  bc agent list          # List all agents
  bc agent list --json   # Output as JSON
  bc agent list --role engineer  # Filter by role`,
	RunE: runAgentList,
}

// agentAttachCmd attaches to an agent session (replaces bc attach)
var agentAttachCmd = &cobra.Command{
	Use:   "attach <agent>",
	Short: "Attach to an agent's session",
	Long: `Attach to an agent's tmux session for direct interaction.

Use Ctrl+b d to detach and return to your shell.

Examples:
  bc agent attach eng-01   # Attach to eng-01`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentAttach,
}

// agentPeekCmd shows recent output from an agent
var agentPeekCmd = &cobra.Command{
	Use:   "peek <agent>",
	Short: "Show recent output from an agent",
	Long: `Capture and display recent output from an agent's session.

Examples:
  bc agent peek eng-01          # Show last 50 lines
  bc agent peek eng-01 --lines 100  # Show last 100 lines`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentPeek,
}

// agentShowCmd shows detailed information about an agent
var agentShowCmd = &cobra.Command{
	Use:   "show <agent>",
	Short: "Show agent details",
	Long: `Show detailed information about an agent.

Examples:
  bc agent show eng-01       # Show eng-01 details
  bc agent show eng-01 --json  # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentShow,
}

// agentStartCmd starts a stopped agent (resurrects from saved state)
var agentStartCmd = &cobra.Command{
	Use:   "start <agent>",
	Short: "Start a stopped agent",
	Long: `Start a previously stopped agent from its saved state.

This resurrects the agent's tmux session, git worktree, and memory.
The agent must have been previously created and stopped.

Examples:
  bc agent start eng-01       # Start stopped agent eng-01`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentStart,
}

// agentStopCmd stops a single agent (different from bc down which stops all)
var agentStopCmd = &cobra.Command{
	Use:   "stop <agent>",
	Short: "Stop an agent",
	Long: `Stop a specific agent and its tmux session.

Examples:
  bc agent stop eng-01       # Stop eng-01
  bc agent stop eng-01 --force  # Force stop`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentStop,
}

// agentSendCmd sends a message to an agent (replaces bc send)
var agentSendCmd = &cobra.Command{
	Use:   "send <agent> <message>",
	Short: "Send a message to an agent",
	Long: `Send a message or command to an agent's session.

Examples:
  bc agent send eng-01 "run the tests"
  bc agent send coordinator "check status"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runAgentSend,
}

// agentDeleteCmd permanently removes an agent
var agentDeleteCmd = &cobra.Command{
	Use:   "delete <agent>",
	Short: "Permanently delete an agent",
	Long: `Permanently delete an agent from the workspace.

This removes the agent's tmux session, git worktree, channel memberships,
and agent state. Memory is preserved by default for recovery.

Use --force to delete a running agent without stopping it first.
Use --purge to also delete the agent's memory directory.

Examples:
  bc agent delete eng-01              # Delete (preserves memory)
  bc agent delete eng-01 --force      # Force delete running agent
  bc agent delete eng-01 --purge      # Delete including memory
  bc agent delete eng-01 --force --purge  # Force delete with full cleanup`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentDelete,
}

// agentRenameCmd renames an agent
var agentRenameCmd = &cobra.Command{
	Use:   "rename <old-name> <new-name>",
	Short: "Rename an agent",
	Long: `Rename an agent to a new name.

This updates the agent's name, channel memberships, and worktree directory.
By default, running agents cannot be renamed (use --force to override).

Examples:
  bc agent rename eng-01 engineer-01
  bc agent rename eng-01 eng-02 --force  # Rename running agent`,
	Args: cobra.ExactArgs(2),
	RunE: runAgentRename,
}

// agentHealthCmd shows agent health status
var agentHealthCmd = &cobra.Command{
	Use:   "health [agent]",
	Short: "Show agent health status",
	Long: `Show health status for agents, including tmux session status and state freshness.

An agent is considered:
  - healthy:   tmux session alive and state updated within timeout threshold
  - degraded:  tmux session alive but state is stale (not updated within threshold)
  - unhealthy: tmux session not found or agent in error state
  - stuck:     no activity, repeated failures, or work timeout (with --detect-stuck)

Stuck detection criteria (enabled with --detect-stuck):
  - No activity: no events within activity timeout period
  - Repeated failures: same task failed multiple times
  - Work timeout: work started but not completed within work timeout

Use --alert to send notifications to a channel when stuck agents are detected.

Examples:
  bc agent health                    # Show health for all agents
  bc agent health eng-01             # Show health for specific agent
  bc agent health --json             # Output as JSON
  bc agent health --timeout 2m       # Use 2 minute stale threshold
  bc agent health --detect-stuck     # Include stuck detection analysis
  bc agent health --detect-stuck --work-timeout 1h  # Custom work timeout
  bc agent health --detect-stuck --alert engineering  # Alert channel on stuck`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAgentHealth,
}

// agentBroadcastCmd sends a message to all running agents
var agentBroadcastCmd = &cobra.Command{
	Use:   "broadcast <message>",
	Short: "Send a message to all running agents",
	Long: `Broadcast a message to all running agents in the workspace.

Examples:
  bc agent broadcast "run tests"
  bc agent broadcast "check status"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAgentBroadcast,
}

// agentSendRoleCmd sends a message to all agents of a specific role
var agentSendRoleCmd = &cobra.Command{
	Use:   "send-to-role <role> <message>",
	Short: "Send a message to all agents of a specific role",
	Long: `Send a message to all running agents that have the specified role.

Examples:
  bc agent send-to-role engineer "run the tests"
  bc agent send-to-role manager "check status"
  bc agent send-to-role tech-lead "review PRs"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runAgentSendRole,
}

// agentSendPatternCmd sends a message to agents matching a pattern
var agentSendPatternCmd = &cobra.Command{
	Use:   "send-pattern <pattern> <message>",
	Short: "Send a message to agents matching a pattern",
	Long: `Send a message to all running agents whose names match the given pattern.

Pattern uses glob-style matching (* matches any characters).

Examples:
  bc agent send-pattern "engineer-*" "run tests"
  bc agent send-pattern "eng-0*" "check status"
  bc agent send-pattern "*-lead" "review PRs"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runAgentSendPattern,
}

// Flags
var (
	agentCreateTool      string
	agentCreateRole      string
	agentCreateParent    string
	agentCreateTeam      string
	agentListRole        string
	agentListJSON        bool
	agentShowJSON        bool
	agentPeekLines       int
	agentStopForce       bool
	agentDeleteForce     bool
	agentDeletePurge     bool
	agentRenameForce     bool
	agentHealthJSON      bool
	agentHealthTimeout   string
	agentHealthDetect    bool
	agentHealthWorkTmout string
	agentHealthMaxFail   int
	agentHealthAlert     string
)

func init() {
	// Create flags
	agentCreateCmd.Flags().StringVar(&agentCreateTool, "tool", "", "Agent tool (claude, cursor, codex)")
	agentCreateCmd.Flags().StringVar(&agentCreateRole, "role", "null", "Agent role (null, engineer, manager, product-manager, tech-lead, qa). Use 'bc role --help' to create custom roles")
	agentCreateCmd.Flags().StringVar(&agentCreateParent, "parent", "", "Parent agent ID (must have permission to create this role)")
	agentCreateCmd.Flags().StringVar(&agentCreateTeam, "team", "", "Team name (alphanumeric)")

	// List flags
	agentListCmd.Flags().StringVar(&agentListRole, "role", "", "Filter by role")
	agentListCmd.Flags().BoolVar(&agentListJSON, "json", false, "Output as JSON")

	// Show flags
	agentShowCmd.Flags().BoolVar(&agentShowJSON, "json", false, "Output as JSON")

	// Peek flags
	agentPeekCmd.Flags().IntVar(&agentPeekLines, "lines", 50, "Number of lines to show")

	// Stop flags
	agentStopCmd.Flags().BoolVar(&agentStopForce, "force", false, "Force stop without cleanup")

	// Delete flags
	agentDeleteCmd.Flags().BoolVar(&agentDeleteForce, "force", false, "Force delete running agent without stopping first")
	agentDeleteCmd.Flags().BoolVar(&agentDeletePurge, "purge", false, "Also delete agent's memory directory")

	// Rename flags
	agentRenameCmd.Flags().BoolVar(&agentRenameForce, "force", false, "Rename even if agent is running")

	// Health flags
	agentHealthCmd.Flags().BoolVar(&agentHealthJSON, "json", false, "Output as JSON")
	agentHealthCmd.Flags().StringVar(&agentHealthTimeout, "timeout", "60s", "Stale state threshold (e.g., 30s, 2m)")
	agentHealthCmd.Flags().BoolVar(&agentHealthDetect, "detect-stuck", false, "Enable stuck detection analysis")
	agentHealthCmd.Flags().StringVar(&agentHealthWorkTmout, "work-timeout", "30m", "Work timeout for stuck detection (e.g., 30m, 1h)")
	agentHealthCmd.Flags().IntVar(&agentHealthMaxFail, "max-failures", 3, "Max consecutive failures before considered stuck")
	agentHealthCmd.Flags().StringVar(&agentHealthAlert, "alert", "", "Send alert to channel when stuck agents detected (requires --detect-stuck)")

	// Add subcommands
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentAttachCmd)
	agentCmd.AddCommand(agentPeekCmd)
	agentCmd.AddCommand(agentShowCmd)
	agentCmd.AddCommand(agentStartCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentSendCmd)
	agentCmd.AddCommand(agentDeleteCmd)
	agentCmd.AddCommand(agentRenameCmd)
	agentCmd.AddCommand(agentHealthCmd)
	agentCmd.AddCommand(agentBroadcastCmd)
	agentCmd.AddCommand(agentSendRoleCmd)
	agentCmd.AddCommand(agentSendPatternCmd)

	// Add parent command to root
	rootCmd.AddCommand(agentCmd)
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Determine agent name: use provided name or generate one
	var agentName string
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		agentName = strings.TrimSpace(args[0])
		// Validate agent name doesn't contain shell metacharacters
		if !isValidAgentName(agentName) {
			return fmt.Errorf("agent name %q contains invalid characters (use letters, numbers, dash, underscore)", agentName)
		}
	} else {
		// Generate unique name
		existingAgents := mgr.ListAgents()
		existingNames := make([]string, len(existingAgents))
		for i, a := range existingAgents {
			existingNames[i] = a.Name
		}
		generatedName, genErr := names.GenerateUniqueFromList(existingNames, 100)
		if genErr != nil {
			return fmt.Errorf("failed to generate agent name: %w", genErr)
		}
		agentName = generatedName
		fmt.Printf("Generated name: %s\n", agentName)
	}

	// Check if agent already exists
	if existing := mgr.GetAgent(agentName); existing != nil {
		if existing.State != agent.StateStopped {
			return fmt.Errorf("agent %q already exists and is %s", agentName, existing.State)
		}
	}

	// Determine tool
	toolName := agentCreateTool
	if toolName == "" && ws.Config.Tool != "" {
		toolName = ws.Config.Tool
	}

	if ws.Config.AgentCommand != "" && toolName == "" {
		mgr.SetAgentCommand(ws.Config.AgentCommand)
	} else if toolName != "" {
		if !mgr.SetAgentByName(toolName) {
			return fmt.Errorf("unknown tool %q (available: %v)", toolName, agent.ListAvailableTools())
		}
	}

	// Parse role
	role, roleErr := parseRole(agentCreateRole)
	if roleErr != nil {
		return roleErr
	}

	// Enforce root agent singleton - only one root agent allowed
	if string(role) == "root" {
		existingAgents := mgr.ListAgents()
		for _, a := range existingAgents {
			if string(a.Role) == "root" && a.State != agent.StateStopped {
				return fmt.Errorf("only one active root agent is allowed; %q already has root role", a.Name)
			}
		}
	}

	// Validate role exists in workspace (unless it's the special "null" role)
	if string(role) != "null" && string(role) != "root" {
		roleFile := filepath.Join(ws.RolesDir(), string(role)+".md")
		if _, err := os.Stat(roleFile); err != nil {
			return fmt.Errorf("role %q not found - create it first or use an existing role", role)
		}
	}

	// Validate team name if specified
	if agentCreateTeam != "" {
		if !isValidTeamName(agentCreateTeam) {
			return fmt.Errorf("team name must be alphanumeric with optional hyphens/underscores")
		}
		// Validate team exists
		teamStore := team.NewStore(ws.RootDir)
		if !teamStore.Exists(agentCreateTeam) {
			return fmt.Errorf("team %q does not exist - create it first with 'bc team create %s'", agentCreateTeam, agentCreateTeam)
		}
	}

	// Spawn the agent (with parent if specified)
	fmt.Printf("Creating %s (%s)... ", agentName, role)
	spawned, spawnErr := mgr.SpawnAgentWithOptions(agentName, role, ws.RootDir, agentCreateParent, toolName)
	if spawnErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to create %s: %w", agentName, spawnErr)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(spawned.Session))

	// Set team if specified
	if agentCreateTeam != "" {
		if teamErr := mgr.SetAgentTeam(agentName, agentCreateTeam); teamErr != nil {
			log.Warn("failed to set team", "error", teamErr)
		}
	}

	// Log event
	eventData := map[string]any{"role": string(role), "tool": toolName}
	if agentCreateParent != "" {
		eventData["parent"] = agentCreateParent
	}
	if agentCreateTeam != "" {
		eventData["team"] = agentCreateTeam
	}
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.AgentSpawned,
		Agent:   agentName,
		Message: fmt.Sprintf("created with role %s", role),
		Data:    eventData,
	}); err != nil {
		log.Warn("failed to log agent spawn event", "error", err)
	}

	fmt.Println()
	fmt.Println("Agent created successfully!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  bc agent attach %s    # Attach to session\n", agentName)
	fmt.Printf("  bc agent send %s <msg> # Send message\n", agentName)
	fmt.Printf("  bc agent peek %s       # View output\n", agentName)

	return nil
}

func runAgentList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	if refreshErr := mgr.RefreshState(); refreshErr != nil {
		log.Warn("failed to refresh agent state", "error", refreshErr)
	}

	agents := mgr.ListAgents()

	// Filter by role if specified
	if agentListRole != "" {
		filterRole, roleErr := parseRole(agentListRole)
		if roleErr != nil {
			return roleErr
		}
		filtered := make([]*agent.Agent, 0, len(agents))
		for _, a := range agents {
			if a.Role == filterRole {
				filtered = append(filtered, a)
			}
		}
		agents = filtered
	}

	if agentListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agents)
	}

	if len(agents) == 0 {
		fmt.Println("No agents found")
		if agentListRole != "" {
			fmt.Printf("(filtered by role: %s)\n", agentListRole)
		}
		return nil
	}

	// Determine terminal width
	termWidth := 80
	if w, _, termErr := term.GetSize(os.Stdout.Fd()); termErr == nil && w > 0 {
		termWidth = w
	}
	taskWidth := termWidth - 57
	if taskWidth < 20 {
		taskWidth = 20
	}

	fmt.Printf("%-15s %-12s %-10s %-20s %s\n", "AGENT", "ROLE", "STATE", "UPTIME", "TASK")
	fmt.Println(strings.Repeat("-", termWidth))

	for _, a := range agents {
		uptime := "-"
		if a.State != agent.StateStopped {
			uptime = formatDuration(time.Since(a.StartedAt))
		}

		task := a.Task
		if task == "" {
			task = "-"
		}
		if len(task) > taskWidth {
			task = task[:taskWidth-3] + "..."
		}

		stateStr := colorState(a.State)

		fmt.Printf("%-15s %-12s %s %-20s %s\n",
			a.Name,
			a.Role,
			stateStr,
			uptime,
			task,
		)
	}

	return nil
}

func runAgentAttach(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)

	if !mgr.Tmux().HasSession(agentName) {
		return fmt.Errorf("agent '%s' not running", agentName)
	}

	fmt.Printf("Attaching to %s (use Ctrl+b d to detach)...\n", agentName)
	return mgr.AttachToAgent(agentName)
}

func runAgentPeek(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	if a.State == agent.StateStopped {
		return fmt.Errorf("agent '%s' is stopped", agentName)
	}

	output, captureErr := mgr.CaptureOutput(agentName, agentPeekLines)
	if captureErr != nil {
		return fmt.Errorf("failed to capture output: %w", captureErr)
	}

	fmt.Printf("=== %s (last %d lines) ===\n", agentName, agentPeekLines)
	fmt.Println(output)

	return nil
}

func runAgentShow(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	// JSON output
	if agentShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(a)
	}

	// Human-readable output
	fmt.Printf("Agent: %s\n", a.Name)
	fmt.Printf("Role: %s\n", a.Role)
	fmt.Printf("State: %s\n", a.State)
	if a.Team != "" {
		fmt.Printf("Team: %s\n", a.Team)
	}
	fmt.Printf("Session: %s\n", a.Session)
	if a.WorktreeDir != "" {
		fmt.Printf("Worktree: %s\n", a.WorktreeDir)
	}
	if a.Task != "" {
		fmt.Printf("Task: %s\n", a.Task)
	}
	if a.Tool != "" {
		fmt.Printf("Tool: %s\n", a.Tool)
	}
	if a.ParentID != "" {
		fmt.Printf("Parent: %s\n", a.ParentID)
	}
	if len(a.Children) > 0 {
		fmt.Printf("Children: %s\n", strings.Join(a.Children, ", "))
	}
	fmt.Printf("Started: %s\n", a.StartedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", a.UpdatedAt.Format(time.RFC3339))

	return nil
}

func runAgentStart(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Check if agent exists
	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found - create it first with 'bc agent create %s'", agentName, agentName)
	}

	// Check if agent is in stopped state
	if a.State != agent.StateStopped {
		return fmt.Errorf("agent '%s' is not stopped (current state: %s) - cannot start", agentName, a.State)
	}

	fmt.Printf("Starting %s (%s)... ", agentName, a.Role)
	// SpawnAgentWithOptions will detect the stopped state and resurrect it
	spawned, spawnErr := mgr.SpawnAgentWithOptions(agentName, a.Role, ws.RootDir, a.ParentID, a.Tool)
	if spawnErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start %s: %w", agentName, spawnErr)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(spawned.Session))

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.AgentSpawned,
		Agent:   agentName,
		Message: "restarted via bc agent start",
	}); err != nil {
		log.Warn("failed to log agent start event", "error", err)
	}

	return nil
}

func runAgentStop(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	fmt.Printf("Stopping %s... ", agentName)
	if stopErr := mgr.StopAgent(agentName); stopErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to stop %s: %w", agentName, stopErr)
	}
	fmt.Println("✓")

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.AgentStopped,
		Agent:   agentName,
		Message: "stopped via bc agent stop",
	}); err != nil {
		log.Warn("failed to log agent stop event", "error", err)
	}

	return nil
}

func runAgentSend(cmd *cobra.Command, args []string) error {
	agentName := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	if a.State == agent.StateStopped {
		return fmt.Errorf("agent '%s' is stopped", agentName)
	}

	if sendErr := mgr.SendToAgent(agentName, message); sendErr != nil {
		return fmt.Errorf("failed to send to %s: %w", agentName, sendErr)
	}

	// Log event - Agent field is the sender, recipient goes in Data
	sender := os.Getenv("BC_AGENT_ID")
	if sender == "" {
		sender = "root"
	}
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   sender,
		Message: message,
		Data: map[string]any{
			"recipient": agentName,
		},
	}); err != nil {
		log.Warn("failed to log message sent event", "error", err)
	}

	fmt.Printf("Sent to %s: %s\n", agentName, message)
	return nil
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", agentName)
	}

	// Check if agent is running - require --force
	if a.State != agent.StateStopped && !agentDeleteForce {
		return fmt.Errorf("agent '%s' is %s. Use --force to delete a running agent", agentName, a.State)
	}

	// Confirm deletion (show what will happen)
	if !agentDeleteForce {
		fmt.Printf("Delete agent '%s'? This will remove:\n", agentName)
		fmt.Println("  - tmux session")
		fmt.Println("  - git worktree")
		fmt.Println("  - channel memberships")
		fmt.Println("  - agent state")
		if agentDeletePurge {
			fmt.Println("  - memory directory (--purge)")
		} else {
			fmt.Printf("  Note: Memory preserved at .bc/memory/%s (use --purge to delete)\n", agentName)
		}
		fmt.Print("\nType 'yes' to confirm: ")

		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil {
			return fmt.Errorf("deletion canceled")
		}
		if response != "yes" {
			return fmt.Errorf("deletion canceled")
		}
	}

	// Remove from all channels
	channelStore, chanErr := channel.OpenStore(ws.RootDir)
	if chanErr == nil {
		if loadChanErr := channelStore.Load(); loadChanErr == nil {
			channels := channelStore.List()
			for _, ch := range channels {
				for _, member := range ch.Members {
					if member == agentName {
						_ = channelStore.RemoveMember(ch.Name, agentName)
						fmt.Printf("Removed from channel #%s\n", ch.Name)
						break
					}
				}
			}
			_ = channelStore.Save()
		}
		_ = channelStore.Close()
	}

	// Delete agent with options
	fmt.Printf("Deleting %s... ", agentName)
	deleteOpts := agent.DeleteOptions{
		PurgeMemory: agentDeletePurge,
	}
	if delErr := mgr.DeleteAgentWithOptions(agentName, deleteOpts); delErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to delete %s: %w", agentName, delErr)
	}
	fmt.Println("✓")

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	eventData := map[string]any{
		"purge_memory": agentDeletePurge,
		"forced":       agentDeleteForce,
	}
	_ = eventLog.Append(events.Event{
		Type:    events.AgentStopped,
		Agent:   agentName,
		Message: "deleted via bc agent delete",
		Data:    eventData,
	})

	fmt.Printf("Agent '%s' has been permanently deleted.\n", agentName)
	if !agentDeletePurge {
		fmt.Printf("Memory preserved at .bc/memory/%s\n", agentName)
	}
	return nil
}

func runAgentRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Check if agent exists
	a := mgr.GetAgent(oldName)
	if a == nil {
		return fmt.Errorf("agent '%s' not found", oldName)
	}

	// Check if new name already exists
	if existing := mgr.GetAgent(newName); existing != nil {
		return fmt.Errorf("agent '%s' already exists", newName)
	}

	// Check if running (block unless --force)
	if a.State != agent.StateStopped && !agentRenameForce {
		return fmt.Errorf("agent '%s' is running; use --force to rename anyway", oldName)
	}

	fmt.Printf("Renaming agent '%s' to '%s'...\n", oldName, newName)

	// Step 1: Rename agent in manager (updates state)
	fmt.Print("  Updating agent state... ")
	if renameErr := mgr.RenameAgent(oldName, newName); renameErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to rename agent state: %w", renameErr)
	}
	fmt.Println("✓")

	// Step 2: Update channel memberships
	fmt.Print("  Updating channel memberships... ")
	channelStore := channel.NewStore(filepath.Join(ws.StateDir(), "channels"))
	if err := channelStore.Load(); err != nil {
		fmt.Println("✗")
		_ = channelStore.Close()
		return fmt.Errorf("failed to load channel state: %w", err)
	}
	channels := channelStore.List()
	channelsUpdated := 0
	for _, ch := range channels {
		members, memberErr := channelStore.GetMembers(ch.Name)
		if memberErr != nil {
			continue
		}
		for _, member := range members {
			if member == oldName {
				// Remove old name, add new name
				_ = channelStore.RemoveMember(ch.Name, oldName)
				_ = channelStore.AddMember(ch.Name, newName)
				channelsUpdated++
				break
			}
		}
	}
	if err := channelStore.Save(); err != nil {
		fmt.Println("✗")
		_ = channelStore.Close()
		return fmt.Errorf("failed to save channel state: %w", err)
	}
	_ = channelStore.Close()
	fmt.Printf("✓ (%d channels)\n", channelsUpdated)

	// Step 3: Rename worktree directory if exists
	oldWorktree := filepath.Join(ws.WorktreesDir(), oldName)
	newWorktree := filepath.Join(ws.WorktreesDir(), newName)
	if _, statErr := os.Stat(oldWorktree); statErr == nil {
		fmt.Print("  Renaming worktree directory... ")
		if renameErr := os.Rename(oldWorktree, newWorktree); renameErr != nil {
			fmt.Println("✗")
			log.Warn("failed to rename worktree directory", "error", renameErr)
		} else {
			fmt.Println("✓")
		}
	}

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	_ = eventLog.Append(events.Event{
		Type:    events.AgentSpawned, // Using spawned as rename event
		Agent:   newName,
		Message: fmt.Sprintf("renamed from %s", oldName),
		Data: map[string]any{
			"previous_name": oldName,
		},
	})

	fmt.Printf("\nAgent '%s' has been renamed to '%s'.\n", oldName, newName)
	return nil
}

func runAgentBroadcast(cmd *cobra.Command, args []string) error {
	message := strings.TrimSpace(strings.Join(args, " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	agents := mgr.ListAgents()
	if len(agents) == 0 {
		fmt.Println("No agents to broadcast to")
		return nil
	}

	sender := os.Getenv("BC_AGENT_ID")
	if sender == "" {
		sender = "root"
	}

	sent := 0
	skipped := 0
	failed := 0

	for _, a := range agents {
		// Skip stopped agents
		if a.State == agent.StateStopped {
			skipped++
			continue
		}
		// Skip the sender to avoid echo
		if a.Name == sender {
			skipped++
			continue
		}

		if sendErr := mgr.SendToAgent(a.Name, message); sendErr != nil {
			fmt.Printf("  %s: failed - %v\n", a.Name, sendErr)
			failed++
			continue
		}
		fmt.Printf("  %s: sent\n", a.Name)
		sent++
	}

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   sender,
		Message: message,
		Data: map[string]any{
			"broadcast": true,
			"sent":      sent,
			"skipped":   skipped,
			"failed":    failed,
		},
	}); err != nil {
		log.Warn("failed to log broadcast event", "error", err)
	}

	fmt.Printf("\nBroadcast sent to %d agents (%d skipped, %d failed)\n", sent, skipped, failed)
	return nil
}

func runAgentSendRole(cmd *cobra.Command, args []string) error {
	roleName := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Parse and validate role
	role, roleErr := parseRole(roleName)
	if roleErr != nil {
		return roleErr
	}

	agents := mgr.ListAgents()

	sender := os.Getenv("BC_AGENT_ID")
	if sender == "" {
		sender = "root"
	}

	sent := 0
	skipped := 0
	failed := 0

	for _, a := range agents {
		// Skip if role doesn't match
		if a.Role != role {
			continue
		}
		// Skip stopped agents
		if a.State == agent.StateStopped {
			skipped++
			continue
		}
		// Skip the sender to avoid echo
		if a.Name == sender {
			skipped++
			continue
		}

		if sendErr := mgr.SendToAgent(a.Name, message); sendErr != nil {
			fmt.Printf("  %s: failed - %v\n", a.Name, sendErr)
			failed++
			continue
		}
		fmt.Printf("  %s: sent\n", a.Name)
		sent++
	}

	if sent == 0 && skipped == 0 && failed == 0 {
		fmt.Printf("No running agents with role %q found\n", roleName)
		return nil
	}

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   sender,
		Message: message,
		Data: map[string]any{
			"role":    roleName,
			"sent":    sent,
			"skipped": skipped,
			"failed":  failed,
		},
	}); err != nil {
		log.Warn("failed to log role send event", "error", err)
	}

	fmt.Printf("\nSent to %d %s(s) (%d skipped, %d failed)\n", sent, roleName, skipped, failed)
	return nil
}

func runAgentSendPattern(cmd *cobra.Command, args []string) error {
	pattern := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	agents := mgr.ListAgents()

	sender := os.Getenv("BC_AGENT_ID")
	if sender == "" {
		sender = "root"
	}

	sent := 0
	skipped := 0
	failed := 0
	matched := 0

	for _, a := range agents {
		// Check if name matches pattern using filepath.Match (glob-style)
		match, matchErr := filepath.Match(pattern, a.Name)
		if matchErr != nil {
			return fmt.Errorf("invalid pattern %q: %w", pattern, matchErr)
		}
		if !match {
			continue
		}
		matched++

		// Skip stopped agents
		if a.State == agent.StateStopped {
			skipped++
			continue
		}
		// Skip the sender to avoid echo
		if a.Name == sender {
			skipped++
			continue
		}

		if sendErr := mgr.SendToAgent(a.Name, message); sendErr != nil {
			fmt.Printf("  %s: failed - %v\n", a.Name, sendErr)
			failed++
			continue
		}
		fmt.Printf("  %s: sent\n", a.Name)
		sent++
	}

	if matched == 0 {
		fmt.Printf("No agents matching pattern %q found\n", pattern)
		return nil
	}

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	if err := eventLog.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   sender,
		Message: message,
		Data: map[string]any{
			"pattern": pattern,
			"matched": matched,
			"sent":    sent,
			"skipped": skipped,
			"failed":  failed,
		},
	}); err != nil {
		log.Warn("failed to log pattern send event", "error", err)
	}

	fmt.Printf("\nSent to %d of %d matching agents (%d skipped, %d failed)\n", sent, matched, skipped, failed)
	return nil
}

// isValidTeamName validates that a team name is alphanumeric with optional hyphens/underscores.
func isValidTeamName(name string) bool {
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

// isValidAgentName checks if an agent name contains only safe characters
func isValidAgentName(name string) bool {
	return isValidTeamName(name)
}

// AgentHealth represents the health status of an agent.
type AgentHealth struct {
	Name          string `json:"name"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	LastUpdated   string `json:"last_updated"`
	StaleDuration string `json:"stale_duration,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
	StuckReason   string `json:"stuck_reason,omitempty"`
	StuckDetails  string `json:"stuck_details,omitempty"`
	TmuxAlive     bool   `json:"tmux_alive"`
	StateFresh    bool   `json:"state_fresh"`
	IsStuck       bool   `json:"is_stuck,omitempty"`
}

func runAgentHealth(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Parse timeout duration
	timeout, parseErr := time.ParseDuration(agentHealthTimeout)
	if parseErr != nil {
		return fmt.Errorf("invalid timeout format: %w", parseErr)
	}

	// Parse work timeout for stuck detection
	workTimeout, workParseErr := time.ParseDuration(agentHealthWorkTmout)
	if workParseErr != nil {
		return fmt.Errorf("invalid work-timeout format: %w", workParseErr)
	}

	// Validate --alert flag requires --detect-stuck
	if agentHealthAlert != "" && !agentHealthDetect {
		return fmt.Errorf("--alert requires --detect-stuck to be enabled")
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	if refreshErr := mgr.RefreshState(); refreshErr != nil {
		log.Warn("failed to refresh agent state", "error", refreshErr)
	}

	// Get agents to check
	var agents []*agent.Agent
	if len(args) > 0 {
		// Check specific agent
		a := mgr.GetAgent(args[0])
		if a == nil {
			return fmt.Errorf("agent '%s' not found", args[0])
		}
		agents = []*agent.Agent{a}
	} else {
		// Check all agents
		agents = mgr.ListAgents()
	}

	if len(agents) == 0 {
		fmt.Println("No agents found")
		return nil
	}

	// Prepare stuck detection if enabled
	var eventLog *events.Log
	var stuckConfig events.StuckConfig
	if agentHealthDetect {
		eventLog = events.NewLog(filepath.Join(ws.RootDir, ".bc", "events.jsonl"))
		stuckConfig = events.StuckConfig{
			ActivityTimeout: timeout,
			WorkTimeout:     workTimeout,
			MaxFailures:     agentHealthMaxFail,
		}
	}

	// Compute health for each agent
	healthResults := make([]AgentHealth, 0, len(agents))
	for _, a := range agents {
		health := computeAgentHealth(a, mgr, timeout)

		// Add stuck detection if enabled
		if agentHealthDetect && eventLog != nil {
			agentEvents, readErr := eventLog.ReadByAgent(a.Name)
			if readErr != nil {
				log.Warn("failed to read agent events", "agent", a.Name, "error", readErr)
			} else {
				stuck := events.DetectStuck(agentEvents, stuckConfig)
				if stuck.IsStuck {
					health.IsStuck = true
					health.StuckReason = string(stuck.Reason)
					health.StuckDetails = stuck.Details
					// Override status if stuck
					if health.Status == "healthy" || health.Status == "degraded" {
						health.Status = "stuck"
						health.ErrorMessage = stuck.Details
					}
				}
			}
		}

		healthResults = append(healthResults, health)
	}

	// Send alert to channel if --alert is set and there are stuck agents
	if agentHealthAlert != "" {
		if alertErr := sendStuckAlert(ws.RootDir, agentHealthAlert, healthResults, mgr); alertErr != nil {
			log.Warn("failed to send stuck alert", "error", alertErr)
		}
	}

	// Output
	if agentHealthJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(healthResults)
	}

	// Table output
	fmt.Printf("%-15s %-12s %-10s %-8s %-8s %s\n", "AGENT", "ROLE", "STATUS", "TMUX", "FRESH", "LAST UPDATED")
	fmt.Println(strings.Repeat("-", 75))

	for _, h := range healthResults {
		tmuxStr := "✗"
		if h.TmuxAlive {
			tmuxStr = "✓"
		}
		freshStr := "✗"
		if h.StateFresh {
			freshStr = "✓"
		}

		statusColor := h.Status
		switch h.Status {
		case "healthy":
			statusColor = "\033[32m" + h.Status + "\033[0m" // green
		case "degraded":
			statusColor = "\033[33m" + h.Status + "\033[0m" // yellow
		case "unhealthy":
			statusColor = "\033[31m" + h.Status + "\033[0m" // red
		case "stuck":
			statusColor = "\033[35m" + h.Status + "\033[0m" // magenta
		}

		fmt.Printf("%-15s %-12s %-10s %-8s %-8s %s\n",
			h.Name,
			h.Role,
			statusColor,
			tmuxStr,
			freshStr,
			h.LastUpdated,
		)

		if h.ErrorMessage != "" {
			fmt.Printf("  └─ %s\n", h.ErrorMessage)
		}
	}

	// Summary
	var healthy, degraded, unhealthy, stuck int
	for _, h := range healthResults {
		switch h.Status {
		case "healthy":
			healthy++
		case "degraded":
			degraded++
		case "unhealthy":
			unhealthy++
		case "stuck":
			stuck++
		}
	}
	if agentHealthDetect {
		fmt.Printf("\nSummary: %d healthy, %d degraded, %d unhealthy, %d stuck (threshold: %s, work-timeout: %s)\n",
			healthy, degraded, unhealthy, stuck, timeout, agentHealthWorkTmout)
	} else {
		fmt.Printf("\nSummary: %d healthy, %d degraded, %d unhealthy (threshold: %s)\n",
			healthy, degraded, unhealthy, timeout)
	}

	return nil
}

func computeAgentHealth(a *agent.Agent, mgr *agent.Manager, timeout time.Duration) AgentHealth {
	health := AgentHealth{
		Name:        a.Name,
		Role:        string(a.Role),
		LastUpdated: a.UpdatedAt.Format(time.RFC3339),
	}

	// Check tmux session
	health.TmuxAlive = mgr.Tmux().HasSession(a.Name)

	// Check state freshness
	staleDuration := time.Since(a.UpdatedAt)
	health.StateFresh = staleDuration < timeout
	if !health.StateFresh {
		health.StaleDuration = staleDuration.Round(time.Second).String()
	}

	// Determine overall status
	switch {
	case a.State == agent.StateStopped:
		health.Status = "unhealthy"
		health.ErrorMessage = "agent stopped"
	case a.State == agent.StateError:
		health.Status = "unhealthy"
		health.ErrorMessage = "agent in error state"
	case !health.TmuxAlive:
		health.Status = "unhealthy"
		health.ErrorMessage = "tmux session not found"
	case !health.StateFresh:
		health.Status = "degraded"
		health.ErrorMessage = fmt.Sprintf("state stale (%s since last update)", health.StaleDuration)
	default:
		health.Status = "healthy"
	}

	return health
}

// sendStuckAlert sends an alert to the specified channel when stuck agents are detected.
func sendStuckAlert(rootDir, channelName string, healthResults []AgentHealth, mgr *agent.Manager) error {
	// Collect stuck agents
	var stuckAgents []AgentHealth
	for _, h := range healthResults {
		if h.IsStuck || h.Status == "stuck" {
			stuckAgents = append(stuckAgents, h)
		}
	}

	if len(stuckAgents) == 0 {
		// No stuck agents, no alert needed
		return nil
	}

	// Build alert message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("⚠️ ALERT: %d stuck agent(s) detected\n", len(stuckAgents)))
	for _, h := range stuckAgents {
		reason := h.StuckReason
		if reason == "" {
			reason = "unknown"
		}
		details := h.StuckDetails
		if details == "" {
			details = h.ErrorMessage
		}
		sb.WriteString(fmt.Sprintf("  • %s (%s): %s - %s\n", h.Name, h.Role, reason, details))
	}

	message := sb.String()

	// Load channel store
	store, err := channel.OpenStore(rootDir)
	if err != nil {
		return fmt.Errorf("failed to open channel store: %w", err)
	}
	defer func() { _ = store.Close() }()

	if loadErr := store.Load(); loadErr != nil {
		return fmt.Errorf("failed to load channel store: %w", loadErr)
	}

	// Get channel members
	members, membersErr := store.GetMembers(channelName)
	if membersErr != nil {
		return fmt.Errorf("channel %q not found: %w", channelName, membersErr)
	}

	if len(members) == 0 {
		fmt.Printf("Alert: channel %q has no members, alert not sent\n", channelName)
		return nil
	}

	// Record in channel history
	if err := store.AddHistory(channelName, "bc-health", message); err != nil {
		log.Warn("failed to record alert history", "error", err)
	}
	if err := store.Save(); err != nil {
		log.Warn("failed to save alert history", "error", err)
	}

	// Send to all members
	sent := 0
	for _, member := range members {
		a := mgr.GetAgent(member)
		if a == nil || a.State == agent.StateStopped {
			continue
		}
		formattedMsg := fmt.Sprintf("[#%s] bc-health: %s", channelName, message)
		if sendErr := mgr.SendToAgent(member, formattedMsg); sendErr != nil {
			log.Warn("failed to send alert to agent", "agent", member, "error", sendErr)
			continue
		}
		sent++
	}

	fmt.Printf("Alert sent to %d member(s) in channel %q\n", sent, channelName)
	return nil
}
