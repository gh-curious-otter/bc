package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/rpuneet/bc/pkg/memory"
)

func TestMemoryRecord(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Set agent ID
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	output, err := executeCmd("memory", "record", "--outcome", "success", "Fixed a bug")
	if err != nil {
		t.Fatalf("memory record failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Recorded experience") {
		t.Errorf("expected confirmation message, got: %s", output)
	}

	// Verify experience was recorded
	store := memory.NewStore(wsDir, "test-agent")
	exps, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}
	if len(exps) != 1 {
		t.Fatalf("expected 1 experience, got %d", len(exps))
	}
	if exps[0].Description != "Fixed a bug" {
		t.Errorf("unexpected description: %s", exps[0].Description)
	}
	if exps[0].Outcome != "success" {
		t.Errorf("unexpected outcome: %s", exps[0].Outcome)
	}
}

func TestMemoryLearn(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Set agent ID
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	output, err := executeCmd("memory", "learn", "patterns", "Always check errors")
	if err != nil {
		t.Fatalf("memory learn failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Added learning") {
		t.Errorf("expected confirmation message, got: %s", output)
	}

	// Verify learning was added
	store := memory.NewStore(wsDir, "test-agent")
	learnings, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}
	if !strings.Contains(learnings, "patterns") {
		t.Errorf("learnings should contain category 'patterns': %s", learnings)
	}
	if !strings.Contains(learnings, "Always check errors") {
		t.Errorf("learnings should contain the learning: %s", learnings)
	}
}

func TestMemoryShow(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Set agent ID
	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	// Initialize memory with some content
	store := memory.NewStore(wsDir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "Test experience",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}
	if err := store.AddLearning("tips", "Test tip"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	output, err := executeCmd("memory", "show")
	if err != nil {
		t.Fatalf("memory show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Experiences") {
		t.Errorf("output should contain 'Experiences': %s", output)
	}
	if !strings.Contains(output, "Test experience") {
		t.Errorf("output should contain the experience: %s", output)
	}
	if !strings.Contains(output, "Learnings") {
		t.Errorf("output should contain 'Learnings': %s", output)
	}
}

func TestMemoryShowSpecificAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memory for a specific agent
	store := memory.NewStore(wsDir, "other-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "Other agent experience",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	output, err := executeCmd("memory", "show", "other-agent")
	if err != nil {
		t.Fatalf("memory show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "other-agent") {
		t.Errorf("output should reference the agent: %s", output)
	}
	if !strings.Contains(output, "Other agent experience") {
		t.Errorf("output should contain the experience: %s", output)
	}
}

func TestMemorySearch(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memory with searchable content
	store := memory.NewStore(wsDir, "search-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "Fixed authentication bug",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	output, err := executeCmd("memory", "search", "authentication")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "authentication") {
		t.Errorf("output should contain search result: %s", output)
	}
}

func TestMemorySearchNoResults(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create empty memory
	store := memory.NewStore(wsDir, "empty-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	output, err := executeCmd("memory", "search", "nonexistent")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No results found") {
		t.Errorf("output should indicate no results: %s", output)
	}
}

func TestMemoryRecordRequiresAgentID(t *testing.T) {
	setupTestWorkspace(t)

	// Ensure BC_AGENT_ID is not set
	_ = os.Unsetenv("BC_AGENT_ID")

	_, err := executeCmd("memory", "record", "Test")
	if err == nil {
		t.Error("expected error when BC_AGENT_ID not set")
	}
	if !strings.Contains(err.Error(), "this command can only be run by agents in the bc system") {
		t.Errorf("error should indicate agent-only command: %v", err)
	}
}

func TestMemoryShowNoMemory(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Ensure the memory directory doesn't exist
	memoryDir := filepath.Join(wsDir, ".bc", "memory", "nonexistent-agent")
	_ = os.RemoveAll(memoryDir)

	output, err := executeCmd("memory", "show", "nonexistent-agent")
	if err != nil {
		t.Fatalf("memory show failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "No memory found") {
		t.Errorf("output should indicate no memory: %s", output)
	}
}

func TestMemoryRecordRejectsEmpty(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "record", "")
	if err == nil {
		t.Error("expected error for empty experience")
	}
	if !strings.Contains(err.Error(), "experience cannot be empty") {
		t.Errorf("error should mention empty experience: %v", err)
	}
}

func TestMemoryRecordRejectsWhitespace(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "record", "   ")
	if err == nil {
		t.Error("expected error for whitespace-only experience")
	}
	if !strings.Contains(err.Error(), "experience cannot be empty") {
		t.Errorf("error should mention empty experience: %v", err)
	}
}

func TestMemoryLearnRejectsEmptyCategory(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "learn", "", "some learning")
	if err == nil {
		t.Error("expected error for empty category")
	}
	if !strings.Contains(err.Error(), "category cannot be empty") {
		t.Errorf("error should mention empty category: %v", err)
	}
}

func TestMemoryLearnRejectsEmptyLearning(t *testing.T) {
	setupTestWorkspace(t)

	_ = os.Setenv("BC_AGENT_ID", "test-agent")
	defer func() { _ = os.Unsetenv("BC_AGENT_ID") }()

	_, err := executeCmd("memory", "learn", "patterns", "")
	if err == nil {
		t.Error("expected error for empty learning")
	}
	if !strings.Contains(err.Error(), "learning cannot be empty") {
		t.Errorf("error should mention empty learning: %v", err)
	}
}

func TestMemorySearchRankedResults(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memory with multiple experiences to test ranking
	store := memory.NewStore(wsDir, "ranked-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	// Experience with auth in description (should rank higher)
	if err := store.RecordExperience(memory.Experience{
		Description: "Fixed authentication auth flow",
		Outcome:     "success",
		TaskType:    "bugfix",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	// Experience with auth in task type only (lower rank)
	if err := store.RecordExperience(memory.Experience{
		Description: "Updated login page",
		Outcome:     "success",
		TaskType:    "auth",
	}); err != nil {
		t.Fatalf("failed to record experience: %v", err)
	}

	output, err := executeCmd("memory", "search", "auth")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}

	// Should show count of results
	if !strings.Contains(output, "2 found") {
		t.Errorf("output should show result count: %s", output)
	}

	// Should show score
	if !strings.Contains(output, "score:") {
		t.Errorf("output should show relevance score: %s", output)
	}

	// First result should have higher score (auth in description)
	if !strings.Contains(output, "1. [ranked-agent]") {
		t.Errorf("output should number results: %s", output)
	}
}

func TestMemorySearchMultipleAgents(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memories for two agents
	store1 := memory.NewStore(wsDir, "agent-one")
	if err := store1.Init(); err != nil {
		t.Fatalf("failed to init store1: %v", err)
	}
	if err := store1.RecordExperience(memory.Experience{
		Description: "Fixed auth bug in agent one",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	store2 := memory.NewStore(wsDir, "agent-two")
	if err := store2.Init(); err != nil {
		t.Fatalf("failed to init store2: %v", err)
	}
	if err := store2.RecordExperience(memory.Experience{
		Description: "Auth system redesign",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	output, err := executeCmd("memory", "search", "auth")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}

	// Should find results from both agents
	if !strings.Contains(output, "agent-one") {
		t.Errorf("output should contain agent-one: %s", output)
	}
	if !strings.Contains(output, "agent-two") {
		t.Errorf("output should contain agent-two: %s", output)
	}
}

func TestMemorySearchSpecificAgent(t *testing.T) {
	wsDir := setupTestWorkspace(t)

	// Create memories for two agents
	store1 := memory.NewStore(wsDir, "target-agent")
	if err := store1.Init(); err != nil {
		t.Fatalf("failed to init store1: %v", err)
	}
	if err := store1.RecordExperience(memory.Experience{
		Description: "Fixed bug in target",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	store2 := memory.NewStore(wsDir, "other-agent")
	if err := store2.Init(); err != nil {
		t.Fatalf("failed to init store2: %v", err)
	}
	if err := store2.RecordExperience(memory.Experience{
		Description: "Fixed bug in other",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	// Search only target-agent
	output, err := executeCmd("memory", "search", "--agent", "target-agent", "bug")
	if err != nil {
		t.Fatalf("memory search failed: %v\nOutput: %s", err, output)
	}

	// Should only find target-agent
	if !strings.Contains(output, "target-agent") {
		t.Errorf("output should contain target-agent: %s", output)
	}
	if strings.Contains(output, "other-agent") {
		t.Errorf("output should NOT contain other-agent: %s", output)
	}
}

func TestScoreExperience(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment doesn't matter
		name     string
		exp      memory.Experience
		query    string
		minScore int
	}{
		{
			name: "match in description",
			exp: memory.Experience{
				Description: "Fixed authentication bug",
			},
			query:    "auth",
			minScore: 10,
		},
		{
			name: "match in task type",
			exp: memory.Experience{
				Description: "Updated login",
				TaskType:    "auth",
			},
			query:    "auth",
			minScore: 5,
		},
		{
			name: "match in outcome",
			exp: memory.Experience{
				Description: "Some task",
				Outcome:     "auth failed",
			},
			query:    "auth",
			minScore: 3,
		},
		{
			name: "no match",
			exp: memory.Experience{
				Description: "Updated UI",
			},
			query:    "auth",
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreExperience(tt.exp, tt.query)
			if score < tt.minScore {
				t.Errorf("scoreExperience() = %d, want >= %d", score, tt.minScore)
			}
		})
	}
}

func TestScoreLearning(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		query    string
		minScore int
	}{
		{
			name:     "simple match",
			line:     "Always check authentication",
			query:    "auth",
			minScore: 5,
		},
		{
			name:     "header match",
			line:     "## Authentication Patterns",
			query:    "auth",
			minScore: 8, // 5 + 3 for header
		},
		{
			name:     "no match",
			line:     "Some other content",
			query:    "auth",
			minScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scoreLearning(tt.line, tt.query)
			if score < tt.minScore {
				t.Errorf("scoreLearning() = %d, want >= %d", score, tt.minScore)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantD   string // expected duration as string for comparison
		wantErr bool
	}{
		{
			name:    "30 days",
			input:   "30d",
			wantErr: false,
			wantD:   "720h0m0s", // 30 * 24 hours
		},
		{
			name:    "7 days",
			input:   "7d",
			wantErr: false,
			wantD:   "168h0m0s", // 7 * 24 hours
		},
		{
			name:    "24 hours",
			input:   "24h",
			wantErr: false,
			wantD:   "24h0m0s",
		},
		{
			name:    "1 hour",
			input:   "1h",
			wantErr: false,
			wantD:   "1h0m0s",
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, err := parseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if d.String() != tt.wantD {
				t.Errorf("parseDuration(%q) = %s, want %s", tt.input, d.String(), tt.wantD)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct { //nolint:govet // test struct alignment doesn't matter
		bytes int64
		name  string
		want  string
	}{
		{0, "zero", "0 B"},
		{500, "bytes", "500 B"},
		{1500, "kilobytes", "1.5 KB"},
		{1500000, "megabytes", "1.4 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestMemoryPruneCmdExists(t *testing.T) {
	if memoryPruneCmd == nil {
		t.Fatal("memoryPruneCmd should not be nil")
	}
	if memoryPruneCmd.Use != "prune" {
		t.Errorf("memoryPruneCmd.Use = %q, want %q", memoryPruneCmd.Use, "prune")
	}
}

func TestMemoryPruneFlags(t *testing.T) {
	// Check older-than flag
	flag := memoryPruneCmd.Flags().Lookup("older-than")
	if flag == nil {
		t.Fatal("expected 'older-than' flag to exist")
	}
	if flag.DefValue != "30d" {
		t.Errorf("older-than default = %q, want %q", flag.DefValue, "30d")
	}

	// Check dry-run flag
	flag = memoryPruneCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("expected 'dry-run' flag to exist")
	}

	// Check no-backup flag
	flag = memoryPruneCmd.Flags().Lookup("no-backup")
	if flag == nil {
		t.Fatal("expected 'no-backup' flag to exist")
	}

	// Check agent flag
	flag = memoryPruneCmd.Flags().Lookup("agent")
	if flag == nil {
		t.Fatal("expected 'agent' flag to exist")
	}
}

func TestMemoryImportCmdExists(t *testing.T) {
	if memoryImportCmd == nil {
		t.Fatal("memoryImportCmd should not be nil")
	}
	if memoryImportCmd.Use != "import <agent> <file>" {
		t.Errorf("memoryImportCmd.Use = %q, want %q", memoryImportCmd.Use, "import <agent> <file>")
	}
}

func TestMemoryImportFlags(t *testing.T) {
	// Check replace flag
	flag := memoryImportCmd.Flags().Lookup("replace")
	if flag == nil {
		t.Fatal("expected 'replace' flag to exist")
	}

	// Check dry-run flag
	flag = memoryImportCmd.Flags().Lookup("dry-run")
	if flag == nil {
		t.Fatal("expected 'dry-run' flag to exist")
	}
}

func TestMemoryImport(t *testing.T) {
	rootDir := setupTestWorkspace(t)

	// Initialize memory for an agent
	store := memory.NewStore(rootDir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init memory: %v", err)
	}

	// Create import file
	importData := `{
		"experiences": [
			{"description": "test experience", "outcome": "success"}
		],
		"learnings": {
			"patterns": ["learning 1", "learning 2"]
		}
	}`
	importFile := filepath.Join(rootDir, "import.json")
	if err := os.WriteFile(importFile, []byte(importData), 0600); err != nil {
		t.Fatalf("failed to write import file: %v", err)
	}

	// Run import
	output, err := executeCmd("memory", "import", "test-agent", importFile)
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify output
	if !strings.Contains(output, "Experiences: 1") {
		t.Errorf("expected 'Experiences: 1' in output, got: %s", output)
	}
	if !strings.Contains(output, "Learnings: 2") {
		t.Errorf("expected 'Learnings: 2' in output, got: %s", output)
	}

	// Verify imported data
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}
	if len(experiences) != 1 {
		t.Errorf("expected 1 experience, got %d", len(experiences))
	}
}

func TestMemoryImportDryRun(t *testing.T) {
	rootDir := setupTestWorkspace(t)

	// Initialize memory for an agent
	store := memory.NewStore(rootDir, "test-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init memory: %v", err)
	}

	// Create import file
	importData := `{
		"experiences": [
			{"description": "test experience", "outcome": "success"}
		]
	}`
	importFile := filepath.Join(rootDir, "import.json")
	if err := os.WriteFile(importFile, []byte(importData), 0600); err != nil {
		t.Fatalf("failed to write import file: %v", err)
	}

	// Run import with dry-run
	output, err := executeCmd("memory", "import", "test-agent", importFile, "--dry-run")
	if err != nil {
		t.Fatalf("import command failed: %v", err)
	}

	// Verify output shows dry run
	if !strings.Contains(output, "Dry Run") {
		t.Errorf("expected 'Dry Run' in output, got: %s", output)
	}

	// Verify no data was imported
	experiences, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}
	if len(experiences) != 0 {
		t.Errorf("expected 0 experiences (dry run), got %d", len(experiences))
	}
}

func TestMemoryImportFileNotFound(t *testing.T) {
	_ = setupTestWorkspace(t)

	// Run import with non-existent file
	_, err := executeCmd("memory", "import", "test-agent", "/nonexistent/file.json")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

// resetMemoryEditDeleteFlags resets package-level flag vars and Cobra flag state
// that leak between tests. executeCmd only resets flags on rootCmd's direct
// children, not grandchildren like memoryEditCmd and memoryDeleteCmd.
func resetMemoryEditDeleteFlags(t *testing.T) {
	t.Helper()
	memoryEditLearnings = false
	memoryEditExperiences = false
	memoryEditRolePrompt = false
	memoryDeleteExperience = 0
	memoryDeleteLearning = ""
	memoryDeleteForce = false

	// Reset Cobra flag Changed state on grandchild commands
	for _, sub := range memoryCmd.Commands() {
		sub.Flags().VisitAll(func(f *pflag.Flag) {
			f.Changed = false
			if f.Value.Type() == "bool" {
				_ = f.Value.Set("false")
			} else if f.Value.Type() == "int" {
				_ = f.Value.Set("0")
			} else if f.Value.Type() == "string" {
				_ = f.Value.Set(f.DefValue)
			}
		})
	}
}

// --- Memory Edit Tests ---

func TestMemoryEditResolvesLearningsPath(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "edit-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_ = os.Setenv("EDITOR", "true")
	defer func() { _ = os.Unsetenv("EDITOR") }()

	output, err := executeCmd("memory", "edit", "edit-agent", "--learnings")
	if err != nil {
		t.Fatalf("memory edit --learnings failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "learnings.md") {
		t.Errorf("expected output to mention learnings.md, got: %s", output)
	}
}

func TestMemoryEditResolvesExperiencesPath(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "edit-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_ = os.Setenv("EDITOR", "true")
	defer func() { _ = os.Unsetenv("EDITOR") }()

	output, err := executeCmd("memory", "edit", "edit-agent", "--experiences")
	if err != nil {
		t.Fatalf("memory edit --experiences failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "experiences.jsonl") {
		t.Errorf("expected output to mention experiences.jsonl, got: %s", output)
	}
}

func TestMemoryEditResolvesRolePromptPath(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "edit-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	rpPath := filepath.Join(store.MemoryDir(), "role_prompt.md")
	if err := os.WriteFile(rpPath, []byte("# Role\n"), 0600); err != nil {
		t.Fatalf("failed to create role_prompt.md: %v", err)
	}

	_ = os.Setenv("EDITOR", "true")
	defer func() { _ = os.Unsetenv("EDITOR") }()

	output, err := executeCmd("memory", "edit", "edit-agent", "--role-prompt")
	if err != nil {
		t.Fatalf("memory edit --role-prompt failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "role_prompt.md") {
		t.Errorf("expected output to mention role_prompt.md, got: %s", output)
	}
}

func TestMemoryEditRequiresFlag(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "edit-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_, err := executeCmd("memory", "edit", "edit-agent")
	if err == nil {
		t.Error("expected error when no file flag specified")
	}
	if !strings.Contains(err.Error(), "specify which file") {
		t.Errorf("error should mention specifying a file flag: %v", err)
	}
}

func TestMemoryEditMissingAgent(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	setupTestWorkspace(t)

	_ = os.Setenv("EDITOR", "true")
	defer func() { _ = os.Unsetenv("EDITOR") }()

	_, err := executeCmd("memory", "edit", "nonexistent-agent", "--learnings")
	if err == nil {
		t.Error("expected error for missing agent")
	}
	if !strings.Contains(err.Error(), "no memory found") {
		t.Errorf("error should mention no memory found: %v", err)
	}
}

func TestMemoryEditEditorFallback(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "edit-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_ = os.Setenv("EDITOR", "true")
	defer func() { _ = os.Unsetenv("EDITOR") }()

	output, err := executeCmd("memory", "edit", "edit-agent", "--learnings")
	if err != nil {
		t.Fatalf("memory edit failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Edited") {
		t.Errorf("expected confirmation message, got: %s", output)
	}
}

// --- Memory Delete Tests ---

func TestMemoryDeleteExperience(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	for _, desc := range []string{"first task", "second task", "third task"} {
		if err := store.RecordExperience(memory.Experience{
			Description: desc,
			Outcome:     "success",
		}); err != nil {
			t.Fatalf("failed to record: %v", err)
		}
	}

	output, err := executeCmd("memory", "delete", "del-agent", "--experience", "2", "--force")
	if err != nil {
		t.Fatalf("delete experience failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Deleted experience #2") {
		t.Errorf("expected deletion confirmation, got: %s", output)
	}
	if !strings.Contains(output, "second task") {
		t.Errorf("expected deleted item description, got: %s", output)
	}

	exps, err := store.GetExperiences()
	if err != nil {
		t.Fatalf("failed to get experiences: %v", err)
	}
	if len(exps) != 2 {
		t.Fatalf("expected 2 experiences, got %d", len(exps))
	}
	if exps[0].Description != "first task" {
		t.Errorf("expected 'first task', got %q", exps[0].Description)
	}
	if exps[1].Description != "third task" {
		t.Errorf("expected 'third task', got %q", exps[1].Description)
	}
}

func TestMemoryDeleteExperienceInvalidIndex(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.RecordExperience(memory.Experience{
		Description: "only task",
		Outcome:     "success",
	}); err != nil {
		t.Fatalf("failed to record: %v", err)
	}

	_, err := executeCmd("memory", "delete", "del-agent", "--experience", "5", "--force")
	if err == nil {
		t.Error("expected error for out of range index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention out of range: %v", err)
	}
}

func TestMemoryDeleteLearning(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	for _, l := range []string{"first insight", "second insight", "third insight"} {
		if err := store.AddLearning("patterns", l); err != nil {
			t.Fatalf("failed to add learning: %v", err)
		}
	}

	output, err := executeCmd("memory", "delete", "del-agent", "--learning", "patterns", "2", "--force")
	if err != nil {
		t.Fatalf("delete learning failed: %v\nOutput: %s", err, output)
	}
	if !strings.Contains(output, "Deleted learning #2") {
		t.Errorf("expected deletion confirmation, got: %s", output)
	}
	if !strings.Contains(output, "patterns") {
		t.Errorf("expected category name in output, got: %s", output)
	}

	learnings, err := store.GetLearnings()
	if err != nil {
		t.Fatalf("failed to get learnings: %v", err)
	}
	count := strings.Count(learnings, "- ")
	if count != 2 {
		t.Errorf("expected 2 learning entries, got %d", count)
	}
}

func TestMemoryDeleteLearningInvalidIndex(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.AddLearning("patterns", "only one"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	_, err := executeCmd("memory", "delete", "del-agent", "--learning", "patterns", "5", "--force")
	if err == nil {
		t.Error("expected error for out of range index")
	}
	if !strings.Contains(err.Error(), "out of range") {
		t.Errorf("error should mention out of range: %v", err)
	}
}

func TestMemoryDeleteLearningMissingCategory(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	if err := store.AddLearning("tips", "something"); err != nil {
		t.Fatalf("failed to add learning: %v", err)
	}

	_, err := executeCmd("memory", "delete", "del-agent", "--learning", "nonexistent", "1", "--force")
	if err == nil {
		t.Error("expected error for missing category")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention not found: %v", err)
	}
}

func TestMemoryDeleteLearningMissingIndex(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_, err := executeCmd("memory", "delete", "del-agent", "--learning", "patterns")
	if err == nil {
		t.Error("expected error for missing index")
	}
	if !strings.Contains(err.Error(), "missing index") {
		t.Errorf("error should mention missing index: %v", err)
	}
}

func TestMemoryDeleteRequiresMode(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	wsDir := setupTestWorkspace(t)

	store := memory.NewStore(wsDir, "del-agent")
	if err := store.Init(); err != nil {
		t.Fatalf("failed to init store: %v", err)
	}

	_, err := executeCmd("memory", "delete", "del-agent")
	if err == nil {
		t.Error("expected error when no mode flag specified")
	}
	if !strings.Contains(err.Error(), "specify what to delete") {
		t.Errorf("error should mention specifying mode: %v", err)
	}
}

func TestMemoryDeleteMissingAgent(t *testing.T) {
	resetMemoryEditDeleteFlags(t)
	setupTestWorkspace(t)

	_, err := executeCmd("memory", "delete", "nonexistent", "--experience", "1", "--force")
	if err == nil {
		t.Error("expected error for missing agent")
	}
	if !strings.Contains(err.Error(), "no memory found") {
		t.Errorf("error should mention no memory found: %v", err)
	}
}
