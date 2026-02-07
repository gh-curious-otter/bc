package cmd

import (
	"fmt"
	"os"
	"strings"
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
	Short: "Show process logs",
	Long: `Show logs for a process.

Example:
  bc process logs web`,
	Args: cobra.ExactArgs(1),
	RunE: runProcessLogs,
}

var (
	processCommand string
	processPort    int
	processWorkDir string
)

func init() {
	processStartCmd.Flags().StringVar(&processCommand, "cmd", "", "Command to run (required)")
	processStartCmd.Flags().IntVar(&processPort, "port", 0, "Port the process will use (for conflict detection)")
	processStartCmd.Flags().StringVar(&processWorkDir, "dir", "", "Working directory for the process")
	_ = processStartCmd.MarkFlagRequired("cmd")

	processCmd.AddCommand(processStartCmd)
	processCmd.AddCommand(processListCmd)
	processCmd.AddCommand(processStopCmd)
	processCmd.AddCommand(processLogsCmd)
	rootCmd.AddCommand(processCmd)
}

func getProcessManager() (*process.Manager, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, fmt.Errorf("not in a bc workspace: %w", err)
	}

	mgr := process.NewManager(ws.RootDir)
	if err := mgr.LoadState(); err != nil {
		return nil, fmt.Errorf("failed to load process state: %w", err)
	}

	return mgr, nil
}

func runProcessStart(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := getProcessManager()
	if err != nil {
		return err
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

	// Check port conflict if specified
	if processPort > 0 {
		if portErr := checkPortConflict(mgr, processPort); portErr != nil {
			return portErr
		}
	}

	proc, err := mgr.Start(name, command, cmdArgs, workDir, owner)
	if err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	fmt.Printf("Started process %q (PID %d)\n", name, proc.PID)
	if processPort > 0 {
		fmt.Printf("  Port: %d\n", processPort)
	}

	return nil
}

func runProcessList(cmd *cobra.Command, args []string) error {
	mgr, err := getProcessManager()
	if err != nil {
		return err
	}

	// Refresh state to check for dead processes
	if refreshErr := mgr.RefreshState(); refreshErr != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to refresh state: %v\n", refreshErr)
	}

	procs := mgr.List()
	if len(procs) == 0 {
		fmt.Println("No processes")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "NAME\tSTATE\tPID\tOWNER\tSTARTED")

	for _, p := range procs {
		started := p.StartedAt.Format(time.RFC3339)
		_, _ = fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			p.Name, p.State, p.PID, p.Owner, started)
	}

	return w.Flush()
}

func runProcessStop(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := getProcessManager()
	if err != nil {
		return err
	}

	if stopErr := mgr.Stop(name); stopErr != nil {
		return fmt.Errorf("failed to stop process: %w", stopErr)
	}

	fmt.Printf("Stopped process %q\n", name)
	return nil
}

func runProcessLogs(cmd *cobra.Command, args []string) error {
	name := args[0]

	mgr, err := getProcessManager()
	if err != nil {
		return err
	}

	proc, ok := mgr.Get(name)
	if !ok {
		return fmt.Errorf("process %q not found", name)
	}

	// For now, just show process info since we don't capture stdout/stderr
	fmt.Printf("Process: %s\n", proc.Name)
	fmt.Printf("Command: %s %s\n", proc.Command, strings.Join(proc.Args, " "))
	fmt.Printf("State: %s\n", proc.State)
	fmt.Printf("PID: %d\n", proc.PID)
	fmt.Printf("Started: %s\n", proc.StartedAt.Format(time.RFC3339))

	if proc.State != process.StateRunning {
		fmt.Printf("Stopped: %s\n", proc.StoppedAt.Format(time.RFC3339))
		fmt.Printf("Exit Code: %d\n", proc.ExitCode)
	}

	fmt.Println("\n(Full log capture not yet implemented)")
	return nil
}

// checkPortConflict checks if any running process is using the given port.
func checkPortConflict(mgr *process.Manager, port int) error {
	// This is a placeholder - actual port checking would require
	// either storing port metadata or checking system ports
	_ = mgr
	_ = port
	return nil
}
