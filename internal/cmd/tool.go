package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/provider"
	"github.com/rpuneet/bc/pkg/ui"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage AI tool providers",
	Long: `View and check AI tool providers available for agent spawning.

Examples:
  bc tool list          # Show all tools with status
  bc tool check claude  # Detailed check for a specific tool`,
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured tools and their status",
	Long: `Show all registered AI tool providers with installation status, version, and command.

Examples:
  bc tool list        # Table output
  bc tool list --json # JSON output for scripting`,
	RunE: runToolList,
}

var toolCheckCmd = &cobra.Command{
	Use:   "check <tool>",
	Short: "Check details for a specific tool",
	Long: `Show detailed information about a specific AI tool provider.

Examples:
  bc tool check claude  # Check claude installation
  bc tool check codex   # Check codex installation`,
	Args: cobra.ExactArgs(1),
	RunE: runToolCheck,
}

var toolListJSON bool

func init() {
	toolListCmd.Flags().BoolVar(&toolListJSON, "json", false, "Output as JSON")

	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolCheckCmd)

	toolCheckCmd.ValidArgsFunction = completeToolNames

	rootCmd.AddCommand(toolCmd)
}

func completeToolNames(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	providers := provider.ListProviders()
	names := make([]string, len(providers))
	for i, p := range providers {
		names[i] = p.Name()
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}

type toolInfo struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Command string `json:"command"`
	Path    string `json:"path,omitempty"`
}

func getToolInfo(ctx context.Context, p provider.Provider) toolInfo {
	info := toolInfo{
		Name:    p.Name(),
		Command: p.Command(),
	}

	if p.IsInstalled(ctx) {
		info.Status = "installed"
		info.Version = p.Version(ctx)
		if info.Version == "" {
			info.Version = "-"
		}
		path, err := exec.LookPath(p.Name())
		if err == nil {
			info.Path = path
		}
	} else {
		info.Status = "not found"
		info.Version = "-"
	}

	return info
}

func runToolList(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	providers := provider.ListProviders()
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Name() < providers[j].Name()
	})

	infos := make([]toolInfo, len(providers))
	for i, p := range providers {
		infos[i] = getToolInfo(ctx, p)
	}

	if toolListJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(infos)
	}

	tbl := ui.NewTable("TOOL", "STATUS", "VERSION", "COMMAND")
	for _, info := range infos {
		status := info.Status
		if status == "installed" {
			status = ui.GreenText("installed")
		} else {
			status = ui.DimText("not found")
		}
		tbl.AddRow(info.Name, status, info.Version, info.Command)
	}
	tbl.Print()

	return nil
}

func runToolCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	name := args[0]

	p, err := provider.GetProvider(name)
	if err != nil {
		return fmt.Errorf("unknown tool %q (use 'bc tool list' to see available tools)", name)
	}

	info := getToolInfo(ctx, p)

	var statusDisplay string
	if info.Status == "installed" {
		statusDisplay = ui.GreenText("installed")
	} else {
		statusDisplay = ui.RedText("not found")
	}

	pathDisplay := info.Path
	if pathDisplay == "" {
		pathDisplay = "-"
	}

	ui.SimpleTable(
		"Tool", info.Name,
		"Status", statusDisplay,
		"Version", info.Version,
		"Command", info.Command,
		"Path", pathDisplay,
	)

	return nil
}
