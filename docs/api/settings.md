# Settings API

The Settings API provides endpoints to read and update workspace configuration via the bcd HTTP server.

Base URL: `http://localhost:9374`

## Endpoints

### GET /api/settings

Returns the full workspace configuration.

**Response:** `200 OK`

```json
{
  "workspace": {
    "name": "my-project",
    "version": 2,
    "path": ""
  },
  "user": {
    "nickname": "@alice"
  },
  "providers": {
    "default": "claude",
    "claude": {
      "command": "claude --dangerously-skip-permissions",
      "enabled": true,
      "env": {}
    },
    "gemini": {
      "command": "gemini --yolo",
      "enabled": true,
      "env": {}
    }
  },
  "runtime": {
    "backend": "docker",
    "docker": {
      "image": "",
      "network": "",
      "extra_mounts": null,
      "cpus": 0,
      "memory_mb": 0
    }
  },
  "logs": {
    "path": ".bc/logs",
    "max_bytes": 1048576
  },
  "performance": {
    "poll_interval_agents": 2000,
    "poll_interval_channels": 3000,
    "poll_interval_costs": 5000,
    "poll_interval_status": 2000,
    "poll_interval_logs": 3000,
    "poll_interval_teams": 10000,
    "poll_interval_demons": 5000,
    "cache_ttl_tmux": 2000,
    "cache_ttl_commands": 5000,
    "adaptive_fast_interval": 1000,
    "adaptive_normal_interval": 2000,
    "adaptive_slow_interval": 4000,
    "adaptive_max_interval": 8000
  },
  "tui": {
    "theme": "dark",
    "mode": "auto"
  },
  "env": {},
  "roster": {
    "agents": []
  },
  "services": {},
  "server": {
    "addr": "127.0.0.1:9374",
    "cors_origin": "*"
  },
  "scheduler": {
    "tick_interval": 60,
    "job_timeout": 300
  },
  "storage": {
    "sqlite_path": ".bc/bc.db"
  }
}
```

**Error:** `500 Internal Server Error` if no config is loaded.

```json
{
  "error": "no config loaded"
}
```

---

### PUT /api/settings

Partial update of the full configuration. Send only the sections you want to change. Unspecified sections remain unchanged.

The merged config is validated before saving. On success, the updated config is persisted to `.bc/settings.toml`.

**Request body:** JSON object with one or more config sections.

Supported sections: `user`, `providers`, `env`, `logs`, `runtime`, `performance`, `tui`, `workspace`, `roster`, `services`.

**Example: Update user nickname and TUI theme**

```bash
curl -X PUT http://localhost:9374/api/settings \
  -H "Content-Type: application/json" \
  -d '{
    "user": { "nickname": "@bob" },
    "tui": { "theme": "synthwave", "mode": "dark" }
  }'
```

**Example: Update runtime backend**

```bash
curl -X PUT http://localhost:9374/api/settings \
  -H "Content-Type: application/json" \
  -d '{
    "runtime": { "backend": "tmux" }
  }'
```

**Response:** `200 OK` with the full updated config (same schema as GET).

**Errors:**

| Status | Description |
|--------|-------------|
| `400`  | Invalid JSON, invalid section data, or validation failure |
| `500`  | Config not loaded or failed to save |

```json
{
  "error": "validation failed: workspace.name is required"
}
```

---

### PATCH /api/settings/{section}

Update a single config section. The request body is the section object directly (not wrapped in a parent key).

**Supported sections:** `user`, `tui`, `runtime`, `providers`, `services`, `logs`, `performance`, `env`, `roster`.

**Example: Update TUI theme**

```bash
curl -X PATCH http://localhost:9374/api/settings/tui \
  -H "Content-Type: application/json" \
  -d '{ "theme": "matrix", "mode": "dark" }'
```

**Example: Update performance polling**

```bash
curl -X PATCH http://localhost:9374/api/settings/performance \
  -H "Content-Type: application/json" \
  -d '{ "poll_interval_agents": 5000 }'
```

**Example: Update user nickname**

```bash
curl -X PATCH http://localhost:9374/api/settings/user \
  -H "Content-Type: application/json" \
  -d '{ "nickname": "@charlie" }'
```

**Example: Set environment variables**

```bash
curl -X PATCH http://localhost:9374/api/settings/env \
  -H "Content-Type: application/json" \
  -d '{ "GITHUB_TOKEN": "ghp_...", "CUSTOM_VAR": "value" }'
```

**Response:** `200 OK` with the full updated config.

**Errors:**

| Status | Description |
|--------|-------------|
| `400`  | Missing section name, unknown section, invalid JSON, or validation failure |
| `405`  | Method not allowed (only PATCH is supported on section endpoints) |
| `500`  | Config not loaded or failed to save |

```json
{
  "error": "unknown section: invalid"
}
```

## Error Format

All error responses use a consistent JSON format:

```json
{
  "error": "description of the error"
}
```

## Validation

Both PUT and PATCH endpoints validate the merged configuration before saving. Validation rules include:

- `workspace.name` is required
- `workspace.version` must be `2`
- `providers.default` must reference a defined provider or service
- Poll intervals must be at least 500ms
- Cache TTLs must be between 100ms and 60,000ms
- TUI theme must be one of: `dark`, `light`, `matrix`, `synthwave`, `high-contrast`
- TUI mode must be one of: `auto`, `dark`, `light`
- User nickname must start with `@`, be 15 chars or less, alphanumeric and underscores only

If validation fails, no changes are saved and a `400` error is returned with the validation message.
