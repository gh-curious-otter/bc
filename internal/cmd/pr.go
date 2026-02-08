package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/github"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "PR workflow commands",
	Long:  `Commands for PR workflow automation with channel-based notifications.`,
}

var prNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Post PR review requests to #reviews channel",
	Long: `Scan open PRs and post review requests to the #reviews channel.

This command:
1. Lists all open PRs in the repository
2. Filters for PRs that need review (not draft, pending review)
3. Posts review requests to #reviews channel
4. Notifies tech-leads via @mentions

Example:
  bc pr notify              # Notify about all PRs needing review
  bc pr notify --pr 123     # Notify about specific PR`,
	RunE: runPRNotify,
}

var prNumber int

func init() {
	prNotifyCmd.Flags().IntVar(&prNumber, "pr", 0, "Specific PR number to notify about")
	prCmd.AddCommand(prNotifyCmd)
	rootCmd.AddCommand(prCmd)
}

func runPRNotify(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Get open PRs from GitHub
	prs, err := github.ListPRs(ws.RootDir)
	if err != nil {
		return fmt.Errorf("failed to list PRs: %w", err)
	}

	if len(prs) == 0 {
		fmt.Println("No open PRs found.")
		return nil
	}

	// Filter PRs needing review
	var needsReview []github.PR
	for _, pr := range prs {
		// Skip if specific PR requested and this isn't it
		if prNumber > 0 && pr.Number != prNumber {
			continue
		}

		// Skip drafts
		if pr.IsDraft {
			continue
		}

		// Include PRs with pending or no review decision
		if pr.ReviewDecision == "" || pr.ReviewDecision == "REVIEW_REQUIRED" {
			needsReview = append(needsReview, pr)
		}
	}

	if len(needsReview) == 0 {
		if prNumber > 0 {
			fmt.Printf("PR #%d does not need review or was not found.\n", prNumber)
		} else {
			fmt.Println("No PRs currently need review.")
		}
		return nil
	}

	// Open channel store
	store := channel.NewSQLiteStore(ws.RootDir)
	if openErr := store.Open(); openErr != nil {
		return fmt.Errorf("failed to open channel store: %w", openErr)
	}
	defer func() { _ = store.Close() }()

	// Ensure #reviews channel exists
	reviewsChannel, err := store.GetChannel("reviews")
	if err != nil {
		return fmt.Errorf("failed to check reviews channel: %w", err)
	}
	if reviewsChannel == nil {
		_, err = store.CreateChannel("reviews", channel.ChannelTypeGroup, "PR review requests")
		if err != nil {
			return fmt.Errorf("failed to create reviews channel: %w", err)
		}
		fmt.Println("Created #reviews channel")
	}

	// Get tech-leads to notify
	techLeads := findTechLeads(store)
	if len(techLeads) == 0 {
		fmt.Println("Warning: No tech-leads found in channels. Review notifications may not be delivered.")
	}

	// Event log
	log := events.NewLog(filepath.Join(ws.StateDir(), "events.jsonl"))

	// Post review requests
	for _, pr := range needsReview {
		message := formatReviewRequest(pr, techLeads)

		// Add to channel history with review message type
		_, msgErr := store.AddMessage("reviews", "system", message, channel.TypeReview, fmt.Sprintf(`{"pr_number":%d}`, pr.Number))
		if msgErr != nil {
			fmt.Printf("Warning: failed to log message for PR #%d: %v\n", pr.Number, msgErr)
		}

		// Log event
		_ = log.Append(events.Event{
			Type:    events.MessageSent,
			Agent:   "system",
			Message: fmt.Sprintf("PR review request: #%d", pr.Number),
			Data: map[string]any{
				"pr_number": pr.Number,
				"pr_title":  pr.Title,
				"channel":   "reviews",
			},
		})

		fmt.Printf("Posted review request for PR #%d: %s\n", pr.Number, pr.Title)
	}

	fmt.Printf("\nNotified about %d PR(s) in #reviews channel.\n", len(needsReview))
	if len(techLeads) > 0 {
		fmt.Printf("Tech-leads to review: %s\n", strings.Join(techLeads, ", "))
	}

	return nil
}

// formatReviewRequest creates a formatted review request message.
func formatReviewRequest(pr github.PR, techLeads []string) string {
	var b strings.Builder

	// Add @mentions for tech-leads
	if len(techLeads) > 0 {
		for _, tl := range techLeads {
			b.WriteString("@")
			b.WriteString(tl)
			b.WriteString(" ")
		}
	}

	b.WriteString(fmt.Sprintf("PR #%d ready for review: %s", pr.Number, pr.Title))

	return b.String()
}

// findTechLeads looks for tech-lead agents in the channel members.
func findTechLeads(store *channel.SQLiteStore) []string {
	var techLeads []string

	// Check engineering channel for tech-leads
	members, err := store.GetMembers("engineering")
	if err != nil {
		return techLeads
	}

	for _, member := range members {
		if strings.HasPrefix(member, "tech-lead") {
			techLeads = append(techLeads, member)
		}
	}

	return techLeads
}
