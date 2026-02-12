// Package memory provides per-agent memory storage for experiences and learnings.
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// Experience represents a recorded task outcome.
type Experience struct {
	Timestamp   time.Time      `json:"timestamp"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	TaskID      string         `json:"task_id,omitempty"`
	TaskType    string         `json:"task_type,omitempty"`
	Description string         `json:"description"`
	Outcome     string         `json:"outcome"`
	Learnings   []string       `json:"learnings,omitempty"`
	Pinned      bool           `json:"pinned,omitempty"`
}

// Store provides memory storage for an agent.
type Store struct {
	agentName string
	memoryDir string
}

// NewStore creates a new memory store for an agent.
func NewStore(rootDir, agentName string) *Store {
	return &Store{
		agentName: agentName,
		memoryDir: filepath.Join(rootDir, ".bc", "memory", agentName),
	}
}

// Init creates the memory directory structure for an agent.
// Creates:
//   - .bc/memory/<agent-name>/
//   - .bc/memory/<agent-name>/experiences.jsonl
//   - .bc/memory/<agent-name>/learnings.md
func (s *Store) Init() error {
	// Create memory directory
	if err := os.MkdirAll(s.memoryDir, 0750); err != nil {
		return fmt.Errorf("failed to create memory directory: %w", err)
	}

	// Initialize experiences.jsonl if it doesn't exist
	experiencesPath := s.experiencesPath()
	if _, err := os.Stat(experiencesPath); os.IsNotExist(err) {
		f, createErr := os.Create(experiencesPath) //nolint:gosec // path constructed from trusted memoryDir
		if createErr != nil {
			return fmt.Errorf("failed to create experiences file: %w", createErr)
		}
		_ = f.Close()
	}

	// Initialize learnings.md if it doesn't exist
	learningsPath := s.learningsPath()
	if _, err := os.Stat(learningsPath); os.IsNotExist(err) {
		initialContent := fmt.Sprintf("# %s Learnings\n\nThis file contains insights and learnings accumulated by %s.\n\n", s.agentName, s.agentName)
		if writeErr := os.WriteFile(learningsPath, []byte(initialContent), 0600); writeErr != nil { //nolint:gosec // path constructed from trusted memoryDir
			return fmt.Errorf("failed to create learnings file: %w", writeErr)
		}
	}

	return nil
}

// Exists checks if the memory directory exists for this agent.
func (s *Store) Exists() bool {
	_, err := os.Stat(s.memoryDir)
	return err == nil
}

// RecordExperience appends an experience to the experiences.jsonl file.
func (s *Store) RecordExperience(exp Experience) error {
	if exp.Timestamp.IsZero() {
		exp.Timestamp = time.Now().UTC()
	}

	f, err := os.OpenFile(s.experiencesPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600) //nolint:gosec // path constructed from trusted memoryDir
	if err != nil {
		return fmt.Errorf("failed to open experiences file: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, marshalErr := json.Marshal(exp)
	if marshalErr != nil {
		return fmt.Errorf("failed to marshal experience: %w", marshalErr)
	}

	if _, writeErr := f.Write(append(data, '\n')); writeErr != nil {
		return fmt.Errorf("failed to write experience: %w", writeErr)
	}

	return nil
}

// AddLearning appends a learning to the learnings.md file.
// If the category already exists, the learning is appended under that category.
// Otherwise, a new category section is created.
func (s *Store) AddLearning(category, learning string) error {
	// Read existing content
	content, err := s.GetLearnings()
	if err != nil {
		return fmt.Errorf("failed to read learnings: %w", err)
	}

	categoryHeader := "## " + category
	newLearning := "- " + learning

	var newContent string
	if strings.Contains(content, categoryHeader) {
		// Category exists - insert learning after the header
		// Find the category header position
		headerIdx := strings.Index(content, categoryHeader)
		// Find the end of the header line
		headerEndIdx := headerIdx + len(categoryHeader)
		if headerEndIdx < len(content) && content[headerEndIdx] == '\n' {
			headerEndIdx++
		}

		// Skip any blank lines after the header
		for headerEndIdx < len(content) && content[headerEndIdx] == '\n' {
			headerEndIdx++
		}

		// Insert the new learning
		newContent = content[:headerEndIdx] + newLearning + "\n" + content[headerEndIdx:]
	} else {
		// Category doesn't exist - append new section
		newContent = content + "\n## " + category + "\n\n" + newLearning + "\n"
	}

	// Write the updated content
	if err := os.WriteFile(s.learningsPath(), []byte(newContent), 0600); err != nil { //nolint:gosec // path constructed from trusted memoryDir
		return fmt.Errorf("failed to write learnings: %w", err)
	}

	return nil
}

// GetExperiences reads all experiences from the experiences.jsonl file.
func (s *Store) GetExperiences() ([]Experience, error) {
	data, err := os.ReadFile(s.experiencesPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read experiences: %w", err)
	}

	var experiences []Experience
	lines := splitLines(data)
	for i, line := range lines {
		if len(line) == 0 {
			continue
		}
		var exp Experience
		if unmarshalErr := json.Unmarshal(line, &exp); unmarshalErr != nil {
			log.Warn("skipping malformed experience entry", "line", i+1, "error", unmarshalErr)
			continue
		}
		experiences = append(experiences, exp)
	}

	return experiences, nil
}

// GetLearnings reads the learnings.md file content.
func (s *Store) GetLearnings() (string, error) {
	data, err := os.ReadFile(s.learningsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read learnings: %w", err)
	}
	return string(data), nil
}

// clearLearnings resets the learnings file to just the header.
func (s *Store) clearLearnings() error {
	header := fmt.Sprintf("# %s Learnings\n\nThis file contains insights and learnings accumulated by %s.\n\n", s.agentName, s.agentName)
	if err := os.WriteFile(s.learningsPath(), []byte(header), 0600); err != nil { //nolint:gosec // path constructed from trusted memoryDir
		return fmt.Errorf("failed to reset learnings file: %w", err)
	}
	return nil
}

// ClearResult holds the result of a Clear operation.
type ClearResult struct {
	ExperiencesCleared int  // number of experiences removed
	LearningsCleared   bool // whether learnings were cleared
}

// Clear removes all experiences and/or learnings from the agent's memory.
// If clearExperiences is true, removes all experiences.
// If clearLearnings is true, resets learnings to the header.
func (s *Store) Clear(clearExperiences, clearLearningsFlag bool) (*ClearResult, error) {
	result := &ClearResult{}

	if clearExperiences {
		// Count existing experiences before clearing
		existing, _ := s.GetExperiences()
		result.ExperiencesCleared = len(existing)

		// Clear by writing empty experiences file
		if err := os.WriteFile(s.experiencesPath(), []byte{}, 0600); err != nil { //nolint:gosec // path constructed from trusted memoryDir
			return nil, fmt.Errorf("failed to clear experiences: %w", err)
		}
	}

	if clearLearningsFlag {
		if err := s.clearLearnings(); err != nil {
			return nil, err
		}
		result.LearningsCleared = true
	}

	return result, nil
}

// MemoryDir returns the path to the agent's memory directory.
func (s *Store) MemoryDir() string {
	return s.memoryDir
}

func (s *Store) experiencesPath() string {
	return filepath.Join(s.memoryDir, "experiences.jsonl")
}

func (s *Store) learningsPath() string {
	return filepath.Join(s.memoryDir, "learnings.md")
}

// splitLines splits byte data into lines.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

// DefaultMemoryLimit is the default number of recent experiences to include.
const DefaultMemoryLimit = 10

// DefaultSizeThreshold is the default size in bytes before automatic cleanup triggers.
const DefaultSizeThreshold = 1024 * 1024 // 1MB

// PruneOptions configures the prune operation.
type PruneOptions struct {
	OlderThan        time.Duration // Remove experiences older than this duration
	DryRun           bool          // If true, don't actually delete, just report
	Backup           bool          // If true, create backup before pruning
	IncludeLearnings bool          // If true, also clear learnings (reset to header only)
}

// PruneResult contains statistics from a prune operation.
type PruneResult struct {
	BackupPath        string
	BytesBeforePrune  int64
	BytesAfterPrune   int64
	TotalExperiences  int
	PrunedExperiences int
	PreservedPinned   int
	LearningsCleared  bool // True if learnings were cleared
}

// Prune removes old experiences based on the provided options.
// Pinned experiences are always preserved regardless of age.
func (s *Store) Prune(opts PruneOptions) (*PruneResult, error) {
	result := &PruneResult{}

	// Get current experiences
	experiences, err := s.GetExperiences()
	if err != nil {
		return nil, fmt.Errorf("failed to get experiences: %w", err)
	}
	result.TotalExperiences = len(experiences)

	// Get file size before prune
	if info, statErr := os.Stat(s.experiencesPath()); statErr == nil {
		result.BytesBeforePrune = info.Size()
	}

	// Determine cutoff time
	cutoff := time.Now().Add(-opts.OlderThan)

	// Filter experiences to keep
	var kept []Experience
	for _, exp := range experiences {
		// Always keep pinned experiences
		if exp.Pinned {
			kept = append(kept, exp)
			result.PreservedPinned++
			continue
		}

		// Keep if newer than cutoff
		if exp.Timestamp.After(cutoff) || exp.Timestamp.Equal(cutoff) {
			kept = append(kept, exp)
			continue
		}

		// This experience will be pruned
		result.PrunedExperiences++
	}

	// If dry run, just return the result
	if opts.DryRun {
		return result, nil
	}

	// If nothing to prune, return early
	if result.PrunedExperiences == 0 {
		return result, nil
	}

	// Create backup if requested
	if opts.Backup {
		backupPath, backupErr := s.createBackup()
		if backupErr != nil {
			return nil, fmt.Errorf("failed to create backup: %w", backupErr)
		}
		result.BackupPath = backupPath
	}

	// Write the filtered experiences back
	if err := s.writeExperiences(kept); err != nil {
		return nil, fmt.Errorf("failed to write pruned experiences: %w", err)
	}

	// Get file size after prune
	if info, statErr := os.Stat(s.experiencesPath()); statErr == nil {
		result.BytesAfterPrune = info.Size()
	}

	// Clear learnings if requested
	if opts.IncludeLearnings && !opts.DryRun {
		if err := s.clearLearnings(); err != nil {
			return nil, fmt.Errorf("failed to clear learnings: %w", err)
		}
		result.LearningsCleared = true
	} else if opts.IncludeLearnings && opts.DryRun {
		result.LearningsCleared = true // Would be cleared
	}

	return result, nil
}

// ForgetTopic removes a specific learning topic and all its entries.
// Returns the number of entries removed, or an error if the topic doesn't exist.
func (s *Store) ForgetTopic(topic string) (int, error) {
	content, err := s.GetLearnings()
	if err != nil {
		return 0, fmt.Errorf("failed to read learnings: %w", err)
	}

	topicHeader := "## " + topic
	if !strings.Contains(content, topicHeader) {
		return 0, fmt.Errorf("topic %q not found", topic)
	}

	// Find the topic section
	lines := strings.Split(content, "\n")
	var newLines []string
	inTopic := false
	entriesRemoved := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if this is the target topic header
		if trimmed == topicHeader {
			inTopic = true
			continue
		}

		// Check if we've reached the next topic (end of target topic)
		if inTopic && strings.HasPrefix(trimmed, "## ") {
			inTopic = false
		}

		if inTopic {
			// Count removed entries (lines starting with -)
			if strings.HasPrefix(trimmed, "- ") {
				entriesRemoved++
			}
			continue
		}

		newLines = append(newLines, line)
	}

	// Write the updated content
	newContent := strings.Join(newLines, "\n")
	// Clean up any double blank lines that may result
	for strings.Contains(newContent, "\n\n\n") {
		newContent = strings.ReplaceAll(newContent, "\n\n\n", "\n\n")
	}

	if err := os.WriteFile(s.learningsPath(), []byte(newContent), 0600); err != nil { //nolint:gosec // path constructed from trusted memoryDir
		return 0, fmt.Errorf("failed to write learnings: %w", err)
	}

	return entriesRemoved, nil
}

// ListTopics returns all learning topic names.
func (s *Store) ListTopics() ([]string, error) {
	content, err := s.GetLearnings()
	if err != nil {
		return nil, fmt.Errorf("failed to read learnings: %w", err)
	}

	var topics []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			topic := strings.TrimPrefix(trimmed, "## ")
			topics = append(topics, topic)
		}
	}

	return topics, nil
}

// GetSize returns the total size of memory files in bytes.
func (s *Store) GetSize() (int64, error) {
	var total int64

	paths := []string{s.experiencesPath(), s.learningsPath()}
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil {
			total += info.Size()
		}
	}

	return total, nil
}

// NeedsPruning checks if the memory store exceeds the size threshold.
func (s *Store) NeedsPruning(threshold int64) (bool, int64, error) {
	size, err := s.GetSize()
	if err != nil {
		return false, 0, err
	}
	return size > threshold, size, nil
}

// createBackup creates a timestamped backup of the experiences file.
func (s *Store) createBackup() (string, error) {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := filepath.Join(s.memoryDir, fmt.Sprintf("experiences.%s.backup.jsonl", timestamp))

	src, err := os.ReadFile(s.experiencesPath())
	if err != nil {
		return "", fmt.Errorf("failed to read experiences file: %w", err)
	}

	if err := os.WriteFile(backupPath, src, 0600); err != nil { //nolint:gosec // backup path in trusted memory dir
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

// writeExperiences overwrites the experiences file with the given experiences.
func (s *Store) writeExperiences(experiences []Experience) error {
	f, err := os.Create(s.experiencesPath()) //nolint:gosec // path constructed from trusted memoryDir
	if err != nil {
		return fmt.Errorf("failed to create experiences file: %w", err)
	}
	defer func() { _ = f.Close() }()

	for _, exp := range experiences {
		data, marshalErr := json.Marshal(exp)
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal experience: %w", marshalErr)
		}
		if _, writeErr := f.Write(append(data, '\n')); writeErr != nil {
			return fmt.Errorf("failed to write experience: %w", writeErr)
		}
	}

	return nil
}

// GetMemoryContext returns formatted memories suitable for prompt injection.
// It loads the most recent experiences (up to limit) and all learnings,
// formatting them for inclusion in an agent's context.
// Returns empty string if no memories exist (new agent).
func (s *Store) GetMemoryContext(limit int) (string, error) {
	if limit <= 0 {
		limit = DefaultMemoryLimit
	}

	var parts []string

	// Load experiences (most recent first)
	experiences, err := s.GetExperiences()
	if err != nil {
		return "", fmt.Errorf("failed to load experiences: %w", err)
	}

	if len(experiences) > 0 {
		// Get the most recent experiences
		start := 0
		if len(experiences) > limit {
			start = len(experiences) - limit
		}
		recentExperiences := experiences[start:]

		parts = append(parts, "## Recent Experiences\n")
		for _, exp := range recentExperiences {
			entry := fmt.Sprintf("- **%s** (%s): %s",
				exp.TaskType, exp.Outcome, exp.Description)
			if len(exp.Learnings) > 0 {
				entry += "\n  Learnings: " + exp.Learnings[0]
			}
			parts = append(parts, entry)
		}
		parts = append(parts, "")
	}

	// Load learnings
	learnings, err := s.GetLearnings()
	if err != nil {
		return "", fmt.Errorf("failed to load learnings: %w", err)
	}

	// Only include learnings if there's meaningful content beyond the header
	if learnings != "" && len(learnings) > 100 {
		parts = append(parts, "## Key Learnings\n")
		parts = append(parts, learnings)
	}

	if len(parts) == 0 {
		return "", nil // No memories - new agent
	}

	header := "# Agent Memory\n\nThe following is your accumulated experience and learnings from previous tasks:\n\n"
	return header + strings.Join(parts, "\n"), nil
}
