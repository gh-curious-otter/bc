package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var roleCmd = &cobra.Command{
	Use:     "role",
	Aliases: []string{"rl"},
	Short:   "Manage agent roles in the workspace",
	Long: `Manage agent roles via the bcd daemon.

Examples:
  bc role list                                      # List all roles
  bc role show engineer                             # Show engineer role details
  bc role list --json                               # JSON output`,
}

var roleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all roles in the workspace",
	RunE:  runRoleList,
}

var roleShowCmd = &cobra.Command{
	Use:   "show <role>",
	Short: "Show role definition and metadata",
	Args:  cobra.ExactArgs(1),
	RunE:  runRoleShow,
}

func init() {
	roleCmd.AddCommand(roleListCmd)
	roleCmd.AddCommand(roleShowCmd)

	roleListCmd.Flags().Bool("json", false, "Output in JSON format")
	roleListCmd.Flags().Bool("mcp", false, "Show MCP server associations column")

	roleShowCmd.ValidArgsFunction = CompleteRoleNames

	rootCmd.AddCommand(roleCmd)
}

func runRoleList(cmd *cobra.Command, args []string) error {
	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	roles, err := c.Roles.List(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to list roles: %w", err)
	}

	// Get agent counts from agents API
	agentCounts := make(map[string]int)
	agents, agentErr := c.Agents.List(cmd.Context())
	if agentErr == nil {
		for _, ag := range agents {
			agentCounts[ag.Role]++
		}
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		return json.NewEncoder(os.Stdout).Encode(roles)
	}

	if len(roles) == 0 {
		fmt.Println("No roles defined")
		return nil
	}

	showMCP, _ := cmd.Flags().GetBool("mcp")

	type roleRow struct {
		name        string
		description string
		mcpServers  string
		agents      int
	}
	rows := make([]roleRow, 0, len(roles))
	maxNameLen := 4
	maxDescLen := 11

	for name, role := range roles {
		desc := role.Name
		if desc == "" {
			desc = name
		}
		// Use prompt first line as description if no explicit one
		if role.Prompt != "" {
			lines := strings.SplitN(role.Prompt, "\n", 3)
			for _, l := range lines {
				l = strings.TrimSpace(strings.TrimLeft(l, "#"))
				if l != "" {
					desc = l
					break
				}
			}
		}
		if len(desc) > 40 {
			desc = desc[:37] + "..."
		}
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
		if len(desc) > maxDescLen {
			maxDescLen = len(desc)
		}

		mcpStr := ""
		if len(role.MCPServers) > 0 {
			mcpStr = strings.Join(role.MCPServers, ", ")
			if len(mcpStr) > 30 {
				mcpStr = mcpStr[:27] + "..."
			}
		}

		rows = append(rows, roleRow{
			name:        name,
			description: desc,
			mcpServers:  mcpStr,
			agents:      agentCounts[name],
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].name < rows[j].name
	})

	if showMCP {
		fmt.Printf("%-*s  %-6s  %-*s  %s\n", maxNameLen, "ROLE", "AGENTS", maxDescLen, "DESCRIPTION", "MCP")
		fmt.Println(strings.Repeat("-", maxNameLen+maxDescLen+20))
	} else {
		fmt.Printf("%-*s  %-6s  %s\n", maxNameLen, "ROLE", "AGENTS", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", maxNameLen+maxDescLen+10))
	}

	for _, r := range rows {
		if showMCP {
			fmt.Printf("%-*s  %-6d  %-*s  %s\n", maxNameLen, r.name, r.agents, maxDescLen, r.description, r.mcpServers)
		} else {
			fmt.Printf("%-*s  %-6d  %s\n", maxNameLen, r.name, r.agents, r.description)
		}
	}

	fmt.Printf("\n%d role(s) defined\n", len(rows))
	return nil
}

func runRoleShow(cmd *cobra.Command, args []string) error {
	c, err := newDaemonClient(cmd.Context())
	if err != nil {
		return err
	}

	roleName := args[0]
	role, err := c.Roles.Get(cmd.Context(), roleName)
	if err != nil {
		return fmt.Errorf("role %q: %w", roleName, err)
	}

	fmt.Printf("Role: %s\n", role.Name)
	if len(role.MCPServers) > 0 {
		fmt.Printf("MCP Servers: %s\n", strings.Join(role.MCPServers, ", "))
	}
	if len(role.Secrets) > 0 {
		fmt.Printf("Secrets: %s\n", strings.Join(role.Secrets, ", "))
	}
	if len(role.Plugins) > 0 {
		fmt.Printf("Plugins: %s\n", strings.Join(role.Plugins, ", "))
	}
	if len(role.Commands) > 0 {
		cmds := make([]string, 0, len(role.Commands))
		for k := range role.Commands {
			cmds = append(cmds, "/"+k)
		}
		fmt.Printf("Commands: %s\n", strings.Join(cmds, ", "))
	}
	if len(role.Rules) > 0 {
		rules := make([]string, 0, len(role.Rules))
		for k := range role.Rules {
			rules = append(rules, k)
		}
		fmt.Printf("Rules: %s\n", strings.Join(rules, ", "))
	}
	if role.PromptCreate != "" {
		fmt.Printf("On Create: %s\n", strings.TrimSpace(role.PromptCreate))
	}
	if role.PromptStart != "" {
		fmt.Printf("On Start: %s\n", strings.TrimSpace(role.PromptStart))
	}

	fmt.Println()
	fmt.Println("Prompt:")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println(role.Prompt)

	return nil
}
