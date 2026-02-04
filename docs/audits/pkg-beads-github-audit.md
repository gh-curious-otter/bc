# Audit Report: pkg/beads and pkg/github

**Date:** 2026-02-05
**Auditor:** worker-01
**Bead:** bc-34b.5
**Files Reviewed:** `pkg/beads/beads.go`, `pkg/github/github.go`

---

## Executive Summary

Both packages provide CLI wrapper integrations but suffer from **pervasive silent error swallowing**, making debugging impossible. They are approximately 50-60% complete with key operations missing. **Zero test coverage** compounds these issues.

---

## Part 1: pkg/beads/beads.go

### Issues Found

#### 1. HIGH: Silent Error Swallowing in ListIssues

**Severity:** High

**Description:** `ListIssues` silently returns `nil` when the `bd` command fails, with no logging or error propagation. Callers cannot distinguish between "no issues" and "command failed".

**Code:**
```go
// beads.go:42-48
cmd := exec.Command("bd", "list", "--json")
cmd.Dir = workspacePath
output, err := cmd.Output()
if err != nil {
    return nil  // Silent failure! No logging, no error returned
}
```

**Impact:**
- If `bd` is not installed, users see empty issue lists with no explanation
- If `bd` crashes, failures are invisible
- Makes debugging extremely difficult

**Recommendation:** Return `([]Issue, error)` and propagate errors:
```go
func ListIssues(workspacePath string) ([]Issue, error) {
    if !HasBeads(workspacePath) {
        return nil, nil  // Explicit: no beads, no issues
    }
    // ...
    if err != nil {
        return nil, fmt.Errorf("bd list failed: %w", err)
    }
}
```

---

#### 2. HIGH: Silent Error Swallowing in ReadyIssues

**Severity:** High

**Description:** Same pattern as `ListIssues` - command errors and JSON parse errors both silently return `nil`.

**Code:**
```go
// beads.go:100-109
output, err := cmd.Output()
if err != nil {
    return nil  // Silent!
}

var issues []Issue
if err := json.Unmarshal(output, &issues); err != nil {
    return nil  // Silent!
}
```

**Impact:** Ready issue detection fails silently, causing work queue to appear empty.

---

#### 3. MEDIUM: Partial Data on JSONL Parse Error

**Severity:** Medium

**Description:** `parseJSONL` stops on first error and returns partial results without indicating failure.

**Code:**
```go
// beads.go:72-81
for dec.More() {
    var issue Issue
    if err := dec.Decode(&issue); err != nil {
        break  // Stops silently, returns partial data
    }
    // ...
}
return issues  // May be incomplete
```

**Impact:** Corrupted JSONL files return partial data, masking data integrity issues.

**Recommendation:** Either return error on parse failure or log a warning with count of skipped records.

---

#### 4. MEDIUM: AddIssue Loses Error Context

**Severity:** Medium

**Description:** `AddIssue` returns `cmd.Run()` error directly, which loses stderr output that would explain why the command failed.

**Code:**
```go
// beads.go:84-92
func AddIssue(workspacePath, title, description string) error {
    // ...
    cmd := exec.Command("bd", args...)
    cmd.Dir = workspacePath
    return cmd.Run()  // Loses stderr output
}
```

**Recommendation:** Use `cmd.CombinedOutput()` and wrap error with output:
```go
output, err := cmd.CombinedOutput()
if err != nil {
    return fmt.Errorf("bd add failed: %w\n%s", err, output)
}
```

---

#### 5. LOW: No Validation of Issue Fields

**Severity:** Low

**Description:** Issue struct accepts any value for `Priority` field (`any` type), which may cause issues downstream.

**Code:**
```go
// beads.go:22
Priority     any      `json:"priority,omitempty"`
```

**Impact:** Type assertions may panic if priority has unexpected type.

---

### Missing Features (pkg/beads)

| Feature | Status | Priority |
|---------|--------|----------|
| `GetIssue(id string)` | Missing | High |
| `UpdateIssue(id string, fields...)` | Missing | High |
| `StartIssue(id string)` | Missing | Medium |
| `BlockIssue(id, reason string)` | Missing | Medium |
| `AddDependency(from, to string)` | Missing | Medium |
| `RemoveDependency(from, to string)` | Missing | Low |
| `CommentOnIssue(id, comment string)` | Missing | Low |
| `SearchIssues(query string)` | Missing | Low |
| `GetDependencyGraph()` | Missing | Low |
| Error logging/tracing | Missing | High |

---

## Part 2: pkg/github/github.go

### Issues Found

#### 1. HIGH: Silent Error Swallowing in ListIssues

**Severity:** High

**Description:** Same pattern as beads - command failures and JSON parse errors return `nil` silently.

**Code:**
```go
// github.go:63-76
cmd := exec.Command("gh", "issue", "list", ...)
output, err := cmd.Output()
if err != nil {
    return nil  // Silent! gh auth issues invisible
}

var raw []ghIssue
if err := json.Unmarshal(output, &raw); err != nil {
    return nil  // Silent!
}
```

**Impact:**
- `gh` not installed: empty list, no error
- `gh` not authenticated: empty list, no error
- API rate limiting: empty list, no error

---

#### 2. HIGH: Silent Error Swallowing in ListPRs

**Severity:** High

**Description:** Identical issue to `ListIssues`.

**Code:**
```go
// github.go:102-115
output, err := cmd.Output()
if err != nil {
    return nil  // Silent!
}
// ...
if err := json.Unmarshal(output, &raw); err != nil {
    return nil  // Silent!
}
```

---

#### 3. MEDIUM: CreateIssue Loses Error Context

**Severity:** Medium

**Description:** Same as beads `AddIssue` - error lacks stderr context.

**Code:**
```go
// github.go:132-140
func CreateIssue(workspacePath, title, body string) error {
    // ...
    return cmd.Run()  // Loses stderr
}
```

---

#### 4. MEDIUM: Hardcoded Limit of 50 Items

**Severity:** Medium

**Description:** Both `ListIssues` and `ListPRs` hardcode `--limit 50`, which may miss issues/PRs in active repositories.

**Code:**
```go
// github.go:63-66
cmd := exec.Command("gh", "issue", "list",
    "--json", "number,title,state,labels",
    "--limit", "50",  // Hardcoded!
)
```

**Impact:** Large repos will have incomplete data with no indication items were truncated.

**Recommendation:** Make limit configurable or implement pagination.

---

#### 5. LOW: HasGitRemote Only Checks 'origin'

**Severity:** Low

**Description:** `HasGitRemote` only checks for `origin` remote, which may not exist if repo uses different remote name.

**Code:**
```go
// github.go:50-54
func HasGitRemote(workspacePath string) bool {
    cmd := exec.Command("git", "remote", "get-url", "origin")
    // ...
}
```

**Recommendation:** Check for any GitHub remote, not just origin.

---

### Missing Features (pkg/github)

| Feature | Status | Priority |
|---------|--------|----------|
| `GetIssue(number int)` | Missing | High |
| `GetPR(number int)` | Missing | High |
| `CloseIssue(number int)` | Missing | High |
| `CreatePR(...)` | Missing | High |
| `UpdateIssue(number int, ...)` | Missing | Medium |
| `MergePR(number int)` | Missing | Medium |
| `ApprovePR(number int)` | Missing | Medium |
| `RequestChanges(number int, comment)` | Missing | Medium |
| `CommentOnIssue(number int, body)` | Missing | Medium |
| `CommentOnPR(number int, body)` | Missing | Medium |
| `AddLabels(number int, labels...)` | Missing | Low |
| `RemoveLabels(number int, labels...)` | Missing | Low |
| `AssignIssue(number int, user)` | Missing | Low |
| `ListComments(number int)` | Missing | Low |
| Pagination support | Missing | Medium |
| Error logging/tracing | Missing | High |

---

## CLI Integration Analysis

Both packages are used in several places:

| Location | beads Usage | github Usage |
|----------|-------------|--------------|
| `internal/cmd/home.go` | `HasBeads`, `ListIssues` | - |
| `internal/cmd/up.go` | `ReadyIssues`, `ListIssues` | - |
| `internal/cmd/queue.go` | `ReadyIssues`, `ListIssues` | - |
| `internal/tui/workspace.go` | `ListIssues` | `ListPRs` |

**Problem:** All call sites assume functions succeed and handle empty returns, but cannot detect failures.

Example from `internal/cmd/up.go:70-73`:
```go
issues := beads.ReadyIssues(ws.RootDir)
if len(issues) == 0 {
    issues = beads.ListIssues(ws.RootDir)  // Fallback, also may fail silently
}
```

---

## Summary Table

### pkg/beads

| # | Severity | Issue | Line(s) |
|---|----------|-------|---------|
| 1 | High | Silent error in ListIssues | 46-48 |
| 2 | High | Silent error in ReadyIssues | 103-109 |
| 3 | Medium | Partial data on JSONL error | 74-76 |
| 4 | Medium | AddIssue loses stderr | 91 |
| 5 | Low | Untyped Priority field | 22 |

### pkg/github

| # | Severity | Issue | Line(s) |
|---|----------|-------|---------|
| 1 | High | Silent error in ListIssues | 69-76 |
| 2 | High | Silent error in ListPRs | 108-115 |
| 3 | Medium | CreateIssue loses stderr | 139 |
| 4 | Medium | Hardcoded limit of 50 | 65, 104 |
| 5 | Low | Only checks 'origin' remote | 51 |

---

## Recommendations Priority

### Immediate (Before Production)
1. Change all list functions to return `([]T, error)` instead of `[]T`
2. Update all call sites to handle errors (can log and continue)
3. Add stderr capture to all mutation operations

### Short-term
4. Add unit tests with mocked CLI output
5. Implement `GetIssue`/`GetPR` for single-item lookups
6. Make limits configurable

### Medium-term
7. Implement missing CRUD operations
8. Add pagination support for large repos
9. Add optional logging/tracing

---

## Test Coverage Gaps

Both packages have **zero tests**. Minimum coverage should include:

### pkg/beads
1. `TestListIssues_Success` - Parse valid JSON
2. `TestListIssues_JSONL` - Parse JSONL format
3. `TestListIssues_CommandError` - Handle bd failure
4. `TestListIssues_NoBeadsDir` - No .beads directory
5. `TestReadyIssues_FiltersEpics` - Epic filtering
6. `TestAddIssue_WithDescription` - Description flag
7. `TestAssignIssue_CommandBuilding` - Verify args

### pkg/github
1. `TestListIssues_Success` - Parse valid JSON
2. `TestListIssues_NoRemote` - No git remote
3. `TestListIssues_AuthError` - gh not authenticated
4. `TestListPRs_Success` - Parse PR JSON
5. `TestListPRs_LabelExtraction` - Label array handling
6. `TestCreateIssue_WithBody` - Body flag
7. `TestHasGitRemote_NoOrigin` - Missing origin

---

## Appendix: Error Handling Pattern Recommendation

Both packages should adopt a consistent error handling pattern:

```go
// Option 1: Return errors (preferred for libraries)
func ListIssues(workspacePath string) ([]Issue, error) {
    if !HasBeads(workspacePath) {
        return nil, nil  // Explicit no-op
    }

    cmd := exec.Command("bd", "list", "--json")
    cmd.Dir = workspacePath
    output, err := cmd.CombinedOutput()
    if err != nil {
        return nil, fmt.Errorf("bd list: %w\noutput: %s", err, output)
    }
    // ...
}

// Option 2: Accept logger (for graceful degradation)
func ListIssues(workspacePath string, log func(string)) []Issue {
    // ...
    if err != nil {
        if log != nil {
            log(fmt.Sprintf("bd list failed: %v", err))
        }
        return nil
    }
}
```
