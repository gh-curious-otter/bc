package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/demon"
)

var demonCmd = &cobra.Command{
	Use:   "demon",
	Short: "Manage scheduled tasks (demons)",
	Long: `Manage scheduled background tasks.

Demons are scheduled tasks that run at specified intervals using cron syntax.

Example:
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
	Short: "Run a demon manually",
	Long: `Run a demon's command manually, regardless of schedule.

Example:
  bc demon run backup`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonRun,
}

var demonStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop/disable a demon",
	Long: `Disable a demon from running on its schedule.

Example:
  bc demon stop backup`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonStop,
}

var demonEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a stopped demon",
	Long: `Re-enable a previously stopped demon.

Example:
  bc demon enable backup`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonEnable,
}

var (
	demonSchedule string
	demonCommand  string
)

func init() {
	demonCreateCmd.Flags().StringVar(&demonSchedule, "schedule", "", "Cron schedule (required)")
	demonCreateCmd.Flags().StringVar(&demonCommand, "cmd", "", "Command to execute (required)")
	_ = demonCreateCmd.MarkFlagRequired("schedule")
	_ = demonCreateCmd.MarkFlagRequired("cmd")

	demonCmd.AddCommand(demonCreateCmd)
	demonCmd.AddCommand(demonListCmd)
	demonCmd.AddCommand(demonShowCmd)
	demonCmd.AddCommand(demonDeleteCmd)
	demonCmd.AddCommand(demonRunCmd)
	demonCmd.AddCommand(demonStopCmd)
	demonCmd.AddCommand(demonEnableCmd)
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

	fmt.Printf("Created demon %q\n", d.Name)
	fmt.Printf("  Schedule: %s\n", d.Schedule)
	fmt.Printf("  Command:  %s\n", d.Command)
	if !d.NextRun.IsZero() {
		fmt.Printf("  Next run: %s\n", d.NextRun.Format("2006-01-02 15:04:05"))
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
		fmt.Println("No demons configured")
		fmt.Println()
		fmt.Println("Create one with: bc demon create <name> --schedule '<cron>' --cmd '<command>'")
		return nil
	}

	fmt.Printf("%-20s %-20s %-10s %s\n", "NAME", "SCHEDULE", "ENABLED", "COMMAND")
	fmt.Println("--------------------------------------------------------------------")
	for _, d := range demons {
		enabled := "yes"
		if !d.Enabled {
			enabled = "no"
		}
		command := d.Command
		if len(command) > 30 {
			command = command[:27] + "..."
		}
		fmt.Printf("%-20s %-20s %-10s %s\n", d.Name, d.Schedule, enabled, command)
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

	fmt.Printf("Name:      %s\n", d.Name)
	fmt.Printf("Schedule:  %s\n", d.Schedule)
	fmt.Printf("Command:   %s\n", d.Command)
	fmt.Printf("Enabled:   %t\n", d.Enabled)
	fmt.Printf("Created:   %s\n", d.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:   %s\n", d.UpdatedAt.Format("2006-01-02 15:04:05"))
	if !d.LastRun.IsZero() {
		fmt.Printf("Last run:  %s\n", d.LastRun.Format("2006-01-02 15:04:05"))
	}
	if !d.NextRun.IsZero() {
		fmt.Printf("Next run:  %s\n", d.NextRun.Format("2006-01-02 15:04:05"))
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

	fmt.Printf("Deleted demon %q\n", name)
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

	fmt.Printf("Running demon %q: %s\n", name, d.Command)

	// Execute the command
	execCmd := exec.CommandContext(context.Background(), "sh", "-c", d.Command) //nolint:gosec // command from user config
	execCmd.Dir = ws.RootDir
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	if runErr := execCmd.Run(); runErr != nil {
		return fmt.Errorf("command failed: %w", runErr)
	}

	// Record the run
	if recordErr := store.RecordRun(name); recordErr != nil {
		fmt.Printf("Warning: failed to record run: %v\n", recordErr)
	}

	fmt.Printf("\nDemon %q completed successfully\n", name)
	return nil
}

func runDemonStop(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	name := args[0]
	store := demon.NewStore(ws.RootDir)

	if err := store.Disable(name); err != nil {
		return err
	}

	fmt.Printf("Stopped demon %q\n", name)
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

	fmt.Printf("Enabled demon %q\n", name)
	return nil
}
