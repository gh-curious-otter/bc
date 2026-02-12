package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/tui/runtime"
)

var homeCmd = &cobra.Command{
	Use:   "home",
	Short: "Open the bc TUI dashboard",
	Long: `Open the interactive bc TUI dashboard.

The TUI provides a visual interface for managing agents, viewing channels,
monitoring costs, and more.

Requirements:
  - Bun runtime installed (bun.sh)
  - TUI package built (make build-tui)

Examples:
  bc home              # Open TUI dashboard
  bc home --debug      # Open with debug output`,
	RunE: runHome,
}

var homeDebug bool

func init() {
	rootCmd.AddCommand(homeCmd)
	homeCmd.Flags().BoolVar(&homeDebug, "debug", false, "Enable debug output")
}

func runHome(cmd *cobra.Command, args []string) error {
	// Get workspace root
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("failed to get workspace: %w", err)
	}

	// Create bridge configuration
	cfg := runtime.BridgeConfig{
		TUIDir:        "tui",
		EntryPoint:    "src/index.tsx",
		WorkspaceRoot: ws.RootDir,
	}

	// Create and start the Ink bridge
	bridge, err := runtime.NewInkBridge(cfg)
	if err != nil {
		return fmt.Errorf("failed to create TUI bridge: %w\n\nMake sure the TUI is built: make build-tui", err)
	}

	if err := bridge.Start(); err != nil {
		return fmt.Errorf("failed to start TUI: %w", err)
	}
	defer func() { _ = bridge.Close() }()

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Send initial workspace state to TUI
	initialState := map[string]interface{}{
		"type":      "init",
		"workspace": ws.Config.Name,
		"rootDir":   ws.RootDir,
	}
	if err := bridge.SendJSON(initialState); err != nil {
		return fmt.Errorf("failed to send initial state: %w", err)
	}

	// Event loop - handle TUI events and signals
	done := make(chan error, 1)
	go func() {
		done <- bridge.Wait()
	}()

	select {
	case sig := <-sigChan:
		if homeDebug {
			fmt.Fprintf(os.Stderr, "Received signal: %v\n", sig)
		}
		return nil
	case err := <-done:
		if err != nil {
			return fmt.Errorf("TUI exited with error: %w", err)
		}
		return nil
	}
}
