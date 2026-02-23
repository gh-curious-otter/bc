package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/rpuneet/bc/pkg/memory"
)

// Issue #1648: Extracted from memory.go for better code organization
// Memory import/export commands

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

// I/O flags
var (
	memoryExportOutput  string
	memoryExportExp     bool
	memoryExportLearn   bool
	memoryImportReplace bool
	memoryImportDryRun  bool
)

func initMemoryIOFlags() {
	memoryExportCmd.Flags().StringVarP(&memoryExportOutput, "output", "o", "", "Output file (default: stdout)")
	memoryExportCmd.Flags().BoolVar(&memoryExportExp, "experiences", false, "Export only experiences")
	memoryExportCmd.Flags().BoolVar(&memoryExportLearn, "learnings", false, "Export only learnings")

	memoryImportCmd.Flags().BoolVar(&memoryImportReplace, "replace", false, "Replace existing memories instead of merging")
	memoryImportCmd.Flags().BoolVar(&memoryImportDryRun, "dry-run", false, "Preview what would be imported without making changes")

	memoryCmd.AddCommand(memoryExportCmd)
	memoryCmd.AddCommand(memoryForgetCmd)
	memoryCmd.AddCommand(memoryImportCmd)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
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
		return errNotInWorkspace(err)
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
