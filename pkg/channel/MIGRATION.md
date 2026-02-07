# Channel Storage Migration: JSON to SQLite

## Overview

This document describes the migration from JSON-based channel storage (`channels.json`) to SQLite (`channels.db`).

## Current State (JSON)

**File:** `.bc/channels.json`

```json
[
  {
    "name": "general",
    "members": ["engineer-01", "engineer-02"],
    "history": [
      {"time": "2024-01-01T10:00:00Z", "sender": "engineer-01", "message": "Hello"}
    ]
  }
]
```

**Limitations:**
- Full file read/write on every operation
- No indexing - O(n) search
- History limited to 100 messages per channel
- No full-text search
- No mention tracking

## Target State (SQLite)

**File:** `.bc/channels.db`

**Tables:**
- `channels` - Channel metadata
- `channel_members` - Many-to-many membership
- `messages` - All messages with types
- `mentions` - @mention tracking
- `messages_fts` - Full-text search index

## Migration Strategy

### Phase 1: Dual-Write (Backward Compatible)

1. On startup, check for `channels.json`
2. If exists, migrate data to SQLite
3. Write to both JSON and SQLite during transition
4. Read from SQLite

```go
func (s *Store) Load() error {
    // Try SQLite first
    if s.hasSQLite() {
        return s.loadFromSQLite()
    }

    // Fall back to JSON and migrate
    if s.hasJSON() {
        if err := s.loadFromJSON(); err != nil {
            return err
        }
        return s.migrateToSQLite()
    }

    // Fresh start with SQLite
    return s.initSQLite()
}
```

### Phase 2: Migration Function

```go
func (s *Store) migrateToSQLite() error {
    // 1. Create SQLite database
    db, err := sql.Open("sqlite3", s.dbPath)
    if err != nil {
        return err
    }

    // 2. Execute schema
    if _, err := db.Exec(schemaSQL); err != nil {
        return err
    }

    // 3. Migrate channels
    for _, ch := range s.channels {
        // Insert channel
        result, err := db.Exec(
            "INSERT INTO channels (name, type) VALUES (?, ?)",
            ch.Name, "group",
        )
        channelID := result.LastInsertId()

        // Insert members
        for _, member := range ch.Members {
            db.Exec(
                "INSERT INTO channel_members (channel_id, agent_id) VALUES (?, ?)",
                channelID, member,
            )
        }

        // Insert history
        for _, entry := range ch.History {
            db.Exec(
                "INSERT INTO messages (channel_id, sender, content, type, created_at) VALUES (?, ?, ?, ?, ?)",
                channelID, entry.Sender, entry.Message, "text", entry.Time,
            )
        }
    }

    // 4. Rename JSON file to .bak
    os.Rename(s.jsonPath, s.jsonPath+".migrated")

    return nil
}
```

### Phase 3: Remove JSON Support

After confirming SQLite works reliably:
1. Remove JSON read/write code
2. Delete `.bc/channels.json.migrated` files
3. Update documentation

## Data Mapping

| JSON Field | SQLite Table.Column | Notes |
|------------|---------------------|-------|
| `name` | `channels.name` | Direct map |
| `members[]` | `channel_members.agent_id` | One row per member |
| `history[].time` | `messages.created_at` | ISO 8601 format |
| `history[].sender` | `messages.sender` | Direct map |
| `history[].message` | `messages.content` | Direct map |
| (new) | `messages.type` | Default 'text' for migrated |

## Rollback Plan

If issues occur:
1. SQLite operations wrapped in transactions
2. Original JSON preserved as `.migrated`
3. Can restore by renaming `.migrated` back to `.json`

## Testing

1. **Unit tests:** Migration function with sample JSON
2. **Integration test:** Full cycle - create JSON, migrate, verify SQLite
3. **Edge cases:**
   - Empty channels
   - Channels with no history
   - Unicode content
   - Large history (>100 messages)

## Timeline

1. **Week 1:** Schema design (this PR)
2. **Week 2:** SQLite Store implementation (#73)
3. **Week 3:** Migration code and testing
4. **Week 4:** Dual-write deployment
5. **Week 5+:** Monitor and remove JSON support
