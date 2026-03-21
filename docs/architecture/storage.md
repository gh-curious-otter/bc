# Storage & Data Architecture

## Filesystem Layout

bc stores all state globally under `~/.bc/` (not per-project):

```
~/.bc/
  bc.db                     # Main SQLite database (all tables)
  settings.toml             # Global settings
  secret-key                # AES-256 encryption key (0600 perms)
  agents/
    alice/
      auth/.claude/         # Provider auth (Claude settings, sessions)
      auth/.claude.json     # Provider config
    bob/
      auth/.claude/
  logs/
    alice.log               # Agent session logs
    bob.log
    daemon-bcdb.log         # Daemon process logs
```

Each agent has an associated **workspace** — a path to a git repo. The agent's worktree lives inside that repo at `.claude/worktrees/<project>-<agent>/`.

## Database Schema

```mermaid
erDiagram
    teams {
        text id PK
        text name UK
        text parent_id FK
        text description
        text created_at
        text updated_at
    }

    agents {
        text name PK
        text role FK
        text team_id FK
        text state
        text task
        text tool
        text workspace_path
        text worktree_dir
        text session
        text session_id
        text runtime_backend
        text created_at
        text started_at
        text updated_at
        text stopped_at
    }

    roles {
        text name PK
        text description
        text parent_role FK
        text prompt
        text settings_json
        text created_at
        text updated_at
    }

    channels {
        integer id PK
        text name UK
        text type
        text description
        text created_at
        text updated_at
    }

    channel_members {
        integer id PK
        integer channel_id FK
        text agent_name FK
        text joined_at
    }

    messages {
        integer id PK
        integer channel_id FK
        text sender
        text content
        text type
        text metadata
        text created_at
    }

    cost_records {
        integer id PK
        text agent_name FK
        text team_id FK
        text model
        text session_id
        integer input_tokens
        integer output_tokens
        integer cache_creation_tokens
        integer cache_read_tokens
        real cost_usd
        text timestamp
    }

    secrets {
        text name PK
        text value
        text description
        text created_at
        text updated_at
    }

    events {
        integer id PK
        text type
        text agent
        text message
        text data_json
        text timestamp
    }

    cron_jobs {
        text name PK
        text schedule
        text agent FK
        text prompt
        integer enabled
        integer run_count
        text last_run
        text created_at
    }

    mcp_servers {
        text name PK
        text transport
        text command
        text url
        text args_json
        text env_json
        integer enabled
    }

    tools {
        text name PK
        text command
        text install_hint
        integer enabled
    }

    teams ||--o{ teams : "parent"
    teams ||--o{ agents : "contains"
    roles ||--o{ agents : "assigned"
    roles ||--o{ roles : "inherits"
    channels ||--o{ channel_members : "has"
    agents ||--o{ channel_members : "member_of"
    channels ||--o{ messages : "contains"
    agents ||--o{ cost_records : "incurs"
    teams ||--o{ cost_records : "aggregates"
    cron_jobs ||--o| agents : "targets"
```

## Key Design Decisions

### Teams (replacing workspaces)

Teams are hierarchical groups that organize agents as a tree:

```mermaid
graph TD
    ROOT[Engineering Team] --> BACKEND[Backend Team]
    ROOT --> FRONTEND[Frontend Team]
    ROOT --> INFRA[Infrastructure Team]
    BACKEND --> B1[alice - engineer]
    BACKEND --> B2[bob - engineer]
    FRONTEND --> F1[carol - engineer]
    INFRA --> I1[dave - engineer]
```

- A team can contain agents or other teams
- Agents always belong to exactly one team
- Cost aggregation rolls up the tree
- Channel membership can be team-scoped

### Roles in Database

Roles are stored in the `roles` table (not filesystem). Each role defines:
- Prompt template (CLAUDE.md content)
- Settings JSON (model, permissions)
- Parent role (BFS inheritance)
- Associated MCP servers and secrets (via join tables)

CRUD via `POST/GET/PUT/DELETE /api/roles`.

### Workspace Association

Each agent has a `workspace_path` column pointing to a git repository. The agent's worktree is created inside that repo. This replaces the old per-project `.bc/` directory.

## SQLite Configuration

| Pragma | Value | Purpose |
|--------|-------|---------|
| `journal_mode` | WAL | Concurrent reads during writes |
| `foreign_keys` | ON | Referential integrity |
| `busy_timeout` | 30000ms | Handle concurrent agent access |
| `synchronous` | NORMAL | Performance (WAL makes this safe) |
| `cache_size` | -2000 (2MB) | Page cache |
| `temp_store` | MEMORY | Temp tables in RAM |
| `mmap_size` | 256MB | Memory-mapped I/O |

Connection pool: `MaxOpenConns=1`, `MaxIdleConns=1` (SQLite single-writer model).

## Secret Encryption

Secrets are encrypted at rest using AES-256-GCM:

```mermaid
graph LR
    PASS[Passphrase<br/>BC_SECRET_PASSPHRASE<br/>or ~/.bc/secret-key] --> PBKDF2[PBKDF2-SHA256<br/>600k iterations]
    SALT[Random 16-byte salt] --> PBKDF2
    PBKDF2 --> KEY[256-bit AES key]
    KEY --> GCM[AES-256-GCM]
    NONCE[Random nonce] --> GCM
    PLAIN[Secret value] --> GCM
    GCM --> CIPHER[base64 ciphertext<br/>stored in DB]
```

The key file (`~/.bc/secret-key`) is auto-generated with `0600` permissions on first use.

## Cost Data Pipeline

```mermaid
graph LR
    CLAUDE[Claude Code<br/>JSONL session files] --> IMPORT[Cost Importer<br/>every 5 minutes]
    IMPORT --> PARSE[Parse tokens<br/>+ model pricing]
    PARSE --> DB[(cost_records)]
    DB --> API[/api/costs/*]
    API --> WEB[Web UI<br/>Cost Dashboard]
```

The importer scans `~/.bc/agents/*/auth/.claude/` for session JSONL files, extracts token usage, applies model-specific pricing, and inserts records.

## Migration Path

```
OLD (per-project):                NEW (global):
  project/.bc/bc.db        ->     ~/.bc/bc.db
  project/.bc/config.toml  ->     ~/.bc/settings.toml
  project/.bc/agents/      ->     ~/.bc/agents/
  project/.bc/roles/*.md   ->     roles table in bc.db
  project/.bc/logs/        ->     ~/.bc/logs/
```

Migration tool: `bc migrate` — copies data from per-project `.bc/` to `~/.bc/`, converts role files to database records.