# TUI Package Audit

## Overview

Audit of `pkg/tui/` and `pkg/tui/runtime/` for bead bc-34b.8.

## pkg/tui - Core TUI Components

### What's Implemented

| Component | Status | Notes |
|-----------|--------|-------|
| TableView | Complete | Builder pattern, vim navigation, selection callbacks, column alignment, status styling |
| App | Complete | Multi-view container, view navigation, global bindings, header/status bar |
| Types | Complete | Model interface, Row, Column, KeyBinding, Section, Field types |
| Style/Theme | Complete | Ayu-inspired dark theme, pre-built lipgloss styles |
| Keys | Complete | Helper for Enter key detection across terminal types |

### What's Missing

| Component | Documented | Implemented |
|-----------|------------|-------------|
| DetailView | Yes (doc.go:39) | No - types exist but no builder/component |
| ListView | Yes (doc.go:40) | No - not implemented |
| FormView | Yes (doc.go:41) | No - not implemented |
| ModalView | Yes (doc.go:42) | No - not implemented in main pkg |
| Vertical Layout | Yes (doc.go:47) | No |
| Horizontal Layout | Yes (doc.go:48) | No |
| Split Layout | Yes (doc.go:49) | No |

### Test Coverage

- `app_test.go`: Builder, multi-view, navigation, bindings - adequate
- `table_test.go`: Builder, navigation, key handling, selection, empty state - adequate

### Code Quality

- Clean builder pattern with fluent API
- Good separation of concerns
- Types properly documented
- Enter key handling accounts for Cursor/embedded terminals (keys.go:7-19)

## pkg/tui/runtime - Streaming Protocol Runtime

### What's Implemented

| Component | Status | Notes |
|-----------|--------|-------|
| Protocol | Complete | JSON message types for AI↔TUI communication |
| Renderer | Complete | Table, Detail, Modal rendering with styling |
| Driver | Complete | Bubble Tea model with stdin/stdout communication |

### Protocol Design

The protocol is well-designed for AI-driven UIs:

**AI → TUI Messages:**
- `view` - Create/switch views (table, detail, form, modal, list)
- `set` - Set values at JSON paths
- `append` - Append to arrays
- `delete` - Remove paths
- `done` - Signal batch completion
- `error` - Signal errors

**TUI → AI Events:**
- `key` - Key press with view context
- `select` - Row selection
- `input` - Text input submitted
- `ready` - TUI ready with dimensions
- `init` - Initial handshake

### Issues Found

1. **JSON Path Limitations** (renderer.go:284)
   ```go
   // Simple path setting (no array indexing yet)
   ```
   `SetPath` and `AppendPath` don't handle indexed paths like `rows[0].status`.

2. **Simplified Append Logic** (driver.go:294-298)
   ```go
   // Simplified: just append to first section's fields
   if len(d.detailSpec.Sections) == 0 {
       d.detailSpec.Sections = []SectionSpec{{}}
   }
   ```
   Section/field appending always targets first section.

3. **Missing Tests**
   - No tests for `renderer.go`
   - No tests for `driver.go`
   - Only `protocol_test.go` has coverage

4. **Driver readInput** (driver.go:381-391)
   Creates new Scanner on each call - could be optimized to reuse scanner state.

### Protocol Correctness

The protocol appears correct for the implemented views:
- Message types properly serialized/deserialized
- Bidirectional communication works
- Loading states supported for streaming
- Key events include selection context

## Recommendations

### High Priority
1. Add tests for `renderer.go` and `driver.go`
2. Implement full JSON path handling in SetPath/AppendPath
3. Either implement DetailView/ListView/FormView or remove from doc.go

### Medium Priority
1. Fix section targeting in handleAppendMessage
2. Optimize Driver scanner reuse
3. Add integration test for full protocol flow

### Low Priority
1. Implement layout components if needed
2. Add modal support to main tui package (currently only in runtime)

## Files Reviewed

- `pkg/tui/doc.go` - Package documentation
- `pkg/tui/types.go` - Core type definitions
- `pkg/tui/app.go` - App container
- `pkg/tui/table.go` - TableView component
- `pkg/tui/keys.go` - Key handling utilities
- `pkg/tui/app_test.go` - App tests
- `pkg/tui/table_test.go` - Table tests
- `pkg/tui/style/theme.go` - Theme and styles
- `pkg/tui/runtime/protocol.go` - Protocol definitions
- `pkg/tui/runtime/renderer.go` - Spec renderer
- `pkg/tui/runtime/driver.go` - Runtime driver
- `pkg/tui/runtime/protocol_test.go` - Protocol tests
