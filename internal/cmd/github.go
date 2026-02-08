package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/github"
)

var githubCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub integration commands",
	Long:  `List and manage GitHub pull requests and issues with filters. Uses gh CLI auth (see bc github auth when available).`,
}

var githubPrListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long: `List GitHub pull requests with optional filters.

Examples:
  bc github pr list
  bc github pr list --state closed
  bc github pr list --author @me
  bc github pr list --repo owner/repo --label bug
  bc github pr list --json   # JSON output for TUI/scripting`,
	RunE: runGithubPRList,
}

var githubIssueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long: `List GitHub issues with optional filters.

Examples:
  bc github issue list
  bc github issue list --state all
  bc github issue list --assignee @me
  bc github issue list --repo owner/repo --label task
  bc github issue list --json   # JSON output for TUI/scripting`,
	RunE: runGithubIssueList,
}

func init() {
	githubPrListCmd.Flags().String("state", "", "Filter by state: open, closed, merged, all (default: open)")
	githubPrListCmd.Flags().String("repo", "", "Repository in owner/repo format (default: workspace repo)")
	githubPrListCmd.Flags().String("author", "", "Filter by author")
	githubPrListCmd.Flags().String("assignee", "", "Filter by assignee")
	githubPrListCmd.Flags().StringSlice("label", nil, "Filter by label (can be repeated)")
	githubPrListCmd.Flags().Int("limit", 50, "Maximum number of items to fetch")

	githubIssueListCmd.Flags().String("state", "", "Filter by state: open, closed, all (default: open)")
	githubIssueListCmd.Flags().String("repo", "", "Repository in owner/repo format (default: workspace repo)")
	githubIssueListCmd.Flags().String("author", "", "Filter by author")
	githubIssueListCmd.Flags().String("assignee", "", "Filter by assignee")
	githubIssueListCmd.Flags().StringSlice("label", nil, "Filter by label (can be repeated)")
	githubIssueListCmd.Flags().Int("limit", 50, "Maximum number of items to fetch")

	prCmd := &cobra.Command{Use: "pr", Short: "Pull request commands"}
	prCmd.AddCommand(githubPrListCmd)

	issueCmd := &cobra.Command{Use: "issue", Short: "Issue commands"}
	issueCmd.AddCommand(githubIssueListCmd)

	githubCmd.AddCommand(prCmd)
	githubCmd.AddCommand(issueCmd)
	rootCmd.AddCommand(githubCmd)
}

func runGithubPRList(cmd *cobra.Command, args []string) error {
	state, _ := cmd.Flags().GetString("state")
	repo, _ := cmd.Flags().GetString("repo")
	author, _ := cmd.Flags().GetString("author")
	assignee, _ := cmd.Flags().GetString("assignee")
	labels, _ := cmd.Flags().GetStringSlice("label")
	limit, _ := cmd.Flags().GetInt("limit")

	opts := github.ListPROpts{
		State:    state,
		Repo:     repo,
		Author:   author,
		Assignee: assignee,
		Labels:   labels,
		Limit:    limit,
	}
	if repo == "" {
		ws, err := getWorkspace()
		if err != nil {
			return fmt.Errorf("not in a bc workspace and --repo not set: %w", err)
		}
		opts.Workspace = ws.RootDir
	}

	prs, err := github.ListPRsWithOpts(cmd.Context(), opts)
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	out := cmd.OutOrStdout()
	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(prs)
	}

	// Human-readable table for TUI consumption (consistent columns)
	for _, pr := range prs {
		draft := ""
		if pr.IsDraft {
			draft = " (draft)"
		}
		_, _ = fmt.Fprintf(out, "#%d\t%s\t%s\t%s%s\n", pr.Number, pr.State, pr.ReviewDecision, pr.Title, draft)
	}
	if len(prs) == 0 {
		_, _ = fmt.Fprintln(out, "No pull requests found.")
	}
	return nil
}

func runGithubIssueList(cmd *cobra.Command, args []string) error {
	state, _ := cmd.Flags().GetString("state")
	repo, _ := cmd.Flags().GetString("repo")
	author, _ := cmd.Flags().GetString("author")
	assignee, _ := cmd.Flags().GetString("assignee")
	labels, _ := cmd.Flags().GetStringSlice("label")
	limit, _ := cmd.Flags().GetInt("limit")

	opts := github.ListIssuesOpts{
		State:    state,
		Repo:     repo,
		Author:   author,
		Assignee: assignee,
		Labels:   labels,
		Limit:    limit,
	}
	if repo == "" {
		ws, err := getWorkspace()
		if err != nil {
			return fmt.Errorf("not in a bc workspace and --repo not set: %w", err)
		}
		opts.Workspace = ws.RootDir
	}

	issues, err := github.ListIssuesWithOpts(cmd.Context(), opts)
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")
	out := cmd.OutOrStdout()
	if jsonOutput {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(issues)
	}

	// Human-readable table for TUI consumption
	for _, issue := range issues {
		labelStr := ""
		if len(issue.Labels) > 0 {
			labelStr = fmt.Sprintf("\t[%s]", joinLabels(issue.Labels))
		}
		_, _ = fmt.Fprintf(out, "#%d\t%s\t%s%s\n", issue.Number, issue.State, issue.Title, labelStr)
	}
	if len(issues) == 0 {
		_, _ = fmt.Fprintln(out, "No issues found.")
	}
	return nil
}

func joinLabels(labels []string) string {
	if len(labels) == 0 {
		return ""
	}
	s := labels[0]
	for i := 1; i < len(labels); i++ {
		s += "," + labels[i]
	}
	return s
}
