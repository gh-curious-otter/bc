package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// HookEvent is a Claude Code lifecycle hook event type.
type HookEvent string

const (
	// HookEventPreToolUse fires before each tool call — agent is actively working.
	HookEventPreToolUse HookEvent = "pre_tool_use"
	// HookEventPostToolUse fires after each tool call — agent may be idle.
	HookEventPostToolUse HookEvent = "post_tool_use"
	// HookEventStop fires when Claude Code exits — agent is stopped.
	HookEventStop HookEvent = "stop"
)

// hookEventStateMap maps hook events to the target agent state.
var hookEventStateMap = map[HookEvent]State{
	HookEventPreToolUse:  StateWorking,
	HookEventPostToolUse: StateIdle,
	HookEventStop:        StateStopped,
}

// StateForHookEvent returns the target agent State for a hook event.
// Returns false if the event is unknown.
func StateForHookEvent(ev HookEvent) (State, bool) {
	s, ok := hookEventStateMap[ev]
	return s, ok
}

// hookEventFile is the filename written by hook scripts to signal a state event.
const hookEventFile = "hook_event"

// hookEventPath returns the path of the hook event file for an agent.
func hookEventPath(stateDir, agentName string) string {
	return filepath.Join(stateDir, agentName, hookEventFile)
}

// claudeSettings is the minimal shape of .claude/settings.json used by Claude Code.
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

// WriteWorkspaceHookSettings writes .claude/settings.json to the workspace root
// so Claude Code agents automatically emit state events to bcd.
//
// The hook scripts write to .bc/agents/$BC_AGENT_ID/hook_event (file-based, works
// in both tmux and Docker without network configuration). The StatsCollector in
// bcd consumes those files on each poll cycle.
//
// This is idempotent: if settings.json already exists the hooks section is merged.
func WriteWorkspaceHookSettings(workspaceRoot string) error {
	claudeDir := filepath.Join(workspaceRoot, ".claude")
	if err := os.MkdirAll(claudeDir, 0750); err != nil {
		return fmt.Errorf("create .claude dir: %w", err)
	}

	// Each hook writes the event name into .bc/agents/$BC_AGENT_ID/hook_event.
	// Using printf avoids a newline; the file is consumed and deleted by bcd.
	hookCmd := func(event HookEvent) string {
		return fmt.Sprintf(
			`printf '%%s' %q > "${BC_WORKSPACE}/.bc/agents/${BC_AGENT_ID}/%s" 2>/dev/null || true`,
			string(event), hookEventFile,
		)
	}

	settings := claudeSettings{
		Hooks: map[string][]claudeHookMatcher{
			"PreToolUse": {{
				Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookEventPreToolUse)}},
			}},
			"PostToolUse": {{
				Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookEventPostToolUse)}},
			}},
			"Stop": {{
				Hooks: []claudeHook{{Type: "command", Command: hookCmd(HookEventStop)}},
			}},
		},
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Merge if file already exists so we don't clobber user customisations.
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

// loadClaudeSettings reads an existing .claude/settings.json.
func loadClaudeSettings(path string) (*claudeSettings, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is workspace-relative, caller-controlled
	if err != nil {
		return nil, err
	}
	var s claudeSettings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// mergeHooks adds bc hook entries to an existing settings struct,
// preserving any existing hooks the user has set.
func mergeHooks(dst *claudeSettings, src map[string][]claudeHookMatcher) {
	if dst.Hooks == nil {
		dst.Hooks = make(map[string][]claudeHookMatcher)
	}
	for event, matchers := range src {
		if _, exists := dst.Hooks[event]; !exists {
			dst.Hooks[event] = matchers
		}
		// If the event already has matchers, skip to avoid duplicating bc hooks.
	}
}

// ConsumeHookEvent reads and removes the hook event file for an agent,
// returning the event if one was written. Returns "" if no event is pending.
func ConsumeHookEvent(stateDir, agentName string) (HookEvent, bool) {
	path := hookEventPath(stateDir, agentName)
	data, err := os.ReadFile(path) //nolint:gosec // path is internal state dir
	if err != nil {
		return "", false
	}
	_ = os.Remove(path) //nolint:errcheck // best-effort
	ev := HookEvent(data)
	_, ok := hookEventStateMap[ev]
	return ev, ok
}
