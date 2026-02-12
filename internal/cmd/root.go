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

Key features:
  • Coordinate multiple AI coding agents in parallel
  • Isolated git worktrees per agent
  • Channel-based agent communication
  • Cost tracking and limits
  • Hierarchical agent roles (product-manager, manager, engineer)

Documentation: https://github.com/rpuneet/bc`,
	// PersistentPreRun initializes logging based on flags
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		verbose, err := cmd.Flags().GetBool("verbose")
		if err == nil {
			log.SetVerbose(verbose)
		}
	},
	// Run with no args shows help
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
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
	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().Bool("json", false, "Output in JSON format")

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
