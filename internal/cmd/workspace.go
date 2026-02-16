package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/workspace"
)

// workspaceCmd is the parent command for workspace operations
var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage bc workspaces",
	Long: `Manage bc workspaces: discover, list, switch.

Examples:
  bc workspace list                   # List discovered workspaces
  bc workspace list --scan ~/Projects # Scan additional paths
  bc workspace discover               # Discover and register new workspaces`,
}

// workspaceListCmd lists all discovered workspaces
var workspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered workspaces",
	Long: `List all bc workspaces on this machine.

Searches:
  - Global registry (~/.bc/workspaces.json)
  - Common directories (~/Projects, ~/Developer, ~/dev, ~/code, ~/repos, ~/src)
  - Additional paths specified with --scan

Examples:
  bc workspace list                    # List all workspaces
  bc workspace list --json             # Output as JSON
  bc workspace list --scan ~/work      # Include additional path
  bc workspace list --no-cache         # Skip registry, scan only`,
	RunE: runWorkspaceList,
}

// workspaceDiscoverCmd discovers and registers new workspaces
var workspaceDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover and register workspaces",
	Long: `Scan filesystem for bc workspaces and add them to the registry.

This updates ~/.bc/workspaces.json with newly found workspaces.

Examples:
  bc workspace discover                # Scan default locations
  bc workspace discover --scan ~/work  # Include additional path`,
	RunE: runWorkspaceDiscover,
}

func init() {
	// List command flags
	workspaceListCmd.Flags().StringSlice("scan", nil, "Additional paths to scan")
	workspaceListCmd.Flags().Bool("no-cache", false, "Skip registry, scan filesystem only")
	workspaceListCmd.Flags().Int("depth", 3, "Maximum scan depth")

	// Discover command flags
	workspaceDiscoverCmd.Flags().StringSlice("scan", nil, "Additional paths to scan")
	workspaceDiscoverCmd.Flags().Int("depth", 3, "Maximum scan depth")

	// Add subcommands
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceDiscoverCmd)

	// Register with root
	rootCmd.AddCommand(workspaceCmd)
}

func runWorkspaceList(cmd *cobra.Command, args []string) error {
	scanPaths, _ := cmd.Flags().GetStringSlice("scan")
	noCache, _ := cmd.Flags().GetBool("no-cache")
	maxDepth, _ := cmd.Flags().GetInt("depth")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	opts := workspace.DiscoverOptions{
		IncludeCached: !noCache,
		ScanHome:      true,
		ScanPaths:     scanPaths,
		MaxDepth:      maxDepth,
	}

	workspaces, err := workspace.Discover(opts)
	if err != nil {
		return fmt.Errorf("failed to discover workspaces: %w", err)
	}

	if jsonOutput {
		output := struct {
			Workspaces []workspace.DiscoveredWorkspace `json:"workspaces"`
		}{
			Workspaces: workspaces,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Text output
	if len(workspaces) == 0 {
		fmt.Println("No workspaces found")
		return nil
	}

	fmt.Printf("Found %d workspace(s):\n\n", len(workspaces))
	for _, ws := range workspaces {
		icon := "📁"
		if ws.IsV2 {
			icon = "📦"
		}
		source := ""
		if ws.FromCache {
			source = " (registered)"
		}
		fmt.Printf("  %s %s%s\n", icon, ws.Name, source)
		fmt.Printf("     %s\n", ws.Path)
	}

	return nil
}

func runWorkspaceDiscover(cmd *cobra.Command, args []string) error {
	scanPaths, _ := cmd.Flags().GetStringSlice("scan")
	maxDepth, _ := cmd.Flags().GetInt("depth")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	opts := workspace.DiscoverOptions{
		IncludeCached: true,
		ScanHome:      true,
		ScanPaths:     scanPaths,
		MaxDepth:      maxDepth,
	}

	newCount, err := workspace.DiscoverAndRegister(opts)
	if err != nil {
		return fmt.Errorf("failed to discover workspaces: %w", err)
	}

	if jsonOutput {
		output := struct {
			NewWorkspaces int `json:"new_workspaces"`
		}{
			NewWorkspaces: newCount,
		}
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	if newCount == 0 {
		fmt.Println("No new workspaces found")
	} else {
		fmt.Printf("Registered %d new workspace(s)\n", newCount)
	}

	return nil
}
