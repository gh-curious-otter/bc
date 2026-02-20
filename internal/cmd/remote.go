package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/remote"
	"github.com/rpuneet/bc/pkg/ui"
)

// Remote commands for remote agent execution
// Issue #1219: Phase 3 Enterprise - Remote agent execution

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote hosts and agents",
	Long: `Execute agents on remote machines via SSH.

Remote execution enables distributing agent workloads across multiple
machines for enterprise deployments and large-scale orchestration.

Host Management:
  bc remote add <name> <host>    Add a remote host
  bc remote remove <name>        Remove a remote host
  bc remote list                 List configured hosts
  bc remote test <name>          Test host connectivity

Remote Agents:
  bc remote spawn <agent> --host <host> --role <role>
  bc remote stop <agent>
  bc remote agents               List remote agents

Examples:
  bc remote add dev-server dev.example.com --user deploy --key ~/.ssh/id_rsa
  bc remote test dev-server
  bc remote spawn eng-01 --host dev-server --role engineer
  bc remote agents --host dev-server`,
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <hostname>",
	Short: "Add a remote host",
	Long: `Add a new remote host for agent execution.

Arguments:
  name      Alias for the host (e.g., dev-server, prod-1)
  hostname  SSH hostname or IP address

Flags:
  --port, -p       SSH port (default: 22)
  --user, -u       SSH username
  --key, -k        Path to SSH private key
  --description    Description of the host

Examples:
  bc remote add dev-server dev.example.com --user deploy
  bc remote add prod-1 10.0.0.5 --port 2222 --key ~/.ssh/prod_key`,
	Args: cobra.ExactArgs(2),
	RunE: runRemoteAdd,
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote host",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoteRemove,
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured hosts",
	RunE:  runRemoteList,
}

var remoteTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test host connectivity",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoteTest,
}

var remoteSpawnCmd = &cobra.Command{
	Use:   "spawn <agent-name>",
	Short: "Spawn an agent on a remote host",
	Long: `Spawn a new agent on a remote host.

The agent will be created via SSH on the specified host and run
in an isolated environment.

Examples:
  bc remote spawn eng-01 --host dev-server --role engineer
  bc remote spawn qa-01 --host test-server --role engineer`,
	Args: cobra.ExactArgs(1),
	RunE: runRemoteSpawn,
}

var remoteStopCmd = &cobra.Command{
	Use:   "stop <agent-name>",
	Short: "Stop a remote agent",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoteStop,
}

var remoteAgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List remote agents",
	RunE:  runRemoteAgents,
}

var remoteSSHCmd = &cobra.Command{
	Use:   "ssh <name>",
	Short: "Show SSH command for a host",
	Long: `Display the SSH command to connect to a remote host.

Useful for manual debugging or connecting to inspect agents.

Examples:
  bc remote ssh dev-server
  $(bc remote ssh dev-server)  # Execute directly`,
	Args: cobra.ExactArgs(1),
	RunE: runRemoteSSH,
}

// Flags
var (
	remotePort        int
	remoteUser        string
	remoteKey         string
	remoteDescription string
	remoteHost        string
	remoteRole        string
	remoteFilterHost  string
)

func init() {
	// Add host flags
	remoteAddCmd.Flags().IntVarP(&remotePort, "port", "p", 22, "SSH port")
	remoteAddCmd.Flags().StringVarP(&remoteUser, "user", "u", "", "SSH username")
	remoteAddCmd.Flags().StringVarP(&remoteKey, "key", "k", "", "Path to SSH private key")
	remoteAddCmd.Flags().StringVar(&remoteDescription, "description", "", "Host description")

	// Spawn flags
	remoteSpawnCmd.Flags().StringVar(&remoteHost, "host", "", "Remote host name (required)")
	remoteSpawnCmd.Flags().StringVar(&remoteRole, "role", "engineer", "Agent role")
	_ = remoteSpawnCmd.MarkFlagRequired("host")

	// Agents filter
	remoteAgentsCmd.Flags().StringVar(&remoteFilterHost, "host", "", "Filter by host")

	// Register subcommands
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteTestCmd)
	remoteCmd.AddCommand(remoteSpawnCmd)
	remoteCmd.AddCommand(remoteStopCmd)
	remoteCmd.AddCommand(remoteAgentsCmd)
	remoteCmd.AddCommand(remoteSSHCmd)

	rootCmd.AddCommand(remoteCmd)
}

func getRemoteManager() (*remote.Manager, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}

	mgr := remote.NewManager(ws.RootDir)
	if err := mgr.Load(); err != nil {
		return nil, fmt.Errorf("failed to load remote config: %w", err)
	}

	return mgr, nil
}

func runRemoteAdd(cmd *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	hostname := args[1]

	// Get current user if not specified
	user := remoteUser
	if user == "" {
		user = os.Getenv("USER")
	}

	host, err := mgr.AddHost(name, hostname, remotePort, user, remoteKey, remoteDescription)
	if err != nil {
		return err
	}

	cmd.Printf("Added remote host %q (%s@%s:%d)\n", host.Name, host.User, host.Hostname, host.Port)
	cmd.Println()
	cmd.Printf("Test connectivity with: bc remote test %s\n", name)

	return nil
}

func runRemoteRemove(_ *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	if err := mgr.RemoveHost(name); err != nil {
		return err
	}

	fmt.Printf("Removed remote host %q\n", name)
	return nil
}

func runRemoteList(cmd *cobra.Command, _ []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	hosts := mgr.ListHosts()

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(hosts)
	}

	if len(hosts) == 0 {
		fmt.Println()
		fmt.Println("  No remote hosts configured.")
		fmt.Println()
		fmt.Println("  Add a host with: bc remote add <name> <hostname>")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Remote Hosts"))
	fmt.Println("  " + strings.Repeat("─", 60))
	fmt.Println()

	for _, h := range hosts {
		statusIcon := "○"
		statusColor := ui.DimText
		switch h.Status {
		case remote.StatusConnected:
			statusIcon = "●"
			statusColor = ui.GreenText
		case remote.StatusUnreachable:
			statusIcon = "✗"
			statusColor = ui.RedText
		case remote.StatusError:
			statusIcon = "!"
			statusColor = ui.YellowText
		}

		fmt.Printf("  %s %s\n", statusColor(statusIcon), ui.CyanText(h.Name))
		fmt.Printf("      %s@%s:%d\n", h.User, h.Hostname, h.Port)
		if h.Description != "" {
			fmt.Printf("      %s\n", ui.DimText(h.Description))
		}

		// Show agent count on this host
		agents := mgr.ListAgentsByHost(h.Name)
		if len(agents) > 0 {
			fmt.Printf("      Agents: %d running\n", len(agents))
		}
		fmt.Println()
	}

	return nil
}

func runRemoteTest(cmd *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	cmd.Printf("Testing connection to %s...\n", name)

	if err := mgr.TestConnection(context.Background(), name); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	host, _ := mgr.GetHost(name)
	cmd.Printf("✓ Connected to %s (%s@%s:%d)\n", name, host.User, host.Hostname, host.Port)

	return nil
}

func runRemoteSpawn(cmd *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	agentName := args[0]

	cmd.Printf("Spawning agent %q on %q...\n", agentName, remoteHost)

	agent, err := mgr.SpawnAgent(context.Background(), agentName, remoteHost, remoteRole)
	if err != nil {
		return fmt.Errorf("failed to spawn agent: %w", err)
	}

	cmd.Printf("✓ Spawned agent %q on %q (role: %s)\n", agent.Name, agent.Host, agent.Role)
	cmd.Println()
	cmd.Printf("View agents with: bc remote agents --host %s\n", remoteHost)

	return nil
}

func runRemoteStop(_ *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	if err := mgr.StopAgent(context.Background(), name); err != nil {
		return err
	}

	fmt.Printf("✓ Stopped remote agent %q\n", name)
	return nil
}

func runRemoteAgents(cmd *cobra.Command, _ []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	var agents []*remote.RemoteAgent
	if remoteFilterHost != "" {
		agents = mgr.ListAgentsByHost(remoteFilterHost)
	} else {
		agents = mgr.ListAgents()
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(agents)
	}

	if len(agents) == 0 {
		fmt.Println()
		if remoteFilterHost != "" {
			fmt.Printf("  No agents running on %s.\n", remoteFilterHost)
		} else {
			fmt.Println("  No remote agents running.")
		}
		fmt.Println()
		fmt.Println("  Spawn an agent with: bc remote spawn <name> --host <host>")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Remote Agents"))
	fmt.Println("  " + strings.Repeat("─", 60))
	fmt.Println()

	for _, a := range agents {
		statusIcon := "●"
		statusColor := ui.GreenText
		switch a.Status {
		case "stopped":
			statusIcon = "○"
			statusColor = ui.DimText
		case "starting":
			statusIcon = "◐"
			statusColor = ui.YellowText
		case "error":
			statusIcon = "✗"
			statusColor = ui.RedText
		}

		fmt.Printf("  %s %s\n", statusColor(statusIcon), ui.CyanText(a.Name))
		fmt.Printf("      Host: %s  Role: %s\n", a.Host, a.Role)
		fmt.Printf("      Started: %s\n", a.StartedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

func runRemoteSSH(_ *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	sshCmd, err := mgr.SSHCommand(name)
	if err != nil {
		return err
	}

	fmt.Println(sshCmd)
	return nil
}
