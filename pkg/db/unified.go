package db

import "path/filepath"

// BCDBPath returns the path to the unified bc database for the given workspace root.
// All packages use this single file instead of separate per-package databases.
func BCDBPath(workspaceRoot string) string {
	return filepath.Join(workspaceRoot, ".bc", "bc.db")
}
