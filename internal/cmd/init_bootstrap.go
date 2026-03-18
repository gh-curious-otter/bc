package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

// bootstrapServerDaemons starts bcd (bc-bcd:latest) as a Docker-managed
// workspace daemon. bcd uses SQLite (bc.db) by default — no Postgres needed.
// Failures are non-fatal — a warning is printed if Docker is unavailable or
// the bc-bcd:latest image has not been built yet.
func bootstrapServerDaemons(rootDir string) {
	mgr, err := daemon.NewManager(rootDir)
	if err != nil {
		log.Debug("daemon manager unavailable; skipping server bootstrap", "error", err)
		return
	}
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start bcd with workspace mounted — uses SQLite (bc.db) by default
	fmt.Print("  Starting bcd server... ")
	_, bcdErr := mgr.Run(ctx, daemon.RunOptions{
		Name:    "bcd",
		Runtime: daemon.RuntimeDocker,
		Image:   "bc-bcd:latest",
		Ports:   []string{"9374:9374"},
		Volumes: []string{
			rootDir + ":/workspace",
			"/var/run/docker.sock:/var/run/docker.sock",
		},
		Restart: "always",
		Detach:  true,
	})
	if bcdErr != nil {
		fmt.Println(ui.YellowText("✗ (image not found — run: make build-bcd-image)"))
		log.Debug("bcd start failed", "error", bcdErr)
		return
	}
	fmt.Println(ui.GreenText("✓"))

	fmt.Println()
	fmt.Printf("  %s bc workspace ready at http://localhost:9374\n", ui.GreenText("✓"))
	fmt.Println()
}
