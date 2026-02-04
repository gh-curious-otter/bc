# pkg/tmux Audit Report

**Auditor:** worker-02
**Date:** 2026-02-05
**File:** `pkg/tmux/session.go`
**Test Coverage:** 0%

---

## Executive Summary

The `pkg/tmux` package provides tmux session management for agent orchestration. While functional, several issues were identified ranging from potential race conditions to incomplete error handling. The most critical issues involve concurrent buffer access and potential shell injection.

---

## Issues Found

### 1. Race Condition in SendKeys Buffer (Severity: HIGH)

**Location:** `session.go:160-168`

The `SendKeys` function uses tmux's global buffer (`load-buffer` / `paste-buffer`) without any locking. If two goroutines call `SendKeys` concurrently for long messages, they will overwrite each other's buffer content.

```go
// Thread A loads its content
loadCmd := exec.Command("tmux", "load-buffer", tmpPath)  // line 160

// Thread B loads its content (overwrites A's buffer)
// ...

// Thread A pastes - but gets Thread B's content!
pasteCmd := exec.Command("tmux", "paste-buffer", "-t", fullName)  // line 165
```

**Recommendation:** Use named buffers with unique identifiers:
```go
bufferName := fmt.Sprintf("bc-%s-%d", name, time.Now().UnixNano())
loadCmd := exec.Command("tmux", "load-buffer", "-b", bufferName, tmpPath)
pasteCmd := exec.Command("tmux", "paste-buffer", "-b", bufferName, "-d", "-t", fullName)
```

---

### 2. Ignored Error from MkdirAll (Severity: MEDIUM)

**Location:** `session.go:146`

```go
tmpDir := filepath.Join(os.TempDir(), "bc-tmux")
os.MkdirAll(tmpDir, 0700)  // Error ignored!
tmpFile, err := os.CreateTemp(tmpDir, "send-*.txt")
```

If `MkdirAll` fails (permissions, disk full), `CreateTemp` will fail with a confusing error about the directory not existing.

**Recommendation:**
```go
if err := os.MkdirAll(tmpDir, 0700); err != nil {
    return fmt.Errorf("failed to create temp dir: %w", err)
}
```

---

### 3. Magic Sleep in SendKeys (Severity: MEDIUM)

**Location:** `session.go:171`

```go
// Wait for paste to complete before sending Enter
time.Sleep(500 * time.Millisecond)
```

This is a timing-based workaround that:
- Adds 500ms latency to every long message
- May not be sufficient on slow systems
- Is unnecessarily long on fast systems

**Recommendation:** Remove the sleep. `paste-buffer` is synchronous; the Enter key can be sent immediately. If there's a real race, investigate the actual cause.

---

### 4. Potential Shell Injection in Environment Variables (Severity: MEDIUM)

**Location:** `session.go:98-103`

```go
for k, v := range env {
    parts = append(parts, fmt.Sprintf("export %s=%q;", k, v))
}
```

While `%q` provides Go string quoting, the environment variable **key** (`k`) is unquoted. A malicious key like `FOO;rm -rf /;X` would execute arbitrary commands.

```go
// Malicious input:
env := map[string]string{"FOO;rm -rf /;X": "value"}
// Produces:
"export FOO;rm -rf /;X=\"value\";"
```

**Recommendation:** Validate environment variable keys:
```go
if !regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`).MatchString(k) {
    return fmt.Errorf("invalid env var key: %s", k)
}
```

---

### 5. Workspace Hash Collision Risk (Severity: LOW)

**Location:** `session.go:42-43`

```go
h := sha256.Sum256([]byte(workspacePath))
return &Manager{
    SessionPrefix: prefix,
    workspaceHash: fmt.Sprintf("%x", h[:3]),  // Only 6 hex chars = 24 bits
}
```

With only 24 bits (16.7M combinations), the birthday paradox gives ~50% collision probability at ~5,000 workspaces. In practice this is unlikely to cause issues, but worth noting.

**Recommendation:** Consider using 4 bytes (8 hex chars) for ~4B combinations:
```go
workspaceHash: fmt.Sprintf("%x", h[:4]),
```

---

### 6. CreateSession Doesn't Check for Existing Session (Severity: LOW)

**Location:** `session.go:72-86`

The function relies on tmux's error for duplicate sessions. While functional, this results in unclear error messages.

```go
func (m *Manager) CreateSession(name, dir string) error {
    fullName := m.SessionName(name)
    // No HasSession() check here
    args := []string{"new-session", "-d", "-s", fullName}
```

**Recommendation:** Add explicit check for better UX:
```go
if m.HasSession(name) {
    return fmt.Errorf("session %s already exists", fullName)
}
```

---

### 7. ListSessions Doesn't Parse Windows Count (Severity: LOW)

**Location:** `session.go:234-239`

The `Windows` field is never populated despite being in the struct:

```go
sessions = append(sessions, Session{
    Name:      strings.TrimPrefix(name, fullPrefix),
    Created:   parts[1],
    Attached:  parts[2] == "1",
    // Windows:   missing!
    Directory: parts[4],
})
```

**Recommendation:**
```go
windows, _ := strconv.Atoi(parts[3])
sessions = append(sessions, Session{
    // ...
    Windows: windows,
})
```

---

### 8. SetEnvironment Has No Error Context (Severity: LOW)

**Location:** `session.go:274-278`

```go
func (m *Manager) SetEnvironment(name, key, value string) error {
    fullName := m.SessionName(name)
    cmd := exec.Command("tmux", "set-environment", "-t", fullName, key, value)
    return cmd.Run()  // Raw error, no context
}
```

**Recommendation:**
```go
output, err := cmd.CombinedOutput()
if err != nil {
    return fmt.Errorf("failed to set env %s in %s: %w (%s)", key, fullName, err, output)
}
return nil
```

---

### 9. Temp Files Not Cleaned on Crash (Severity: LOW)

**Location:** `session.go:152`

```go
defer os.Remove(tmpPath)
```

If the process is killed between file creation and completion, temp files accumulate in `/tmp/bc-tmux/`. Not critical but could waste disk space over time.

**Recommendation:** Add periodic cleanup or use a temp dir that gets cleaned on reboot:
```go
tmpDir := filepath.Join(os.TempDir(), "bc-tmux", fmt.Sprintf("%d", os.Getpid()))
```

---

## Summary Table

| # | Issue | Severity | Effort to Fix |
|---|-------|----------|---------------|
| 1 | Race condition in SendKeys buffer | HIGH | Low |
| 2 | Ignored MkdirAll error | MEDIUM | Trivial |
| 3 | Magic 500ms sleep | MEDIUM | Low |
| 4 | Shell injection via env key | MEDIUM | Low |
| 5 | Workspace hash collision risk | LOW | Trivial |
| 6 | No pre-check for existing session | LOW | Trivial |
| 7 | Windows count not parsed | LOW | Trivial |
| 8 | SetEnvironment lacks error context | LOW | Trivial |
| 9 | Temp file cleanup on crash | LOW | Low |

---

## Recommendations

1. **Add tests** - 0% coverage is unacceptable for production code
2. **Fix HIGH severity issues first** - especially the race condition
3. **Add input validation** - for session names and env var keys
4. **Consider adding a mutex** - if concurrent SendKeys calls are expected
