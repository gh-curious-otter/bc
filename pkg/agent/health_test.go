package agent

import (
	"context"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rpuneet/bc/pkg/events"
)

// testTmuxChecker implements TmuxChecker for health tests.
type testTmuxChecker struct {
	sessions map[string]bool
}

func newTestTmuxChecker() *testTmuxChecker {
	return &testTmuxChecker{sessions: make(map[string]bool)}
}

func (m *testTmuxChecker) HasSession(_ context.Context, name string) bool {
	return m.sessions[name]
}

func (m *testTmuxChecker) SetSession(name string, alive bool) {
	m.sessions[name] = alive
}

// createTestStore creates a SQLiteStore with a root agent for health tests.
func createTestStore(t *testing.T, session string) *SQLiteStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	root := &Agent{
		Name:      "root",
		Role:      RoleRoot,
		State:     StateIdle,
		Session:   session,
		Workspace: dir,
		IsRoot:    true,
		StartedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := store.Save(root); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}
	return store
}

func TestHealthChecker_Check_Healthy(t *testing.T) {
	store := createTestStore(t, "root-session")

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil)
	result := checker.Check(context.Background())

	if result.Status != HealthStatusHealthy {
		t.Errorf("expected healthy, got %s", result.Status)
	}
	if !result.TmuxAlive {
		t.Error("expected TmuxAlive to be true")
	}
	if !result.StateFresh {
		t.Error("expected StateFresh to be true")
	}
}

func TestHealthChecker_Check_UnhealthyTmuxDead(t *testing.T) {
	store := createTestStore(t, "root-session")

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", false)

	checker := NewHealthChecker(store, tmux, nil)
	result := checker.Check(context.Background())

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
	if result.TmuxAlive {
		t.Error("expected TmuxAlive to be false")
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message for unhealthy status")
	}
}

func TestHealthChecker_Check_DegradedStaleState(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	// Save root with stale UpdatedAt
	staleTime := time.Now().Add(-2 * time.Minute)
	root := &Agent{
		Name:      "root",
		Role:      RoleRoot,
		State:     StateIdle,
		Session:   "root-session",
		Workspace: dir,
		IsRoot:    true,
		StartedAt: time.Now().Add(-5 * time.Minute),
		UpdatedAt: staleTime,
	}
	// Save directly with stale time — SQLiteStore.Save sets UpdatedAt to now,
	// so we use raw SQL instead.
	if saveErr := store.Save(root); saveErr != nil {
		t.Fatalf("failed to save root: %v", saveErr)
	}
	// Override UpdatedAt to make it stale
	_, _ = store.db.ExecContext(context.Background(), "UPDATE agents SET updated_at = ? WHERE name = 'root'", formatTime(staleTime))

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithStaleThreshold(30*time.Second))
	result := checker.Check(context.Background())

	if result.Status != HealthStatusDegraded {
		t.Errorf("expected degraded, got %s", result.Status)
	}
	if result.StateFresh {
		t.Error("expected StateFresh to be false")
	}
}

func TestHealthChecker_Check_NoRootState(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	tmux := newTestTmuxChecker()
	checker := NewHealthChecker(store, tmux, nil)
	result := checker.Check(context.Background())

	if result.Status != HealthStatusUnhealthy {
		t.Errorf("expected unhealthy, got %s", result.Status)
	}
	if result.ErrorMessage == "" {
		t.Error("expected error message")
	}
}

func TestHealthChecker_UnhealthyCallback(t *testing.T) {
	store := createTestStore(t, "dead-session")

	tmux := newTestTmuxChecker() // no sessions = dead

	var callbackCalled atomic.Bool
	callback := func(_ *HealthCheckResult) {
		callbackCalled.Store(true)
	}

	checker := NewHealthChecker(store, tmux, nil,
		WithUnhealthyCallback(callback))

	checker.runCheck(context.Background())

	if !callbackCalled.Load() {
		t.Error("expected unhealthy callback to be called")
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	store := createTestStore(t, "root-session")

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithHealthCheckInterval(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to be running")
	}

	time.Sleep(100 * time.Millisecond)

	result := checker.LastResult()
	if result == nil {
		t.Error("expected last result to be set")
	}

	checker.Stop()
	if checker.IsRunning() {
		t.Error("expected checker to be stopped")
	}
}

func TestHealthChecker_EmitsEvents(t *testing.T) {
	store := createTestStore(t, "root-session")
	dir := t.TempDir()

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	eventLog := events.NewLog(filepath.Join(dir, "events.jsonl"))
	checker := NewHealthChecker(store, tmux, eventLog)

	checker.runCheck(context.Background())

	evts, readErr := eventLog.Read()
	if readErr != nil {
		t.Fatalf("failed to read events: %v", readErr)
	}

	if len(evts) == 0 {
		t.Error("expected at least one event")
	}

	found := false
	for _, e := range evts {
		if e.Type == events.HealthCheck {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected health.check event")
	}
}

func TestHealthChecker_LastResult(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	tmux := newTestTmuxChecker()
	checker := NewHealthChecker(store, tmux, nil)

	if checker.LastResult() != nil {
		t.Error("expected nil before first check")
	}

	checker.Check(context.Background())

	result := checker.LastResult()
	if result == nil {
		t.Error("expected result after check")
	}
}

func TestHealthChecker_DoubleStart(t *testing.T) {
	store := createTestStore(t, "root-session")

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithHealthCheckInterval(50*time.Millisecond))

	ctx := context.Background()
	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to be running after first start")
	}

	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to still be running after double start")
	}

	checker.Stop()
}

func TestHealthChecker_DoubleStop(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "state.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	tmux := newTestTmuxChecker()
	checker := NewHealthChecker(store, tmux, nil)

	checker.Stop()
	if checker.IsRunning() {
		t.Error("expected checker not running after stop without start")
	}

	checker.Stop()
	if checker.IsRunning() {
		t.Error("expected checker not running after double stop")
	}
}

func TestHealthChecker_ContextCancellation(t *testing.T) {
	store := createTestStore(t, "root-session")

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithHealthCheckInterval(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to be running")
	}

	time.Sleep(75 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	if checker.IsRunning() {
		t.Error("expected checker to stop after context cancellation")
	}
}
