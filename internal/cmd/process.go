package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/process"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Manage background processes",
	Long: `Commands for managing background processes in the workspace.

Example:
  bc process start web --cmd 'npm run dev'
  bc process list
  bc process logs web
  bc process stop web`,
}

var processStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a background process",
	Long: `Start a named background process.

Example:
  bc process start web --cmd 'npm run dev'
  bc process start api --cmd 'go run ./cmd/server' --port 8080`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessStart,
}

var processListCmd = &cobra.Command{
	Use:   "list",
	Short: "List processes",
	Long: `List all managed processes.

Example:
  bc process list`,
	RunE: runProcessList,
}

var processStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a process",
	Long: `Stop a running process.

Example:
  bc process stop web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessStop,
}

var processLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "View process logs",
	Long: `View the output logs for a process.

Example:
  bc process logs web
  bc process logs web --tail 100
  bc process logs web --follow`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessLogs,
}

var processAttachCmd = &cobra.Command{
	Use:   "attach <name>",
	Short: "Attach to a process (stream logs)",
	Long: `Attach to a running process and stream its output.

Press Ctrl+C to detach.

Example:
  bc process attach web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessAttach,
}

var processShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show process details",
	Long: `Show detailed information about a process.

Example:
  bc process show web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessShow,
}

var (
	processCommand  string
	processPort     int
	processWorkDir  string
	processLogsTail int
	processFollow   bool
)

func init() {
	processStartCmd.Flags().StringVar(&processCommand, "cmd", "", "Command to run (required)")
	processStartCmd.Flags().IntVar(&processPort, "port", 0, "Port the process will use (for conflict detection)")
	processStartCmd.Flags().StringVar(&processWorkDir, "dir", "", "Working directory for the process")
	_ = processStartCmd.MarkFlagRequired("cmd")

	processLogsCmd.Flags().IntVar(&processLogsTail, "tail", 50, "Number of lines to show (0 for all)")
	processLogsCmd.Flags().BoolVarP(&processFollow, "follow", "f", false, "Follow log output")

	processCmd.AddCommand(processStartCmd)
	processCmd.AddCommand(processListCmd)
	processCmd.AddCommand(processStopCmd)
	processCmd.AddCommand(processLogsCmd)
	processCmd.AddCommand(processAttachCmd)
	processCmd.AddCommand(processShowCmd)
	rootCmd.AddCommand(processCmd)
}

func getProcessRegistry() (*process.Registry, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, fmt.Errorf("not in a bc workspace: %w", err)
	}

	reg := process.NewRegistry(ws.RootDir)
	if err := reg.Init(); err != nil {
		return nil, fmt.Errorf("failed to init process registry: %w", err)
	}

	return reg, nil
}

func runProcessStart(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := getProcessRegistry()
	if err != nil {
		return err
	}

	// Check if already registered
	if existing := reg.Get(name); existing != nil && existing.Running {
		return fmt.Errorf("process %q is already running (PID %d)", name, existing.PID)
	}

	// Check port conflict
	if processPort > 0 && reg.IsPortInUse(processPort) {
		conflict := reg.GetByPort(processPort)
		return fmt.Errorf("port %d is already in use by process %q", processPort, conflict.Name)
	}

	// Parse command string into command and args
	parts := strings.Fields(processCommand)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	command := parts[0]
	cmdArgs := parts[1:]

	// Get owner from environment
	owner := os.Getenv("BC_AGENT_ID")

	// Use current directory if no workdir specified
	workDir := processWorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	// Create log file for output capture
	if err := reg.EnsureLogsDir(name); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	logPath := reg.GetLogPath(name)
	logFile, openErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600) //nolint:gosec // path from trusted registry
	if openErr != nil {
		return fmt.Errorf("failed to create log file: %w", openErr)
	}

	// Start the process
	execCmd := exec.CommandContext(context.Background(), command, cmdArgs...) //nolint:gosec // user-provided command
	execCmd.Dir = workDir
	execCmd.Stdout = logFile
	execCmd.Stderr = logFile

	if startErr := execCmd.Start(); startErr != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to start process: %w", startErr)
	}

	// Don't wait for process, just close the file handle
	// The process will keep writing to the log file
	_ = logFile.Close()

	// Register the process
	proc := &process.Process{
		Name:    name,
		Command: processCommand,
		Owner:   owner,
		WorkDir: workDir,
		PID:     execCmd.Process.Pid,
		Port:    processPort,
	}

	if regErr := reg.Register(proc); regErr != nil {
		// Kill the process if we can't register it
		_ = execCmd.Process.Kill()
		return fmt.Errorf("failed to register process: %w", regErr)
	}

	fmt.Printf("Started process %q (PID %d)\n", name, proc.PID)
	if processPort > 0 {
		fmt.Printf("  Port: %d\n", processPort)
	}

	return nil
}

func runProcessList(cmd *cobra.Command, args []string) error {
	reg, err := getProcessRegistry()
	if err != nil {
		return err
	}

	procs := reg.List()
	if len(procs) == 0 {
		fmt.Println("No processes")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tSTATUS\tPID\tPORT\tOWNER\tSTARTED")

	for _, p := range procs {
		status := "stopped"
		if p.Running {
			status = "running"
		}
		started := p.StartedAt.Format(time.RFC3339)
		port := "-"
		if p.Port > 0 {
			port = fmt.Sprintf("%d", p.Port)
		}
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%s\n",
			p.Name, status, p.PID, port, p.Owner, started)
	}

	return w.Flush()
}

func runProcessStop(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := getProcessRegistry()
	if err != nil {
		return err
	}

	proc := reg.Get(name)
	if proc == nil {
		return fmt.Errorf("process %q not found", name)
	}

	if !proc.Running {
		return fmt.Errorf("process %q is not running", name)
	}

	// Try to stop the process
	if proc.PID > 0 {
		p, findErr := os.FindProcess(proc.PID)
		if findErr == nil {
			// Try graceful shutdown first (SIGTERM)
			if sigErr := p.Signal(syscall.SIGTERM); sigErr != nil {
				// If SIGTERM fails, try SIGKILL
				_ = p.Kill()
			}
		}
	}

	// Mark as stopped in registry
	if stopErr := reg.MarkStopped(name); stopErr != nil {
		return fmt.Errorf("failed to update registry: %w", stopErr)
	}

	fmt.Printf("Stopped process %q\n", name)
	return nil
}

func runProcessLogs(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := getProcessRegistry()
	if err != nil {
		return err
	}

	proc := reg.Get(name)
	if proc == nil {
		return fmt.Errorf("process %q not found", name)
	}

	// If following, use attach functionality
	if processFollow {
		return followLogs(reg, name)
	}

	// Get logs
	logs, logsErr := reg.GetLogs(name, processLogsTail)
	if logsErr != nil {
		return fmt.Errorf("failed to read logs: %w", logsErr)
	}

	if len(logs) == 0 {
		fmt.Printf("No logs for process %q\n", name)
		return nil
	}

	fmt.Print(string(logs))
	if len(logs) > 0 && logs[len(logs)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func runProcessAttach(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := getProcessRegistry()
	if err != nil {
		return err
	}

	proc := reg.Get(name)
	if proc == nil {
		return fmt.Errorf("process %q not found", name)
	}

	if !proc.Running {
		return fmt.Errorf("process %q is not running", name)
	}

	fmt.Printf("Attached to %q (PID %d). Press Ctrl+C to detach.\n", name, proc.PID)
	return followLogs(reg, name)
}

func followLogs(reg *process.Registry, name string) error {
	logPath := reg.GetLogPath(name)

	// Open file for reading
	file, err := os.Open(logPath) //nolint:gosec // path from trusted registry
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Waiting for logs from %q...\n", name)
		} else {
			return fmt.Errorf("failed to open log file: %w", err)
		}
	}

	// Seek to end for follow
	if file != nil {
		_, _ = file.Seek(0, 2) // Seek to end
	}

	// Poll for new content
	buf := make([]byte, 4096)
	for {
		// Check if file exists now
		if file == nil {
			file, _ = os.Open(logPath) //nolint:gosec // path from trusted registry
			if file != nil {
				_, _ = file.Seek(0, 2)
			}
		}

		if file != nil {
			n, readErr := file.Read(buf)
			if n > 0 {
				_, _ = os.Stdout.Write(buf[:n])
			}
			if readErr != nil {
				// EOF or error - wait and retry
				time.Sleep(100 * time.Millisecond)
				continue
			}
		} else {
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func runProcessShow(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := getProcessRegistry()
	if err != nil {
		return err
	}

	proc := reg.Get(name)
	if proc == nil {
		return fmt.Errorf("process %q not found", name)
	}

	fmt.Printf("Name:    %s\n", proc.Name)
	fmt.Printf("Command: %s\n", proc.Command)
	fmt.Printf("Status:  %s\n", statusStr(proc.Running))
	fmt.Printf("PID:     %d\n", proc.PID)
	if proc.Port > 0 {
		fmt.Printf("Port:    %d\n", proc.Port)
	}
	if proc.Owner != "" {
		fmt.Printf("Owner:   %s\n", proc.Owner)
	}
	if proc.WorkDir != "" {
		fmt.Printf("WorkDir: %s\n", proc.WorkDir)
	}
	fmt.Printf("Started: %s\n", proc.StartedAt.Format(time.RFC3339))
	fmt.Printf("LogFile: %s\n", reg.GetLogPath(name))
	return nil
}

func statusStr(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}
