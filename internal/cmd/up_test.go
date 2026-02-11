package cmd

import (
	"strings"
	"testing"
)

func TestBuildBootstrapPrompt_Structure(t *testing.T) {
	prompt := buildBootstrapPrompt("/test/workspace")

	// Verify workspace info
	if !strings.Contains(prompt, "/test/workspace") {
		t.Error("prompt should contain workspace path")
	}

	// Verify root identity
	if !strings.Contains(prompt, "ROOT: ORCHESTRATOR & SYSTEM AGENT") {
		t.Error("prompt should identify root agent")
	}

	// Verify responsibilities
	if !strings.Contains(prompt, "=== ROOT RESPONSIBILITIES ===") {
		t.Error("prompt should contain responsibilities section")
	}
	if !strings.Contains(prompt, "System Health") {
		t.Error("prompt should mention System Health")
	}

	// Verify BC commands section
	if !strings.Contains(prompt, "=== BC COMMANDS ===") {
		t.Error("prompt should contain BC commands section")
	}
	// Check for key command categories
	if !strings.Contains(prompt, "** Agent Operations **") {
		t.Error("prompt should contain Agent Operations")
	}
	if !strings.Contains(prompt, "** Configuration **") {
		t.Error("prompt should contain Configuration")
	}
	if !strings.Contains(prompt, "** Role Management **") {
		t.Error("prompt should contain Role Management")
	}

	// Check for specific new commands
	if !strings.Contains(prompt, "bc config show") {
		t.Error("prompt should reference bc config commands")
	}
	if !strings.Contains(prompt, "bc role list") {
		t.Error("prompt should reference bc role commands")
	}
}

func TestUpCmd_AgentFlag(t *testing.T) {
	// Reset flag values before test
	upAgent = ""

	// Parse --agent flag
	if err := upCmd.ParseFlags([]string{"--agent", "cursor"}); err != nil {
		t.Fatalf("ParseFlags failed: %v", err)
	}

	if upAgent != "cursor" {
		t.Errorf("--agent flag: got %q, want %q", upAgent, "cursor")
	}
}

func TestUpCmd_DefaultValues(t *testing.T) {
	agentFlag := upCmd.Flags().Lookup("agent")
	if agentFlag == nil {
		t.Fatal("agent flag not found")
	}
	if agentFlag.DefValue != "" {
		t.Errorf("agent default: got %q, want empty string", agentFlag.DefValue)
	}
}
