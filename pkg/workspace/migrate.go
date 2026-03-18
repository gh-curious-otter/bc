package workspace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotV1Workspace is returned when migration is attempted on a non-v1 workspace.
var ErrNotV1Workspace = errors.New("not a v1 workspace (no .bc/config.json found)")

// V1Config represents the legacy JSON config format used in bc v1.
// Fields are mapped from the original config.json schema.
type V1Config struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Provider    string            `json:"provider"`    // default provider name
	Command     string            `json:"command"`     // default provider command
	Providers   map[string]string `json:"providers"`   // name → command
	Nickname    string            `json:"nickname"`    // user nickname
	Runtime     string            `json:"runtime"`     // "tmux" or "docker"
}

// MigrateResult summarises what the migration changed.
type MigrateResult struct { //nolint:govet // fieldalignment: readability preferred over padding
	BackupPath      string // path to the backed-up config.json
	ConfigMigrated  bool   // config.json → config.toml
	AgentFiles      int    // .json agent files found (auto-migrated on next Load)
	ChannelJSON     bool   // channels.json found (auto-migrated on next channel open)
}

// V1ConfigPath returns the path to the v1 config.json.
func V1ConfigPath(rootDir string) string {
	return filepath.Join(rootDir, ".bc", "config.json")
}

// LoadV1Config reads and parses the v1 config.json from rootDir.
// Returns ErrNotV1Workspace if the file does not exist.
func LoadV1Config(rootDir string) (*V1Config, error) {
	path := V1ConfigPath(rootDir)
	data, err := os.ReadFile(path) //nolint:gosec // path is constructed from user-provided workspace root
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotV1Workspace
		}
		return nil, fmt.Errorf("read config.json: %w", err)
	}

	var cfg V1Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}
	return &cfg, nil
}

// MigrateV1ToV2 migrates a v1 workspace to v2 format.
//
// Steps performed:
//  1. Read .bc/config.json
//  2. Write .bc/config.json.bak (backup)
//  3. Convert and write .bc/config.toml (v2 format)
//
// Agent JSON→SQLite and channel JSON→SQLite migrations happen automatically
// the next time those stores are opened (existing auto-migration in pkg/agent
// and pkg/channel).
//
// Returns ErrNotV1Workspace if .bc/config.json does not exist.
func MigrateV1ToV2(rootDir string) (*MigrateResult, error) {
	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	v1cfg, err := LoadV1Config(absRoot)
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(absRoot, ".bc")
	result := &MigrateResult{}

	// ── 1. Backup original config.json ───────────────────────────────────────

	v1Path := V1ConfigPath(absRoot)
	backupPath := v1Path + ".bak"
	data, err := os.ReadFile(v1Path) //nolint:gosec // same path, already stat'd
	if err != nil {
		return nil, fmt.Errorf("read config.json for backup: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return nil, fmt.Errorf("write backup: %w", err)
	}
	result.BackupPath = backupPath

	// ── 2. Build v2 Config from v1 fields ────────────────────────────────────

	name := v1cfg.Name
	if name == "" {
		name = filepath.Base(absRoot)
	}
	v2cfg := DefaultConfig(name)

	if v1cfg.Nickname != "" {
		normalized, err := NormalizeNickname(v1cfg.Nickname)
		if err == nil {
			v2cfg.User.Nickname = normalized
		}
	}

	if v1cfg.Runtime != "" {
		v2cfg.Runtime.Backend = v1cfg.Runtime
	}

	// Map v1 provider → v2 ProviderConfig entries.
	// v1 stored provider commands either in the top-level Command field
	// (for the default provider) or in the Providers map.
	providerCmds := buildProviderMap(v1cfg)
	if len(providerCmds) > 0 {
		// Set the default provider
		defaultProv := strings.ToLower(v1cfg.Provider)
		if defaultProv == "" {
			defaultProv = firstKey(providerCmds)
		}
		v2cfg.Providers.Default = defaultProv

		for prov, cmd := range providerCmds {
			pc := &ProviderConfig{Command: cmd, Enabled: true}
			switch strings.ToLower(prov) {
			case "claude":
				v2cfg.Providers.Claude = pc
			case "gemini":
				v2cfg.Providers.Gemini = pc
			case "cursor":
				v2cfg.Providers.Cursor = pc
			case "codex":
				v2cfg.Providers.Codex = pc
			case "aider":
				v2cfg.Providers.Aider = pc
			case "opencode":
				v2cfg.Providers.OpenCode = pc
			case "openclaw":
				v2cfg.Providers.OpenClaw = pc
			}
		}
	}

	// ── 3. Write config.toml ─────────────────────────────────────────────────

	tomlPath := filepath.Join(stateDir, "config.toml")
	if err := v2cfg.Save(tomlPath); err != nil {
		return nil, fmt.Errorf("write config.toml: %w", err)
	}
	result.ConfigMigrated = true

	// ── 4. Count legacy files so we can report them ───────────────────────────

	// Agent JSON files (auto-migrated by pkg/agent on next LoadState)
	result.AgentFiles = countJSONAgentFiles(stateDir)

	// channels.json (auto-migrated by pkg/channel on Open)
	if _, statErr := os.Stat(filepath.Join(stateDir, "channels.json")); statErr == nil {
		result.ChannelJSON = true
	}

	// Ensure required sub-directories exist
	for _, sub := range []string{"agents", "roles", "channels", "prompts"} {
		_ = os.MkdirAll(filepath.Join(stateDir, sub), 0750)
	}

	return result, nil
}

// buildProviderMap merges the v1 top-level Command (for the default provider)
// with any entries in the Providers map.
func buildProviderMap(v1cfg *V1Config) map[string]string {
	m := make(map[string]string)

	// Populate from the Providers map first.
	for k, v := range v1cfg.Providers {
		if k != "" && v != "" {
			m[strings.ToLower(k)] = v
		}
	}

	// If a top-level Command is given, apply it to the default provider.
	if v1cfg.Command != "" && v1cfg.Provider != "" {
		m[strings.ToLower(v1cfg.Provider)] = v1cfg.Command
	}

	// If nothing explicit, seed with the known provider defaults.
	if len(m) == 0 && v1cfg.Provider != "" {
		m[strings.ToLower(v1cfg.Provider)] = defaultCommandFor(v1cfg.Provider)
	}

	return m
}

// defaultCommandFor returns a sensible default command for well-known providers.
func defaultCommandFor(provider string) string {
	switch strings.ToLower(provider) {
	case "claude":
		return "claude --dangerously-skip-permissions"
	case "gemini":
		return "gemini --yolo"
	case "cursor":
		return "cursor"
	case "codex":
		return "codex"
	case "aider":
		return "aider"
	default:
		return provider
	}
}

// firstKey returns an arbitrary key from a map (used when no default is set).
func firstKey(m map[string]string) string {
	for k := range m {
		return k
	}
	return ""
}

// CountLegacyAgentFiles counts legacy per-agent JSON files in the agents directory.
// Exported so callers (e.g. the migrate command) can show a file count preview.
func CountLegacyAgentFiles(stateDir string) int {
	return countJSONAgentFiles(stateDir)
}

// countJSONAgentFiles counts legacy per-agent JSON files in the agents directory.
func countJSONAgentFiles(stateDir string) int {
	agentsDir := filepath.Join(stateDir, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			count++
		}
	}

	// Also count the top-level agents.json / root.json
	for _, f := range []string{
		filepath.Join(stateDir, "agents.json"),
		filepath.Join(stateDir, "root.json"),
	} {
		if _, err := os.Stat(f); err == nil {
			count++
		}
	}
	return count
}
