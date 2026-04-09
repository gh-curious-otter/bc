package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

// runHome is called from runRoot when a workspace is found.
// The bc home command has been removed; the TUI opens automatically.

func runHome(cmd *cobra.Command, args []string) error {
	log.Debug("home command started")

	// Find workspace
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Find TUI directory - prefer cwd for worktree support
	// In worktrees, ws.RootDir resolves through symlinked .bc to parent repo,
	// but we want to use the worktree's local tui/ directory if it exists.
	tuiRoot, err := findTUIRoot(ws.RootDir)
	if err != nil {
		return err
	}
	tuiDir := filepath.Join(tuiRoot, "tui")
	tuiEntry := filepath.Join(tuiDir, "dist", "index.js")

	// Check if TUI is built
	if _, statErr := os.Stat(tuiEntry); os.IsNotExist(statErr) {
		log.Debug("TUI not built, checking for source")

		// Check if TUI source exists
		tuiSrc := filepath.Join(tuiDir, "src", "index.tsx")
		if _, srcErr := os.Stat(tuiSrc); os.IsNotExist(srcErr) {
			return fmt.Errorf("TUI not found. Run from the bc repository root")
		}

		// Prompt to build
		fmt.Println("TUI not built. Building now...")
		buildCtx := context.Background()
		buildCmd := exec.CommandContext(buildCtx, "make", "build-tui")
		buildCmd.Dir = tuiRoot
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
		if buildErr := buildCmd.Run(); buildErr != nil {
			return fmt.Errorf("failed to build TUI: %w\nRun 'make build-tui-local' manually", buildErr)
		}
	}

	// Find bun or node
	runtime, err := findJSRuntime()
	if err != nil {
		return err
	}
	log.Debug("using JS runtime", "runtime", runtime)

	// Run the TUI with signal handling
	log.Debug("starting TUI", "entry", tuiEntry)

	// Create context with signal handling for graceful shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// #nosec G204 - runtime is from exec.LookPath, safe to use
	tuiCmd := exec.CommandContext(ctx, runtime, "run", tuiEntry)
	tuiCmd.Dir = tuiRoot
	tuiCmd.Stdin = os.Stdin
	tuiCmd.Stdout = os.Stdout
	tuiCmd.Stderr = os.Stderr

	// Set environment for bc CLI path
	// Get the current executable path so TUI can call bc
	bcBin, _ := os.Executable()
	tuiCmd.Env = append(os.Environ(),
		fmt.Sprintf("BC_ROOT=%s", ws.RootDir),
		fmt.Sprintf("BC_BIN=%s", bcBin),
	)

	return tuiCmd.Run()
}

// findJSRuntime finds bun or node executable.
func findJSRuntime() (string, error) {
	// Prefer bun
	if path, err := exec.LookPath("bun"); err == nil {
		return path, nil
	}

	// Fall back to node
	if path, err := exec.LookPath("node"); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("bun or node not found. Install bun: https://bun.sh")
}

// findTUIRoot finds the directory containing the tui/ folder.
// In worktrees, ws.RootDir resolves through symlinked .bc to the parent repo,
// but we want to use the worktree's local tui/ if it exists.
// Priority: cwd > ws.RootDir
func findTUIRoot(wsRoot string) (string, error) {
	// First, check current working directory
	cwd, err := os.Getwd()
	if err == nil {
		tuiDir := filepath.Join(cwd, "tui")
		if _, statErr := os.Stat(tuiDir); statErr == nil {
			log.Debug("using TUI from cwd", "path", tuiDir)
			return cwd, nil
		}
	}

	// Fall back to workspace root
	tuiDir := filepath.Join(wsRoot, "tui")
	if _, statErr := os.Stat(tuiDir); statErr == nil {
		log.Debug("using TUI from workspace root", "path", tuiDir)
		return wsRoot, nil
	}

	return "", fmt.Errorf("TUI directory not found in cwd (%s) or workspace root (%s)", cwd, wsRoot)
}
