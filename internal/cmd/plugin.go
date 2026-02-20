package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/plugin"
	"github.com/rpuneet/bc/pkg/ui"
)

// Plugin commands for bc plugin system
// Issue #1213: Phase 4 Ecosystem - Plugin system

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage bc plugins",
	Long: `Install, manage, and develop plugins that extend bc.

Plugins can add:
  - Custom agents (e.g., specialized AI models)
  - Tools (e.g., CI/CD integrations)
  - Roles (e.g., industry-specific workflows)

Commands:
  bc plugin list                   List installed plugins
  bc plugin install <source>       Install a plugin
  bc plugin uninstall <name>       Remove a plugin
  bc plugin enable <name>          Enable a plugin
  bc plugin disable <name>         Disable a plugin
  bc plugin info <name>            Show plugin details

Development:
  bc plugin init <name>            Create plugin scaffold
  bc plugin dev                    Run plugin in dev mode

Examples:
  bc plugin install ./my-plugin           # Install from local path
  bc plugin install github-actions        # Install from registry
  bc plugin list                          # Show installed plugins
  bc plugin info github-actions           # Show plugin details
  bc plugin disable github-actions        # Disable plugin`,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE:  runPluginList,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <source>",
	Short: "Install a plugin",
	Long: `Install a plugin from a local path or registry.

Source can be:
  - Local directory path (./my-plugin)
  - Plugin name from registry (github-actions)
  - Git URL (https://github.com/user/plugin)

Examples:
  bc plugin install ./my-plugin
  bc plugin install github-actions
  bc plugin install github-actions@1.0.0`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginUninstallCmd = &cobra.Command{
	Use:   "uninstall <name>",
	Short: "Remove a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginUninstall,
}

var pluginEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginEnable,
}

var pluginDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a plugin",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginDisable,
}

var pluginInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show plugin details",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginInfo,
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for plugins",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginSearch,
}

var pluginInitCmd = &cobra.Command{
	Use:   "init <name>",
	Short: "Create plugin scaffold",
	Long: `Initialize a new plugin project with scaffold files.

Creates:
  - plugin.toml: Plugin manifest
  - src/main.go: Entry point
  - README.md: Documentation template

Examples:
  bc plugin init my-tool
  bc plugin init my-agent --type agent`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInit,
}

// Flags
var pluginInitType string

func init() {
	// Plugin subcommands
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginUninstallCmd)
	pluginCmd.AddCommand(pluginEnableCmd)
	pluginCmd.AddCommand(pluginDisableCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	pluginCmd.AddCommand(pluginSearchCmd)
	pluginCmd.AddCommand(pluginInitCmd)

	pluginInitCmd.Flags().StringVar(&pluginInitType, "type", "tool", "Plugin type: agent, tool, or role")

	rootCmd.AddCommand(pluginCmd)
}

func getPluginManager() (*plugin.Manager, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}

	mgr := plugin.NewManager(ws.StateDir())
	if err := mgr.Load(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to load plugins: %w", err)
	}

	return mgr, nil
}

func runPluginList(cmd *cobra.Command, _ []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	plugins := mgr.List()

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(plugins)
	}

	if len(plugins) == 0 {
		fmt.Println()
		fmt.Println("  No plugins installed.")
		fmt.Println()
		fmt.Println("  Install a plugin with: bc plugin install <source>")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Installed Plugins"))
	fmt.Println("  " + strings.Repeat("─", 60))
	fmt.Println()

	for _, p := range plugins {
		stateIcon := "●"
		stateColor := ui.GreenText
		switch p.State {
		case plugin.StateDisabled:
			stateIcon = "○"
			stateColor = ui.DimText
		case plugin.StateError:
			stateIcon = "✗"
			stateColor = ui.RedText
		}

		fmt.Printf("  %s %s\n", stateColor(stateIcon), ui.CyanText(p.Manifest.Name))
		fmt.Printf("      Version: %s  Type: %s\n", p.Manifest.Version, p.Manifest.Type)
		if p.Manifest.Description != "" {
			fmt.Printf("      %s\n", ui.DimText(p.Manifest.Description))
		}
		fmt.Println()
	}

	return nil
}

func runPluginInstall(_ *cobra.Command, args []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	source := args[0]
	fmt.Printf("Installing plugin from: %s\n", source)

	p, err := mgr.Install(context.Background(), source)
	if err != nil {
		return fmt.Errorf("installation failed: %w", err)
	}

	fmt.Printf("✓ Installed %s v%s (%s)\n", p.Manifest.Name, p.Manifest.Version, p.Manifest.Type)
	return nil
}

func runPluginUninstall(_ *cobra.Command, args []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	name := args[0]
	if err := mgr.Uninstall(context.Background(), name); err != nil {
		return err
	}

	fmt.Printf("✓ Uninstalled %s\n", name)
	return nil
}

func runPluginEnable(_ *cobra.Command, args []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	name := args[0]
	if err := mgr.Enable(name); err != nil {
		return err
	}

	fmt.Printf("✓ Enabled %s\n", name)
	return nil
}

func runPluginDisable(_ *cobra.Command, args []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	name := args[0]
	if err := mgr.Disable(name); err != nil {
		return err
	}

	fmt.Printf("✓ Disabled %s\n", name)
	return nil
}

func runPluginInfo(cmd *cobra.Command, args []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	name := args[0]
	p, err := mgr.Info(name)
	if err != nil {
		return err
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(p)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText(p.Manifest.Name))
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Printf("  Version:     %s\n", p.Manifest.Version)
	fmt.Printf("  Type:        %s\n", p.Manifest.Type)
	fmt.Printf("  State:       %s\n", p.State)
	if p.Manifest.Description != "" {
		fmt.Printf("  Description: %s\n", p.Manifest.Description)
	}
	if p.Manifest.Author != "" {
		fmt.Printf("  Author:      %s\n", p.Manifest.Author)
	}
	if p.Manifest.License != "" {
		fmt.Printf("  License:     %s\n", p.Manifest.License)
	}
	if p.Manifest.Homepage != "" {
		fmt.Printf("  Homepage:    %s\n", p.Manifest.Homepage)
	}
	if p.Manifest.Repository != "" {
		fmt.Printf("  Repository:  %s\n", p.Manifest.Repository)
	}
	fmt.Printf("  Path:        %s\n", p.Path)
	fmt.Printf("  Installed:   %s\n", p.InstalledAt.Format(time.RFC3339))
	if len(p.Manifest.Capabilities) > 0 {
		fmt.Printf("  Capabilities: %s\n", strings.Join(p.Manifest.Capabilities, ", "))
	}
	fmt.Println()

	return nil
}

func runPluginSearch(_ *cobra.Command, args []string) error {
	mgr, err := getPluginManager()
	if err != nil {
		return err
	}

	query := args[0]
	results, err := mgr.Search(context.Background(), query)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Printf("No plugins found matching: %s\n", query)
		return nil
	}

	fmt.Printf("Found %d plugins:\n", len(results))
	for _, r := range results {
		fmt.Printf("  %s v%s - %s\n", r.Name, r.Version, r.Description)
	}

	return nil
}

func runPluginInit(_ *cobra.Command, args []string) error {
	name := args[0]

	// Validate type
	switch pluginInitType {
	case plugin.TypeAgent, plugin.TypeTool, plugin.TypeRole:
		// Valid
	default:
		return fmt.Errorf("invalid type %q (use agent, tool, or role)", pluginInitType)
	}

	// Create directory
	if err := os.MkdirAll(name, 0750); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create plugin.toml
	manifest := fmt.Sprintf(`# Plugin manifest
name = "%s"
version = "0.1.0"
description = "A bc %s plugin"
author = ""
license = "MIT"
type = "%s"
entrypoint = "src/main.go"
bc_version = ">= 1.0.0"

# Optional: plugin capabilities
# capabilities = ["capability1", "capability2"]

# Optional: dependencies
# [[dependencies]]
# name = "other-plugin"
# version = ">= 1.0.0"
`, name, pluginInitType, pluginInitType)

	if err := os.WriteFile(filepath.Join(name, "plugin.toml"), []byte(manifest), 0600); err != nil {
		return fmt.Errorf("failed to create plugin.toml: %w", err)
	}

	// Create src directory and main.go
	srcDir := filepath.Join(name, "src")
	if err := os.MkdirAll(srcDir, 0750); err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
	}

	mainGo := fmt.Sprintf(`package main

// %s - A bc %s plugin
//
// This is the entry point for your plugin.
// Implement your plugin logic here.

func main() {
	// Plugin initialization
}
`, name, pluginInitType)

	if err := os.WriteFile(filepath.Join(srcDir, "main.go"), []byte(mainGo), 0600); err != nil {
		return fmt.Errorf("failed to create main.go: %w", err)
	}

	// Create README.md
	readme := fmt.Sprintf(`# %s

A bc %s plugin.

## Installation

%s
bc plugin install ./%s
%s

## Usage

Describe how to use your plugin here.

## Development

%s
bc plugin dev
%s

## License

MIT
`, name, pluginInitType, "```bash", name, "```", "```bash", "```")

	if err := os.WriteFile(filepath.Join(name, "README.md"), []byte(readme), 0600); err != nil {
		return fmt.Errorf("failed to create README.md: %w", err)
	}

	fmt.Printf("✓ Created plugin scaffold: %s\n", name)
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("  1. cd %s\n", name)
	fmt.Println("  2. Edit plugin.toml with your details")
	fmt.Println("  3. Implement your plugin in src/main.go")
	fmt.Printf("  4. Install with: bc plugin install ./%s\n", name)
	fmt.Println()

	return nil
}
