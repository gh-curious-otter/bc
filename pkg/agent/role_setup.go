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

// SetupAgentFromRole resolves a role via BFS inheritance and writes all
// Claude Code configuration files to the agent's working directory:
//
//   - CLAUDE.md              ← role prompt body
//   - .mcp.json              ← resolved MCP servers with secret injection
//   - .claude/settings.json  ← role settings (hooks, permissions)
//   - .claude/commands/*.md  ← custom slash commands
//   - .claude/skills/*.md    ← reusable skills
//   - .claude/agents/*.md    ← subagent definitions
//   - .claude/rules/*.md     ← topic-specific rules
//   - REVIEW.md              ← code review checklist
//
// SetupAgentFromRoleWithRuntime sets up agent workspace files for the given role
// and runtime backend. Docker agents skip stdio-transport MCP servers (unreachable).
func SetupAgentFromRoleWithRuntime(workspacePath, agentName, roleName, targetDir, runtimeBackend string) error {
	return setupAgentFromRole(workspacePath, agentName, roleName, targetDir, runtimeBackend)
}

// SetupAgentFromRole sets up agent workspace files for the given role.
// Defaults to tmux runtime (all MCP transports available).
func SetupAgentFromRole(workspacePath, agentName, roleName, targetDir string) error {
	return setupAgentFromRole(workspacePath, agentName, roleName, targetDir, "tmux")
}

func setupAgentFromRole(workspacePath, agentName, roleName, targetDir, runtimeBackend string) error {
	stateDir := filepath.Join(workspacePath, ".bc")
	rm := workspace.NewRoleManager(stateDir)

	resolved, err := rm.ResolveRole(roleName)
	if err != nil {
		log.Warn("failed to resolve role, skipping setup", "role", roleName, "error", err)
		return nil
	}

	secrets := loadSecrets(workspacePath, resolved.Secrets)
	var errs []string

	// CLAUDE.md (project-level prompt)
	if resolved.Prompt != "" {
		if e := writeTextFile(targetDir, "CLAUDE.md", resolved.Prompt); e != nil {
			errs = append(errs, e.Error())
		}
	}

	// .mcp.json (project-level MCP config)
	if e := writeMCPJSON(workspacePath, agentName, resolved, secrets, targetDir, runtimeBackend); e != nil {
		errs = append(errs, e.Error())
	}

	// .claude/settings.json (project-level settings)
	if len(resolved.Settings) > 0 {
		if e := writeJSONFile(filepath.Join(targetDir, ".claude"), "settings.json", resolved.Settings); e != nil {
			errs = append(errs, e.Error())
		}
	}

	// Write plugin config to the agent's auth dir (user-level .claude/)
	// so Docker agents have plugins available without host mounts.
	authClaudeDir := filepath.Join(workspacePath, ".bc", "agents", agentName, "auth", ".claude")
	if len(resolved.Plugins) > 0 {
		if e := writePluginConfig(authClaudeDir, resolved.Plugins); e != nil {
			errs = append(errs, e.Error())
		}
	}

	// .claude/commands/*.md, skills/*.md, agents/*.md, rules/*.md
	claudeDir := filepath.Join(targetDir, ".claude")
	for _, pair := range []struct {
		files map[string]string
		dir   string
	}{
		{resolved.Commands, "commands"},
		{resolved.Skills, "skills"},
		{resolved.Agents, "agents"},
		{resolved.Rules, "rules"},
	} {
		if e := writeMDFiles(filepath.Join(claudeDir, pair.dir), pair.files); e != nil {
			errs = append(errs, e.Error())
		}
	}

	// REVIEW.md
	if resolved.Review != "" {
		if e := writeTextFile(targetDir, "REVIEW.md", resolved.Review); e != nil {
			errs = append(errs, e.Error())
		}
	}

	if len(errs) > 0 {
		log.Warn("some role files failed to write", "agent", agentName, "errors", len(errs))
		return fmt.Errorf("role setup: %s", strings.Join(errs, "; "))
	}

	log.Debug("agent role setup complete", "agent", agentName, "role", roleName)
	return nil
}

// BuildSetupCommand is deprecated — MCP servers and plugins are now configured
// via file-based config (.mcp.json and .claude.json) written by SetupAgentFromRole.
// Running `claude mcp add` at startup kills running Claude instances and causes
// restart loops in Docker containers.
//
// Kept as a no-op for backward compatibility. Will be removed in a future release.
func BuildSetupCommand(_, _ string) string {
	return ""
}

// ── MCP config ──────────────────────────────────────────────────────────────

type mcpConfig struct {
	MCPServers map[string]mcpServerEntry `json:"mcpServers"`
}

type mcpServerEntry struct {
	Env     map[string]string `json:"env,omitempty"`
	Command string            `json:"command,omitempty"`
	URL     string            `json:"url,omitempty"`
	Type    string            `json:"type,omitempty"`
	Args    []string          `json:"args,omitempty"`
}

var secretRefPattern = regexp.MustCompile(`\$\{secret:([^}]+)\}`)

func writeMCPJSON(workspacePath, agentName string, resolved *workspace.ResolvedRole, secrets map[string]string, targetDir, runtimeBackend string) error {
	isDocker := runtimeBackend == "docker"
	cfg := mcpConfig{MCPServers: make(map[string]mcpServerEntry)}

	mcpStore, mcpErr := pkgmcp.NewStore(workspacePath)
	if mcpErr != nil {
		log.Debug("MCP store unavailable", "error", mcpErr)
		return writeJSONFile(targetDir, ".mcp.json", cfg)
	}
	defer mcpStore.Close() //nolint:errcheck

	for _, name := range resolved.MCPServers {
		def, getErr := mcpStore.Get(name)
		if getErr != nil || def == nil || !def.Enabled {
			continue
		}
		// Docker agents can't use stdio-transport MCP servers (no access to
		// host processes). Skip with warning — use tmux runtime for full MCP.
		if isDocker && def.Transport != "sse" {
			log.Warn("skipping stdio MCP server for Docker agent (unreachable)",
				"agent", agentName, "mcp", name,
				"hint", "use tmux runtime for stdio MCP servers")
			continue
		}
		entry := mcpServerEntry{Command: def.Command, Args: def.Args, URL: def.URL}
		if isDocker && entry.URL != "" {
			entry.URL = rewriteDockerURL(entry.URL)
		}
		if def.Transport == "sse" {
			entry.Type = "sse"
		}
		if len(def.Env) > 0 {
			entry.Env = make(map[string]string, len(def.Env))
			for k, v := range def.Env {
				// Try to resolve ${secret:NAME} to actual value
				resolved := resolveSecretValue(v, secrets)
				if resolved != "" {
					entry.Env[k] = resolved
				} else {
					// Secret not available — use env var reference so Claude Code
					// can resolve it from the container environment instead
					entry.Env[k] = "${" + k + "}"
				}
			}
		}
		cfg.MCPServers[name] = entry
	}

	return writeJSONFile(targetDir, ".mcp.json", cfg)
}

// rewriteDockerURL rewrites localhost URLs to host.docker.internal so Docker
// containers can reach services on the host. Works on macOS and Windows;
// on Linux, host.docker.internal requires --add-host in docker run.
func rewriteDockerURL(u string) string {
	u = strings.Replace(u, "localhost", "host.docker.internal", 1)
	u = strings.Replace(u, "127.0.0.1", "host.docker.internal", 1)
	return u
}

// ── Secrets ─────────────────────────────────────────────────────────────────

func loadSecrets(workspacePath string, names []string) map[string]string {
	m := make(map[string]string)
	if len(names) == 0 {
		return m
	}
	ss, err := secret.NewStore(workspacePath, "")
	if err != nil {
		return m
	}
	defer ss.Close() //nolint:errcheck
	for _, n := range names {
		if v, e := ss.GetValue(n); e == nil {
			m[n] = v
		}
	}
	return m
}

func resolveSecretValue(value string, secrets map[string]string) string {
	if !strings.Contains(value, "${secret:") {
		return value
	}
	return secretRefPattern.ReplaceAllStringFunc(value, func(match string) string {
		sub := secretRefPattern.FindStringSubmatch(match)
		if len(sub) < 2 {
			return match
		}
		if v, ok := secrets[sub[1]]; ok {
			return v
		}
		return ""
	})
}

// ── Plugin config ────────────────────────────────────────────────────────────

// writePluginConfig is a no-op for now.
// Plugins must be installed inside the Docker container at runtime
// (via /plugin command) or pre-installed in the Docker image.
// Copying host installed_plugins.json doesn't work because paths
// reference the host filesystem which doesn't exist in containers.
func writePluginConfig(_ string, _ []string) error {
	return nil
}

// ── File writers ────────────────────────────────────────────────────────────

func writeTextFile(dir, name, content string) error {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(filepath.Join(dir, name), []byte(content), 0600)
}

func writeJSONFile(dir, name string, data any) error {
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", name, err)
	}
	return os.WriteFile(filepath.Join(dir, name), b, 0600)
}

func writeMDFiles(dir string, files map[string]string) error {
	if len(files) == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	for name, content := range files {
		fname := name
		if !strings.HasSuffix(fname, ".md") {
			fname += ".md"
		}
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if err := os.WriteFile(filepath.Join(dir, fname), []byte(content), 0600); err != nil {
			return fmt.Errorf("write %s: %w", fname, err)
		}
	}
	return nil
}
