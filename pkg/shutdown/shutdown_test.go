package shutdown

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestManager_OnShutdown(t *testing.T) {
	m := New()
	defer m.Reset()

	var called atomic.Bool
	m.OnShutdown(PriorityNormal, "test", func(_ context.Context) error {
		called.Store(true)
		return nil
	})

	if m.HandlerCount() != 1 {
		t.Errorf("expected 1 handler, got %d", m.HandlerCount())
	}
}

func TestManager_Priority(t *testing.T) {
	m := New()
	m.SetTimeout(5 * time.Second)

	var count atomic.Int32

	m.OnShutdown(PriorityLow, "low", func(_ context.Context) error {
		count.Add(1)
		return nil
	})

	m.OnShutdown(PriorityHigh, "high", func(_ context.Context) error {
		count.Add(1)
		return nil
	})

	m.OnShutdown(PriorityCritical, "critical", func(_ context.Context) error {
		count.Add(1)
		return nil
	})

	// Trigger shutdown and wait
	go func() {
		time.Sleep(50 * time.Millisecond)
		m.Trigger()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	m.Start()

	select {
	case <-m.Context().Done():
		// Shutdown complete
	case <-ctx.Done():
		t.Fatal("timeout waiting for shutdown")
	}

	// All handlers should have run
	if count.Load() != 3 {
		t.Errorf("expected 3 handlers to run, got %d", count.Load())
	}
}

func TestManager_Timeout(t *testing.T) {
	m := New()
	m.SetTimeout(100 * time.Millisecond)

	m.OnShutdown(PriorityNormal, "slow", func(ctx context.Context) error {
		select {
		case <-time.After(5 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	start := time.Now()

	go func() {
		time.Sleep(10 * time.Millisecond)
		m.Trigger()
	}()

	m.Start()

	select {
	case <-m.Context().Done():
		// Shutdown complete
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown took too long")
	}

	elapsed := time.Since(start)
	if elapsed > 500*time.Millisecond {
		t.Errorf("shutdown should have timed out faster, took %v", elapsed)
	}
}

func TestManager_ErrorHandling(t *testing.T) {
	m := New()
	m.SetTimeout(5 * time.Second)

	var successCalled atomic.Bool
	testErr := errors.New("test error")

	m.OnShutdown(PriorityHigh, "failing", func(_ context.Context) error {
		return testErr
	})

	m.OnShutdown(PriorityLow, "success", func(_ context.Context) error {
		successCalled.Store(true)
		return nil
	})

	go func() {
		time.Sleep(10 * time.Millisecond)
		m.Trigger()
	}()

	m.Start()

	select {
	case <-m.Context().Done():
		// Shutdown complete
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for shutdown")
	}

	// Success handler should still run even if another fails
	if !successCalled.Load() {
		t.Error("success handler should have been called despite other handler failing")
	}
}

func TestManager_Context(t *testing.T) {
	m := New()

	ctx := m.Context()
	if ctx == nil {
		t.Fatal("context should not be nil")
	}

	select {
	case <-ctx.Done():
		t.Error("context should not be done before shutdown")
	default:
		// Expected
	}
}

func TestManager_Reset(t *testing.T) {
	m := New()

	m.OnShutdown(PriorityNormal, "test", func(_ context.Context) error {
		return nil
	})

	if m.HandlerCount() != 1 {
		t.Error("expected 1 handler before reset")
	}

	m.Reset()

	if m.HandlerCount() != 0 {
		t.Error("expected 0 handlers after reset")
	}
}

func TestManager_MultipleStart(t *testing.T) {
	m := New()
	m.SetTimeout(time.Second)

	// Starting multiple times should be safe
	m.Start()
	m.Start()
	m.Start()

	// Give the goroutine time to start
	time.Sleep(10 * time.Millisecond)

	// Trigger shutdown
	m.Trigger()

	select {
	case <-m.Context().Done():
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown should complete")
	}
}

func TestGlobalFunctions(t *testing.T) {
	// Reset global state
	global = New()
	defer func() { global = New() }()

	var called atomic.Bool
	OnShutdown(PriorityNormal, func(_ context.Context) error {
		called.Store(true)
		return nil
	})

	SetTimeout(time.Second)

	ctx := Context()
	if ctx == nil {
		t.Fatal("Context() should return non-nil")
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		Trigger()
	}()

	Start()

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("global shutdown should complete")
	}

	if !called.Load() {
		t.Error("global handler should have been called")
	}
}
