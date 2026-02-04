package cmd

import (
	"fmt"
	"os"

	"github.com/rpuneet/bc/pkg/workspace"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [directory]",
	Short: "Initialize a new bc workspace",
	Long: `Initialize a new bc workspace in the specified directory (or current directory).

This creates a .bc directory with configuration for managing agents.

Example:
  bc init                    # Initialize current directory
  bc init ~/Projects/myapp   # Initialize specific directory`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var initMaxWorkers int

func init() {
	initCmd.Flags().IntVar(&initMaxWorkers, "max-workers", 3, "Maximum number of worker agents")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	
	// Check if already initialized
	if workspace.IsWorkspace(dir) {
		return fmt.Errorf("workspace already initialized in %s", dir)
	}
	
	// Initialize workspace
	ws, err := workspace.Init(dir)
	if err != nil {
		return err
	}
	
	// Apply flags
	ws.Config.MaxWorkers = initMaxWorkers
	
	if err := ws.Save(); err != nil {
		return err
	}
	
	// Ensure directories exist
	if err := ws.EnsureDirs(); err != nil {
		return err
	}
	
	fmt.Printf("Initialized bc workspace in %s\n", ws.RootDir)
	fmt.Printf("  State directory: %s\n", ws.StateDir())
	fmt.Printf("  Max workers: %d\n", ws.Config.MaxWorkers)
	fmt.Printf("  Agents run with full permissions\n")
	
	return nil
}

// getWorkspace finds the current workspace.
func getWorkspace() (*workspace.Workspace, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return workspace.Find(cwd)
}
