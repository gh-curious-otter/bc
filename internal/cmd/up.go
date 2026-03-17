package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system.

Starts the root agent and all agents defined in the roster configuration.
If no roster is configured, only the root agent is started.

Examples:
  bc up                      # Start all agents in roster
  bc up --agent cursor       # Use Cursor AI for agents`,
	RunE: runUp,
}

var (
	upAgent   string
	upRuntime string
)

func init() {
	upCmd.Flags().StringVar(&upAgent, "agent", "", "Agent type from config (e.g. claude, cursor, cursor-agent, codex)")
	upCmd.Flags().StringVar(&upRuntime, "runtime", "", "Runtime backend override: tmux or docker")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, args []string) error {
	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w\nRun 'bc init' first", err)
	}

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)

	// Create workspace-scoped agent manager (always uses config default)
	mgr := newAgentManager(ws)

	// Load existing agent state to preserve other agents when starting root
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	// Use custom agent command: --agent flag > default
	if upAgent != "" {
		if !mgr.SetAgentByName(upAgent) {
			return fmt.Errorf("unknown agent %q (check config [[agents]] for valid names)", upAgent)
		}
	}

	// Start root agent — SpawnAgent handles all cases:
	// - No existing root → create fresh
	// - Existing root with live session → return "already running"
	// - Existing root with dead session → respawn (recreate tmux)
	fmt.Print("Starting root... ")
	coord, err := mgr.SpawnAgentWithOptions(agent.SpawnOptions{
		Name:      "root",
		Role:      agent.RoleRoot,
		Workspace: ws.RootDir,
		Runtime:   upRuntime,
	})
	if err != nil {
		fmt.Println("✗")
		// Check if root is already running
		if existing := mgr.GetAgent("root"); existing != nil && mgr.RuntimeForAgent(existing.Name).HasSession(cmd.Context(), existing.Name) {
			fmt.Printf("\nRoot agent already running!\n")
			fmt.Printf("  Session: %s\n", existing.Session)
			fmt.Printf("  State: %s\n", existing.State)
			fmt.Println()
			fmt.Println("Use 'bc agent attach root' to attach or 'bc down' first to restart.")
			return nil
		}
		return fmt.Errorf("failed to start root: %w", err)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.RuntimeForAgent(coord.Name).SessionName(coord.Session))

	// Log event
	logEvent(ws, events.Event{
		Type:  events.AgentSpawned,
		Agent: "root",
	})

	// Wait for agent to initialize (Gemini/Claude needs time to start REPL)
	time.Sleep(3 * time.Second)

	allAgents := []string{"root"}

	// Create default channels for all agents
	createDefaultChannels(ws.RootDir, allAgents)

	// Send bootstrap prompt to root (with memory if available)
	fmt.Print("Sending bootstrap prompt to root... ")
	prompt := buildBootstrapPrompt(ws.RootDir)

	if err := mgr.SendToAgent("root", prompt); err != nil {
		fmt.Println("✗")
		fmt.Printf("  Warning: failed to send bootstrap prompt: %v\n", err)
	} else {
		fmt.Println("✓")
	}

	fmt.Println()
	if len(allAgents) > 1 {
		fmt.Printf("%d agents started!\n", len(allAgents))
	} else {
		fmt.Println("Root agent started!")
	}
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc status          # View agent status")
	fmt.Println("  bc agent attach root  # Attach to root session")
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
1. System Health: Monitor all agents via bc status, bc workspace stats
2. Agent Health: Detect stuck agents via bc agent peek, send nudges when needed
3. Event Monitoring: Track activity via bc logs
4. Cost Monitoring: Track resource usage via bc cost show

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
bc workspace stats                            # Workspace statistics

** Event Monitoring **
bc logs                             # Show all events
bc logs --agent NAME                # Filter by agent
bc logs --type TYPE                 # Filter by event type
bc logs --since DURATION            # Events since duration (1h, 30m)
bc logs --tail N                    # Last N events
bc logs --full                      # Full messages
bc logs --json                      # JSON output

** Cost Monitoring **
bc cost show                        # Current agent costs
bc cost show AGENT                  # Specific agent costs
bc cost summary --workspace         # Workspace total
bc cost dashboard                   # Comprehensive view

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
bc workspace stats                            # Workspace statistics
bc workspace stats --json                     # JSON format

=== MONITORING WORKFLOW ===
1. Check system: bc status
2. Review activity: bc logs --since 1h
3. If agent stuck: bc agent peek NAME → bc agent send NAME "nudge message"
4. Monitor costs: bc cost show (periodic)

=== HEALTH CHECK PATTERNS ===
Stuck agent detection:
- bc status shows long-running task
- bc agent peek NAME shows no progress
- Action: bc agent send NAME "Brief nudge about specific issue"
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
