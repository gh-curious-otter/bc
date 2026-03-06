package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
	"github.com/rpuneet/bc/pkg/workspace"
)

var (
	initQuick  bool
	initPreset string
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new bc v2 workspace",
	Long: `Initialize a new bc v2 workspace in the specified directory (or current directory).

This creates a .bc directory with v2 configuration for managing agents.

v2 workspace structure:
  .bc/
    config.toml    # Workspace configuration
    roles/         # Agent role definitions
      root.md      # Root agent role
    agents/        # Per-agent state files

Examples:
  bc init                        # Interactive wizard
  bc init --quick                # Quick init with defaults
  bc init --preset solo          # Use solo developer preset
  bc init --preset small-team    # Use small team preset
  bc init --preset full-team     # Use full team preset
  bc init ~/Projects/myapp       # Initialize specific directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initQuick, "quick", false, "Quick init with defaults (skip wizard)")
	initCmd.Flags().StringVar(&initPreset, "preset", "", "Use preset configuration (solo, small-team, full-team)")
}

// isV1Workspace checks if a directory has a v1 workspace (config.json).
func isV1Workspace(dir string) bool {
	configPath := filepath.Join(dir, ".bc", "config.json")
	_, err := os.Stat(configPath)
	return err == nil
}

// isV2Workspace checks if a directory has a v2 workspace (config.toml).
func isV2Workspace(dir string) bool {
	configPath := filepath.Join(dir, ".bc", "config.toml")
	_, err := os.Stat(configPath)
	return err == nil
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	log.Debug("init command started", "dir", dir, "quick", initQuick, "preset", initPreset)

	// Handle preset flag
	if initPreset != "" {
		preset := WizardPreset(initPreset)
		switch preset {
		case PresetSolo, PresetSmallTeam, PresetFullTeam:
			return InitWithPreset(dir, preset)
		default:
			return fmt.Errorf("unknown preset: %s (valid: solo, small-team, full-team)", initPreset)
		}
	}

	// Handle quick flag - use solo preset with defaults
	if initQuick {
		return InitWithPreset(dir, PresetSolo)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}
	log.Debug("resolved directory", "absDir", absDir)

	// Check for existing v2 workspace
	if isV2Workspace(absDir) {
		log.Debug("v2 workspace already exists", "dir", absDir)
		return fmt.Errorf("v2 workspace already initialized in %s", absDir)
	}

	// Check for existing v1 workspace
	if isV1Workspace(absDir) {
		log.Debug("v1 workspace detected", "dir", absDir)
		fmt.Fprintln(os.Stderr, "Warning: Existing v1 workspace detected.")
		fmt.Fprintln(os.Stderr, "bc v2 is a clean break - v1 data will not be migrated.")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "To proceed:")
		fmt.Fprintln(os.Stderr, "  - Backup .bc/ if needed")
		fmt.Fprintln(os.Stderr, "  - Remove .bc/ directory")
		fmt.Fprintln(os.Stderr, "  - Run bc init again")
		return fmt.Errorf("cannot initialize: v1 workspace exists")
	}

	// Run interactive wizard
	return RunWizard(dir)
}

// initV2Workspace creates a new v2 workspace structure.
func initV2Workspace(rootDir string) error {
	stateDir := filepath.Join(rootDir, ".bc")

	// Create state directory
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return fmt.Errorf("failed to create .bc directory: %w", err)
	}

	// Create agents directory
	agentsDir := filepath.Join(stateDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Create and save v2 config
	name := filepath.Base(rootDir)
	cfg := workspace.DefaultV2Config(name)
	configPath := workspace.ConfigPath(rootDir)

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create roles directory and default root.md
	roleMgr := workspace.NewRoleManager(stateDir)
	created, err := roleMgr.EnsureDefaultRoot()
	if err != nil {
		return fmt.Errorf("failed to create role files: %w", err)
	}

	// Initialize channel database
	channelStore := channel.NewSQLiteStore(rootDir)
	if openErr := channelStore.Open(); openErr != nil {
		return fmt.Errorf("failed to initialize channel database: %w", openErr)
	}
	_ = channelStore.Close()

	// Register in global registry
	reg, err := workspace.LoadRegistry()
	if err == nil {
		reg.Register(rootDir, name)
		_ = reg.Save()
	}

	// Print success message
	fmt.Printf("Initialized bc v2 workspace in %s\n", rootDir)
	fmt.Printf("\n")
	fmt.Printf("  Created:\n")
	fmt.Printf("    .bc/config.toml     # Workspace configuration\n")
	fmt.Printf("    .bc/agents/         # Agent state directory\n")
	fmt.Printf("    .bc/roles/          # Role definitions\n")
	if created {
		fmt.Printf("    .bc/roles/root.md   # Root agent role\n")
	}
	fmt.Printf("    .bc/channels.db     # Channel database\n")
	fmt.Printf("\n")
	fmt.Printf("  Default tool: %s\n", cfg.Tools.Default)
	fmt.Printf("  Memory: %s (%s)\n", cfg.Memory.Backend, cfg.Memory.Path)
	fmt.Printf("\n")
	fmt.Printf("Next steps:\n")
	fmt.Printf("  bc up       # Start agents\n")
	fmt.Printf("  bc status   # Check agent status\n")

	return nil
}

// getWorkspace finds the current workspace.
// Supports both v1 (config.json) and v2 (config.toml) workspaces.
// Checks BC_WORKSPACE env var first (for agents in worktrees), then walks up directory tree.
func getWorkspace() (*workspace.Workspace, error) {
	// Check BC_WORKSPACE first (agents set this to point to main workspace)
	if wsPath := os.Getenv("BC_WORKSPACE"); wsPath != "" {
		return workspace.Load(wsPath)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return workspace.Find(cwd)
}

// stateDBPath returns the path to the workspace's state.db for events and agents.
func stateDBPath(ws *workspace.Workspace) string {
	return filepath.Join(ws.StateDir(), "state.db")
}

// openEventLog opens the SQLite event log for the given workspace.
// Returns nil (with a warning) on error so callers can proceed without events.
func openEventLog(ws *workspace.Workspace) events.EventStore {
	el, err := events.NewSQLiteLog(stateDBPath(ws))
	if err != nil {
		log.Warn("failed to open event log", "error", err)
		return nil
	}
	return el
}

// logEvent is a convenience wrapper that opens the event log, appends one event,
// and closes. Failures are logged as warnings and never block the caller.
func logEvent(ws *workspace.Workspace, event events.Event) {
	el := openEventLog(ws)
	if el == nil {
		return
	}
	defer func() { _ = el.Close() }()
	if err := el.Append(event); err != nil {
		log.Warn("failed to log event", "type", string(event.Type), "error", err)
	}
}

// errorAgentNotRunning returns an error message for commands that require BC_AGENT_ID.
func errorAgentNotRunning(commandUsage string) error {
	return fmt.Errorf("this command can only be run by agents in the bc system (use: bc agent send <agent-name> %q)", commandUsage)
}

// errorWorktreeNotSet returns an error message for commands that require BC_AGENT_WORKTREE.
func errorWorktreeNotSet() error {
	return fmt.Errorf("this command must run inside an agent session. Attach to an agent with 'bc agent attach <name>' first")
}

// errNotInWorkspace returns an actionable error for commands that require a bc workspace.
func errNotInWorkspace(err error) error {
	if err != nil {
		return fmt.Errorf("not in a bc workspace (run 'bc init' to initialize one): %w", err)
	}
	return fmt.Errorf("not in a bc workspace. Run 'bc init' in your project directory to create one")
}

// requireWorkspace returns the current workspace or an actionable error.
// This is a convenience wrapper around getWorkspace() with standard error handling.
func requireWorkspace() (*workspace.Workspace, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}
	return ws, nil
}

// WorkspaceContext holds workspace and agent manager for command handlers.
// Use withWorkspace() or withAgentManager() to create instances.
type WorkspaceContext struct {
	Workspace *workspace.Workspace
	Manager   *agent.Manager
}

// withWorkspace executes fn with the current workspace.
// Returns errNotInWorkspace if not in a bc workspace.
// Used by commands that only need workspace access (config, stats, etc.).
func withWorkspace(fn func(ws *workspace.Workspace) error) error { //nolint:unused // Will be used as commands migrate to this pattern
	ws, err := requireWorkspace()
	if err != nil {
		return err
	}
	return fn(ws)
}

// withAgentManager executes fn with workspace and initialized agent manager.
// The agent manager state is loaded before fn is called.
// Any state loading errors are logged as warnings (non-fatal).
func withAgentManager(fn func(ctx *WorkspaceContext) error) error {
	ws, err := requireWorkspace()
	if err != nil {
		return err
	}

	mgr := agent.NewWorkspaceManager(ws.AgentsDir(), ws.RootDir)
	if loadErr := mgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}

	return fn(&WorkspaceContext{
		Workspace: ws,
		Manager:   mgr,
	})
}

// runInitInteractive runs an interactive workspace initialization with nickname prompt.
func runInitInteractive(_ *cobra.Command, dir string) error {
	log.Debug("interactive init started", "dir", dir)

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve directory: %w", err)
	}

	// Check for existing workspaces
	if isV2Workspace(absDir) {
		return fmt.Errorf("workspace already initialized in %s", absDir)
	}
	if isV1Workspace(absDir) {
		fmt.Fprintln(os.Stderr, "Warning: Existing v1 workspace detected.")
		fmt.Fprintln(os.Stderr, "Run 'bc init' after removing .bc/ directory to migrate.")
		return fmt.Errorf("cannot initialize: v1 workspace exists")
	}

	// Prompt for nickname
	nickname, err := promptNickname()
	if err != nil {
		return err
	}

	// Initialize workspace with nickname
	return initV2WorkspaceWithNickname(absDir, nickname)
}

// promptNickname prompts the user for their display name.
func promptNickname() (string, error) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Printf("  Your nickname [%s]: ", workspace.DefaultNickname)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)

	// Normalize and validate
	nickname, err := workspace.NormalizeNickname(input)
	if err != nil {
		// Show helpful error
		fmt.Printf("  %s\n", ui.RedText(fmt.Sprintf("Error: %s", err)))
		fmt.Printf("  Using default: %s\n", workspace.DefaultNickname)
		return workspace.DefaultNickname, nil
	}

	// Show auto-correction if @ was added
	if input != "" && !strings.HasPrefix(input, "@") {
		fmt.Printf("  → Auto-corrected to %s\n", nickname)
	}

	return nickname, nil
}

// initV2WorkspaceWithNickname creates a new v2 workspace with a custom nickname.
func initV2WorkspaceWithNickname(rootDir string, nickname string) error {
	stateDir := filepath.Join(rootDir, ".bc")

	// Create state directory
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return fmt.Errorf("failed to create .bc directory: %w", err)
	}

	// Create agents directory
	agentsDir := filepath.Join(stateDir, "agents")
	if err := os.MkdirAll(agentsDir, 0750); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Create and save v2 config with nickname
	name := filepath.Base(rootDir)
	cfg := workspace.DefaultV2Config(name)
	cfg.User.Nickname = nickname
	configPath := workspace.ConfigPath(rootDir)

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Create roles directory and default root.md
	roleMgr := workspace.NewRoleManager(stateDir)
	created, err := roleMgr.EnsureDefaultRoot()
	if err != nil {
		return fmt.Errorf("failed to create role files: %w", err)
	}

	// Initialize channel database
	channelStore := channel.NewSQLiteStore(rootDir)
	if openErr := channelStore.Open(); openErr != nil {
		return fmt.Errorf("failed to initialize channel database: %w", openErr)
	}
	_ = channelStore.Close()

	// Register in global registry
	reg, regErr := workspace.LoadRegistry()
	if regErr == nil {
		reg.Register(rootDir, name)
		_ = reg.Save()
	}

	// Print success message
	fmt.Println()
	fmt.Printf("  %s Workspace initialized at %s\n", ui.GreenText("✓"), rootDir)
	fmt.Printf("  %s Nickname set to %s\n", ui.GreenText("✓"), nickname)
	fmt.Println()
	fmt.Println("  Created:")
	fmt.Println("    .bc/config.toml     # Workspace configuration")
	fmt.Println("    .bc/agents/         # Agent state directory")
	fmt.Println("    .bc/roles/          # Role definitions")
	if created {
		fmt.Println("    .bc/roles/root.md   # Root agent role")
	}
	fmt.Println("    .bc/channels.db     # Channel database")
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Println("    bc          # Open the dashboard")
	fmt.Println("    bc up       # Start agents")
	fmt.Println("    bc status   # Check agent status")

	return nil
}
