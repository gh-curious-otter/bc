package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rpuneet/bc/pkg/log"
	pkgmcp "github.com/rpuneet/bc/pkg/mcp"
	"github.com/rpuneet/bc/pkg/secret"
	"github.com/rpuneet/bc/pkg/workspace"
)

// mcpConfig represents the .mcp.json file format that Claude Code expects.
type mcpConfig struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

// mcpServerEntry represents a single MCP server entry in .mcp.json.
type mcpServerEntry struct {
	Env     map[string]string `json:"env,omitempty"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	Type    string            `json:"type,omitempty"`
}

// secretRefPattern matches ${secret:NAME} references in env var values.
var secretRefPattern = regexp.MustCompile(`\$\{secret:([^}]+)\}`)

// GenerateAgentMCPConfig generates a .mcp.json file in the agent's working directory.
// It resolves the role's MCP server list via BFS inheritance, looks up each server
// definition from the workspace MCP store, resolves ${secret:NAME} env var references,
// and always includes the bc MCP server.
func GenerateAgentMCPConfig(workspacePath, agentName, roleName, targetDir string) error {
	cfg := mcpConfig{
		MCPServers: make(map[string]mcpServerEntry),
	}

	// Always include the bc MCP server with SSE transport
	cfg.MCPServers["bc"] = mcpServerEntry{
		Type: "sse",
		URL:  "http://localhost:9374/mcp/sse",
	}

	stateDir := filepath.Join(workspacePath, ".bc")

	// Resolve the role to get inherited MCP server names
	rm := workspace.NewRoleManager(stateDir)
	resolved, resolveErr := rm.ResolveRole(roleName)
	if resolveErr != nil {
		log.Debug("failed to resolve role for MCP config", "role", roleName, "error", resolveErr)
		return writeMCPConfig(targetDir, &cfg)
	}

	// Open the MCP server store to look up server definitions
	mcpStore, mcpErr := pkgmcp.NewStore(workspacePath)
	if mcpErr != nil {
		log.Debug("MCP store unavailable, using role servers only", "error", mcpErr)
		return writeMCPConfig(targetDir, &cfg)
	}
	defer mcpStore.Close() //nolint:errcheck

	// Build secrets map for resolving ${secret:NAME} references
	secrets := loadSecrets(workspacePath, resolved.Secrets)

	// Look up each MCP server from the workspace MCP store
	for _, serverName := range resolved.MCPServers {
		if serverName == "bc" {
			continue // Already added
		}

		serverDef, err := mcpStore.Get(serverName)
		if err != nil {
			log.Debug("MCP server not found in store", "server", serverName, "agent", agentName)
			continue
		}
		if !serverDef.Enabled {
			continue
		}

		entry := mcpServerEntry{
			Command: serverDef.Command,
			Args:    serverDef.Args,
			URL:     serverDef.URL,
		}
		if serverDef.Transport == "sse" {
			entry.Type = "sse"
		}

		// Resolve env vars with secret references
		if len(serverDef.Env) > 0 {
			entry.Env = make(map[string]string, len(serverDef.Env))
			for k, v := range serverDef.Env {
				entry.Env[k] = resolveSecretValue(v, secrets)
			}
		}

		cfg.MCPServers[serverName] = entry
	}

	return writeMCPConfig(targetDir, &cfg)
}

// loadSecrets loads secret values from the secret store for the given secret names.
func loadSecrets(workspacePath string, secretNames []string) map[string]string {
	secrets := make(map[string]string)
	if len(secretNames) == 0 {
		return secrets
	}

	ss, err := secret.NewStore(workspacePath, "")
	if err != nil {
		log.Debug("secret store unavailable", "error", err)
		return secrets
	}
	defer ss.Close() //nolint:errcheck

	for _, name := range secretNames {
		val, getErr := ss.GetValue(name)
		if getErr != nil {
			log.Debug("secret not found", "secret", name)
			continue
		}
		secrets[name] = val
	}

	return secrets
}

// resolveSecretValue replaces ${secret:NAME} references in a single string
// with values from the secrets map. If a secret is not found, the reference
// is replaced with an empty string.
func resolveSecretValue(value string, secrets map[string]string) string {
	if !strings.Contains(value, "${secret:") {
		return value
	}

	return secretRefPattern.ReplaceAllStringFunc(value, func(match string) string {
		submatch := secretRefPattern.FindStringSubmatch(match)
		if len(submatch) < 2 {
			return match
		}
		name := submatch[1]
		if v, ok := secrets[name]; ok {
			return v
		}
		log.Debug("secret not found for MCP env var", "secret", name)
		return ""
	})
}

// writeMCPConfig writes the .mcp.json file to the target directory.
func writeMCPConfig(targetDir string, cfg *mcpConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	mcpPath := filepath.Join(targetDir, ".mcp.json")
	if writeErr := os.WriteFile(mcpPath, data, 0600); writeErr != nil {
		return fmt.Errorf("failed to write .mcp.json: %w", writeErr)
	}

	log.Debug("generated .mcp.json", "path", mcpPath, "servers", len(cfg.MCPServers))
	return nil
}
