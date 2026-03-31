package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gh-curious-otter/bc/pkg/log"
	"github.com/gh-curious-otter/bc/pkg/ui"
	"github.com/gh-curious-otter/bc/pkg/workspace"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop bc services",
	Long: `Stop bc-<id>-daemon, bc-stats, and bc-sql Docker containers.

Examples:
  bc down
  bc down --workspace /path/to/workspace`,
	RunE: runDown,
}

var downWorkspace string

func init() {
	downCmd.Flags().StringVar(&downWorkspace, "workspace", "", "Workspace directory (defaults to current workspace)")
	rootCmd.AddCommand(downCmd)
}

func runDown(cmd *cobra.Command, _ []string) error {
	var ws *workspace.Workspace
	var err error
	if downWorkspace != "" {
		ws, err = workspace.Load(downWorkspace)
		if err != nil {
			return fmt.Errorf("cannot load workspace at %s: %w", downWorkspace, err)
		}
	} else {
		ws, err = getWorkspace()
		if err != nil {
			return errNotInWorkspace(err)
		}
	}

	ctx := cmd.Context()

	fmt.Printf("Stopping bc in %s\n\n", ws.RootDir)

	// Use docker compose if docker-compose.yml exists
	composePath := filepath.Join(ws.RootDir, "docker-compose.yml")
	if _, statErr := os.Stat(composePath); statErr == nil {
		fmt.Println("  Stopping via docker compose...")
		c := exec.CommandContext(ctx, "docker", "compose", "-f", composePath, "down") //nolint:gosec
		c.Dir = ws.RootDir
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if runErr := c.Run(); runErr != nil {
			return fmt.Errorf("docker compose down failed: %w", runErr)
		}
		fmt.Println(ui.GreenText("  bc services stopped via docker compose"))
		return nil
	}

	// Fallback: stop individual containers
	id := wsID(ws.RootDir)
	daemonName := fmt.Sprintf("bc-%s-daemon", id)

	var stopped int
	for _, name := range []string{daemonName, "bc-stats", "bc-sql"} {
		//nolint:gosec // trusted
		out, _ := exec.CommandContext(ctx, "docker", "inspect", "-f", "{{.State.Running}}", name).Output()
		if strings.TrimSpace(string(out)) != "true" {
			continue
		}
		fmt.Printf("  Stopping %s... ", name)
		//nolint:gosec // trusted
		if output, stopErr := exec.CommandContext(ctx, "docker", "stop", name).CombinedOutput(); stopErr != nil {
			fmt.Println(ui.YellowText(fmt.Sprintf("failed (%v)", stopErr)))
			log.Debug("docker stop failed", "name", name, "output", string(output))
			continue
		}
		fmt.Println(ui.GreenText("stopped"))
		stopped++
	}

	if stopped == 0 {
		fmt.Println("  No services running")
	} else {
		fmt.Println()
		fmt.Printf("  %s Stopped %d service(s)\n", ui.GreenText("ok"), stopped)
	}
	return nil
}
