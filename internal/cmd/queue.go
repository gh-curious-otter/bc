package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/queue"
	"github.com/rpuneet/bc/pkg/ui"
)

// Queue commands for dual queue system
// Issue #1234: Dual Queue System for hierarchical task management

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Manage work and merge queues",
	Long: `Dual queue system for hierarchical task management.

Work Queue (tasks TO you):
  bc queue work                    List your work queue
  bc queue work accept <id>        Accept a work item
  bc queue work complete <id>      Complete work (triggers merge submit)

Merge Queue (work FROM others awaiting your review):
  bc queue merge                   List pending merges
  bc queue merge approve <id>      Approve and merge
  bc queue merge reject <id>       Reject with reason

Submit Flow:
  bc queue submit <id> --to <agent>   Submit completed work for review

The dual queue enables hierarchical task flow:
  ROOT -> MANAGER (work queue) -> ENGINEER (work queue)
  ENGINEER (complete) -> MANAGER (merge queue) -> ROOT (merge queue)`,
}

// Work queue commands
var queueWorkCmd = &cobra.Command{
	Use:   "work",
	Short: "Manage your work queue",
	Long: `View and manage tasks assigned to you.

Examples:
  bc queue work                 List all work items
  bc queue work --status pending   Filter by status
  bc queue work accept 1        Accept work item #1
  bc queue work complete 1      Mark work item #1 as complete`,
	RunE: runQueueWorkList,
}

var queueWorkAcceptCmd = &cobra.Command{
	Use:   "accept <id>",
	Short: "Accept a work item",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueWorkAccept,
}

var queueWorkStartCmd = &cobra.Command{
	Use:   "start <id>",
	Short: "Start working on an item",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueWorkStart,
}

var queueWorkCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Complete a work item",
	Long: `Mark a work item as complete with the branch containing the work.

Examples:
  bc queue work complete 1 --branch eng-01/issue-123/feature
  bc queue work complete 1    # Uses current git branch`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueWorkComplete,
}

// Merge queue commands
var queueMergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Manage your merge queue",
	Long: `View and manage branches submitted for your review.

Examples:
  bc queue merge                    List pending merges
  bc queue merge approve 1          Approve merge #1
  bc queue merge reject 1 --reason "Tests failing"`,
	RunE: runQueueMergeList,
}

var queueMergeApproveCmd = &cobra.Command{
	Use:   "approve <id>",
	Short: "Approve a merge request",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueMergeApprove,
}

var queueMergeRejectCmd = &cobra.Command{
	Use:   "reject <id>",
	Short: "Reject a merge request",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueMergeReject,
}

var queueMergeCompleteCmd = &cobra.Command{
	Use:   "complete <id>",
	Short: "Mark merge as completed",
	Args:  cobra.ExactArgs(1),
	RunE:  runQueueMergeComplete,
}

// Submit command
var queueSubmitCmd = &cobra.Command{
	Use:   "submit <work-id>",
	Short: "Submit completed work for review",
	Long: `Submit a completed work item to another agent's merge queue.

This creates a merge request in the target agent's queue,
notifying them that your work is ready for review.

Examples:
  bc queue submit 1 --to mgr-01    Submit work #1 to mgr-01's merge queue`,
	Args: cobra.ExactArgs(1),
	RunE: runQueueSubmit,
}

// Flags
var (
	queueStatusFilter string
	queueBranch       string
	queueToAgent      string
	queueRejectReason string
)

func init() {
	// Work queue subcommands
	queueWorkCmd.AddCommand(queueWorkAcceptCmd)
	queueWorkCmd.AddCommand(queueWorkStartCmd)
	queueWorkCmd.AddCommand(queueWorkCompleteCmd)

	// Merge queue subcommands
	queueMergeCmd.AddCommand(queueMergeApproveCmd)
	queueMergeCmd.AddCommand(queueMergeRejectCmd)
	queueMergeCmd.AddCommand(queueMergeCompleteCmd)

	// Main queue subcommands
	queueCmd.AddCommand(queueWorkCmd)
	queueCmd.AddCommand(queueMergeCmd)
	queueCmd.AddCommand(queueSubmitCmd)

	// Flags
	queueWorkCmd.Flags().StringVar(&queueStatusFilter, "status", "", "Filter by status (pending, accepted, in_progress, completed)")
	queueMergeCmd.Flags().StringVar(&queueStatusFilter, "status", "", "Filter by status (pending, approved, rejected, merged)")
	queueWorkCompleteCmd.Flags().StringVar(&queueBranch, "branch", "", "Branch containing the work (defaults to current branch)")
	queueSubmitCmd.Flags().StringVar(&queueToAgent, "to", "", "Target agent for merge review (required)")
	queueMergeRejectCmd.Flags().StringVar(&queueRejectReason, "reason", "", "Reason for rejection (required)")

	_ = queueSubmitCmd.MarkFlagRequired("to")
	_ = queueMergeRejectCmd.MarkFlagRequired("reason")

	rootCmd.AddCommand(queueCmd)
}

func getQueueStore() (*queue.Store, string, error) {
	ws, err := getWorkspace()
	if err != nil {
		return nil, "", errNotInWorkspace(err)
	}

	// Get current agent ID
	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return nil, "", fmt.Errorf("BC_AGENT_ID not set - this command requires running as an agent")
	}

	store := queue.NewStore(ws.StateDir())
	if err := store.Open(context.Background()); err != nil {
		return nil, "", fmt.Errorf("failed to open queue store: %w", err)
	}

	return store, agentID, nil
}

func runQueueWorkList(cmd *cobra.Command, _ []string) error {
	store, agentID, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	ctx := context.Background()
	items, err := store.ListWork(ctx, agentID, queueStatusFilter)
	if err != nil {
		return err
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
		fmt.Println()
		fmt.Println("  No work items in queue.")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Work Queue"))
	fmt.Println("  " + strings.Repeat("─", 60))
	fmt.Println()

	for _, item := range items {
		statusIcon := "○"
		statusColor := ui.DimText
		switch item.Status {
		case queue.StatusPending:
			statusIcon = "○"
			statusColor = ui.YellowText
		case queue.StatusAccepted:
			statusIcon = "◐"
			statusColor = ui.CyanText
		case queue.StatusInProgress:
			statusIcon = "●"
			statusColor = ui.BlueText
		case queue.StatusCompleted:
			statusIcon = "✓"
			statusColor = ui.GreenText
		case queue.StatusFailed:
			statusIcon = "✗"
			statusColor = ui.RedText
		}

		fmt.Printf("  %s #%d %s\n", statusColor(statusIcon), item.ID, ui.BoldText(item.Title))
		fmt.Printf("      Status: %s  Priority: %s\n", item.Status, priorityLabel(item.Priority))
		if item.FromAgent != "" {
			fmt.Printf("      From: %s\n", item.FromAgent)
		}
		if item.IssueRef != "" {
			fmt.Printf("      Issue: %s\n", item.IssueRef)
		}
		fmt.Println()
	}

	return nil
}

func runQueueWorkAccept(_ *cobra.Command, args []string) error {
	store, _, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid work item ID: %s", args[0])
	}

	ctx := context.Background()
	if err := store.AcceptWork(ctx, id); err != nil {
		return err
	}

	fmt.Printf("✓ Accepted work item #%d\n", id)
	fmt.Println("  Start with: bc queue work start", id)
	return nil
}

func runQueueWorkStart(_ *cobra.Command, args []string) error {
	store, _, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid work item ID: %s", args[0])
	}

	ctx := context.Background()
	if err := store.StartWork(ctx, id); err != nil {
		return err
	}

	fmt.Printf("✓ Started work on item #%d\n", id)
	fmt.Println("  Complete with: bc queue work complete", id)
	return nil
}

func runQueueWorkComplete(_ *cobra.Command, args []string) error {
	store, _, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid work item ID: %s", args[0])
	}

	branch := queueBranch
	if branch == "" {
		// TODO: Get current git branch
		return fmt.Errorf("--branch is required (current branch detection not yet implemented)")
	}

	ctx := context.Background()
	if err := store.CompleteWork(ctx, id, branch); err != nil {
		return err
	}

	fmt.Printf("✓ Completed work item #%d\n", id)
	fmt.Printf("  Branch: %s\n", branch)
	fmt.Println("  Submit for review with: bc queue submit", id, "--to <agent>")
	return nil
}

func runQueueMergeList(cmd *cobra.Command, _ []string) error {
	store, agentID, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	ctx := context.Background()
	items, err := store.ListMerge(ctx, agentID, queueStatusFilter)
	if err != nil {
		return err
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
		fmt.Println()
		fmt.Println("  No items in merge queue.")
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Merge Queue"))
	fmt.Println("  " + strings.Repeat("─", 60))
	fmt.Println()

	for _, item := range items {
		statusIcon := "○"
		statusColor := ui.DimText
		switch item.Status {
		case queue.MergeStatusPending:
			statusIcon = "○"
			statusColor = ui.YellowText
		case queue.MergeStatusReviewed:
			statusIcon = "◐"
			statusColor = ui.CyanText
		case queue.MergeStatusApproved:
			statusIcon = "✓"
			statusColor = ui.GreenText
		case queue.MergeStatusMerged:
			statusIcon = "●"
			statusColor = ui.GreenText
		case queue.MergeStatusRejected:
			statusIcon = "✗"
			statusColor = ui.RedText
		case queue.MergeStatusConflict:
			statusIcon = "!"
			statusColor = ui.YellowText
		}

		fmt.Printf("  %s #%d %s\n", statusColor(statusIcon), item.ID, ui.BoldText(item.Title))
		fmt.Printf("      Branch: %s\n", item.Branch)
		fmt.Printf("      From: %s  Status: %s\n", item.FromAgent, item.Status)
		if item.IssueRef != "" {
			fmt.Printf("      Issue: %s\n", item.IssueRef)
		}
		if item.Reason != "" {
			fmt.Printf("      Reason: %s\n", ui.RedText(item.Reason))
		}
		fmt.Println()
	}

	return nil
}

func runQueueMergeApprove(_ *cobra.Command, args []string) error {
	store, agentID, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid merge item ID: %s", args[0])
	}

	ctx := context.Background()
	if err := store.ApproveMerge(ctx, id, agentID); err != nil {
		return err
	}

	fmt.Printf("✓ Approved merge #%d\n", id)
	fmt.Println("  Complete merge with: bc queue merge complete", id)
	return nil
}

func runQueueMergeReject(_ *cobra.Command, args []string) error {
	store, agentID, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid merge item ID: %s", args[0])
	}

	ctx := context.Background()
	if err := store.RejectMerge(ctx, id, agentID, queueRejectReason); err != nil {
		return err
	}

	fmt.Printf("✗ Rejected merge #%d\n", id)
	fmt.Printf("  Reason: %s\n", queueRejectReason)
	return nil
}

func runQueueMergeComplete(_ *cobra.Command, args []string) error {
	store, _, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid merge item ID: %s", args[0])
	}

	ctx := context.Background()
	if err := store.CompleteMerge(ctx, id); err != nil {
		return err
	}

	fmt.Printf("✓ Merge #%d completed\n", id)
	return nil
}

func runQueueSubmit(_ *cobra.Command, args []string) error {
	store, _, err := getQueueStore()
	if err != nil {
		return err
	}
	defer store.Close() //nolint:errcheck // deferred close

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid work item ID: %s", args[0])
	}

	ctx := context.Background()
	merge, err := store.Submit(ctx, id, queueToAgent)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Submitted to %s's merge queue\n", queueToAgent)
	fmt.Printf("  Merge ID: #%d\n", merge.ID)
	fmt.Printf("  Branch: %s\n", merge.Branch)
	return nil
}

func priorityLabel(p int) string {
	switch p {
	case queue.PriorityUrgent:
		return ui.RedText("URGENT")
	case queue.PriorityHigh:
		return ui.YellowText("high")
	case queue.PriorityNormal:
		return "normal"
	case queue.PriorityLow:
		return ui.DimText("low")
	default:
		return "unknown"
	}
}
