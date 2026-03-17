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

// AgentAuthDir returns the per-agent auth directory.
// This directory is mounted into the container as ~/.claude/ and persists
// across container restarts. Each agent has its own isolated credentials.
//
// Layout: <workspaceDir>/.bc/agents/<agentName>/auth/.claude/
func AgentAuthDir(workspaceDir, agentName string) string {
	return filepath.Join(workspaceDir, ".bc", "agents", agentName, "auth", ".claude")
}

// EnsureAuthDir creates the per-agent auth directory if it doesn't exist.
// If the directory is newly created (empty), seeds it from the host's ~/.claude
// credentials so agents don't require a separate login.
func EnsureAuthDir(workspaceDir, agentName string) (string, error) {
	dir := AgentAuthDir(workspaceDir, agentName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create agent auth dir: %w", err)
	}

	// If the auth dir is empty, seed from host credentials
	entries, err := os.ReadDir(dir)
	if err == nil && len(entries) == 0 {
		seedAuthFromHost(dir)
	}

	return dir, nil
}

// seedAuthFromHost copies credential files from the host's ~/.claude directory
// into the agent's auth directory so agents inherit the host's authentication.
func seedAuthFromHost(agentAuthDir string) {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	hostClaudeDir := filepath.Join(home, ".claude")
	if _, err := os.Stat(hostClaudeDir); os.IsNotExist(err) {
		return
	}

	// Copy credential-related files (not the entire directory)
	credFiles := []string{
		".credentials.json",
		"credentials.json",
		"settings.json",
		"auth.json",
	}

	for _, name := range credFiles {
		src := filepath.Join(hostClaudeDir, name)
		data, err := os.ReadFile(src) //nolint:gosec // src is constructed from known credential file names
		if err != nil {
			continue // file doesn't exist or not readable
		}
		dst := filepath.Join(agentAuthDir, name)
		if writeErr := os.WriteFile(dst, data, 0600); writeErr != nil {
			log.Debug("failed to seed auth file", "file", name, "error", writeErr)
		}
	}

	log.Debug("seeded agent auth from host credentials", "agent_auth_dir", agentAuthDir)
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

// authEnv returns environment variables with HOME set to the agent's auth
// directory parent, so claude stores credentials in the agent's isolated dir.
func authEnv(authDir string) []string {
	// authDir is .../auth/.claude, so parent is .../auth/
	// Setting HOME to .../auth/ means claude writes to $HOME/.claude/ = authDir
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
