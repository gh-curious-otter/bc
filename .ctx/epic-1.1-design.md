# Epic 1.1: Workspace Restructure - Technical Design

**Status:** APPROVED (Manager decision)
**Target:** Complete v2 workspace structure with TOML config

---

## Design Decisions

### 1. TOML Schema (config.toml)

**Decision:** Full TOML configuration replacing config.json

```toml
# .bc/config.toml - bc v2 workspace configuration

[workspace]
name = "project-name"
version = 2                    # v2 format marker

[worktrees]
path = ".bc/worktrees"         # Relative to workspace root
auto_cleanup = true            # Remove worktrees when agents stop

[tools]
default = "claude"             # Default tool for new agents

[tools.claude]
command = "claude --dangerously-skip-permissions"
enabled = true

[tools.codex]
command = "codex"
enabled = false

[memory]
backend = "file"               # "file" or "mem0"
path = ".bc/memory"            # Relative to workspace root

[beads]
enabled = true                 # Hard dependency per vision
issues_dir = ".beads/issues"   # Beads issue directory

[channels]
default = ["general", "engineering", "leadership"]
persist_history = true
history_limit = 1000
```

**Rationale:**
- Multiple tools supported (expandable to codex, etc.)
- Worktree config centralized
- Beads config here (required dependency)
- Memory config here (backend selection)
- Channels config here (default channels)

### 2. Role Files (.bc/roles/)

**Decision:** Pure Markdown with optional YAML frontmatter

```markdown
---
# .bc/roles/engineer.md
name: engineer
capabilities:
  - implement_tasks
parent_roles:
  - manager
  - product-manager
---

# Engineer Role

You are an engineer agent in the bc multi-agent system.

## Responsibilities
- Implement assigned tasks
- Write clean, tested code
- Report progress via `bc report`
- Submit completed work for merge

## Available Commands
- `bc report working "task description"` - Report current work
- `bc report done "summary"` - Report task completion
- `bc queue submit` - Submit work for merge review

## Guidelines
1. Work only in your assigned worktree
2. Create focused, single-purpose commits
3. Run tests before submitting
4. Communicate blockers promptly
```

**Required files:**
- `root.md` - Required, auto-generated if missing
- Other roles are optional (engineer.md, manager.md, qa.md)

**Rationale:**
- Frontmatter for machine-readable metadata
- Markdown body for LLM prompt injection
- Simple, human-editable format

### 3. Agent State Files

**Decision:** Per-agent JSON files in `.bc/agents/<name>.json`

```json
{
  "name": "engineer-01",
  "role": "engineer",
  "tool": "claude",
  "team": "frontend-team",
  "parent": "manager-atlas",
  "state": "working",
  "worktree": ".bc/worktrees/engineer-01",
  "work_queue": [
    {
      "id": "issue-123",
      "title": "Fix login bug",
      "status": "working",
      "assigned_by": "manager-atlas",
      "branch": "fix/login-bug",
      "created_at": "2024-01-15T10:00:00Z"
    }
  ],
  "merge_queue": [],
  "started_at": "2024-01-15T09:00:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

**Special file:** `.bc/agents/root.json`
```json
{
  "name": "root",
  "role": "root",
  "state": "idle",
  "is_singleton": true,
  "merge_queue": [...],
  "children": ["manager-atlas", "pm-furiosa"],
  "started_at": "...",
  "updated_at": "..."
}
```

**Rationale:**
- Per-agent files avoid lock contention
- Atomic file operations for safety
- Work/merge queues embedded per-agent
- Clear separation of concerns

### 4. Migration Detection

**Decision:** Warn but don't migrate

On `bc init`:
1. Check for existing `.bc/` directory
2. If `config.json` exists (v1), show warning:
   ```
   Warning: Existing v1 workspace detected.
   bc v2 is a clean break - v1 data will not be migrated.

   To proceed:
   - Backup .bc/ if needed
   - Remove .bc/ directory
   - Run `bc init` again
   ```
3. Do not auto-migrate or backup

**Rationale:**
- Clean break per vision
- User controls their data
- No silent data manipulation

---

## Directory Structure

```
.bc/
├── config.toml              # Workspace configuration
├── roles/
│   ├── root.md              # Root agent role (required)
│   ├── manager.md           # Optional
│   ├── engineer.md          # Optional
│   └── qa.md                # Optional
├── agents/
│   ├── root.json            # Root singleton state
│   ├── manager-atlas.json   # Per-agent state
│   └── engineer-01.json     # Per-agent state
├── memory/
│   └── <agent-name>/        # Per-agent memory (Epic 2.4)
├── worktrees/
│   └── <agent-name>/        # Git worktrees
├── channels/
│   └── <channel-name>.jsonl # Channel history
├── bin/
│   └── git                  # Git wrapper script
└── events.jsonl             # Event log (keep from v1)
```

---

## Implementation Tasks

### Task 1.1.1: TOML Config Schema
- Define Go structs for config.toml
- Use github.com/BurntSushi/toml
- Add validation functions
- Write tests

### Task 1.1.2: Role File Management
- Implement role file loading
- Parse YAML frontmatter
- Auto-generate default root.md
- Write tests

### Task 1.1.3: Per-Agent State Files
- Migrate from single agents.json to per-agent files
- Implement atomic read/write
- Add locking for concurrent access
- Write tests

### Task 1.1.4: Update `bc init` Command
- Detect v1 workspace and warn
- Create new directory structure
- Generate default config.toml
- Generate default root.md
- Write tests

### Task 1.1.5: Update Workspace Loading
- Load config.toml instead of config.json
- Support role file discovery
- Per-agent state loading
- Backward compatibility warning (not migration)
- Write tests

---

## Dependencies

- github.com/BurntSushi/toml (TOML parsing)
- gopkg.in/yaml.v3 (frontmatter parsing)

---

## Success Criteria

- [ ] `bc init` creates v2 structure
- [ ] `config.toml` fully functional
- [ ] Role files loaded and injected
- [ ] Per-agent state files working
- [ ] v1 detection with clear warning
- [ ] All tests passing
