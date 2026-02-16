package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/demon"
	"github.com/rpuneet/bc/pkg/integrations"
	testpkg "github.com/rpuneet/bc/pkg/testing"
	"github.com/rpuneet/bc/pkg/workspace"
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

var demonTestCmd = &cobra.Command{
	Use:   "test <pattern>",
	Short: "Run tests and create GitHub issues for failures",
	Long: `Run Go tests and automatically create GitHub issues for test failures.

Parses go test -json output, identifies failures, and creates GitHub issues with failure details.

Examples:
  bc demon test ./...                          # Test all packages once
  bc demon test ./pkg/agent                    # Test specific package
  bc demon test ./... --create-demon           # Create hourly testing demon
  bc demon test ./... --create-demon --schedule '*/30 * * * *'  # Custom schedule`,
	Args: cobra.ExactArgs(1),
	RunE: runDemonTest,
}

var demonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the demon scheduler",
	Long: `Start the background scheduler that automatically runs demons.

The scheduler checks enabled demons every 30 seconds and runs any
that are due based on their cron schedule.

Examples:
  bc demon start       # Start the scheduler
  bc demon status      # Check if scheduler is running
  bc demon stop        # Stop the scheduler`,
	RunE: runDemonStart,
}

var demonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the demon scheduler",
	Long: `Stop the background scheduler.

This will stop automatic execution of demons. You can still
run demons manually with 'bc demon run <name>'.

Examples:
  bc demon stop        # Stop the scheduler
  bc demon status      # Verify scheduler stopped`,
	RunE: runDemonStop,
}

var demonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show scheduler status and upcoming runs",
	Long: `Show the status of the demon scheduler and next scheduled runs.

Examples:
  bc demon status      # Show scheduler status
  bc demon status --json  # JSON output for TUI`,
	RunE: runDemonStatus,
}

// Hidden command for internal use - runs the scheduler loop
var demonSchedulerLoopCmd = &cobra.Command{
	Use:    "scheduler-loop",
	Hidden: true,
	RunE:   runDemonSchedulerLoop,
}

var schedulerLoopRoot string

var (
	demonSchedule        string
	demonCommand         string
	demonPrompt          string
	demonPromptFile      string
	demonTail            int
	demonEditSchedule    string
	demonEditCommand     string
	demonEditPrompt      string
	demonEditPromptFile  string
	demonEditDescription string
	demonEditEnabled     string
	demonTestCreateDemon bool
	demonTestSchedule    string
)

func init() {
	demonCreateCmd.Flags().StringVar(&demonSchedule, "schedule", "", "Cron schedule (required)")
	demonCreateCmd.Flags().StringVar(&demonCommand, "cmd", "", "Command to execute (required)")
	demonCreateCmd.Flags().StringVar(&demonPrompt, "prompt", "", "Inline prompt for AI-powered tasks (optional)")
	demonCreateCmd.Flags().StringVar(&demonPromptFile, "prompt-file", "", "Path to prompt file (optional)")
	_ = demonCreateCmd.MarkFlagRequired("schedule")
	_ = demonCreateCmd.MarkFlagRequired("cmd")

	demonLogsCmd.Flags().IntVar(&demonTail, "tail", 0, "Show only the last N entries")

	demonEditCmd.Flags().StringVar(&demonEditSchedule, "schedule", "", "New cron schedule")
	demonEditCmd.Flags().StringVar(&demonEditCommand, "cmd", "", "New command to execute")
	demonEditCmd.Flags().StringVar(&demonEditPrompt, "prompt", "", "New prompt for AI-powered tasks")
	demonEditCmd.Flags().StringVar(&demonEditPromptFile, "prompt-file", "", "Path to new prompt file")
	demonEditCmd.Flags().StringVar(&demonEditDescription, "description", "", "New description")
	demonEditCmd.Flags().StringVar(&demonEditEnabled, "enabled", "", "Enable/disable demon (true/false)")

	demonTestCmd.Flags().BoolVar(&demonTestCreateDemon, "create-demon", false, "Create a demon for scheduled testing")
	demonTestCmd.Flags().StringVar(&demonTestSchedule, "schedule", "0 * * * *", "Cron schedule for testing demon (default: hourly)")

	demonSchedulerLoopCmd.Flags().StringVar(&schedulerLoopRoot, "root", "", "Workspace root directory")
	_ = demonSchedulerLoopCmd.MarkFlagRequired("root")

	demonCmd.AddCommand(demonCreateCmd)
	demonCmd.AddCommand(demonListCmd)
	demonCmd.AddCommand(demonShowCmd)
	demonCmd.AddCommand(demonDeleteCmd)
	demonCmd.AddCommand(demonRunCmd)
	demonCmd.AddCommand(demonEnableCmd)
	demonCmd.AddCommand(demonDisableCmd)
	demonCmd.AddCommand(demonLogsCmd)
	demonCmd.AddCommand(demonEditCmd)
	demonCmd.AddCommand(demonTestCmd)
	demonCmd.AddCommand(demonStartCmd)
	demonCmd.AddCommand(demonStopCmd)
	demonCmd.AddCommand(demonStatusCmd)
	demonCmd.AddCommand(demonSchedulerLoopCmd)
	rootCmd.AddCommand(demonCmd)
}

// validIdentifier checks if a string is a valid identifier (alphanumeric, dash, underscore)
func validIdentifier(s string) bool {
	if s == "" {
		return false
	}
	// Must start with letter or underscore, then letters, numbers, dashes, underscores
	pattern := regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)
	return pattern.MatchString(s)
}

// validateDemonName validates a demon name and returns an error if invalid.
func validateDemonName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("demon name cannot be empty")
	}
	if !validIdentifier(name) {
		return fmt.Errorf("demon name %q must start with a letter or underscore and contain only letters, numbers, dashes, and underscores", name)
	}
	return nil
}

func runDemonCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	name := strings.TrimSpace(args[0])
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

	// Validate command is not empty
	if strings.TrimSpace(demonCommand) == "" {
		return fmt.Errorf("demon command cannot be empty")
	}

	store := demon.NewStore(ws.RootDir)

	d, err := store.CreateWithPrompt(name, demonSchedule, demonCommand, demonPrompt, demonPromptFile)
	if err != nil {
		return err
	}

	cmd.Printf("Created demon %q\n", d.Name)
	cmd.Printf("  Schedule: %s\n", d.Schedule)
	cmd.Printf("  Command:  %s\n", d.Command)
	if d.Prompt != "" {
		cmd.Printf("  Prompt:   %s\n", d.Prompt)
	}
	if d.PromptFile != "" {
		cmd.Printf("  Prompt File: %s\n", d.PromptFile)
	}
	if !d.NextRun.IsZero() {
		cmd.Printf("  Next run: %s\n", d.NextRun.Format("2006-01-02 15:04:05"))
	}

	return nil
}

func runDemonList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	store := demon.NewStore(ws.RootDir)
	demons, err := store.List()
	if err != nil {
		return err
	}

	// Check for JSON output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Return empty array if no demons, for TUI compatibility
		if demons == nil {
			demons = []*demon.Demon{}
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(demons)
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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		return errNotInWorkspace(err)
	}

	name := args[0]
	if nameErr := validateDemonName(name); nameErr != nil {
		return nameErr
	}

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
		demonEditDescription != "" || demonEditEnabled != "" ||
		demonEditPrompt != "" || demonEditPromptFile != ""

	if hasFlags {
		// Update using flags
		return updateDemonWithFlags(cmd, store, name)
	}

	// No flags - open in editor
	return editDemonInEditor(cmd, store, name, d)
}

func updateDemonWithFlags(cmd *cobra.Command, store *demon.Store, name string) error {
	// Validate schedule if provided
	if demonEditSchedule != "" {
		if _, err := demon.ParseCron(demonEditSchedule); err != nil {
			return fmt.Errorf("invalid cron schedule %q: %w", demonEditSchedule, err)
		}
	}

	// Validate command if provided (not empty)
	if demonEditCommand != "" && strings.TrimSpace(demonEditCommand) == "" {
		return fmt.Errorf("demon command cannot be empty")
	}

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
		if demonEditPrompt != "" {
			d.Prompt = demonEditPrompt
			d.PromptFile = "" // Clear prompt file if inline prompt provided
			changes = append(changes, fmt.Sprintf("prompt: %s", demonEditPrompt))
		}
		if demonEditPromptFile != "" {
			if _, err := os.Stat(demonEditPromptFile); err == nil {
				d.PromptFile = demonEditPromptFile
				d.Prompt = "" // Clear inline prompt if prompt file provided
				changes = append(changes, fmt.Sprintf("prompt-file: %s", demonEditPromptFile))
			}
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

func runDemonTest(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("test pattern required (e.g., ./...)")
	}
	pattern := args[0]

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Check GitHub integration enabled
	if ws.V2Config == nil || ws.V2Config.Tools.GitHub == nil || !ws.V2Config.Tools.GitHub.Enabled {
		return fmt.Errorf("GitHub integration not enabled - set [tools.github] enabled = true in .bc/config.toml")
	}

	if demonTestCreateDemon {
		// Create demon for scheduled testing
		testCmd := fmt.Sprintf("bc demon test %s", pattern)
		store := demon.NewStore(ws.RootDir)
		_, err := store.Create("test-watcher", demonTestSchedule, testCmd)
		if err != nil {
			return fmt.Errorf("failed to create demon: %w", err)
		}
		cmd.Printf("Created demon 'test-watcher' with schedule: %s\n", demonTestSchedule)
		cmd.Printf("Demon will run: %s\n", testCmd)
		return nil
	}

	// Run tests with JSON output
	return runTestsAndReportIssues(cmd, ws, pattern)
}

func runDemonStart(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	scheduler := demon.NewScheduler(ws.RootDir)

	if err := scheduler.Start(); err != nil {
		return err
	}

	status, _ := scheduler.Status()
	cmd.Printf("Scheduler started (PID %d)\n", status.PID)
	cmd.Println("Demons will run automatically based on their schedules.")
	cmd.Println()
	cmd.Println("Use 'bc demon status' to see upcoming runs.")
	cmd.Println("Use 'bc demon stop' to stop the scheduler.")

	return nil
}

func runDemonStop(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	scheduler := demon.NewScheduler(ws.RootDir)

	if err := scheduler.Stop(); err != nil {
		return err
	}

	cmd.Println("Scheduler stopped")
	cmd.Println("Demons will no longer run automatically.")
	cmd.Println()
	cmd.Println("Use 'bc demon run <name>' to run demons manually.")

	return nil
}

func runDemonStatus(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	scheduler := demon.NewScheduler(ws.RootDir)
	status, err := scheduler.Status()
	if err != nil {
		return err
	}

	// Check for JSON output
	jsonOutput, _ := cmd.Flags().GetBool("json")
	if jsonOutput {
		// Include next runs in JSON output
		nextRuns, _ := scheduler.GetNextRuns()
		output := struct {
			Status   *demon.SchedulerStatus `json:"status"`
			NextRuns []demon.DemonNextRun   `json:"next_runs"`
		}{
			Status:   status,
			NextRuns: nextRuns,
		}
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(output)
	}

	// Text output
	if status.Running {
		cmd.Printf("Scheduler: running (PID %d)\n", status.PID)
		if status.Uptime != "" {
			cmd.Printf("Uptime:    %s\n", status.Uptime)
		}
	} else {
		cmd.Println("Scheduler: stopped")
		cmd.Println()
		cmd.Println("Start with: bc demon start")
		return nil
	}

	// Show next scheduled runs
	nextRuns, err := scheduler.GetNextRuns()
	if err != nil {
		return err
	}

	if len(nextRuns) == 0 {
		cmd.Println()
		cmd.Println("No enabled demons configured.")
		cmd.Println("Create one with: bc demon create <name> --schedule '<cron>' --cmd '<command>'")
		return nil
	}

	cmd.Println()
	cmd.Println("Upcoming runs:")
	cmd.Printf("%-20s %-24s %s\n", "DEMON", "NEXT RUN", "COMMAND")
	cmd.Println("--------------------------------------------------------------------")

	for _, run := range nextRuns {
		nextRunStr := "never"
		if !run.NextRun.IsZero() {
			nextRunStr = run.NextRun.Local().Format("2006-01-02 15:04:05")
		}
		command := run.Command
		if len(command) > 25 {
			command = command[:22] + "..."
		}
		cmd.Printf("%-20s %-24s %s\n", run.Name, nextRunStr, command)
	}

	return nil
}

func runDemonSchedulerLoop(cmd *cobra.Command, args []string) error {
	if schedulerLoopRoot == "" {
		return fmt.Errorf("--root flag is required")
	}

	scheduler := demon.NewScheduler(schedulerLoopRoot)
	return scheduler.RunLoop(cmd.Context())
}

func runTestsAndReportIssues(cmd *cobra.Command, ws *workspace.Workspace, pattern string) error {
	ctx := context.Background()

	// Run go test -json
	testCmd := exec.CommandContext(ctx, "go", "test", "-json", pattern)
	testCmd.Dir = ws.RootDir
	output, testErr := testCmd.CombinedOutput()

	// Parse test output
	failures, parseErr := testpkg.ParseTestJSON(bytes.NewReader(output))
	if parseErr != nil {
		return fmt.Errorf("failed to parse test output: %w", parseErr)
	}

	if len(failures) == 0 {
		cmd.Println("All tests passed ✓")
		return nil
	}

	// Create GitHub integration
	gh, err := integrations.NewGitHubIntegration(ws)
	if err != nil {
		return err
	}

	// Check auth
	if err := gh.CheckAuth(ctx); err != nil {
		return fmt.Errorf("GitHub authentication failed: %w", err)
	}

	// Create issues for failures
	createdCount := 0
	existingCount := 0
	for _, failure := range failures {
		title := failure.IssueTitle()
		body := failure.FormatIssueBody()

		// Check if issue already exists
		searchQuery := fmt.Sprintf("%s in:title", failure.FullName)
		exists, checkErr := gh.IssueExists(ctx, searchQuery)
		if checkErr != nil {
			cmd.Printf("Warning: failed to check for existing issue for %s: %v\n", failure.FullName, checkErr)
		}

		if exists {
			existingCount++
			cmd.Printf("Issue already exists for %s\n", failure.FullName)
			continue
		}

		// Create new issue
		issueURL, createErr := gh.CreateIssue(ctx, title, body, []string{"bug", "automated", "test-failure"})
		if createErr != nil {
			cmd.Printf("Failed to create issue for %s: %v\n", failure.FullName, createErr)
			continue
		}

		createdCount++
		cmd.Printf("Created issue: %s\n", issueURL)
	}

	cmd.Printf("\nTest Results: %d failed, %d new issues created, %d existing issues\n",
		len(failures), createdCount, existingCount)

	// Return error if tests failed
	if testErr != nil {
		return fmt.Errorf("tests failed: %w", testErr)
	}

	return nil
}
