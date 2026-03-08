package container

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/rpuneet/bc/pkg/log"
)

// credentialsFile holds extracted auth credentials for a provider.
type credentialsFile struct {
	Provider string `json:"provider"`
	Token    string `json:"token"`
}

// EnsureCredentials extracts auth tokens from the host system and writes them
// to a per-agent credentials directory that can be mounted into containers.
// Each agent gets its own isolated copy so credentials can diverge at runtime.
//
// For macOS: extracts from the Keychain.
// For Linux: extracts from file-based credential stores.
//
// Returns the path to the agent's credentials directory.
func EnsureCredentials(workspaceDir, agentName string) (string, error) {
	credsDir := filepath.Join(workspaceDir, ".bc", credentialsDirName, agentName)
	if err := os.MkdirAll(credsDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create credentials dir: %w", err)
	}

	// Extract Claude OAuth token
	if err := extractClaudeToken(credsDir); err != nil {
		log.Debug("failed to extract claude token (may need login)", "error", err)
	}

	// Extract other provider tokens from env vars and write to files
	// This way they're in files (not visible in docker inspect) and mounted read-only.
	envProviders := map[string]string{
		"anthropic": "ANTHROPIC_API_KEY",
		"google":    "GOOGLE_API_KEY",
		"gemini":    "GEMINI_API_KEY",
		"openai":    "OPENAI_API_KEY",
	}
	for name, envKey := range envProviders {
		if val := os.Getenv(envKey); val != "" {
			cred := credentialsFile{Provider: name, Token: val}
			writeCredential(credsDir, name, cred)
		}
	}

	return credsDir, nil
}

// EnsureAllCredentials extracts credentials for all agents or a default set.
// Used by `bc agent auth` to refresh credentials workspace-wide.
func EnsureAllCredentials(workspaceDir string) (string, error) {
	return EnsureCredentials(workspaceDir, "_default")
}

// extractClaudeToken pulls the Claude Code OAuth token from the system keychain.
func extractClaudeToken(credsDir string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keychain extraction only supported on macOS")
	}

	// Try both keychain entries Claude Code uses
	ctx := context.Background()
	for _, service := range []string{"claude-code", "claude"} {
		//nolint:gosec // service names are hardcoded strings
		cmd := exec.CommandContext(ctx, "security", "find-generic-password", "-s", service, "-w")
		output, err := cmd.Output()
		if err != nil {
			continue
		}
		token := string(output)
		if len(token) > 0 {
			// Remove trailing newline
			if token[len(token)-1] == '\n' {
				token = token[:len(token)-1]
			}
			cred := credentialsFile{Provider: service, Token: token}
			writeCredential(credsDir, service, cred)
			return nil
		}
	}

	return fmt.Errorf("no claude token found in keychain")
}

// writeCredential writes a credential to a JSON file with restricted permissions.
func writeCredential(dir, name string, cred credentialsFile) {
	data, err := json.Marshal(cred)
	if err != nil {
		log.Warn("failed to marshal credential", "name", name, "error", err)
		return
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		log.Warn("failed to write credential", "name", name, "error", err)
	}
}

// AgentCredentialsDir returns the per-agent credentials directory path.
func AgentCredentialsDir(workspaceDir, agentName string) string {
	return filepath.Join(workspaceDir, ".bc", credentialsDirName, agentName)
}
