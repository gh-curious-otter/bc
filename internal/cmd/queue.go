package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/github"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage work queue",
	Long: `Manage the work queue for tracking tasks and epics.

The work queue displays GitHub issues labeled as 'task' or 'epic'.
Use this command to view, add, and manage work items.

Examples:
  bc queue                    # list all work items
  bc queue list               # list all work items
  bc queue add "Fix bug"      # add a task to the queue
  bc queue add "New feature" --epic  # add an epic`,
	Args: cobra.NoArgs,
	RunE: runQueueList,
}

var queueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List work queue items",
	Long:  `List all work items (tasks and epics) from GitHub Issues.`,
	RunE:  runQueueList,
}

var queueAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a work item to the queue",
	Long: `Add a new task or epic to the work queue.

Creates a GitHub issue with the 'task' label (or 'epic' if --epic is specified).

Examples:
  bc queue add "Fix login bug"
  bc queue add "Fix login bug" -d "Users can't login with SSO"
  bc queue add "New auth system" --epic
  bc queue add "Add tests" --label priority-high`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueAdd,
}

func init() {
	// Add subcommands
	queueCmd.AddCommand(queueListCmd)
	queueCmd.AddCommand(queueAddCmd)

	// Add flags to queue add command
	queueAddCmd.Flags().StringP("description", "d", "", "Description/body for the work item")
	queueAddCmd.Flags().Bool("epic", false, "Create as an epic instead of a task")
	queueAddCmd.Flags().StringSlice("label", nil, "Additional labels to apply")

	// Disabled: work management is done through GitHub
	// rootCmd.AddCommand(queueCmd)
}

// QueueItem represents a work item in the queue.
type QueueItem struct {
	Title  string   `json:"title"`
	Type   string   `json:"type"` // "epic" or "task"
	State  string   `json:"state"`
	URL    string   `json:"url,omitempty"`
	Labels []string `json:"labels,omitempty"`
	Number int      `json:"number"`
}

func runQueueList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	issues, err := github.ListIssues(ws.RootDir)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	// Filter to only epic and task labels
	var items []QueueItem
	for _, issue := range issues {
		itemType := getItemType(issue)
		if itemType == "" {
			continue // Not a task or epic
		}
		items = append(items, QueueItem{
			Number: issue.Number,
			Title:  issue.Title,
			Type:   itemType,
			State:  issue.State,
			Labels: issue.Labels,
		})
	}

	// Sort by number (newest first)
	slices.SortFunc(items, func(a, b QueueItem) int {
		return b.Number - a.Number
	})

	// Check for JSON output
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}

	// Display table format
	if len(items) == 0 {
		fmt.Println("No work items in queue")
		fmt.Println()
		fmt.Println("Run 'bc queue add <title>' to add a task")
		fmt.Println("Run 'bc queue add <title> --epic' to add an epic")
		return nil
	}

	// Table header
	fmt.Printf("%-6s %-6s %-50s %s\n", "ID", "TYPE", "TITLE", "STATE")
	fmt.Println(strings.Repeat("-", 80))

	for _, item := range items {
		title := item.Title
		if len(title) > 48 {
			title = title[:45] + "..."
		}
		fmt.Printf("#%-5d %-6s %-50s %s\n", item.Number, item.Type, title, item.State)
	}

	return nil
}

func runQueueAdd(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	title := args[0]

	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return err
	}

	isEpic, err := cmd.Flags().GetBool("epic")
	if err != nil {
		return err
	}

	extraLabels, err := cmd.Flags().GetStringSlice("label")
	if err != nil {
		return err
	}

	// Build labels
	var labels []string
	if isEpic {
		labels = append(labels, "epic")
	} else {
		labels = append(labels, "task")
	}
	labels = append(labels, extraLabels...)

	// Create the issue
	if err := createIssueWithLabels(ws.RootDir, title, description, labels); err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	itemType := "task"
	if isEpic {
		itemType = "epic"
	}
	fmt.Printf("Created %s: %s\n", itemType, title)

	return nil
}

// getItemType returns "epic", "task", or "" based on issue labels and title.
// Epic takes precedence over task when both labels are present.
func getItemType(issue github.Issue) string {
	// Check epic first so it takes precedence
	for _, label := range issue.Labels {
		if label == "epic" {
			return "epic"
		}
	}
	for _, label := range issue.Labels {
		if label == "task" {
			return "task"
		}
	}
	// Fallback: check for [EPIC] prefix in title
	if strings.HasPrefix(issue.Title, "[EPIC]") || strings.HasPrefix(issue.Title, "[Epic]") {
		return "epic"
	}
	return ""
}

// createIssueWithLabels creates a GitHub issue with the specified labels.
func createIssueWithLabels(workspacePath, title, body string, labels []string) error {
	args := []string{"issue", "create", "--title", title}
	if body != "" {
		args = append(args, "--body", body)
	}
	for _, label := range labels {
		args = append(args, "--label", label)
	}

	cmd := exec.CommandContext(context.Background(), "gh", args...) //nolint:gosec // gh command with trusted args
	cmd.Dir = workspacePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
