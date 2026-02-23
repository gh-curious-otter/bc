package cmd

import (
	"context"
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
	"github.com/rpuneet/bc/pkg/telemetry"
	"github.com/rpuneet/bc/pkg/ui"
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
  bc agent                               # List all agents (same as bc agent list)
  bc agent send-pattern "eng-*" "hello"  # Send to matching agents`,
	// #925: Default to list for consistency with bc channel
	RunE: runAgentList,
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
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unexpected argument %q. To filter by role, use: bc agent list --role %s", args[0], args[0])
		}
		return nil
	},
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

Use --preview to see what action will be taken before sending (Intent Preview).
This shows agent details and asks for confirmation.

Examples:
  bc agent send eng-01 "run the tests"
  bc agent send coordinator "check status"
  bc agent send eng-01 "implement login" --preview  # Preview before sending`,
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

Use --force to delete an agent without stopping it first.
Use --purge to also delete the agent's memory directory.

Examples:
  bc agent delete eng-01              # Delete stopped agent (preserves memory)
  bc agent delete eng-01 --force      # Force delete (any state)
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

// agentHealthCmd is defined in agent_health.go (issue #1648)

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
	agentCreateTool   string
	agentCreateRole   string
	agentCreateParent string
	agentCreateTeam   string
	agentListRole     string
	agentListJSON     bool
	agentListFull     bool
	agentShowJSON     bool
	agentShowFull     bool
	agentPeekLines    int
	agentStopForce    bool
	agentDeleteForce  bool
	agentDeletePurge  bool
	agentRenameForce  bool
	agentSendPreview  bool
	// Health flags are defined in agent_health.go (issue #1648)
)

func init() {
	// Create flags
	agentCreateCmd.Flags().StringVar(&agentCreateTool, "tool", "", "Agent tool (claude, gemini, cursor, codex, opencode, openclaw, aider)")
	agentCreateCmd.Flags().StringVar(&agentCreateRole, "role", "", "Agent role (required). Use 'bc role list' to see available roles")
	agentCreateCmd.Flags().StringVar(&agentCreateParent, "parent", "", "Parent agent ID (must have permission to create this role)")
	agentCreateCmd.Flags().StringVar(&agentCreateTeam, "team", "", "Team name (alphanumeric)")
	_ = agentCreateCmd.MarkFlagRequired("role")

	// List flags
	agentListCmd.Flags().StringVar(&agentListRole, "role", "", "Filter by role")
	agentListCmd.Flags().BoolVar(&agentListJSON, "json", false, "Output as JSON (compact by default)")
	agentListCmd.Flags().BoolVar(&agentListFull, "full", false, "Include full agent data including prompts (with --json)")

	// Show flags
	agentShowCmd.Flags().BoolVar(&agentShowJSON, "json", false, "Output as JSON (compact by default)")
	agentShowCmd.Flags().BoolVar(&agentShowFull, "full", false, "Include full agent data including prompts (with --json)")

	// Peek flags
	agentPeekCmd.Flags().IntVar(&agentPeekLines, "lines", 50, "Number of lines to show")

	// Stop flags
	agentStopCmd.Flags().BoolVar(&agentStopForce, "force", false, "Force stop without cleanup")

	// Delete flags
	agentDeleteCmd.Flags().BoolVar(&agentDeleteForce, "force", false, "Force delete running agent without stopping first")
	agentDeleteCmd.Flags().BoolVar(&agentDeletePurge, "purge", false, "Also delete agent's memory directory")

	// Rename flags
	agentRenameCmd.Flags().BoolVar(&agentRenameForce, "force", false, "Rename even if agent is running")

	// Health flags are set up in agent_health.go via initAgentHealthFlags() (issue #1648)
	initAgentHealthFlags()

	// Send flags
	agentSendCmd.Flags().BoolVar(&agentSendPreview, "preview", false, "Show preview of action before sending (Intent Preview)")

	// Add shell completion for agent name arguments
	agentAttachCmd.ValidArgsFunction = CompleteAgentNames
	agentPeekCmd.ValidArgsFunction = CompleteAgentNames
	agentShowCmd.ValidArgsFunction = CompleteAgentNames
	agentStartCmd.ValidArgsFunction = CompleteAgentNames
	agentStopCmd.ValidArgsFunction = CompleteAgentNames
	agentSendCmd.ValidArgsFunction = CompleteAgentNames
	agentDeleteCmd.ValidArgsFunction = CompleteAgentNames
	agentRenameCmd.ValidArgsFunction = CompleteAgentNames

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
		return errNotInWorkspace(err)
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

	// Validate team name if specified (do this before role validation)
	if agentCreateTeam != "" {
		if !isValidTeamName(agentCreateTeam) {
			return fmt.Errorf("team name must be alphanumeric with optional hyphens/underscores")
		}

		// Validate team exists
		teamStore := team.NewStore(filepath.Join(ws.StateDir(), "teams"))
		if !teamStore.Exists(agentCreateTeam) {
			return fmt.Errorf("team %q does not exist. Create it first with: bc team create %s", agentCreateTeam, agentCreateTeam)
		}
	}

	// Validate role is not empty or "null"
	if agentCreateRole == "" || agentCreateRole == "null" {
		return fmt.Errorf("role is required. Use --role to specify a valid role (e.g., engineer, manager). Run 'bc role list' to see available roles")
	}

	// Parse role
	role, roleErr := parseRole(agentCreateRole)
	if roleErr != nil {
		return roleErr
	}

	// Prevent root agent creation via this command
	if role == agent.RoleRoot {
		return fmt.Errorf("cannot create root agent via 'bc agent create'. Use 'bc up' to initialize the root agent")
	}

	// Validate role exists in workspace
	roleFile := filepath.Join(ws.RolesDir(), string(role)+".md")
	if _, err := os.Stat(roleFile); err != nil {
		// List available roles for helpful error message
		availableRoles := []string{}
		if entries, dirErr := os.ReadDir(ws.RolesDir()); dirErr == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
					availableRoles = append(availableRoles, strings.TrimSuffix(entry.Name(), ".md"))
				}
			}
		}
		if len(availableRoles) > 0 {
			return fmt.Errorf("role %q not found. Available roles: %s", role, strings.Join(availableRoles, ", "))
		}
		return fmt.Errorf("role %q not found. Create it first with 'bc role create %s'", role, role)
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
	log.Debug("agent list command started", "role", agentListRole, "json", agentListJSON)

	return withAgentManager(func(ctx *WorkspaceContext) error {
		if refreshErr := ctx.Manager.RefreshState(); refreshErr != nil {
			log.Warn("failed to refresh agent state", "error", refreshErr)
		}

		agents := ctx.Manager.ListAgents()

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

		log.Debug("agents loaded", "count", len(agents))

		if agentListJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			if agentListFull {
				return enc.Encode(agents)
			}
			// Compact output: omit memory/prompts for cleaner output
			return enc.Encode(toCompactAgents(agents))
		}

		if len(agents) == 0 {
			ui.Warning("No agents found")
			if agentListRole != "" {
				fmt.Printf("(filtered by role: %s)\n", agentListRole)
			}
			return nil
		}

		// Determine terminal width for task truncation
		termWidth := 80
		if w, _, termErr := term.GetSize(os.Stdout.Fd()); termErr == nil && w > 0 {
			termWidth = w
		}
		taskWidth := termWidth - 57
		if taskWidth < 20 {
			taskWidth = 20
		}

		// Use pkg/ui table for consistent formatting
		table := ui.NewTable("AGENT", "ROLE", "STATE", "UPTIME", "TASK")

		for _, a := range agents {
			uptime := "-"
			if a.State != agent.StateStopped {
				uptime = formatDuration(time.Since(a.StartedAt))
			}

			task := normalizeTask(a.Task)
			if task == "" {
				task = "-"
			}
			if len(task) > taskWidth {
				task = task[:taskWidth-3] + "..."
			}

			stateStr := colorState(a.State)

			table.AddRow(a.Name, string(a.Role), stateStr, uptime, task)
		}

		table.Print()
		return nil
	})
}

func runAgentAttach(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)

	if !mgr.Tmux().HasSession(context.TODO(), agentName) {
		return fmt.Errorf("agent %q not running", agentName)
	}

	fmt.Printf("Attaching to %s (use Ctrl+b d to detach)...\n", agentName)
	return mgr.AttachToAgent(agentName)
}

func runAgentPeek(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", agentName)
	}

	if a.State == agent.StateStopped {
		return fmt.Errorf("agent %q is stopped (use 'bc agent start %s' to start it)", agentName, agentName)
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
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", agentName)
	}

	// JSON output
	if agentShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if agentShowFull {
			return enc.Encode(a)
		}
		// Compact output: omit memory/prompts for cleaner output
		return enc.Encode(toCompactAgent(a))
	}

	// Human-readable output using pkg/ui
	pairs := []string{
		"Agent", a.Name,
		"Role", string(a.Role),
		"State", string(a.State),
	}
	if a.Team != "" {
		pairs = append(pairs, "Team", a.Team)
	}
	pairs = append(pairs, "Session", a.Session)
	if a.WorktreeDir != "" {
		pairs = append(pairs, "Worktree", a.WorktreeDir)
	}
	if a.Task != "" {
		pairs = append(pairs, "Task", normalizeTask(a.Task))
	}
	if a.Tool != "" {
		pairs = append(pairs, "Tool", a.Tool)
	}
	if a.ParentID != "" {
		pairs = append(pairs, "Parent", a.ParentID)
	}
	if len(a.Children) > 0 {
		pairs = append(pairs, "Children", strings.Join(a.Children, ", "))
	}
	pairs = append(pairs,
		"Started", a.StartedAt.Format(time.RFC3339),
		"Updated", a.UpdatedAt.Format(time.RFC3339),
	)
	ui.SimpleTable(pairs...)

	return nil
}

func runAgentStart(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Check if agent exists
	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found - create it first with 'bc agent create %s'", agentName, agentName)
	}

	// Check if agent is in stopped state
	if a.State != agent.StateStopped {
		return fmt.Errorf("agent %q is already running (state: %s). Stop it first with: bc agent stop %s", agentName, a.State, agentName)
	}

	fmt.Printf("Starting %s (%s)... ", agentName, a.Role)
	// SpawnAgentWithOptions will detect the stopped state and resurrect it
	spawned, spawnErr := mgr.SpawnAgentWithOptions(agentName, a.Role, ws.RootDir, a.ParentID, a.Tool)
	if spawnErr != nil {
		fmt.Println("✗")
		// #1661: Emit telemetry for agent error
		telemetry.AgentError(cmd.Context(), agentName, spawnErr, "start")
		return fmt.Errorf("failed to start %s: %w", agentName, spawnErr)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(spawned.Session))

	// #1661: Emit telemetry for agent spawn
	telemetry.AgentSpawn(cmd.Context(), agentName, string(a.Role), map[string]any{
		"session": spawned.Session,
		"action":  "restart",
	})

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
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", agentName)
	}

	fmt.Printf("Stopping %s... ", agentName)
	if stopErr := mgr.StopAgent(agentName); stopErr != nil {
		fmt.Println("✗")
		// #1661: Emit telemetry for agent error
		telemetry.AgentError(cmd.Context(), agentName, stopErr, "stop")
		return fmt.Errorf("failed to stop %s: %w", agentName, stopErr)
	}
	fmt.Println("✓")

	// #1661: Emit telemetry for agent stop
	telemetry.AgentStop(cmd.Context(), agentName, "user_requested")

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
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", agentName)
	}

	if a.State == agent.StateStopped {
		return fmt.Errorf("agent %q is stopped (use 'bc agent start %s' to start it)", agentName, agentName)
	}

	// Intent Preview: show what will happen and ask for confirmation
	if agentSendPreview {
		fmt.Println()
		fmt.Println("╭─────────────────────────────────────────────────────────────╮")
		fmt.Println("│                     Intent Preview                          │")
		fmt.Println("╰─────────────────────────────────────────────────────────────╯")
		fmt.Println()

		// Agent details
		fmt.Printf("  Agent:    %s\n", a.Name)
		fmt.Printf("  Role:     %s\n", a.Role)
		fmt.Printf("  State:    %s\n", a.State)
		if a.Team != "" {
			fmt.Printf("  Team:     %s\n", a.Team)
		}
		if a.Tool != "" {
			fmt.Printf("  Tool:     %s\n", a.Tool)
		}
		if a.Task != "" {
			fmt.Printf("  Current:  %s\n", normalizeTask(a.Task))
		}
		fmt.Println()

		// Message to send
		fmt.Printf("  Message:  %s\n", message)
		fmt.Println()

		// Action summary
		fmt.Println("  Action:   Will send message to agent's tmux session")
		fmt.Printf("            The agent will process: %q\n", truncateMessage(message, 50))
		fmt.Println()

		// Confirmation
		fmt.Print("  Proceed? [y/N]: ")
		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil {
			return fmt.Errorf("send canceled")
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("Send canceled.")
			return nil
		}
		fmt.Println()
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
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", agentName)
	}

	// Check if agent is not stopped - require --force or stop first
	if a.State != agent.StateStopped && !agentDeleteForce {
		return fmt.Errorf("agent %q is %s (not stopped). Stop it first with 'bc agent stop %s' or use --force to delete anyway", agentName, a.State, agentName)
	}

	// Confirm deletion (show what will happen)
	if !agentDeleteForce {
		fmt.Printf("Delete agent %q? This will remove:\n", agentName)
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

	// Remove from all teams (issue #730)
	teamStore := team.NewStore(ws.RootDir)
	if teamErr := teamStore.RemoveAgentFromAllTeams(agentName); teamErr != nil {
		// Log warning but don't fail deletion
		fmt.Printf("Warning: failed to clean up team memberships: %v\n", teamErr)
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
		return errNotInWorkspace(err)
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Check if agent exists
	a := mgr.GetAgent(oldName)
	if a == nil {
		return fmt.Errorf("agent %q not found (use 'bc agent list' to see available agents)", oldName)
	}

	// Check if new name already exists
	if existing := mgr.GetAgent(newName); existing != nil {
		return fmt.Errorf("agent %q already exists", newName)
	}

	// Check if running (block unless --force)
	if a.State != agent.StateStopped && !agentRenameForce {
		return fmt.Errorf("agent %q is running; use --force to rename anyway", oldName)
	}

	fmt.Printf("Renaming agent %q to '%s'...\n", oldName, newName)

	// Step 1: Rename agent in manager (updates state)
	fmt.Print("  Updating agent state... ")
	if renameErr := mgr.RenameAgent(oldName, newName); renameErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to rename agent state: %w", renameErr)
	}
	fmt.Println("✓")

	// Step 2: Rename tmux session if exists
	if mgr.Tmux().HasSession(context.TODO(), oldName) {
		fmt.Print("  Renaming tmux session... ")
		if renameErr := mgr.Tmux().RenameSession(context.TODO(), oldName, newName); renameErr != nil {
			fmt.Println("✗")
			log.Warn("failed to rename tmux session", "error", renameErr)
		} else {
			fmt.Println("✓")
		}
	}

	// Step 3: Update channel memberships (renumber after adding tmux step)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
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

// compactAgent is a JSON-friendly agent representation without verbose fields.
// Used for --json output without --full flag to reduce output size.
//
//nolint:govet // fieldalignment: JSON field order preferred for readability
type compactAgent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Role        string    `json:"role"`
	State       string    `json:"state"`
	Task        string    `json:"task,omitempty"`
	Team        string    `json:"team,omitempty"`
	Tool        string    `json:"tool,omitempty"`
	ParentID    string    `json:"parent_id,omitempty"`
	Children    []string  `json:"children,omitempty"`
	Session     string    `json:"session"`
	WorktreeDir string    `json:"worktree_dir,omitempty"`
	StartedAt   time.Time `json:"started_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// toCompactAgent converts a full agent to compact representation.
func toCompactAgent(a *agent.Agent) compactAgent {
	return compactAgent{
		ID:          a.ID,
		Name:        a.Name,
		Role:        string(a.Role),
		State:       string(a.State),
		Task:        a.Task,
		Team:        a.Team,
		Tool:        a.Tool,
		ParentID:    a.ParentID,
		Children:    a.Children,
		Session:     a.Session,
		WorktreeDir: a.WorktreeDir,
		StartedAt:   a.StartedAt,
		UpdatedAt:   a.UpdatedAt,
	}
}

// toCompactAgents converts a slice of agents to compact representations.
func toCompactAgents(agents []*agent.Agent) []compactAgent {
	result := make([]compactAgent, len(agents))
	for i, a := range agents {
		result[i] = toCompactAgent(a)
	}
	return result
}
