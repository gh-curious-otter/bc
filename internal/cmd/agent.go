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
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/names"
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
  bc agent stop eng-01                   # Stop agent`,
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

Example:
  bc agent attach eng-01   # Attach to eng-01`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentAttach,
}

// agentPeekCmd shows recent output from an agent
var agentPeekCmd = &cobra.Command{
	Use:   "peek <agent>",
	Short: "View recent output from an agent",
	Long: `Capture and display recent output from an agent's session.

Examples:
  bc agent peek eng-01          # Show last 50 lines
  bc agent peek eng-01 --lines 100  # Show last 100 lines`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentPeek,
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

This removes the agent's tmux session, git worktree, memory directory,
and all state. This action cannot be undone.

Examples:
  bc agent delete eng-01       # Delete eng-01
  bc agent delete eng-01 --force  # Delete without confirmation`,
	Args: cobra.ExactArgs(1),
	RunE: runAgentDelete,
}

// Flags
var (
	agentCreateTool   string
	agentCreateRole   string
	agentCreateParent string
	agentCreateTeam   string
	agentListRole     string
	agentListJSON     bool
	agentPeekLines    int
	agentStopForce    bool
	agentDeleteForce  bool
)

func init() {
	// Create flags
	agentCreateCmd.Flags().StringVar(&agentCreateTool, "tool", "", "Agent tool (claude, cursor, codex)")
	agentCreateCmd.Flags().StringVar(&agentCreateRole, "role", "worker", "Agent role (worker, engineer, manager, product-manager, tech-lead, qa)")
	agentCreateCmd.Flags().StringVar(&agentCreateParent, "parent", "", "Parent agent ID (must have permission to create this role)")
	agentCreateCmd.Flags().StringVar(&agentCreateTeam, "team", "", "Team name (alphanumeric)")

	// List flags
	agentListCmd.Flags().StringVar(&agentListRole, "role", "", "Filter by role")
	agentListCmd.Flags().BoolVar(&agentListJSON, "json", false, "Output as JSON")

	// Peek flags
	agentPeekCmd.Flags().IntVar(&agentPeekLines, "lines", 50, "Number of lines to show")

	// Stop flags
	agentStopCmd.Flags().BoolVar(&agentStopForce, "force", false, "Force stop without cleanup")

	// Delete flags
	agentDeleteCmd.Flags().BoolVar(&agentDeleteForce, "force", false, "Delete without confirmation")

	// Add subcommands
	agentCmd.AddCommand(agentCreateCmd)
	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentAttachCmd)
	agentCmd.AddCommand(agentPeekCmd)
	agentCmd.AddCommand(agentStopCmd)
	agentCmd.AddCommand(agentSendCmd)
	agentCmd.AddCommand(agentDeleteCmd)

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

	// Validate team name if specified
	if agentCreateTeam != "" {
		if !isValidTeamName(agentCreateTeam) {
			return fmt.Errorf("team name must be alphanumeric with optional hyphens/underscores")
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
	_ = eventLog.Append(events.Event{
		Type:    events.AgentSpawned,
		Agent:   agentName,
		Message: fmt.Sprintf("created with role %s", role),
		Data:    eventData,
	})

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
	_ = eventLog.Append(events.Event{
		Type:    events.AgentStopped,
		Agent:   agentName,
		Message: "stopped via bc agent stop",
	})

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
	_ = eventLog.Append(events.Event{
		Type:    events.MessageSent,
		Agent:   sender,
		Message: message,
		Data: map[string]any{
			"recipient": agentName,
		},
	})

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

	// Confirm deletion unless --force is used
	if !agentDeleteForce {
		fmt.Printf("Delete agent '%s'? This will remove:\n", agentName)
		fmt.Println("  - tmux session")
		fmt.Println("  - git worktree")
		fmt.Println("  - memory directory")
		fmt.Println("  - agent state")
		fmt.Print("\nType 'yes' to confirm: ")

		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil {
			return fmt.Errorf("deletion canceled")
		}
		if response != "yes" {
			return fmt.Errorf("deletion canceled")
		}
	}

	fmt.Printf("Deleting %s... ", agentName)
	if delErr := mgr.DeleteAgent(agentName); delErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to delete %s: %w", agentName, delErr)
	}
	fmt.Println("✓")

	// Log event
	eventLog := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	_ = eventLog.Append(events.Event{
		Type:    events.AgentStopped,
		Agent:   agentName,
		Message: "deleted via bc agent delete",
	})

	fmt.Printf("Agent '%s' has been permanently deleted.\n", agentName)
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
