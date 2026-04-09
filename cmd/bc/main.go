// Package main is the entry point for the bc CLI.
package main

import (
	"os"

	"github.com/rpuneet/bc/internal/cmd"
)

// Version information set by ldflags during build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersionInfo(version, commit, date)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
