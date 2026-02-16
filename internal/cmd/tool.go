package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/tools"
)

func init() {
	rootCmd.AddCommand(toolCmd)
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolShowCmd)
	toolCmd.AddCommand(toolEnableCmd)
	toolCmd.AddCommand(toolDisableCmd)
	toolCmd.AddCommand(toolExecCmd)

	// Flags
	toolListCmd.Flags().Bool("json", false, "Output as JSON")
	toolListCmd.Flags().Bool("enabled", false, "Show only enabled tools")
	toolShowCmd.Flags().Bool("json", false, "Output as JSON")
}

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage external tool integrations",
	Long: `Manage external tool integrations like GitHub, GitLab, and AI assistants.

Tools are configured in config.toml under [tools.*] sections. Each tool has:
- name: Identifier for the tool
- command: The CLI command to execute
- enabled: Whether the tool is active

Examples:
  bc tool list              List all configured tools
  bc tool show github       Show details for a specific tool
  bc tool enable github     Enable a tool
  bc tool disable gitlab    Disable a tool
  bc tool exec github -- issue list   Execute a tool command`,
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		// Load tools from workspace config
		if err := loadToolsFromConfig(ws); err != nil {
			return fmt.Errorf("failed to load tools: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		enabledOnly, _ := cmd.Flags().GetBool("enabled")

		var toolList []*tools.Tool
		if enabledOnly {
			toolList = tools.ListEnabled()
		} else {
			toolList = tools.List()
		}

		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(toolList)
		}

		if len(toolList) == 0 {
			fmt.Println("No tools configured")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "NAME\tSTATUS\tCOMMAND")
		for _, t := range toolList {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\n", t.Name, t.Status(), t.Command)
		}
		return w.Flush()
	},
}

var toolShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show tool details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		if err := loadToolsFromConfig(ws); err != nil {
			return fmt.Errorf("failed to load tools: %w", err)
		}

		name := args[0]
		tool, ok := tools.Get(name)
		if !ok {
			return fmt.Errorf("tool not found: %s", name)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			return json.NewEncoder(os.Stdout).Encode(tool)
		}

		fmt.Printf("Name:      %s\n", tool.Name)
		fmt.Printf("Command:   %s\n", tool.Command)
		fmt.Printf("Enabled:   %t\n", tool.Enabled)
		fmt.Printf("Status:    %s\n", tool.Status())
		fmt.Printf("Installed: %t\n", tool.IsInstalled())
		if tool.Scope != "" {
			fmt.Printf("Scope:     %s\n", tool.Scope)
		}
		if tool.RateLimit > 0 {
			fmt.Printf("Rate Limit: %d/hour\n", tool.RateLimit)
		}
		if tool.TokenEnv != "" {
			fmt.Printf("Token Env: %s\n", tool.TokenEnv)
		}
		if tool.URL != "" {
			fmt.Printf("URL:       %s\n", tool.URL)
		}
		return nil
	},
}

var toolEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a tool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		if err := loadToolsFromConfig(ws); err != nil {
			return fmt.Errorf("failed to load tools: %w", err)
		}

		name := args[0]
		if err := tools.Enable(name); err != nil {
			return err
		}

		fmt.Printf("Enabled tool: %s\n", name)
		return nil
	},
}

var toolDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a tool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		if err := loadToolsFromConfig(ws); err != nil {
			return fmt.Errorf("failed to load tools: %w", err)
		}

		name := args[0]
		if err := tools.Disable(name); err != nil {
			return err
		}

		fmt.Printf("Disabled tool: %s\n", name)
		return nil
	},
}

var toolExecCmd = &cobra.Command{
	Use:   "exec <name> [-- args...]",
	Short: "Execute a tool command",
	Long: `Execute a command using a configured tool.

Examples:
  bc tool exec github -- issue list
  bc tool exec gh -- pr view 123
  bc tool exec gitlab -- merge-request list`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := getWorkspace()
		if err != nil {
			return err
		}

		if loadErr := loadToolsFromConfig(ws); loadErr != nil {
			return fmt.Errorf("failed to load tools: %w", loadErr)
		}

		name := args[0]
		toolArgs := args[1:]

		ctx := context.Background()
		result, execErr := tools.Exec(ctx, name, toolArgs...)
		if execErr != nil {
			return execErr
		}

		fmt.Print(result.Output)
		if result.Error != nil {
			return fmt.Errorf("command failed with exit code %d: %w", result.ExitCode, result.Error)
		}
		return nil
	},
}

// loadToolsFromConfig loads tools from workspace configuration.
func loadToolsFromConfig(ws interface{}) error {
	// Reset default registry
	tools.DefaultRegistry = tools.NewRegistry()

	// Register built-in tools from config
	// In a full implementation, this would read from ws.Config().Tools
	// For now, we register common tools

	defaultTools := []*tools.Tool{
		{Name: "claude", Command: "claude --dangerously-skip-permissions", Enabled: true},
		{Name: "github", Command: "gh", Enabled: true},
		{Name: "gitlab", Command: "glab", Enabled: false},
		{Name: "jira", Command: "jira", Enabled: false},
		{Name: "codex", Command: "codex --full-auto", Enabled: false},
	}

	for _, t := range defaultTools {
		if err := tools.Register(t); err != nil {
			return err
		}
	}

	return nil
}
