# Audit Report: pkg/workspace

**Package:** `pkg/workspace/workspace.go`, `pkg/workspace/registry.go`
**Auditor:** worker-03
**Date:** 2026-02-05
**Work Item:** work-009, bead bc-34b.4

## Summary

The `pkg/workspace` package provides workspace discovery, initialization, and a global registry of workspaces. It currently has **zero tests**. This audit identified **5 issues** ranging from low to high severity.

---

## Issue 1: Registry Prune Not Persisted in home.go

**Severity:** HIGH

**Description:**
In `internal/cmd/home.go`, `Prune()` is called but `Save()` is only called conditionally, meaning pruned entries (deleted workspaces) won't be persisted to disk.

**Problematic Code:**
```go
// internal/cmd/home.go:40-53
func runHome(cmd *cobra.Command, args []string) error {
    reg, err := workspace.LoadRegistry()
    if err != nil {
        return fmt.Errorf("failed to load workspace registry: %w", err)
    }
    reg.Prune()  // Removes deleted workspaces from memory

    // If no workspaces registered, try to register the current one
    if len(reg.Workspaces) == 0 {
        ws, err := getWorkspace()
        if err == nil {
            reg.Register(ws.RootDir, ws.Config.Name)
            reg.Save()  // Only saves here! Prune results lost otherwise
        }
    }
    // ... continues without Save() if workspaces exist
}
```

**Impact:**
Deleted workspaces will reappear in the list every time `bc home` is run until the user happens to have zero registered workspaces.

**Recommendation:**
Always save after pruning:
```go
reg.Prune()
if err := reg.Save(); err != nil {
    // Log warning, don't fail
}
```

---

## Issue 2: Init() Overwrites Existing Config if Called Directly

**Severity:** MEDIUM

**Description:**
While the CLI command checks `IsWorkspace()` before calling `Init()`, the `Init()` function itself does not guard against overwriting an existing configuration.

**Problematic Code:**
```go
// pkg/workspace/workspace.go:42-72
func Init(rootDir string) (*Workspace, error) {
    absRoot, err := filepath.Abs(rootDir)
    // ...

    // Create state directory (MkdirAll is safe)
    stateDir := filepath.Join(absRoot, ".bc")
    if err := os.MkdirAll(stateDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create state directory: %w", err)
    }

    // Create default config
    config := DefaultConfig(absRoot)

    // Save config - OVERWRITES existing!
    configPath := filepath.Join(stateDir, "config.json")
    // ...
    if err := os.WriteFile(configPath, data, 0644); err != nil {  // No existence check
        return nil, err
    }
```

**Risk:**
Direct callers of `Init()` (e.g., future code, tests, other tools) may accidentally overwrite user configuration.

**Recommendation:**
Add an existence check in `Init()`:
```go
func Init(rootDir string) (*Workspace, error) {
    absRoot, err := filepath.Abs(rootDir)
    if err != nil {
        return nil, err
    }

    // Check for existing workspace
    if IsWorkspace(absRoot) {
        return nil, fmt.Errorf("workspace already exists at %s", absRoot)
    }
    // ... rest of init
}
```

---

## Issue 3: GlobalDir() Silently Returns Empty String on Error

**Severity:** MEDIUM

**Description:**
`GlobalDir()` returns an empty string if `os.UserHomeDir()` fails, which can lead to attempting to read/write files at invalid paths.

**Problematic Code:**
```go
// pkg/workspace/registry.go:24-31
func GlobalDir() string {
    home, err := os.UserHomeDir()
    if err != nil {
        return ""  // Silent failure
    }
    return filepath.Join(home, ".bc")
}

// RegistryPath returns the path to ~/.bc/workspaces.json.
func RegistryPath() string {
    return filepath.Join(GlobalDir(), "workspaces.json")  // Could be "workspaces.json" (relative!)
}
```

**Impact:**
If `UserHomeDir()` fails (rare but possible in containers/chroots), `RegistryPath()` returns `"workspaces.json"` - a relative path that would write to the current directory.

**Recommendation:**
Return an error or use a fallback:
```go
func GlobalDir() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("cannot determine home directory: %w", err)
    }
    return filepath.Join(home, ".bc"), nil
}
```

---

## Issue 4: Registry Path Comparison Not Normalized

**Severity:** LOW

**Description:**
Registry operations compare paths with simple string equality. Paths like `/foo/bar` and `/foo/bar/` or symlinked paths would be treated as different workspaces.

**Problematic Code:**
```go
// pkg/workspace/registry.go:77-83
func (r *Registry) Register(path, name string) {
    for i, w := range r.Workspaces {
        if w.Path == path {  // Simple string comparison
            r.Workspaces[i].Name = name
            r.Workspaces[i].LastAccessed = now
            return
        }
    }
    // ...
}
```

**Impact:**
- Same workspace could be registered multiple times with slightly different paths
- Trailing slashes, symlinks, or relative vs absolute paths cause duplicates

**Recommendation:**
Normalize paths before comparison:
```go
func normalizePath(p string) string {
    abs, err := filepath.Abs(p)
    if err != nil {
        return filepath.Clean(p)
    }
    // Optional: resolve symlinks with filepath.EvalSymlinks
    return filepath.Clean(abs)
}
```

---

## Issue 5: No Concurrent Access Protection for Registry

**Severity:** LOW

**Description:**
The Registry has no file locking. If multiple `bc` processes access the registry simultaneously (e.g., two terminals running `bc init`), data could be lost.

**Problematic Code:**
```go
// pkg/workspace/registry.go:59-71
func (r *Registry) Save() error {
    dir := filepath.Dir(r.path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return err
    }

    data, err := json.MarshalIndent(r, "", "  ")
    if err != nil {
        return err
    }

    return os.WriteFile(r.path, data, 0644)  // No locking, not atomic
}
```

**Impact:**
Race condition: Process A reads registry, Process B reads registry, A writes, B writes (clobbering A's changes).

**Recommendation:**
Use atomic write pattern (write to temp file, then rename):
```go
func (r *Registry) Save() error {
    // ...
    tmpPath := r.path + ".tmp"
    if err := os.WriteFile(tmpPath, data, 0644); err != nil {
        return err
    }
    return os.Rename(tmpPath, r.path)  // Atomic on POSIX
}
```

---

## Positive Findings

### Find() Upward Search is Correct

The upward directory search is implemented correctly:

```go
// pkg/workspace/workspace.go:108-130
func Find(dir string) (*Workspace, error) {
    absDir, err := filepath.Abs(dir)
    if err != nil {
        return nil, err
    }

    current := absDir
    for {
        stateDir := filepath.Join(current, ".bc")
        if _, err := os.Stat(stateDir); err == nil {
            return Load(current)
        }

        parent := filepath.Dir(current)
        if parent == current {  // Correct root detection
            return nil, fmt.Errorf("no workspace found (searched from %s to root)", absDir)
        }
        current = parent
    }
}
```

- Uses `filepath.Abs` to normalize the starting directory
- Correctly detects filesystem root when `parent == current`
- Returns informative error message

### Init() Directory Creation is Robust

- Uses `os.MkdirAll` which is idempotent (handles existing directories)
- Sets appropriate permissions (0755)

### Registry Pruning Has No Memory Leaks

```go
// pkg/workspace/registry.go:114-126
func (r *Registry) Prune() int {
    pruned := 0
    valid := make([]RegistryEntry, 0, len(r.Workspaces))  // New slice with capacity hint
    for _, w := range r.Workspaces {
        if IsWorkspace(w.Path) {
            valid = append(valid, w)
        } else {
            pruned++
        }
    }
    r.Workspaces = valid  // Old slice can be GC'd
    return pruned
}
```

The old slice is properly replaced, allowing garbage collection.

### Registry IS Used

The registry is actively used in:
- `internal/cmd/init.go:62-65` - Registers new workspaces on init
- `internal/cmd/home.go:40-51` - Lists workspaces in TUI, prunes, auto-registers current

---

## Summary Table

| Issue | Severity | Effort to Fix |
|-------|----------|---------------|
| Prune not persisted in home.go | HIGH | Low |
| Init() overwrites existing config | MEDIUM | Low |
| GlobalDir() silent empty string | MEDIUM | Low |
| Path comparison not normalized | LOW | Medium |
| No concurrent access protection | LOW | Medium |

---

## Recommended Priority

1. **Immediate:** Fix `home.go` to save after pruning
2. **Immediate:** Add existence check to `Init()`
3. **Short-term:** Handle `GlobalDir()` error properly
4. **Short-term:** Normalize paths in registry operations
5. **Ongoing:** Add comprehensive tests covering edge cases
