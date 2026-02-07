package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
)

var queueCmd = &cobra.Command{
	Use:   "queue [item-id]",
	Short: "Manage the work queue",
	Long: `List and manage work items in the queue.

Example:
  bc queue                            # list all items
  bc queue work-001                   # show full details for work-001
  bc queue --detail work-001          # same as above
  bc queue --json                     # list as JSON
  bc queue add "Fix auth bug"         # add work item
  bc queue assign work-001 worker-01  # assign to worker
  bc queue load                       # populate from beads issues`,
	Args: cobra.MaximumNArgs(1),
	RunE: runQueueList,
}

var queueAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a work item to the queue",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueAdd,
}

var queueAssignCmd = &cobra.Command{
	Use:   "assign <item-id> <agent>",
	Short: "Assign a work item to an agent",
	Args:  cobra.ExactArgs(2),
	RunE:  runQueueAssign,
}

var queueLoadCmd = &cobra.Command{
	Use:   "load",
	Short: "Populate queue from beads issues",
	RunE:  runQueueLoad,
}

var queueCompleteCmd = &cobra.Command{
	Use:   "complete <item-id>",
	Short: "Mark a work item as done (e.g. when work was done outside agent session)",
	Long:  `Marks the item done, saves the queue, and closes the linked beads issue if any.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueComplete,
}

var (
	queueDesc       string
	queueDetailID   string
	queueFilterType string
	queueAddType    string
)

func init() {
	queueAddCmd.Flags().StringVarP(&queueDesc, "desc", "d", "", "Work item description")
	queueAddCmd.Flags().StringVar(&queueAddType, "type", "code", "Task type (code, review, merge, qa)")
	queueCmd.Flags().StringVar(&queueDetailID, "detail", "", "Show full details for a specific work item")
	queueCmd.Flags().StringVar(&queueFilterType, "type", "", "Filter by task type (code, review, merge, qa)")
	queueCmd.AddCommand(queueAddCmd)
	queueCmd.AddCommand(queueAssignCmd)
	queueCmd.AddCommand(queueCompleteCmd)
	queueCmd.AddCommand(queueLoadCmd)
	rootCmd.AddCommand(queueCmd)
}

func loadQueue(ws interface{ StateDir() string }) (*queue.Queue, error) {
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	if err := q.Load(); err != nil {
		return nil, fmt.Errorf("failed to load queue: %w", err)
	}
	return q, nil
}

func runQueueList(cmd *cobra.Command, args []string) error {
	// Check if detail view is requested via --detail flag or positional arg
	detailID := queueDetailID
	if detailID == "" && len(args) == 1 {
		detailID = args[0]
	}
	if detailID != "" {
		return runQueueDetail(cmd, detailID)
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	q, err := loadQueue(ws)
	if err != nil {
		return err
	}

	// Get items - filter by type if specified
	var items []queue.WorkItem
	if queueFilterType != "" {
		taskType := queue.TaskType(queueFilterType)
		if !isValidTaskType(taskType) {
			return fmt.Errorf("invalid task type: %s (valid: code, review, merge, qa)", queueFilterType)
		}
		items = q.ListByType(taskType)
	} else {
		items = q.ListAll()
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(items)
	}

	if len(items) == 0 {
		fmt.Println("No work items in queue")
		fmt.Println()
		fmt.Println("Run 'bc queue load' to populate from beads issues")
		fmt.Println("Run 'bc queue add <title>' to add items manually")
		return nil
	}

	// Table header
	fmt.Printf("%-10s %-8s %-10s %-10s %-12s %-35s %s\n", "ID", "TYPE", "STATUS", "MERGE", "ASSIGNED", "TITLE", "BEADS")
	fmt.Println(strings.Repeat("-", 105))

	for _, item := range items {
		assigned := item.AssignedTo
		if assigned == "" {
			assigned = "-"
		}
		beadsID := item.BeadsID
		if beadsID == "" {
			beadsID = "-"
		}
		title := item.Title
		if len(title) > 33 {
			title = title[:30] + "..."
		}

		stateStr := colorQueueStatus(item.Status)
		mergeStr := colorMergeStatus(item.Merge)
		typeStr := colorTaskType(item.EffectiveType())

		fmt.Printf("%-10s %s %s %s %-12s %-35s %s\n",
			item.ID, typeStr, stateStr, mergeStr, assigned, title, beadsID,
		)
	}

	fmt.Println()
	stats := q.Stats()
	fmt.Printf("Total: %d | Pending: %d | Assigned: %d | Working: %d | Done: %d | Failed: %d | Merged: %d | Unmerged: %d\n",
		stats.Total, stats.Pending, stats.Assigned, stats.Working, stats.Done, stats.Failed, stats.Merged, stats.Unmerged)

	return nil
}

func runQueueDetail(cmd *cobra.Command, itemID string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	q, err := loadQueue(ws)
	if err != nil {
		return err
	}
	item := q.Get(itemID)
	if item == nil {
		return fmt.Errorf("work item %s not found", itemID)
	}

	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return err
	}
	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(item)
	}

	assigned := item.AssignedTo
	if assigned == "" {
		assigned = "-"
	}
	beadsID := item.BeadsID
	if beadsID == "" {
		beadsID = "-"
	}

	fmt.Printf("ID:        %s\n", item.ID)
	fmt.Printf("Title:     %s\n", item.Title)
	fmt.Printf("Type:      %s\n", item.EffectiveType())
	fmt.Printf("Status:    %s\n", item.Status)
	fmt.Printf("Assigned:  %s\n", assigned)
	fmt.Printf("Beads ID:  %s\n", beadsID)
	fmt.Printf("Created:   %s\n", item.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated:   %s\n", item.UpdatedAt.Format(time.RFC3339))

	// Merge info
	if item.Merge != "" {
		fmt.Printf("\nMerge:\n")
		fmt.Printf("  Status:  %s\n", item.Merge)
		if item.Branch != "" {
			fmt.Printf("  Branch:  %s\n", item.Branch)
		}
		if item.MergeCommit != "" {
			fmt.Printf("  Commit:  %s\n", item.MergeCommit)
		}
		if !item.MergedAt.IsZero() {
			fmt.Printf("  Merged:  %s\n", item.MergedAt.Format(time.RFC3339))
		}
	}

	if item.Description != "" {
		fmt.Printf("\nDescription:\n  %s\n", strings.ReplaceAll(item.Description, "\n", "\n  "))
	}

	// Note: Issue details now come from GitHub Issues (beads removed)
	if item.BeadsID != "" {
		fmt.Printf("\nLegacy Bead ID: %s\n", item.BeadsID)
	}

	return nil
}

func runQueueAdd(cmd *cobra.Command, args []string) error {
	title := strings.TrimSpace(args[0])
	if title == "" {
		return fmt.Errorf("work item title cannot be empty")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	q, err := loadQueue(ws)
	if err != nil {
		return err
	}

	// Parse task type
	taskType := queue.TaskType(queueAddType)
	if !isValidTaskType(taskType) {
		return fmt.Errorf("invalid task type: %s (valid: code, review, merge, qa)", queueAddType)
	}

	item := q.AddWithType(title, queueDesc, "", taskType)
	if err := q.Save(); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	fmt.Printf("Added %s (%s): %s\n", item.ID, item.Type, item.Title)
	return nil
}

func runQueueAssign(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	itemID := args[0]
	agentName := args[1]

	q, err := loadQueue(ws)
	if err != nil {
		return err
	}

	// Get item before assigning to check BeadsID
	item := q.Get(itemID)
	if item == nil {
		return fmt.Errorf("work item %s not found", itemID)
	}

	if err := q.Assign(itemID, agentName); err != nil {
		return err
	}
	if err := q.Save(); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	// Log event
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	_ = log.Append(events.Event{
		Type:  events.WorkAssigned,
		Agent: agentName,
		Data:  map[string]any{"work_id": itemID},
	})

	// Note: Issue tracking now uses GitHub Issues (beads removed)

	fmt.Printf("Assigned %s to %s\n", itemID, agentName)
	return nil
}

func runQueueLoad(cmd *cobra.Command, args []string) error {
	// Note: beads package has been deprecated. Use GitHub Issues for task tracking.
	fmt.Println("The 'bc queue load' command is deprecated.")
	fmt.Println("Work items are now tracked via GitHub Issues.")
	fmt.Println()
	fmt.Println("To view issues:  gh issue list --state open")
	fmt.Println("To create:       gh issue create --title '<title>'")
	return nil
}

func runQueueComplete(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	itemID := args[0]
	q, err := loadQueue(ws)
	if err != nil {
		return err
	}
	item := q.Get(itemID)
	if item == nil {
		return fmt.Errorf("work item %s not found", itemID)
	}

	if err := q.UpdateStatus(itemID, queue.StatusDone); err != nil {
		return err
	}
	if err := q.Save(); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	_ = log.Append(events.Event{
		Type:    events.WorkCompleted,
		Message: fmt.Sprintf("marked %s done via bc queue complete", itemID),
		Data:    map[string]any{"work_id": itemID},
	})

	// Note: Issue tracking now uses GitHub Issues (beads removed)

	fmt.Printf("Marked %s done", itemID)
	if item.BeadsID != "" {
		fmt.Printf(" (closed %s)", item.BeadsID)
	}
	fmt.Println()
	return nil
}

func colorMergeStatus(s queue.MergeStatus) string {
	const (
		reset  = "\033[0m"
		green  = "\033[32m"
		yellow = "\033[33m"
		red    = "\033[31m"
		gray   = "\033[90m"
	)

	if s == "" {
		return fmt.Sprintf("%-10s", "-")
	}

	padded := fmt.Sprintf("%-10s", s)

	switch s {
	case queue.MergeMerged:
		return green + padded + reset
	case queue.MergeUnmerged:
		return yellow + padded + reset
	case queue.MergeMerging:
		return yellow + padded + reset
	case queue.MergeConflict:
		return red + padded + reset
	default:
		return gray + padded + reset
	}
}

func colorQueueStatus(s queue.ItemStatus) string {
	const (
		reset  = "\033[0m"
		green  = "\033[32m"
		yellow = "\033[33m"
		red    = "\033[31m"
		cyan   = "\033[36m"
		gray   = "\033[90m"
	)

	padded := fmt.Sprintf("%-10s", s)

	switch s {
	case queue.StatusPending:
		return cyan + padded + reset
	case queue.StatusAssigned:
		return yellow + padded + reset
	case queue.StatusWorking:
		return green + padded + reset
	case queue.StatusDone:
		return green + padded + reset
	case queue.StatusFailed:
		return red + padded + reset
	default:
		return padded
	}
}

func colorTaskType(t queue.TaskType) string {
	const (
		reset  = "\033[0m"
		blue   = "\033[34m"
		green  = "\033[32m"
		yellow = "\033[33m"
		cyan   = "\033[36m"
	)

	padded := fmt.Sprintf("%-8s", t)

	switch t {
	case queue.TaskTypeCode:
		return cyan + padded + reset
	case queue.TaskTypeReview:
		return blue + padded + reset
	case queue.TaskTypeMerge:
		return green + padded + reset
	case queue.TaskTypeQA:
		return yellow + padded + reset
	default:
		return padded
	}
}

func isValidTaskType(t queue.TaskType) bool {
	switch t {
	case queue.TaskTypeCode, queue.TaskTypeReview, queue.TaskTypeMerge, queue.TaskTypeQA:
		return true
	default:
		return false
	}
}
