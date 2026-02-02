# Data Models Documentation

This document describes all data structures and storage formats used by Gas Town (gt) and Beads (bd) for agent orchestration and issue tracking.

## Table of Contents

1. [Configuration Files](#1-configuration-files)
2. [Runtime State](#2-runtime-state)
3. [Event Log](#3-event-log)
4. [Beads Integration](#4-beads-integration)
5. [Mail System](#5-mail-system)

---

## 1. Configuration Files

### 1.1 town.json

Located at: `{town_root}/mayor/town.json`

Defines the top-level "town" configuration - the root of the Gas Town hierarchy.

```json
{
  "type": "town",
  "version": 2,
  "name": ".gt",
  "owner": "user@example.com",
  "public_name": ".gt",
  "created_at": "2026-01-13T03:02:36.498577+05:30"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "town" |
| `version` | int | Schema version (current: 2) |
| `name` | string | Internal town name |
| `owner` | string | Owner's email address |
| `public_name` | string | Display name |
| `created_at` | string | ISO 8601 timestamp |

### 1.2 rigs.json

Located at: `{town_root}/mayor/rigs.json`

Registry of all rigs (projects) managed by the town.

```json
{
  "version": 1,
  "rigs": {
    "beacon": {
      "git_url": "/Users/user/Projects/beacon",
      "added_at": "2026-01-30T16:06:15.888181+05:30",
      "beads": {
        "repo": "",
        "prefix": "be"
      }
    },
    "imx": {
      "git_url": "/Users/user/Projects/imx",
      "added_at": "2026-01-30T16:06:24.202454+05:30",
      "beads": {
        "repo": "",
        "prefix": "imx"
      }
    }
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `version` | int | Schema version (current: 1) |
| `rigs` | object | Map of rig name to rig entry |
| `rigs[name].git_url` | string | Path or URL to git repository |
| `rigs[name].added_at` | string | ISO 8601 timestamp |
| `rigs[name].beads.repo` | string | Optional beads repository path |
| `rigs[name].beads.prefix` | string | Issue ID prefix (e.g., "be" for "be-123") |

### 1.3 overseer.json

Located at: `{town_root}/mayor/overseer.json`

Identifies the human operator (overseer) of the town.

```json
{
  "type": "overseer",
  "version": 1,
  "name": "Puneet Rai",
  "email": "puneetrai04@gmail.com",
  "username": "puneetrai04",
  "source": "git-config"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "overseer" |
| `version` | int | Schema version (current: 1) |
| `name` | string | Human name |
| `email` | string | Email address |
| `username` | string | Username |
| `source` | string | Where info was obtained (e.g., "git-config") |

### 1.4 config.json (per-rig)

Located at: `{town_root}/{rig}/config.json`

Configuration for an individual rig.

```json
{
  "type": "rig",
  "version": 1,
  "name": "beacon",
  "git_url": "/Users/user/Projects/beacon",
  "default_branch": "main",
  "created_at": "2026-01-30T16:06:13.03266+05:30",
  "beads": {
    "prefix": "be"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "rig" |
| `version` | int | Schema version (current: 1) |
| `name` | string | Rig name |
| `git_url` | string | Path or URL to git repository |
| `default_branch` | string | Default branch name |
| `created_at` | string | ISO 8601 timestamp |
| `beads.prefix` | string | Issue ID prefix |

### 1.5 escalation.json

Located at: `{town_root}/settings/escalation.json`

Defines escalation routes for different priority levels.

```json
{
  "type": "escalation",
  "version": 1,
  "routes": {
    "critical": ["bead", "mail:mayor", "email:human", "sms:human"],
    "high": ["bead", "mail:mayor", "email:human"],
    "medium": ["bead", "mail:mayor"],
    "low": ["bead"]
  },
  "contacts": {},
  "stale_threshold": "4h",
  "max_reescalations": 2
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "escalation" |
| `version` | int | Schema version (current: 1) |
| `routes` | object | Map of priority level to escalation actions |
| `contacts` | object | Contact information for escalation |
| `stale_threshold` | string | Duration before issue is considered stale |
| `max_reescalations` | int | Maximum re-escalation attempts |

**Escalation Actions:**
- `bead` - Create/update a beads issue
- `mail:mayor` - Send mail to mayor agent
- `email:human` - Send email to human operator
- `sms:human` - Send SMS to human operator

### 1.6 Beads config.yaml

Located at: `{town_root}/.beads/config.yaml`

Configuration for the beads database.

```yaml
# Issue prefix for this repository
# issue-prefix: ""

# Use no-db mode: load from JSONL, no SQLite
# no-db: false

# Disable daemon for RPC communication
# no-daemon: false

# Disable auto-flush of database to JSONL
# no-auto-flush: false

# Disable auto-import from JSONL when newer
# no-auto-import: false

# Enable JSON output by default
# json: false

# Default actor for audit trails
# actor: ""

# Path to database
# db: ""

# Auto-start daemon if not running
# auto-start-daemon: true

# Debounce interval for auto-flush
# flush-debounce: "5s"

# Git branch for beads commits
# sync-branch: "beads-sync"
```

---

## 2. Runtime State

### 2.1 witness.json

Located at: `{town_root}/{rig}/.runtime/witness.json`

Tracks the state of the Witness agent (monitors polecat health and progress).

```json
{
  "rig_name": "beacon",
  "state": "running",
  "started_at": "2026-01-30T16:06:30.170211+05:30",
  "config": {
    "max_workers": 0,
    "spawn_delay_ms": 0,
    "auto_spawn": false
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `rig_name` | string | Name of the rig being monitored |
| `state` | string | Agent state: "stopped", "running", "paused" |
| `started_at` | string | ISO 8601 timestamp of last start |
| `config.max_workers` | int | Maximum concurrent workers (0 = unlimited) |
| `config.spawn_delay_ms` | int | Delay between spawning workers |
| `config.auto_spawn` | bool | Whether to auto-spawn polecats |

### 2.2 refinery.json

Located at: `{town_root}/{rig}/.runtime/refinery.json`

Tracks the state of the Refinery agent (processes merge queue).

```json
{
  "rig_name": "beacon",
  "state": "running",
  "started_at": "2026-01-30T16:06:35.505476+05:30"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `rig_name` | string | Name of the rig |
| `state` | string | Agent state: "stopped", "running", "paused" |
| `started_at` | string | ISO 8601 timestamp of last start |

### 2.3 Agent State Values

The agent state machine uses these values:

| State | Description |
|-------|-------------|
| `stopped` | Agent is not running |
| `running` | Agent is actively operating |
| `paused` | Agent is paused (not operating but not stopped) |

---

## 3. Event Log

### 3.1 .events.jsonl Format

Located at: `{town_root}/.events.jsonl`

Append-only log of all events in the town. Uses JSON Lines format (one JSON object per line).

```jsonl
{"ts":"2026-01-12T22:27:57Z","source":"gt","type":"session_start","actor":"mayor","payload":{...},"visibility":"feed"}
{"ts":"2026-01-12T23:01:57Z","source":"gt","type":"spawn","actor":"gt","payload":{...},"visibility":"feed"}
{"ts":"2026-01-12T23:03:19Z","source":"gt","type":"sling","actor":"mayor","payload":{...},"visibility":"feed"}
```

### 3.2 Event Base Structure

```json
{
  "ts": "2026-01-12T22:27:57Z",
  "source": "gt",
  "type": "session_start",
  "actor": "mayor",
  "payload": {},
  "visibility": "feed"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ts` | string | ISO 8601 timestamp (UTC) |
| `source` | string | Event source (always "gt") |
| `type` | string | Event type (see below) |
| `actor` | string | Who triggered the event |
| `payload` | object | Event-specific data |
| `visibility` | string | Visibility level: "feed", "private" |

### 3.3 Event Types

#### session_start

Emitted when an agent session starts.

```json
{
  "ts": "2026-01-12T22:27:57Z",
  "source": "gt",
  "type": "session_start",
  "actor": "mayor",
  "payload": {
    "actor_pid": "mayor-91023",
    "cwd": "/Users/user/Projects/.gt/mayor",
    "role": "mayor",
    "session_id": "mayor-91023",
    "topic": "patrol"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `actor_pid` | string | Unique process identifier |
| `cwd` | string | Working directory |
| `role` | string | Agent role (e.g., "mayor", "beacon/witness") |
| `session_id` | string | Session identifier |
| `topic` | string | Optional topic/purpose |

#### spawn

Emitted when a polecat (worker) is spawned.

```json
{
  "ts": "2026-01-12T23:01:57Z",
  "source": "gt",
  "type": "spawn",
  "actor": "gt",
  "payload": {
    "polecat": "furiosa",
    "rig": "beacon"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `polecat` | string | Polecat name |
| `rig` | string | Rig name |

#### sling

Emitted when work is assigned ("slung") to a polecat.

```json
{
  "ts": "2026-01-12T23:03:19Z",
  "source": "gt",
  "type": "sling",
  "actor": "mayor",
  "payload": {
    "bead": "be-c1a",
    "target": "beacon/polecats/furiosa"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `bead` | string | Bead/issue ID being assigned |
| `target` | string | Target polecat path |

#### nudge

Emitted when an agent is nudged (prompted to take action).

```json
{
  "ts": "2026-01-12T23:19:16Z",
  "source": "gt",
  "type": "nudge",
  "actor": "mayor",
  "payload": {
    "reason": "[from mayor] Are you still working? Reply with your current status.",
    "rig": "beacon",
    "target": "beacon/nux"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `reason` | string | Nudge message/reason |
| `rig` | string | Rig name |
| `target` | string | Target agent |

#### done

Emitted when a polecat completes its work.

```json
{
  "ts": "2026-01-12T23:04:17Z",
  "source": "gt",
  "type": "done",
  "actor": "beacon/furiosa",
  "payload": {
    "bead": "furiosa-mkbrp1ke",
    "branch": "polecat/furiosa-mkbrp1ke"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `bead` | string | Work item identifier |
| `branch` | string | Git branch with the work |

#### merge

Emitted when work is merged.

```json
{
  "ts": "2026-01-12T23:10:00Z",
  "source": "gt",
  "type": "merge",
  "actor": "beacon/refinery",
  "payload": {
    "mr": "be-abc",
    "branch": "polecat/furiosa-mkbrp1ke",
    "commit": "abc123"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `mr` | string | Merge request ID |
| `branch` | string | Source branch |
| `commit` | string | Merge commit SHA |

#### mail

Emitted when mail is sent between agents.

```json
{
  "ts": "2026-01-12T23:36:11Z",
  "source": "gt",
  "type": "mail",
  "actor": "beacon/refinery",
  "payload": {
    "subject": "MR pending - branch not pushed",
    "to": "beacon/polecat/furiosa"
  },
  "visibility": "feed"
}
```

| Payload Field | Type | Description |
|---------------|------|-------------|
| `subject` | string | Mail subject |
| `to` | string | Recipient address |

---

## 4. Beads Integration

### 4.1 Issue/Bead Structure

Issues are stored in `.beads/issues.jsonl` and the SQLite database (beads.db).

```json
{
  "id": "be-0js",
  "title": "Update app entry point with production DI",
  "description": "dispatched_by: mayor\n\nIn BeaconApp.swift, wire up SignalManager...",
  "status": "closed",
  "priority": 2,
  "issue_type": "task",
  "assignee": "beacon/polecats/dementus",
  "owner": "user@example.com",
  "created_at": "2026-01-13T04:13:15.8983+05:30",
  "created_by": "mayor",
  "updated_at": "2026-01-13T09:35:05.520198+05:30",
  "closed_at": "2026-01-13T09:35:05.520198+05:30",
  "close_reason": "Merged: 30126b4",
  "labels": ["gt:task"],
  "dependencies": [
    {
      "issue_id": "be-0js",
      "depends_on_id": "be-avh",
      "type": "blocks",
      "created_at": "2026-01-13T04:13:25.776107+05:30",
      "created_by": "mayor"
    }
  ],
  "ephemeral": false
}
```

### 4.2 Issue Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique identifier (prefix + base62 hash) |
| `title` | string | Brief summary |
| `description` | string | Full description (may contain structured fields) |
| `status` | string | Issue status (see Status Values) |
| `priority` | int | Priority level 0-4 (see Priority Values) |
| `issue_type` | string | Issue type (see Issue Types) |
| `assignee` | string | Assigned agent identity |
| `owner` | string | Issue owner email |
| `created_at` | string | ISO 8601 creation timestamp |
| `created_by` | string | Creator identity |
| `updated_at` | string | ISO 8601 last update timestamp |
| `closed_at` | string | ISO 8601 close timestamp (if closed) |
| `close_reason` | string | Reason for closing |
| `parent` | string | Parent issue ID (for hierarchies) |
| `children` | []string | Child issue IDs |
| `depends_on` | []string | Issues this depends on |
| `blocks` | []string | Issues blocked by this |
| `blocked_by` | []string | Issues blocking this |
| `labels` | []string | Labels/tags |
| `dependencies` | []object | Detailed dependency records |
| `ephemeral` | bool | If true, not exported to JSONL (wisp) |
| `hook_bead` | string | For agent beads: current hooked work |
| `agent_state` | string | For agent beads: lifecycle state |
| `deleted_at` | string | Soft delete timestamp |
| `deleted_by` | string | Who deleted |
| `delete_reason` | string | Why deleted |
| `original_type` | string | Type before deletion |

### 4.3 Status Values

| Status | Description |
|--------|-------------|
| `open` | Available for work |
| `in_progress` | Being worked on |
| `closed` | Completed |
| `hooked` | Attached to an agent's hook |
| `tombstone` | Soft-deleted |

### 4.4 Priority Values

| Priority | Level | Description |
|----------|-------|-------------|
| 0 | Urgent | Critical, immediate attention |
| 1 | High | Important, soon |
| 2 | Normal | Default priority |
| 3 | Low | Not urgent |
| 4 | Backlog | Future work |

### 4.5 Issue Types

| Type | Label | Description |
|------|-------|-------------|
| task | `gt:task` | Single unit of work |
| epic | `gt:epic` | Collection of related work |
| molecule | `gt:molecule` | Workflow with steps |
| agent | `gt:agent` | Agent identity bead |
| message | `gt:message` | Mail message |
| event | `gt:event` | System event |

### 4.6 Agent Bead Structure

Agent beads track agent identity and state.

```json
{
  "id": "be-beacon-polecat-furiosa",
  "title": "be-beacon-polecat-furiosa",
  "description": "be-beacon-polecat-furiosa\n\nrole_type: polecat\nrig: beacon\nagent_state: spawning\nhook_bead: be-c1a\nrole_bead: hq-polecat-role\ncleanup_status: has_unpushed\nactive_mr: be-9cn\nnotification_level: null",
  "status": "open",
  "priority": 2,
  "issue_type": "agent",
  "labels": ["gt:agent"]
}
```

**Agent Fields (in description):**

| Field | Description |
|-------|-------------|
| `role_type` | Agent role: polecat, witness, refinery, crew, mayor, deacon |
| `rig` | Rig the agent belongs to |
| `agent_state` | State: idle, spawning, working, done, stuck |
| `hook_bead` | Currently hooked work item |
| `role_bead` | Reference to role definition |
| `cleanup_status` | Cleanup state: clean, has_unpushed, null |
| `active_mr` | Active merge request ID |
| `notification_level` | Notification preference |

### 4.7 Merge Request Fields

Merge requests are issues with structured fields in the description.

```json
{
  "id": "be-0ap",
  "title": "Merge: slit-mkwqzx53",
  "description": "branch: polecat/slit-mkwqzx53\ntarget: main\nsource_issue: slit-mkwqzx53\nrig: beacon\nagent_bead: be-beacon-polecat-slit\nretry_count: 0\nlast_conflict_sha: null\nconflict_task_id: null",
  "status": "closed",
  "labels": ["gt:merge-request"],
  "ephemeral": true
}
```

**MR Fields:**

| Field | Description |
|-------|-------------|
| `branch` | Source branch name |
| `target` | Target branch (e.g., "main") |
| `source_issue` | Work item being merged |
| `worker` | Who did the work |
| `rig` | Which rig |
| `merge_commit` | SHA of merge commit |
| `close_reason` | Reason: merged, rejected, conflict, superseded |
| `agent_bead` | Agent bead ID that created this MR |
| `retry_count` | Number of conflict-resolution cycles |
| `last_conflict_sha` | SHA of main when conflict occurred |
| `conflict_task_id` | Link to conflict-resolution task |
| `convoy_id` | Parent convoy ID (if part of convoy) |
| `convoy_created_at` | Convoy creation time |

### 4.8 Dependency Types

| Type | Description |
|------|-------------|
| `blocks` | This issue blocks another |
| `parent-child` | Hierarchical relationship |

### 4.9 Routes File

Located at: `{town_root}/.beads/routes.jsonl`

Maps issue prefixes to directories.

```jsonl
{"prefix":"hq-","path":"."}
{"prefix":"be-","path":"beacon"}
{"prefix":"bi-","path":"bitchat"}
{"prefix":"se-","path":"sense"}
{"prefix":"imx-","path":"imx"}
```

| Field | Type | Description |
|-------|------|-------------|
| `prefix` | string | Issue ID prefix |
| `path` | string | Relative path from town root |

---

## 5. Mail System

### 5.1 Message Structure

Messages are stored as beads issues with type "message".

```json
{
  "id": "hq-3lp",
  "title": "READY_WORK: P1 be-0ue + be-18l",
  "description": "Ready tasks:\n- be-0ue [P1]: Automate polecat lifecycle management\n- be-18l [P2]: Create GitHub Actions build workflow\n\n2 polecats running, slots available.",
  "status": "open",
  "priority": 2,
  "issue_type": "message",
  "assignee": "mayor/",
  "owner": "user@example.com",
  "created_at": "2026-01-13T08:31:36.5151+05:30",
  "created_by": "beacon/witness",
  "labels": ["from:beacon/witness", "thread:thread-890beb2019e7"],
  "ephemeral": true
}
```

### 5.2 Message Types

| Type | Description |
|------|-------------|
| `task` | Message requiring action |
| `scavenge` | Optional first-come-first-served work |
| `notification` | Informational message (default) |
| `reply` | Response to another message |

### 5.3 Priority Levels

| Priority | Int Value | Description |
|----------|-----------|-------------|
| urgent | 0 | Immediate attention required |
| high | 1 | Important, soon |
| normal | 2 | Default |
| low | 3 | Not urgent |

### 5.4 Delivery Modes

| Mode | Description |
|------|-------------|
| queue | Message in mailbox for periodic checking |
| interrupt | Inject directly into agent's session |

### 5.5 Message Labels

Messages use labels to store metadata:

| Label Prefix | Description | Example |
|--------------|-------------|---------|
| `from:` | Sender identity | `from:beacon/witness` |
| `thread:` | Thread identifier | `thread:thread-abc123` |
| `reply-to:` | Parent message ID | `reply-to:msg-def456` |
| `msg-type:` | Message type | `msg-type:task` |
| `cc:` | CC recipient | `cc:mayor/` |
| `queue:` | Queue name | `queue:merge-queue` |
| `channel:` | Channel name | `channel:announcements` |
| `claimed-by:` | Who claimed queue message | `claimed-by:beacon/furiosa` |
| `claimed-at:` | Claim timestamp | `claimed-at:2026-01-13T08:31:36Z` |

### 5.6 Routing and Addressing

**Address Formats:**

| Format | Description | Example |
|--------|-------------|---------|
| `overseer` | Human operator | `overseer` |
| `mayor/` | Town mayor | `mayor/` |
| `deacon/` | Town deacon | `deacon/` |
| `{rig}/` | Rig broadcast | `beacon/` |
| `{rig}/{agent}` | Rig-level agent | `beacon/witness` |
| `{rig}/{name}` | Named worker (normalized) | `beacon/Toast` |

**Normalization Rules:**
- `rig/polecats/name` normalizes to `rig/name`
- `rig/crew/name` normalizes to `rig/name`
- Town-level agents (mayor, deacon) always have trailing slash

### 5.7 Queue Messages

Queue messages are not addressed to a specific recipient but claimed by eligible agents.

```json
{
  "id": "msg-abc123",
  "from": "beacon/witness",
  "queue": "merge-queue",
  "subject": "Ready for merge",
  "body": "Branch polecat/furiosa ready",
  "claimed_by": "beacon/refinery",
  "claimed_at": "2026-01-13T08:31:36Z"
}
```

### 5.8 Channel Messages

Channel messages are broadcast to all subscribers.

```json
{
  "id": "msg-def456",
  "from": "mayor/",
  "channel": "announcements",
  "subject": "System maintenance",
  "body": "Planned downtime at 2am UTC"
}
```

---

## JSON Schema Examples

### Complete Issue Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "id": { "type": "string" },
    "title": { "type": "string" },
    "description": { "type": "string" },
    "status": {
      "type": "string",
      "enum": ["open", "in_progress", "closed", "hooked", "tombstone"]
    },
    "priority": { "type": "integer", "minimum": 0, "maximum": 4 },
    "issue_type": { "type": "string" },
    "assignee": { "type": "string" },
    "owner": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" },
    "created_by": { "type": "string" },
    "updated_at": { "type": "string", "format": "date-time" },
    "closed_at": { "type": "string", "format": "date-time" },
    "close_reason": { "type": "string" },
    "parent": { "type": "string" },
    "children": { "type": "array", "items": { "type": "string" } },
    "depends_on": { "type": "array", "items": { "type": "string" } },
    "blocks": { "type": "array", "items": { "type": "string" } },
    "blocked_by": { "type": "array", "items": { "type": "string" } },
    "labels": { "type": "array", "items": { "type": "string" } },
    "dependencies": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "issue_id": { "type": "string" },
          "depends_on_id": { "type": "string" },
          "type": { "type": "string", "enum": ["blocks", "parent-child"] },
          "created_at": { "type": "string", "format": "date-time" },
          "created_by": { "type": "string" }
        }
      }
    },
    "ephemeral": { "type": "boolean" },
    "hook_bead": { "type": "string" },
    "agent_state": { "type": "string" }
  },
  "required": ["id", "title", "status", "priority"]
}
```

### Complete Event Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "ts": { "type": "string", "format": "date-time" },
    "source": { "type": "string", "const": "gt" },
    "type": {
      "type": "string",
      "enum": ["session_start", "spawn", "sling", "nudge", "done", "merge", "mail"]
    },
    "actor": { "type": "string" },
    "payload": { "type": "object" },
    "visibility": { "type": "string", "enum": ["feed", "private"] }
  },
  "required": ["ts", "source", "type", "actor", "payload", "visibility"]
}
```

### Complete Message Schema

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "id": { "type": "string" },
    "from": { "type": "string" },
    "to": { "type": "string" },
    "subject": { "type": "string" },
    "body": { "type": "string" },
    "timestamp": { "type": "string", "format": "date-time" },
    "read": { "type": "boolean" },
    "priority": {
      "type": "string",
      "enum": ["urgent", "high", "normal", "low"]
    },
    "type": {
      "type": "string",
      "enum": ["task", "scavenge", "notification", "reply"]
    },
    "delivery": {
      "type": "string",
      "enum": ["queue", "interrupt"]
    },
    "thread_id": { "type": "string" },
    "reply_to": { "type": "string" },
    "pinned": { "type": "boolean" },
    "wisp": { "type": "boolean" },
    "cc": { "type": "array", "items": { "type": "string" } },
    "queue": { "type": "string" },
    "channel": { "type": "string" },
    "claimed_by": { "type": "string" },
    "claimed_at": { "type": "string", "format": "date-time" }
  },
  "required": ["id", "from", "subject"]
}
```
