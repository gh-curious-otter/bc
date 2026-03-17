package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/client"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the bcd daemon",
	Long: `Manage the bc daemon (bcd) which serves as the central coordination server.

The daemon manages agents, channels, and workspace state. CLI commands
communicate with the daemon via HTTP.

Examples:
  bc daemon start          # Start daemon in foreground
  bc daemon start -d       # Start daemon in background (daemonized)
  bc daemon stop           # Graceful shutdown
  bc daemon status         # Health check + uptime
  bc daemon logs           # Show daemon logs`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the bcd daemon",
	Long: `Start the bc daemon (bcd).

By default runs in the foreground. Use -d to daemonize.

Examples:
  bc daemon start          # Foreground
  bc daemon start -d       # Background (daemonized)`,
	RunE: runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the bcd daemon",
	Long: `Gracefully stop the bc daemon.

Examples:
  bc daemon stop`,
	RunE: runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long: `Check daemon health, uptime, and connection info.

Examples:
  bc daemon status`,
	RunE: runDaemonStatus,
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show daemon logs",
	Long: `Display daemon log output.

Examples:
  bc daemon logs
  bc daemon logs --tail 50`,
	RunE: runDaemonLogs,
}

var daemonStartDaemonize bool

func init() {
	daemonStartCmd.Flags().BoolVarP(&daemonStartDaemonize, "daemonize", "d", false, "Run in background")

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)

	rootCmd.AddCommand(daemonCmd)
}

func runDaemonStart(cmd *cobra.Command, args []string) error {
	// TODO(#1938): Implement daemon start with HTTP server
	fmt.Println("Starting bcd daemon...")
	fmt.Println("Note: daemon implementation pending (#1938)")
	return nil
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	c := getClient()
	if err := c.Ping(cmd.Context()); err != nil {
		return fmt.Errorf("daemon is not running")
	}
	// TODO(#1938): Implement graceful shutdown
	fmt.Println("Stopping bcd daemon...")
	return nil
}

func runDaemonStatus(cmd *cobra.Command, args []string) error {
	c := getClient()
	if err := c.Ping(cmd.Context()); err != nil {
		fmt.Println("Daemon: not running")
		return nil
	}
	fmt.Println("Daemon: running")
	fmt.Printf("Address: %s\n", c.BaseURL)
	return nil
}

func runDaemonLogs(cmd *cobra.Command, args []string) error {
	// TODO(#1938): Implement log streaming
	fmt.Println("Note: daemon log streaming pending (#1938)")
	return nil
}

// getClient returns an HTTP client for the bcd daemon.
// This replaces getWorkspace() as the primary way commands access state.
func getClient() *client.Client {
	return client.New("")
}
