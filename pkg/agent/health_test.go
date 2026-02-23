package agent

import (
	"context"
	"os"
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

func TestHealthChecker_Check_Healthy(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root with session
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "root-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

	// Mock tmux with alive session
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
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root with session
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "root-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

	// Mock tmux with dead session
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
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root first (this sets fresh timestamp)
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "root-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

	// Now manually write stale state by bypassing Save()
	// This simulates a state file that hasn't been updated in a while
	staleStartedAt := time.Now().Add(-5 * time.Minute)
	staleUpdatedAt := time.Now().Add(-2 * time.Minute) // 2 minutes ago
	staleJSON := `{"name":"root","role":"manager","tool":"","state":"idle","session":"root-session","started_at":"` +
		staleStartedAt.Format(time.RFC3339Nano) + `","updated_at":"` +
		staleUpdatedAt.Format(time.RFC3339Nano) + `","is_singleton":true}`

	rootPath := filepath.Join(bcDir, "agents", "root.json")
	if err := writeTestFile(t, rootPath, staleJSON); err != nil {
		t.Fatalf("failed to write stale state: %v", err)
	}

	// Mock tmux with alive session
	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithStaleThreshold(30*time.Second)) // 30s threshold
	result := checker.Check(context.Background())

	if result.Status != HealthStatusDegraded {
		t.Errorf("expected degraded, got %s", result.Status)
	}
	if result.StateFresh {
		t.Error("expected StateFresh to be false")
	}
}

// writeTestFile is a helper to write test files.
func writeTestFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0600)
}

func TestHealthChecker_Check_NoRootState(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

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
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root with dead session
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "dead-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

	tmux := newTestTmuxChecker() // no sessions = dead

	var callbackCalled atomic.Bool
	callback := func(result *HealthCheckResult) {
		callbackCalled.Store(true)
	}

	checker := NewHealthChecker(store, tmux, nil,
		WithUnhealthyCallback(callback))

	// Trigger a check via runCheck (which calls callback)
	checker.runCheck(context.Background())

	if !callbackCalled.Load() {
		t.Error("expected unhealthy callback to be called")
	}
}

func TestHealthChecker_StartStop(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create healthy root
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "root-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

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

	// Wait for at least one check
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
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root
	state, createErr := store.Create("root", RoleRoot, "claude")
	if createErr != nil {
		t.Fatalf("failed to create root: %v", createErr)
	}
	state.Session = "root-session"
	if saveErr := store.Save(state); saveErr != nil {
		t.Fatalf("failed to save root: %v", saveErr)
	}

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
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root
	_, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}

	tmux := newTestTmuxChecker()
	checker := NewHealthChecker(store, tmux, nil)

	// Before any check, LastResult should be nil
	if checker.LastResult() != nil {
		t.Error("expected nil before first check")
	}

	checker.Check(context.Background())

	result := checker.LastResult()
	if result == nil {
		t.Error("expected result after check")
	}
}

// --- Edge case tests for Start/Stop ---

func TestHealthChecker_DoubleStart(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "root-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithHealthCheckInterval(50*time.Millisecond))

	ctx := context.Background()

	// Start first time
	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to be running after first start")
	}

	// Start second time - should be no-op
	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to still be running after double start")
	}

	checker.Stop()
}

func TestHealthChecker_DoubleStop(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	tmux := newTestTmuxChecker()
	checker := NewHealthChecker(store, tmux, nil)

	// Stop without starting - should be no-op
	checker.Stop()
	if checker.IsRunning() {
		t.Error("expected checker not running after stop without start")
	}

	// Double stop - should be no-op
	checker.Stop()
	if checker.IsRunning() {
		t.Error("expected checker not running after double stop")
	}
}

func TestHealthChecker_ContextCancellation(t *testing.T) {
	dir := t.TempDir()
	bcDir := filepath.Join(dir, ".bc")
	store := NewRootStateStore(bcDir)

	// Create root
	state, err := store.Create("root", RoleRoot, "claude")
	if err != nil {
		t.Fatalf("failed to create root: %v", err)
	}
	state.Session = "root-session"
	if err := store.Save(state); err != nil {
		t.Fatalf("failed to save root: %v", err)
	}

	tmux := newTestTmuxChecker()
	tmux.SetSession("root-session", true)

	checker := NewHealthChecker(store, tmux, nil,
		WithHealthCheckInterval(50*time.Millisecond))

	ctx, cancel := context.WithCancel(context.Background())

	checker.Start(ctx)
	if !checker.IsRunning() {
		t.Error("expected checker to be running")
	}

	// Wait for at least one check
	time.Sleep(75 * time.Millisecond)

	// Cancel context - should stop the checker
	cancel()

	// Give time for the loop to notice cancellation
	time.Sleep(100 * time.Millisecond)

	if checker.IsRunning() {
		t.Error("expected checker to stop after context cancellation")
	}
}
