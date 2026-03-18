package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/cron"
	"github.com/rpuneet/bc/pkg/ui"
)

var cronCmd = &cobra.Command{
	Use:     "cron",
	Aliases: []string{"cr"},
	Short:   "Manage scheduled agent tasks",
	Long: `Manage cron jobs that trigger agent prompts or shell commands on a schedule.

Cron expressions use standard 5-field format:
  ┌────── minute (0-59)
  │ ┌──── hour (0-23)
  │ │ ┌── day of month (1-31)
  │ │ │ ┌ month (1-12)
  │ │ │ │ ┌ day of week (0-6, 0=Sun)
  * * * * *

Examples:
  bc cron add daily-lint --schedule "0 9 * * *" --agent qa-01 --prompt "Run make lint"
  bc cron list                          # List all cron jobs
  bc cron show daily-lint               # Show job details
  bc cron enable daily-lint             # Enable a disabled job
  bc cron disable daily-lint            # Disable without deleting
  bc cron run daily-lint                # Trigger manually
  bc cron logs daily-lint --last 10     # Show last 10 executions
  bc cron remove daily-lint             # Delete a job`,
}

var cronAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new cron job",
	Long: `Create a new scheduled cron job.

One of --agent+--prompt or --command is required.

Examples:
  bc cron add daily-lint --schedule "0 9 * * *" --agent qa-01 --prompt "Run make lint and report"
  bc cron add hourly-check --schedule "0 * * * *" --command "make check"
  bc cron add weekday-standup --schedule "0 9 * * 1-5" --agent root --prompt "Send standup"`,
	Args: cobra.ExactArgs(1),
	RunE: runCronAdd,
}

var cronListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cron jobs",
	Long: `Display all scheduled cron jobs with their status.

Examples:
  bc cron list
  bc cron list --json`,
	RunE: runCronList,
}

var cronShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show cron job details",
	Long: `Display full details of a cron job including schedule and run history stats.

Examples:
  bc cron show daily-lint
  bc cron show daily-lint --json`,
	Args: cobra.ExactArgs(1),
	RunE: runCronShow,
}

var cronRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a cron job",
	Long: `Delete a cron job and its execution logs.

Examples:
  bc cron remove daily-lint`,
	Args: cobra.ExactArgs(1),
	RunE: runCronRemove,
}

var cronEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a cron job",
	Long: `Enable a disabled cron job. The next run time is recalculated from now.

Examples:
  bc cron enable daily-lint`,
	Args: cobra.ExactArgs(1),
	RunE: runCronEnable,
}

var cronDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a cron job",
	Long: `Disable a cron job without deleting it.

Examples:
  bc cron disable daily-lint`,
	Args: cobra.ExactArgs(1),
	RunE: runCronDisable,
}

var cronRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Manually trigger a cron job",
	Long: `Trigger a cron job immediately outside its normal schedule.
The job must be enabled. The daemon (bcd) executes the actual agent interaction;
this command records the trigger and updates run stats.

Examples:
  bc cron run daily-lint`,
	Args: cobra.ExactArgs(1),
	RunE: runCronRun,
}

var cronLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show execution history for a cron job",
	Long: `Display the execution log for a cron job.

Examples:
  bc cron logs daily-lint
  bc cron logs daily-lint --last 5
  bc cron logs daily-lint --json`,
	Args: cobra.ExactArgs(1),
	RunE: runCronLogs,
}

// Flags
var (
	cronAddSchedule string
	cronAddAgent    string
	cronAddPrompt   string
	cronAddCommand  string
	cronAddDisabled bool

	cronListJSON bool
	cronShowJSON bool
	cronLogsJSON bool
	cronLogsLast int
)

func init() {
	// add flags
	cronAddCmd.Flags().StringVar(&cronAddSchedule, "schedule", "", "5-field cron expression (required)")
	cronAddCmd.Flags().StringVar(&cronAddAgent, "agent", "", "Target agent name")
	cronAddCmd.Flags().StringVar(&cronAddPrompt, "prompt", "", "Prompt to send to the agent")
	cronAddCmd.Flags().StringVar(&cronAddCommand, "command", "", "Shell command to run (alternative to --agent+--prompt)")
	cronAddCmd.Flags().BoolVar(&cronAddDisabled, "disabled", false, "Create job in disabled state")
	_ = cronAddCmd.MarkFlagRequired("schedule")

	// list flags
	cronListCmd.Flags().BoolVar(&cronListJSON, "json", false, "Output as JSON")

	// show flags
	cronShowCmd.Flags().BoolVar(&cronShowJSON, "json", false, "Output as JSON")

	// logs flags
	cronLogsCmd.Flags().IntVar(&cronLogsLast, "last", 20, "Number of entries to show")
	cronLogsCmd.Flags().BoolVar(&cronLogsJSON, "json", false, "Output as JSON")

	// sub-commands
	cronCmd.AddCommand(cronAddCmd)
	cronCmd.AddCommand(cronListCmd)
	cronCmd.AddCommand(cronShowCmd)
	cronCmd.AddCommand(cronRemoveCmd)
	cronCmd.AddCommand(cronEnableCmd)
	cronCmd.AddCommand(cronDisableCmd)
	cronCmd.AddCommand(cronRunCmd)
	cronCmd.AddCommand(cronLogsCmd)

	rootCmd.AddCommand(cronCmd)
}

func openCronStore() (*cron.Store, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, errNotInWorkspace(err)
	}
	store, err := cron.Open(ws.RootDir)
	if err != nil {
		return nil, fmt.Errorf("open cron store: %w", err)
	}
	return store, nil
}

func runCronAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	if !validIdentifier(name) {
		return fmt.Errorf("invalid job name %q (use letters, numbers, dash, underscore)", name)
	}

	// Validate: need agent+prompt or command
	if cronAddCommand == "" && (cronAddAgent == "" || cronAddPrompt == "") {
		return fmt.Errorf("either --command or both --agent and --prompt are required")
	}

	if err := cron.ValidateSchedule(cronAddSchedule); err != nil {
		return fmt.Errorf("invalid schedule: %w", err)
	}

	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	job := &cron.Job{
		Name:      name,
		Schedule:  cronAddSchedule,
		AgentName: cronAddAgent,
		Prompt:    cronAddPrompt,
		Command:   cronAddCommand,
		Enabled:   !cronAddDisabled,
	}

	if err := store.AddJob(cmd.Context(), job); err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}

	// Fetch to show computed next_run
	added, err := store.GetJob(cmd.Context(), name)
	if err != nil || added == nil {
		fmt.Printf("✓ cron job %q added\n", name)
		return nil
	}

	fmt.Printf("✓ cron job %q added\n", name)
	fmt.Printf("  Schedule: %s\n", added.Schedule)
	if added.NextRun != nil {
		fmt.Printf("  Next run: %s\n", formatRelTime(*added.NextRun))
	}
	return nil
}

func runCronList(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	jobs, err := store.ListJobs(cmd.Context())
	if err != nil {
		return fmt.Errorf("list cron jobs: %w", err)
	}

	if cronListJSON {
		return printJSON(jobs)
	}

	if len(jobs) == 0 {
		fmt.Println("No cron jobs. Add one with: bc cron add NAME --schedule \"* * * * *\" --agent AGENT --prompt \"...\"")
		return nil
	}

	table := ui.NewTable("NAME", "SCHEDULE", "AGENT", "ENABLED", "NEXT RUN", "RUNS")
	for _, j := range jobs {
		enabled := "yes"
		if !j.Enabled {
			enabled = "no"
		}
		nextRun := "-"
		if j.NextRun != nil {
			nextRun = formatRelTime(*j.NextRun)
		}
		agent := j.AgentName
		if agent == "" {
			agent = "(shell)"
		}
		table.AddRow(j.Name, j.Schedule, agent, enabled, nextRun, fmt.Sprintf("%d", j.RunCount))
	}
	table.Print()
	return nil
}

func runCronShow(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	job, err := store.GetJob(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("get cron job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("cron job %q not found", args[0])
	}

	if cronShowJSON {
		return printJSON(job)
	}

	pairs := []string{
		"Name", job.Name,
		"Schedule", job.Schedule,
		"Enabled", boolStr(job.Enabled),
		"Run Count", fmt.Sprintf("%d", job.RunCount),
		"Created", job.CreatedAt.Format(time.RFC3339),
	}
	if job.AgentName != "" {
		pairs = append(pairs, "Agent", job.AgentName)
	}
	if job.Command != "" {
		pairs = append(pairs, "Command", job.Command)
	}
	if job.Prompt != "" {
		pairs = append(pairs, "Prompt", truncateStr(job.Prompt, 80))
	}
	if job.LastRun != nil {
		pairs = append(pairs, "Last Run", formatRelTime(*job.LastRun))
	}
	if job.NextRun != nil {
		pairs = append(pairs, "Next Run", formatRelTime(*job.NextRun))
	}

	ui.SimpleTable(pairs...)
	return nil
}

func runCronRemove(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	if err := store.DeleteJob(cmd.Context(), args[0]); err != nil {
		return err
	}
	fmt.Printf("✓ cron job %q removed\n", args[0])
	return nil
}

func runCronEnable(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	if err := store.SetEnabled(cmd.Context(), args[0], true); err != nil {
		return err
	}
	fmt.Printf("✓ cron job %q enabled\n", args[0])
	return nil
}

func runCronDisable(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	if err := store.SetEnabled(cmd.Context(), args[0], false); err != nil {
		return err
	}
	fmt.Printf("✓ cron job %q disabled\n", args[0])
	return nil
}

func runCronRun(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	job, err := store.GetJob(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("get cron job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("cron job %q not found", args[0])
	}
	if !job.Enabled {
		return fmt.Errorf("cron job %q is disabled — enable it first with: bc cron enable %s", args[0], args[0])
	}

	if err := store.RecordManualTrigger(cmd.Context(), args[0]); err != nil {
		return fmt.Errorf("record trigger: %w", err)
	}

	fmt.Printf("✓ triggered cron job %q\n", args[0])
	if job.AgentName != "" {
		fmt.Printf("  Agent:  %s\n", job.AgentName)
		fmt.Printf("  Prompt: %s\n", truncateStr(job.Prompt, 60))
		fmt.Println("  Note:   agent interaction handled by bcd daemon")
	} else if job.Command != "" {
		fmt.Printf("  Command: %s\n", job.Command)
		fmt.Println("  Note:    command execution handled by bcd daemon")
	}
	return nil
}

func runCronLogs(cmd *cobra.Command, args []string) error {
	store, err := openCronStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // best-effort

	// Verify job exists
	job, err := store.GetJob(cmd.Context(), args[0])
	if err != nil {
		return fmt.Errorf("get cron job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("cron job %q not found", args[0])
	}

	entries, err := store.GetLogs(cmd.Context(), args[0], cronLogsLast)
	if err != nil {
		return fmt.Errorf("get cron logs: %w", err)
	}

	if cronLogsJSON {
		return printJSON(entries)
	}

	if len(entries) == 0 {
		fmt.Printf("No execution history for %q\n", args[0])
		return nil
	}

	table := ui.NewTable("RUN AT", "STATUS", "DURATION", "COST")
	for _, e := range entries {
		dur := fmt.Sprintf("%dms", e.DurationMS)
		cost := "-"
		if e.CostUSD > 0 {
			cost = fmt.Sprintf("$%.4f", e.CostUSD)
		}
		table.AddRow(e.RunAt.Format("2006-01-02 15:04:05"), e.Status, dur, cost)
	}
	table.Print()
	return nil
}

// printJSON marshals v to indented JSON and writes it to stdout.
func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// formatRelTime formats a time as "in 2h3m" or "3h ago".
func formatRelTime(t time.Time) string {
	diff := time.Until(t)
	if diff > 0 {
		return "in " + formatDuration(diff)
	}
	return formatDuration(-diff) + " ago"
}

// boolStr returns "yes" or "no".
func boolStr(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// truncateStr truncates s to max runes, adding "…" if needed.
func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}

