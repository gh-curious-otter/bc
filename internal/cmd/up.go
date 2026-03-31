package cmd

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/gh-curious-otter/bc/pkg/log"
	"github.com/gh-curious-otter/bc/pkg/ui"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start bc services",
	Long: `Start bc-db, bc-<id>-daemon, and bc-playwright in Docker.

Examples:
  bc up
  bc up --port 9000
  bc up --port 8080 --workspace /path/to/workspace`,
	RunE: runUp,
}

var (
	upPort      string
	upWorkspace string
)

func init() {
	upCmd.Flags().StringVar(&upPort, "port", "9374", "Host port for bcd")
	upCmd.Flags().StringVar(&upWorkspace, "workspace", "", "Workspace directory (defaults to current workspace)")
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

	fmt.Printf("Starting bc in %s\n\n", ws.RootDir)

	id := wsID(ws.RootDir)

	// Shared volume for screenshots and temp files between containers
	const sharedVolume = "bc-shared-tmp"

	// 1. bc-db — unified database (TimescaleDB = Postgres + hypertables)
	if err := dockerRun(ctx, "bc-db", []string{
		"-p", "5432:5432",
		"-e", "POSTGRES_PASSWORD=bc",
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
		"-e", "DATABASE_URL=postgres://bc:bc@host.docker.internal:5432/bc",
		"-e", "STATS_DATABASE_URL=postgres://bc:bc@host.docker.internal:5432/bc",
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

	// 5. bc-playwright — Playwright MCP server with Chromium + noVNC
	// --init prevents zombie processes, --ipc=host prevents Chromium OOM crashes
	// See: https://playwright.dev/docs/docker
	if err := dockerRun(ctx, "bc-playwright", []string{
		"--init",
		"--ipc=host",
		"-p", "3100:3000",
		"-p", "6080:6080",
		"-v", sharedVolume + ":/tmp/bc-shared",
		"-e", "DISPLAY=:99",
		"--restart", "always",
		"bc-playwright:latest",
	}); err != nil {
		// Non-fatal — Playwright is optional
		fmt.Printf("  %s playwright skipped: %v\n", ui.YellowText("note"), err)
	}

	fmt.Println()
	fmt.Printf("  %s bc workspace ready\n", ui.GreenText("ok"))
	fmt.Printf("  bcd:        http://%s\n", addr)
	fmt.Println("  bc-db:      localhost:5432")
	fmt.Println("  playwright: http://localhost:6080 (noVNC), MCP localhost:3100")
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
