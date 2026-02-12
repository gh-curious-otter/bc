package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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

var demonEditCmd = &cobra.Command{
	Use:   "edit <name>",
	Short: "Edit a demon's configuration",
	Long: `Edit a demon's configuration using flags or an editor.

Examples:
  bc demon edit backup --schedule '0 9 * * *'     # Change schedule
  bc demon edit backup --cmd 'bc backup --full'   # Change command
  bc demon edit backup --enabled=false            # Disable demon
  bc demon edit backup --description 'Daily backup'
  bc demon edit backup                            # Open in $EDITOR`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonEdit,
}

var (
	demonSchedule        string
	demonCommand         string
	demonTail            int
	demonEditSchedule    string
	demonEditCommand     string
	demonEditDescription string
	demonEditEnabled     string
)

func init() {
	demonCreateCmd.Flags().StringVar(&demonSchedule, "schedule", "", "Cron schedule (required)")
	demonCreateCmd.Flags().StringVar(&demonCommand, "cmd", "", "Command to execute (required)")
	_ = demonCreateCmd.MarkFlagRequired("schedule")
	_ = demonCreateCmd.MarkFlagRequired("cmd")

	demonLogsCmd.Flags().IntVar(&demonTail, "tail", 0, "Show only the last N entries")

	demonEditCmd.Flags().StringVar(&demonEditSchedule, "schedule", "", "New cron schedule")
	demonEditCmd.Flags().StringVar(&demonEditCommand, "cmd", "", "New command to execute")
	demonEditCmd.Flags().StringVar(&demonEditDescription, "description", "", "New description")
	demonEditCmd.Flags().StringVar(&demonEditEnabled, "enabled", "", "Enable/disable demon (true/false)")

	demonCmd.AddCommand(demonCreateCmd)
	demonCmd.AddCommand(demonListCmd)
	demonCmd.AddCommand(demonShowCmd)
	demonCmd.AddCommand(demonDeleteCmd)
	demonCmd.AddCommand(demonRunCmd)
	demonCmd.AddCommand(demonEnableCmd)
	demonCmd.AddCommand(demonDisableCmd)
	demonCmd.AddCommand(demonLogsCmd)
	demonCmd.AddCommand(demonEditCmd)
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

func runDemonEdit(cmd *cobra.Command, args []string) error {
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

	// Check if any flags were provided
	hasFlags := demonEditSchedule != "" || demonEditCommand != "" ||
		demonEditDescription != "" || demonEditEnabled != ""

	if hasFlags {
		// Update using flags
		return updateDemonWithFlags(cmd, store, name)
	}

	// No flags - open in editor
	return editDemonInEditor(cmd, store, name, d)
}

func updateDemonWithFlags(cmd *cobra.Command, store *demon.Store, name string) error {
	var changes []string

	err := store.Update(name, func(d *demon.Demon) {
		if demonEditSchedule != "" {
			d.Schedule = demonEditSchedule
			changes = append(changes, fmt.Sprintf("schedule: %s", demonEditSchedule))
		}
		if demonEditCommand != "" {
			d.Command = demonEditCommand
			changes = append(changes, fmt.Sprintf("command: %s", demonEditCommand))
		}
		if demonEditDescription != "" {
			d.Description = demonEditDescription
			changes = append(changes, fmt.Sprintf("description: %s", demonEditDescription))
		}
		if demonEditEnabled != "" {
			enabled := demonEditEnabled == "true" || demonEditEnabled == "1" || demonEditEnabled == "yes"
			d.Enabled = enabled
			changes = append(changes, fmt.Sprintf("enabled: %t", enabled))
		}
	})
	if err != nil {
		return fmt.Errorf("failed to update demon: %w", err)
	}

	cmd.Printf("✓ Updated demon %q\n", name)
	for _, change := range changes {
		cmd.Printf("  %s\n", change)
	}

	return nil
}

func editDemonInEditor(cmd *cobra.Command, store *demon.Store, name string, d *demon.Demon) error {
	// Create temp file with demon config as JSON
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("demon-%s-*.json", name))
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()

	// Write current config to temp file
	configData := map[string]interface{}{
		"name":        d.Name,
		"schedule":    d.Schedule,
		"command":     d.Command,
		"description": d.Description,
		"enabled":     d.Enabled,
	}
	encoder := json.NewEncoder(tmpFile)
	encoder.SetIndent("", "  ")
	if encodeErr := encoder.Encode(configData); encodeErr != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write config: %w", encodeErr)
	}
	_ = tmpFile.Close()

	// Open editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	// #nosec G204 - editor command is from user's EDITOR env var
	editorCmd := exec.CommandContext(context.Background(), editor, tmpFile.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if runErr := editorCmd.Run(); runErr != nil {
		return fmt.Errorf("failed to open editor: %w", runErr)
	}

	// Read updated config
	updatedData, err := os.ReadFile(tmpFile.Name()) //nolint:gosec // G304: temp file we created
	if err != nil {
		return fmt.Errorf("failed to read updated config: %w", err)
	}

	var updated map[string]interface{}
	if unmarshalErr := json.Unmarshal(updatedData, &updated); unmarshalErr != nil {
		return fmt.Errorf("invalid JSON in edited file: %w", unmarshalErr)
	}

	// Apply updates
	err = store.Update(name, func(demon *demon.Demon) {
		if schedule, ok := updated["schedule"].(string); ok {
			demon.Schedule = schedule
		}
		if command, ok := updated["command"].(string); ok {
			demon.Command = command
		}
		if description, ok := updated["description"].(string); ok {
			demon.Description = description
		}
		if enabled, ok := updated["enabled"].(bool); ok {
			demon.Enabled = enabled
		}
	})
	if err != nil {
		return fmt.Errorf("failed to save changes: %w", err)
	}

	cmd.Printf("✓ Updated demon %q\n", name)
	return nil
}
