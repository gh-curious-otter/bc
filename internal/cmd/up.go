package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system.

By default, this only starts the root agent which orchestrates the workspace.
Other agents can be created or started as needed by the root agent.

Example:
  bc up                      # Start root agent
  bc up --agent cursor       # Use Cursor AI for root agent`,
	RunE: runUp,
}

var upAgent string

func init() {
	upCmd.Flags().StringVar(&upAgent, "agent", "", "Agent type from config (e.g. claude, cursor, cursor-agent, codex)")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w\nRun 'bc init' first", err)
	}

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)

	// Create workspace-scoped agent manager
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)

	// Use custom agent command: workspace config > --agent flag > default
	if ws.Config.AgentCommand != "" {
		mgr.SetAgentCommand(ws.Config.AgentCommand)
	} else if upAgent != "" {
		if !mgr.SetAgentByName(upAgent) {
			return fmt.Errorf("unknown agent %q (check config [[agents]] for valid names)", upAgent)
		}
	}

	// Check for existing root state and handle recovery
	rootStore := agent.NewRootStateStore(ws.StateDir())
	recovery, err := rootStore.CheckRecovery(mgr.Tmux())
	if err != nil {
		return fmt.Errorf("failed to check root state: %w", err)
	}

	if recovery.IsRunning {
		// Root is already running
		fmt.Println("Root agent already running!")
		fmt.Printf("  Session: %s\n", recovery.State.Session)
		fmt.Printf("  State: %s\n", recovery.State.State)
		if len(recovery.State.Children) > 0 {
			fmt.Printf("  Children: %s\n", strings.Join(recovery.State.Children, ", "))
		}
		fmt.Println()
		fmt.Println("Use 'bc attach root' to attach or 'bc down' first to restart.")
		return nil
	}

	if recovery.NeedsRecover {
		// Root state exists but session is dead - recover
		fmt.Println("Recovering crashed root agent...")
		fmt.Printf("  Previous session: %s\n", recovery.State.Session)
		if len(recovery.State.Children) > 0 {
			fmt.Printf("  Children to preserve: %s\n", strings.Join(recovery.State.Children, ", "))
		}
		fmt.Println()
	}

	// Event log
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Start root (acts as root agent)
	fmt.Print("Starting root... ")
	coord, err := mgr.SpawnAgent("root", agent.RoleRoot, ws.RootDir)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start root: %w", err)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(coord.Session))

	// Create or update root state
	if recovery.NeedsCreate || recovery.NeedsRecover {
		if recovery.NeedsCreate {
			// Create new root state
			_, createErr := rootStore.Create("root", agent.RoleRoot, ws.DefaultTool())
			if createErr != nil && createErr != agent.ErrRootExists {
				fmt.Printf("  Warning: failed to create root state: %v\n", createErr)
			}
		}
		// Update session in root state
		if updateErr := rootStore.MarkRecovered(coord.Session); updateErr != nil {
			fmt.Printf("  Warning: failed to update root session: %v\n", updateErr)
		}
	}

	_ = log.Append(events.Event{
		Type:  events.AgentSpawned,
		Agent: "root",
	})

	// Wait for agent to initialize (Gemini/Claude needs time to start REPL)
	time.Sleep(3 * time.Second)

	// Create default channels for root
	createDefaultChannels(ws.RootDir, []string{"root"})

	// Send bootstrap prompt to root
	fmt.Print("Sending bootstrap prompt to root... ")
	prompt := buildBootstrapPrompt(ws.RootDir)
	if err := mgr.SendToAgent("root", prompt); err != nil {
		fmt.Println("✗")
		fmt.Printf("  Warning: failed to send bootstrap prompt: %v\n", err)
	} else {
		fmt.Println("✓")
	}

	fmt.Println()
	fmt.Println("Root agent started!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc status          # View agent status")
	fmt.Println("  bc attach root     # Attach to root session")
	fmt.Println("  bc down            # Stop all agents")

	return nil
}

func buildBootstrapPrompt(rootDir string) string {
	return fmt.Sprintf(`- ROOT: ORCHESTRATOR & SYSTEM AGENT

=== CORE IDENTITY ===
Role: Root orchestrator - system-level agent ensuring workspace health
Purpose: Monitor all agents, detect issues, maintain smooth operation
Authority: System-level oversight - NEVER assign work, only maintain health
Tools: ONLY bc commands - no direct file manipulation or other tools
Workspace: %s

=== ROOT RESPONSIBILITIES ===
1. System Health: Monitor all agents via bc status, bc dashboard
2. Agent Health: Detect stuck agents via bc agent peek, send nudges when needed
4. Worktree Health: Monitor via bc worktree list, prune orphaned worktrees
5. Event Monitoring: Track activity via bc logs
6. Cost Monitoring: Track resource usage via bc cost show
7. Queue Awareness: Monitor work items via bc queue (awareness only, not assignment)

=== BOUNDARIES ===
NEVER: Assign work (manager role), use channels (cause flooding), manipulate files directly
ONLY: Use bc commands for health monitoring and system coordination
COMMUNICATE: Via bc agent send for health nudges only, not work assignment

=== BC COMMAND REFERENCE ===

** Agent Operations **
bc agent list                       # List all agents
bc agent peek NAME                  # View agent output (detect if stuck)
bc agent send NAME "message"        # Send health nudge
bc agent stop NAME                  # Stop agent (use sparingly)
bc agent create NAME --role ROLE    # Create new agent

** System Status **
bc status                           # All agents overview
bc status --json                    # JSON format
bc dashboard                        # Real-time workspace view
bc dashboard --json                 # JSON format

** Event Monitoring **
bc logs                             # Show all events
bc logs --agent NAME                # Filter by agent
bc logs --type TYPE                 # Filter by event type
bc logs --since DURATION            # Events since duration (1h, 30m)
bc logs --tail N                    # Last N events
bc logs --full                      # Full messages
bc logs --json                      # JSON output

** Merge Management **
bc merge --status                   # Show pending merges
bc merge --status --json            # JSON format
bc merge AGENT                      # Merge agents work
bc merge AGENT --dry-run            # Preview without merging
bc merge AGENT --rebase             # Rebase onto main first

** Worktree Health **
bc worktree list                    # List all worktrees and status
bc worktree check                   # Check if agent in correct worktree
bc worktree prune                   # Clean orphaned worktrees

** Work Queue (Awareness Only) **
bc queue                            # View work items
bc queue list                       # List all work items

** Cost Monitoring **
bc cost show                        # Current agent costs
bc cost show AGENT                  # Specific agent costs
bc cost summary --workspace         # Workspace total
bc cost dashboard                   # Comprehensive view

** Memory Management **
bc memory show                      # Current agent memory
bc memory show AGENT                # Specific agent memory
bc memory record "text"             # Record experience
bc memory learn "topic" "text"      # Add learning
bc memory search "query"            # Search memories
bc memory prune --older-than 30d    # Remove old experiences
bc memory prune --agent NAME        # Prune specific agent

** Configuration **
bc config show                      # Show all config
bc config get KEY                   # Get config value (e.g. tools.default)
bc config set KEY VALUE             # Set config value
bc config list                      # List all config keys
bc config edit                      # Open config in editor
bc config validate                  # Validate config file
bc config reset                     # Reset to defaults

** Role Management **
bc role list                        # List all roles
bc role show ROLE                   # Show role details
bc role create --name NAME          # Create new role
bc role edit ROLE                   # Edit role definition
bc role delete ROLE                 # Delete a role
bc role validate                    # Validate role files

** Workspace Control **
bc up                               # Start all agents
bc down                             # Stop all agents gracefully
bc down --force                     # Force stop

** Statistics **
bc stats                            # Workspace statistics
bc stats --json                     # JSON format

=== MONITORING WORKFLOW ===
1. Check system: bc status
2. Review activity: bc logs --since 1h
3. Check merges: bc merge --status
4. Check worktrees: bc worktree list
5. If agent stuck: bc agent peek NAME → bc agent send NAME "nudge message"
6. Monitor costs: bc cost show (periodic)

=== HEALTH CHECK PATTERNS ===
Stuck agent detection:
- bc status shows long-running task
- bc agent peek NAME shows no progress
- Action: bc agent send NAME "Brief nudge about specific issue"

Worktree issues:
- bc worktree list shows ORPHANED
- Action: bc worktree prune
`, rootDir)
}

// createDefaultChannels sets up the default communication channels.
func createDefaultChannels(rootDir string, agents []string) {
	// Use SQLite store for channels (v2)
	store := channel.NewSQLiteStore(rootDir)
	if err := store.Open(); err != nil {
		fmt.Printf("  Warning: failed to open channel store: %v\n", err)
		return
	}
	defer func() { _ = store.Close() }()

	type chanDef struct {
		name        string
		description string
		channelType channel.ChannelType
		members     []string
	}

	channels := []chanDef{
		{name: "all", description: "Broadcast channel", channelType: channel.ChannelTypeGroup, members: agents},
	}

	// Per-agent channels for direct messaging
	for _, agentName := range agents {
		channels = append(channels, chanDef{
			name:        agentName,
			channelType: channel.ChannelTypeDirect,
			members:     []string{agentName},
			description: fmt.Sprintf("Direct channel for %s", agentName),
		})
	}

	for _, ch := range channels {
		// Check if channel already exists
		existing, _ := store.GetChannel(ch.name)
		if existing != nil {
			// Channel exists, just ensure members are added
			for _, member := range ch.members {
				_ = store.AddMember(ch.name, member)
			}
			continue
		}

		// Create channel with proper type
		if _, createErr := store.CreateChannel(ch.name, ch.channelType, ch.description); createErr != nil {
			fmt.Printf("  Warning: failed to create channel #%s: %v\n", ch.name, createErr)
			continue
		}

		// Add members
		for _, member := range ch.members {
			_ = store.AddMember(ch.name, member)
		}
	}
}
