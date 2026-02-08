package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/memory"
)

// SearchResult represents a ranked search result.
type SearchResult struct {
	AgentID    string
	Source     string // "experience" or "learning"
	Content    string
	Context    string // additional context (outcome, task type, etc.)
	Score      int    // relevance score (higher = more relevant)
	LineNumber int    // for learnings: line number in file
}

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

var memoryPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old memory entries",
	Long: `Remove old experiences to prevent unbounded memory growth.

Creates a backup before deleting. Use --dry-run to preview what would be deleted.

Example:
  bc memory prune --older-than 30d          # Remove entries older than 30 days
  bc memory prune --older-than 7d --dry-run # Preview what would be removed
  bc memory prune --agent engineer-01       # Prune specific agent only`,
	RunE: runMemoryPrune,
}

var (
	memoryOutcome     string
	memoryTaskID      string
	memoryTaskType    string
	memoryShowExp     bool
	memoryShowLearn   bool
	memorySearchAgent string
	memoryPruneAgent  string
	memoryPruneOlder  string
	memoryPruneDryRun bool
)

func init() {
	memoryRecordCmd.Flags().StringVar(&memoryOutcome, "outcome", "success", "Outcome of the task (success, failure, partial)")
	memoryRecordCmd.Flags().StringVar(&memoryTaskID, "task-id", "", "Task ID for the experience")
	memoryRecordCmd.Flags().StringVar(&memoryTaskType, "task-type", "", "Task type (code, review, qa, etc.)")

	memoryShowCmd.Flags().BoolVar(&memoryShowExp, "experiences", false, "Show only experiences")
	memoryShowCmd.Flags().BoolVar(&memoryShowLearn, "learnings", false, "Show only learnings")

	memorySearchCmd.Flags().StringVar(&memorySearchAgent, "agent", "", "Search specific agent's memory")

	memoryPruneCmd.Flags().StringVar(&memoryPruneAgent, "agent", "", "Prune specific agent's memory")
	memoryPruneCmd.Flags().StringVar(&memoryPruneOlder, "older-than", "30d", "Remove entries older than duration (e.g., 7d, 30d)")
	memoryPruneCmd.Flags().BoolVar(&memoryPruneDryRun, "dry-run", false, "Preview what would be deleted without removing")

	memoryCmd.AddCommand(memoryRecordCmd)
	memoryCmd.AddCommand(memoryLearnCmd)
	memoryCmd.AddCommand(memoryShowCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryPruneCmd)
	rootCmd.AddCommand(memoryCmd)
}

func runMemoryRecord(cmd *cobra.Command, args []string) error {
	description := strings.TrimSpace(args[0])
	if description == "" {
		return fmt.Errorf("experience cannot be empty")
	}

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
		Description: description,
		Outcome:     memoryOutcome,
		TaskID:      memoryTaskID,
		TaskType:    memoryTaskType,
	}

	if err := store.RecordExperience(exp); err != nil {
		return fmt.Errorf("failed to record experience: %w", err)
	}

	cmd.Printf("Recorded experience: %s\n", description)
	return nil
}

func runMemoryLearn(cmd *cobra.Command, args []string) error {
	category := strings.TrimSpace(args[0])
	if category == "" {
		return fmt.Errorf("category cannot be empty")
	}
	learning := strings.TrimSpace(args[1])
	if learning == "" {
		return fmt.Errorf("learning cannot be empty")
	}

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

	// Collect and score all results
	var results []SearchResult

	for _, agentID := range agents {
		store := memory.NewStore(ws.RootDir, agentID)

		// Search experiences
		experiences, _ := store.GetExperiences()
		for _, exp := range experiences {
			score := scoreExperience(exp, query)
			if score > 0 {
				context := fmt.Sprintf("Outcome: %s", exp.Outcome)
				if exp.TaskType != "" {
					context += fmt.Sprintf(", Type: %s", exp.TaskType)
				}
				if exp.TaskID != "" {
					context += fmt.Sprintf(", Task: %s", exp.TaskID)
				}
				results = append(results, SearchResult{
					AgentID: agentID,
					Source:  "experience",
					Content: exp.Description,
					Context: context,
					Score:   score,
				})
			}
		}

		// Search learnings
		learnings, _ := store.GetLearnings()
		lines := strings.Split(learnings, "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			score := scoreLearning(trimmed, query)
			if score > 0 {
				results = append(results, SearchResult{
					AgentID:    agentID,
					Source:     "learning",
					Content:    trimmed,
					LineNumber: i + 1,
					Score:      score,
				})
			}
		}
	}

	if len(results) == 0 {
		cmd.Printf("No results found for '%s'\n", args[0])
		return nil
	}

	// Sort by score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Display ranked results
	cmd.Printf("=== Search Results for '%s' (%d found) ===\n\n", args[0], len(results))

	for i, r := range results {
		if r.Source == "experience" {
			cmd.Printf("%d. [%s] Experience (score: %d)\n", i+1, r.AgentID, r.Score)
			cmd.Printf("   %s\n", r.Content)
			cmd.Printf("   %s\n\n", r.Context)
		} else {
			cmd.Printf("%d. [%s] Learning (score: %d, line %d)\n", i+1, r.AgentID, r.Score, r.LineNumber)
			cmd.Printf("   %s\n\n", r.Content)
		}
	}

	return nil
}

// scoreExperience calculates relevance score for an experience.
// Higher score = more relevant. Returns 0 if no match.
func scoreExperience(exp memory.Experience, query string) int {
	score := 0

	// Check description (highest weight)
	descLower := strings.ToLower(exp.Description)
	if strings.Contains(descLower, query) {
		score += 10
		// Bonus for exact word match
		if strings.Contains(descLower, " "+query+" ") ||
			strings.HasPrefix(descLower, query+" ") ||
			strings.HasSuffix(descLower, " "+query) {
			score += 5
		}
	}

	// Check outcome
	if strings.Contains(strings.ToLower(exp.Outcome), query) {
		score += 3
	}

	// Check task type
	if strings.Contains(strings.ToLower(exp.TaskType), query) {
		score += 5
	}

	// Check task ID
	if strings.Contains(strings.ToLower(exp.TaskID), query) {
		score += 5
	}

	// Check learnings in experience
	for _, learning := range exp.Learnings {
		if strings.Contains(strings.ToLower(learning), query) {
			score += 7
		}
	}

	return score
}

// scoreLearning calculates relevance score for a learning line.
// Higher score = more relevant. Returns 0 if no match.
func scoreLearning(line, query string) int {
	lineLower := strings.ToLower(line)
	if !strings.Contains(lineLower, query) {
		return 0
	}

	score := 5

	// Bonus for header lines (categories)
	if strings.HasPrefix(line, "##") {
		score += 3
	}

	// Bonus for exact word match
	if strings.Contains(lineLower, " "+query+" ") ||
		strings.HasPrefix(lineLower, query+" ") ||
		strings.HasSuffix(lineLower, " "+query) {
		score += 5
	}

	// Bonus for multiple occurrences
	count := strings.Count(lineLower, query)
	if count > 1 {
		score += count - 1
	}

	return score
}

func runMemoryPrune(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	// Parse duration
	cutoff, err := parseDuration(memoryPruneOlder)
	if err != nil {
		return fmt.Errorf("invalid duration: %w", err)
	}
	cutoffTime := time.Now().Add(-cutoff)

	// Determine which agents to prune
	var agents []string
	if memoryPruneAgent != "" {
		agents = []string{memoryPruneAgent}
	} else {
		// Prune all agents with memory directories
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

	totalPruned := 0
	for _, agentID := range agents {
		store := memory.NewStore(ws.RootDir, agentID)
		if !store.Exists() {
			continue
		}

		// Get experiences
		experiences, err := store.GetExperiences()
		if err != nil {
			cmd.Printf("Warning: failed to read %s experiences: %v\n", agentID, err)
			continue
		}

		// Find entries to prune
		var toKeep []memory.Experience
		var toPrune []memory.Experience
		for _, exp := range experiences {
			if exp.Timestamp.Before(cutoffTime) {
				toPrune = append(toPrune, exp)
			} else {
				toKeep = append(toKeep, exp)
			}
		}

		if len(toPrune) == 0 {
			continue
		}

		if memoryPruneDryRun {
			cmd.Printf("[%s] Would prune %d entries (keeping %d)\n", agentID, len(toPrune), len(toKeep))
			for _, exp := range toPrune {
				cmd.Printf("  - [%s] %s\n", exp.Timestamp.Format("2006-01-02"), exp.Description)
			}
		} else {
			// Create backup before pruning
			if err := store.BackupExperiences(); err != nil {
				cmd.Printf("Warning: failed to backup %s: %v (continuing anyway)\n", agentID, err)
			}

			// Write kept experiences back
			if err := store.WriteExperiences(toKeep); err != nil {
				return fmt.Errorf("failed to write pruned experiences for %s: %w", agentID, err)
			}

			cmd.Printf("[%s] Pruned %d entries (kept %d)\n", agentID, len(toPrune), len(toKeep))
			totalPruned += len(toPrune)
		}
	}

	if memoryPruneDryRun {
		cmd.Println("\nDry run - no changes made. Remove --dry-run to prune.")
	} else if totalPruned > 0 {
		cmd.Printf("\nTotal pruned: %d entries\n", totalPruned)
	} else {
		cmd.Printf("No entries older than %s found\n", memoryPruneOlder)
	}

	return nil
}

// parseDuration parses duration strings like "7d", "30d", "1h".
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("duration too short: %s", s)
	}

	unit := s[len(s)-1]
	valueStr := s[:len(s)-1]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", valueStr)
	}

	switch unit {
	case 'd':
		return time.Duration(value) * 24 * time.Hour, nil
	case 'h':
		return time.Duration(value) * time.Hour, nil
	case 'm':
		return time.Duration(value) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c (use d, h, or m)", unit)
	}
}
