// Package cmd implements the bc CLI commands.
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

var (
	// Version info set from main.
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// SetVersionInfo sets the version information from build flags.
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}

// rootCmd is the base command for bc.
var rootCmd = &cobra.Command{
	Use:   "bc",
	Short: "A simpler, more controllable agent orchestrator",
	Long: `bc is a multi-agent orchestration system for AI coding assistants.

Coordinate multiple AI agents with predictable behavior and cost awareness.
Supports Claude Code, Cursor, Codex, and other AI coding tools.

Key features:
  • Coordinate multiple AI coding agents in parallel
  • Isolated git worktrees per agent
  • Channel-based agent communication
  • Cost tracking and limits
  • Hierarchical agent roles (product-manager, manager, engineer)

Documentation: https://github.com/rpuneet/bc`,
	// PersistentPreRun initializes logging and profiling based on flags
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err == nil {
			log.SetVerbose(verbose)
		}
		// Start profiling if requested
		if err := setupProfiling(); err != nil {
			log.Error("failed to start profiling", "error", err)
		}
	},
	// PersistentPostRun cleans up profiling
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		stopProfiling()
	},
	// Run with no args: open home if initialized, else prompt for init
	RunE: runRoot,
}

// versionCmd shows version information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "bc %s\n", version)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", commit)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  built:  %s\n", date)
	},
}

func init() {
	// Disable auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")

	// Version flag
	rootCmd.Flags().BoolP("version", "V", false, "Print version information")

	// Profiling flags
	registerProfileFlags()

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// Root returns the root command for testing and extension.
func Root() *cobra.Command {
	return rootCmd
}

// runRoot handles the default bc command (no subcommand).
// If workspace is initialized → open TUI home
// If not initialized → prompt to init
// In non-interactive mode → show help
func runRoot(cmd *cobra.Command, args []string) error {
	// Check for version flag
	showVersion, err := cmd.Flags().GetBool("version")
	if err == nil && showVersion {
		versionCmd.Run(cmd, args)
		return nil
	}

	// Check if running in an interactive terminal
	// If not (e.g., piped input, test environment), show help
	if !term.IsTerminal(os.Stdin.Fd()) {
		return cmd.Help()
	}

	// Try to find workspace
	ws, err := getWorkspace()
	if err == nil && ws != nil {
		// Workspace exists - open TUI home
		log.Debug("workspace found, opening home", "root", ws.RootDir)
		return runHome(cmd, args)
	}

	// No workspace - prompt to initialize
	return promptInit(cmd)
}

// promptInit displays an interactive prompt to initialize a new workspace.
func promptInit(cmd *cobra.Command) error {
	fmt.Println()
	fmt.Println("  \033[1mbc - AI Agent Orchestration\033[0m")
	fmt.Println()
	fmt.Println("  No workspace found in current directory.")
	fmt.Println()
	fmt.Print("  Would you like to initialize a new workspace? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(strings.ToLower(input))

	// Default to yes if empty or 'y'
	if input == "" || input == "y" || input == "yes" {
		return runInteractiveInit(cmd)
	}

	// User said no - show help
	fmt.Println()
	return cmd.Help()
}

// runInteractiveInit runs an interactive workspace initialization.
func runInteractiveInit(cmd *cobra.Command) error {
	fmt.Println()
	fmt.Println("  Initializing bc workspace...")
	fmt.Println()

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Run init with interactive nickname prompt
	return runInitInteractive(cmd, cwd)
}
