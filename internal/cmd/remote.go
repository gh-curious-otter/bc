package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/remote"
	"github.com/rpuneet/bc/pkg/ui"
)

// Remote commands for remote agent execution
// Issue #1219: Phase 3 Enterprise - Remote agent execution

var remoteCmd = &cobra.Command{
	Use:   "remote",
	Short: "Manage remote hosts for agent execution",
	Long: `Configure and manage remote hosts for distributed agent execution.

Remote execution allows bc to spawn and manage agents on remote machines
via SSH, enabling distributed workloads across multiple servers.

Commands:
  bc remote add <name> <user@host>   Add a remote host
  bc remote list                     List configured hosts
  bc remote remove <name>            Remove a remote host
  bc remote test <name>              Test connection to host

Examples:
  bc remote add dev deploy@dev.example.com
  bc remote add prod deploy@prod.example.com --port 2222 --key ~/.ssh/deploy_key
  bc remote list
  bc remote test dev
  bc remote remove dev

Once configured, spawn agents remotely:
  bc agent spawn eng-01 --remote dev`,
}

var remoteAddCmd = &cobra.Command{
	Use:   "add <name> <user@host>",
	Short: "Add a remote host",
	Long: `Add a new remote host for agent execution.

The host specification should be in the format user@hostname.
SSH key-based authentication is recommended.

Examples:
  bc remote add dev deploy@dev.example.com
  bc remote add prod admin@192.168.1.100 --port 2222
  bc remote add staging deploy@staging.example.com --key ~/.ssh/staging_key`,
	Args: cobra.ExactArgs(2),
	RunE: runRemoteAdd,
}

var remoteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured remote hosts",
	RunE:  runRemoteList,
}

var remoteRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a remote host",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoteRemove,
}

var remoteTestCmd = &cobra.Command{
	Use:   "test <name>",
	Short: "Test connection to a remote host",
	Long: `Test SSH connection to a remote host and check bc availability.

This verifies:
- SSH connectivity
- Authentication
- bc installation on remote host

Example:
  bc remote test dev`,
	Args: cobra.ExactArgs(1),
	RunE: runRemoteTest,
}

var remoteExecCmd = &cobra.Command{
	Use:   "exec <name> <command>",
	Short: "Execute a command on a remote host",
	Long: `Execute a command on a remote host via SSH.

Example:
  bc remote exec dev "bc agent list"
  bc remote exec dev "ls -la"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runRemoteExec,
}

// Flags
var (
	remotePort int
	remoteKey  string
)

func init() {
	remoteCmd.AddCommand(remoteAddCmd)
	remoteCmd.AddCommand(remoteListCmd)
	remoteCmd.AddCommand(remoteRemoveCmd)
	remoteCmd.AddCommand(remoteTestCmd)
	remoteCmd.AddCommand(remoteExecCmd)

	remoteAddCmd.Flags().IntVar(&remotePort, "port", 22, "SSH port")
	remoteAddCmd.Flags().StringVar(&remoteKey, "key", "", "Path to SSH private key")

	rootCmd.AddCommand(remoteCmd)
}

func getRemoteManager() (*remote.Manager, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}

	mgr := remote.NewManager(ws.StateDir())
	if err := mgr.Load(); err != nil {
		return nil, fmt.Errorf("failed to load remote config: %w", err)
	}

	return mgr, nil
}

func runRemoteAdd(_ *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	userHost := args[1]

	// Parse user@host
	parts := strings.SplitN(userHost, "@", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid host format: expected user@hostname, got %q", userHost)
	}

	user := parts[0]
	hostname := parts[1]

	// Check for port in hostname (host:port)
	port := remotePort
	if idx := strings.LastIndex(hostname, ":"); idx != -1 {
		portStr := hostname[idx+1:]
		hostname = hostname[:idx]
		if p, err := strconv.Atoi(portStr); err == nil {
			port = p
		}
	}

	host := &remote.Host{
		Name:     name,
		Hostname: hostname,
		Port:     port,
		User:     user,
		KeyPath:  remoteKey,
	}

	if err := mgr.Add(host); err != nil {
		return err
	}

	fmt.Printf("✓ Added remote host: %s (%s@%s:%d)\n", name, user, hostname, port)
	return nil
}

func runRemoteList(cmd *cobra.Command, _ []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	hosts := mgr.List()

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(hosts)
	}

	if len(hosts) == 0 {
		fmt.Println()
		fmt.Println("  No remote hosts configured.")
		fmt.Println()
		fmt.Println("  Add a host with: bc remote add <name> <user@hostname>")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Remote Hosts"))
	fmt.Println("  " + strings.Repeat("─", 60))
	fmt.Println()

	for _, h := range hosts {
		stateIcon := "○"
		stateColor := ui.DimText
		switch h.State {
		case remote.StateOnline:
			stateIcon = "●"
			stateColor = ui.GreenText
		case remote.StateOffline:
			stateIcon = "✗"
			stateColor = ui.RedText
		}

		fmt.Printf("  %s %s\n", stateColor(stateIcon), ui.CyanText(h.Name))
		fmt.Printf("      %s@%s:%d\n", h.User, h.Hostname, h.Port)
		if h.KeyPath != "" {
			fmt.Printf("      Key: %s\n", h.KeyPath)
		}
		if h.Error != "" {
			fmt.Printf("      %s\n", ui.RedText("Error: "+h.Error))
		}
		fmt.Println()
	}

	return nil
}

func runRemoteRemove(_ *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	if err := mgr.Remove(name); err != nil {
		return err
	}

	fmt.Printf("✓ Removed remote host: %s\n", name)
	return nil
}

func runRemoteTest(cmd *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	fmt.Printf("Testing connection to %s...\n", name)

	result, err := mgr.Test(context.Background(), name)
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
		return enc.Encode(result)
	}

	if result.Success {
		fmt.Printf("✓ Connection successful (latency: %v)\n", result.Latency)
		if result.BCVersion != "" && result.BCVersion != "not installed" {
			fmt.Printf("  bc version: %s\n", result.BCVersion)
		} else {
			fmt.Printf("  %s\n", ui.YellowText("bc not installed on remote host"))
		}
	} else {
		fmt.Printf("✗ Connection failed: %s\n", result.Error)
	}

	return nil
}

func runRemoteExec(cmd *cobra.Command, args []string) error {
	mgr, err := getRemoteManager()
	if err != nil {
		return err
	}

	name := args[0]
	command := strings.Join(args[1:], " ")

	output, err := mgr.Exec(context.Background(), name, command)
	if err != nil {
		if output != "" {
			fmt.Print(output)
		}
		return err
	}

	fmt.Print(output)
	return nil
}
