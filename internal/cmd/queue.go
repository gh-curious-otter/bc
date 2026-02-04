package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rpuneet/bc/pkg/beads"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/queue"
	"github.com/spf13/cobra"
)

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage the work queue",
	Long: `List and manage work items in the queue.

Example:
  bc queue                            # list all items
  bc queue --json                     # list as JSON
  bc queue add "Fix auth bug"         # add work item
  bc queue assign work-001 worker-01  # assign to worker
  bc queue load                       # populate from beads issues`,
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

var queueDesc string

func init() {
	queueAddCmd.Flags().StringVarP(&queueDesc, "desc", "d", "", "Work item description")
	queueCmd.AddCommand(queueAddCmd)
	queueCmd.AddCommand(queueAssignCmd)
	queueCmd.AddCommand(queueLoadCmd)
	rootCmd.AddCommand(queueCmd)
}

func loadQueue(ws interface{ StateDir() string }) *queue.Queue {
	q := queue.New(filepath.Join(ws.StateDir(), "queue.json"))
	q.Load()
	return q
}

func runQueueList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	q := loadQueue(ws)
	items := q.ListAll()

	jsonOutput, _ := cmd.Flags().GetBool("json")
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
	fmt.Printf("%-10s %-10s %-15s %-40s %s\n", "ID", "STATUS", "ASSIGNED", "TITLE", "BEADS")
	fmt.Println(strings.Repeat("-", 90))

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
		if len(title) > 38 {
			title = title[:35] + "..."
		}

		stateStr := colorQueueStatus(item.Status)

		fmt.Printf("%-10s %s %-15s %-40s %s\n",
			item.ID, stateStr, assigned, title, beadsID,
		)
	}

	fmt.Println()
	stats := q.Stats()
	fmt.Printf("Total: %d | Pending: %d | Assigned: %d | Working: %d | Done: %d | Failed: %d\n",
		stats.Total, stats.Pending, stats.Assigned, stats.Working, stats.Done, stats.Failed)

	return nil
}

func runQueueAdd(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	q := loadQueue(ws)
	item := q.Add(args[0], queueDesc, "")
	if err := q.Save(); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	fmt.Printf("Added %s: %s\n", item.ID, item.Title)
	return nil
}

func runQueueAssign(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	itemID := args[0]
	agentName := args[1]

	q := loadQueue(ws)

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
	log.Append(events.Event{
		Type:  events.WorkAssigned,
		Agent: agentName,
		Data:  map[string]any{"work_id": itemID},
	})

	// Sync assignment to beads if linked
	if item.BeadsID != "" {
		if err := beads.AssignIssue(ws.RootDir, item.BeadsID, agentName); err != nil {
			// Log but don't fail - beads sync is best-effort
			log.Append(events.Event{
				Type:    events.AgentReport,
				Agent:   agentName,
				Message: fmt.Sprintf("warning: failed to assign beads issue %s: %v", item.BeadsID, err),
			})
		}
	}

	fmt.Printf("Assigned %s to %s\n", itemID, agentName)
	return nil
}

func runQueueLoad(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	q := loadQueue(ws)

	// Try ready issues first, fall back to all issues
	issues := beads.ReadyIssues(ws.RootDir)
	if len(issues) == 0 {
		issues = beads.ListIssues(ws.RootDir)
	}

	if len(issues) == 0 {
		fmt.Println("No beads issues found")
		return nil
	}

	added := 0
	for _, issue := range issues {
		if q.HasBeadsID(issue.ID) {
			continue
		}
		q.Add(issue.Title, issue.Description, issue.ID)
		added++
	}

	if err := q.Save(); err != nil {
		return fmt.Errorf("failed to save queue: %w", err)
	}

	// Log event
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))
	log.Append(events.Event{
		Type:    events.QueueLoaded,
		Message: fmt.Sprintf("loaded %d items from beads", added),
		Data:    map[string]any{"added": added, "total_issues": len(issues)},
	})

	fmt.Printf("Loaded %d new items from beads (%d already in queue)\n", added, len(issues)-added)
	return nil
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
