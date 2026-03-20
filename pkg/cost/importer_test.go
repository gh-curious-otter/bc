package cost

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Open(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestImporter_ImportAll_IngestsRecords(t *testing.T) {
	s := openTestStore(t)
	projectsDir := t.TempDir()

	// Write a fake JSONL session file.
	sessionDir := filepath.Join(projectsDir, "-my-project")
	if err := os.MkdirAll(sessionDir, 0750); err != nil {
		t.Fatal(err)
	}
	content := `{"type":"assistant","sessionId":"sess1","timestamp":"2026-03-10T12:00:00Z","cwd":"/my-project","message":{"model":"claude-opus-4-6","usage":{"input_tokens":100,"output_tokens":50}}}
{"type":"assistant","sessionId":"sess1","timestamp":"2026-03-10T12:00:01Z","cwd":"/my-project","message":{"model":"claude-opus-4-6","usage":{"input_tokens":200,"output_tokens":80}}}
`
	jsonlPath := filepath.Join(sessionDir, "sess1.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	imp := &Importer{store: s, workspaceDir: t.TempDir()}

	// Override dirs by calling importFile directly via a helper.
	n, err := imp.importFile(context.Background(), jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("want 2 records imported, got %d", n)
	}

	summary, err := s.WorkspaceSummary()
	if err != nil {
		t.Fatal(err)
	}
	if summary.RecordCount != 2 {
		t.Errorf("want 2 records in store, got %d", summary.RecordCount)
	}
	if summary.InputTokens != 300 {
		t.Errorf("want 300 input tokens, got %d", summary.InputTokens)
	}
}

func TestImporter_Idempotent(t *testing.T) {
	s := openTestStore(t)
	content := `{"type":"assistant","sessionId":"s2","timestamp":"2026-03-10T12:00:00Z","cwd":"/proj","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":50,"output_tokens":20}}}
`
	jsonlPath := filepath.Join(t.TempDir(), "s2.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	imp := &Importer{store: s, workspaceDir: t.TempDir()}

	n1, err := imp.importFile(context.Background(), jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	n2, err := imp.importFile(context.Background(), jsonlPath)
	if err != nil {
		t.Fatal(err)
	}

	if n1 != 1 {
		t.Errorf("first import: want 1, got %d", n1)
	}
	if n2 != 0 {
		t.Errorf("second import should be 0 (idempotent), got %d", n2)
	}
}

func TestImporter_NewEntriesAfterWatermark(t *testing.T) {
	s := openTestStore(t)

	jsonlPath := filepath.Join(t.TempDir(), "s3.jsonl")
	line1 := `{"type":"assistant","sessionId":"s3","timestamp":"2026-03-10T10:00:00Z","cwd":"/p","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":10,"output_tokens":5}}}` + "\n"
	if err := os.WriteFile(jsonlPath, []byte(line1), 0600); err != nil {
		t.Fatal(err)
	}

	imp := &Importer{store: s, workspaceDir: t.TempDir()}

	if _, err := imp.importFile(context.Background(), jsonlPath); err != nil {
		t.Fatal(err)
	}

	// Append a new entry with a later timestamp.
	line2 := `{"type":"assistant","sessionId":"s3","timestamp":"2026-03-10T11:00:00Z","cwd":"/p","message":{"model":"claude-sonnet-4-6","usage":{"input_tokens":20,"output_tokens":8}}}` + "\n"
	f, err := os.OpenFile(jsonlPath, os.O_APPEND|os.O_WRONLY, 0600) //nolint:gosec // test file, path is controlled
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(line2)
	_ = f.Close()

	n, err := imp.importFile(context.Background(), jsonlPath)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("want 1 new record after watermark, got %d", n)
	}
}

func TestImporter_ResolveAgent_DockerPath(t *testing.T) {
	wsDir := t.TempDir()
	imp := &Importer{store: nil, workspaceDir: wsDir}

	agentsDir := filepath.Join(wsDir, ".bc", "agents")
	dockerPath := filepath.Join(agentsDir, "my-agent", "auth", ".claude", "projects", "-proj", "sess.jsonl")

	agent := imp.resolveAgent("/some/cwd", dockerPath)
	if agent != "my-agent" {
		t.Errorf("want agent 'my-agent', got %q", agent)
	}
}

func TestImporter_ResolveAgent_HostPath(t *testing.T) {
	wsDir := t.TempDir()
	imp := &Importer{store: nil, workspaceDir: wsDir}

	hostPath := "/home/user/.claude/projects/-some-project/sess.jsonl"
	agent := imp.resolveAgent("/workspace/my-project", hostPath)
	if agent != "my-project" {
		t.Errorf("want 'my-project' from CWD, got %q", agent)
	}
}

func TestImporterSchema_MigratesColumns(t *testing.T) {
	s := openTestStore(t)
	// Verify the new columns exist by inserting a record that uses them.
	_, err := s.db.ExecContext(context.Background(),
		`INSERT INTO cost_records (agent_id, model, session_id, input_tokens, output_tokens, total_tokens, cache_creation_tokens, cache_read_tokens, cost_usd, timestamp) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"agent-a", "claude-sonnet-4-6", "sess-x", 1, 1, 2, 3, 4, 0.001,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		t.Fatalf("expected migration to add columns, got: %v", err)
	}
}
