package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/channel"
	"github.com/rpuneet/bc/pkg/github"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "GitHub issue commands",
	Long:  `Create, view, comment on, and react to GitHub issues. Uses gh auth.`,
}

var issueCreateCmd = &cobra.Command{
	Use:   "create --title <title> [--body <body>]",
	Short: "Create a GitHub issue",
	Long:  `Create an issue in the current repo. Uses gh auth from workspace.`,
	RunE:  runIssueCreate,
}

var issueViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View an issue",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueView,
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment <number> [--body <text>]",
	Short: "Add a comment to an issue",
	Long:  `Add a comment. Use --agent to prefix body with **[agent]** for consistent formatting.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueComment,
}

var issueReactCmd = &cobra.Command{
	Use:   "react <number> <reaction>",
	Short: "Add a reaction to an issue",
	Long:  `Add a reaction (e.g. +1, heart, rocket, thumbsup). See GitHub API for allowed values.`,
	Args:  cobra.ExactArgs(2),
	RunE:  runIssueReact,
}

var (
	issueTitle        string
	issueBody         string
	issueComment      string
	issueCommentAgent string
)

func init() {
	issueCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVar(&issueBody, "body", "", "Issue body")
	_ = issueCreateCmd.MarkFlagRequired("title")

	issueCommentCmd.Flags().StringVar(&issueComment, "body", "", "Comment text")
	issueCommentCmd.Flags().StringVar(&issueCommentAgent, "agent", "", "Agent ID to prefix comment with **[agent]**")

	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCommentCmd)
	issueCmd.AddCommand(issueReactCmd)
	rootCmd.AddCommand(issueCmd)
}

func runIssueCreate(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}
	if err := github.CreateIssue(ws.RootDir, issueTitle, issueBody); err != nil {
		return err
	}
	fmt.Println("Issue created.")
	return nil
}

func runIssueView(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}
	var num int
	if _, parseErr := fmt.Sscanf(args[0], "%d", &num); parseErr != nil || num <= 0 {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}
	detail, err := github.ViewIssue(ws.RootDir, num)
	if err != nil {
		return err
	}
	fmt.Printf("#%d %s [%s]\n", detail.Number, detail.Title, detail.State)
	if detail.Author != "" {
		fmt.Printf("Author: %s\n", detail.Author)
	}
	if detail.URL != "" {
		fmt.Printf("URL: %s\n", detail.URL)
	}
	if detail.Body != "" {
		fmt.Println()
		fmt.Println(detail.Body)
	}
	return nil
}

func runIssueComment(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}
	var num int
	if _, parseErr := fmt.Sscanf(args[0], "%d", &num); parseErr != nil || num <= 0 {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}
	body := strings.TrimSpace(issueComment)
	if body == "" {
		return fmt.Errorf("comment body is required (use --body)")
	}
	if issueCommentAgent != "" {
		body = channel.FormatAgentComment(issueCommentAgent, body)
	}
	if err := github.IssueComment(ws.RootDir, num, body); err != nil {
		return err
	}
	fmt.Println("Comment added.")
	return nil
}

func runIssueReact(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}
	var num int
	if _, parseErr := fmt.Sscanf(args[0], "%d", &num); parseErr != nil || num <= 0 {
		return fmt.Errorf("invalid issue number: %s", args[0])
	}
	content := strings.TrimSpace(strings.ToLower(args[1]))
	// Normalize common names to GitHub API values
	switch content {
	case "thumbsup", "thumb_up", "👍":
		content = "+1"
	case "thumbsdown", "thumb_down", "👎":
		content = "-1"
	case "heart", "❤️":
		content = "heart"
	case "rocket", "🚀":
		content = "rocket"
	case "eyes", "👀":
		content = "eyes"
	case "hooray":
		content = "hooray"
	case "laugh", "laughing":
		content = "laugh"
	case "confused":
		content = "confused"
	}
	if err := github.AddReaction(ws.RootDir, num, content); err != nil {
		return err
	}
	fmt.Printf("Reaction %q added.\n", content)
	return nil
}
