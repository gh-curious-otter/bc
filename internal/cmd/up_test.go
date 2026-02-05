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
	os.MkdirAll(promptDir, 0755)

	content := "You are an engineer. Build great software."
	os.WriteFile(filepath.Join(promptDir, "engineer.md"), []byte(content), 0644)

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
	os.MkdirAll(promptDir, 0755)
	os.WriteFile(filepath.Join(promptDir, "empty.md"), []byte(""), 0644)

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
