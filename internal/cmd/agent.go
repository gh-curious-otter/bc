package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/client"
	"github.com/rpuneet/bc/pkg/container"
	"github.com/rpuneet/bc/pkg/cost"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/provider"
	"github.com/rpuneet/bc/pkg/ui"
	"github.com/rpuneet/bc/pkg/workspace"
)

// newAgentManager creates an agent manager with the appropriate runtime backend.
// Uses workspace config to determine the default backend. Both tmux and docker
// backends are always available so agents can use either runtime.
func newAgentManager(ws *workspace.Workspace) *agent.Manager {
	backend := ""
	if ws.Config != nil {
		backend = ws.Config.Runtime.Backend
	}

	if backend == "docker" {
		var wsCfg workspace.DockerRuntimeConfig
		if ws.Config != nil {
			wsCfg = ws.Config.Runtime.Docker
		}
		dockerCfg := container.ConfigFromWorkspace(wsCfg)
		be, err := container.NewBackend(dockerCfg, "bc-", ws.RootDir, provider.DefaultRegistry)
		if err != nil {
			log.Warn("Docker unavailable, falling back to tmux", "error", err)
		} else {
			return agent.NewWorkspaceManagerWithRuntime(ws.AgentsDir(), ws.RootDir, be, "docker")
		}
	}
	return agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
}

// agentCmd is the parent command for all agent operations
var agentCmd = &cobra.Command{
	Use:     "agent",
	Aliases: []string{"ag"},
	Short:   "Manage bc agents",
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
  bc agent peek eng-01              # Show last 50 lines
  bc agent peek eng-01 --lines 100  # Show last 100 lines
  bc agent peek eng-01 --follow     # Stream live output (Ctrl+C to stop)`,
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

This resurrects the agent's tmux session and memory.
The agent must have been previously created and stopped.
By default, resumes the previous session if available.

The agent's tool (claude, gemini, cursor, etc.) is fixed at creation time
and cannot be changed on restart. Use --runtime to switch infrastructure
backends (tmux vs docker) without changing the agent's identity.

Examples:
  bc agent start eng-01                    # Start stopped agent (resumes session)
  bc agent start eng-01 --fresh            # Force new session
  bc agent start eng-01 --runtime docker   # Override runtime backend`,
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

This removes the agent's tmux session, channel memberships,
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

This updates the agent's name and channel memberships.
By default, running agents cannot be renamed (use --force to override).

Examples:
  bc agent rename eng-01 engineer-01
  bc agent rename eng-01 eng-02 --force  # Rename running agent`,
	Args: cobra.ExactArgs(2),
	RunE: runAgentRename,
}

// agentHealthCmd is defined in agent_health.go (issue #1648)

// agentSessionsCmd lists session history for an agent
var agentSessionsCmd = &cobra.Command{
	Use:   "sessions <agent>",
	Short: "List session history for an agent",
	Long: `Show stored session IDs for an agent.

The current session ID (if captured) is listed first, followed by archived
session IDs from previous runs.

Examples:
  bc agent sessions eng-01       # List session IDs
  bc agent sessions eng-01 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentSessions,
}

var agentSessionsJSON bool

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
	agentCreateTool    string
	agentCreateRole    string
	agentCreateParent  string
	agentCreateTeam    string
	agentCreateEnv     string
	agentCreateRuntime string
	agentStartRuntime  string
	agentStartFresh    bool
	agentStartResume   string // explicit session ID to resume
	agentListRole      string
	agentListStatus    string
	agentListJSON      bool
	agentListFull      bool
	agentShowJSON      bool
	agentShowFull      bool
	agentPeekLines     int
	agentPeekFollow    bool
	agentStopForce     bool
	agentDeleteForce   bool
	agentDeletePurge   bool
	agentRenameForce   bool
	agentSendPreview   bool
	agentLogsSince     string
	// Health flags are defined in agent_health.go (issue #1648)
)

func init() {
	// Create flags
	agentCreateCmd.Flags().StringVar(&agentCreateTool, "tool", "", "Agent tool (claude, gemini, cursor, codex, opencode, openclaw, aider)")
	agentCreateCmd.Flags().StringVar(&agentCreateRole, "role", "", "Agent role (required). Use 'bc role list' to see available roles")
	agentCreateCmd.Flags().StringVar(&agentCreateParent, "parent", "", "Parent agent ID (must have permission to create this role)")
	agentCreateCmd.Flags().StringVar(&agentCreateTeam, "team", "", "Team name (alphanumeric)")
	agentCreateCmd.Flags().StringVar(&agentCreateEnv, "env", "", "Path to env file (KEY=VALUE per line)")
	agentCreateCmd.Flags().StringVar(&agentCreateRuntime, "runtime", "", "Runtime backend override: tmux or docker")
	_ = agentCreateCmd.MarkFlagRequired("role")

	// List flags
	agentListCmd.Flags().StringVar(&agentListRole, "role", "", "Filter by role")
	agentListCmd.Flags().StringVar(&agentListStatus, "status", "", "Filter by status (running, stopped, error)")
	agentListCmd.Flags().BoolVar(&agentListJSON, "json", false, "Output as JSON (compact by default)")
	agentListCmd.Flags().BoolVar(&agentListFull, "full", false, "Include full agent data including prompts (with --json)")

	// Show flags
	agentShowCmd.Flags().BoolVar(&agentShowJSON, "json", false, "Output as JSON (compact by default)")
	agentShowCmd.Flags().BoolVar(&agentShowFull, "full", false, "Include full agent data including prompts (with --json)")

	// Peek flags
	agentPeekCmd.Flags().IntVar(&agentPeekLines, "lines", 50, "Number of lines to show")
	agentPeekCmd.Flags().BoolVarP(&agentPeekFollow, "follow", "f", false, "Stream live output (like tail -f)")

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

	// Start flags
	agentStartCmd.Flags().StringVar(&agentStartRuntime, "runtime", "", "Runtime backend override: tmux or docker")
	agentStartCmd.Flags().BoolVar(&agentStartFresh, "fresh", false, "Force new session (ignore saved session)")
	agentStartCmd.Flags().StringVar(&agentStartResume, "resume", "", "Resume a specific session by ID (e.g. --resume cc78cadf-89ce-4820-ab6e-950afd2b6838)")

	// Sessions flags
	agentSessionsCmd.Flags().BoolVar(&agentSessionsJSON, "json", false, "Output as JSON")

	// Add shell completion for agent name arguments
	agentAttachCmd.ValidArgsFunction = CompleteAgentNames
	agentPeekCmd.ValidArgsFunction = CompleteAgentNames
	agentShowCmd.ValidArgsFunction = CompleteAgentNames
	agentStartCmd.ValidArgsFunction = CompleteAgentNames
	agentStopCmd.ValidArgsFunction = CompleteAgentNames
	agentSendCmd.ValidArgsFunction = CompleteAgentNames
	agentDeleteCmd.ValidArgsFunction = CompleteAgentNames
	agentRenameCmd.ValidArgsFunction = CompleteAgentNames
	agentSessionsCmd.ValidArgsFunction = CompleteAgentNames

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
	agentCmd.AddCommand(agentSessionsCmd)
	agentCmd.AddCommand(agentBroadcastCmd)
	agentCmd.AddCommand(agentSendRoleCmd)
	agentCmd.AddCommand(agentSendPatternCmd)
	agentCmd.AddCommand(agentAuthCmd)
	agentCmd.AddCommand(agentCostCmd)
	agentCmd.AddCommand(agentLogsCmd)

	// Logs flags
	agentLogsCmd.Flags().StringVar(&agentLogsSince, "since", "", "Show events since duration (e.g., 1h, 30m)")

	// Add parent command to root
	rootCmd.AddCommand(agentCmd)
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
	// Validate agent name if provided
	var agentName string
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		agentName = strings.TrimSpace(args[0])
		if !isValidAgentName(agentName) {
			return fmt.Errorf("agent name %q contains invalid characters (use letters, numbers, dash, underscore)", agentName)
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

	// Validate role exists in workspace (local check for better error messages)
	ws, wsErr := getWorkspace()
	if wsErr == nil {
		roleFile := filepath.Join(ws.RolesDir(), string(role)+".md")
		if _, statErr := os.Stat(roleFile); statErr != nil {
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
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	// Generate name if not provided
	if agentName == "" {
		generated, genErr := c.Agents.GenerateName(cmd.Context())
		if genErr != nil {
			return fmt.Errorf("failed to generate agent name: %w", genErr)
		}
		agentName = generated
		fmt.Printf("Generated name: %s\n", agentName)
	}

	// Determine tool
	toolName := agentCreateTool
	if toolName == "" && wsErr == nil {
		toolName = ws.DefaultProvider()
	}

	// Create via client
	fmt.Printf("Creating %s (%s)... ", agentName, role)
	info, createErr := c.Agents.Create(cmd.Context(), client.CreateAgentReq{
		Name:    agentName,
		Role:    string(role),
		Tool:    toolName,
		Runtime: agentCreateRuntime,
		Parent:  agentCreateParent,
		EnvFile: agentCreateEnv,
	})
	if createErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to create %s: %w", agentName, createErr)
	}
	fmt.Printf("✓ (session: %s)\n", info.Session)

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

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	agentList, err := c.Agents.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("list agents: %w", err)
	}

	// Filter by role if specified
	if agentListRole != "" {
		filterRole, roleErr := parseRole(agentListRole)
		if roleErr != nil {
			return roleErr
		}
		filtered := make([]client.AgentInfo, 0, len(agentList))
		for _, a := range agentList {
			if a.Role == string(filterRole) {
				filtered = append(filtered, a)
			}
		}
		agentList = filtered
	}

	// Filter by status if specified
	if agentListStatus != "" {
		filtered := make([]client.AgentInfo, 0, len(agentList))
		for _, a := range agentList {
			if matchesAgentStatus(agent.State(a.State), agentListStatus) {
				filtered = append(filtered, a)
			}
		}
		agentList = filtered
	}

	log.Debug("agents loaded", "count", len(agentList))

	if agentListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agentList)
	}

	if len(agentList) == 0 {
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

	for _, a := range agentList {
		uptime := "-"
		if agent.State(a.State) != agent.StateStopped {
			uptime = formatDuration(time.Since(a.StartedAt))
		}

		task := normalizeTask(a.Task)
		if task == "" {
			task = "-"
		}
		if len(task) > taskWidth {
			task = task[:taskWidth-3] + "..."
		}

		stateStr := colorState(agent.State(a.State))

		table.AddRow(a.Name, a.Role, stateStr, uptime, task)
	}

	table.Print()
	return nil
}

func runAgentAttach(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr := newAgentManager(ws)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	if !mgr.RuntimeForAgent(agentName).HasSession(cmd.Context(), agentName) {
		return fmt.Errorf("agent %q not running", agentName)
	}

	fmt.Printf("Attaching to %s (use Ctrl+b d to detach)...\n", agentName)
	return mgr.AttachToAgent(agentName)
}

func runAgentPeek(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	// --follow mode: keep local tmux access
	if agentPeekFollow {
		ws, err := getWorkspace()
		if err != nil {
			return errNotInWorkspace(err)
		}

		mgr := newAgentManager(ws)
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

		fmt.Printf("=== %s (following, Ctrl+C to stop) ===\n", agentName)

		ctx, cancel := context.WithCancel(cmd.Context())
		defer cancel()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			cancel()
		}()

		return mgr.FollowOutput(ctx, agentName, agentPeekLines, os.Stdout)
	}

	// Static peek: use daemon client
	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	output, peekErr := c.Agents.Peek(cmd.Context(), agentName, agentPeekLines)
	if peekErr != nil {
		return fmt.Errorf("failed to peek %s: %w", agentName, peekErr)
	}

	fmt.Printf("=== %s (last %d lines) ===\n", agentName, agentPeekLines)
	fmt.Println(output)

	return nil
}

func runAgentShow(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	a, err := c.Agents.Get(cmd.Context(), agentName)
	if err != nil {
		return fmt.Errorf("agent %q not found: %w", agentName, err)
	}

	// JSON output
	if agentShowJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(a)
	}

	// Human-readable output using pkg/ui
	pairs := []string{
		"Agent", a.Name,
		"Role", a.Role,
		"State", a.State,
	}
	if a.Team != "" {
		pairs = append(pairs, "Team", a.Team)
	}
	pairs = append(pairs, "Session", a.Session)
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
	if a.SessionID != "" {
		pairs = append(pairs, "Session ID", a.SessionID)
	}
	pairs = append(pairs,
		"Created", a.CreatedAt.Format(time.RFC3339),
		"Started", a.StartedAt.Format(time.RFC3339),
	)
	if a.StoppedAt != nil {
		pairs = append(pairs, "Stopped", a.StoppedAt.Format(time.RFC3339))
	}
	pairs = append(pairs, "Updated", a.UpdatedAt.Format(time.RFC3339))
	ui.SimpleTable(pairs...)

	return nil
}

func runAgentStart(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	if agentStartFresh && agentStartResume != "" {
		return fmt.Errorf("--fresh and --resume are mutually exclusive")
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	fmt.Printf("Starting %s... ", agentName)
	a, startErr := c.Agents.Start(cmd.Context(), agentName, agentStartRuntime, agentStartFresh)
	if startErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start %s: %w", agentName, startErr)
	}
	fmt.Printf("✓ (session: %s)\n", a.Session)

	return nil
}

func runAgentStop(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	fmt.Printf("Stopping %s... ", agentName)
	if stopErr := c.Agents.Stop(cmd.Context(), agentName); stopErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to stop %s: %w", agentName, stopErr)
	}
	fmt.Println("✓")

	return nil
}

func runAgentSend(cmd *cobra.Command, args []string) error {
	agentName := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	// Intent Preview: show what will happen and ask for confirmation
	if agentSendPreview {
		a, _ := c.Agents.Get(cmd.Context(), agentName)

		fmt.Println()
		fmt.Println("╭─────────────────────────────────────────────────────────────╮")
		fmt.Println("│                     Intent Preview                          │")
		fmt.Println("╰─────────────────────────────────────────────────────────────╯")
		fmt.Println()

		if a != nil {
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
		} else {
			fmt.Printf("  Agent:    %s\n", agentName)
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

	if sendErr := c.Agents.Send(cmd.Context(), agentName, message); sendErr != nil {
		return fmt.Errorf("failed to send to %s: %w", agentName, sendErr)
	}

	fmt.Printf("Sent to %s: %s\n", agentName, message)
	return nil
}

func runAgentDelete(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	// Confirm deletion (show what will happen)
	if !agentDeleteForce {
		fmt.Printf("Delete agent %q? This will remove:\n", agentName)
		fmt.Println("  - tmux session")
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
		if strings.TrimSpace(strings.ToLower(response)) != "yes" {
			return fmt.Errorf("deletion canceled")
		}
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	fmt.Printf("Deleting %s... ", agentName)
	if delErr := c.Agents.Delete(cmd.Context(), agentName); delErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to delete %s: %w", agentName, delErr)
	}
	fmt.Println("✓")

	fmt.Printf("Agent '%s' has been permanently deleted.\n", agentName)

	// Purge memory directory if requested (local file operation)
	if agentDeletePurge {
		ws, wsErr := getWorkspace()
		if wsErr == nil {
			memDir := filepath.Join(ws.StateDir(), "memory", agentName)
			if purgeErr := os.RemoveAll(memDir); purgeErr != nil {
				fmt.Printf("Warning: failed to purge memory directory: %v\n", purgeErr)
			} else {
				fmt.Printf("Memory directory purged.\n")
			}
		}
	}
	return nil
}

func runAgentRename(cmd *cobra.Command, args []string) error {
	oldName := args[0]
	newName := args[1]

	if oldName == newName {
		return fmt.Errorf("old and new names are the same")
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	fmt.Printf("Renaming agent %q to '%s'...\n", oldName, newName)
	if renameErr := c.Agents.Rename(cmd.Context(), oldName, newName); renameErr != nil {
		return fmt.Errorf("failed to rename agent: %w", renameErr)
	}

	fmt.Printf("\nAgent '%s' has been renamed to '%s'.\n", oldName, newName)
	return nil
}

func runAgentBroadcast(cmd *cobra.Command, args []string) error {
	message := strings.TrimSpace(strings.Join(args, " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	sent, broadcastErr := c.Agents.Broadcast(cmd.Context(), message)
	if broadcastErr != nil {
		return fmt.Errorf("broadcast failed: %w", broadcastErr)
	}

	fmt.Printf("Broadcast sent to %d agents\n", sent)
	return nil
}

func runAgentSendRole(cmd *cobra.Command, args []string) error {
	roleName := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	result, sendErr := c.Agents.SendToRole(cmd.Context(), roleName, message)
	if sendErr != nil {
		return fmt.Errorf("send-to-role failed: %w", sendErr)
	}

	for _, name := range result.Matched {
		fmt.Printf("  %s: sent\n", name)
	}

	if result.Sent == 0 && result.Skipped == 0 && result.Failed == 0 {
		fmt.Printf("No running agents with role %q found\n", roleName)
		return nil
	}

	fmt.Printf("\nSent to %d %s(s) (%d skipped, %d failed)\n", result.Sent, roleName, result.Skipped, result.Failed)
	return nil
}

func runAgentSendPattern(cmd *cobra.Command, args []string) error {
	pattern := args[0]
	message := strings.TrimSpace(strings.Join(args[1:], " "))
	if message == "" {
		return fmt.Errorf("message cannot be empty")
	}

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	result, sendErr := c.Agents.SendToPattern(cmd.Context(), pattern, message)
	if sendErr != nil {
		return fmt.Errorf("send-pattern failed: %w", sendErr)
	}

	if len(result.Matched) == 0 {
		fmt.Printf("No agents matching pattern %q found\n", pattern)
		return nil
	}

	for _, name := range result.Matched {
		fmt.Printf("  %s: sent\n", name)
	}

	fmt.Printf("\nSent to %d of %d matching agents (%d skipped, %d failed)\n", result.Sent, len(result.Matched), result.Skipped, result.Failed)
	return nil
}

// parseRole parses and validates a role string.
func parseRole(roleStr string) (agent.Role, error) {
	roleStr = strings.ToLower(strings.TrimSpace(roleStr))
	if roleStr == "" {
		return agent.RoleRoot, nil // Default to root if not specified
	}
	// "null" role is a special case - represents an agent with no system prompt
	if roleStr == "null" {
		return agent.Role("null"), nil
	}
	// All roles are now custom - loaded from .bc/roles/<role>.md files
	// Just validate that the role name is sensible
	if !isValidRoleName(roleStr) {
		return "", fmt.Errorf("invalid role name %q (must be alphanumeric with hyphens)", roleStr)
	}
	return agent.Role(roleStr), nil
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
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Role      string     `json:"role"`
	State     string     `json:"state"`
	Task      string     `json:"task,omitempty"`
	Team      string     `json:"team,omitempty"`
	Tool      string     `json:"tool,omitempty"`
	ParentID  string     `json:"parent_id,omitempty"`
	Children  []string   `json:"children,omitempty"`
	Session   string     `json:"session"`
	SessionID string     `json:"session_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	StartedAt time.Time  `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// toCompactAgent converts a full agent to compact representation.
func toCompactAgent(a *agent.Agent) compactAgent {
	return compactAgent{
		ID:        a.ID,
		Name:      a.Name,
		Role:      string(a.Role),
		State:     string(a.State),
		Task:      a.Task,
		Team:      a.Team,
		Tool:      a.Tool,
		ParentID:  a.ParentID,
		Children:  a.Children,
		Session:   a.Session,
		SessionID: a.SessionID,
		CreatedAt: a.CreatedAt,
		StartedAt: a.StartedAt,
		StoppedAt: a.StoppedAt,
		UpdatedAt: a.UpdatedAt,
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

// matchesAgentStatus checks if an agent state matches a status filter.
// Maps detailed internal states to the simplified 4-state model from #1918.
func matchesAgentStatus(state agent.State, status string) bool {
	switch status {
	case "running":
		return state == agent.StateIdle || state == agent.StateWorking || state == agent.StateStarting
	case "stopped":
		return state == agent.StateStopped
	case "error":
		return state == agent.StateError
	case "starting":
		return state == agent.StateStarting
	default:
		// Allow matching by exact internal state name
		return string(state) == status
	}
}

// agentCostCmd shows per-agent cost breakdown
var agentCostCmd = &cobra.Command{
	Use:   "cost <agent>",
	Short: "Show per-agent cost breakdown",
	Long: `Show the cost breakdown for a specific agent including tokens and USD cost.

Examples:
  bc agent cost eng-01       # Show eng-01 cost
  bc agent cost eng-01 --json  # Output as JSON`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentCost,
}

// agentLogsCmd shows agent event history
var agentLogsCmd = &cobra.Command{
	Use:   "logs <agent>",
	Short: "Show agent event history",
	Long: `Show the event log history for a specific agent.

Examples:
  bc agent logs eng-01               # Show all events
  bc agent logs eng-01 --since 1h    # Show events from last hour`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentLogs,
}

func runAgentCost(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Try to get cost data
	costStore := newCostStore(ws.RootDir)
	if costStore == nil {
		fmt.Printf("Agent: %s\n", agentName)
		fmt.Println("No cost data available (cost tracking not enabled)")
		return nil
	}
	defer func() { _ = costStore.Close() }()

	summary, costErr := costStore.AgentSummary(agentName)
	if costErr != nil || summary == nil {
		fmt.Printf("Agent: %s\n", agentName)
		fmt.Println("No cost data recorded yet")
		return nil
	}

	fmt.Printf("Agent: %s\n", agentName)
	fmt.Printf("  Input tokens:  %d\n", summary.InputTokens)
	fmt.Printf("  Output tokens: %d\n", summary.OutputTokens)
	fmt.Printf("  Total tokens:  %d\n", summary.TotalTokens)
	fmt.Printf("  Total cost:    $%.4f\n", summary.TotalCostUSD)
	fmt.Printf("  Requests:      %d\n", summary.RecordCount)

	return nil
}

func runAgentLogs(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	el := openEventLog(ws)
	if el == nil {
		fmt.Println("No event log available")
		return nil
	}
	defer func() { _ = el.Close() }()

	agentEvents, readErr := el.ReadByAgent(agentName)
	if readErr != nil {
		return fmt.Errorf("failed to read agent events: %w", readErr)
	}

	// Filter by --since if specified
	if agentLogsSince != "" {
		since, parseErr := time.ParseDuration(agentLogsSince)
		if parseErr != nil {
			return fmt.Errorf("invalid --since duration %q: %w", agentLogsSince, parseErr)
		}
		cutoff := time.Now().Add(-since)
		filtered := make([]events.Event, 0, len(agentEvents))
		for _, e := range agentEvents {
			if e.Timestamp.After(cutoff) {
				filtered = append(filtered, e)
			}
		}
		agentEvents = filtered
	}

	if len(agentEvents) == 0 {
		fmt.Printf("No events found for agent %q\n", agentName)
		return nil
	}

	fmt.Printf("=== Events for %s (%d total) ===\n\n", agentName, len(agentEvents))
	for _, e := range agentEvents {
		fmt.Printf("[%s] %s: %s\n", e.Timestamp.Format("15:04:05"), e.Type, e.Message)
	}

	return nil
}

// newCostStore opens the cost store, returning nil if unavailable.
func newCostStore(workspacePath string) costStoreCloser {
	cs := cost.NewStore(workspacePath)
	if err := cs.Open(); err != nil {
		return nil
	}
	return cs
}

// costStoreCloser wraps cost.Store for agent cost queries.
type costStoreCloser interface {
	AgentSummary(agentID string) (*cost.Summary, error)
	Close() error
}

// agentAuthCmd manages per-agent authentication for Docker containers.
var agentAuthCmd = &cobra.Command{
	Use:   "auth <agent-name>",
	Short: "Authenticate an agent for Docker containers",
	Long: `Run OAuth login for a specific agent. Each agent has its own isolated
credentials directory. Opens a browser for authentication.

Usage:
  bc agent auth my-agent        # Login for a specific agent
  bc agent auth my-agent status # Check auth status`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		agentName := args[0]

		// Subcommand: status
		if len(args) > 1 && args[1] == "status" {
			ok, statusErr := container.IsAuthenticated(cmd.Context(), ws.RootDir, agentName)
			if statusErr != nil {
				return statusErr
			}
			if ok {
				fmt.Printf("Agent %q is authenticated.\n", agentName)
			} else {
				fmt.Printf("Agent %q is not authenticated. Run: bc agent auth %s\n", agentName, agentName)
			}
			return nil
		}

		// Run login
		return container.LoginIfNeeded(cmd.Context(), ws.RootDir, agentName)
	},
}

func runAgentSessions(cmd *cobra.Command, args []string) error {
	agentName := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr := newAgentManager(ws)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	a := mgr.GetAgent(agentName)
	if a == nil {
		return fmt.Errorf("agent %q not found", agentName)
	}

	type sessionEntry struct {
		ID        string    `json:"id"`
		Timestamp time.Time `json:"timestamp,omitempty"`
		Current   bool      `json:"current,omitempty"`
	}

	var entries []sessionEntry

	// Current stored session ID from state DB
	if a.SessionID != "" {
		entries = append(entries, sessionEntry{ID: a.SessionID, Current: true})
	}

	// Session history files from .bc/agents/<name>/session_history/
	histDir := filepath.Join(ws.StateDir(), "agents", agentName, "session_history")
	files, readErr := os.ReadDir(histDir)
	if readErr == nil {
		// Sort descending (newest first)
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() > files[j].Name()
		})
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			data, readFileErr := os.ReadFile(filepath.Join(histDir, f.Name())) //nolint:gosec // trusted path
			if readFileErr != nil {
				continue
			}
			id := strings.TrimSpace(string(data))
			if id == "" || id == a.SessionID {
				continue // skip duplicates
			}
			// Parse timestamp from filename (2006-01-02T15:04:05.txt)
			name := strings.TrimSuffix(f.Name(), ".txt")
			ts, parseErr := time.Parse("2006-01-02T15:04:05", name)
			entry := sessionEntry{ID: id}
			if parseErr == nil {
				entry.Timestamp = ts
			}
			entries = append(entries, entry)
		}
	}

	if agentSessionsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	if len(entries) == 0 {
		fmt.Printf("No session IDs stored for agent %s.\n", agentName)
		fmt.Printf("Session IDs are captured automatically when the agent stops.\n")
		return nil
	}

	fmt.Printf("Sessions for %s:\n\n", agentName)
	for _, e := range entries {
		current := ""
		if e.Current {
			current = " " + ui.GreenText("(current)")
		}
		ts := ""
		if !e.Timestamp.IsZero() {
			ts = "  " + ui.DimText(e.Timestamp.Format("2006-01-02 15:04:05"))
		}
		fmt.Printf("  %s%s%s\n", e.ID, current, ts)
	}
	fmt.Printf("\nResume a session: bc agent start %s --resume <id>\n", agentName)

	return nil
}
