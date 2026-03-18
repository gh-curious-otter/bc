package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/ui"
)

// bootstrapServerDaemons starts bcdb (postgres:17) and bcd (bc-bcd:latest) as
// Docker-managed workspace daemons. It is called automatically by bc init.
// Failures are non-fatal — a warning is printed if Docker is unavailable or the
// bc-bcd:latest image has not been built yet.
func bootstrapServerDaemons(rootDir string) {
	mgr, err := daemon.NewManager(rootDir)
	if err != nil {
		log.Debug("daemon manager unavailable; skipping server bootstrap", "error", err)
		return
	}
	defer func() { _ = mgr.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start bcdb (Postgres)
	fmt.Print("  Starting bcdb (postgres:17)... ")
	_, dbErr := mgr.Run(ctx, daemon.RunOptions{
		Name:    "bcdb",
		Runtime: daemon.RuntimeDocker,
		Image:   "postgres:17",
		Ports:   []string{"5432:5432"},
		Env: []string{
			"POSTGRES_DB=bc",
			"POSTGRES_USER=bc",
			"POSTGRES_PASSWORD=bc",
		},
		Restart: "always",
		Detach:  true,
	})
	if dbErr != nil {
		fmt.Println(ui.YellowText("✗ (Docker unavailable — run manually: bc daemon run --name bcdb --runtime docker --image postgres:17)"))
		log.Debug("bcdb start failed", "error", dbErr)
		return
	}
	fmt.Println(ui.GreenText("✓"))

	// Start bcd
	fmt.Print("  Starting bcd server...       ")
	_, bcdErr := mgr.Run(ctx, daemon.RunOptions{
		Name:    "bcd",
		Runtime: daemon.RuntimeDocker,
		Image:   "bc-bcd:latest",
		Ports:   []string{"9374:9374"},
		Env:     []string{"DATABASE_URL=postgres://bc:bc@localhost:5432/bc"},
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
