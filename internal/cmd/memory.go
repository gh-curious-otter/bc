package cmd

import (
	"encoding/json"
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

Requires BC_AGENT_ID environment variable to be set.

Examples:
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

Examples:
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

var memoryExportCmd = &cobra.Command{
	Use:   "export <agent>",
	Short: "Export agent memory to JSON",
	Long: `Export an agent's memory (experiences and learnings) to JSON format.

Output can be written to stdout or saved to a file with --output flag.

Example:
  bc memory export engineer-01                # Print JSON to stdout
  bc memory export engineer-01 --output mem.json  # Save to file
  bc memory export engineer-01 --experiences      # Export only experiences
  bc memory export engineer-01 --learnings        # Export only learnings`,
	Args: cobra.ExactArgs(1),
	RunE: runMemoryExport,
}

var memoryForgetCmd = &cobra.Command{
	Use:   "forget <agent> <topic>",
	Short: "Remove a learning topic from memory",
	Long: `Remove a specific learning topic and all its entries from an agent's memory.

Use 'bc memory list' to see available topics.

Example:
  bc memory forget engineer-01 patterns      # Remove "patterns" topic
  bc memory forget engineer-01 anti-patterns # Remove "anti-patterns" topic`,
	Args: cobra.ExactArgs(2),
	RunE: runMemoryForget,
}

var memoryImportCmd = &cobra.Command{
	Use:   "import <agent> <file>",
	Short: "Import memories from a file",
	Long: `Import experiences and learnings from a JSON file.

The import file should contain an object with optional "experiences" and "learnings" arrays.
By default, imported memories are merged with existing ones.
Use --replace to overwrite all existing memories.

File format (JSON):
  {
    "experiences": [
      {"description": "...", "outcome": "success", ...}
    ],
    "learnings": {
      "category": ["learning1", "learning2"]
    }
  }

Example:
  bc memory import engineer-01 backup.json
  bc memory import engineer-01 backup.json --replace
  bc memory import engineer-01 backup.json --dry-run`,
	Args: cobra.ExactArgs(2),
	RunE: runMemoryImport,
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
	memoryClearExp         bool
	memoryClearLearn       bool
	memoryClearForce       bool
	memoryExportOutput     string
	memoryExportExp        bool
	memoryExportLearn      bool
	memoryImportReplace    bool
	memoryImportDryRun     bool
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

	memoryClearCmd.Flags().BoolVar(&memoryClearExp, "experiences", false, "Clear only experiences")
	memoryClearCmd.Flags().BoolVar(&memoryClearLearn, "learnings", false, "Clear only learnings")
	memoryClearCmd.Flags().BoolVar(&memoryClearForce, "force", false, "Skip confirmation prompt")

	memoryExportCmd.Flags().StringVarP(&memoryExportOutput, "output", "o", "", "Output file (default: stdout)")
	memoryExportCmd.Flags().BoolVar(&memoryExportExp, "experiences", false, "Export only experiences")
	memoryExportCmd.Flags().BoolVar(&memoryExportLearn, "learnings", false, "Export only learnings")

	memoryImportCmd.Flags().BoolVar(&memoryImportReplace, "replace", false, "Replace existing memories instead of merging")
	memoryImportCmd.Flags().BoolVar(&memoryImportDryRun, "dry-run", false, "Preview what would be imported without making changes")

	memoryCmd.AddCommand(memoryRecordCmd)
	memoryCmd.AddCommand(memoryLearnCmd)
	memoryCmd.AddCommand(memoryShowCmd)
	memoryCmd.AddCommand(memorySearchCmd)
	memoryCmd.AddCommand(memoryPruneCmd)
	memoryCmd.AddCommand(memoryListCmd)
	memoryCmd.AddCommand(memoryClearCmd)
	memoryCmd.AddCommand(memoryExportCmd)
	memoryCmd.AddCommand(memoryForgetCmd)
	memoryCmd.AddCommand(memoryImportCmd)
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
		return fmt.Errorf("agent not specified; use --agent flag or set BC_AGENT_ID")
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

	// Determine agent from flag or environment variable
	agentID := memoryLearnAgent
	if agentID == "" {
		agentID = os.Getenv("BC_AGENT_ID")
	}
	if agentID == "" {
		return fmt.Errorf("agent not specified; use --agent flag or set BC_AGENT_ID")
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
		return fmt.Errorf("not in a bc workspace: %w", err)
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
		return fmt.Errorf("not in a bc workspace: %w", err)
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

// MemoryExport represents the exported memory structure.
//
//nolint:govet // JSON field order is more important than memory layout
type MemoryExport struct {
	Agent       string              `json:"agent"`
	ExportedAt  time.Time           `json:"exported_at"`
	Experiences []memory.Experience `json:"experiences,omitempty"`
	Learnings   string              `json:"learnings,omitempty"`
}

func runMemoryExport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	agentID := args[0]
	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		return fmt.Errorf("no memory found for agent %s", agentID)
	}

	export := MemoryExport{
		Agent:      agentID,
		ExportedAt: time.Now().UTC(),
	}

	exportBoth := !memoryExportExp && !memoryExportLearn

	// Get experiences
	if exportBoth || memoryExportExp {
		experiences, expErr := store.GetExperiences()
		if expErr != nil {
			return fmt.Errorf("failed to get experiences: %w", expErr)
		}
		export.Experiences = experiences
	}

	// Get learnings
	if exportBoth || memoryExportLearn {
		learnings, learnErr := store.GetLearnings()
		if learnErr != nil {
			return fmt.Errorf("failed to get learnings: %w", learnErr)
		}
		export.Learnings = learnings
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal export: %w", err)
	}

	// Write output
	if memoryExportOutput != "" {
		if err := os.WriteFile(memoryExportOutput, data, 0600); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		cmd.Printf("Exported memory for %s to %s\n", agentID, memoryExportOutput)
		cmd.Printf("  Experiences: %d\n", len(export.Experiences))
		if export.Learnings != "" {
			cmd.Printf("  Learnings: %d bytes\n", len(export.Learnings))
		}
	} else {
		cmd.Println(string(data))
	}

	return nil
}

func runMemoryForget(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	agentID := args[0]
	topic := args[1]

	store := memory.NewStore(ws.RootDir, agentID)
	if !store.Exists() {
		return fmt.Errorf("no memory found for agent %s", agentID)
	}

	// List available topics to help user
	topics, listErr := store.ListTopics()
	if listErr != nil {
		return fmt.Errorf("failed to list topics: %w", listErr)
	}

	entriesRemoved, err := store.ForgetTopic(topic)
	if err != nil {
		// If topic not found, show available topics
		if len(topics) > 0 {
			cmd.Printf("Available topics for %s: %s\n", agentID, strings.Join(topics, ", "))
		}
		return err
	}

	cmd.Printf("Removed topic %q from %s (%d entries deleted)\n", topic, agentID, entriesRemoved)
	return nil
}

// MemoryImport represents the import file format.
type MemoryImport struct {
	Learnings   map[string][]string `json:"learnings,omitempty"`
	Experiences []memory.Experience `json:"experiences,omitempty"`
}

func runMemoryImport(cmd *cobra.Command, args []string) error {
	ws, err := getWorkspace()
	if err != nil {
		return fmt.Errorf("not in a bc workspace: %w", err)
	}

	agentID := args[0]
	filePath := args[1]

	// Read the import file
	data, err := os.ReadFile(filePath) //nolint:gosec // path provided by user
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	// Parse the import file
	var importData MemoryImport
	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("failed to parse import file: %w", err)
	}

	store := memory.NewStore(ws.RootDir, agentID)

	// Initialize memory if it doesn't exist
	if !store.Exists() {
		if initErr := store.Init(); initErr != nil {
			return fmt.Errorf("failed to initialize memory: %w", initErr)
		}
	}

	// Dry run mode - just show what would be imported
	if memoryImportDryRun {
		cmd.Println("=== Dry Run (no changes will be made) ===")
		cmd.Println()
		cmd.Printf("Agent: %s\n", agentID)
		cmd.Printf("File: %s\n", filePath)
		cmd.Println()

		if memoryImportReplace {
			cmd.Println("Mode: REPLACE (existing memories will be cleared)")
		} else {
			cmd.Println("Mode: MERGE (memories will be added to existing)")
		}
		cmd.Println()

		cmd.Printf("Experiences to import: %d\n", len(importData.Experiences))
		learningCount := 0
		for _, learnings := range importData.Learnings {
			learningCount += len(learnings)
		}
		cmd.Printf("Learnings to import: %d (in %d categories)\n", learningCount, len(importData.Learnings))
		return nil
	}

	// Replace mode - clear existing memories first
	if memoryImportReplace {
		if _, clearErr := store.Clear(true, true); clearErr != nil {
			return fmt.Errorf("failed to clear existing memories: %w", clearErr)
		}
		cmd.Printf("Cleared existing memories for %s\n", agentID)
	}

	// Import experiences
	expCount := 0
	for _, exp := range importData.Experiences {
		if err := store.RecordExperience(exp); err != nil {
			cmd.Printf("Warning: failed to import experience: %v\n", err)
			continue
		}
		expCount++
	}

	// Import learnings
	learnCount := 0
	for category, learnings := range importData.Learnings {
		for _, learning := range learnings {
			if err := store.AddLearning(category, learning); err != nil {
				cmd.Printf("Warning: failed to import learning: %v\n", err)
				continue
			}
			learnCount++
		}
	}

	cmd.Printf("Imported memories for %s:\n", agentID)
	cmd.Printf("  Experiences: %d\n", expCount)
	cmd.Printf("  Learnings: %d\n", learnCount)

	return nil
}
