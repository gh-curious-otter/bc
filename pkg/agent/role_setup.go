package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gh-curious-otter/bc/pkg/log"
	pkgmcp "github.com/gh-curious-otter/bc/pkg/mcp"
	"github.com/gh-curious-otter/bc/pkg/secret"
	pkgtool "github.com/gh-curious-otter/bc/pkg/tool"
	"github.com/gh-curious-otter/bc/pkg/workspace"
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
	// Merge role settings into existing settings.json to preserve hooks
	// written by WriteWorkspaceHookSettings (called before role setup).
	if len(resolved.Settings) > 0 {
		if e := mergeSettingsJSON(filepath.Join(targetDir, ".claude"), resolved.Settings); e != nil {
			errs = append(errs, e.Error())
		}
	}

	// Write plugin config to the agent's Claude home dir (~/.claude/ in container).
	// This is the "claude/" dir that gets mounted as /home/agent/.claude.
	agentClaudeDir := filepath.Join(workspacePath, ".bc", "agents", agentName, "claude")
	if len(resolved.Plugins) > 0 {
		if e := writePluginConfig(agentClaudeDir, resolved.Plugins); e != nil {
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
		subDir := filepath.Join(claudeDir, pair.dir)
		if e := cleanStaleMDFiles(subDir, pair.files); e != nil {
			errs = append(errs, e.Error())
		}
		if e := writeMDFiles(subDir, pair.files); e != nil {
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

	// Try unified tool store first for MCP server configs, fall back to mcp.Store.
	toolStore := pkgtool.NewStore(filepath.Join(workspacePath, ".bc"))
	var toolStoreOpen bool
	if openErr := toolStore.Open(); openErr == nil {
		toolStoreOpen = true
		defer toolStore.Close() //nolint:errcheck
	}

	mcpStore, mcpErr := pkgmcp.NewStore(workspacePath)
	if mcpErr != nil && !toolStoreOpen {
		log.Debug("both tool and MCP stores unavailable", "error", mcpErr)
		return writeJSONFile(targetDir, ".mcp.json", cfg)
	}
	if mcpErr == nil {
		defer mcpStore.Close() //nolint:errcheck
	}

	for _, name := range resolved.MCPServers {
		// Try unified tool store first
		var transport, command, url string
		var args []string
		var env map[string]string
		var enabled bool
		var found bool

		if toolStoreOpen {
			t, tErr := toolStore.Get(context.Background(), name)
			if tErr == nil && t != nil && t.Type == pkgtool.ToolTypeMCP {
				transport = t.Transport
				command = t.Command
				url = t.URL
				args = t.Args
				env = t.Env
				enabled = t.Enabled
				found = true
			}
		}

		// Fall back to mcp.Store
		if !found && mcpErr == nil {
			def, getErr := mcpStore.Get(name)
			if getErr == nil && def != nil {
				transport = string(def.Transport)
				command = def.Command
				url = def.URL
				args = def.Args
				env = def.Env
				enabled = def.Enabled
				found = true
			}
		}

		if !found || !enabled {
			continue
		}
		// Docker agents can't use stdio-transport MCP servers (no access to
		// host processes). Skip with warning — use tmux runtime for full MCP.
		if isDocker && transport != "sse" {
			log.Warn("skipping stdio MCP server for Docker agent (unreachable)",
				"agent", agentName, "mcp", name,
				"hint", "use tmux runtime for stdio MCP servers")
			continue
		}
		entry := mcpServerEntry{Command: command, Args: args, URL: url}
		if isDocker && entry.URL != "" {
			entry.URL = rewriteDockerURL(entry.URL)
		}
		if entry.URL != "" && strings.Contains(entry.URL, "/_mcp/sse") {
			entry.URL = strings.Replace(entry.URL, "/_mcp/sse", "/_mcp/"+agentName+"/sse", 1)
		}
		if transport == "sse" {
			entry.Type = "sse"
		}
		if len(env) > 0 {
			entry.Env = make(map[string]string, len(env))
			for k, v := range env {
				resolved := resolveSecretValue(v, secrets)
				if resolved != "" {
					entry.Env[k] = resolved
				} else {
					entry.Env[k] = "${" + k + "}"
				}
			}
		}
		cfg.MCPServers[name] = entry
	}

	// Always ensure the bc MCP server is included with agent-scoped URL.
	// This is required for send_message, report_status, and other bc tools.
	if _, hasBc := cfg.MCPServers["bc"]; !hasBc {
		bcDef, bcErr := mcpStore.Get("bc")
		if bcErr == nil && bcDef != nil {
			bcURL := bcDef.URL
			if isDocker && bcURL != "" {
				bcURL = rewriteDockerURL(bcURL)
			}
			if bcURL != "" && strings.Contains(bcURL, "/_mcp/sse") {
				bcURL = strings.Replace(bcURL, "/_mcp/sse", "/_mcp/"+agentName+"/sse", 1)
			}
			cfg.MCPServers["bc"] = mcpServerEntry{URL: bcURL, Type: "sse"}
		}
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

// pluginEntry represents a single plugin in installed_plugins.json.
type pluginEntry struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Enabled bool   `json:"enabled"`
}

// pluginManifest is the top-level structure for installed_plugins.json.
type pluginManifest struct {
	Plugins map[string]pluginEntry `json:"plugins"`
}

// writePluginConfig writes an installed_plugins.json manifest so Claude Code
// knows which plugins to load. The file is placed in the agent's Claude home
// directory (mounted as /home/agent/.claude in Docker containers).
func writePluginConfig(claudeDir string, plugins []string) error {
	manifest := pluginManifest{
		Plugins: make(map[string]pluginEntry, len(plugins)),
	}
	for _, name := range plugins {
		manifest.Plugins[name] = pluginEntry{
			Name:    name,
			Source:  "claude-plugins-official",
			Enabled: true,
		}
	}
	return writeJSONFile(claudeDir, "installed_plugins.json", manifest)
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

// cleanStaleMDFiles removes .md files from dir that are not in the new file set.
// Non-.md files and the directory itself are left untouched.
func cleanStaleMDFiles(dir string, newFiles map[string]string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // directory doesn't exist yet, nothing to clean
		}
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	// Build a set of expected .md filenames from the new file map.
	expected := make(map[string]struct{}, len(newFiles))
	for name := range newFiles {
		fname := name
		if !strings.HasSuffix(fname, ".md") {
			fname += ".md"
		}
		expected[fname] = struct{}{}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		if _, ok := expected[name]; ok {
			continue
		}
		// Stale .md file — remove it.
		if rmErr := os.Remove(filepath.Join(dir, name)); rmErr != nil {
			return fmt.Errorf("remove stale file %s: %w", name, rmErr)
		}
	}
	return nil
}

// mergeSettingsJSON reads existing settings.json from dir, merges role settings
// into it (role settings override non-hook keys, but existing hooks are preserved),
// and writes the merged result back.
func mergeSettingsJSON(dir string, roleSettings map[string]any) error {
	settingsPath := filepath.Join(dir, "settings.json")

	// Read existing settings (may contain hooks from WriteWorkspaceHookSettings).
	existing := make(map[string]any)
	data, err := os.ReadFile(settingsPath)
	if err == nil {
		_ = json.Unmarshal(data, &existing) //nolint:errcheck // if malformed, start fresh
	}

	// Preserve existing hooks — save them before merging.
	savedHooks, hasHooks := existing["hooks"]

	// Merge role settings into existing (role overrides existing keys).
	for k, v := range roleSettings {
		existing[k] = v
	}

	// Restore hooks if they existed and role settings didn't explicitly set them,
	// or merge them back so hooks from WriteWorkspaceHookSettings are not lost.
	if hasHooks {
		if _, roleHasHooks := roleSettings["hooks"]; !roleHasHooks {
			existing["hooks"] = savedHooks
		}
	}

	return writeJSONFile(dir, "settings.json", existing)
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

// validateAgentTools checks that CLI tools listed in the role config are available.
// Returns a list of issues found (empty = all good).
func validateAgentTools(workspacePath, roleName string) []string {
	if roleName == "" {
		return nil
	}

	stateDir := filepath.Join(workspacePath, ".bc")
	rm := workspace.NewRoleManager(stateDir)

	resolved, err := rm.ResolveRole(roleName)
	if err != nil {
		return []string{fmt.Sprintf("cannot resolve role %q: %v", roleName, err)}
	}

	var issues []string

	// Check CLI tools from role config
	for _, toolName := range resolved.CLITools {
		if _, err := execLookPath(toolName); err != nil {
			issues = append(issues, fmt.Sprintf("CLI tool %q not found in PATH", toolName))
		}
	}

	// Check MCP servers from role config — verify definition exists and health check
	mcpStore, mcpErr := pkgmcp.NewStore(workspacePath)
	if mcpErr != nil {
		issues = append(issues, fmt.Sprintf("MCP store unavailable: %v", mcpErr))
		return issues
	}
	defer mcpStore.Close() //nolint:errcheck

	for _, name := range resolved.MCPServers {
		def, getErr := mcpStore.Get(name)
		if getErr != nil || def == nil {
			issues = append(issues, fmt.Sprintf("MCP server %q not defined in store", name))
			continue
		}

		// Health check based on transport type
		switch string(def.Transport) {
		case "sse":
			if def.URL != "" {
				if err := checkSSEEndpoint(def.URL); err != nil {
					issues = append(issues, fmt.Sprintf("MCP server %q SSE endpoint unreachable: %v", name, err))
				}
			}
		case "stdio":
			if def.Command != "" {
				cmd := strings.Fields(def.Command)[0]
				if _, err := execLookPath(cmd); err != nil {
					issues = append(issues, fmt.Sprintf("MCP server %q command %q not found in PATH", name, cmd))
				}
			}
		}
	}

	return issues
}

// execLookPath is a testable wrapper around exec.LookPath.
var execLookPath = defaultLookPath

func defaultLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// checkSSEEndpoint verifies an MCP SSE endpoint is reachable by sending a
// HEAD request with a short timeout. Returns nil if the endpoint responds.
func checkSSEEndpoint(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	_ = resp.Body.Close()
	return nil
}
