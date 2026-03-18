package cost

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// Importer scans Claude Code JSONL session files and imports token usage into
// the cost.db store. It tracks which session files have been imported to avoid
// double-counting.
type Importer struct {
	store        *Store
	workspaceDir string
}

// NewImporter creates an Importer for the given workspace.
func NewImporter(store *Store, workspaceDir string) *Importer {
	return &Importer{store: store, workspaceDir: workspaceDir}
}

// ImportAll scans all known Claude projects directories and imports new sessions.
// It is safe to call repeatedly — already-imported sessions are skipped.
func (imp *Importer) ImportAll(ctx context.Context) (int, error) {
	dirs := imp.claudeProjectsDirs()
	total := 0
	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			continue // not present — skip
		}
		files, err := FindSessionFiles(dir)
		if err != nil {
			log.Warn("cost importer: failed to scan dir", "dir", dir, "error", err)
			continue
		}
		for _, f := range files {
			if err := ctx.Err(); err != nil {
				return total, err
			}
			n, err := imp.importFile(ctx, f)
			if err != nil {
				log.Warn("cost importer: failed to import file", "file", f, "error", err)
				continue
			}
			total += n
		}
	}
	return total, nil
}

// claudeProjectsDirs returns all directories to scan for JSONL session files.
// Host agents use ~/.claude/projects/, Docker agents use
// .bc/agents/<name>/auth/.claude/projects/.
func (imp *Importer) claudeProjectsDirs() []string {
	var dirs []string

	// Host Claude Code projects directory
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".claude", "projects"))
	}

	// Per-agent Docker auth directories
	agentsDir := filepath.Join(imp.workspaceDir, ".bc", "agents")
	entries, err := os.ReadDir(agentsDir)
	if err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			authProjects := filepath.Join(agentsDir, e.Name(), "auth", ".claude", "projects")
			if _, err := os.Stat(authProjects); err == nil {
				dirs = append(dirs, authProjects)
			}
		}
	}

	return dirs
}

// importFile imports all new entries from a single JSONL file into the store.
// Returns the number of records inserted.
func (imp *Importer) importFile(ctx context.Context, path string) (int, error) {
	// Determine which entries in this file have already been imported.
	lastImport, err := imp.lastImportedTimestamp(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("query import state: %w", err)
	}

	entries, err := ParseSessionFile(path)
	if err != nil {
		return 0, fmt.Errorf("parse session file %s: %w", path, err)
	}

	var inserted int
	var latest time.Time
	for _, e := range entries {
		// Skip entries already imported (using timestamp watermark per file).
		if !lastImport.IsZero() && !e.Timestamp.After(lastImport) {
			continue
		}
		if e.Timestamp.After(latest) {
			latest = e.Timestamp
		}

		agentID := imp.resolveAgent(e.CWD, path)
		costUSD := CalcCost(e.Model, e.InputTokens, e.OutputTokens, e.CacheCreationTokens, e.CacheReadTokens)

		if err := imp.insertRecord(ctx, e, agentID, costUSD); err != nil {
			log.Warn("cost importer: failed to insert record", "session", e.SessionID, "error", err)
			continue
		}
		inserted++
	}

	if inserted > 0 {
		if err := imp.recordImport(ctx, path, latest, inserted); err != nil {
			log.Warn("cost importer: failed to record import state", "file", path, "error", err)
		}
	}
	return inserted, nil
}

// resolveAgent maps a session's CWD (or the JSONL file path) to a bc agent name.
// Docker agent JSONL files live under .bc/agents/<name>/auth/..., so we can
// extract the name from the path. For host sessions we fall back to the
// workspace name derived from CWD.
func (imp *Importer) resolveAgent(cwd, path string) string {
	// Docker path: .bc/agents/<name>/auth/.claude/projects/...
	agentsDir := filepath.Join(imp.workspaceDir, ".bc", "agents") + string(filepath.Separator)
	if strings.HasPrefix(path, agentsDir) {
		rest := strings.TrimPrefix(path, agentsDir)
		parts := strings.SplitN(rest, string(filepath.Separator), 2)
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}

	// Host session: use the last component of the CWD as a loose agent ID.
	// This won't always match a bc agent name, but provides grouping.
	if cwd != "" {
		return filepath.Base(cwd)
	}
	return "unknown"
}

// initImporterSchema adds the cost_imports and session_id/cache columns if missing.
// Called once from Store.Open via migrate().
func initImporterSchema(db *sql.DB) error {
	ctx := context.Background()

	schema := `
		CREATE TABLE IF NOT EXISTS cost_imports (
			source_path  TEXT NOT NULL,
			watermark    TEXT NOT NULL,
			record_count INTEGER NOT NULL DEFAULT 0,
			imported_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (source_path)
		);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("cost_imports schema: %w", err)
	}

	// Add optional columns to cost_records (migrations — fail silently if already present).
	migrations := []string{
		`ALTER TABLE cost_records ADD COLUMN session_id TEXT`,
		`ALTER TABLE cost_records ADD COLUMN cache_creation_tokens INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE cost_records ADD COLUMN cache_read_tokens INTEGER NOT NULL DEFAULT 0`,
	}
	for _, m := range migrations {
		_, _ = db.ExecContext(ctx, m) // ignore "duplicate column" errors
	}
	return nil
}

func (imp *Importer) lastImportedTimestamp(ctx context.Context, path string) (time.Time, error) {
	row := imp.store.db.QueryRowContext(ctx,
		`SELECT watermark FROM cost_imports WHERE source_path = ?`, path)
	var watermark string
	if err := row.Scan(&watermark); err == sql.ErrNoRows {
		return time.Time{}, nil
	} else if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, watermark)
	return t, err
}

func (imp *Importer) insertRecord(ctx context.Context, e SessionEntry, agentID string, costUSD float64) error {
	total := e.InputTokens + e.OutputTokens + e.CacheCreationTokens + e.CacheReadTokens
	_, err := imp.store.db.ExecContext(ctx,
		`INSERT INTO cost_records
		 (agent_id, model, session_id, input_tokens, output_tokens, total_tokens,
		  cache_creation_tokens, cache_read_tokens, cost_usd, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agentID, e.Model, e.SessionID,
		e.InputTokens, e.OutputTokens, total,
		e.CacheCreationTokens, e.CacheReadTokens,
		costUSD,
		e.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (imp *Importer) recordImport(ctx context.Context, path string, watermark time.Time, count int) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := imp.store.db.ExecContext(ctx,
		`INSERT INTO cost_imports (source_path, watermark, record_count, imported_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(source_path) DO UPDATE SET
		   watermark    = excluded.watermark,
		   record_count = cost_imports.record_count + excluded.record_count,
		   imported_at  = excluded.imported_at`,
		path, watermark.UTC().Format(time.RFC3339Nano), count, now,
	)
	return err
}
