package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rpuneet/bc/pkg/log"
)

// AgentAuthDir returns the per-agent .claude settings directory.
// Mounted as /home/agent/.claude inside the container and persists across
// restarts. Each agent has isolated credentials.
//
// Layout: <workspaceDir>/.bc/agents/<agentName>/auth/.claude/
func AgentAuthDir(workspaceDir, agentName string) string {
	return filepath.Join(workspaceDir, ".bc", "agents", agentName, "auth", ".claude")
}

// AgentAuthTokenFile returns the path of the per-agent .claude.json token file.
// Layout: <workspaceDir>/.bc/agents/<agentName>/auth/.claude.json
func AgentAuthTokenFile(workspaceDir, agentName string) string {
	return filepath.Join(workspaceDir, ".bc", "agents", agentName, "auth", ".claude.json")
}

// EnsureAuthDir creates the per-agent auth directories.
// Docker agents do NOT inherit host credentials — they start fresh.
// Auth is handled via ANTHROPIC_API_KEY env var or interactive login
// inside the container.
func EnsureAuthDir(workspaceDir, agentName string) (string, error) {
	workspaceName := filepath.Base(workspaceDir)
	dir := AgentAuthDir(workspaceDir, agentName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create agent auth dir: %w", err)
	}

	// Write default settings so agents skip interactive prompts
	settingsPath := filepath.Join(dir, "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		writeDefaultSettings(settingsPath)
	}

	// Pre-create project trust directories so workspace trust prompt is skipped
	ensureProjectTrust(dir, workspaceName, agentName)

	return dir, nil
}

// writeDefaultSettings creates a settings.json that pre-configures Claude Code
// so agents skip interactive prompts and start working immediately.
func writeDefaultSettings(path string) {
	settings := map[string]any{
		"theme":                            "dark",
		"skipDangerousModePermissionPrompt": true,
		"autoUpdaterStatus":                "disabled",
		"verbose":                          false,
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	if writeErr := os.WriteFile(path, data, 0600); writeErr != nil {
		log.Debug("failed to write default settings", "error", writeErr)
	}
}

// ensureProjectTrust pre-creates Claude's per-project directory so the workspace
// trust prompt is skipped on agent start.
func ensureProjectTrust(claudeDir, workspaceName, agentName string) {
	projectsDir := filepath.Join(claudeDir, "projects")
	worktreeName := "bc-" + workspaceName + "-" + agentName
	encodedPath := "-workspace--claude-worktrees-" + worktreeName

	for _, p := range []string{"-workspace", encodedPath} {
		dir := filepath.Join(projectsDir, p)
		_ = os.MkdirAll(dir, 0700) //nolint:errcheck // best-effort
	}
}
