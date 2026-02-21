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
  bc issue list --labels test-failure  # List by label
  bc issue list --assignee @me         # List assigned to me
  bc issue list --type bug             # List by type`,
	RunE: runIssueList,
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
)

func init() {
	// issue create flags
	issueCreateCmd.Flags().StringVar(&issueType, "type", "bug", "Issue type (epic, bug, task, chore, feature, enhancement)")
	issueCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title")
	issueCreateCmd.Flags().StringVar(&issueDescription, "description", "", "Issue description")
	issueCreateCmd.Flags().StringVar(&issueLabels, "labels", "", "Comma-separated labels")
	issueCreateCmd.Flags().StringVar(&issueSeverity, "severity", "medium", "Severity (critical, high, medium, low)")
	issueCreateCmd.Flags().StringVar(&issueReproSteps, "reproduction", "", "Reproduction steps")
	issueCreateCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Assign to user")

	// issue list flags
	issueListCmd.Flags().StringVar(&issueLabels, "labels", "", "Filter by labels")
	issueListCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Filter by assignee (@me for self)")
	issueListCmd.Flags().StringVar(&issueType, "type", "", "Filter by type (epic, bug, task, etc.)")

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

	// Add subcommands
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueEditCmd)
	issueCmd.AddCommand(issueCloseCmd)
	issueCmd.AddCommand(issueAssignCmd)

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
	ghArgs := make([]string, 0, 10)
	ghArgs = append(ghArgs, "issue", "list")

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
		ghArgs = append(ghArgs, "--json", "number,title,labels,state,assignees,createdAt")
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
