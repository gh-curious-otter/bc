package container

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// defaultClaudeSettings is the minimal settings file that prevents the
// interactive theme selection prompt on first run of Claude Code.
var defaultClaudeSettings = map[string]string{
	"theme": "dark",
}

// SeedClaudeSettings writes a default settings.json into the given Claude
// volume directory if one does not already exist. This prevents Docker agents
// from getting stuck on the interactive theme picker that Claude Code shows
// on first run.
func SeedClaudeSettings(volumeDir string) error {
	settingsPath := filepath.Join(volumeDir, "settings.json")

	// Don't overwrite existing settings — the agent may have customized them.
	if _, err := os.Stat(settingsPath); err == nil {
		return nil
	}

	data, err := json.Marshal(defaultClaudeSettings)
	if err != nil {
		return err
	}

	return os.WriteFile(settingsPath, data, 0600)
}
