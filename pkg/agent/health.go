// Package agent provides agent lifecycle management.
package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rpuneet/bc/pkg/events"
	"github.com/rpuneet/bc/pkg/log"
)

const (
	// DefaultHealthCheckInterval is how often health checks run.
	DefaultHealthCheckInterval = 30 * time.Second
	// DefaultStaleThreshold is how long before root state is considered stale.
	DefaultStaleThreshold = 60 * time.Second
)

// TmuxChecker interface for checking tmux session status.
type TmuxChecker interface {
	HasSession(ctx context.Context, name string) bool
}

// HealthStatus represents the current health of the root agent.
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult contains the outcome of a single health check.
type HealthCheckResult struct {
	LastUpdated  time.Time
	CheckedAt    time.Time
	ErrorMessage string
	Status       HealthStatus
	TmuxAlive    bool
	StateFresh   bool
}

// HealthChecker monitors root agent health and triggers recovery.
type HealthChecker struct {
	store          *SQLiteStore
	tmux           TmuxChecker
	eventLog       events.EventStore
	onUnhealthy    func(*HealthCheckResult) // callback when unhealthy detected
	lastResult     *HealthCheckResult
	stopCh         chan struct{}
	interval       time.Duration
	staleThreshold time.Duration
	mu             sync.RWMutex
	running        bool
}

// HealthCheckerOption configures the HealthChecker.
type HealthCheckerOption func(*HealthChecker)

// WithHealthCheckInterval sets the check interval.
func WithHealthCheckInterval(d time.Duration) HealthCheckerOption {
	return func(h *HealthChecker) {
		h.interval = d
	}
}

// WithStaleThreshold sets how long before state is considered stale.
func WithStaleThreshold(d time.Duration) HealthCheckerOption {
	return func(h *HealthChecker) {
		h.staleThreshold = d
	}
}

// WithUnhealthyCallback sets the callback for unhealthy detection.
func WithUnhealthyCallback(fn func(*HealthCheckResult)) HealthCheckerOption {
	return func(h *HealthChecker) {
		h.onUnhealthy = fn
	}
}

// NewHealthChecker creates a new health checker for the root agent.
func NewHealthChecker(store *SQLiteStore, tmux TmuxChecker, eventLog events.EventStore, opts ...HealthCheckerOption) *HealthChecker {
	h := &HealthChecker{
		store:          store,
		tmux:           tmux,
		eventLog:       eventLog,
		interval:       DefaultHealthCheckInterval,
		staleThreshold: DefaultStaleThreshold,
		stopCh:         make(chan struct{}),
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Check performs a single health check and returns the result.
func (h *HealthChecker) Check(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		CheckedAt: time.Now(),
	}

	// Load root state from SQLite
	root, err := h.store.LoadRoot()
	if err != nil || root == nil {
		result.Status = HealthStatusUnhealthy
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("failed to load root state: %v", err)
		} else {
			result.ErrorMessage = "root agent not found"
		}
		h.updateLastResult(result)
		return result
	}

	// Check state freshness
	result.LastUpdated = root.UpdatedAt
	staleDuration := time.Since(root.UpdatedAt)
	result.StateFresh = staleDuration < h.staleThreshold

	// Check tmux session
	if root.Session != "" {
		result.TmuxAlive = h.tmux.HasSession(ctx, root.Session)
	}

	// Determine overall status
	switch {
	case !result.TmuxAlive:
		result.Status = HealthStatusUnhealthy
		result.ErrorMessage = "tmux session not found or unresponsive"
	case !result.StateFresh:
		result.Status = HealthStatusDegraded
		result.ErrorMessage = fmt.Sprintf("root state stale (last updated %v ago)", staleDuration.Round(time.Second))
	default:
		result.Status = HealthStatusHealthy
	}

	h.updateLastResult(result)
	return result
}

// updateLastResult stores the result thread-safely.
func (h *HealthChecker) updateLastResult(result *HealthCheckResult) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastResult = result
}

// LastResult returns the most recent health check result.
func (h *HealthChecker) LastResult() *HealthCheckResult {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.lastResult
}

// Start begins periodic health checks in a goroutine.
func (h *HealthChecker) Start(ctx context.Context) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.stopCh = make(chan struct{})
	h.mu.Unlock()

	go h.loop(ctx)
}

// Stop halts periodic health checks.
func (h *HealthChecker) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running {
		return
	}
	h.running = false
	close(h.stopCh)
}

// IsRunning returns whether the health checker is actively running.
func (h *HealthChecker) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

// loop runs the periodic health check.
func (h *HealthChecker) loop(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	// Do an initial check immediately
	h.runCheck(ctx)

	for {
		select {
		case <-ctx.Done():
			h.Stop()
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.runCheck(ctx)
		}
	}
}

// runCheck performs a check and handles the result.
func (h *HealthChecker) runCheck(ctx context.Context) {
	result := h.Check(ctx)

	// Emit event
	h.emitHealthEvent(result)

	// Trigger callback if unhealthy
	if result.Status == HealthStatusUnhealthy && h.onUnhealthy != nil {
		h.onUnhealthy(result)
	}
}

// emitHealthEvent logs the health check result to the event log.
func (h *HealthChecker) emitHealthEvent(result *HealthCheckResult) {
	if h.eventLog == nil {
		return
	}

	var eventType events.EventType
	switch result.Status {
	case HealthStatusUnhealthy:
		eventType = events.HealthFailed
	case HealthStatusHealthy:
		eventType = events.HealthCheck
	default:
		eventType = events.HealthCheck
	}

	if err := h.eventLog.Append(events.Event{
		Type:    eventType,
		Agent:   "root",
		Message: result.ErrorMessage,
		Data: map[string]any{
			"status":       string(result.Status),
			"tmux_alive":   result.TmuxAlive,
			"state_fresh":  result.StateFresh,
			"last_updated": result.LastUpdated.Format(time.RFC3339),
		},
	}); err != nil {
		log.Warn("failed to log health event", "error", err)
	}
}
