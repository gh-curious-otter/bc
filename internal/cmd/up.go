package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc agents",
	Long: `Start the bc agent system via the bcd daemon.

Starts the root agent through the running bcd daemon.

Examples:
  bc up                      # Start root agent
  bc up --agent cursor       # Use Cursor AI for agents
  bc up --runtime docker     # Use Docker runtime`,
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

=== BC COMMAND REFERENCE ===

** Agent Operations **
bc agent list                       # List all agents
bc agent peek NAME                  # View agent output (detect if stuck)
bc agent send NAME "message"        # Send health nudge
bc agent stop NAME                  # Stop agent (use sparingly)
bc agent create NAME --role ROLE    # Create new agent

** System Status **
bc status                           # All agents overview
bc logs                             # Show all events
bc logs --agent NAME                # Filter by agent

** Configuration **
bc config show                      # Show all config
bc config get KEY                   # Get config value
bc config set KEY VALUE             # Set config value

** Role Management **
bc role list                        # List all roles
bc role show ROLE                   # Show role details
bc role create --name NAME          # Create new role

=== MONITORING WORKFLOW ===
1. Check system: bc status
2. Review activity: bc logs --since 1h
3. If agent stuck: bc agent peek NAME → bc agent send NAME "nudge message"
`, rootDir)
}

func runUp(cmd *cobra.Command, args []string) error {
	log.Debug("up command started", "agent", upAgent, "runtime", upRuntime)

	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	ws, wsErr := getWorkspace()
	if wsErr != nil {
		return errNotInWorkspace(wsErr)
	}

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)
	fmt.Print("Starting root... ")

	result, upErr := c.Workspaces.Up(cmd.Context(), upAgent, upRuntime)
	if upErr != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to start: %w", upErr)
	}

	status, _ := result["status"].(string)
	if status == "already_running" {
		fmt.Println("already running")
		fmt.Println()
		fmt.Println("Root agent is already running.")
		fmt.Println("Use 'bc agent attach root' to attach or 'bc down' first to restart.")
		return nil
	}

	fmt.Println("✓")
	fmt.Println()
	fmt.Println("Root agent started!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bc status             # View agent status")
	fmt.Println("  bc agent attach root  # Attach to root session")
	fmt.Println("  bc down               # Stop all agents")

	return nil
}
