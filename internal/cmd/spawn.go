package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
)

var spawnCmd = &cobra.Command{
	Use:        "spawn <name>",
	Short:      "Spawn a new worker agent (deprecated: use 'bc agent create')",
	Deprecated: "use 'bc agent create' instead",
	Long: `Spawn a new worker agent dynamically.

This allows coordinators and managers to create new workers on-demand,
rather than only at startup. The new worker will use the same tool
as other agents in the workspace (or you can specify a different tool).

Examples:
  bc spawn worker-05                      # Spawn worker with default tool
  bc spawn worker-05 --tool cursor        # Spawn worker using Cursor
  bc spawn engineer-01 --role engineer    # Spawn an engineer role agent
  bc spawn pm --role product-manager      # Spawn a product manager`,
	Args: cobra.ExactArgs(1),
	RunE: runSpawn,
}

var (
	spawnTool string
	spawnRole string
)

func init() {
	spawnCmd.Flags().StringVar(&spawnTool, "tool", "", "Agent tool type (e.g., claude, cursor, codex, server)")
	spawnCmd.Flags().StringVar(&spawnRole, "role", "worker", "Agent role (worker, engineer, manager, product-manager, coordinator)")
	rootCmd.AddCommand(spawnCmd)
}

func runSpawn(cmd *cobra.Command, args []string) error {
	agentName := strings.TrimSpace(args[0])
	if agentName == "" {
		return fmt.Errorf("agent name cannot be empty")
	}

	log.Debug("spawn command started", "agent", agentName, "tool", spawnTool, "role", spawnRole)

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}
	log.Debug("workspace found", "root", ws.RootDir)

	// Create workspace-scoped agent manager
	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if err = mgr.LoadState(); err != nil {
		log.Warn("failed to load agent state", "error", err)
	}

	// Check if agent already exists
	if existing := mgr.GetAgent(agentName); existing != nil {
		if existing.State != agent.StateStopped {
			return fmt.Errorf("agent %q already exists and is %s", agentName, existing.State)
		}
		// Stopped agent will be respawned
	}

	// Determine tool: --tool flag > workspace config Tool > workspace config AgentCommand > default
	toolName := spawnTool
	if toolName == "" && ws.Config.Tool != "" {
		toolName = ws.Config.Tool
	}

	// If a custom agent command is set in workspace, use that
	if ws.Config.AgentCommand != "" && toolName == "" {
		mgr.SetAgentCommand(ws.Config.AgentCommand)
	} else if toolName != "" {
		if !mgr.SetAgentByName(toolName) {
			return fmt.Errorf("unknown tool %q (available: %v)", toolName, agent.ListAvailableTools())
		}
	}

	// Parse role
	role, err := parseRole(spawnRole)
	if err != nil {
		return err
	}

	// Spawn the agent
	fmt.Printf("Spawning %s (%s)... ", agentName, role)
	spawned, err := mgr.SpawnAgentWithTool(agentName, role, ws.RootDir, toolName)
	if err != nil {
		fmt.Println("✗")
		return fmt.Errorf("failed to spawn %s: %w", agentName, err)
	}
	fmt.Printf("✓ (session: %s)\n", mgr.Tmux().SessionName(spawned.Session))

	// Log event
	logEvent(ws, events.Event{
		Type:    events.AgentSpawned,
		Agent:   agentName,
		Message: fmt.Sprintf("dynamically spawned with role %s", role),
		Data:    map[string]any{"role": string(role), "tool": toolName},
	})

	// Print helpful info
	fmt.Println()
	fmt.Println("Agent spawned successfully!")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Printf("  bc attach %s        # Attach to agent's session\n", agentName)
	fmt.Printf("  bc send %s <msg>    # Send a message to the agent\n", agentName)
	fmt.Println("  bc status             # View all agent status")

	return nil
}

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
