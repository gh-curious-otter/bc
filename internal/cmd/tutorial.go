package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/ui"
)

// Tutorial represents an interactive tutorial module.
type Tutorial struct {
	Name        string
	Title       string
	Description string
	Steps       []TutorialStep
}

// TutorialStep represents a single step in a tutorial.
type TutorialStep struct {
	Title       string
	Explanation string
	Command     string
	Hint        string
}

// Available tutorials
var tutorials = map[string]Tutorial{
	"getting-started": {
		Name:        "getting-started",
		Title:       "Getting Started with bc",
		Description: "Learn the basics of bc agent orchestration",
		Steps: []TutorialStep{
			{
				Title:       "Welcome to bc!",
				Explanation: "bc is an AI agent orchestration CLI that helps you manage multiple AI assistants working together on your projects.\n\nIn this tutorial, you'll learn:\n- How to create and manage agents\n- How agents communicate via channels\n- Basic workflow commands",
			},
			{
				Title:       "Check Your Workspace",
				Explanation: "First, let's check if you're in a bc workspace.\n\nA workspace is a directory with a .bc/ folder containing your configuration.",
				Command:     "bc status",
				Hint:        "If you see 'not in a bc workspace', run 'bc init' first.",
			},
			{
				Title:       "Create Your First Agent",
				Explanation: "Agents are AI assistants that run in isolated tmux sessions.\nEach agent has a role that defines its capabilities.\n\nLet's create an engineer agent:",
				Command:     "bc agent create eng-01 --role engineer",
				Hint:        "The --role flag is required. Common roles: engineer, manager, qa",
			},
			{
				Title:       "List Your Agents",
				Explanation: "You can see all your agents and their status with the list command:",
				Command:     "bc agent list",
				Hint:        "Use --json for machine-readable output",
			},
			{
				Title:       "Send a Message to an Agent",
				Explanation: "Communicate with agents by sending them messages.\nThe message is typed into their tmux session:",
				Command:     "bc agent send eng-01 \"Hello, what can you help me with?\"",
				Hint:        "Use 'bc agent peek eng-01' to see the agent's output",
			},
			{
				Title:       "View Agent Output",
				Explanation: "See what an agent is working on by peeking at their session:",
				Command:     "bc agent peek eng-01",
				Hint:        "Use --lines N to see more or fewer lines",
			},
			{
				Title:       "Attach to an Agent",
				Explanation: "For direct interaction, attach to an agent's tmux session:",
				Command:     "bc agent attach eng-01",
				Hint:        "Press Ctrl+b then d to detach and return to your shell",
			},
			{
				Title:       "Stop an Agent",
				Explanation: "When you're done, stop the agent to free resources:",
				Command:     "bc agent stop eng-01",
				Hint:        "Use 'bc agent start eng-01' to restart a stopped agent",
			},
			{
				Title:       "Open the Dashboard",
				Explanation: "For a visual overview, open the TUI dashboard:",
				Command:     "bc",
				Hint:        "Press q to exit, ? for help within the dashboard",
			},
			{
				Title:       "Tutorial Complete!",
				Explanation: "Congratulations! You've learned the basics of bc.\n\nNext steps:\n- Read the docs: bc --help\n- Explore channels: bc channel --help\n- Track costs: bc cost --help\n\nHappy orchestrating!",
			},
		},
	},
	"agents": {
		Name:        "agents",
		Title:       "Managing Agents",
		Description: "Deep dive into agent lifecycle and management",
		Steps: []TutorialStep{
			{
				Title:       "Agent Roles",
				Explanation: "Agents have roles that define their capabilities:\n\n- engineer: Implements features, fixes bugs\n- manager: Coordinates work, reviews PRs\n- product-manager: Defines requirements\n- tech-lead: Architectural decisions\n- qa: Testing and quality assurance\n\nView available roles:",
				Command:     "bc role list",
			},
			{
				Title:       "Creating Agents",
				Explanation: "Create agents with specific roles and tools:\n\nThe default AI tool is Claude, but you can specify others:",
				Command:     "bc agent create qa-01 --role qa",
				Hint:        "Add --tool cursor to use Cursor AI instead",
			},
			{
				Title:       "Agent Health",
				Explanation: "Monitor agent health and detect stuck agents:",
				Command:     "bc agent health",
				Hint:        "Use --detect-stuck to find agents that aren't making progress",
			},
			{
				Title:       "Broadcast Messages",
				Explanation: "Send a message to all running agents at once:",
				Command:     "bc agent broadcast \"Please report your status\"",
			},
			{
				Title:       "Agent Cleanup",
				Explanation: "Delete agents you no longer need:",
				Command:     "bc agent delete eng-01",
				Hint:        "Use --force to delete running agents, --purge to remove memory",
			},
		},
	},
	"channels": {
		Name:        "channels",
		Title:       "Channel Communication",
		Description: "Learn how agents communicate via channels",
		Steps: []TutorialStep{
			{
				Title:       "What are Channels?",
				Explanation: "Channels are persistent message streams for agent communication.\n\nDefault channels:\n- #general: Team-wide announcements\n- #eng: Engineering discussions\n- #pr: Pull request notifications",
			},
			{
				Title:       "List Channels",
				Explanation: "See all available channels:",
				Command:     "bc channel list",
			},
			{
				Title:       "View Channel Messages",
				Explanation: "Read recent messages in a channel:",
				Command:     "bc channel history general --limit 10",
			},
			{
				Title:       "Send to a Channel",
				Explanation: "Post a message to a channel:",
				Command:     "bc channel send general \"Hello team!\"",
				Hint:        "Messages are delivered to all channel members",
			},
			{
				Title:       "Create a Channel",
				Explanation: "Create a new channel for specific topics:",
				Command:     "bc channel create backend",
				Hint:        "Agents can be added as members with bc channel join",
			},
		},
	},
}

var tutorialCmd = &cobra.Command{
	Use:   "tutorial [name]",
	Short: "Interactive tutorials for learning bc",
	Long: `Start an interactive tutorial to learn bc commands and workflows.

Available tutorials:
  getting-started   Learn the basics of bc agent orchestration
  agents            Deep dive into agent lifecycle and management
  channels          Learn how agents communicate via channels

Examples:
  bc tutorial                    # Start the getting-started tutorial
  bc tutorial --list             # List all available tutorials
  bc tutorial agents             # Start the agents tutorial`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTutorial,
}

var tutorialList bool

func init() {
	tutorialCmd.Flags().BoolVar(&tutorialList, "list", false, "List available tutorials")
	rootCmd.AddCommand(tutorialCmd)
}

func runTutorial(cmd *cobra.Command, args []string) error {
	// List tutorials if --list flag is set or "list" argument is provided (#1532)
	if tutorialList || (len(args) > 0 && args[0] == "list") {
		return listTutorials()
	}

	// Default to getting-started tutorial
	tutorialName := "getting-started"
	if len(args) > 0 {
		tutorialName = args[0]
	}

	tutorial, ok := tutorials[tutorialName]
	if !ok {
		fmt.Printf("Unknown tutorial: %s\n\n", tutorialName)
		return listTutorials()
	}

	return runInteractiveTutorial(tutorial)
}

func listTutorials() error {
	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText("Available Tutorials"))
	fmt.Println("  " + strings.Repeat("─", 50))
	fmt.Println()

	for _, name := range []string{"getting-started", "agents", "channels"} {
		t := tutorials[name]
		fmt.Printf("  %s\n", ui.CyanText(t.Name))
		fmt.Printf("    %s\n", t.Description)
		fmt.Printf("    %s\n", ui.GrayText(fmt.Sprintf("%d steps", len(t.Steps))))
		fmt.Println()
	}

	fmt.Println("  Run: bc tutorial <name>")
	fmt.Println()
	return nil
}

func runInteractiveTutorial(tutorial Tutorial) error {
	reader := bufio.NewReader(os.Stdin)
	isInteractive := term.IsTerminal(os.Stdin.Fd())

	// Print header
	fmt.Println()
	fmt.Printf("  %s\n", ui.BoldText(tutorial.Title))
	fmt.Println("  " + strings.Repeat("═", 50))
	fmt.Println()

	for i, step := range tutorial.Steps {
		// Step header
		fmt.Printf("  %s [%d/%d]\n", ui.CyanText(step.Title), i+1, len(tutorial.Steps))
		fmt.Println("  " + strings.Repeat("─", 40))
		fmt.Println()

		// Explanation
		lines := strings.Split(step.Explanation, "\n")
		for _, line := range lines {
			fmt.Printf("  %s\n", line)
		}
		fmt.Println()

		// Command to run (if any)
		if step.Command != "" {
			fmt.Printf("  %s\n", ui.GrayText("Try this command:"))
			fmt.Printf("  %s %s\n", ui.GreenText(">"), step.Command)
			fmt.Println()
		}

		// Hint (if any)
		if step.Hint != "" {
			fmt.Printf("  %s %s\n", ui.YellowText("Hint:"), step.Hint)
			fmt.Println()
		}

		// Navigation prompt
		if i < len(tutorial.Steps)-1 {
			if isInteractive {
				fmt.Print("  [Enter] Continue  [s] Skip  [q] Quit: ")
				input, err := reader.ReadString('\n')
				if err != nil {
					return nil
				}
				input = strings.ToLower(strings.TrimSpace(input))
				if input == "q" {
					fmt.Println("\n  Tutorial exited. Run 'bc tutorial' to resume.")
					return nil
				}
				// 's' just continues to next step
			} else {
				// Non-interactive: just continue
				fmt.Println()
			}
		}

		fmt.Println()
	}

	fmt.Printf("  %s\n", ui.GreenText("Tutorial complete!"))
	fmt.Println()

	return nil
}
