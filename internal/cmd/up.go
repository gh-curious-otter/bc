package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gh-curious-otter/bc/pkg/log"
	"github.com/gh-curious-otter/bc/pkg/ui"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc server",
	Long: `Start the bc server (API, web UI, MCP, agent management).

By default the server runs in the foreground (for Docker/Railway).
Use -d to run as a background daemon.

Examples:
  bc up                              # Foreground (Docker/Railway)
  bc up -d                           # Background daemon
  bc up --addr 0.0.0.0:9374         # Custom listen address
  bc up --workspace /path/to/ws     # Explicit workspace`,
	RunE: runUp,
}

var (
	upAddr      string
	upWorkspace string
	upDaemon    bool
	upCORS      string
	upAPIKey    string
)

func init() {
	upCmd.Flags().StringVar(&upAddr, "addr", "127.0.0.1:9374", "Listen address (host:port)")
	upCmd.Flags().StringVar(&upWorkspace, "workspace", "", "Workspace directory (defaults to current workspace)")
	upCmd.Flags().BoolVarP(&upDaemon, "daemon", "d", false, "Run as background daemon")
	upCmd.Flags().StringVar(&upCORS, "cors-origin", "*", "CORS allowed origin")
	upCmd.Flags().StringVar(&upAPIKey, "api-key", os.Getenv("BC_API_KEY"), "API key for Bearer token auth (or set BC_API_KEY)")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, _ []string) error {
	wsRoot := upWorkspace
	if wsRoot == "" {
		ws, err := getWorkspace()
		if err != nil {
			return errNotInWorkspace(err)
		}
		wsRoot = ws.RootDir
	} else {
		// Validate the workspace path
		if _, err := workspace.Load(wsRoot); err != nil {
			return fmt.Errorf("cannot load workspace at %s: %w", wsRoot, err)
		}
	}

	// Read server config from settings.json for defaults
	if ws, loadErr := workspace.Load(wsRoot); loadErr == nil && ws.Config != nil {
		// Use settings.json addr if --addr wasn't explicitly set
		if !cmd.Flags().Changed("addr") {
			host := ws.Config.Server.Host
			if host == "" {
				host = "127.0.0.1"
			}
			port := 9374
			if ws.Config.Server.Port > 0 {
				port = ws.Config.Server.Port
			}
			upAddr = fmt.Sprintf("%s:%d", host, port)
		}
	}

	// Daemon mode: re-exec bc up in background
	if upDaemon {
		return runUpDaemon(wsRoot)
	}

	// Foreground mode: run server directly
	fmt.Printf("Starting bc server in %s\n", wsRoot)
	fmt.Printf("  addr: %s\n\n", upAddr)

	return RunServer(upAddr, wsRoot, upCORS, upAPIKey)
}

// runUpDaemon starts bc up in the background by re-executing the bc binary.
// Logs go to .bc/bcd.log, PID to .bc/bcd.pid.
func runUpDaemon(wsRoot string) error {
	ws, err := workspace.Load(wsRoot)
	if err != nil {
		return fmt.Errorf("cannot load workspace: %w", err)
	}

	// Check if already running
	pidPath := filepath.Join(ws.StateDir(), "bcd.pid")
	if pidData, readErr := os.ReadFile(pidPath); readErr == nil { //nolint:gosec // controlled workspace path
		pid := strings.TrimSpace(string(pidData))
		checkCmd := exec.CommandContext(context.Background(), "kill", "-0", pid) //nolint:gosec // trusted
		if checkCmd.Run() == nil {
			fmt.Printf("  bc server already running (PID %s)\n", pid)
			fmt.Printf("  http://%s\n", upAddr)
			return nil
		}
	}

	// Find our own binary to re-exec
	selfPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find bc binary: %w", err)
	}

	logPath := filepath.Join(ws.StateDir(), "bcd.log")

	// Build args for foreground mode (without -d)
	args := []string{
		"up",
		"--addr", upAddr,
		"--workspace", wsRoot,
		"--cors-origin", upCORS,
	}
	if upAPIKey != "" {
		args = append(args, "--api-key", upAPIKey)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) //nolint:gosec // controlled path
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	cmd := exec.CommandContext(context.Background(), selfPath, args...) //nolint:gosec // trusted binary
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Dir = wsRoot
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start bc server: %w", err)
	}
	_ = logFile.Close()

	// Write PID file
	if writeErr := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0600); writeErr != nil {
		log.Warn("failed to write PID file", "path", pidPath, "error", writeErr)
	}

	// Detach -- don't wait for the process
	_ = cmd.Process.Release()

	fmt.Printf("  %s bc server started (PID %d)\n", ui.GreenText("ok"), cmd.Process.Pid)
	fmt.Printf("  http://%s\n", upAddr)
	fmt.Printf("  logs: %s\n", logPath)
	fmt.Printf("  pid:  %s\n", pidPath)
	fmt.Println()

	return nil
}

// wsID returns a short workspace hash for container naming.
func wsID(path string) string {
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", h[:3])
}

// dockerRun starts a container if not already running.
func dockerRun(ctx context.Context, name string, args []string) error {
	// Check if already running
	//nolint:gosec // trusted
	out, _ := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name).Output()
	if strings.TrimSpace(string(out)) == "true" {
		fmt.Printf("  %s %s (already running)\n", ui.GreenText("ok"), name)
		return nil
	}

	// Remove stale container
	//nolint:gosec // trusted
	_ = exec.CommandContext(ctx, "docker", "rm", "-f", name).Run()

	// Start
	fmt.Printf("  Starting %s... ", name)
	cmdArgs := append([]string{"run", "-d", "--name", name}, args...)
	//nolint:gosec // trusted
	cmd := exec.CommandContext(ctx, "docker", cmdArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Println(ui.YellowText(fmt.Sprintf("failed (%v)", err)))
		log.Debug("docker run failed", "name", name, "output", string(output))
		return fmt.Errorf("container %s: %w", name, err)
	}
	fmt.Println(ui.GreenText("started"))
	return nil
}

// waitPG polls pg_isready inside a container.
func waitPG(ctx context.Context, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		//nolint:gosec // trusted
		if exec.CommandContext(ctx, "docker", "exec", name, "pg_isready", "-U", "bc").Run() == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("timeout waiting for %s", name)
}

// waitHTTP polls a health endpoint.
func waitHTTP(ctx context.Context, addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := fmt.Sprintf("http://%s/health", addr)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if resp, err := http.DefaultClient.Do(req); err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("timeout")
}
