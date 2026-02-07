package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/memory"
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage agent memory (experiences and learnings)",
	Long: `Commands for managing per-agent memory storage.

Each agent has a memory directory at .bc/memory/<agent-name>/ containing:
  - experiences.jsonl: Recorded task outcomes
  - learnings.md: Accumulated insights

Example:
  bc memory record "Fixed auth bug"         # Record experience
  bc memory learn "patterns" "Always test"  # Add learning
  bc memory show                            # Show memory for current agent
  bc memory show engineer-01                # Show specific agent's memory
  bc memory search "auth"                   # Search memories`,
}

var memoryRecordCmd = &cobra.Command{
	Use:   "record <description>",
	Short: "Record an experience to memory",
	Long: `Record a task outcome or experience to the agent's memory.

Requires BC_AGENT_ID environment variable to be set.

Example:
  bc memory record "Fixed auth bug - used JWT tokens"
  bc memory record --outcome success "Implemented feature X"
  bc memory record --task-id TASK-123 "Completed task"`,
	Args: cobra.ExactArgs(1),
	RunE: runMemoryRecord,
}

var memoryLearnCmd = &cobra.Command{
	Use:   "learn <category> <learning>",
	Short: "Add a learning to memory",
	Long: `Add an insight or learning to the agent's memory.

Requires BC_AGENT_ID environment variable to be set.

Categories: patterns, anti-patterns, tips, gotchas

Example:
  bc memory learn patterns "Always check error returns"
  bc memory learn tips "Use context for cancellation"
  bc memory learn anti-patterns "Don't ignore errors"`,
	Args: cobra.ExactArgs(2),
	RunE: runMemoryLearn,
}

var memoryShowCmd = &cobra.Command{
	Use:   "show [agent]",
	Short: "Show agent memory",
	Long: `Display the memory contents for an agent.

If no agent is specified, uses BC_AGENT_ID environment variable.

Example:
  bc memory show                # Show current agent's memory
  bc memory show engineer-01    # Show specific agent's memory
  bc memory show --experiences  # Show only experiences
  bc memory show --learnings    # Show only learnings`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMemoryShow,
}

var memorySearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search agent memories",
	Long: `Search through experiences and learnings for matching content.

Example:
  bc memory search "auth"
  bc memory search --agent engineer-01 "bug"`,
	Args: cobra.ExactArgs(1),
	RunE: runMemorySearch,
}

var (
	memoryOutcome     string
	memoryTaskID      string
	memoryTaskType    string
	memoryShowExp     bool
	memoryShowLearn   bool
	memorySearchAgent string
)

func init() {
	memoryRecordCmd.Flags().StringVar(&memoryOutcome, "outcome", "success", "Outcome of the task (success, failure, partial)")
	memoryRecordCmd.Flags().StringVar(&memoryTaskID, "task-id", "", "Task ID for the experience")
	memoryRecordCmd.Flags().StringVar(&memoryTaskType, "task-type", "", "Task type (code, review, qa, etc.)")

	memoryShowCmd.Flags().BoolVar(&memoryShowExp, "experiences", false, "Show only experiences")
	memoryShowCmd.Flags().BoolVar(&memoryShowLearn, "learnings", false, "Show only learnings")

	memorySearchCmd.Flags().StringVar(&memorySearchAgent, "agent", "", "Search specific agent's memory")

	memoryCmd.AddCommand(memoryRecordCmd)
	memoryCmd.AddCommand(memoryLearnCmd)
	memoryCmd.AddCommand(memoryShowCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	rootCmd.AddCommand(memoryCmd)
}

func runMemoryRecord(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("BC_AGENT_ID not set (this command is meant to be called by agents)")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		if initErr := store.Init(); initErr != nil {
			return fmt.Errorf("failed to initialize memory: %w", initErr)
		}
	}

	exp := memory.Experience{
		Description: args[0],
		Outcome:     memoryOutcome,
		TaskID:      memoryTaskID,
		TaskType:    memoryTaskType,
	}

	if err := store.RecordExperience(exp); err != nil {
		return fmt.Errorf("failed to record experience: %w", err)
	}

	cmd.Printf("Recorded experience: %s\n", args[0])
	return nil
}

func runMemoryLearn(cmd *cobra.Command, args []string) error {
	agentID := os.Getenv("BC_AGENT_ID")
	if agentID == "" {
		return fmt.Errorf("BC_AGENT_ID not set (this command is meant to be called by agents)")
	}

	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	category := args[0]
	learning := args[1]

	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		if initErr := store.Init(); initErr != nil {
			return fmt.Errorf("failed to initialize memory: %w", initErr)
		}
	}

	if err := store.AddLearning(category, learning); err != nil {
		return fmt.Errorf("failed to add learning: %w", err)
	}

	cmd.Printf("Added learning (%s): %s\n", category, learning)
	return nil
}

func runMemoryShow(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Determine which agent's memory to show
	agentID := ""
	if len(args) > 0 {
		agentID = args[0]
	} else {
		agentID = os.Getenv("BC_AGENT_ID")
		if agentID == "" {
			return fmt.Errorf("specify an agent name or set BC_AGENT_ID")
		}
	}

	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		cmd.Printf("No memory found for agent %s\n", agentID)
		return nil
	}

	showBoth := !memoryShowExp && !memoryShowLearn

	// Show experiences
	if showBoth || memoryShowExp {
		experiences, err := store.GetExperiences()
		if err != nil {
			return fmt.Errorf("failed to get experiences: %w", err)
		}

		cmd.Printf("=== %s Experiences ===\n\n", agentID)
		if len(experiences) == 0 {
			cmd.Println("No experiences recorded.")
			cmd.Println()
		} else {
			for i, exp := range experiences {
				cmd.Printf("%d. [%s] %s\n", i+1, exp.Outcome, exp.Description)
				if exp.TaskID != "" {
					cmd.Printf("   Task: %s", exp.TaskID)
					if exp.TaskType != "" {
						cmd.Printf(" (%s)", exp.TaskType)
					}
					cmd.Println()
				}
				if !exp.Timestamp.IsZero() {
					cmd.Printf("   Time: %s\n", exp.Timestamp.Format("2006-01-02 15:04:05"))
				}
			}
			cmd.Println()
		}
	}

	// Show learnings
	if showBoth || memoryShowLearn {
		learnings, err := store.GetLearnings()
		if err != nil {
			return fmt.Errorf("failed to get learnings: %w", err)
		}

		cmd.Printf("=== %s Learnings ===\n\n", agentID)
		if learnings == "" {
			cmd.Println("No learnings recorded.")
			cmd.Println()
		} else {
			cmd.Println(learnings)
		}
	}

	return nil
}

func runMemorySearch(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	query := strings.ToLower(args[0])

	// Determine which agents to search
	var agents []string
	if memorySearchAgent != "" {
		agents = []string{memorySearchAgent}
	} else {
		// Search all agents with memory directories
		memoryRoot := filepath.Join(ws.RootDir, ".bc", "memory")
		entries, err := os.ReadDir(memoryRoot)
		if err != nil {
			if os.IsNotExist(err) {
				cmd.Println("No agent memories found")
				return nil
			}
			return fmt.Errorf("failed to read memory directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() {
				agents = append(agents, entry.Name())
			}
		}
	}

	if len(agents) == 0 {
		cmd.Println("No agent memories found")
		return nil
	}

	found := false
	for _, agentID := range agents {
		store := memory.NewStore(ws.RootDir, agentID)

		// Search experiences
		experiences, _ := store.GetExperiences()
		for _, exp := range experiences {
			if strings.Contains(strings.ToLower(exp.Description), query) ||
				strings.Contains(strings.ToLower(exp.Outcome), query) {
				if !found {
					cmd.Println("=== Search Results ===")
					cmd.Println()
					found = true
				}
				cmd.Printf("[%s] Experience: %s\n", agentID, exp.Description)
				cmd.Printf("  Outcome: %s\n", exp.Outcome)
				cmd.Println()
			}
		}

		// Search learnings
		learnings, _ := store.GetLearnings()
		lines := strings.Split(learnings, "\n")
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), query) {
				if !found {
					cmd.Println("=== Search Results ===")
					cmd.Println()
					found = true
				}
				// Print context: the line and surrounding lines
				cmd.Printf("[%s] Learnings (line %d): %s\n\n", agentID, i+1, strings.TrimSpace(line))
			}
		}
	}

	if !found {
		cmd.Printf("No results found for '%s'\n", args[0])
	}

	return nil
}
