package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/workspace"
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
  bc init                    # Initialize current directory
  bc init ~/Projects/myapp   # Initialize specific directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
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
	log.Debug("init command started", "dir", dir)

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

	// Initialize v2 workspace
	return initV2Workspace(absDir)
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
func getWorkspace() (*workspace.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return workspace.Find(cwd)
}

// errorAgentNotRunning returns an error message for commands that require BC_AGENT_ID.
func errorAgentNotRunning(commandUsage string) error {
	return fmt.Errorf("this command can only be run by agents in the bc system (use: bc agent send <agent-name> %q)", commandUsage)
}

// errorWorktreeNotSet returns an error message for commands that require BC_AGENT_WORKTREE.
func errorWorktreeNotSet() error {
	return fmt.Errorf("this command must run inside an agent session. Attach to an agent with 'bc agent attach <name>' first")
}
