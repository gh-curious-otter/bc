package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/ui"
)

// MCP commands for Model Context Protocol integration
// Issue #1212: Phase 4 Ecosystem - MCP integration

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol (MCP) integration",
	Long: `Manage MCP server and client connections for AI tool integration.

MCP (Model Context Protocol) enables bc to expose workspace resources and
connect to external AI tools through a standardized protocol.

Server Mode:
  bc mcp server start    Start MCP server for this workspace
  bc mcp server stop     Stop MCP server
  bc mcp server status   Show server status

Client Mode:
  bc mcp connect <url>   Connect to an MCP server
  bc mcp disconnect      Disconnect from server
  bc mcp tools list      List available tools
  bc mcp tools call      Call an MCP tool

Resources exposed by bc MCP server:
  agents://     Agent states and metadata
  channels://   Channel messages
  costs://      Cost tracking data
  memory://     Agent memory and learnings

Examples:
  bc mcp server start                    # Start server on default port
  bc mcp server start --port 9090        # Start on custom port
  bc mcp connect stdio://claude          # Connect via stdio
  bc mcp tools list                      # List available tools
  bc mcp tools call get_agent_status --args '{"agent":"eng-01"}'`,
}

var mcpServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage MCP server",
	Long: `Start, stop, and manage the MCP server for this workspace.

The MCP server exposes bc workspace resources to AI clients:
- Agent states and metadata (agents://)
- Channel messages (channels://)
- Cost tracking data (costs://)
- Agent memory (memory://)`,
}

var mcpServerStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start MCP server",
	Long: `Start the MCP server for this workspace.

The server listens for MCP client connections and exposes workspace
resources through the standard MCP protocol.

Examples:
  bc mcp server start                    # Start with default settings
  bc mcp server start --port 9090        # Custom port
  bc mcp server start --transport stdio  # Use stdio transport`,
	RunE: runMCPServerStart,
}

var mcpServerStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop MCP server",
	RunE:  runMCPServerStop,
}

var mcpServerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show MCP server status",
	RunE:  runMCPServerStatus,
}

var mcpConnectCmd = &cobra.Command{
	Use:   "connect <server-url>",
	Short: "Connect to an MCP server",
	Long: `Connect to an MCP server as a client.

Supported URL schemes:
  stdio://<command>      Connect via stdio to a process
  http://<host>:<port>   Connect via HTTP
  https://<host>:<port>  Connect via HTTPS
  sse://<host>:<port>    Connect via Server-Sent Events

Examples:
  bc mcp connect stdio://claude
  bc mcp connect http://localhost:9090
  bc mcp connect sse://mcp.example.com:8080`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPConnect,
}

var mcpDisconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect from MCP server",
	RunE:  runMCPDisconnect,
}

var mcpToolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage MCP tools",
}

var mcpToolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available MCP tools",
	RunE:  runMCPToolsList,
}

var mcpToolsCallCmd = &cobra.Command{
	Use:   "call <tool-name>",
	Short: "Call an MCP tool",
	Long: `Invoke an MCP tool with the given arguments.

Examples:
  bc mcp tools call get_agent_status --args '{"agent":"eng-01"}'
  bc mcp tools call send_message --args '{"channel":"general","message":"hello"}'`,
	Args: cobra.ExactArgs(1),
	RunE: runMCPToolsCall,
}

var mcpResourcesCmd = &cobra.Command{
	Use:   "resources",
	Short: "Manage MCP resources",
}

var mcpResourcesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available MCP resources",
	RunE:  runMCPResourcesList,
}

var mcpResourcesReadCmd = &cobra.Command{
	Use:   "read <uri>",
	Short: "Read an MCP resource",
	Args:  cobra.ExactArgs(1),
	RunE:  runMCPResourcesRead,
}

// Flags
var (
	mcpServerPort      int
	mcpServerTransport string
	mcpToolArgs        string
)

func init() {
	// Server commands
	mcpServerCmd.AddCommand(mcpServerStartCmd)
	mcpServerCmd.AddCommand(mcpServerStopCmd)
	mcpServerCmd.AddCommand(mcpServerStatusCmd)

	mcpServerStartCmd.Flags().IntVar(&mcpServerPort, "port", 9090, "Server port (for HTTP transport)")
	mcpServerStartCmd.Flags().StringVar(&mcpServerTransport, "transport", "stdio", "Transport type: stdio, http, sse")

	// Tools commands
	mcpToolsCmd.AddCommand(mcpToolsListCmd)
	mcpToolsCmd.AddCommand(mcpToolsCallCmd)

	mcpToolsCallCmd.Flags().StringVar(&mcpToolArgs, "args", "{}", "Tool arguments as JSON")

	// Resources commands
	mcpResourcesCmd.AddCommand(mcpResourcesListCmd)
	mcpResourcesCmd.AddCommand(mcpResourcesReadCmd)

	// Main MCP command
	mcpCmd.AddCommand(mcpServerCmd)
	mcpCmd.AddCommand(mcpConnectCmd)
	mcpCmd.AddCommand(mcpDisconnectCmd)
	mcpCmd.AddCommand(mcpToolsCmd)
	mcpCmd.AddCommand(mcpResourcesCmd)

	rootCmd.AddCommand(mcpCmd)
}

func runMCPServerStart(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	fmt.Printf("Starting MCP server for workspace: %s\n", ws.Config.Name)
	fmt.Printf("Transport: %s\n", mcpServerTransport)

	switch mcpServerTransport {
	case mcp.TransportStdio:
		fmt.Println("Running in stdio mode...")
		fmt.Println("Server is ready to accept MCP client connections via stdio")
		// TODO: Implement stdio server loop
		return fmt.Errorf("stdio server not yet implemented")

	case mcp.TransportHTTP:
		fmt.Printf("Starting HTTP server on port %d...\n", mcpServerPort)
		// TODO: Implement HTTP server
		return fmt.Errorf("HTTP server not yet implemented")

	case mcp.TransportSSE:
		fmt.Printf("Starting SSE server on port %d...\n", mcpServerPort)
		// TODO: Implement SSE server
		return fmt.Errorf("SSE server not yet implemented")

	default:
		return fmt.Errorf("unsupported transport: %s (use stdio, http, or sse)", mcpServerTransport)
	}
}

func runMCPServerStop(cmd *cobra.Command, args []string) error {
	// TODO: Implement server stop
	fmt.Println("Stopping MCP server...")
	return fmt.Errorf("not yet implemented")
}

func runMCPServerStatus(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	// TODO: Get actual server status
	status := struct {
		Workspace string `json:"workspace"`
		Transport string `json:"transport,omitempty"`
		Port      int    `json:"port,omitempty"`
		Running   bool   `json:"running"`
	}{
		Workspace: ws.Config.Name,
		Running:   false, // TODO: Check actual status
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(status)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("MCP Server Status"))
	fmt.Println("  " + strings.Repeat("─", 40))
	fmt.Printf("  Workspace: %s\n", status.Workspace)
	fmt.Printf("  Running:   %v\n", status.Running)
	if status.Running {
		fmt.Printf("  Transport: %s\n", status.Transport)
		if status.Port > 0 {
			fmt.Printf("  Port:      %d\n", status.Port)
		}
	}
	fmt.Println()

	return nil
}

func runMCPConnect(cmd *cobra.Command, args []string) error {
	serverURL := args[0]

	fmt.Printf("Connecting to MCP server: %s\n", serverURL)
	// TODO: Implement client connection
	return fmt.Errorf("not yet implemented")
}

func runMCPDisconnect(cmd *cobra.Command, args []string) error {
	fmt.Println("Disconnecting from MCP server...")
	// TODO: Implement disconnect
	return fmt.Errorf("not yet implemented")
}

func runMCPToolsList(cmd *cobra.Command, args []string) error {
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	// Define bc's built-in tools that would be exposed via MCP
	tools := []mcp.Tool{
		{
			Name:        "get_agent_status",
			Description: "Get the current status of an agent",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"agent":{"type":"string","description":"Agent name"}},"required":["agent"]}`),
		},
		{
			Name:        "list_agents",
			Description: "List all agents in the workspace",
			InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
		},
		{
			Name:        "send_message",
			Description: "Send a message to an agent or channel",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"target":{"type":"string","description":"Agent name or channel name"},"message":{"type":"string","description":"Message to send"}},"required":["target","message"]}`),
		},
		{
			Name:        "get_channel_history",
			Description: "Get message history from a channel",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"channel":{"type":"string","description":"Channel name"},"limit":{"type":"integer","description":"Max messages to return","default":10}},"required":["channel"]}`),
		},
		{
			Name:        "get_cost_summary",
			Description: "Get cost summary for the workspace",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"since":{"type":"string","description":"Duration (e.g., '7d', '24h')","default":"7d"}}}`),
		},
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(tools)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Available MCP Tools"))
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()

	for _, tool := range tools {
		fmt.Printf("  %s\n", ui.CyanText(tool.Name))
		if tool.Description != "" {
			fmt.Printf("    %s\n", tool.Description)
		}
		fmt.Println()
	}

	return nil
}

func runMCPToolsCall(cmd *cobra.Command, args []string) error {
	toolName := args[0]

	var toolArgs map[string]any
	if err := json.Unmarshal([]byte(mcpToolArgs), &toolArgs); err != nil {
		return fmt.Errorf("invalid JSON arguments: %w", err)
	}

	fmt.Printf("Calling tool: %s\n", toolName)
	fmt.Printf("Arguments: %v\n", toolArgs)
	// TODO: Implement tool call
	return fmt.Errorf("not yet implemented")
}

func runMCPResourcesList(cmd *cobra.Command, args []string) error {
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	// Define bc's built-in resources
	resources := []mcp.Resource{
		{
			URI:         "agents://",
			Name:        "Agents",
			Description: "Agent states and metadata",
			MimeType:    "application/json",
		},
		{
			URI:         "channels://",
			Name:        "Channels",
			Description: "Channel messages and history",
			MimeType:    "application/json",
		},
		{
			URI:         "costs://",
			Name:        "Costs",
			Description: "Cost tracking and budget data",
			MimeType:    "application/json",
		},
		{
			URI:         "memory://",
			Name:        "Memory",
			Description: "Agent memory and learnings",
			MimeType:    "application/json",
		},
		{
			URI:         "events://",
			Name:        "Events",
			Description: "Event log and audit trail",
			MimeType:    "application/json",
		},
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(resources)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Available MCP Resources"))
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()

	for _, res := range resources {
		fmt.Printf("  %s\n", ui.CyanText(res.URI))
		fmt.Printf("    Name: %s\n", res.Name)
		if res.Description != "" {
			fmt.Printf("    Description: %s\n", res.Description)
		}
		fmt.Println()
	}

	return nil
}

func runMCPResourcesRead(cmd *cobra.Command, args []string) error {
	uri := args[0]

	fmt.Printf("Reading resource: %s\n", uri)
	// TODO: Implement resource read
	return fmt.Errorf("not yet implemented")
}
