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
func EnsureAuthDir(workspaceDir, agentName string) (string, error) {
	dir := AgentAuthDir(workspaceDir, agentName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create agent auth dir: %w", err)
	}

	// Seed on first use: if .claude.json token is absent the agent will prompt for login
	tokenFile := AgentAuthTokenFile(workspaceDir, agentName)
	if _, err := os.Stat(tokenFile); os.IsNotExist(err) {
		seedAuthFromHost(filepath.Dir(dir)) // pass auth/ parent so seeding writes to correct locations
	}

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
