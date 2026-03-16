// Package main is the entry point for the bcd daemon server.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/workspace"
	"github.com/rpuneet/bc/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:9374", "Listen address")
	wsDir := flag.String("workspace", "", "Workspace root directory (default: auto-detect)")
	flag.Parse()

	// Resolve workspace directory.
	var ws *workspace.Workspace
	var err error
	if *wsDir != "" {
		ws, err = workspace.Load(*wsDir)
	} else {
		ws, err = workspace.Find(".")
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cfg := server.Config{
		Addr: *addr,
		Dir:  ws.RootDir,
	}

	srv, err := server.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating server: %v\n", err)
		os.Exit(1)
	}

	stateDir := ws.StateDir()

	// Write PID and info files.
	if err := daemon.WritePID(stateDir, os.Getpid()); err != nil {
		fmt.Fprintf(os.Stderr, "error writing pid file: %v\n", err)
		os.Exit(1)
	}
	if err := daemon.WriteInfo(stateDir, *addr); err != nil {
		_ = daemon.RemovePID(stateDir)
		fmt.Fprintf(os.Stderr, "error writing info file: %v\n", err)
		os.Exit(1)
	}

	// Set up signal handling for graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Info("received signal", "signal", sig)
		cancel()
	}()

	log.Info("bcd starting", "addr", *addr, "workspace", ws.RootDir)

	if err := srv.Start(ctx); err != nil {
		log.Error("server error", "error", err)
	}

	// Cleanup PID and info files on shutdown.
	_ = daemon.RemovePID(stateDir)
	_ = daemon.RemoveInfo(stateDir)
	log.Info("bcd stopped")
}
