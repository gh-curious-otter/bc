package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateForHookEvent(t *testing.T) {
	tests := []struct {
		event HookEvent
		want  State
		ok    bool
	}{
		{HookEventPreToolUse, StateWorking, true},
		{HookEventPostToolUse, StateIdle, true},
		{HookEventStop, StateStopped, true},
		{HookPreToolUse, StateWorking, true},
		{HookPostToolUse, StateIdle, true},
		{HookSessionStart, StateIdle, true},
		{HookSessionEnd, StateStopped, true},
		{HookTaskCompleted, StateDone, true},
		{HookPermissionRequest, StateStuck, true},
		{HookEvent("unknown"), "", false},
		{HookEvent(""), "", false},
	}
	for _, tc := range tests {
		got, ok := StateForHookEvent(tc.event)
		if ok != tc.ok {
			t.Errorf("StateForHookEvent(%q) ok=%v, want %v", tc.event, ok, tc.ok)
		}
		if ok && got != tc.want {
			t.Errorf("StateForHookEvent(%q) = %q, want %q", tc.event, got, tc.want)
		}
	}
}

func TestIsKnownEvent(t *testing.T) {
	// State-changing events
	if !IsKnownEvent(HookPreToolUse) {
		t.Error("PreToolUse should be known")
	}
	// Informational events
	if !IsKnownEvent(HookNotification) {
		t.Error("Notification should be known")
	}
	if !IsKnownEvent(HookChannelMessage) {
		t.Error("ChannelMessage should be known")
	}
	// Unknown
	if IsKnownEvent(HookEvent("bogus")) {
		t.Error("bogus should not be known")
	}
}

func TestWriteWorkspaceHookSettings_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	if err := WriteWorkspaceHookSettings(dir); err != nil {
		t.Fatalf("WriteWorkspaceHookSettings: %v", err)
	}
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}
	content := string(data)
	// Should have all 21 valid Claude Code hook events (StopFailure excluded)
	for _, event := range []string{
		"SessionStart", "SessionEnd", "UserPromptSubmit",
		"PreToolUse", "PostToolUse", "PostToolUseFailure",
		"PermissionRequest", "Stop",
		"SubagentStart", "SubagentStop", "TaskCompleted",
		"WorktreeCreate", "WorktreeRemove",
		"PreCompact", "PostCompact",
		"Elicitation", "ElicitationResult",
	} {
		if !strings.Contains(content, `"`+event+`"`) {
			t.Errorf("settings.json missing hook event %q", event)
		}
	}
	// Should use HTTP POST, not file-based
	if !strings.Contains(content, "/api/agents/") {
		t.Error("settings.json should contain HTTP hook URL")
	}
}

func TestWriteWorkspaceHookSettings_Idempotent(t *testing.T) {
	dir := t.TempDir()
	for i := range 3 {
		if err := WriteWorkspaceHookSettings(dir); err != nil {
			t.Fatalf("call %d: WriteWorkspaceHookSettings: %v", i, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json")) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("settings.json not found: %v", err)
	}
	// Each hook event should appear exactly once as a key
	count := strings.Count(string(data), `"PreToolUse"`)
	if count != 1 {
		t.Errorf("PreToolUse appears %d times, want 1", count)
	}
}

func TestWriteWorkspaceHookSettings_MergesExisting(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		t.Fatal(err)
	}
	existing := `{"hooks":{"Notification":[{"hooks":[{"type":"command","command":"echo hi"}]}]}}`
	if err := os.WriteFile(filepath.Join(claudeDir, "settings.json"), []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteWorkspaceHookSettings(dir); err != nil {
		t.Fatalf("WriteWorkspaceHookSettings: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(claudeDir, "settings.json")) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("settings.json not found: %v", err)
	}
	content := string(data)
	// Existing user hook preserved (not overwritten)
	if !strings.Contains(content, "echo hi") {
		t.Error("existing Notification hook was removed during merge")
	}
	if !strings.Contains(content, "PreToolUse") {
		t.Error("PreToolUse hook not added during merge")
	}
}

func TestConsumeHookEvent_NoFile(t *testing.T) {
	dir := t.TempDir()
	ev, ok := ConsumeHookEvent(dir, "alice")
	if ok {
		t.Errorf("expected ok=false, got event=%q", ev)
	}
}

func TestConsumeHookEvent_Valid(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "alice")
	if err := os.MkdirAll(agentDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, hookEventFile), []byte("pre_tool_use"), 0600); err != nil {
		t.Fatal(err)
	}
	ev, ok := ConsumeHookEvent(dir, "alice")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if ev != HookEventPreToolUse {
		t.Errorf("event = %q, want %q", ev, HookEventPreToolUse)
	}
	if _, err := os.Stat(filepath.Join(agentDir, hookEventFile)); err == nil {
		t.Error("hook event file should be deleted after consumption")
	}
}

func TestConsumeHookEvent_Unknown(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "bob")
	if err := os.MkdirAll(agentDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, hookEventFile), []byte("bogus_event"), 0600); err != nil {
		t.Fatal(err)
	}
	_, ok := ConsumeHookEvent(dir, "bob")
	if ok {
		t.Error("expected ok=false for unknown event")
	}
}
