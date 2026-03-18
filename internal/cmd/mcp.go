package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/ui"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Manage MCP server configurations",
	Long: `Manage Model Context Protocol (MCP) server configurations.

MCP servers provide tools and resources to AI agents. Configurations are
stored per-workspace and can be referenced by roles.

Examples:
  bc mcp list                                     # List all MCP servers
  bc mcp add github --command npx --args "@modelcontextprotocol/server-github"
  bc mcp add sqlite --command npx --args "@modelcontextprotocol/server-sqlite,/path/to/db"
  bc mcp add remote --transport sse --url "https://api.example.com/mcp"
  bc mcp add github --command npx --env "GITHUB_TOKEN=tok_123"
  bc mcp show github                              # Show server details
  bc mcp remove github                            # Remove a server
  bc mcp disable github                           # Disable a server
  bc mcp enable github                            # Re-enable a server`,
}

var mcpAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add an MCP server configuration",
	Long: `Add a new MCP server configuration to the workspace.

For stdio transport (default), specify --command and optionally --args.
For SSE transport, specify --transport sse and --url.

Environment variables can be passed with --env as KEY=VALUE pairs.

Examples:
  bc mcp add github --command npx --args "@modelcontextprotocol/server-github"
  bc mcp add db --command npx --args "@modelcontextprotocol/server-sqlite,/tmp/test.db"
  bc mcp add remote --transport sse --url "https://api.example.com/mcp"
  bc mcp add github --command npx --env 'GITHUB_TOKEN=${secret:GITHUB_TOKEN}' --env "OWNER=me"

Use ${secret:NAME} references for sensitive values (see 'bc secret set').`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPAdd,
}

var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List MCP server configurations",
	RunE:  runMCPList,
}

var mcpShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show MCP server configuration details",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPShow,
}

var mcpRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove an MCP server configuration",
	Args:    cobra.ExactArgs(1),
	RunE:    runMCPRemove,
}

var mcpEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable an MCP server configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPEnable,
}

var mcpDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable an MCP server configuration",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPDisable,
}

// Flags for mcp add.
var (
	mcpAddTransport string
	mcpAddCommand   string
	mcpAddArgs      string
	mcpAddURL       string
	mcpAddEnv       []string
)

func init() {
	mcpAddCmd.Flags().StringVar(&mcpAddTransport, "transport", "stdio", "Transport type (stdio or sse)")
	mcpAddCmd.Flags().StringVar(&mcpAddCommand, "command", "", "Command to run (for stdio transport)")
	mcpAddCmd.Flags().StringVar(&mcpAddArgs, "args", "", "Comma-separated arguments")
	mcpAddCmd.Flags().StringVar(&mcpAddURL, "url", "", "Server URL (for sse transport)")
	mcpAddCmd.Flags().StringArrayVar(&mcpAddEnv, "env", nil, "Environment variables (KEY=VALUE, repeatable)")

	mcpCmd.AddCommand(mcpAddCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpShowCmd)
	mcpCmd.AddCommand(mcpRemoveCmd)
	mcpCmd.AddCommand(mcpEnableCmd)
	mcpCmd.AddCommand(mcpDisableCmd)
	rootCmd.AddCommand(mcpCmd)
}

func openMCPStore() (*mcp.Store, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}
	return mcp.NewStore(ws.RootDir)
}

func runMCPAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !validIdentifier(name) {
		return fmt.Errorf("server name %q contains invalid characters (use letters, numbers, dash, underscore)", name)
	}

	transport := mcp.Transport(mcpAddTransport)

	// Parse env vars
	env := make(map[string]string)
	for _, e := range mcpAddEnv {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			return fmt.Errorf("invalid env format %q (expected KEY=VALUE)", e)
		}
		env[k] = v
	}

	// Parse args
	var serverArgs []string
	if mcpAddArgs != "" {
		serverArgs = strings.Split(mcpAddArgs, ",")
	}

	cfg := &mcp.ServerConfig{
		Name:      name,
		Transport: transport,
		Command:   mcpAddCommand,
		Args:      serverArgs,
		URL:       mcpAddURL,
		Env:       env,
		Enabled:   true,
	}

	store, err := openMCPStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.Add(cfg); err != nil {
		return err
	}

	fmt.Printf("Added MCP server %q (%s)\n", name, transport)
	return nil
}

func runMCPList(cmd *cobra.Command, args []string) error {
	store, err := openMCPStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	configs, err := store.List()
	if err != nil {
		return err
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		response := struct {
			Servers []*mcp.ServerConfig `json:"servers"`
		}{Servers: configs}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

	if len(configs) == 0 {
		ui.Warning("No MCP servers configured")
		ui.BlankLine()
		ui.Info("Run 'bc mcp add <name> --command <cmd>' to add one")
		return nil
	}

	table := ui.NewTable("NAME", "TRANSPORT", "COMMAND/URL", "ENABLED")
	for _, cfg := range configs {
		target := cfg.Command
		if cfg.Transport == mcp.TransportSSE {
			target = cfg.URL
		}
		enabled := "yes"
		if !cfg.Enabled {
			enabled = "no"
		}
		table.AddRow(cfg.Name, string(cfg.Transport), target, enabled)
	}
	table.Print()
	return nil
}

func runMCPShow(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !validIdentifier(name) {
		return fmt.Errorf("server name %q contains invalid characters", name)
	}

	store, err := openMCPStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	cfg, err := store.Get(name)
	if err != nil {
		return err
	}
	if cfg == nil {
		return fmt.Errorf("mcp server %q not found (use 'bc mcp list' to see available servers)", name)
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(cfg)
	}

	enabled := "yes"
	if !cfg.Enabled {
		enabled = "no"
	}

	ui.SimpleTable(
		"Name", cfg.Name,
		"Transport", string(cfg.Transport),
		"Enabled", enabled,
	)

	if cfg.Command != "" {
		ui.SimpleTable("Command", cfg.Command)
	}
	if len(cfg.Args) > 0 {
		ui.SimpleTable("Args", strings.Join(cfg.Args, ", "))
	}
	if cfg.URL != "" {
		ui.SimpleTable("URL", cfg.URL)
	}
	if len(cfg.Env) > 0 {
		pairs := make([]string, 0, len(cfg.Env))
		for k := range cfg.Env {
			pairs = append(pairs, k+"=***")
		}
		ui.SimpleTable("Env", strings.Join(pairs, ", "))
	}

	return nil
}

func runMCPRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !validIdentifier(name) {
		return fmt.Errorf("server name %q contains invalid characters", name)
	}

	store, err := openMCPStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.Remove(name); err != nil {
		return err
	}

	fmt.Printf("Removed MCP server %q\n", name)
	return nil
}

func runMCPEnable(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !validIdentifier(name) {
		return fmt.Errorf("server name %q contains invalid characters", name)
	}

	store, err := openMCPStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.SetEnabled(name, true); err != nil {
		return err
	}

	fmt.Printf("Enabled MCP server %q\n", name)
	return nil
}

func runMCPDisable(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !validIdentifier(name) {
		return fmt.Errorf("server name %q contains invalid characters", name)
	}

	store, err := openMCPStore()
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	if err := store.SetEnabled(name, false); err != nil {
		return err
	}

	fmt.Printf("Disabled MCP server %q\n", name)
	return nil
}
