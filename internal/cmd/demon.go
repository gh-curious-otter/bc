package cmd

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/demon"
)

var demonCmd = &cobra.Command{
	Use:   "demon",
	Short: "Manage scheduled tasks (demons)",
	Long: `Manage scheduled background tasks.

Demons are scheduled tasks that run at specified intervals using cron syntax.

Cron format: minute hour day-of-month month day-of-week
  0 * * * *     = every hour
  */5 * * * *   = every 5 minutes
  0 9 * * *     = daily at 9am
  0 17 * * 1-5  = weekdays at 5pm

Examples:
  bc demon create backup --schedule '0 * * * *' --cmd 'bc backup'
  bc demon list
  bc demon show backup
  bc demon delete backup`,
}

var demonCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a scheduled demon",
	Long: `Create a new scheduled demon with cron syntax.

Cron format: minute hour day-of-month month day-of-week

Examples:
  bc demon create hourly-check --schedule '0 * * * *' --cmd 'bc status'
  bc demon create daily-backup --schedule '0 9 * * *' --cmd 'bc backup'
  bc demon create every-5min --schedule '*/5 * * * *' --cmd 'bc health'
  bc demon create weekday-report --schedule '0 17 * * 1-5' --cmd 'bc report'`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonCreate,
}

var demonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all demons",
	RunE:  runDemonList,
}

var demonShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show demon details",
	Args:  cobra.ExactArgs(1),
	RunE:  runDemonShow,
}

var demonDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a demon",
	Args:  cobra.ExactArgs(1),
	RunE:  runDemonDelete,
}

var demonRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Manually trigger a demon",
	Long: `Execute a demon's command immediately.

Examples:
  bc demon run backup`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonRun,
}

var demonEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a demon",
	Args:  cobra.ExactArgs(1),
	RunE:  runDemonEnable,
}

var demonDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a demon (stop scheduling)",
	Args:  cobra.ExactArgs(1),
	RunE:  runDemonDisable,
}

var demonLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show execution history for a demon",
	Long: `Show the execution history for a demon.

Examples:
  bc demon logs backup           # Show all run history
  bc demon logs backup --tail 5  # Show last 5 runs`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonLogs,
}

var (
	demonSchedule string
	demonCommand  string
	demonTail     int
)

func init() {
	demonCreateCmd.Flags().StringVar(&demonSchedule, "schedule", "", "Cron schedule (required)")
	demonCreateCmd.Flags().StringVar(&demonCommand, "cmd", "", "Command to execute (required)")
	_ = demonCreateCmd.MarkFlagRequired("schedule")
	_ = demonCreateCmd.MarkFlagRequired("cmd")

	demonLogsCmd.Flags().IntVar(&demonTail, "tail", 0, "Show only the last N entries")

	demonCmd.AddCommand(demonCreateCmd)
	demonCmd.AddCommand(demonListCmd)
	demonCmd.AddCommand(demonShowCmd)
	demonCmd.AddCommand(demonDeleteCmd)
	demonCmd.AddCommand(demonRunCmd)
	demonCmd.AddCommand(demonEnableCmd)
	demonCmd.AddCommand(demonDisableCmd)
	demonCmd.AddCommand(demonLogsCmd)
	rootCmd.AddCommand(demonCmd)
}

func runDemonCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	d, err := store.Create(name, demonSchedule, demonCommand)
	if err != nil {
		return err
	}

	cmd.Printf("Created demon %q\n", d.Name)
	cmd.Printf("  Schedule: %s\n", d.Schedule)
	cmd.Printf("  Command:  %s\n", d.Command)
	if !d.NextRun.IsZero() {
		cmd.Printf("  Next run: %s\n", d.NextRun.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runDemonList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store := demon.NewStore(ws.RootDir)
	demons, err := store.List()
	if err != nil {
		return err
	}

	if len(demons) == 0 {
		cmd.Println("No demons configured")
		cmd.Println()
		cmd.Println("Create one with: bc demon create <name> --schedule '<cron>' --cmd '<command>'")
		return nil
	}

	cmd.Printf("%-20s %-20s %-10s %s\n", "NAME", "SCHEDULE", "ENABLED", "COMMAND")
	cmd.Println("--------------------------------------------------------------------")
	for _, d := range demons {
		enabled := "yes"
		if !d.Enabled {
			enabled = "no"
		}
		command := d.Command
		if len(command) > 30 {
			command = command[:27] + "..."
		}
		cmd.Printf("%-20s %-20s %-10s %s\n", d.Name, d.Schedule, enabled, command)
	}

	return nil
}

func runDemonShow(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	d, err := store.Get(name)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("demon %q not found", name)
	}

	cmd.Printf("Name:      %s\n", d.Name)
	cmd.Printf("Schedule:  %s\n", d.Schedule)
	cmd.Printf("Command:   %s\n", d.Command)
	cmd.Printf("Enabled:   %t\n", d.Enabled)
	cmd.Printf("Created:   %s\n", d.CreatedAt.Format("2006-01-02 15:04:05"))
	cmd.Printf("Updated:   %s\n", d.UpdatedAt.Format("2006-01-02 15:04:05"))
	if !d.LastRun.IsZero() {
		cmd.Printf("Last run:  %s\n", d.LastRun.Format("2006-01-02 15:04:05"))
	}
	if !d.NextRun.IsZero() {
		cmd.Printf("Next run:  %s\n", d.NextRun.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runDemonDelete(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	if err := store.Delete(name); err != nil {
		return err
	}

	cmd.Printf("Deleted demon %q\n", name)
	return nil
}

func runDemonRun(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	d, err := store.Get(name)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("demon %q not found", name)
	}

	cmd.Printf("Running demon %q: %s\n", name, d.Command)

	// Execute the command with timing
	startTime := time.Now()
	ctx := context.Background()
	execCmd := exec.CommandContext(ctx, "sh", "-c", d.Command) //nolint:gosec // command from trusted demon config
	execCmd.Dir = ws.RootDir
	output, execErr := execCmd.CombinedOutput()
	duration := time.Since(startTime)

	exitCode := 0
	success := true
	if execErr != nil {
		success = false
		if exitErr, ok := execErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return fmt.Errorf("failed to execute command: %w", execErr)
		}
	}

	// Record the run (updates LastRun, RunCount, NextRun)
	if recordErr := store.RecordRun(name); recordErr != nil {
		return fmt.Errorf("command executed but failed to record: %w", recordErr)
	}

	// Record the run log (detailed execution history)
	runLog := demon.RunLog{
		Timestamp: startTime.UTC(),
		Duration:  duration.Milliseconds(),
		ExitCode:  exitCode,
		Success:   success,
	}
	if logErr := store.RecordRunLog(name, runLog); logErr != nil {
		// Log error but don't fail - the command did execute
		cmd.PrintErrf("Warning: failed to record log: %v\n", logErr)
	}

	// Print output
	if len(output) > 0 {
		cmd.Println("---")
		cmd.Print(string(output))
		if output[len(output)-1] != '\n' {
			cmd.Println()
		}
		cmd.Println("---")
	}

	if exitCode != 0 {
		cmd.Printf("Exit code: %d\n", exitCode)
	}
	cmd.Println("Run recorded.")

	return nil
}

func runDemonEnable(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	if err := store.Enable(name); err != nil {
		return err
	}

	d, _ := store.Get(name)
	cmd.Printf("Enabled demon %q\n", name)
	if d != nil && !d.NextRun.IsZero() {
		cmd.Printf("Next run: %s\n", d.NextRun.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runDemonDisable(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	if err := store.Disable(name); err != nil {
		return err
	}

	cmd.Printf("Disabled demon %q\n", name)
	return nil
}

func runDemonLogs(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	// Check if demon exists
	d, err := store.Get(name)
	if err != nil {
		return err
	}
	if d == nil {
		return fmt.Errorf("demon %q not found", name)
	}

	logs, err := store.GetRunLogs(name, demonTail)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		cmd.Printf("No execution history for demon %q\n", name)
		cmd.Println()
		cmd.Printf("Run with: bc demon run %s\n", name)
		return nil
	}

	cmd.Printf("Execution history for %q (%d runs):\n\n", name, d.RunCount)
	cmd.Printf("%-24s %-12s %-10s %s\n", "TIMESTAMP", "DURATION", "STATUS", "EXIT CODE")
	cmd.Println("--------------------------------------------------------------------")

	for _, log := range logs {
		status := "success"
		if !log.Success {
			status = "failed"
		}

		duration := fmt.Sprintf("%dms", log.Duration)
		if log.Duration >= 1000 {
			duration = fmt.Sprintf("%.1fs", float64(log.Duration)/1000)
		}

		cmd.Printf("%-24s %-12s %-10s %d\n",
			log.Timestamp.Local().Format("2006-01-02 15:04:05"),
			duration,
			status,
			log.ExitCode)
	}

	return nil
}
