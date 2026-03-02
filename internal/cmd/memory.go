package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/log"
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

Examples:
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

This command is typically run from within an agent session, or use --agent to specify which agent's memory to update.

Examples:
  bc memory record "Fixed auth bug - used JWT tokens"
  bc memory record --outcome success "Implemented feature X"
  bc memory record --agent eng-01 "Completed task"`,
	Args: cobra.ExactArgs(1),
	RunE: runMemoryRecord,
}

var memoryLearnCmd = &cobra.Command{
	Use:   "learn <category> <learning>",
	Short: "Add a learning to memory",
	Long: `Add an insight or learning to the agent's memory.

This command is typically run from within an agent session, or use --agent to specify which agent's memory to update.

Categories: patterns, anti-patterns, tips, gotchas

Examples:
  bc memory learn patterns "Always check error returns"
  bc memory learn tips "Use context for cancellation"
  bc memory learn --agent eng-01 anti-patterns "Don't ignore errors"`,
	Args: cobra.ExactArgs(2),
	RunE: runMemoryLearn,
}

var memoryShowCmd = &cobra.Command{
	Use:   "show [agent]",
	Short: "Show agent memory",
	Long: `Display the memory contents for an agent.

If no agent is specified, shows memory for the current agent session.

Examples:
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

Examples:
  bc memory search "auth"
  bc memory search --agent engineer-01 "bug"`,
	Args: cobra.ExactArgs(1),
	RunE: runMemorySearch,
}

var memoryPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old experiences from memory",
	Long: `Prune old experiences from agent memory to prevent unbounded growth.

Removes experiences older than the specified duration. Pinned experiences
are always preserved regardless of age.

By default, creates a backup before pruning. Use --no-backup to skip.

Use --learnings to also clear learnings (reset to header only).

Examples:
  bc memory prune --older-than 30d              # Remove experiences older than 30 days
  bc memory prune --older-than 7d --dry-run     # Preview what would be removed
  bc memory prune --older-than 90d --no-backup  # Prune without backup
  bc memory prune --agent engineer-01           # Prune specific agent
  bc memory prune --learnings                   # Also clear learnings`,
	RunE: runMemoryPrune,
}

var memoryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all agent memories",
	Long: `List all learning topics, experiences, and memory usage.

By default, lists all learning topics (categories) across all agents.
Use flags to customize the output.

Examples:
  bc memory list                    # List all learning topics
  bc memory list --experiences      # List all experiences
  bc memory list --with-size        # Show memory usage per agent
  bc memory list --agent NAME       # List specific agent's memory`,
	RunE: runMemoryList,
}

var memoryClearCmd = &cobra.Command{
	Use:   "clear <agent>",
	Short: "Clear an agent's memory",
	Long: `Clear all experiences and/or learnings from an agent's memory.

By default, clears both experiences and learnings.
Use --experiences or --learnings to clear selectively.
Requires confirmation unless --force is specified.

Example:
  bc memory clear engineer-01              # Clear all (requires confirmation)
  bc memory clear engineer-01 --force      # Clear all without confirmation
  bc memory clear engineer-01 --experiences  # Clear only experiences
  bc memory clear engineer-01 --learnings    # Clear only learnings`,
	Args: cobra.ExactArgs(1),
	RunE: runMemoryClear,
}

// Export/Import commands moved to memory_io.go

var memoryEditCmd = &cobra.Command{
	Use:   "edit <agent>",
	Short: "Edit agent memory files in $EDITOR",
	Long: `Open an agent's memory file in your default editor.

Specify which file to edit with one of the flags:
  --learnings      Open learnings.md
  --experiences    Open experiences.jsonl
  --role-prompt    Open role_prompt.md

Uses $EDITOR environment variable, falls back to vi.

Example:
  bc memory edit engineer-01 --learnings      # Edit learnings
  bc memory edit engineer-01 --experiences    # Edit experiences
  bc memory edit engineer-01 --role-prompt    # Edit role prompt`,
	Args: cobra.ExactArgs(1),
	RunE: runMemoryEdit,
}

var memoryDeleteCmd = &cobra.Command{
	Use:   "delete <agent>",
	Short: "Delete a specific experience or learning",
	Long: `Delete a specific item from an agent's memory.

Use --experience to delete an experience by index.
Use --learning to delete a learning by category and index.

Use 'bc memory show <agent>' to see indices.
Indices are 1-based (first item is 1, not 0).

Example:
  bc memory delete engineer-01 --experience 3              # Delete 3rd experience
  bc memory delete engineer-01 --learning patterns 2       # Delete 2nd learning in "patterns"
  bc memory delete engineer-01 --experience 1 --force      # Delete without confirmation`,
	Args: cobra.MinimumNArgs(1),
	RunE: runMemoryDelete,
}

var memoryMergeCmd = &cobra.Command{
	Use:   "merge <target-agent> <source-agent>",
	Short: "Merge learnings from another agent",
	Long: `Merge learnings from one agent's memory into another.

This copies learnings from the source agent to the target agent,
avoiding duplicates. Only learnings are merged, not experiences.

Example:
  bc memory merge engineer-02 engineer-01   # Copy eng-01's learnings to eng-02
  bc memory merge new-agent senior-agent    # Share senior's knowledge with new agent`,
	Args: cobra.ExactArgs(2),
	RunE: runMemoryMerge,
}

var (
	memoryOutcome          string
	memoryTaskID           string
	memoryTaskType         string
	memoryRecordAgent      string
	memoryLearnAgent       string
	memoryShowExp          bool
	memoryShowLearn        bool
	memorySearchAgent      string
	memoryPruneAgent       string
	memoryOlderThan        string
	memoryDryRun           bool
	memoryNoBackup         bool
	memoryIncludeLearnings bool
	memoryListAgent        string
	memoryListExp          bool
	memoryListSize         bool
	memoryListJSON         bool
	memoryClearExp         bool
	memoryClearLearn       bool
	memoryClearForce       bool
	memoryEditLearnings    bool
	memoryEditExperiences  bool
	memoryEditRolePrompt   bool
	memoryDeleteExperience int
	memoryDeleteLearning   string
	memoryDeleteForce      bool
)

func init() {
	memoryRecordCmd.Flags().StringVar(&memoryOutcome, "outcome", "success", "Outcome of the task (success, failure, partial)")
	memoryRecordCmd.Flags().StringVar(&memoryTaskID, "task-id", "", "Task ID for the experience")
	memoryRecordCmd.Flags().StringVar(&memoryTaskType, "task-type", "", "Task type (code, review, qa, etc.)")
	memoryRecordCmd.Flags().StringVar(&memoryRecordAgent, "agent", "", "Agent to record experience for (default: BC_AGENT_ID)")

	memoryLearnCmd.Flags().StringVar(&memoryLearnAgent, "agent", "", "Agent to record learning for (default: BC_AGENT_ID)")

	memoryShowCmd.Flags().BoolVar(&memoryShowExp, "experiences", false, "Show only experiences")
	memoryShowCmd.Flags().BoolVar(&memoryShowLearn, "learnings", false, "Show only learnings")

	memorySearchCmd.Flags().StringVar(&memorySearchAgent, "agent", "", "Search specific agent's memory")

	memoryPruneCmd.Flags().StringVar(&memoryPruneAgent, "agent", "", "Prune specific agent's memory (default: all agents)")
	memoryPruneCmd.Flags().StringVar(&memoryOlderThan, "older-than", "30d", "Remove experiences older than this duration (e.g., 7d, 30d, 90d)")
	memoryPruneCmd.Flags().BoolVar(&memoryDryRun, "dry-run", false, "Preview what would be removed without actually deleting")
	memoryPruneCmd.Flags().BoolVar(&memoryNoBackup, "no-backup", false, "Skip creating backup before pruning")
	memoryPruneCmd.Flags().BoolVar(&memoryIncludeLearnings, "learnings", false, "Also clear learnings (reset to header only)")

	memoryListCmd.Flags().StringVar(&memoryListAgent, "agent", "", "List specific agent's memory")
	memoryListCmd.Flags().BoolVar(&memoryListExp, "experiences", false, "List experiences instead of learning topics")
	memoryListCmd.Flags().BoolVar(&memoryListSize, "with-size", false, "Show memory usage per agent")
	memoryListCmd.Flags().BoolVar(&memoryListJSON, "json", false, "Output as JSON")

	memoryClearCmd.Flags().BoolVar(&memoryClearExp, "experiences", false, "Clear only experiences")
	memoryClearCmd.Flags().BoolVar(&memoryClearLearn, "learnings", false, "Clear only learnings")
	memoryClearCmd.Flags().BoolVar(&memoryClearForce, "force", false, "Skip confirmation prompt")

	memoryEditCmd.Flags().BoolVar(&memoryEditLearnings, "learnings", false, "Edit learnings.md")
	memoryEditCmd.Flags().BoolVar(&memoryEditExperiences, "experiences", false, "Edit experiences.jsonl")
	memoryEditCmd.Flags().BoolVar(&memoryEditRolePrompt, "role-prompt", false, "Edit role_prompt.md")
	memoryEditCmd.MarkFlagsMutuallyExclusive("learnings", "experiences", "role-prompt")

	memoryDeleteCmd.Flags().IntVar(&memoryDeleteExperience, "experience", 0, "Delete experience at this index (1-based)")
	memoryDeleteCmd.Flags().StringVar(&memoryDeleteLearning, "learning", "", "Delete learning from this category (index as next arg)")
	memoryDeleteCmd.Flags().BoolVar(&memoryDeleteForce, "force", false, "Skip confirmation prompt")

	// I/O flags (in memory_io.go)
	initMemoryIOFlags()

	memoryCmd.AddCommand(memoryRecordCmd)
	memoryCmd.AddCommand(memoryLearnCmd)
	memoryCmd.AddCommand(memoryShowCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryPruneCmd)
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryClearCmd)
	memoryCmd.AddCommand(memoryEditCmd)
	memoryCmd.AddCommand(memoryDeleteCmd)
	memoryCmd.AddCommand(memoryMergeCmd)
	rootCmd.AddCommand(memoryCmd)
}

func runMemoryRecord(cmd *cobra.Command, args []string) error {
	description := strings.TrimSpace(args[0])
	if description == "" {
		return fmt.Errorf("experience cannot be empty")
	}

	// Determine agent from flag or environment variable
	agentID := memoryRecordAgent
	if agentID == "" {
		agentID = os.Getenv("BC_AGENT_ID")
	}
	if agentID == "" {
		return errorAgentNotRunning(fmt.Sprintf("bc memory record %q", description))
	}

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
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

	// Determine agent from flag or environment variable
	agentID := memoryLearnAgent
	if agentID == "" {
		agentID = os.Getenv("BC_AGENT_ID")
	}
	if agentID == "" {
		return errorAgentNotRunning(fmt.Sprintf("bc memory learn %s %q", category, learning))
	}

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
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
	log.Debug("memory show command started")

	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Determine which agent's memory to show
	agentID := ""
	if len(args) > 0 {
		agentID = args[0]
	} else {
		agentID = os.Getenv("BC_AGENT_ID")
		if agentID == "" {
			return fmt.Errorf("specify an agent name with --agent, or run this command from within an agent session")
		}
	}

	log.Debug("loading memory", "agent", agentID)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
	}

	// Parse the duration
	duration, err := parseDuration(memoryOlderThan)
	if err != nil {
		return fmt.Errorf("invalid duration '%s': %w", memoryOlderThan, err)
	}

	// Determine which agents to prune
	var agents []string
	if memoryPruneAgent != "" {
		agents = []string{memoryPruneAgent}
	} else {
		// Prune all agents with memory directories
		memoryRoot := filepath.Join(ws.RootDir, ".bc", "memory")
		entries, readErr := os.ReadDir(memoryRoot)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				cmd.Println("No agent memories found")
				return nil
			}
			return fmt.Errorf("failed to read memory directory: %w", readErr)
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

	if memoryDryRun {
		cmd.Println("=== Dry Run (no changes will be made) ===")
		cmd.Println()
	}

	totalPruned := 0
	totalPreserved := 0

	for _, agentID := range agents {
		store := memory.NewStore(ws.RootDir, agentID)
		if !store.Exists() {
			continue
		}

		opts := memory.PruneOptions{
			OlderThan:        duration,
			DryRun:           memoryDryRun,
			Backup:           !memoryNoBackup,
			IncludeLearnings: memoryIncludeLearnings,
		}

		result, pruneErr := store.Prune(opts)
		if pruneErr != nil {
			cmd.Printf("Error pruning %s: %v\n", agentID, pruneErr)
			continue
		}

		if result.PrunedExperiences > 0 || result.PreservedPinned > 0 || result.LearningsCleared {
			cmd.Printf("[%s] ", agentID)
			if memoryDryRun {
				cmd.Printf("Would prune %d/%d experiences", result.PrunedExperiences, result.TotalExperiences)
			} else {
				cmd.Printf("Pruned %d/%d experiences", result.PrunedExperiences, result.TotalExperiences)
			}
			if result.PreservedPinned > 0 {
				cmd.Printf(" (preserved %d pinned)", result.PreservedPinned)
			}
			if result.LearningsCleared {
				if memoryDryRun {
					cmd.Printf(", would clear learnings")
				} else {
					cmd.Printf(", cleared learnings")
				}
			}
			if result.BackupPath != "" {
				cmd.Printf("\n    Backup: %s", result.BackupPath)
			}
			if result.BytesBeforePrune > 0 && !memoryDryRun {
				saved := result.BytesBeforePrune - result.BytesAfterPrune
				cmd.Printf("\n    Freed: %s", formatBytes(saved))
			}
			cmd.Println()
		}

		totalPruned += result.PrunedExperiences
		totalPreserved += result.PreservedPinned
	}

	cmd.Println()
	if memoryDryRun {
		cmd.Printf("Summary: Would prune %d experiences across %d agent(s)\n", totalPruned, len(agents))
	} else {
		if totalPruned > 0 {
			cmd.Printf("Summary: Pruned %d experiences across %d agent(s)\n", totalPruned, len(agents))
		} else {
			cmd.Println("No experiences older than", memoryOlderThan, "found")
		}
	}

	return nil
}

func runMemoryList(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	// Determine which agents to list
	var agents []string
	if memoryListAgent != "" {
		agents = []string{memoryListAgent}
	} else {
		// List all agents with memory directories
		memoryRoot := filepath.Join(ws.RootDir, ".bc", "memory")
		entries, readErr := os.ReadDir(memoryRoot)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				cmd.Println("No agent memories found")
				return nil
			}
			return fmt.Errorf("failed to read memory directory: %w", readErr)
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

	// Sort agents alphabetically
	sort.Strings(agents)

	// JSON output for TUI integration
	if memoryListJSON {
		return listMemoryJSON(cmd, ws.RootDir, agents)
	}

	if memoryListExp {
		// List experiences
		return listExperiences(cmd, ws.RootDir, agents)
	}

	// List learning topics (default)
	return listLearningTopics(cmd, ws.RootDir, agents, memoryListSize)
}

// listLearningTopics lists all learning categories across agents.
func listLearningTopics(cmd *cobra.Command, rootDir string, agents []string, withSize bool) error {
	// Track topics per agent
	type agentTopics struct {
		agent  string
		topics []string
		size   int64
	}

	var allAgentTopics []agentTopics

	for _, agentID := range agents {
		store := memory.NewStore(rootDir, agentID)
		if !store.Exists() {
			continue
		}

		at := agentTopics{agent: agentID}

		// Get size if requested
		if withSize {
			size, _ := store.GetSize()
			at.size = size
		}

		// Get learnings and extract topics (## headings)
		learnings, err := store.GetLearnings()
		if err != nil {
			continue
		}

		lines := strings.Split(learnings, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "## ") {
				topic := strings.TrimPrefix(trimmed, "## ")
				at.topics = append(at.topics, topic)
			}
		}

		allAgentTopics = append(allAgentTopics, at)
	}

	if len(allAgentTopics) == 0 {
		cmd.Println("No learning topics found")
		return nil
	}

	cmd.Println("=== Learning Topics ===")
	cmd.Println()

	for _, at := range allAgentTopics {
		if withSize {
			cmd.Printf("[%s] (%s)\n", at.agent, formatBytes(at.size))
		} else {
			cmd.Printf("[%s]\n", at.agent)
		}

		if len(at.topics) == 0 {
			cmd.Println("  (no topics)")
		} else {
			for _, topic := range at.topics {
				cmd.Printf("  - %s\n", topic)
			}
		}
		cmd.Println()
	}

	return nil
}

// AgentMemorySummary represents a summary of an agent's memory for JSON output.
type AgentMemorySummary struct {
	Agent           string `json:"agent"`
	LastUpdated     string `json:"last_updated,omitempty"`
	ExperienceCount int    `json:"experience_count"`
	LearningCount   int    `json:"learning_count"`
}

// MemoryListResponse is the JSON response for memory list command.
type MemoryListResponse struct {
	Agents []AgentMemorySummary `json:"agents"`
}

// listMemoryJSON outputs agent memory summaries as JSON for TUI integration.
func listMemoryJSON(_ *cobra.Command, rootDir string, agents []string) error {
	var summaries []AgentMemorySummary

	for _, agentID := range agents {
		store := memory.NewStore(rootDir, agentID)
		if !store.Exists() {
			continue
		}

		summary := AgentMemorySummary{Agent: agentID}

		// Count experiences
		experiences, err := store.GetExperiences()
		if err == nil {
			summary.ExperienceCount = len(experiences)
		}

		// Count learning topics (## headings in learnings)
		learnings, err := store.GetLearnings()
		if err == nil {
			lines := strings.Split(learnings, "\n")
			for _, line := range lines {
				if strings.HasPrefix(strings.TrimSpace(line), "## ") {
					summary.LearningCount++
				}
			}
		}

		// Get last updated time from experiences file
		expPath := filepath.Join(store.MemoryDir(), "experiences.jsonl")
		if info, err := os.Stat(expPath); err == nil {
			summary.LastUpdated = info.ModTime().Format(time.RFC3339)
		}

		summaries = append(summaries, summary)
	}

	response := MemoryListResponse{Agents: summaries}
	// #1817: Use json.NewEncoder with os.Stdout directly (like status.go)
	// to ensure JSON goes to stdout, not stderr via cmd.Println
	enc := json.NewEncoder(os.Stdout)
	return enc.Encode(response)
}

// listExperiences lists all experiences across agents.
func listExperiences(cmd *cobra.Command, rootDir string, agents []string) error {
	totalExp := 0

	for _, agentID := range agents {
		store := memory.NewStore(rootDir, agentID)
		if !store.Exists() {
			continue
		}

		experiences, err := store.GetExperiences()
		if err != nil {
			continue
		}

		if len(experiences) == 0 {
			continue
		}

		cmd.Printf("[%s] %d experience(s)\n", agentID, len(experiences))

		// Show most recent 5 experiences per agent
		start := 0
		if len(experiences) > 5 {
			start = len(experiences) - 5
			cmd.Printf("  (showing last 5 of %d)\n", len(experiences))
		}

		for i := start; i < len(experiences); i++ {
			exp := experiences[i]
			date := exp.Timestamp.Format("2006-01-02")
			cmd.Printf("  - [%s] %s: %s\n", exp.Outcome, date, truncate(exp.Description, 60))
		}
		cmd.Println()

		totalExp += len(experiences)
	}

	if totalExp == 0 {
		cmd.Println("No experiences found")
		return nil
	}

	cmd.Printf("Total: %d experience(s) across %d agent(s)\n", totalExp, len(agents))
	return nil
}

// truncate shortens a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// parseDuration parses a duration string like "30d", "7d", "24h".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Check for day suffix (Go's time.ParseDuration doesn't support "d")
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid day count: %w", err)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	// Try standard Go duration parsing for h, m, s
	return time.ParseDuration(s)
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func runMemoryClear(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	agentID := args[0]
	store := memory.NewStore(ws.RootDir, agentID)

	if !store.Exists() {
		return fmt.Errorf("no memory found for agent %s", agentID)
	}

	// Determine what to clear
	clearExp := memoryClearExp || (!memoryClearExp && !memoryClearLearn)
	clearLearn := memoryClearLearn || (!memoryClearExp && !memoryClearLearn)

	// Get current counts for confirmation message
	var expCount int
	if clearExp {
		experiences, _ := store.GetExperiences()
		expCount = len(experiences)
	}

	// Confirmation prompt unless --force
	if !memoryClearForce {
		what := []string{}
		if clearExp {
			what = append(what, fmt.Sprintf("%d experience(s)", expCount))
		}
		if clearLearn {
			what = append(what, "learnings")
		}

		cmd.Printf("This will clear %s for agent %s.\n", strings.Join(what, " and "), agentID)
		cmd.Print("Are you sure? [y/N]: ")

		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil || (response != "y" && response != "Y") {
			cmd.Println("Aborted.")
			return nil
		}
	}

	result, err := store.Clear(clearExp, clearLearn)
	if err != nil {
		return fmt.Errorf("failed to clear memory: %w", err)
	}

	// Report what was cleared
	if result.ExperiencesCleared > 0 {
		cmd.Printf("Cleared %d experience(s)\n", result.ExperiencesCleared)
	}
	if result.LearningsCleared {
		cmd.Println("Cleared learnings (reset to header)")
	}
	cmd.Printf("Memory cleared for agent %s\n", agentID)

	return nil
}

// Export/Import/Forget functions moved to memory_io.go

func runMemoryEdit(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	agentID := args[0]
	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		return fmt.Errorf("no memory found for agent %s", agentID)
	}

	// Determine which file to edit
	var filePath string
	switch {
	case memoryEditLearnings:
		filePath = filepath.Join(store.MemoryDir(), "learnings.md")
	case memoryEditExperiences:
		filePath = filepath.Join(store.MemoryDir(), "experiences.jsonl")
	case memoryEditRolePrompt:
		filePath = filepath.Join(store.MemoryDir(), "role_prompt.md")
	default:
		return fmt.Errorf("specify which file to edit: --learnings, --experiences, or --role-prompt")
	}

	// Check file exists
	if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}

	// Get editor from $EDITOR, fall back to vi
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Open editor with inherited terminal
	ctx := cmd.Context()
	editorCmd := exec.CommandContext(ctx, editor, filePath) //nolint:gosec // editor is user-controlled via $EDITOR
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if runErr := editorCmd.Run(); runErr != nil {
		return fmt.Errorf("editor exited with error: %w", runErr)
	}

	cmd.Printf("Edited %s for agent %s\n", filepath.Base(filePath), agentID)
	return nil
}

func runMemoryDelete(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	agentID := args[0]
	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		return fmt.Errorf("no memory found for agent %s", agentID)
	}

	switch {
	case memoryDeleteExperience > 0:
		return deleteExperience(cmd, store, agentID, memoryDeleteExperience)
	case memoryDeleteLearning != "":
		// Index is the next positional arg after agent
		if len(args) < 2 {
			return fmt.Errorf("missing index: usage: bc memory delete <agent> --learning <category> <index>")
		}
		index, parseErr := strconv.Atoi(args[1])
		if parseErr != nil {
			return fmt.Errorf("invalid index %q: must be a number", args[1])
		}
		return deleteLearning(cmd, store, agentID, memoryDeleteLearning, index)
	default:
		return fmt.Errorf("specify what to delete: --experience <index> or --learning <category> <index>")
	}
}

func deleteExperience(cmd *cobra.Command, store *memory.Store, agentID string, index int) error {
	// Show the item for confirmation
	experiences, err := store.GetExperiences()
	if err != nil {
		return fmt.Errorf("failed to get experiences: %w", err)
	}
	idx := index - 1
	if idx < 0 || idx >= len(experiences) {
		return fmt.Errorf("index %d out of range (1-%d)", index, len(experiences))
	}

	exp := experiences[idx]
	if !memoryDeleteForce {
		cmd.Printf("Delete experience #%d from %s?\n", index, agentID)
		cmd.Printf("  [%s] %s\n", exp.Outcome, exp.Description)
		cmd.Print("Are you sure? [y/N]: ")

		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil || (response != "y" && response != "Y") {
			cmd.Println("Aborted.")
			return nil
		}
	}

	deleted, err := store.DeleteExperience(index)
	if err != nil {
		return fmt.Errorf("failed to delete experience: %w", err)
	}

	cmd.Printf("Deleted experience #%d from %s:\n", index, agentID)
	cmd.Printf("  [%s] %s\n", deleted.Outcome, deleted.Description)
	if !deleted.Timestamp.IsZero() {
		cmd.Printf("  Time: %s\n", deleted.Timestamp.Format("2006-01-02 15:04:05"))
	}
	return nil
}

func deleteLearning(cmd *cobra.Command, store *memory.Store, agentID, category string, index int) error {
	if !memoryDeleteForce {
		cmd.Printf("Delete learning #%d from category %q for %s?\n", index, category, agentID)
		cmd.Print("Are you sure? [y/N]: ")

		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil || (response != "y" && response != "Y") {
			cmd.Println("Aborted.")
			return nil
		}
	}

	deleted, err := store.DeleteLearning(category, index)
	if err != nil {
		return fmt.Errorf("failed to delete learning: %w", err)
	}

	cmd.Printf("Deleted learning #%d from %s [%s]:\n", index, agentID, category)
	cmd.Printf("  %s\n", deleted)
	return nil
}

func runMemoryMerge(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return errNotInWorkspace(err)
	}

	targetAgent := args[0]
	sourceAgent := args[1]

	// Check source exists
	srcStore := memory.NewStore(ws.RootDir, sourceAgent)
	if !srcStore.Exists() {
		return fmt.Errorf("source agent %q has no memory", sourceAgent)
	}

	// Initialize target if needed
	dstStore := memory.NewStore(ws.RootDir, targetAgent)
	if !dstStore.Exists() {
		if initErr := dstStore.Init(); initErr != nil {
			return fmt.Errorf("failed to initialize memory for %s: %w", targetAgent, initErr)
		}
		cmd.Printf("Initialized memory for %s\n", targetAgent)
	}

	// Merge learnings
	added, err := dstStore.MergeLearnings(srcStore)
	if err != nil {
		return fmt.Errorf("failed to merge learnings: %w", err)
	}

	if added == 0 {
		cmd.Printf("No new learnings to merge from %s to %s\n", sourceAgent, targetAgent)
	} else {
		cmd.Printf("Merged %d new learning(s) from %s to %s\n", added, sourceAgent, targetAgent)
	}

	return nil
}
