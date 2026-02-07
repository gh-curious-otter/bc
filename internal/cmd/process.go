package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/process"
)

var processCmd = &cobra.Command{
	Use:   "process",
	Short: "Manage background processes",
	Long: `Manage background processes running in the workspace.

Example:
  bc process start server --cmd 'npm start' --port 3000
  bc process list
  bc process stop server`,
}

var processStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a background process",
	Long: `Start a named background process.

Example:
  bc process start web --cmd 'npm start'
  bc process start api --cmd 'go run main.go' --port 8080
  bc process start db --cmd 'docker run -d postgres'`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessStart,
}

var processListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all processes",
	RunE:  runProcessList,
}

var processStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a running process",
	Args:  cobra.ExactArgs(1),
	RunE:  runProcessStop,
}

var processShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show process details",
	Args:  cobra.ExactArgs(1),
	RunE:  runProcessShow,
}

var (
	processCommand string
	processPort    int
)

func init() {
	processStartCmd.Flags().StringVar(&processCommand, "cmd", "", "Command to execute (required)")
	processStartCmd.Flags().IntVar(&processPort, "port", 0, "Port the process will listen on")
	_ = processStartCmd.MarkFlagRequired("cmd")

	processCmd.AddCommand(processStartCmd)
	processCmd.AddCommand(processListCmd)
	processCmd.AddCommand(processStopCmd)
	processCmd.AddCommand(processShowCmd)
	rootCmd.AddCommand(processCmd)
}

func runProcessStart(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	registry := process.NewRegistry(ws.RootDir)
	if err := registry.Init(); err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	// Check if process already exists
	if existing := registry.Get(name); existing != nil && existing.Running {
		return fmt.Errorf("process %q is already running (PID %d)", name, existing.PID)
	}

	// Check port conflict
	if processPort > 0 {
		if conflicting := registry.GetByPort(processPort); conflicting != nil {
			return fmt.Errorf("port %d is already in use by process %q", processPort, conflicting.Name)
		}
	}

	// Get agent ID if available
	owner := os.Getenv("BC_AGENT_ID")

	// Start the process
	ctx := context.Background()
	execCmd := exec.CommandContext(ctx, "sh", "-c", processCommand) //nolint:gosec // command from user input
	execCmd.Dir = ws.RootDir
	execCmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group
	}

	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	pid := execCmd.Process.Pid

	// Register the process
	p := &process.Process{
		Name:    name,
		Command: processCommand,
		PID:     pid,
		Port:    processPort,
		Owner:   owner,
		WorkDir: ws.RootDir,
	}

	if err := registry.Register(p); err != nil {
		// Try to kill the process if registration fails
		_ = execCmd.Process.Kill()
		return fmt.Errorf("failed to register process: %w", err)
	}

	fmt.Printf("Started process %q (PID %d)\n", name, pid)
	fmt.Printf("  Command: %s\n", processCommand)
	if processPort > 0 {
		fmt.Printf("  Port:    %d\n", processPort)
	}

	return nil
}

func runProcessList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	registry := process.NewRegistry(ws.RootDir)
	if err := registry.Init(); err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	processes := registry.List()
	if len(processes) == 0 {
		fmt.Println("No processes registered")
		fmt.Println()
		fmt.Println("Start one with: bc process start <name> --cmd '<command>'")
		return nil
	}

	fmt.Printf("%-15s %-8s %-8s %-15s %s\n", "NAME", "PID", "PORT", "STATUS", "COMMAND")
	fmt.Println("----------------------------------------------------------------------")
	for _, p := range processes {
		status := "stopped"
		pid := "-"
		if p.Running {
			status = "running"
			pid = fmt.Sprintf("%d", p.PID)
			// Check if process is actually running
			if !isProcessRunning(p.PID) {
				status = "dead"
			}
		}
		port := "-"
		if p.Port > 0 {
			port = fmt.Sprintf("%d", p.Port)
		}
		command := p.Command
		if len(command) > 25 {
			command = command[:22] + "..."
		}
		fmt.Printf("%-15s %-8s %-8s %-15s %s\n", p.Name, pid, port, status, command)
	}

	return nil
}

func runProcessStop(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	registry := process.NewRegistry(ws.RootDir)
	if initErr := registry.Init(); initErr != nil {
		return fmt.Errorf("failed to initialize registry: %w", initErr)
	}

	p := registry.Get(name)
	if p == nil {
		return fmt.Errorf("process %q not found", name)
	}

	if !p.Running {
		fmt.Printf("Process %q is already stopped\n", name)
		return nil
	}

	// Try to kill the process
	proc, err := os.FindProcess(p.PID)
	if err == nil {
		// Send SIGTERM first
		if termErr := proc.Signal(syscall.SIGTERM); termErr != nil {
			// Process may already be dead
			fmt.Printf("Process %q (PID %d) not found, cleaning up registry\n", name, p.PID)
		} else {
			fmt.Printf("Stopped process %q (PID %d)\n", name, p.PID)
		}
	}

	// Mark as stopped in registry
	if err := registry.MarkStopped(name); err != nil {
		return fmt.Errorf("failed to update registry: %w", err)
	}

	return nil
}

func runProcessShow(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	registry := process.NewRegistry(ws.RootDir)
	if err := registry.Init(); err != nil {
		return fmt.Errorf("failed to initialize registry: %w", err)
	}

	p := registry.Get(name)
	if p == nil {
		return fmt.Errorf("process %q not found", name)
	}

	status := "stopped"
	if p.Running {
		status = "running"
		if !isProcessRunning(p.PID) {
			status = "dead (not responding)"
		}
	}

	fmt.Printf("Name:      %s\n", p.Name)
	fmt.Printf("Command:   %s\n", p.Command)
	fmt.Printf("PID:       %d\n", p.PID)
	fmt.Printf("Status:    %s\n", status)
	if p.Port > 0 {
		fmt.Printf("Port:      %d\n", p.Port)
	}
	if p.Owner != "" {
		fmt.Printf("Owner:     %s\n", p.Owner)
	}
	if p.WorkDir != "" {
		fmt.Printf("WorkDir:   %s\n", p.WorkDir)
	}
	if !p.StartedAt.IsZero() {
		fmt.Printf("Started:   %s\n", p.StartedAt.Format("2006-01-02 15:04:05"))
	}

	return nil
}

// isProcessRunning checks if a process with the given PID is running.
func isProcessRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Use Signal(0) to check.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}
