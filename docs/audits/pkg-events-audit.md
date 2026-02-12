# Audit Report: pkg/events

**Package:** `pkg/events/events.go`
**Auditor:** worker-03
**Date:** 2026-02-05

## Summary

The `pkg/events` package provides an append-only JSONL event log. It currently has **zero tests**. This audit identified **4 issues** ranging from medium to high severity.

---

## Issue 1: No Concurrency Protection for Writes

**Severity:** HIGH

**Description:**
The `Append` function relies solely on `O_APPEND` for write atomicity. While POSIX guarantees atomic appends for writes under `PIPE_BUF` (typically 4KB), there is no mutex or file locking to protect against:
- Multiple goroutines writing simultaneously from the same process
- Multiple processes (agents) writing to the same log file

**Problematic Code:**
```go
// pkg/events/events.go:50-68
func (l *Log) Append(event Event) error {
    // No mutex protection here
    f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer f.Close()

    data, err := json.Marshal(event)
    // ...
    _, err = f.Write(data)  // No file locking
    return err
}
```

**Risk:**
If the JSON event exceeds ~4KB (e.g., large `Data` map), writes could interleave, producing corrupt JSONL lines.

**Recommendation:**
Add a `sync.Mutex` for in-process protection and consider `syscall.Flock` for cross-process safety:

```go
type Log struct {
    path string
    mu   sync.Mutex
}

func (l *Log) Append(event Event) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    // ... existing code
}
```

---

## Issue 2: ReadLast Does Not Validate Input

**Severity:** MEDIUM

**Description:**
`ReadLast(n)` does not validate the `n` parameter. Passing `n <= 0` produces incorrect results.

**Problematic Code:**
```go
// pkg/events/events.go:94-103
func (l *Log) ReadLast(n int) ([]Event, error) {
    all, err := l.Read()
    if err != nil {
        return nil, err
    }
    if len(all) <= n {
        return all, nil
    }
    return all[len(all)-n:], nil  // Bug: n=0 returns empty, n<0 panics
}
```

**Edge Cases:**
| Input | Result |
|-------|--------|
| `n = 0` | Returns empty slice (arguably correct but should be explicit) |
| `n < 0` | **Runtime panic** - slice bounds out of range |

**Recommendation:**
```go
func (l *Log) ReadLast(n int) ([]Event, error) {
    if n <= 0 {
        return nil, nil  // Or return error for n < 0
    }
    // ... rest of implementation
}
```

---

## Issue 3: Unbounded Log Growth (No Rotation)

**Severity:** HIGH

**Description:**
The event log grows indefinitely with no rotation or truncation mechanism. Additionally, `Read()` loads the **entire file into memory**, which will cause memory exhaustion on large logs.

**Problematic Code:**
```go
// pkg/events/events.go:71-91
func (l *Log) Read() ([]Event, error) {
    // ...
    var events []Event  // Unbounded slice growth
    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        // Every event ever written is loaded into memory
        events = append(events, ev)
    }
    return events, scanner.Err()
}
```

**Usage Pattern (from codebase):**
- `internal/cmd/logs.go` - reads all events for display

**Risk:**
After extended use, the log file could grow to GB size, causing:
- `Read()` to OOM crash the process
- Slow startup times as every command reads the full log

**Recommendation:**
1. Add log rotation (e.g., keep last 10K events or 10MB)
2. Implement efficient `ReadLast` using reverse file reading instead of loading everything
3. Add a `Truncate(keepLast int)` method

```go
func (l *Log) Rotate(maxEvents int) error {
    events, err := l.ReadLast(maxEvents)
    if err != nil {
        return err
    }
    // Rewrite file with only recent events
    // Use atomic write (temp file + rename)
}
```

---

## Issue 4: Malformed Lines Silently Skipped

**Severity:** MEDIUM

**Description:**
When `Read()` encounters a malformed JSON line, it silently skips it with no logging or error reporting. This makes debugging data corruption invisible.

**Problematic Code:**
```go
// pkg/events/events.go:83-88
for scanner.Scan() {
    var ev Event
    if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
        continue // skip malformed lines - SILENT DATA LOSS
    }
    events = append(events, ev)
}
```

**Risk:**
- Corrupt data goes undetected
- Debugging issues becomes difficult
- If Issue 1 causes interleaved writes, those corrupted lines vanish silently

**Recommendation:**
Options (choose based on use case):
1. **Log a warning:** Add logging for skipped lines
2. **Return partial results with error count:**
```go
type ReadResult struct {
    Events   []Event
    Skipped  int
    Errors   []error
}
```
3. **Strict mode option:** Fail on first malformed line

---

## Additional Observations

### No Tests
The package has zero test coverage. Critical paths to test:
- Append atomicity under concurrent writes
- ReadLast edge cases (n=0, n<0, n>len)
- Malformed input handling
- File not found behavior
- Large event handling (>4KB)

### Inefficient ReadByAgent
`ReadByAgent()` reads the entire log to filter. For frequent queries, consider indexing or filtering at write time.

---

## Summary Table

| Issue | Severity | Effort to Fix |
|-------|----------|---------------|
| No concurrency protection | HIGH | Low |
| ReadLast input validation | MEDIUM | Low |
| Unbounded log growth | HIGH | Medium |
| Silent malformed line skip | MEDIUM | Low |

---

## Recommended Priority

1. **Immediate:** Add mutex for concurrency safety
2. **Immediate:** Add input validation to ReadLast
3. **Short-term:** Add log rotation mechanism
4. **Short-term:** Add warning logging for skipped lines
5. **Ongoing:** Add comprehensive tests
