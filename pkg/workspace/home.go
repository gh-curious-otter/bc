package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// BCHome returns the global bc home directory (~/.bc).
// Respects BC_HOME env var override.
func BCHome() (string, error) {
	if env := os.Getenv("BC_HOME"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".bc"), nil
}

// GlobalStateDir returns the state directory for a workspace.
// Path: ~/.bc/workspaces/<workspace-id>/
// The workspace ID is the first 6 hex chars of SHA256(absRootDir).
// Respects BC_STATE_DIR env var override.
func GlobalStateDir(rootDir string) (string, error) {
	if env := os.Getenv("BC_STATE_DIR"); env != "" {
		return env, nil
	}

	absRoot, err := filepath.Abs(rootDir)
	if err != nil {
		return "", err
	}

	bcHome, err := BCHome()
	if err != nil {
		return "", err
	}

	id := workspaceID(absRoot)
	return filepath.Join(bcHome, "workspaces", id), nil
}

// workspaceID returns a short hash of the workspace root path.
// Uses first 6 hex chars of SHA256 — same format as existing wsID().
func workspaceID(absRoot string) string {
	h := sha256.Sum256([]byte(absRoot))
	return hex.EncodeToString(h[:3]) // 6 hex chars
}

// EnsureBCHome creates the global ~/.bc directory structure if it doesn't exist.
func EnsureBCHome() error {
	bcHome, err := BCHome()
	if err != nil {
		return err
	}
	dirs := []string{
		bcHome,
		filepath.Join(bcHome, "workspaces"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create %s: %w", dir, err)
		}
	}
	return nil
}
