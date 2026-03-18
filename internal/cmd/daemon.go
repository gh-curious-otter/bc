package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/agent"
	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/client"
	"github.com/rpuneet/bc/pkg/daemon"
	"github.com/rpuneet/bc/pkg/log"
	"github.com/rpuneet/bc/pkg/server"
	"github.com/rpuneet/bc/pkg/shutdown"
)

// daemonCmd is the parent for bc daemon subcommands.
var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage workspace processes and the bcd server",
	Long: `Manage long-lived workspace processes (databases, servers, etc.)
and the bcd coordination server.

  bc daemon start          — start the bcd HTTP server
  bc daemon run --name db  — run a workspace process
  bc daemon list           — list running workspace processes
  bc daemon stop [name]    — stop bcd server or a named process
  bc daemon status         — check bcd server health
  bc daemon logs [name]    — view bcd or process logs`,
}

// --- bcd server commands ---

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the bcd daemon",
	Long: `Start the bc coordination daemon (bcd).

bcd is an HTTP server that manages agent, channel, and workspace state.
By default it listens on :4880. Use -d to run in the background.

Examples:
  bc daemon start          # Foreground (blocks)
  bc daemon start -d       # Background (daemonized)`,
	RunE: runDaemonStart,
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop [name]",
	Short: "Stop the bcd server or a named workspace process",
	Long: `Stop the bcd server or a named workspace process.

Without an argument, sends a shutdown signal to the bcd HTTP server.
With a name, stops the named workspace process.

Examples:
  bc daemon stop           # Stop bcd server
  bc daemon stop postgres  # Stop workspace process`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runDaemonStop,
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show bcd server status",
	Long: `Check bcd server health, address, and uptime.

Examples:
  bc daemon status`,
	RunE: runDaemonStatus,
}

var daemonLogsCmd = &cobra.Command{
	Use:   "logs [name]",
	Short: "Show bcd server or process logs",
	Long: `Show logs for the bcd server or a named workspace process.

Without an argument, shows bcd server logs.
With a name, shows the named workspace process logs.

Examples:
  bc daemon logs           # bcd server logs
  bc daemon logs postgres  # workspace process logs`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runDaemonLogs,
}

// --- workspace process commands ---

var daemonRunCmd = &cobra.Command{
	Use:   "run --name <name> --runtime <bash|docker> [options]",
	Short: "Run a named workspace process",
	Long: `Run a long-lived workspace process in a tmux session (bash) or Docker container.

Examples:
  bc daemon run --name api --runtime bash --cmd "go run ./cmd/api"
  bc daemon run --name db --runtime docker --image postgres:17 --port 5432:5432`,
	RunE: runDaemonRun,
}

var daemonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspace processes",
	Long: `List all workspace processes managed by bc daemon.

Examples:
  bc daemon list`,
	Aliases: []string{"ls"},
	RunE:    runDaemonList,
}

var daemonRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a workspace process",
	Long: `Restart a named workspace process using its saved configuration.

Examples:
  bc daemon restart postgres`,
	Args: cobra.ExactArgs(1),
	RunE: runDaemonRestart,
}

var daemonRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a stopped workspace process",
	Long: `Remove a stopped workspace process record. The process must be stopped first.

Examples:
  bc daemon rm postgres`,
	Args: cobra.ExactArgs(1),
	RunE: runDaemonRm,
}

// Flags for bc daemon start
var daemonStartDaemonize bool

// Flags for bc daemon run
var (
	daemonRunName    string
	daemonRunRuntime string
	daemonRunCmd_    string // underscore to avoid conflict with cobra command var
	daemonRunImage   string
	daemonRunPorts   []string
	daemonRunVolumes []string
	daemonRunEnv     []string
	daemonRunEnvFile string
	daemonRunRestart string
	daemonRunDetach  bool
)

// Flags for bc daemon logs
var daemonLogsLines int

func init() {
	// start flags
	daemonStartCmd.Flags().BoolVarP(&daemonStartDaemonize, "daemonize", "d", false, "Run in background (daemonized)")

	// run flags
	daemonRunCmd.Flags().StringVar(&daemonRunName, "name", "", "Process name (required)")
	daemonRunCmd.Flags().StringVar(&daemonRunRuntime, "runtime", "", "Runtime: bash or docker (required)")
	daemonRunCmd.Flags().StringVar(&daemonRunCmd_, "cmd", "", "Command to run (bash runtime)")
	daemonRunCmd.Flags().StringVar(&daemonRunImage, "image", "", "Docker image (docker runtime)")
	daemonRunCmd.Flags().StringArrayVar(&daemonRunPorts, "port", nil, "Port mapping, e.g. 5432:5432 (repeatable)")
	daemonRunCmd.Flags().StringArrayVar(&daemonRunVolumes, "volume", nil, "Volume mount, e.g. /var/run/docker.sock:/var/run/docker.sock (repeatable)")
	daemonRunCmd.Flags().StringArrayVar(&daemonRunEnv, "env", nil, "Env var KEY=VALUE (repeatable)")
	daemonRunCmd.Flags().StringVar(&daemonRunEnvFile, "env-file", "", "File of KEY=VALUE env vars")
	daemonRunCmd.Flags().StringVar(&daemonRunRestart, "restart", "no", "Restart policy: no|always|on-failure")
	daemonRunCmd.Flags().BoolVarP(&daemonRunDetach, "detach", "d", true, "Run in background (default true)")

	// logs flags
	daemonLogsCmd.Flags().IntVar(&daemonLogsLines, "tail", 50, "Number of lines to show")

	daemonCmd.AddCommand(daemonStartCmd)
	daemonCmd.AddCommand(daemonStopCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	daemonCmd.AddCommand(daemonLogsCmd)
	daemonCmd.AddCommand(daemonRunCmd)
	daemonCmd.AddCommand(daemonListCmd)
	daemonCmd.AddCommand(daemonRestartCmd)
	daemonCmd.AddCommand(daemonRmCmd)

	rootCmd.AddCommand(daemonCmd)
}

// --- bcd server handlers ---

func runDaemonStart(cmd *cobra.Command, _ []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	if daemonStartDaemonize {
		return daemonizeStart(ws.RootDir)
	}

	// Set up services
	agentMgr := newAgentManager(ws)
	if loadErr := agentMgr.LoadState(); loadErr != nil {
		log.Warn("failed to load agent state", "error", loadErr)
	}
	agentSvc := agent.NewAgentService(agentMgr, nil, nil)

	chStore, chErr := channel.OpenStore(ws.RootDir)
	if chErr != nil {
		log.Warn("failed to open channel store", "error", chErr)
		chStore = channel.NewStore(ws.RootDir)
	}
	if loadErr := chStore.Load(); loadErr != nil {
		log.Warn("failed to load channel state", "error", loadErr)
	}
	channelSvc := channel.NewChannelService(chStore)

	daemonMgr, daemonErr := daemon.NewManager(ws.RootDir)
	if daemonErr != nil {
		log.Warn("failed to open daemon manager", "error", daemonErr)
	}

	cfg := server.DefaultConfig()
	srv := server.New(cfg, agentSvc, channelSvc, daemonMgr, ws)

	// Register cleanup
	shutdown.OnShutdownNamed(shutdown.PriorityHigh, "bcd-server", func(ctx context.Context) error {
		return srv.Shutdown(ctx)
	})
	if daemonMgr != nil {
		shutdown.OnShutdownNamed(shutdown.PriorityLow, "bcd-daemon-db", func(_ context.Context) error {
			return daemonMgr.Close()
		})
	}

	fmt.Printf("bcd listening on %s (workspace: %s)\n", cfg.Addr, ws.RootDir)
	fmt.Println("Press Ctrl+C to stop")

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	shutdown.OnShutdown(shutdown.PriorityCritical, func(_ context.Context) error {
		cancel()
		return nil
	})
	shutdown.Start()

	return srv.Start(ctx)
}

// daemonizeStart re-executes bc in the background to daemonize.
func daemonizeStart(workspaceDir string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable: %w", err)
	}

	logFile := workspaceDir + "/.bc/bcd.log"
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600) //nolint:gosec // path from workspace
	if err != nil {
		return fmt.Errorf("open daemon log: %w", err)
	}
	defer func() { _ = f.Close() }()

	//nolint:gosec // trusted self-exec
	proc := exec.Command(exe, "daemon", "start")
	proc.Stdout = f
	proc.Stderr = f
	proc.Env = append(os.Environ(), "BC_DAEMON_BG=1")

	if err := proc.Start(); err != nil {
		return fmt.Errorf("start daemon: %w", err)
	}
	if err := proc.Process.Release(); err != nil {
		log.Debug("failed to release daemon process", "error", err)
	}

	fmt.Printf("bcd started in background (PID %d)\n", proc.Process.Pid)
	fmt.Printf("Logs: %s\n", logFile)
	return nil
}

func runDaemonStop(cmd *cobra.Command, args []string) error {
	// With a name argument: stop workspace process
	if len(args) == 1 {
		ws, err := getWorkspace()
		if err != nil {
			return errNotInWorkspace(err)
		}
		mgr, err := daemon.NewManager(ws.RootDir)
		if err != nil {
			return fmt.Errorf("open daemon manager: %w", err)
		}
		defer func() { _ = mgr.Close() }()

		if err := mgr.Stop(cmd.Context(), args[0]); err != nil {
			return err
		}
		fmt.Printf("✓ stopped %s\n", args[0])
		return nil
	}

	// Without name: stop the bcd HTTP server
	c := getClient()
	if err := c.Ping(cmd.Context()); err != nil {
		fmt.Println("bcd is not running")
		return nil
	}
	// TODO: implement /api/shutdown endpoint for graceful stop
	fmt.Println("Stopping bcd... (send SIGTERM to the process)")
	return nil
}

func runDaemonStatus(cmd *cobra.Command, _ []string) error {
	c := getClient()
	if err := c.Ping(cmd.Context()); err != nil {
		fmt.Println("bcd: not running")
		return nil
	}
	fmt.Println("bcd: running")
	fmt.Printf("Address: %s\n", c.BaseURL)
	return nil
}

func runDaemonLogs(cmd *cobra.Command, args []string) error {
	// With name: workspace process logs
	if len(args) == 1 {
		ws, err := getWorkspace()
		if err != nil {
			return errNotInWorkspace(err)
		}
		mgr, err := daemon.NewManager(ws.RootDir)
		if err != nil {
			return fmt.Errorf("open daemon manager: %w", err)
		}
		defer func() { _ = mgr.Close() }()

		output, err := mgr.Logs(cmd.Context(), args[0], daemonLogsLines)
		if err != nil {
			return err
		}
		fmt.Print(output)
		return nil
	}

	// Without name: bcd server logs
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}
	logFile := ws.RootDir + "/.bc/bcd.log"
	data, err := os.ReadFile(logFile) //nolint:gosec // path from workspace
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("(no bcd logs found)")
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	if daemonLogsLines > 0 && len(lines) > daemonLogsLines {
		lines = lines[len(lines)-daemonLogsLines:]
	}
	fmt.Print(strings.Join(lines, "\n"))
	return nil
}

// --- workspace process handlers ---

func runDaemonRun(cmd *cobra.Command, _ []string) error {
	if daemonRunName == "" {
		return fmt.Errorf("--name is required")
	}
	if daemonRunRuntime == "" {
		return fmt.Errorf("--runtime is required (bash or docker)")
	}

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr, err := daemon.NewManager(ws.RootDir)
	if err != nil {
		return fmt.Errorf("open daemon manager: %w", err)
	}
	defer func() { _ = mgr.Close() }()

	fmt.Printf("Starting %s (%s)... ", daemonRunName, daemonRunRuntime)

	d, err := mgr.Run(cmd.Context(), daemon.RunOptions{
		Name:    daemonRunName,
		Runtime: daemonRunRuntime,
		Cmd:     daemonRunCmd_,
		Image:   daemonRunImage,
		Ports:   daemonRunPorts,
		Volumes: daemonRunVolumes,
		Env:     daemonRunEnv,
		EnvFile: daemonRunEnvFile,
		Restart: daemonRunRestart,
		Detach:  daemonRunDetach,
	})
	if err != nil {
		fmt.Println("✗")
		return err
	}

	fmt.Println("✓")
	printDaemon(d)
	return nil
}

func runDaemonList(cmd *cobra.Command, _ []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr, err := daemon.NewManager(ws.RootDir)
	if err != nil {
		return fmt.Errorf("open daemon manager: %w", err)
	}
	defer func() { _ = mgr.Close() }()

	daemons, err := mgr.List(cmd.Context())
	if err != nil {
		return err
	}

	if len(daemons) == 0 {
		fmt.Println("No workspace processes running.")
		fmt.Println("Start one with: bc daemon run --name <name> --runtime bash --cmd <cmd>")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tRUNTIME\tSTATUS\tSTARTED\t")
	for _, d := range daemons {
		started := "-"
		if !d.StartedAt.IsZero() {
			started = time.Since(d.StartedAt).Round(time.Second).String() + " ago"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t\n", d.Name, d.Runtime, d.Status, started)
	}
	return w.Flush()
}

func runDaemonRestart(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr, err := daemon.NewManager(ws.RootDir)
	if err != nil {
		return fmt.Errorf("open daemon manager: %w", err)
	}
	defer func() { _ = mgr.Close() }()

	fmt.Printf("Restarting %s... ", args[0])
	d, err := mgr.Restart(cmd.Context(), args[0])
	if err != nil {
		fmt.Println("✗")
		return err
	}
	fmt.Println("✓")
	printDaemon(d)
	return nil
}

func runDaemonRm(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	mgr, err := daemon.NewManager(ws.RootDir)
	if err != nil {
		return fmt.Errorf("open daemon manager: %w", err)
	}
	defer func() { _ = mgr.Close() }()

	if err := mgr.Remove(cmd.Context(), args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ removed %s\n", args[0])
	return nil
}

// printDaemon prints a summary of a daemon.
func printDaemon(d *daemon.Daemon) {
	fmt.Printf("  Name:    %s\n", d.Name)
	fmt.Printf("  Runtime: %s\n", d.Runtime)
	fmt.Printf("  Status:  %s\n", d.Status)
	if d.Runtime == daemon.RuntimeBash {
		fmt.Printf("  Cmd:     %s\n", d.Cmd)
	} else {
		fmt.Printf("  Image:   %s\n", d.Image)
	}
}

// getClient returns an HTTP client for the bcd daemon.
func getClient() *client.Client {
	return client.New("")
}
