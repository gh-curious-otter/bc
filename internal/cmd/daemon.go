package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
	"github.com/rpuneet/bc/server"
)

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the bcd daemon server",
	Long: `Manage the bcd daemon server lifecycle.

The bcd daemon provides an HTTP API for managing agents, channels,
costs, and events in a workspace.

Examples:
  bc daemon start              # Start daemon in background
  bc daemon start -f           # Start daemon in foreground
  bc daemon start -a :8080     # Start on custom address
  bc daemon status             # Show daemon status
  bc daemon stop               # Stop the daemon`,
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start bcd daemon",
	Long: `Start the bcd daemon server.

By default, the daemon starts in the background. Use --foreground to run
in the current terminal session.

Examples:
  bc daemon start              # Start in background
  bc daemon start -f           # Start in foreground
  bc daemon start -a :9000     # Custom listen address`,
	RunE: runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop bcd daemon",
	Long: `Stop the running bcd daemon server.

Sends SIGTERM and waits up to 5 seconds for graceful shutdown.
If the daemon does not stop, sends SIGKILL.

Examples:
  bc daemon stop`,
	RunE: runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show daemon status",
	Long: `Show the current status of the bcd daemon.

Displays PID, listen address, uptime, and health check results.

Examples:
  bc daemon status             # Show status
  bc daemon status --json      # JSON output`,
	RunE: runDaemonStatus,
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)

	daemonStartCmd.Flags().StringP("addr", "a", "127.0.0.1:9374", "Listen address")
	daemonStartCmd.Flags().BoolP("foreground", "f", false, "Run in foreground")
}

func runDaemonStart(cmd *cobra.Command, _ []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return err
	}

	stateDir := ws.StateDir()

	if daemon.IsRunning(stateDir) {
		info, infoErr := daemon.ReadInfo(stateDir)
		if infoErr == nil {
			ui.Info("Daemon already running (PID %d, addr %s)", info.PID, info.Addr)
		} else {
			ui.Info("Daemon already running")
		}
		return nil
	}

	addr, err := cmd.Flags().GetString("addr")
	if err != nil {
		return err
	}
	foreground, err := cmd.Flags().GetBool("foreground")
	if err != nil {
		return err
	}

	if foreground {
		return runDaemonForeground(ws.RootDir, stateDir, addr)
	}

	return runDaemonBackground(ws.RootDir, stateDir, addr)
}

func runDaemonForeground(rootDir, stateDir, addr string) error {
	cfg := server.Config{
		Addr: addr,
		Dir:  rootDir,
	}

	srv, err := server.New(cfg)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	if err := daemon.WritePID(stateDir, os.Getpid()); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	if err := daemon.WriteInfo(stateDir, addr); err != nil {
		_ = daemon.RemovePID(stateDir)
		return fmt.Errorf("write info: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	ui.Success("Daemon started (PID %d, addr %s)", os.Getpid(), addr)

	srvErr := srv.Start(ctx)

	_ = daemon.RemovePID(stateDir)
	_ = daemon.RemoveInfo(stateDir)

	if srvErr != nil {
		return fmt.Errorf("server: %w", srvErr)
	}
	return nil
}

func runDaemonBackground(rootDir, stateDir, addr string) error {
	// Find the bcd binary: look next to current binary, then in PATH.
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}
	bcdPath := filepath.Join(filepath.Dir(self), "bcd")
	if _, err := os.Stat(bcdPath); err != nil {
		bcdPath, err = exec.LookPath("bcd")
		if err != nil {
			return fmt.Errorf("bcd binary not found (install it or run with --foreground): %w", err)
		}
	}

	// Open log file for daemon output.
	logPath := filepath.Join(stateDir, "bcd.log")
	// #nosec G304 - path is constructed from workspace state directory
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	// #nosec G204 - bcdPath is resolved from known locations
	proc := exec.Command(bcdPath, "--addr", addr, "--workspace", rootDir)
	proc.Dir = rootDir
	proc.Env = append(os.Environ(), "BC_WORKSPACE="+rootDir)
	proc.Stdout = logFile
	proc.Stderr = logFile

	if err := proc.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start bcd: %w", err)
	}
	_ = logFile.Close()

	log.Debug("bcd process started", "pid", proc.Process.Pid)

	// Wait for health check.
	healthURL := fmt.Sprintf("http://%s/health", addr)
	healthy := false
	deadline := time.Now().Add(5 * time.Second)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL) //nolint:noctx // short-lived health poll
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				healthy = true
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !healthy {
		ui.Warning("Daemon started (PID %d) but health check did not pass", proc.Process.Pid)
		ui.Info("Check logs: %s", logPath)
		return nil
	}

	ui.Success("Daemon started (PID %d, addr %s)", proc.Process.Pid, addr)
	return nil
}

func runDaemonStop(_ *cobra.Command, _ []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return err
	}

	stateDir := ws.StateDir()

	info, err := daemon.ReadInfo(stateDir)
	if err != nil {
		ui.Info("Daemon is not running")
		return nil
	}

	if !daemon.IsRunning(stateDir) {
		// Stale files, clean up.
		_ = daemon.RemovePID(stateDir)
		_ = daemon.RemoveInfo(stateDir)
		ui.Info("Daemon is not running (cleaned up stale files)")
		return nil
	}

	// Send SIGTERM.
	if err := syscall.Kill(info.PID, syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM to PID %d: %w", info.PID, err)
	}

	// Wait up to 5 seconds for the process to exit.
	stopped := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !daemon.IsRunning(stateDir) {
			stopped = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if !stopped {
		log.Warn("daemon did not stop after SIGTERM, sending SIGKILL", "pid", info.PID)
		if err := syscall.Kill(info.PID, syscall.SIGKILL); err != nil {
			return fmt.Errorf("send SIGKILL to PID %d: %w", info.PID, err)
		}
	}

	// Clean up files.
	_ = daemon.RemovePID(stateDir)
	_ = daemon.RemoveInfo(stateDir)

	ui.Success("Daemon stopped (PID %d)", info.PID)
	return nil
}

func runDaemonStatus(cmd *cobra.Command, _ []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return err
	}

	stateDir := ws.StateDir()
	jsonOutput, _ := cmd.Flags().GetBool("json")

	info, err := daemon.ReadInfo(stateDir)
	if err != nil {
		if jsonOutput {
			return writeJSON(cmd, map[string]any{"running": false})
		}
		ui.Info("Daemon is not running")
		return nil
	}

	running := daemon.IsRunning(stateDir)
	if !running {
		// Stale files.
		_ = daemon.RemovePID(stateDir)
		_ = daemon.RemoveInfo(stateDir)
		if jsonOutput {
			return writeJSON(cmd, map[string]any{"running": false})
		}
		ui.Info("Daemon is not running (cleaned up stale files)")
		return nil
	}

	// Try health check.
	healthStatus := "unknown"
	client := &http.Client{Timeout: 2 * time.Second}
	healthURL := fmt.Sprintf("http://%s/health", info.Addr)
	resp, err := client.Get(healthURL) //nolint:noctx // short-lived health check
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			healthStatus = "healthy"
		} else {
			healthStatus = fmt.Sprintf("unhealthy (status %d)", resp.StatusCode)
		}
	} else {
		healthStatus = "unreachable"
	}

	uptime := time.Since(info.StartedAt).Truncate(time.Second)

	if jsonOutput {
		return writeJSON(cmd, map[string]any{
			"running":    true,
			"pid":        info.PID,
			"addr":       info.Addr,
			"started_at": info.StartedAt,
			"uptime":     uptime.String(),
			"health":     healthStatus,
		})
	}

	ui.Header("Daemon Status")
	ui.Println("  PID:        %d", info.PID)
	ui.Println("  Address:    %s", info.Addr)
	ui.Println("  Uptime:     %s", uptime)
	ui.Println("  Health:     %s", healthStatus)
	ui.Println("  Started:    %s", info.StartedAt.Format(time.RFC3339))
	return nil
}

// writeJSON encodes v as JSON to the command's output.
func writeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
