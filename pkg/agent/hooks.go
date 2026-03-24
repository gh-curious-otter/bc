package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// HookEvent is a lifecycle event type — either a Claude Code hook or a bc-internal event.
type HookEvent string

// ── Claude Code hook events (configured in .claude/settings.json) ──

const (
	HookSessionStart       HookEvent = "SessionStart"
	HookSessionEnd         HookEvent = "SessionEnd"
	HookUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookPreToolUse         HookEvent = "PreToolUse"
	HookPostToolUse        HookEvent = "PostToolUse"
	HookPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookPermissionRequest  HookEvent = "PermissionRequest"
	HookStop               HookEvent = "Stop"
	HookStopFailure        HookEvent = "StopFailure"
	HookNotification       HookEvent = "Notification"
	HookSubagentStart      HookEvent = "SubagentStart"
	HookSubagentStop       HookEvent = "SubagentStop"
	HookTaskCompleted      HookEvent = "TaskCompleted"
	HookTeammateIdle       HookEvent = "TeammateIdle"
	HookInstructionsLoaded HookEvent = "InstructionsLoaded"
	HookConfigChange       HookEvent = "ConfigChange"
	HookWorktreeCreate     HookEvent = "WorktreeCreate"
	HookWorktreeRemove     HookEvent = "WorktreeRemove"
	HookPreCompact         HookEvent = "PreCompact"
	HookPostCompact        HookEvent = "PostCompact"
	HookElicitation        HookEvent = "Elicitation"
	HookElicitationResult  HookEvent = "ElicitationResult"
)

// ── bc-internal events (POSTed by bcd Go code, not Claude Code hooks) ──

const (
	HookChannelMessage HookEvent = "ChannelMessage"
	HookChannelSent    HookEvent = "ChannelSent"
	HookAgentMessage   HookEvent = "AgentMessage"
	HookCostUpdate     HookEvent = "CostUpdate"
)

// hookEventStateMap maps hook events to the target agent state.
// Events not in this map don't change agent state (they're informational).
var hookEventStateMap = map[HookEvent]State{
	HookSessionStart:       StateIdle,
	HookSessionEnd:         StateStopped,
	HookUserPromptSubmit:   StateWorking,
	HookPreToolUse:         StateWorking,
	HookPostToolUse:        StateIdle,
	HookPostToolUseFailure: StateWorking,
	HookPermissionRequest:  StateStuck,
	HookStop:               StateIdle, // turn complete, not session end
	HookStopFailure:        StateError,
	HookSubagentStart:      StateWorking,
	HookSubagentStop:       StateWorking,
	HookTaskCompleted:      StateDone,
	HookWorktreeCreate:     StateStarting,
	HookPreCompact:         StateWorking,
	HookPostCompact:        StateWorking,
	HookElicitation:        StateStuck,
	HookElicitationResult:  StateWorking,
}

// StateForHookEvent returns the target agent State for a hook event.
// Returns false if the event doesn't trigger a state change (informational events).
func StateForHookEvent(ev HookEvent) (State, bool) {
	s, ok := hookEventStateMap[ev]
	return s, ok
}

// IsKnownEvent returns true if the event type is recognized (even if informational).
func IsKnownEvent(ev HookEvent) bool {
	if _, ok := hookEventStateMap[ev]; ok {
		return true
	}
	// Informational events that don't change state
	switch ev {
	case HookNotification, HookTeammateIdle, HookInstructionsLoaded,
		HookConfigChange, HookWorktreeRemove,
		HookChannelMessage, HookChannelSent, HookAgentMessage, HookCostUpdate:
		return true
	}
	return false
}

// HookPayload is the JSON payload received by the /hook endpoint.
// Different events populate different fields.
type HookPayload struct { //nolint:govet // fieldalignment: optimized for readability over 8 bytes
	// Tool events (PreToolUse, PostToolUse, PostToolUseFailure)
	ToolInput any `json:"tool_input,omitempty"` // full tool input object

	// Channel events (bc-internal)
	Mentions []string `json:"mentions,omitempty"`

	// Common fields
	Event HookEvent `json:"event"`
	State string    `json:"state,omitempty"` // override state (optional)
	Task  string    `json:"task,omitempty"`  // task description for UI

	// Tool events
	ToolName string `json:"tool_name,omitempty"`
	Command  string `json:"command,omitempty"` // bash command being run
	Error    string `json:"error,omitempty"`   // error message (failures)

	// Subagent events
	SubagentID   string `json:"subagent_id,omitempty"`
	SubagentType string `json:"subagent_type,omitempty"`

	// Channel events (bc-internal)
	Channel string `json:"channel,omitempty"`
	Sender  string `json:"sender,omitempty"`
	Message string `json:"message,omitempty"`

	// Config/Instructions events
	File string `json:"file,omitempty"`

	// Cost events (bc-internal)
	Model string `json:"model,omitempty"`

	// Non-pointer fields last to minimize GC scan area
	CostUSD      float64 `json:"cost_usd,omitempty"`
	InputTokens  int64   `json:"input_tokens,omitempty"`
	OutputTokens int64   `json:"output_tokens,omitempty"`
}

// ── Settings.json writer (generates HTTP-based hooks) ──

type claudeSettings struct {
	Hooks map[string][]claudeHookMatcher `json:"hooks,omitempty"`
}

type claudeHookMatcher struct {
	Matcher string       `json:"matcher,omitempty"`
	Hooks   []claudeHook `json:"hooks"`
}

type claudeHook struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// WriteWorkspaceHookSettings writes .claude/settings.json with HTTP-based hooks
// that POST to bcd's /api/agents/{name}/hook endpoint for instant status updates.
//
// Uses curl to POST JSON payloads. Tool-aware hooks read stdin JSON via jq.
// This is idempotent: if settings.json already exists the hooks section is merged.
func WriteWorkspaceHookSettings(workspaceRoot string) error {
	claudeDir := filepath.Join(workspaceRoot, ".claude")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	// Hook commands use $BC_BCD_ADDR env var (set per-agent based on runtime).
	// Falls back to localhost for backward compat.
	bcdAddr := "${BC_BCD_ADDR:-http://127.0.0.1:9374}"

	// Simple hook command (no stdin parsing)
	hookCmd := func(event HookEvent, stateTarget State, taskDesc string) string {
		payload := fmt.Sprintf(`{"event":"%s","state":"%s","task":"%s"}`, event, stateTarget, taskDesc)
		return fmt.Sprintf(
			`curl -sX POST %s/api/agents/${BC_AGENT_ID}/hook -H 'Content-Type: application/json' -d '%s' 2>/dev/null || true`,
			bcdAddr, payload,
		)
	}

	// Tool-aware hook command (reads tool_name from stdin via jq)
	toolHookCmd := func(event HookEvent, stateTarget State, taskPrefix string) string {
		return fmt.Sprintf(
			`bash -c 'HOOK_INPUT=$(cat); PAYLOAD=$(echo "$HOOK_INPUT" | jq -c "{event:\"%s\",state:\"%s\",tool_name:.tool_name,task:(\"%s: \"+.tool_name),command:.tool_input.command}"); curl -sX POST %s/api/agents/${BC_AGENT_ID}/hook -H "Content-Type: application/json" -d "$PAYLOAD" 2>/dev/null || true'`,
			event, stateTarget, taskPrefix, bcdAddr,
		)
	}

	settings := claudeSettings{
		Hooks: map[string][]claudeHookMatcher{
			"SessionStart":       {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookSessionStart, StateIdle, "Session started")}}}},
			"SessionEnd":         {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookSessionEnd, StateStopped, "Session ended")}}}},
			"UserPromptSubmit":   {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookUserPromptSubmit, StateWorking, "Processing prompt...")}}}},
			"PreToolUse":         {{Hooks: []claudeHook{{Type: "command", Command: toolHookCmd(HookPreToolUse, StateWorking, "Running")}}}},
			"PostToolUse":        {{Hooks: []claudeHook{{Type: "command", Command: toolHookCmd(HookPostToolUse, StateIdle, "Done")}}}},
			"PostToolUseFailure": {{Hooks: []claudeHook{{Type: "command", Command: toolHookCmd(HookPostToolUseFailure, StateWorking, "Failed")}}}},
			"PermissionRequest":  {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookPermissionRequest, StateStuck, "Waiting for permission")}}}},
			"Stop":               {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookStop, StateIdle, "Turn complete")}}}},
			"Notification":       {{Hooks: []claudeHook{{Type: "command", Command: hookCmd("Notification", "", "")}}}},
			"SubagentStart": {{Hooks: []claudeHook{{Type: "command", Command: fmt.Sprintf(
				`bash -c 'BCD=%s; HOOK_INPUT=$(cat); PAYLOAD=$(echo "$HOOK_INPUT" | jq -c "{event:\"SubagentStart\",state:\"working\",task:(\"Subagent: \"+(.agent_type // \"unknown\")),subagent_id:.agent_id,subagent_type:.agent_type}"); curl -sX POST $BCD/api/agents/${BC_AGENT_ID}/hook -H "Content-Type: application/json" -d "$PAYLOAD" 2>/dev/null || true'`,
				bcdAddr,
			)}}}},
			"SubagentStop":       {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookSubagentStop, StateWorking, "Subagent completed")}}}},
			"TaskCompleted":      {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookTaskCompleted, StateDone, "Task completed")}}}},
			"TeammateIdle":       {{Hooks: []claudeHook{{Type: "command", Command: hookCmd("TeammateIdle", "", "")}}}},
			"InstructionsLoaded": {{Hooks: []claudeHook{{Type: "command", Command: hookCmd("InstructionsLoaded", "", "")}}}},
			"ConfigChange":       {{Hooks: []claudeHook{{Type: "command", Command: hookCmd("ConfigChange", "", "")}}}},
			"WorktreeCreate":     {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookWorktreeCreate, StateStarting, "Creating worktree")}}}},
			"WorktreeRemove":     {{Hooks: []claudeHook{{Type: "command", Command: hookCmd("WorktreeRemove", "", "")}}}},
			"PreCompact":         {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookPreCompact, StateWorking, "Compacting context...")}}}},
			"PostCompact":        {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookPostCompact, StateWorking, "Context compacted")}}}},
			"Elicitation":        {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookElicitation, StateStuck, "MCP input needed")}}}},
			"ElicitationResult":  {{Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookElicitationResult, StateWorking, "MCP input received")}}}},
		},
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Merge if file already exists so we don't clobber user customizations.
	if existing, err := loadClaudeSettings(settingsPath); err == nil {
		mergeHooks(existing, settings.Hooks)
		data, marshalErr := json.MarshalIndent(existing, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("marshal hook settings: %w", marshalErr)
		}
		return os.WriteFile(settingsPath, data, 0600)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hook settings: %w", err)
	}
	return os.WriteFile(settingsPath, data, 0600)
}

func loadClaudeSettings(path string) (*claudeSettings, error) {
	data, err := os.ReadFile(path) //nolint:gosec // workspace-relative
	if err != nil {
		return nil, err
	}
	var s claudeSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// invalidHookKeys are Claude Code hook event names that bc has generated in the
// past but are not actually valid. They must be actively removed from existing
// settings files to prevent Claude from rejecting the entire settings file.
var invalidHookKeys = []string{"StopFailure"}

func mergeHooks(dst *claudeSettings, src map[string][]claudeHookMatcher) {
	if dst.Hooks == nil {
		dst.Hooks = make(map[string][]claudeHookMatcher)
	}
	// Remove known-invalid keys that may exist from prior bc versions.
	for _, bad := range invalidHookKeys {
		delete(dst.Hooks, bad)
	}
	// Overwrite all bc-managed hooks so URL/env changes propagate.
	for event, matchers := range src {
		dst.Hooks[event] = matchers
	}
}
