package cmd

import (
	"context"
	"encoding/json"
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

Examples:
  bc process start web --cmd 'npm run dev'
  bc process list
  bc process logs web
  bc process stop web`,
}

var processStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a background process",
	Long: `Start a named background process.

Examples:
  bc process start web --cmd 'npm run dev'
  bc process start api --cmd 'go run ./cmd/server' --port 8080`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessStart,
}

var processListCmd = &cobra.Command{
	Use:   "list",
	Short: "List processes",
	Long: `List all managed processes.

Examples:
  bc process list`,
	RunE: runProcessList,
}

var processStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a process",
	Long: `Stop a running process.

Examples:
  bc process stop web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessStop,
}

var processLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show process logs",
	Long: `View logs for a process.

Examples:
  bc process logs web
  bc process logs web -n 100`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessLogs,
}

var processAttachCmd = &cobra.Command{
	Use:   "attach <name>",
	Short: "Attach to a running process",
	Long: `Attach to a running process to view its output in real-time.

Examples:
  bc process attach web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessAttach,
}

var processShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show process details",
	Long: `Show detailed information about a process.

Examples:
  bc process show web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessShow,
}

var processRestartCmd = &cobra.Command{
	Use:   "restart <name>",
	Short: "Restart a process",
	Long: `Restart a running process gracefully.

Stops the process with SIGTERM, waits for termination, then starts it again
with the same configuration.

Examples:
  bc process restart web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessRestart,
}

var (
	processCommand  string
	processPort     int
	processWorkDir  string
	processLogLines int
)

func init() {
	processStartCmd.Flags().StringVar(&processCommand, "cmd", "", "Command to run (required)")
	processStartCmd.Flags().IntVar(&processPort, "port", 0, "Port the process will use (for conflict detection)")
	processStartCmd.Flags().StringVar(&processWorkDir, "dir", "", "Working directory for the process")
	_ = processStartCmd.MarkFlagRequired("cmd")

	processLogsCmd.Flags().IntVarP(&processLogLines, "lines", "n", 50, "Number of lines to show")

	processCmd.AddCommand(processStartCmd)
	processCmd.AddCommand(processListCmd)
	processCmd.AddCommand(processStopCmd)
	processCmd.AddCommand(processLogsCmd)
	processCmd.AddCommand(processAttachCmd)
	processCmd.AddCommand(processShowCmd)
	processCmd.AddCommand(processRestartCmd)
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

	// Create log file
	logFile, logErr := reg.CreateLogFile(name)
	if logErr != nil {
		return fmt.Errorf("failed to create log file: %w", logErr)
	}

	// Start the process with output captured to log file
	execCmd := exec.CommandContext(context.Background(), command, cmdArgs...) //nolint:gosec // user-provided command
	execCmd.Dir = workDir
	execCmd.Stdout = logFile
	execCmd.Stderr = logFile

	if startErr := execCmd.Start(); startErr != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to start process: %w", startErr)
	}

	// Close log file in background after process exits
	go func() {
		_ = execCmd.Wait()
		_ = logFile.Close()
	}()

	// Register the process
	proc := &process.Process{
		Name:    name,
		Command: processCommand,
		Owner:   owner,
		WorkDir: workDir,
		LogFile: reg.LogPath(name),
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

	// Check for JSON output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Wrap in object for TUI compatibility
		if procs == nil {
			procs = []*process.Process{}
		}
		response := struct {
			Processes []*process.Process `json:"processes"`
		}{Processes: procs}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(response)
	}

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

	// Read logs
	logs, readErr := reg.ReadLogs(name, processLogLines)
	if readErr != nil {
		return fmt.Errorf("failed to read logs: %w", readErr)
	}

	if logs == "" {
		fmt.Printf("No logs available for process %q\n", name)
		return nil
	}

	fmt.Print(logs)
	return nil
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

	fmt.Printf("Process: %s\n", proc.Name)
	fmt.Printf("Command: %s\n", proc.Command)
	fmt.Printf("Status: %s\n", statusStr(proc.Running))
	fmt.Printf("PID: %d\n", proc.PID)
	if proc.Port > 0 {
		fmt.Printf("Port: %d\n", proc.Port)
	}
	if proc.Owner != "" {
		fmt.Printf("Owner: %s\n", proc.Owner)
	}
	if proc.WorkDir != "" {
		fmt.Printf("WorkDir: %s\n", proc.WorkDir)
	}
	if proc.LogFile != "" {
		fmt.Printf("LogFile: %s\n", proc.LogFile)
	}
	fmt.Printf("Started: %s\n", proc.StartedAt.Format(time.RFC3339))
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

	// Print process info header
	fmt.Printf("=== Attached to %s (PID %d) ===\n", proc.Name, proc.PID)
	fmt.Printf("Command: %s\n", proc.Command)
	fmt.Printf("Log file: %s\n", reg.LogPath(name))
	fmt.Println("---")

	// Show recent logs
	logs, readErr := reg.ReadLogs(name, 50)
	if readErr != nil {
		return fmt.Errorf("failed to read logs: %w", readErr)
	}

	if logs != "" {
		fmt.Print(logs)
	}

	fmt.Println("\n(Detached - process continues in background)")
	return nil
}

func statusStr(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}

func runProcessRestart(cmd *cobra.Command, args []string) error {
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
		return fmt.Errorf("process %q is not running (use 'bc process start' to start it)", name)
	}

	// Save process config before stopping
	savedCommand := proc.Command
	savedPort := proc.Port
	savedWorkDir := proc.WorkDir

	fmt.Printf("Stopping process %q...\n", name)

	// Stop the process gracefully
	if proc.PID > 0 {
		p, findErr := os.FindProcess(proc.PID)
		if findErr == nil {
			// Try graceful shutdown first (SIGTERM)
			if sigErr := p.Signal(syscall.SIGTERM); sigErr != nil {
				// If SIGTERM fails, try SIGKILL
				_ = p.Kill()
			}

			// Wait for process to terminate (with timeout)
			done := make(chan struct{})
			go func() {
				_, _ = p.Wait()
				close(done)
			}()

			select {
			case <-done:
				// Process terminated
			case <-time.After(5 * time.Second):
				// Timeout - force kill
				_ = p.Kill()
			}
		}
	}

	// Mark as stopped in registry
	if stopErr := reg.MarkStopped(name); stopErr != nil {
		return fmt.Errorf("failed to update registry: %w", stopErr)
	}

	fmt.Printf("Starting process %q...\n", name)

	// Parse command string into command and args
	parts := strings.Fields(savedCommand)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	command := parts[0]
	cmdArgs := parts[1:]

	// Get owner from environment
	owner := os.Getenv("BC_AGENT_ID")

	// Create log file
	logFile, logErr := reg.CreateLogFile(name)
	if logErr != nil {
		return fmt.Errorf("failed to create log file: %w", logErr)
	}

	// Start the process with output captured to log file
	execCmd := exec.CommandContext(context.Background(), command, cmdArgs...) //nolint:gosec // user-provided command
	execCmd.Dir = savedWorkDir
	execCmd.Stdout = logFile
	execCmd.Stderr = logFile

	if startErr := execCmd.Start(); startErr != nil {
		_ = logFile.Close()
		return fmt.Errorf("failed to start process: %w", startErr)
	}

	// Close log file in background after process exits
	go func() {
		_ = execCmd.Wait()
		_ = logFile.Close()
	}()

	// Register the process
	newProc := &process.Process{
		Name:    name,
		Command: savedCommand,
		Owner:   owner,
		WorkDir: savedWorkDir,
		LogFile: reg.LogPath(name),
		PID:     execCmd.Process.Pid,
		Port:    savedPort,
	}

	if regErr := reg.Register(newProc); regErr != nil {
		// Kill the process if we can't register it
		_ = execCmd.Process.Kill()
		return fmt.Errorf("failed to register process: %w", regErr)
	}

	fmt.Printf("Restarted process %q (PID %d)\n", name, newProc.PID)
	return nil
}
