// Package workspace provides workspace/project management.
package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// UserRCConfigPath returns the path to the user's .bcrc file.
// Default: ~/.bcrc
func UserRCConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".bcrc")
}

// UserRCConfig represents user-level defaults stored in ~/.bcrc.
// These values are merged with workspace config when loading.
type UserRCConfig struct {
	User     UserRCUserConfig     `toml:"user"`
	Defaults UserRCDefaultsConfig `toml:"defaults"`
	Tools    UserRCToolsConfig    `toml:"tools"`
}

// UserRCUserConfig holds user identity settings.
type UserRCUserConfig struct {
	Nickname string `toml:"nickname,omitempty"` // Default nickname for new workspaces
}

// UserRCDefaultsConfig holds default behavior settings.
type UserRCDefaultsConfig struct {
	DefaultRole   string `toml:"default_role,omitempty"`    // Default role for new agents
	AutoStartRoot bool   `toml:"auto_start_root,omitempty"` // Auto-start root agent on bc up
}

// UserRCToolsConfig holds tool preferences.
type UserRCToolsConfig struct {
	Preferred []string `toml:"preferred,omitempty"` // Preferred tools in order
}

// DefaultUserRCConfig returns sensible defaults for a new .bcrc file.
func DefaultUserRCConfig() UserRCConfig {
	return UserRCConfig{
		User: UserRCUserConfig{
			Nickname: DefaultNickname,
		},
		Defaults: UserRCDefaultsConfig{
			DefaultRole:   "engineer",
			AutoStartRoot: true,
		},
		Tools: UserRCToolsConfig{
			Preferred: []string{"claude-code", "gemini"},
		},
	}
}

// LoadUserRCConfig loads the user's .bcrc file.
// Returns nil if the file doesn't exist.
func LoadUserRCConfig() (*UserRCConfig, error) {
	path := UserRCConfigPath()
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path) //nolint:gosec // path from UserHomeDir
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No .bcrc file is fine
		}
		return nil, fmt.Errorf("failed to read .bcrc: %w", err)
	}

	return ParseUserRCConfig(data)
}

// ParseUserRCConfig parses TOML data into a UserRCConfig.
func ParseUserRCConfig(data []byte) (*UserRCConfig, error) {
	var cfg UserRCConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse .bcrc: %w", err)
	}
	return &cfg, nil
}

// Save writes the user config to the .bcrc file.
func (c *UserRCConfig) Save() error {
	path := UserRCConfigPath()
	if path == "" {
		return fmt.Errorf("could not determine home directory")
	}

	f, err := os.Create(path) //nolint:gosec // path from UserHomeDir
	if err != nil {
		return fmt.Errorf("failed to create .bcrc: %w", err)
	}
	defer f.Close() //nolint:errcheck // defer close

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to write .bcrc: %w", err)
	}

	return nil
}

// UserRCExists returns true if ~/.bcrc exists.
func UserRCExists() bool {
	path := UserRCConfigPath()
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

// MergeWithUserRC merges user-level defaults from .bcrc into a workspace config.
// Workspace config takes precedence over user config for all explicitly set values.
// This function only fills in unset values from .bcrc.
func (c *Config) MergeWithUserRC(rc *UserRCConfig) {
	if rc == nil {
		return
	}

	// Merge user identity (only if not set in workspace)
	if c.User.Nickname == "" || c.User.Nickname == DefaultNickname {
		if rc.User.Nickname != "" {
			c.User.Nickname = rc.User.Nickname
		}
	}
}

// GetPreferredTool returns the first available preferred tool from .bcrc,
// or the workspace default if no preference is set.
func (c *Config) GetPreferredTool(rc *UserRCConfig) string {
	if rc == nil || len(rc.Tools.Preferred) == 0 {
		return c.Providers.Default
	}

	// Check if any preferred tool is available in workspace
	for _, tool := range rc.Tools.Preferred {
		if c.HasProviderDefined(tool) {
			return tool
		}
	}

	return c.Providers.Default
}
