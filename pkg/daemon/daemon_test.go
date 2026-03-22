package daemon

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestManager creates a Manager backed by a temporary directory.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(func() { _ = mgr.Close() })
	return mgr
}

// saveDaemon is a test helper that inserts a daemon record directly into the DB.
func saveDaemon(t *testing.T, mgr *Manager, d *Daemon) {
	t.Helper()
	ctx := context.Background()
	if err := mgr.save(ctx, d); err != nil {
		t.Fatalf("save daemon %q: %v", d.Name, err)
	}
}

// --- Manager creation and initialization ---

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	daemons, err := mgr.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(daemons) != 0 {
		t.Errorf("expected 0 daemons, got %d", len(daemons))
	}
}

func TestNewManagerCreatesLogsDir(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	logsDir := filepath.Join(dir, ".bc", "logs")
	info, err := os.Stat(logsDir)
	if err != nil {
		t.Fatalf("logs dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("logs path is not a directory")
	}
}

func TestNewManagerCreatesDB(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr.Close() }()

	dbPath := filepath.Join(dir, ".bc", "daemons.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("db file not created: %v", err)
	}
}

func TestNewManagerWorkspaceHash(t *testing.T) {
	mgr := newTestManager(t)
	if mgr.workspaceHash == "" {
		t.Error("workspace hash should not be empty")
	}
	if len(mgr.workspaceHash) != 6 {
		t.Errorf("workspace hash should be 6 hex chars, got %q", mgr.workspaceHash)
	}
}

func TestCloseManager(t *testing.T) {
	mgr := newTestManager(t)
	if err := mgr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestNewManagerDBOpenError(t *testing.T) {
	// Use /dev/null as the workspace dir — db.Open should fail
	// because /dev/null/.bc/daemons.db is not a valid path.
	_, err := NewManager("/dev/null")
	if err == nil {
		t.Fatal("expected error for invalid db path")
	}
	if !strings.Contains(err.Error(), "open daemons db") {
		t.Errorf("error %q should mention open daemons db", err.Error())
	}
}

// --- Name validation ---

func TestIsValidDaemonName(t *testing.T) {
	valid := []string{"db", "my-db", "my_db", "DB01", "a", strings.Repeat("a", 63)}
	for _, name := range valid {
		if !isValidDaemonName(name) {
			t.Errorf("isValidDaemonName(%q) = false, want true", name)
		}
	}

	invalid := []string{"", "has space", "has/slash", "has.dot", "../escape", strings.Repeat("a", 64)}
	for _, name := range invalid {
		if isValidDaemonName(name) {
			t.Errorf("isValidDaemonName(%q) = true, want false", name)
		}
	}
}

func TestIsValidDaemonNameEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"single char", "x", true},
		{"max length", strings.Repeat("z", 63), true},
		{"over max length", strings.Repeat("z", 64), false},
		{"with colon", "my:db", false},
		{"with at sign", "my@db", false},
		{"with newline", "my\ndb", false},
		{"with tab", "my\tdb", false},
		{"uppercase only", "MYDB", true},
		{"digits only", "123", true},
		{"mixed", "My-Db_01", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDaemonName(tt.input)
			if got != tt.want {
				t.Errorf("isValidDaemonName(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// --- Run validation ---

func TestRunValidation(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	tests := []struct {
		name    string
		wantErr string
		opts    RunOptions
	}{
		{
			name:    "missing name",
			opts:    RunOptions{Runtime: RuntimeTmux, Cmd: "echo hi"},
			wantErr: "name is required",
		},
		{
			name:    "invalid name",
			opts:    RunOptions{Name: "bad name!", Runtime: RuntimeTmux, Cmd: "echo hi"},
			wantErr: "invalid daemon name",
		},
		{
			name:    "missing runtime",
			opts:    RunOptions{Name: "test"},
			wantErr: "runtime must be",
		},
		{
			name:    "invalid runtime",
			opts:    RunOptions{Name: "test", Runtime: "podman"},
			wantErr: "runtime must be",
		},
		{
			name:    "tmux without cmd",
			opts:    RunOptions{Name: "test", Runtime: RuntimeTmux},
			wantErr: "--cmd is required",
		},
		{
			name:    "docker without image",
			opts:    RunOptions{Name: "test", Runtime: RuntimeDocker},
			wantErr: "--image is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mgr.Run(ctx, tt.opts)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestRunBashAliasNormalized(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// "bash" runtime without cmd should produce the tmux-specific error,
	// proving that bash was normalized to tmux.
	_, err := mgr.Run(ctx, RunOptions{
		Name:    "test",
		Runtime: "bash",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--cmd is required") {
		t.Errorf("expected tmux cmd error after bash normalization, got: %v", err)
	}
}

func TestRunAlreadyRunning(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// Insert a "running" daemon directly via SQL to avoid syncStatus in Get.
	// The Run method calls Get which calls syncStatus — for the test to
	// work we need a daemon whose syncStatus keeps it as running.
	// Since we can't have a real tmux session, we test this by checking
	// that after syncStatus marks it failed, Run allows re-running it
	// (which is the stopped/failed path). Instead, test the validation
	// by verifying that a valid Run opts for an existing stopped daemon
	// proceeds past the "already running" check.
	d := &Daemon{
		Name:      "mydb",
		Runtime:   RuntimeTmux,
		Cmd:       "sleep 999",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	// A stopped daemon should not trigger the "already running" error.
	// It will fail at startTmux instead.
	_, err := mgr.Run(ctx, RunOptions{
		Name:    "mydb",
		Runtime: RuntimeTmux,
		Cmd:     "echo hi",
	})
	if err == nil {
		return // tmux was available
	}
	// Should fail at tmux start, not at "already running"
	if strings.Contains(err.Error(), "already running") {
		t.Errorf("stopped daemon should not be 'already running', got: %v", err)
	}
}

func TestRunDefaultRestart(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// Will fail at startTmux, but should get past validation.
	_, err := mgr.Run(ctx, RunOptions{
		Name:    "test-default-restart",
		Runtime: RuntimeTmux,
		Cmd:     "echo hi",
		// Restart intentionally left empty — should default to "no"
	})
	if err == nil {
		return // tmux was available
	}
	// Should not fail on validation
	if strings.Contains(err.Error(), "name is required") ||
		strings.Contains(err.Error(), "runtime must be") ||
		strings.Contains(err.Error(), "--cmd is required") {
		t.Errorf("should not fail on validation, got: %v", err)
	}
}

func TestRunWithEnvFile(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	envFile := filepath.Join(t.TempDir(), "test.env")
	if err := os.WriteFile(envFile, []byte("KEY1=val1\nKEY2=val2\n"), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	// Will fail at startTmux but should get past env file reading.
	_, err := mgr.Run(ctx, RunOptions{
		Name:    "test-env",
		Runtime: RuntimeTmux,
		Cmd:     "echo hi",
		EnvFile: envFile,
	})
	if err != nil && strings.Contains(err.Error(), "read env file") {
		t.Errorf("should have read env file successfully, got: %v", err)
	}
}

func TestRunWithBadEnvFile(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	_, err := mgr.Run(ctx, RunOptions{
		Name:    "test-env-bad",
		Runtime: RuntimeTmux,
		Cmd:     "echo hi",
		EnvFile: "/nonexistent/path/env",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent env file")
	}
	if !strings.Contains(err.Error(), "read env file") {
		t.Errorf("error %q should mention read env file", err.Error())
	}
}

// --- Save and Get (SQLite persistence) ---

func TestSaveAndGet(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d := &Daemon{
		Name:      "testdb",
		Runtime:   RuntimeDocker,
		Image:     "postgres:16",
		Status:    StatusStopped, // Use stopped so syncStatus is a no-op
		Restart:   "always",
		Ports:     []string{"5432:5432"},
		EnvVars:   []string{"POSTGRES_PASSWORD=secret"},
		CreatedAt: now,
		StartedAt: now,
		PID:       0,
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "testdb")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected daemon, got nil")
	}

	if got.Name != "testdb" {
		t.Errorf("name = %q, want testdb", got.Name)
	}
	if got.Runtime != RuntimeDocker {
		t.Errorf("runtime = %q, want docker", got.Runtime)
	}
	if got.Image != "postgres:16" {
		t.Errorf("image = %q, want postgres:16", got.Image)
	}
	if got.Status != StatusStopped {
		t.Errorf("status = %q, want stopped", got.Status)
	}
	if got.Restart != "always" {
		t.Errorf("restart = %q, want always", got.Restart)
	}
	if len(got.Ports) != 1 || got.Ports[0] != "5432:5432" {
		t.Errorf("ports = %v, want [5432:5432]", got.Ports)
	}
	if len(got.EnvVars) != 1 || got.EnvVars[0] != "POSTGRES_PASSWORD=secret" {
		t.Errorf("env = %v, want [POSTGRES_PASSWORD=secret]", got.EnvVars)
	}
}

func TestSaveUpdatesExisting(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	d := &Daemon{
		Name:      "updateme",
		Runtime:   RuntimeTmux,
		Cmd:       "sleep 100",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: now,
		StartedAt: now,
	}
	saveDaemon(t, mgr, d)

	// Update status
	stopped := time.Now().UTC().Truncate(time.Second)
	d.StoppedAt = &stopped
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "updateme")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusStopped {
		t.Errorf("status = %q after update, want stopped", got.Status)
	}
	if got.StoppedAt == nil {
		t.Error("stopped_at should be set after update")
	}
}

func TestSaveWithEmptySlices(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "minimal",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "minimal")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected daemon")
	}
	if len(got.Ports) != 0 {
		t.Errorf("ports should be empty, got %v", got.Ports)
	}
	if len(got.EnvVars) != 0 {
		t.Errorf("env should be empty, got %v", got.EnvVars)
	}
}

func TestSaveWithZeroStartedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "zero-start",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "zero-start")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected daemon")
	}
	if !got.StartedAt.IsZero() {
		t.Errorf("started_at should be zero, got %v", got.StartedAt)
	}
}

func TestSaveWithStoppedAt(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)
	stopped := now.Add(time.Minute)
	d := &Daemon{
		Name:      "with-stopped",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: now,
		StartedAt: now,
		StoppedAt: &stopped,
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "with-stopped")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.StoppedAt == nil {
		t.Fatal("stopped_at should be set")
	}
	if !got.StoppedAt.Equal(stopped) {
		t.Errorf("stopped_at = %v, want %v", got.StoppedAt, stopped)
	}
}

func TestSaveWithPID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "with-pid",
		Runtime:   RuntimeTmux,
		Cmd:       "sleep 1",
		Status:    StatusStopped,
		Restart:   "no",
		PID:       12345,
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "with-pid")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.PID != 12345 {
		t.Errorf("pid = %d, want 12345", got.PID)
	}
}

func TestSaveWithContainerID(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:        "with-cid",
		Runtime:     RuntimeDocker,
		Image:       "nginx:latest",
		ContainerID: "abc123def456",
		Status:      StatusStopped,
		Restart:     "no",
		CreatedAt:   time.Now(),
		StartedAt:   time.Now(),
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "with-cid")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ContainerID != "abc123def456" {
		t.Errorf("container_id = %q, want abc123def456", got.ContainerID)
	}
}

func TestSaveMultiplePorts(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "multi-port",
		Runtime:   RuntimeDocker,
		Image:     "app:latest",
		Status:    StatusStopped,
		Restart:   "no",
		Ports:     []string{"8080:80", "8443:443", "9090:9090"},
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "multi-port")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Ports) != 3 {
		t.Errorf("ports = %v, want 3 entries", got.Ports)
	}
}

func TestSaveMultipleEnvVars(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "multi-env",
		Runtime:   RuntimeDocker,
		Image:     "app:latest",
		Status:    StatusStopped,
		Restart:   "no",
		EnvVars:   []string{"FOO=bar", "BAZ=qux", "KEY=value with spaces"},
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	got, err := mgr.Get(ctx, "multi-env")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.EnvVars) != 3 {
		t.Errorf("env = %v, want 3 entries", got.EnvVars)
	}
	if got.EnvVars[2] != "KEY=value with spaces" {
		t.Errorf("env[2] = %q, want KEY=value with spaces", got.EnvVars[2])
	}
}

// --- Get ---

func TestGetNotFound(t *testing.T) {
	mgr := newTestManager(t)

	d, err := mgr.Get(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d != nil {
		t.Errorf("expected nil, got %+v", d)
	}
}

// --- List ---

func TestListEmpty(t *testing.T) {
	mgr := newTestManager(t)

	daemons, err := mgr.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(daemons) != 0 {
		t.Errorf("expected 0 daemons, got %d", len(daemons))
	}
}

func TestListMultiple(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	now := time.Now()
	for i, name := range []string{"alpha", "beta", "gamma"} {
		d := &Daemon{
			Name:      name,
			Runtime:   RuntimeTmux,
			Cmd:       "sleep 1",
			Status:    StatusStopped,
			Restart:   "no",
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			StartedAt: now.Add(time.Duration(i) * time.Second),
		}
		saveDaemon(t, mgr, d)
	}

	daemons, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(daemons) != 3 {
		t.Fatalf("expected 3 daemons, got %d", len(daemons))
	}

	// Should be ordered by created_at ASC
	if daemons[0].Name != "alpha" {
		t.Errorf("first daemon = %q, want alpha", daemons[0].Name)
	}
	if daemons[1].Name != "beta" {
		t.Errorf("second daemon = %q, want beta", daemons[1].Name)
	}
	if daemons[2].Name != "gamma" {
		t.Errorf("third daemon = %q, want gamma", daemons[2].Name)
	}
}

func TestListSyncsRunningToFailed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// Save a daemon marked as "running" but with no real tmux session.
	d := &Daemon{
		Name:      "phantom",
		Runtime:   RuntimeTmux,
		Cmd:       "sleep 1",
		Status:    StatusRunning,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	daemons, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(daemons) != 1 {
		t.Fatalf("expected 1 daemon, got %d", len(daemons))
	}
	// syncStatus should detect that the tmux session doesn't exist
	// and mark it as failed.
	if daemons[0].Status != StatusFailed {
		t.Errorf("status = %q, want failed (synced from non-existent tmux session)", daemons[0].Status)
	}
	if daemons[0].StoppedAt == nil {
		t.Error("stopped_at should be set after syncStatus marks it failed")
	}
}

func TestListMixedStatuses(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	now := time.Now()
	statuses := []Status{StatusStopped, StatusFailed, StatusStopped}
	for i, name := range []string{"d1", "d2", "d3"} {
		d := &Daemon{
			Name:      name,
			Runtime:   RuntimeTmux,
			Cmd:       "echo",
			Status:    statuses[i],
			Restart:   "no",
			CreatedAt: now.Add(time.Duration(i) * time.Second),
			StartedAt: now.Add(time.Duration(i) * time.Second),
		}
		saveDaemon(t, mgr, d)
	}

	daemons, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(daemons) != 3 {
		t.Fatalf("expected 3, got %d", len(daemons))
	}
	// stopped/failed should remain unchanged by syncStatus
	if daemons[0].Status != StatusStopped {
		t.Errorf("d1 status = %q, want stopped", daemons[0].Status)
	}
	if daemons[1].Status != StatusFailed {
		t.Errorf("d2 status = %q, want failed", daemons[1].Status)
	}
}

// --- Remove ---

func TestRemoveNotFound(t *testing.T) {
	mgr := newTestManager(t)

	err := mgr.Remove(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error removing nonexistent daemon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention not found", err.Error())
	}
}

func TestRemoveStopped(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "removeme",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	if err := mgr.Remove(ctx, "removeme"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	got, err := mgr.Get(ctx, "removeme")
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if got != nil {
		t.Error("daemon should be nil after removal")
	}
}

func TestRemoveFailed(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "failed-one",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusFailed,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	if err := mgr.Remove(ctx, "failed-one"); err != nil {
		t.Fatalf("Remove failed daemon: %v", err)
	}
}

func TestRemoveRunningBlockedAfterSync(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// When we save a "running" docker daemon and then call Remove,
	// Get will syncStatus and mark it as failed (no real container).
	// So Remove should succeed since the daemon is now failed.
	d := &Daemon{
		Name:      "running-docker",
		Runtime:   RuntimeDocker,
		Image:     "postgres:16",
		Status:    StatusRunning,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	// After syncStatus, this should be marked failed and removable.
	err := mgr.Remove(ctx, "running-docker")
	if err != nil {
		t.Fatalf("Remove after sync should succeed, got: %v", err)
	}
}

// --- Stop ---

func TestStopNotFound(t *testing.T) {
	mgr := newTestManager(t)

	err := mgr.Stop(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error stopping nonexistent daemon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention not found", err.Error())
	}
}

func TestStopAlreadyStopped(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "stopped-one",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	err := mgr.Stop(ctx, "stopped-one")
	if err == nil {
		t.Fatal("expected error stopping already-stopped daemon")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error %q should mention not running", err.Error())
	}
}

func TestStopFailedDaemon(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "failed-stop",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusFailed,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	err := mgr.Stop(ctx, "failed-stop")
	if err == nil {
		t.Fatal("expected error stopping failed daemon")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Errorf("error %q should mention not running", err.Error())
	}
}

// --- Restart ---

func TestRestartNotFound(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.Restart(context.Background(), "ghost")
	if err == nil {
		t.Fatal("expected error restarting nonexistent daemon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention not found", err.Error())
	}
}

func TestRestartStoppedTmux(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "restart-me",
		Runtime:   RuntimeTmux,
		Cmd:       "echo restarted",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	// Restart skips Stop for non-running daemons and calls Run.
	// Run will attempt startTmux — may or may not succeed.
	_, err := mgr.Restart(ctx, "restart-me")
	if err == nil {
		return // tmux was available
	}
	// Should fail at startTmux, not at "not found"
	if strings.Contains(err.Error(), "not found") {
		t.Errorf("should have found the daemon, got: %v", err)
	}
}

// --- Logs ---

func TestLogsNotFound(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.Logs(context.Background(), "ghost", 10)
	if err == nil {
		t.Fatal("expected error for nonexistent daemon")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q should mention not found", err.Error())
	}
}

func TestLogsTmuxNoFile(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "no-logs",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	logs, err := mgr.Logs(ctx, "no-logs", 10)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if logs != "(no logs yet)" {
		t.Errorf("expected '(no logs yet)', got %q", logs)
	}
}

func TestLogsTmuxWithFile(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "has-logs",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	logPath := mgr.logFile("has-logs")
	content := "line1\nline2\nline3\nline4\nline5\n"
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	logs, err := mgr.Logs(ctx, "has-logs", 3)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	lines := strings.Split(logs, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 lines, got %d: %q", len(lines), logs)
	}
}

func TestLogsTmuxAllLines(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "all-logs",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	logPath := mgr.logFile("all-logs")
	content := "line1\nline2\n"
	if err := os.WriteFile(logPath, []byte(content), 0600); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	logs, err := mgr.Logs(ctx, "all-logs", 100)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if !strings.Contains(logs, "line1") || !strings.Contains(logs, "line2") {
		t.Errorf("expected all lines, got %q", logs)
	}
}

func TestLogsTmuxZeroLines(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "zero-lines",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusStopped,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	logPath := mgr.logFile("zero-lines")
	if err := os.WriteFile(logPath, []byte("line1\nline2\n"), 0600); err != nil {
		t.Fatalf("write log file: %v", err)
	}

	logs, err := mgr.Logs(ctx, "zero-lines", 0)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	if !strings.Contains(logs, "line1") {
		t.Errorf("expected all lines with lines=0, got %q", logs)
	}
}

// --- syncStatus ---

func TestSyncStatusStoppedIsNoop(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:    "already-stopped",
		Runtime: RuntimeTmux,
		Status:  StatusStopped,
	}

	mgr.syncStatus(ctx, d)
	if d.Status != StatusStopped {
		t.Errorf("status should remain stopped, got %q", d.Status)
	}
}

func TestSyncStatusFailedIsNoop(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:    "already-failed",
		Runtime: RuntimeTmux,
		Status:  StatusFailed,
	}

	mgr.syncStatus(ctx, d)
	if d.Status != StatusFailed {
		t.Errorf("status should remain failed, got %q", d.Status)
	}
}

func TestSyncStatusRunningTmuxNotAlive(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:      "dead-tmux",
		Runtime:   RuntimeTmux,
		Cmd:       "echo hi",
		Status:    StatusRunning,
		Restart:   "no",
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}
	saveDaemon(t, mgr, d)

	mgr.syncStatus(ctx, d)
	if d.Status != StatusFailed {
		t.Errorf("status = %q, want failed (tmux session not alive)", d.Status)
	}
	if d.StoppedAt == nil {
		t.Error("stopped_at should be set")
	}
}

func TestSyncStatusRunningDockerNotAlive(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:        "dead-docker",
		Runtime:     RuntimeDocker,
		Image:       "nginx:latest",
		ContainerID: "abc123",
		Status:      StatusRunning,
		Restart:     "no",
		CreatedAt:   time.Now(),
		StartedAt:   time.Now(),
	}
	saveDaemon(t, mgr, d)

	mgr.syncStatus(ctx, d)
	if d.Status != StatusFailed {
		t.Errorf("status = %q, want failed (docker container not alive)", d.Status)
	}
	if d.StoppedAt == nil {
		t.Error("stopped_at should be set")
	}
}

// --- containerName ---

func TestContainerName(t *testing.T) {
	mgr := newTestManager(t)

	name := mgr.containerName("mydb")
	if name == "" {
		t.Error("container name should not be empty")
	}
	if !strings.HasPrefix(name, "bc-") {
		t.Errorf("container name %q should start with bc-", name)
	}
	if !strings.HasSuffix(name, "-mydb") {
		t.Errorf("container name %q should end with -mydb", name)
	}
}

func TestContainerNameDeterministic(t *testing.T) {
	dir := t.TempDir()
	mgr1, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr1.Close() }()

	mgr2, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer func() { _ = mgr2.Close() }()

	if mgr1.containerName("test") != mgr2.containerName("test") {
		t.Error("container names should be deterministic for same workspace")
	}
}

func TestContainerNameIncludesHash(t *testing.T) {
	mgr := newTestManager(t)
	name := mgr.containerName("db")
	// Format: bc-<hash>-<name>
	parts := strings.SplitN(name, "-", 3)
	if len(parts) != 3 {
		t.Errorf("container name %q should have format bc-<hash>-<name>", name)
	}
	if parts[0] != "bc" {
		t.Errorf("first part = %q, want bc", parts[0])
	}
	if parts[2] != "db" {
		t.Errorf("third part = %q, want db", parts[2])
	}
}

// --- logFile ---

func TestLogFile(t *testing.T) {
	mgr := newTestManager(t)

	path := mgr.logFile("mydb")
	if !strings.HasSuffix(path, "daemon-mydb.log") {
		t.Errorf("logFile = %q, want suffix daemon-mydb.log", path)
	}
	if !strings.Contains(path, ".bc"+string(filepath.Separator)+"logs"+string(filepath.Separator)) {
		t.Errorf("logFile = %q, want to contain .bc/logs/", path)
	}
}

// --- nullStr ---

func TestNullStr(t *testing.T) {
	if nullStr("") != nil {
		t.Error("nullStr(\"\") should return nil")
	}
	s := nullStr("hello")
	if s == nil || *s != "hello" {
		t.Error("nullStr(\"hello\") should return pointer to \"hello\"")
	}
}

// --- readEnvFile ---

func TestReadEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/env"

	content := "# comment\nKEY1=value1\nKEY2=value2\n\nKEY3=value with spaces\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	env, err := readEnvFile(path)
	if err != nil {
		t.Fatalf("readEnvFile: %v", err)
	}
	if len(env) != 3 {
		t.Errorf("expected 3 env vars, got %d: %v", len(env), env)
	}
	if env[0] != "KEY1=value1" {
		t.Errorf("env[0] = %q, want KEY1=value1", env[0])
	}
}

func TestReadEnvFileNotFound(t *testing.T) {
	_, err := readEnvFile("/nonexistent/path/env")
	if err == nil {
		t.Fatal("expected error for nonexistent env file")
	}
}

func TestReadEnvFileEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.env")
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	env, err := readEnvFile(path)
	if err != nil {
		t.Fatalf("readEnvFile: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected 0 env vars from empty file, got %d", len(env))
	}
}

func TestReadEnvFileCommentsOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "comments.env")
	content := "# this is a comment\n# another comment\n\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	env, err := readEnvFile(path)
	if err != nil {
		t.Fatalf("readEnvFile: %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected 0 env vars from comments-only file, got %d", len(env))
	}
}

func TestReadEnvFileWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ws.env")
	content := "  KEY1=value1  \n  # comment  \n  KEY2=value2  \n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	env, err := readEnvFile(path)
	if err != nil {
		t.Fatalf("readEnvFile: %v", err)
	}
	if len(env) != 2 {
		t.Errorf("expected 2 env vars, got %d: %v", len(env), env)
	}
}

// --- scanDaemon with bash runtime normalization ---

func TestScanDaemonBashNormalization(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// Insert a record with runtime="bash" directly via SQL
	_, err := mgr.db.ExecContext(ctx, `
		INSERT INTO daemons (name, runtime, cmd, status, restart, created_at, ports, env)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"legacy-bash", "bash", "echo hi", "stopped", "no",
		time.Now().UTC().Format(time.RFC3339), "[]", "[]",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	d, err := mgr.Get(ctx, "legacy-bash")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if d == nil {
		t.Fatal("expected daemon")
	}
	if d.Runtime != RuntimeTmux {
		t.Errorf("runtime = %q, want tmux (normalized from bash)", d.Runtime)
	}
}

// --- Full lifecycle (using DB-level operations) ---

func TestDaemonLifecycleThroughDB(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Second)

	// 1. Create (stopped, so syncStatus is no-op)
	d := &Daemon{
		Name:      "lifecycle",
		Runtime:   RuntimeTmux,
		Cmd:       "sleep 999",
		Status:    StatusStopped,
		Restart:   "on-failure",
		Ports:     []string{"8080:80"},
		EnvVars:   []string{"FOO=bar"},
		CreatedAt: now,
		StartedAt: now,
	}
	saveDaemon(t, mgr, d)

	// 2. Verify it exists
	got, err := mgr.Get(ctx, "lifecycle")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected daemon")
	}
	if got.Status != StatusStopped {
		t.Errorf("status = %q, want stopped", got.Status)
	}

	// 3. Remove it (already stopped)
	err = mgr.Remove(ctx, "lifecycle")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// 4. Verify removed
	got, err = mgr.Get(ctx, "lifecycle")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Error("daemon should be nil after removal")
	}
}

// --- RunRejectsInvalidName ---

func TestRunRejectsInvalidName(t *testing.T) {
	mgr := newTestManager(t)

	_, err := mgr.Run(context.Background(), RunOptions{
		Name:    "bad name!",
		Runtime: RuntimeTmux,
		Cmd:     "echo hi",
	})
	if err == nil {
		t.Fatal("expected validation error for invalid name")
	}
	if !strings.Contains(err.Error(), "invalid daemon name") {
		t.Errorf("error %q should mention invalid daemon name", err.Error())
	}
}

// --- Integration tests with real tmux ---

func tmuxAvailable() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

func TestRunStopTmuxIntegration(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	mgr := newTestManager(t)
	ctx := context.Background()

	// Run a real tmux daemon
	d, err := mgr.Run(ctx, RunOptions{
		Name:    "integ-run",
		Runtime: RuntimeTmux,
		Cmd:     "sleep 3600",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if d.Status != StatusRunning {
		t.Errorf("status = %q, want running", d.Status)
	}
	if d.Name != "integ-run" {
		t.Errorf("name = %q, want integ-run", d.Name)
	}
	if d.Restart != "no" {
		t.Errorf("restart = %q, want no (default)", d.Restart)
	}

	// Verify it shows up in List
	daemons, err := mgr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, dd := range daemons {
		if dd.Name == "integ-run" && dd.Status == StatusRunning {
			found = true
		}
	}
	if !found {
		t.Error("running daemon not found in List")
	}

	// Verify Get returns running
	got, err := mgr.Get(ctx, "integ-run")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil || got.Status != StatusRunning {
		t.Errorf("Get status = %v, want running", got)
	}

	// Stop it
	stopErr := mgr.Stop(ctx, "integ-run")
	if stopErr != nil {
		t.Fatalf("Stop: %v", stopErr)
	}

	// Verify stopped
	got, err = mgr.Get(ctx, "integ-run")
	if err != nil {
		t.Fatalf("Get after stop: %v", err)
	}
	if got.Status != StatusStopped {
		t.Errorf("status = %q after stop, want stopped", got.Status)
	}
	if got.StoppedAt == nil {
		t.Error("stopped_at should be set after stop")
	}

	// Remove it
	if err := mgr.Remove(ctx, "integ-run"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
}

func TestRunDuplicateTmuxIntegration(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	mgr := newTestManager(t)
	ctx := context.Background()

	d, err := mgr.Run(ctx, RunOptions{
		Name:    "integ-dup",
		Runtime: RuntimeTmux,
		Cmd:     "sleep 3600",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Stop(ctx, d.Name)
		_ = mgr.Remove(ctx, d.Name)
	})

	// Try to run again — should fail with "already running"
	_, err = mgr.Run(ctx, RunOptions{
		Name:    "integ-dup",
		Runtime: RuntimeTmux,
		Cmd:     "sleep 3600",
	})
	if err == nil {
		t.Fatal("expected error for duplicate running daemon")
	}
	if !strings.Contains(err.Error(), "already running") {
		t.Errorf("error %q should mention already running", err.Error())
	}
}

func TestRestartTmuxIntegration(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	mgr := newTestManager(t)
	ctx := context.Background()

	d, err := mgr.Run(ctx, RunOptions{
		Name:    "integ-restart",
		Runtime: RuntimeTmux,
		Cmd:     "sleep 3600",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Stop(ctx, "integ-restart")
		_ = mgr.Remove(ctx, "integ-restart")
	})

	if d.Status != StatusRunning {
		t.Fatalf("initial status = %q, want running", d.Status)
	}

	// Restart it (stop + run)
	restarted, err := mgr.Restart(ctx, "integ-restart")
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	if restarted.Status != StatusRunning {
		t.Errorf("restarted status = %q, want running", restarted.Status)
	}
}

func TestRunWithEnvTmuxIntegration(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	mgr := newTestManager(t)
	ctx := context.Background()

	envFile := filepath.Join(t.TempDir(), "test.env")
	if err := os.WriteFile(envFile, []byte("TEST_KEY=test_value\n"), 0600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	d, err := mgr.Run(ctx, RunOptions{
		Name:    "integ-env",
		Runtime: RuntimeTmux,
		Cmd:     "sleep 3600",
		Env:     []string{"INLINE_KEY=inline_val"},
		EnvFile: envFile,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Stop(ctx, d.Name)
		_ = mgr.Remove(ctx, d.Name)
	})

	if d.Status != StatusRunning {
		t.Errorf("status = %q, want running", d.Status)
	}
	// EnvVars should include both inline and file envs
	if len(d.EnvVars) != 2 {
		t.Errorf("env vars = %v, want 2 entries", d.EnvVars)
	}
}

func TestLogsTmuxIntegration(t *testing.T) {
	if !tmuxAvailable() {
		t.Skip("tmux not available")
	}

	mgr := newTestManager(t)
	ctx := context.Background()

	d, err := mgr.Run(ctx, RunOptions{
		Name:    "integ-logs",
		Runtime: RuntimeTmux,
		Cmd:     "echo hello-from-daemon",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	t.Cleanup(func() {
		_ = mgr.Stop(ctx, d.Name)
		_ = mgr.Remove(ctx, d.Name)
	})

	// Logs should work without error (content may or may not be captured yet)
	_, err = mgr.Logs(ctx, "integ-logs", 10)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
}

func TestLogsDockerFallback(t *testing.T) {
	// Test the docker logs path — daemon saved as docker runtime, stopped.
	// docker logs will fail (no container), but the code returns whatever it got.
	mgr := newTestManager(t)
	ctx := context.Background()

	d := &Daemon{
		Name:        "docker-logs",
		Runtime:     RuntimeDocker,
		Image:       "nginx:latest",
		ContainerID: "nonexistent-container",
		Status:      StatusStopped,
		Restart:     "no",
		CreatedAt:   time.Now(),
		StartedAt:   time.Now(),
	}
	saveDaemon(t, mgr, d)

	// Docker logs will try docker logs command — should return output
	// (even if empty/error). The code returns nil error in both paths.
	logs, err := mgr.Logs(ctx, "docker-logs", 10)
	if err != nil {
		t.Fatalf("Logs: %v", err)
	}
	// We don't know the exact output but it shouldn't be "(no logs yet)"
	// since that's only for tmux runtime.
	_ = logs
}

func TestRunDockerFailsWithoutDocker(t *testing.T) {
	mgr := newTestManager(t)
	ctx := context.Background()

	// Attempt to run a docker daemon — will fail because Docker is not running.
	// This exercises the startDocker error path in Run.
	_, err := mgr.Run(ctx, RunOptions{
		Name:    "docker-fail",
		Runtime: RuntimeDocker,
		Image:   "alpine:latest",
	})
	if err == nil {
		// Docker was available — skip the assertion
		// Clean up
		_ = mgr.Stop(ctx, "docker-fail")
		_ = mgr.Remove(ctx, "docker-fail")
		return
	}
	if !strings.Contains(err.Error(), "start docker daemon") {
		t.Errorf("error %q should mention start docker daemon", err.Error())
	}

	// Verify the daemon was saved with failed status
	got, getErr := mgr.Get(ctx, "docker-fail")
	if getErr != nil {
		t.Fatalf("Get: %v", getErr)
	}
	if got != nil && got.Status != StatusFailed {
		t.Errorf("status = %q, want failed after docker start failure", got.Status)
	}
}

// --- Status constants ---

func TestStatusConstants(t *testing.T) {
	if StatusRunning != "running" {
		t.Errorf("StatusRunning = %q, want running", StatusRunning)
	}
	if StatusStopped != "stopped" {
		t.Errorf("StatusStopped = %q, want stopped", StatusStopped)
	}
	if StatusFailed != "failed" {
		t.Errorf("StatusFailed = %q, want failed", StatusFailed)
	}
}

func TestRuntimeConstants(t *testing.T) {
	if RuntimeTmux != "tmux" {
		t.Errorf("RuntimeTmux = %q, want tmux", RuntimeTmux)
	}
	if RuntimeDocker != "docker" {
		t.Errorf("RuntimeDocker = %q, want docker", RuntimeDocker)
	}
}
