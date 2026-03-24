package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/rpuneet/bc/pkg/ui"
)

// bootstrapServerDaemons starts bc-sql and bc-stats during bc init.
func bootstrapServerDaemons(_ string) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := dockerRun(ctx, "bc-sql", []string{
		"-p", "5432:5432",
		"-e", "POSTGRES_PASSWORD=bc",
		"-v", "bc-sql-data:/var/lib/postgresql/data",
		"--restart", "always",
		"bc-bcsql:latest",
	}); err != nil {
		fmt.Printf("  %s bc-sql: %v\n", ui.YellowText("warning"), err)
	}

	if err := dockerRun(ctx, "bc-stats", []string{
		"-p", "5433:5432",
		"-e", "POSTGRES_PASSWORD=bc",
		"-v", "bc-stats-data:/var/lib/postgresql/data",
		"--restart", "always",
		"bc-bcstats:latest",
	}); err != nil {
		fmt.Printf("  %s bc-stats: %v\n", ui.YellowText("warning"), err)
	}

	fmt.Printf("\n  %s databases ready\n\n", ui.GreenText("ok"))
}
