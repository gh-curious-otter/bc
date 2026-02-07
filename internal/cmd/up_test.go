package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rpuneet/bc/pkg/queue"
)

func TestLoadRolePrompt_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	promptDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptDir, 0750); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}

	content := "You are an engineer. Build great software."
	if err := os.WriteFile(filepath.Join(promptDir, "engineer.md"), []byte(content), 0600); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	got := loadRolePrompt(tmpDir, "engineer")
	if got != content {
		t.Errorf("loadRolePrompt() = %q, want %q", got, content)
	}
}

func TestLoadRolePrompt_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()

	got := loadRolePrompt(tmpDir, "nonexistent")
	if got != "" {
		t.Errorf("loadRolePrompt() for missing file = %q, want empty string", got)
	}
}

func TestLoadRolePrompt_MissingDir(t *testing.T) {
	got := loadRolePrompt("/nonexistent/path", "engineer")
	if got != "" {
		t.Errorf("loadRolePrompt() for missing dir = %q, want empty string", got)
	}
}

func TestLoadRolePrompt_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	promptDir := filepath.Join(tmpDir, "prompts")
	if err := os.MkdirAll(promptDir, 0750); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(promptDir, "empty.md"), []byte(""), 0600); err != nil {
		t.Fatalf("failed to write prompt file: %v", err)
	}

	got := loadRolePrompt(tmpDir, "empty")
	if got != "" {
		t.Errorf("loadRolePrompt() for empty file = %q, want empty string", got)
	}
}

func TestBuildBootstrapPrompt_Structure(t *testing.T) {
	agents := []string{"coordinator", "engineer-01", "qa-01"}
	items := []queue.WorkItem{
		{ID: "work-001", Title: "Fix auth bug", BeadsID: "bc-1a.1", Description: "Auth is broken"},
		{ID: "work-002", Title: "Add dark mode", BeadsID: "bc-2b.1"},
	}

	prompt := buildBootstrapPrompt(agents, items, "/test/workspace")

	// Verify workspace info
	if !strings.Contains(prompt, "/test/workspace") {
		t.Error("prompt should contain workspace path")
	}

	// Verify team listing
	if !strings.Contains(prompt, "coordinator, engineer-01, qa-01") {
		t.Error("prompt should contain team listing")
	}

	// Verify work queue header
	if !strings.Contains(prompt, "=== WORK QUEUE ===") {
		t.Error("prompt should contain work queue header")
	}

	// Verify work items
	if !strings.Contains(prompt, "[work-001]") {
		t.Error("prompt should contain work-001 ID")
	}
	if !strings.Contains(prompt, "Fix auth bug") {
		t.Error("prompt should contain work item title")
	}
	if !strings.Contains(prompt, "bc-1a.1") {
		t.Error("prompt should contain beads ID")
	}
	if !strings.Contains(prompt, "Auth is broken") {
		t.Error("prompt should contain description")
	}
	if !strings.Contains(prompt, "[work-002]") {
		t.Error("prompt should contain work-002 ID")
	}

	// Verify workflow sections
	if !strings.Contains(prompt, "Phase 1") {
		t.Error("prompt should contain Phase 1")
	}
	if !strings.Contains(prompt, "Phase 2") {
		t.Error("prompt should contain Phase 2")
	}
	if !strings.Contains(prompt, "Phase 3") {
		t.Error("prompt should contain Phase 3")
	}

	// Verify BC commands section
	if !strings.Contains(prompt, "=== BC COMMANDS ===") {
		t.Error("prompt should contain BC commands section")
	}
	if !strings.Contains(prompt, "bc status") {
		t.Error("prompt should reference bc status command")
	}
	if !strings.Contains(prompt, "bc queue assign") {
		t.Error("prompt should reference bc queue assign command")
	}
}

func TestBuildBootstrapPrompt_EmptyItems(t *testing.T) {
	agents := []string{"coordinator"}
	items := []queue.WorkItem{}

	prompt := buildBootstrapPrompt(agents, items, "/test")

	// Should still have structure even with no items
	if !strings.Contains(prompt, "=== WORK QUEUE ===") {
		t.Error("prompt should contain work queue header even when empty")
	}
	if !strings.Contains(prompt, "=== YOUR WORKFLOW ===") {
		t.Error("prompt should contain workflow section")
	}
}

func TestBuildBootstrapPrompt_ItemWithoutDescription(t *testing.T) {
	agents := []string{"coordinator"}
	items := []queue.WorkItem{
		{ID: "work-001", Title: "Simple task", BeadsID: "bc-1a.1"},
	}

	prompt := buildBootstrapPrompt(agents, items, "/test")

	if !strings.Contains(prompt, "Simple task") {
		t.Error("prompt should contain item title")
	}
}

func TestBuildBootstrapPrompt_CoordinatorRole(t *testing.T) {
	agents := []string{"coordinator"}
	items := []queue.WorkItem{}

	prompt := buildBootstrapPrompt(agents, items, "/test")

	if !strings.Contains(prompt, "You are the coordinator agent") {
		t.Error("prompt should identify the coordinator role")
	}
}

// Flag parsing tests for upCmd

func TestUpCmd_EngineersFlag(t *testing.T) {
	// Reset flag values before test
	upEngineers = 3
	upTechLeads = 2
	upQA = 2
	upWorkers = 0
	upAgent = ""

	// Parse --engineers flag
	if err := upCmd.ParseFlags([]string{"--engineers", "5"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upEngineers != 5 {
		t.Errorf("--engineers flag: got %d, want 5", upEngineers)
	}
}

func TestUpCmd_QAFlag(t *testing.T) {
	// Reset flag values before test
	upEngineers = 3
	upTechLeads = 2
	upQA = 2
	upWorkers = 0
	upAgent = ""

	// Parse --qa flag
	if err := upCmd.ParseFlags([]string{"--qa", "4"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upQA != 4 {
		t.Errorf("--qa flag: got %d, want 4", upQA)
	}
}

func TestUpCmd_TechLeadsFlag(t *testing.T) {
	// Reset flag values before test
	upEngineers = 3
	upTechLeads = 2
	upQA = 2
	upWorkers = 0
	upAgent = ""

	// Parse --tech-leads flag
	if err := upCmd.ParseFlags([]string{"--tech-leads", "3"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upTechLeads != 3 {
		t.Errorf("--tech-leads flag: got %d, want 3", upTechLeads)
	}
}

func TestUpCmd_AgentFlag(t *testing.T) {
	// Reset flag values before test
	upEngineers = 3
	upTechLeads = 2
	upQA = 2
	upWorkers = 0
	upAgent = ""

	// Parse --agent flag
	if err := upCmd.ParseFlags([]string{"--agent", "cursor"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upAgent != "cursor" {
		t.Errorf("--agent flag: got %q, want %q", upAgent, "cursor")
	}
}

func TestUpCmd_WorkersDeprecatedFlag(t *testing.T) {
	// Reset flag values before test
	upEngineers = 3
	upTechLeads = 2
	upQA = 2
	upWorkers = 0
	upAgent = ""

	// Parse deprecated --workers flag
	if err := upCmd.ParseFlags([]string{"--workers", "7"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upWorkers != 7 {
		t.Errorf("--workers flag: got %d, want 7", upWorkers)
	}
}

func TestUpCmd_MultipleFlags(t *testing.T) {
	// Reset flag values before test
	upEngineers = 3
	upTechLeads = 2
	upQA = 2
	upWorkers = 0
	upAgent = ""

	// Parse multiple flags together
	if err := upCmd.ParseFlags([]string{
		"--engineers", "6",
		"--tech-leads", "4",
		"--qa", "3",
		"--agent", "codex",
	}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upEngineers != 6 {
		t.Errorf("--engineers flag: got %d, want 6", upEngineers)
	}
	if upTechLeads != 4 {
		t.Errorf("--tech-leads flag: got %d, want 4", upTechLeads)
	}
	if upQA != 3 {
		t.Errorf("--qa flag: got %d, want 3", upQA)
	}
	if upAgent != "codex" {
		t.Errorf("--agent flag: got %q, want %q", upAgent, "codex")
	}
}

func TestUpCmd_DefaultValues(t *testing.T) {
	// Check that flags have the expected default values
	// Note: We check the flag definitions, not the current values
	// since ParseFlags may have been called earlier in the test suite

	engFlag := upCmd.Flags().Lookup("engineers")
	if engFlag == nil {
		t.Fatal("engineers flag not found")
	}
	if engFlag.DefValue != "3" {
		t.Errorf("engineers default: got %q, want %q", engFlag.DefValue, "3")
	}

	qaFlag := upCmd.Flags().Lookup("qa")
	if qaFlag == nil {
		t.Fatal("qa flag not found")
	}
	if qaFlag.DefValue != "2" {
		t.Errorf("qa default: got %q, want %q", qaFlag.DefValue, "2")
	}

	tlFlag := upCmd.Flags().Lookup("tech-leads")
	if tlFlag == nil {
		t.Fatal("tech-leads flag not found")
	}
	if tlFlag.DefValue != "2" {
		t.Errorf("tech-leads default: got %q, want %q", tlFlag.DefValue, "2")
	}

	agentFlag := upCmd.Flags().Lookup("agent")
	if agentFlag == nil {
		t.Fatal("agent flag not found")
	}
	if agentFlag.DefValue != "" {
		t.Errorf("agent default: got %q, want empty string", agentFlag.DefValue)
	}

	workersFlag := upCmd.Flags().Lookup("workers")
	if workersFlag == nil {
		t.Fatal("workers flag not found")
	}
	if workersFlag.DefValue != "0" {
		t.Errorf("workers default: got %q, want %q", workersFlag.DefValue, "0")
	}
}
