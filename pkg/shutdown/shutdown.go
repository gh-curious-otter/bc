// Package shutdown provides graceful shutdown handling for bc.
//
// The shutdown package manages cleanup during process termination,
// ensuring resources are properly released when SIGINT/SIGTERM is received.
//
// Features:
//   - Signal handler registration (SIGINT, SIGTERM)
//   - Cleanup function registration with priorities
//   - Timeout-based graceful shutdown
//   - Context cancellation propagation
//
// Usage:
//
//	// Register cleanup handlers
//	shutdown.OnShutdown(shutdown.PriorityHigh, func(ctx context.Context) error {
//	    return closeDatabase()
//	})
//
//	// Start shutdown handling (blocks until signal received)
//	shutdown.Wait()
//
// Issue #1660: Graceful shutdown handling
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// Priority determines the order of cleanup execution.
// Lower values execute first.
type Priority int

const (
	// PriorityCritical runs first (e.g., save state)
	PriorityCritical Priority = 0
	// PriorityHigh runs early (e.g., stop accepting new work)
	PriorityHigh Priority = 10
	// PriorityNormal is the default priority
	PriorityNormal Priority = 50
	// PriorityLow runs late (e.g., close connections)
	PriorityLow Priority = 90
	// PriorityFinal runs last (e.g., flush logs)
	PriorityFinal Priority = 100
)

// DefaultTimeout is the default time to wait for cleanup.
const DefaultTimeout = 30 * time.Second

// CleanupFunc is a function called during shutdown.
// It receives a context that will be canceled when the timeout expires.
type CleanupFunc func(ctx context.Context) error

// handler stores a cleanup function with its priority.
type handler struct {
	fn       CleanupFunc
	name     string
	priority Priority
}

// Manager coordinates graceful shutdown.
//
//nolint:govet // fieldalignment: logical grouping preferred
type Manager struct {
	mu         sync.Mutex
	handlers   []handler
	timeout    time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownCh chan struct{}
	started    bool
	signals    []os.Signal
}

// global is the default shutdown manager.
var global = New()

// New creates a new shutdown manager.
func New() *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{
		handlers:   make([]handler, 0),
		timeout:    DefaultTimeout,
		ctx:        ctx,
		cancel:     cancel,
		shutdownCh: make(chan struct{}),
		signals:    []os.Signal{syscall.SIGINT, syscall.SIGTERM},
	}
}

// OnShutdown registers a cleanup function with the given priority.
// Functions with lower priority values are called first.
func OnShutdown(priority Priority, fn CleanupFunc) {
	global.OnShutdown(priority, "", fn)
}

// OnShutdownNamed registers a named cleanup function.
func OnShutdownNamed(priority Priority, name string, fn CleanupFunc) {
	global.OnShutdown(priority, name, fn)
}

// SetTimeout sets the maximum time to wait for cleanup.
func SetTimeout(d time.Duration) {
	global.SetTimeout(d)
}

// Context returns a context that is canceled on shutdown.
func Context() context.Context {
	return global.Context()
}

// Start begins listening for shutdown signals.
// This should be called early in the application lifecycle.
func Start() {
	global.Start()
}

// Wait blocks until a shutdown signal is received, then runs cleanup.
func Wait() {
	global.Wait()
}

// Trigger initiates shutdown programmatically.
func Trigger() {
	global.Trigger()
}

// OnShutdown registers a cleanup function.
func (m *Manager) OnShutdown(priority Priority, name string, fn CleanupFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler{
		priority: priority,
		name:     name,
		fn:       fn,
	})
}

// SetTimeout sets the cleanup timeout.
func (m *Manager) SetTimeout(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = d
}

// Context returns the shutdown context.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Start begins signal handling.
func (m *Manager) Start() {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	m.mu.Unlock()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, m.signals...)

	go func() {
		select {
		case sig := <-sigCh:
			log.Info("received shutdown signal", "signal", sig.String())
			m.runCleanup()
		case <-m.shutdownCh:
			// Triggered programmatically
			m.runCleanup()
		}
	}()
}

// Wait blocks until shutdown completes.
func (m *Manager) Wait() {
	m.Start()
	<-m.ctx.Done()
}

// Trigger initiates shutdown.
func (m *Manager) Trigger() {
	select {
	case m.shutdownCh <- struct{}{}:
	default:
		// Already shutting down
	}
}

// runCleanup executes all registered cleanup functions.
func (m *Manager) runCleanup() {
	m.mu.Lock()
	handlers := make([]handler, len(m.handlers))
	copy(handlers, m.handlers)
	timeout := m.timeout
	m.mu.Unlock()

	// Sort by priority (lower first)
	sort.Slice(handlers, func(i, j int) bool {
		return handlers[i].priority < handlers[j].priority
	})

	// Create timeout context
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Info("starting graceful shutdown", "handlers", len(handlers), "timeout", timeout.String())

	// Run cleanup handlers
	var wg sync.WaitGroup
	var errorCount atomic.Int32

	for _, h := range handlers {
		wg.Add(1)
		go func(h handler) {
			defer wg.Done()
			name := h.name
			if name == "" {
				name = "unnamed"
			}
			log.Debug("running cleanup handler", "name", name, "priority", h.priority)
			if err := h.fn(ctx); err != nil {
				log.Warn("cleanup handler failed", "name", name, "error", err)
				errorCount.Add(1)
			}
		}(h)
	}

	// Wait for all handlers or timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if errorCount.Load() > 0 {
			log.Info("graceful shutdown complete", "errors", errorCount.Load())
		} else {
			log.Info("graceful shutdown complete")
		}
	case <-ctx.Done():
		log.Warn("shutdown timeout exceeded, forcing exit")
	}

	// Cancel the main context to signal shutdown complete
	m.cancel()
}

// Reset clears all handlers (mainly for testing).
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = make([]handler, 0)
	m.started = false

	// Recreate context
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.shutdownCh = make(chan struct{})
}

// HandlerCount returns the number of registered handlers.
func (m *Manager) HandlerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.handlers)
}
