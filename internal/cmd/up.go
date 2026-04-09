package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	bcdb "github.com/gh-curious-otter/bc/pkg/db"
	"github.com/gh-curious-otter/bc/pkg/log"
	"github.com/gh-curious-otter/bc/pkg/ui"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc services",
	Long: `Start bc services (Docker containers or local daemon).

Examples:
  bc up                    # Start Docker containers (db + bcd)
  bc up -d                 # Start bcd as local background daemon (no Docker)
  bc up --port 9000        # Custom port
  bc up -d --port 8080     # Local daemon on custom port`,
	RunE: runUp,
}

var (
	upPort      string
	upWorkspace string
	upDaemon    bool
)

func init() {
	upCmd.Flags().StringVar(&upPort, "port", "9374", "Host port for bcd")
	upCmd.Flags().StringVar(&upWorkspace, "workspace", "", "Workspace directory (defaults to current workspace)")
	upCmd.Flags().BoolVarP(&upDaemon, "daemon", "d", false, "Run bcd as local background process (no Docker)")
	rootCmd.AddCommand(upCmd)
}

func runUp(cmd *cobra.Command, _ []string) error {
	var ws *workspace.Workspace
	var err error
	if upWorkspace != "" {
		ws, err = workspace.Load(upWorkspace)
		if err != nil {
			return fmt.Errorf("cannot load workspace at %s: %w", upWorkspace, err)
		}
	} else {
		ws, err = getWorkspace()
		if err != nil {
			return errNotInWorkspace(err)
		}
	}

	ctx := cmd.Context()

	// Daemon mode: run bcd as local background process (no Docker)
	if upDaemon {
		return runUpDaemon(ws)
	}

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)

	id := wsID(ws.RootDir)

	// Shared volume for screenshots and temp files between containers
	const sharedVolume = "bc-shared-tmp"

	// 1. bc-db — unified database (TimescaleDB = Postgres + hypertables)
	if err := dockerRun(ctx, "bc-db", []string{
		"-p", "5432:5432",
		"-e", "POSTGRES_PASSWORD=" + bcdb.DefaultPassword(),
		"-v", "bc-db-data:/var/lib/postgresql/data",
		"--restart", "always",
		"bc-bcdb:latest",
	}); err != nil {
		return fmt.Errorf("bc-db failed to start: %w", err)
	}

	// 2. Wait for database
	fmt.Print("  Waiting for database... ")
	if err := waitPG(ctx, "bc-db", 30*time.Second); err != nil {
		return fmt.Errorf("bc-db not ready: %w", err)
	}
	fmt.Println(ui.GreenText("ready"))

	// 3. bc-<id>-daemon with docker.sock + workspace mount
	daemonName := fmt.Sprintf("bc-%s-daemon", id)
	daemonArgs := []string{
		"-p", upPort + ":9374",
		"-v", ws.RootDir + ":/workspace",
		"-v", "/var/run/docker.sock:/var/run/docker.sock",
		"-v", sharedVolume + ":/tmp/bc-shared",
		"-e", "DATABASE_URL=postgres://bc:" + bcdb.DefaultPassword() + "@host.docker.internal:5432/bc",
		"-e", "STATS_DATABASE_URL=postgres://bc:" + bcdb.DefaultPassword() + "@host.docker.internal:5432/bc",
		"-e", "BC_HOST_WORKSPACE=" + ws.RootDir,
		"--restart", "always",
	}
	// Linux needs explicit host.docker.internal mapping (macOS/Windows have it built in).
	if runtime.GOOS == "linux" {
		daemonArgs = append(daemonArgs, "--add-host=host.docker.internal:host-gateway")
	}
	daemonArgs = append(daemonArgs,
		"bc-daemon:latest",
		"--addr", "0.0.0.0:9374",
		"--workspace", "/workspace",
	)
	if err := dockerRun(ctx, daemonName, daemonArgs); err != nil {
		fmt.Printf("  %s daemon failed to start: %v\n", ui.YellowText("warning"), err)
	}

	// 4. Wait for bcd
	addr := fmt.Sprintf("127.0.0.1:%s", upPort)
	fmt.Print("  Waiting for bcd... ")
	if waitHTTP(ctx, addr, 30*time.Second) != nil {
		fmt.Println(ui.YellowText("slow"))
	} else {
		fmt.Println(ui.GreenText("ready"))
	}

	fmt.Println()
	fmt.Printf("  %s bc workspace ready\n", ui.GreenText("ok"))
	fmt.Printf("  bcd:        http://%s\n", addr)
	fmt.Println("  bc-db:      localhost:5432")
	fmt.Println()

	return nil
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

// wsID returns a short workspace hash for container naming.
func wsID(path string) string {
	h := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", h[:3])
}

// runUpDaemon starts bcd as a local background process using nohup.
// Logs go to .bc/bcd.log, PID to .bc/bcd.pid.
func runUpDaemon(ws *workspace.Workspace) error {
	// Find bcd binary
	bcdPath, err := exec.LookPath("bcd")
	if err != nil {
		// Try in same directory as bc binary
		selfPath, _ := os.Executable()
		bcdPath = filepath.Join(filepath.Dir(selfPath), "bcd")
		if _, statErr := os.Stat(bcdPath); statErr != nil {
			return fmt.Errorf("bcd binary not found in PATH or next to bc binary")
		}
	}

	// Check if already running
	pidPath := filepath.Join(ws.StateDir(), "bcd.pid")
	if pidData, readErr := os.ReadFile(pidPath); readErr == nil {
		pid := strings.TrimSpace(string(pidData))
		// Check if process is still alive
		checkCmd := exec.Command("kill", "-0", pid) //nolint:gosec // trusted
		if checkCmd.Run() == nil {
			host := "127.0.0.1"
			port := upPort
			if ws.Config != nil {
				if ws.Config.Server.Host != "" {
					host = ws.Config.Server.Host
				}
				if ws.Config.Server.Port > 0 && upPort == "9374" {
					port = fmt.Sprintf("%d", ws.Config.Server.Port)
				}
			}
			fmt.Printf("  bcd already running (PID %s)\n", pid)
			fmt.Printf("  http://%s:%s\n", host, port)
			return nil
		}
	}

	logPath := filepath.Join(ws.StateDir(), "bcd.log")

	// Read server config from settings.json
	host := "127.0.0.1"
	port := upPort
	if ws.Config != nil {
		if ws.Config.Server.Host != "" {
			host = ws.Config.Server.Host
		}
		if ws.Config.Server.Port > 0 && upPort == "9374" {
			// Only use settings.json port if --port flag wasn't explicitly set
			port = fmt.Sprintf("%d", ws.Config.Server.Port)
		}
	}

	// Start bcd in background
	args := []string{bcdPath, "--addr", host + ":" + port, "--workspace", ws.RootDir}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) //nolint:gosec // controlled path
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec // trusted binary
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	cmd.Dir = ws.RootDir
	cmd.Env = os.Environ()

	// Check storage config — if timescale, auto-start bc-db and set DATABASE_URL.
	useTimescale := ws.Config != nil && (ws.Config.Storage.Default == "timescale" || ws.Config.Storage.Default == "sql")
	if useTimescale {
		// Build DSN from settings.json
		ts := ws.Config.Storage.Timescale
		dbHost := ts.Host
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := ts.Port
		if dbPort == 0 {
			dbPort = 5432
		}
		dbUser := ts.User
		if dbUser == "" {
			dbUser = "bc"
		}
		dbPass := ts.Password
		if dbPass == "" {
			dbPass = bcdb.DefaultPassword()
		}
		dbName := ts.Database
		if dbName == "" {
			dbName = "bc"
		}
		dbURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", dbUser, dbPass, dbHost, dbPort, dbName)

		// Auto-start bc-db container if it exists
		//nolint:gosec // trusted
		dbCheck, _ := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", "bc-db").Output()
		dbRunning := strings.TrimSpace(string(dbCheck)) == "true"
		if !dbRunning {
			//nolint:gosec // trusted
			dbExists, _ := exec.Command("docker", "inspect", "-f", "{{.State.Status}}", "bc-db").Output()
			if strings.TrimSpace(string(dbExists)) != "" {
				fmt.Print("  Starting bc-db... ")
				//nolint:gosec // trusted
				if startErr := exec.Command("docker", "start", "bc-db").Run(); startErr == nil {
					if waitErr := waitPG(context.Background(), "bc-db", 15*time.Second); waitErr == nil {
						dbRunning = true
						fmt.Println(ui.GreenText("ready"))
					} else {
						fmt.Println(ui.YellowText("started but not ready"))
					}
				} else {
					fmt.Println(ui.YellowText("failed"))
				}
			}
		}
		if dbRunning {
			cmd.Env = append(cmd.Env,
				"DATABASE_URL="+dbURL,
				"STATS_DATABASE_URL="+dbURL,
			)
			fmt.Printf("  storage: timescale (%s:%d)\n", dbHost, dbPort)
		} else {
			fmt.Printf("  %s storage=timescale but bc-db not available, bcd will read settings.json\n", ui.YellowText("warning"))
		}
	} else {
		fmt.Printf("  storage: sqlite\n")
	}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return fmt.Errorf("start bcd: %w", err)
	}
	_ = logFile.Close()

	// Write PID file
	if writeErr := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0600); writeErr != nil {
		log.Warn("failed to write PID file", "path", pidPath, "error", writeErr)
	}

	// Detach — don't wait for the process
	_ = cmd.Process.Release()

	fmt.Printf("  %s bcd started (PID %d)\n", ui.GreenText("ok"), cmd.Process.Pid)
	fmt.Printf("  bcd:  http://%s:%s\n", host, port)
	fmt.Printf("  logs: %s\n", logPath)
	fmt.Printf("  pid:  %s\n", pidPath)
	fmt.Println()

	return nil
}
