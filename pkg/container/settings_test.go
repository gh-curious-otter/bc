package container

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSeedClaudeSettings_CreatesFile(t *testing.T) {
	dir := t.TempDir()

	if err := SeedClaudeSettings(dir); err != nil {
		t.Fatalf("SeedClaudeSettings() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "settings.json")) //nolint:gosec // test uses temp dir
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings.json: %v", err)
	}

	if settings["theme"] != "dark" {
		t.Errorf("theme = %v, want %q", settings["theme"], "dark")
	}
	if settings["skipDangerousModePermissionPrompt"] != true {
		t.Errorf("skipDangerousModePermissionPrompt = %v, want true", settings["skipDangerousModePermissionPrompt"])
	}
}

func TestSeedClaudeSettings_PreservesExisting(t *testing.T) {
	dir := t.TempDir()

	existing := []byte(`{"theme":"light","custom":"value"}`)
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, existing, 0600); err != nil { //nolint:gosec // test uses temp dir
		t.Fatalf("failed to write existing settings: %v", err)
	}

	if err := SeedClaudeSettings(dir); err != nil {
		t.Fatalf("SeedClaudeSettings() error = %v", err)
	}

	data, err := os.ReadFile(settingsPath) //nolint:gosec // test uses temp dir
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("failed to parse settings.json: %v", err)
	}

	// Existing user values must be preserved
	if settings["theme"] != "light" {
		t.Errorf("theme = %v, want %q (should preserve user value)", settings["theme"], "light")
	}
	if settings["custom"] != "value" {
		t.Errorf("custom = %v, want %q (should preserve user value)", settings["custom"], "value")
	}
}
