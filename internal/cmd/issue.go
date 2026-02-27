package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage GitHub issues",
	Long: `Manage GitHub issues from test results.

The issue command provides automated issue creation from test failures.

Examples:
  bc issue create --type bug --title "Test failure"    # Create bug issue
  bc issue create --type enhancement --title "Improve" # Create enhancement
  bc issue list --labels bug,test-failure              # List issues`,
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a GitHub issue",
	Long: `Create a GitHub issue with specified type and details.

Examples:
  bc issue create --type bug --title "Test failure: agent timeout"
  bc issue create --type enhancement --title "Add retry logic"`,
	RunE: runIssueCreate,
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List GitHub issues",
	Long: `List GitHub issues with optional filters.

Examples:
  bc issue list                        # List all open issues
  bc issue list --state closed         # List closed issues
  bc issue list --state all            # List all issues
  bc issue list --labels test-failure  # List by label
  bc issue list --assignee @me         # List assigned to me
  bc issue list --type bug             # List by type`,
	RunE: runIssueList,
}

var issueReopenCmd = &cobra.Command{
	Use:   "reopen <id>",
	Short: "Reopen a closed issue",
	Long: `Reopen a previously closed GitHub issue.

Examples:
  bc issue reopen 123
  bc issue reopen 123 --comment "Reopening for further work"`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueReopen,
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment <id> <message>",
	Short: "Add a comment to an issue",
	Long: `Add a comment to a GitHub issue.

Examples:
  bc issue comment 123 "Working on this"
  bc issue comment 123 "Fixed in PR #456"`,
	Args: cobra.MinimumNArgs(2),
	RunE: runIssueComment,
}

var issueSearchCmd = &cobra.Command{
	Use:   "search <keywords>",
	Short: "Search issues before creating new ones",
	Long: `Search existing issues to avoid creating duplicates.

IMPORTANT: Always search before creating a new issue to reduce duplicate rate.

The search looks through issue titles, bodies, and comments.

Examples:
  bc issue search "80x24 layout"           # Find layout-related issues
  bc issue search "memory view empty"      # Find memory view issues
  bc issue search "focus navigation"       # Find focus/nav issues
  bc issue search "ESC key" --state all    # Include closed issues`,
	Args: cobra.MinimumNArgs(1),
	RunE: runIssueSearch,
}

var issueViewCmd = &cobra.Command{
	Use:   "view <id>",
	Short: "View issue details",
	Long: `View detailed information about a GitHub issue.

Shows title, body, labels, assignees, comments, and history.

Examples:
  bc issue view 123                    # View issue #123
  bc issue view 123 --comments         # Include comments`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueView,
}

var issueEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Edit an issue",
	Long: `Edit a GitHub issue's title, body, or labels.

Examples:
  bc issue edit 123 --title "New title"
  bc issue edit 123 --add-label bug
  bc issue edit 123 --remove-label enhancement`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueEdit,
}

var issueCloseCmd = &cobra.Command{
	Use:   "close <id>",
	Short: "Close an issue",
	Long: `Close a GitHub issue with optional reason.

Examples:
  bc issue close 123
  bc issue close 123 --reason completed
  bc issue close 123 --reason "not planned" --comment "Duplicate of #456"`,
	Args: cobra.ExactArgs(1),
	RunE: runIssueClose,
}

var issueAssignCmd = &cobra.Command{
	Use:   "assign <id> <assignee>",
	Short: "Assign issue to user or agent",
	Long: `Assign a GitHub issue to a user or agent.

Examples:
  bc issue assign 123 @username        # Assign to GitHub user
  bc issue assign 123 eng-01           # Assign to agent (adds label)
  bc issue assign 123 --unassign       # Remove all assignees`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runIssueAssign,
}

var (
	issueType         string
	issueTitle        string
	issueDescription  string
	issueLabels       string
	issueSeverity     string
	issueReproSteps   string
	issueAssignee     string
	issueAddLabel     string
	issueRemoveLabel  string
	issueCloseReason  string
	issueComment      string
	issueShowComments bool
	issueUnassign     bool
	issueState        string
	issueSearchLimit  int
)

// validIssueTypes defines the allowed issue types
var validIssueTypes = []string{"bug", "enhancement", "test-failure", "feature", "documentation", "epic", "task", "chore"}

// validSeverities defines the allowed severity levels
var validSeverities = []string{"critical", "high", "medium", "low"}

// isValidIssueType checks if the given type is valid
func isValidIssueType(t string) bool {
	for _, valid := range validIssueTypes {
		if t == valid {
			return true
		}
	}
	return false
}

// isValidSeverity checks if the given severity is valid
func isValidSeverity(s string) bool {
	for _, valid := range validSeverities {
		if s == valid {
			return true
		}
	}
	return false
}

func init() {
	// issue create flags
	issueCreateCmd.Flags().StringVar(&issueType, "type", "bug", "Issue type: bug, enhancement, test-failure, feature, documentation, epic, task, chore")
	issueCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVar(&issueDescription, "description", "", "Issue description")
	issueCreateCmd.Flags().StringVar(&issueLabels, "labels", "", "Comma-separated labels")
	issueCreateCmd.Flags().StringVar(&issueSeverity, "severity", "medium", "Severity: critical, high, medium, low")
	issueCreateCmd.Flags().StringVar(&issueReproSteps, "reproduction", "", "Reproduction steps")
	issueCreateCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Assign to user")

	// issue list flags
	issueListCmd.Flags().StringVar(&issueLabels, "labels", "", "Filter by labels")
	issueListCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Filter by assignee (@me for self)")
	issueListCmd.Flags().StringVar(&issueType, "type", "", "Filter by type (epic, bug, task, etc.)")
	issueListCmd.Flags().StringVar(&issueState, "state", "open", "Issue state: open, closed, all")

	// issue reopen flags
	issueReopenCmd.Flags().StringVar(&issueComment, "comment", "", "Add reopening comment")

	// issue comment flags (no flags needed, message is positional)

	// issue view flags
	issueViewCmd.Flags().BoolVar(&issueShowComments, "comments", false, "Include comments")

	// issue edit flags
	issueEditCmd.Flags().StringVar(&issueTitle, "title", "", "New title")
	issueEditCmd.Flags().StringVar(&issueDescription, "body", "", "New body")
	issueEditCmd.Flags().StringVar(&issueAddLabel, "add-label", "", "Add label")
	issueEditCmd.Flags().StringVar(&issueRemoveLabel, "remove-label", "", "Remove label")
	issueEditCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Set assignee")

	// issue close flags
	issueCloseCmd.Flags().StringVar(&issueCloseReason, "reason", "completed", "Close reason (completed, not_planned, duplicate)")
	issueCloseCmd.Flags().StringVar(&issueComment, "comment", "", "Add closing comment")

	// issue assign flags
	issueAssignCmd.Flags().BoolVar(&issueUnassign, "unassign", false, "Remove all assignees")

	// issue search flags
	issueSearchCmd.Flags().StringVar(&issueState, "state", "open", "Issue state: open, closed, all")
	issueSearchCmd.Flags().IntVar(&issueSearchLimit, "limit", 10, "Maximum results to show")

	// Add subcommands
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueSearchCmd) // Search before create!
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueEditCmd)
	issueCmd.AddCommand(issueCloseCmd)
	issueCmd.AddCommand(issueAssignCmd)
	issueCmd.AddCommand(issueReopenCmd)
	issueCmd.AddCommand(issueCommentCmd)

	rootCmd.AddCommand(issueCmd)
}

// IssueData represents GitHub issue creation data
type IssueData struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Type        string   `json:"type"`
	Severity    string   `json:"severity"`
	TestFailure string   `json:"test_failure,omitempty"`
	Labels      []string `json:"labels,omitempty"`
}

// CreateIssueFromTestFailure creates a GitHub issue from a test failure
func CreateIssueFromTestFailure(testName, output, reproduction string) (*IssueData, error) {
	title := fmt.Sprintf("Test failure: %s", testName)

	body := fmt.Sprintf(`## Test Failure Report

**Test:** %s

### Reproduction
%s

### Test Output
%s

### Environment
- Generated by: bc test (automated)
- Severity: high

---
🤖 Auto-generated by bc test demons
`, testName, reproduction, "```\n"+output+"\n```")

	labels := []string{"bug", "test-failure", "automated"}

	issue := &IssueData{
		Title:       title,
		Body:        body,
		Labels:      labels,
		Type:        "bug",
		Severity:    "high",
		TestFailure: testName,
	}

	return issue, nil
}

func runIssueCreate(cmd *cobra.Command, args []string) error {
	log.Debug("issue create command started")

	if issueTitle == "" {
		return fmt.Errorf("--title is required")
	}

	// Validate issue type
	if !isValidIssueType(issueType) {
		return fmt.Errorf("invalid --type %q: must be one of %v", issueType, validIssueTypes)
	}

	// Validate severity
	if !isValidSeverity(issueSeverity) {
		return fmt.Errorf("invalid --severity %q: must be one of %v", issueSeverity, validSeverities)
	}

	// Build issue body
	var bodyParts []string
	// Capitalize first letter of issue type
	typeCapitalized := issueType
	if len(issueType) > 0 {
		typeCapitalized = strings.ToUpper(issueType[:1]) + issueType[1:]
	}
	bodyParts = append(bodyParts, fmt.Sprintf("## %s Issue", typeCapitalized))

	if issueDescription != "" {
		bodyParts = append(bodyParts, fmt.Sprintf("\n### Description\n%s", issueDescription))
	}

	if issueReproSteps != "" {
		bodyParts = append(bodyParts, fmt.Sprintf("\n### Reproduction Steps\n%s", issueReproSteps))
	}

	bodyParts = append(bodyParts, fmt.Sprintf("\n### Severity\n%s", issueSeverity))
	bodyParts = append(bodyParts, "\n---\n🤖 Created by bc issue command")

	body := strings.Join(bodyParts, "\n")

	// Build gh command with preallocated capacity
	ghArgs := make([]string, 0, 8)
	ghArgs = append(ghArgs, "issue", "create", "--title", issueTitle, "--body", body)

	// Add labels
	labels := []string{issueType}
	if issueLabels != "" {
		labels = append(labels, strings.Split(issueLabels, ",")...)
	}
	ghArgs = append(ghArgs, "--label", strings.Join(labels, ","))

	// Execute gh command with context
	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	return nil
}

func runIssueList(cmd *cobra.Command, args []string) error {
	log.Debug("issue list command started")

	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Build gh command
	ghArgs := make([]string, 0, 12)
	ghArgs = append(ghArgs, "issue", "list")

	// Add state filter (open, closed, all)
	if issueState != "" && issueState != "open" {
		ghArgs = append(ghArgs, "--state", issueState)
	}

	if issueLabels != "" {
		ghArgs = append(ghArgs, "--label", issueLabels)
	}

	if issueAssignee != "" {
		ghArgs = append(ghArgs, "--assignee", issueAssignee)
	}

	// Filter by type (issue types are stored as labels)
	if issueType != "" {
		ghArgs = append(ghArgs, "--label", issueType)
	}

	if jsonOutput {
		ghArgs = append(ghArgs, "--json", "number,title,body,labels,state,assignees,author,createdAt,updatedAt")
	}

	// Execute gh command with context
	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command

	var out bytes.Buffer
	ghCmd.Stdout = &out
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	if jsonOutput {
		// Pretty print JSON
		var data any
		if err := json.Unmarshal(out.Bytes(), &data); err != nil {
			fmt.Print(out.String())
			return nil
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Print(out.String())
	return nil
}

func runIssueView(cmd *cobra.Command, args []string) error {
	log.Debug("issue view command started")

	issueID := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Build gh command for issue details
	ghArgs := []string{"issue", "view", issueID}

	if jsonOutput {
		ghArgs = append(ghArgs, "--json", "number,title,body,state,labels,assignees,author,createdAt,updatedAt,comments")
	}

	if issueShowComments {
		ghArgs = append(ghArgs, "--comments")
	}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command

	var out bytes.Buffer
	ghCmd.Stdout = &out
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to view issue: %w", err)
	}

	if jsonOutput {
		var data any
		if err := json.Unmarshal(out.Bytes(), &data); err != nil {
			fmt.Print(out.String())
			return nil
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Print(out.String())
	return nil
}

func runIssueEdit(cmd *cobra.Command, args []string) error {
	log.Debug("issue edit command started")

	issueID := args[0]

	// Build gh command
	ghArgs := []string{"issue", "edit", issueID}

	if issueTitle != "" {
		ghArgs = append(ghArgs, "--title", issueTitle)
	}

	if issueDescription != "" {
		ghArgs = append(ghArgs, "--body", issueDescription)
	}

	if issueAddLabel != "" {
		ghArgs = append(ghArgs, "--add-label", issueAddLabel)
	}

	if issueRemoveLabel != "" {
		ghArgs = append(ghArgs, "--remove-label", issueRemoveLabel)
	}

	if issueAssignee != "" {
		ghArgs = append(ghArgs, "--add-assignee", issueAssignee)
	}

	// Check if any edit flags were provided
	if len(ghArgs) == 3 {
		return fmt.Errorf("no edit options specified (use --title, --body, --add-label, --remove-label, or --assignee)")
	}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to edit issue: %w", err)
	}

	fmt.Printf("Issue #%s updated\n", issueID)
	return nil
}

func runIssueClose(cmd *cobra.Command, args []string) error {
	log.Debug("issue close command started")

	issueID := args[0]

	// Build gh command
	ghArgs := []string{"issue", "close", issueID}

	// Map reason to gh close reason
	switch issueCloseReason {
	case "completed", "":
		ghArgs = append(ghArgs, "--reason", "completed")
	case "not_planned", "not planned", "wontfix":
		ghArgs = append(ghArgs, "--reason", "not planned")
	case "duplicate":
		// For duplicate, we add a comment instead since gh doesn't have duplicate reason
		ghArgs = append(ghArgs, "--reason", "not planned")
		if issueComment == "" {
			issueComment = "Closed as duplicate"
		}
	default:
		ghArgs = append(ghArgs, "--reason", issueCloseReason)
	}

	if issueComment != "" {
		ghArgs = append(ghArgs, "--comment", issueComment)
	}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to close issue: %w", err)
	}

	fmt.Printf("Issue #%s closed (%s)\n", issueID, issueCloseReason)
	return nil
}

func runIssueAssign(cmd *cobra.Command, args []string) error {
	log.Debug("issue assign command started")

	issueID := args[0]

	// Build gh command
	ghArgs := []string{"issue", "edit", issueID}

	if issueUnassign {
		// Get current assignees and remove them
		ctx := context.Background()
		getCmd := exec.CommandContext(ctx, "gh", "issue", "view", issueID, "--json", "assignees") //nolint:gosec
		var out bytes.Buffer
		getCmd.Stdout = &out

		if err := getCmd.Run(); err != nil {
			return fmt.Errorf("failed to get issue assignees: %w", err)
		}

		var result struct {
			Assignees []struct {
				Login string `json:"login"`
			} `json:"assignees"`
		}
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			return fmt.Errorf("failed to parse assignees: %w", err)
		}

		for _, a := range result.Assignees {
			ghArgs = append(ghArgs, "--remove-assignee", a.Login)
		}

		if len(result.Assignees) == 0 {
			fmt.Printf("Issue #%s has no assignees\n", issueID)
			return nil
		}
	} else if len(args) < 2 {
		return fmt.Errorf("assignee required (or use --unassign)")
	} else {
		assignee := args[1]
		// Remove @ prefix if present
		assignee = strings.TrimPrefix(assignee, "@")
		ghArgs = append(ghArgs, "--add-assignee", assignee)
	}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to assign issue: %w", err)
	}

	if issueUnassign {
		fmt.Printf("Issue #%s unassigned\n", issueID)
	} else {
		fmt.Printf("Issue #%s assigned to %s\n", issueID, args[1])
	}
	return nil
}

func runIssueReopen(cmd *cobra.Command, args []string) error {
	log.Debug("issue reopen command started")

	issueID := args[0]

	// Build gh command
	ghArgs := []string{"issue", "reopen", issueID}

	if issueComment != "" {
		ghArgs = append(ghArgs, "--comment", issueComment)
	}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to reopen issue: %w", err)
	}

	fmt.Printf("Issue #%s reopened\n", issueID)
	return nil
}

func runIssueComment(cmd *cobra.Command, args []string) error {
	log.Debug("issue comment command started")

	issueID := args[0]
	// Join remaining args as the comment body
	commentBody := strings.Join(args[1:], " ")

	if commentBody == "" {
		return fmt.Errorf("comment message is required")
	}

	// Build gh command
	ghArgs := []string{"issue", "comment", issueID, "--body", commentBody}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command
	ghCmd.Stdout = os.Stdout
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to add comment: %w", err)
	}

	fmt.Printf("Comment added to issue #%s\n", issueID)
	return nil
}

func runIssueSearch(cmd *cobra.Command, args []string) error {
	log.Debug("issue search command started")

	// Join all args as search query
	query := strings.Join(args, " ")

	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Build gh search command
	// gh search issues searches across title, body, and comments
	ghArgs := []string{"search", "issues", query, "--repo", getRepoName()}

	// Add state filter
	if issueState != "" && issueState != "all" {
		ghArgs = append(ghArgs, "--state", issueState)
	}

	// Add limit
	ghArgs = append(ghArgs, "--limit", fmt.Sprintf("%d", issueSearchLimit))

	if jsonOutput {
		ghArgs = append(ghArgs, "--json", "number,title,state,labels,author,createdAt")
	}

	ctx := context.Background()
	ghCmd := exec.CommandContext(ctx, "gh", ghArgs...) //nolint:gosec // gh is a trusted command

	var out bytes.Buffer
	ghCmd.Stdout = &out
	ghCmd.Stderr = os.Stderr

	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("failed to search issues: %w", err)
	}

	output := out.String()

	if output == "" || strings.TrimSpace(output) == "" {
		fmt.Printf("No issues found matching: %s\n", query)
		fmt.Println("\nYou may proceed with: bc issue create --title \"...\"")
		return nil
	}

	if jsonOutput {
		var data any
		if err := json.Unmarshal(out.Bytes(), &data); err != nil {
			fmt.Print(output)
			return nil
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}

	fmt.Printf("Found issues matching: %s\n\n", query)
	fmt.Print(output)
	fmt.Println("\nIf none match your issue, proceed with: bc issue create --title \"...\"")
	fmt.Println("If one matches, consider commenting or reopening instead of creating a duplicate.")

	return nil
}

// getRepoName returns the current repository name from gh
func getRepoName() string {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner") //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		return "" // Let gh figure it out from current directory
	}
	return strings.TrimSpace(string(out))
}
