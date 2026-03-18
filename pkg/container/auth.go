package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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
// This is the primary auth token Claude Code reads at startup ($HOME/.claude.json).
// It lives alongside (not inside) the .claude/ dir so it can be bind-mounted
// individually as /home/agent/.claude.json without touching the rest of HOME.
//
// Layout: <workspaceDir>/.bc/agents/<agentName>/auth/.claude.json
func AgentAuthTokenFile(workspaceDir, agentName string) string {
	return filepath.Join(workspaceDir, ".bc", "agents", agentName, "auth", ".claude.json")
}

// EnsureAuthDir creates the per-agent auth directories and seeds credentials
// from the host on first use so agents start pre-authenticated.
// The workspaceName is filepath.Base(workspaceDir) — used to pre-create
// Claude's project trust directory for the agent's worktree path.
func EnsureAuthDir(workspaceDir, agentName string) (string, error) {
	workspaceName := filepath.Base(workspaceDir)
	dir := AgentAuthDir(workspaceDir, agentName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create agent auth dir: %w", err)
	}

	// Seed on first use: if .claude.json token is absent the agent will prompt for login
	tokenFile := AgentAuthTokenFile(workspaceDir, agentName)
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		seedAuthFromHost(filepath.Dir(dir)) // pass auth/ parent so seeding writes to correct locations
	}

	// Pre-create project trust directories so the workspace trust prompt is skipped.
	ensureProjectTrust(dir, workspaceName, agentName)

	return dir, nil
}

// seedAuthFromHost seeds agent auth from the host's credentials.
// parentDir is the auth/ directory: it receives .claude.json and .claude/subdir.
func seedAuthFromHost(parentDir string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Copy ~/.claude.json — primary auth token, lives at $HOME level
	claudeJSON := filepath.Join(home, ".claude.json")
	if data, readErr := os.ReadFile(claudeJSON); readErr == nil { //nolint:gosec // known credential file
		dst := filepath.Join(parentDir, ".claude.json")
		if writeErr := os.WriteFile(dst, data, 0600); writeErr != nil {
			log.Debug("failed to seed .claude.json", "error", writeErr)
		}
	}

	// Copy credential files from ~/.claude/ into auth/.claude/
	hostClaudeDir := filepath.Join(home, ".claude")
	agentClaudeDir := filepath.Join(parentDir, ".claude")
	if err := os.MkdirAll(agentClaudeDir, 0700); err != nil {
		return
	}

	for _, name := range []string{".credentials.json", "credentials.json", "settings.json", "auth.json"} {
		src := filepath.Join(hostClaudeDir, name)
		data, err := os.ReadFile(src) //nolint:gosec // known credential file names
		if err != nil {
			continue
		}
		dst := filepath.Join(agentClaudeDir, name)
		if writeErr := os.WriteFile(dst, data, 0600); writeErr != nil {
			log.Debug("failed to seed auth file", "file", name, "error", writeErr)
		}
	}

	// Write default settings.json if not seeded from host — skips theme picker
	// and permission prompts so agents start directly without interaction.
	settingsPath := filepath.Join(agentClaudeDir, "settings.json")
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		writeDefaultSettings(settingsPath)
	}

	log.Debug("seeded agent auth from host credentials", "auth_dir", parentDir)
}

// IsAuthenticated checks if an agent has valid auth credentials.
// Runs `claude auth status` with HOME set to the agent's auth dir parent.
func IsAuthenticated(ctx context.Context, workspaceDir, agentName string) (bool, error) {
	authDir := AgentAuthDir(workspaceDir, agentName)
	if _, err := os.Stat(authDir); os.IsNotExist(err) {
		return false, nil
	}

	// claude auth status --json returns {"loggedIn": true/false, ...}
	//nolint:gosec // trusted binary + agent name from internal state
	cmd := exec.CommandContext(ctx, "claude", "auth", "status", "--json")
	cmd.Env = authEnv(authDir)
	output, err := cmd.Output()
	if err != nil {
		return false, nil
	}

	var status struct {
		LoggedIn bool `json:"loggedIn"`
	}
	if err := json.Unmarshal(output, &status); err != nil {
		return false, nil
	}
	return status.LoggedIn, nil
}

// Login runs `claude auth login` for a specific agent.
// This opens a browser on the host for OAuth. The credentials are stored
// in the agent's isolated auth directory.
//
// Must be run interactively (stdin/stdout/stderr attached to terminal).
func Login(ctx context.Context, workspaceDir, agentName string) error {
	authDir, err := EnsureAuthDir(workspaceDir, agentName)
	if err != nil {
		return err
	}

	//nolint:gosec // trusted binary
	cmd := exec.CommandContext(ctx, "claude", "auth", "login")
	cmd.Env = authEnv(authDir)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// LoginIfNeeded checks auth status and runs login if not authenticated.
func LoginIfNeeded(ctx context.Context, workspaceDir, agentName string) error {
	ok, _ := IsAuthenticated(ctx, workspaceDir, agentName)
	if ok {
		log.Debug("agent already authenticated", "agent", agentName)
		return nil
	}

	fmt.Printf("Agent %q needs authentication. Opening browser for login...\n", agentName)
	return Login(ctx, workspaceDir, agentName)
}

// writeDefaultSettings creates a settings.json that pre-configures Claude Code
// so agents skip interactive prompts and start working immediately.
func writeDefaultSettings(path string) {
	settings := map[string]any{
		// UI — skip theme picker, use dark mode
		"theme": "dark",
		// Permissions — agents run with --dangerously-skip-permissions,
		// this suppresses the confirmation prompt for that mode
		"skipDangerousModePermissionPrompt": true,
		// Auto-update — disabled inside containers (image controls version)
		"autoUpdaterStatus": "disabled",
		// Verbose tool output — helps with debugging agent actions
		"verbose": false,
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
// trust prompt is skipped on agent start. Claude encodes directory paths by
// replacing / with - (e.g., /workspace/.claude/worktrees/bc-myproj-eng-01
// becomes -workspace--claude-worktrees-bc-myproj-eng-01).
func ensureProjectTrust(claudeDir, workspaceName, agentName string) {
	projectsDir := filepath.Join(claudeDir, "projects")

	// The agent's worktree inside Docker is at:
	//   /workspace/.claude/worktrees/bc-<workspaceName>-<agentName>
	// Claude encodes this as:
	//   -workspace--claude-worktrees-bc-<workspaceName>-<agentName>
	worktreeName := "bc-" + workspaceName + "-" + agentName
	encodedPath := "-workspace--claude-worktrees-" + worktreeName

	// Also trust /workspace itself (root of the mounted workspace)
	for _, p := range []string{"-workspace", encodedPath} {
		dir := filepath.Join(projectsDir, p)
		_ = os.MkdirAll(dir, 0700) //nolint:errcheck // best-effort
	}
}

// authEnv returns environment variables with HOME set to the auth parent dir,
// so claude reads/writes to the agent's isolated persistent credential locations.
func authEnv(authDir string) []string {
	// authDir is .../auth/.claude — parent is .../auth/
	// Setting HOME to .../auth/ means:
	//   $HOME/.claude.json = .../auth/.claude.json  (seeded token file)
	//   $HOME/.claude/     = .../auth/.claude/       (settings dir, = authDir)
	home := filepath.Dir(authDir)
	env := os.Environ()
	result := make([]string, 0, len(env)+1)
	for _, e := range env {
		// Replace HOME
		if len(e) > 5 && e[:5] == "HOME=" {
			continue
		}
		result = append(result, e)
	}
	result = append(result, "HOME="+home)
	return result
}
