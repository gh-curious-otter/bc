package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

// workspaceAddCmd manually adds a workspace to the registry
// Issue #1218: Multi-workspace orchestration
var workspaceAddCmd = &cobra.Command{
	Use:   "add <path>",
	Short: "Add a workspace to the registry",
	Long: `Register a workspace in the global registry for quick access.

Examples:
  bc workspace add .                        # Add current directory
  bc workspace add ~/projects/frontend      # Add by path
  bc workspace add . --alias fe             # Add with short alias
  bc workspace add ~/api --alias backend    # Add with alias`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceAdd,
}

// workspaceRemoveCmd removes a workspace from the registry
var workspaceRemoveCmd = &cobra.Command{
	Use:   "remove <alias|path>",
	Short: "Remove a workspace from the registry",
	Long: `Unregister a workspace from the global registry.

This does not delete the workspace, just removes it from the registry.

Examples:
  bc workspace remove fe                    # Remove by alias
  bc workspace remove ~/projects/frontend   # Remove by path`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkspaceRemove,
}

// workspaceSwitchCmd sets the active workspace
var workspaceSwitchCmd = &cobra.Command{
	Use:   "switch <alias|path>",
	Short: "Switch active workspace",
	Long: `Set the active workspace for cross-workspace operations.

Examples:
  bc workspace switch fe                    # Switch by alias
  bc workspace switch ~/projects/frontend   # Switch by path
  bc workspace switch --clear               # Clear active workspace`,
	Args: cobra.MaximumNArgs(1),
	RunE: runWorkspaceSwitch,
}

func init() {
	// List command flags
	workspaceListCmd.Flags().StringSlice("scan", nil, "Additional paths to scan")
	workspaceListCmd.Flags().Bool("no-cache", false, "Skip registry, scan filesystem only")
	workspaceListCmd.Flags().Int("depth", 3, "Maximum scan depth")

	// Discover command flags
	workspaceDiscoverCmd.Flags().StringSlice("scan", nil, "Additional paths to scan")
	workspaceDiscoverCmd.Flags().Int("depth", 3, "Maximum scan depth")

	// Add command flags (#1218)
	workspaceAddCmd.Flags().String("alias", "", "Short alias for quick access")

	// Switch command flags (#1218)
	workspaceSwitchCmd.Flags().Bool("clear", false, "Clear active workspace")

	// Add subcommands
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceDiscoverCmd)
	workspaceCmd.AddCommand(workspaceAddCmd)
	workspaceCmd.AddCommand(workspaceRemoveCmd)
	workspaceCmd.AddCommand(workspaceSwitchCmd)

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

// runWorkspaceAdd handles the workspace add command.
// Issue #1218: Multi-workspace orchestration.
func runWorkspaceAdd(cmd *cobra.Command, args []string) error {
	path := args[0]
	alias, _ := cmd.Flags().GetString("alias")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Verify it's a workspace
	if !workspace.IsWorkspace(absPath) {
		return fmt.Errorf("not a bc workspace: %s (no .bc directory found)", absPath)
	}

	// Load workspace to get name
	ws, err := workspace.Load(absPath)
	if err != nil {
		return fmt.Errorf("failed to load workspace: %w", err)
	}
	name := ws.Config.Name

	// Load registry and add
	reg, err := workspace.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if err := reg.RegisterWithAlias(absPath, name, alias); err != nil {
		return err
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	if jsonOutput {
		output := struct {
			Path  string `json:"path"`
			Name  string `json:"name"`
			Alias string `json:"alias,omitempty"`
		}{
			Path:  absPath,
			Name:  name,
			Alias: alias,
		}
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	if alias != "" {
		fmt.Printf("Added workspace '%s' (%s) as '%s'\n", name, absPath, alias)
	} else {
		fmt.Printf("Added workspace '%s' (%s)\n", name, absPath)
	}

	return nil
}

// runWorkspaceRemove handles the workspace remove command.
func runWorkspaceRemove(cmd *cobra.Command, args []string) error {
	identifier := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	reg, err := workspace.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	entry := reg.FindByNameOrAlias(identifier)
	if entry == nil {
		return fmt.Errorf("workspace not found: %s", identifier)
	}

	// Store info before removal
	removedPath := entry.Path
	removedName := entry.Name

	reg.Unregister(entry.Path)

	// Clear active if this was the active workspace
	if active := reg.GetActive(); active != nil && active.Path == removedPath {
		_ = reg.SetActive("")
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	if jsonOutput {
		output := struct {
			Removed string `json:"removed"`
			Name    string `json:"name"`
		}{
			Removed: removedPath,
			Name:    removedName,
		}
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	fmt.Printf("Removed workspace '%s' from registry\n", removedName)
	return nil
}

// runWorkspaceSwitch handles the workspace switch command.
func runWorkspaceSwitch(cmd *cobra.Command, args []string) error {
	clearActive, _ := cmd.Flags().GetBool("clear")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	reg, err := workspace.LoadRegistry()
	if err != nil {
		return fmt.Errorf("failed to load registry: %w", err)
	}

	if clearActive || len(args) == 0 {
		if err := reg.SetActive(""); err != nil {
			return err
		}
		if err := reg.Save(); err != nil {
			return fmt.Errorf("failed to save registry: %w", err)
		}
		if jsonOutput {
			fmt.Println("{\"active\": null}")
		} else {
			fmt.Println("Cleared active workspace")
		}
		return nil
	}

	identifier := args[0]
	if err := reg.SetActive(identifier); err != nil {
		return err
	}

	if err := reg.Save(); err != nil {
		return fmt.Errorf("failed to save registry: %w", err)
	}

	active := reg.GetActive()
	if jsonOutput {
		output := struct {
			Active string `json:"active"`
			Path   string `json:"path"`
			Name   string `json:"name"`
		}{
			Active: reg.Active,
			Path:   active.Path,
			Name:   active.Name,
		}
		enc := json.NewEncoder(os.Stdout)
		return enc.Encode(output)
	}

	fmt.Printf("Switched to workspace '%s' (%s)\n", active.Name, active.Path)
	return nil
}
