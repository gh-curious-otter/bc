package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

// bootstrapServerDaemons starts bcd in a tmux session.
// bcd uses SQLite (bc.db) by default — no Postgres needed.
// Failures are non-fatal — a warning is printed if the binary is unavailable.
func bootstrapServerDaemons(rootDir string) {
	mgr, err := daemon.NewManager(rootDir)
	if err != nil {
		log.Debug("daemon manager unavailable; skipping server bootstrap", "error", err)
		return
	}
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start bcd in a tmux session using the local binary
	fmt.Print("  Starting bcd server... ")
	_, bcdErr := mgr.Run(ctx, daemon.RunOptions{
		Name:    "bcd",
		Runtime: daemon.RuntimeTmux,
		Cmd:     "bc daemon start",
		Restart: "on-failure",
		Detach:  true,
	})
	if bcdErr != nil {
		fmt.Println(ui.YellowText(fmt.Sprintf("✗ (failed to start bcd: %v)", bcdErr)))
		fmt.Println("    Ensure 'bc' is in your PATH and tmux is installed.")
		return
	}
	fmt.Println(ui.GreenText("✓"))

	fmt.Println()
	fmt.Printf("  %s bc workspace ready at http://localhost:9374\n", ui.GreenText("✓"))
	fmt.Println()
}
