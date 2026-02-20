// Package cmd implements the bc CLI commands.
package cmd

import (
	"fmt"

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

Getting Started:
  bc init                              # Initialize workspace
  bc up                                # Start root agent
  bc agent create eng-01 --role engineer  # Create engineer agent
  bc status                            # View agent status
  bc home                              # Open TUI dashboard

Common Workflows:
  Start working:    bc up && bc status
  Monitor agents:   bc status --activity
  Send message:     bc channel send eng "message"
  Debug agent:      bc logs --agent eng-01 --tail 50
  Cost check:       bc cost show

Key Features:
  • Coordinate multiple AI coding agents in parallel
  • Isolated git worktrees per agent
  • Channel-based agent communication
  • Cost tracking and limits
  • Hierarchical agent roles (product-manager, manager, engineer)

Environment Variables:
  BC_AGENT_ID       Current agent name (set automatically in agent sessions)
  BC_AGENT_ROLE     Current agent role
  BC_WORKSPACE      Path to workspace root
  BC_AGENT_WORKTREE Path to agent's worktree
  BC_BIN            Path to bc binary (default: bc in PATH)
  BC_ROOT           Workspace root directory
  NO_COLOR          Disable colored output

Documentation: https://github.com/rpuneet/bc
Full CLI reference: https://github.com/rpuneet/bc/docs/cli.md`,
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
	// Run with no args shows help
	Run: func(cmd *cobra.Command, args []string) {
		showVersion, err := cmd.Flags().GetBool("version")
		if err == nil && showVersion {
			versionCmd.Run(cmd, args)
			return
		}
		_ = cmd.Help()
	},
}

// versionCmd shows version information.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print version, commit hash, and build date.

Examples:
  bc version       # Show version info
  bc --version     # Same as above (short flag)
  bc -V            # Same as above`,
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
