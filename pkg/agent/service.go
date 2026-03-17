package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rpuneet/bc/pkg/log"
)

// EventPublisher is the interface for publishing agent lifecycle events.
type EventPublisher interface {
	Publish(eventType string, data map[string]any)
}

// CostQuerier is the interface for querying agent cost data.
type CostQuerier interface {
	AgentCostSummary(agentID string) (*CostSummary, error)
}

// CostSummary holds cost breakdown for an agent.
type CostSummary struct {
	AgentID      string  `json:"agent_id"`
	InputTokens  int64   `json:"input_tokens"`
	OutputTokens int64   `json:"output_tokens"`
	TotalTokens  int64   `json:"total_tokens"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	RequestCount int64   `json:"request_count"`
}

// ListOptions configures agent listing.
type ListOptions struct {
	Role   string // Filter by role (empty = all)
	Status string // Filter by status/state (empty = all)
}

// CreateOptions holds parameters for creating an agent via the service.
type CreateOptions struct {
	Name    string
	Role    Role
	Tool    string
	TTL     int    // Auto-stop after N seconds (0 = no TTL)
	EnvFile string
	Runtime string
	Parent  string
}

// StartOptions configures agent start behavior.
type StartOptions struct {
	Fresh   bool   // Force new session (ignore session_id)
	Runtime string // Runtime backend override
}

// AgentService provides the application-level API for agent management.
// It wraps Manager with event publishing, cost queries, and TTL enforcement.
// This is the boundary that the daemon (issue #1938) will use.
type AgentService struct {
	manager *Manager
	events  EventPublisher
	costs   CostQuerier

	stopCh chan struct{}
	mu     sync.RWMutex
}

// NewAgentService creates a new agent service wrapping the given manager.
func NewAgentService(mgr *Manager, events EventPublisher, costs CostQuerier) *AgentService {
	return &AgentService{
		manager: mgr,
		events:  events,
		costs:   costs,
		stopCh:  make(chan struct{}),
	}
}

// Manager returns the underlying agent manager.
func (s *AgentService) Manager() *Manager {
	return s.manager
}

// List returns agents matching the given options.
func (s *AgentService) List(ctx context.Context, opts ListOptions) ([]*Agent, error) {
	agents := s.manager.ListAgents()

	// Apply filters
	if opts.Role == "" && opts.Status == "" {
		return agents, nil
	}

	filtered := make([]*Agent, 0, len(agents))
	for _, a := range agents {
		if opts.Role != "" && string(a.Role) != opts.Role {
			continue
		}
		if opts.Status != "" && !matchesStatus(a.State, opts.Status) {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered, nil
}

// matchesStatus checks if an agent state matches a status filter.
// Maps the detailed internal states to the simplified 4-state model.
func matchesStatus(state State, status string) bool {
	switch status {
	case "running":
		return state == StateIdle || state == StateWorking || state == StateStarting
	case "stopped":
		return state == StateStopped
	case "error":
		return state == StateError
	case "starting":
		return state == StateStarting
	default:
		// Allow matching by exact internal state name too
		return string(state) == status
	}
}

// Create creates a new agent.
func (s *AgentService) Create(ctx context.Context, opts CreateOptions) (*Agent, error) {
	a, err := s.manager.SpawnAgentWithOptions(SpawnOptions{
		Name:      opts.Name,
		Role:      opts.Role,
		Workspace: s.manager.workspacePath,
		ParentID:  opts.Parent,
		Tool:      opts.Tool,
		EnvFile:   opts.EnvFile,
		Runtime:   opts.Runtime,
		TTL:       opts.TTL,
	})
	if err != nil {
		return nil, err
	}

	s.publishEvent("agent.created", map[string]any{
		"name": a.Name,
		"role": string(a.Role),
		"tool": a.Tool,
	})

	return a, nil
}

// Start starts a stopped agent, optionally with a fresh session.
func (s *AgentService) Start(ctx context.Context, name string, opts StartOptions) (*Agent, error) {
	existing := s.manager.GetAgent(name)
	if existing == nil {
		return nil, fmt.Errorf("agent %q not found", name)
	}

	if existing.State != StateStopped && existing.State != StateError {
		return nil, fmt.Errorf("agent %q is already running (state: %s)", name, existing.State)
	}

	a, err := s.manager.SpawnAgentWithOptions(SpawnOptions{
		Name:      name,
		Role:      existing.Role,
		Workspace: s.manager.workspacePath,
		ParentID:  existing.ParentID,
		Tool:      existing.Tool,
		EnvFile:   existing.EnvFile,
		Runtime:   opts.Runtime,
		Fresh:     opts.Fresh,
	})
	if err != nil {
		return nil, err
	}

	s.publishEvent("agent.started", map[string]any{
		"name":       a.Name,
		"session_id": a.SessionID,
	})

	return a, nil
}

// Stop stops a running agent.
func (s *AgentService) Stop(ctx context.Context, name string) error {
	if err := s.manager.StopAgent(name); err != nil {
		return err
	}

	s.publishEvent("agent.stopped", map[string]any{
		"name":   name,
		"reason": "user_request",
	})

	return nil
}

// Delete permanently removes an agent. Agent must be stopped first.
func (s *AgentService) Delete(ctx context.Context, name string) error {
	a := s.manager.GetAgent(name)
	if a == nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if a.State != StateStopped {
		return fmt.Errorf("agent %q must be stopped before deletion (state: %s)", name, a.State)
	}

	if err := s.manager.DeleteAgent(name); err != nil {
		return err
	}

	s.publishEvent("agent.deleted", map[string]any{
		"name": name,
	})

	return nil
}

// Send sends a message to a running agent.
func (s *AgentService) Send(ctx context.Context, name, message string) error {
	a := s.manager.GetAgent(name)
	if a == nil {
		return fmt.Errorf("agent %q not found", name)
	}
	if a.State == StateStopped {
		return fmt.Errorf("agent %q is stopped", name)
	}
	return s.manager.SendToAgent(name, message)
}

// Peek returns recent output from an agent.
func (s *AgentService) Peek(ctx context.Context, name string, lines int) (string, error) {
	a := s.manager.GetAgent(name)
	if a == nil {
		return "", fmt.Errorf("agent %q not found", name)
	}
	return s.manager.CaptureOutput(name, lines)
}

// Cost returns the cost summary for an agent.
func (s *AgentService) Cost(ctx context.Context, name string) (*CostSummary, error) {
	if s.costs == nil {
		return nil, fmt.Errorf("cost tracking not configured")
	}
	return s.costs.AgentCostSummary(name)
}

// Broadcast sends a message to all running agents.
// Returns the number of agents the message was sent to.
func (s *AgentService) Broadcast(ctx context.Context, message string) (int, error) {
	agents := s.manager.ListAgents()
	sent := 0
	for _, a := range agents {
		if a.State == StateStopped || a.State == StateError {
			continue
		}
		if err := s.manager.SendToAgent(a.Name, message); err != nil {
			log.Warn("broadcast: failed to send to agent", "agent", a.Name, "error", err)
			continue
		}
		sent++
	}
	return sent, nil
}

// Refresh refreshes agent states from runtime backends.
func (s *AgentService) Refresh() error {
	return s.manager.RefreshState()
}

// Get returns a single agent by name.
func (s *AgentService) Get(ctx context.Context, name string) (*Agent, error) {
	a := s.manager.GetAgent(name)
	if a == nil {
		return nil, fmt.Errorf("agent %q not found", name)
	}
	return a, nil
}

// StartTTLWatcher starts a background goroutine that auto-stops agents
// whose TTL has expired. Call Stop() to terminate the watcher.
func (s *AgentService) StartTTLWatcher(ctx context.Context) {
	go s.ttlWatchLoop(ctx)
}

// StopWatcher terminates the TTL watcher.
func (s *AgentService) StopWatcher() {
	close(s.stopCh)
}

func (s *AgentService) ttlWatchLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkTTLs(ctx)
		}
	}
}

func (s *AgentService) checkTTLs(ctx context.Context) {
	agents := s.manager.ListAgents()
	now := time.Now()

	for _, a := range agents {
		if a.TTL <= 0 || a.State == StateStopped {
			continue
		}

		elapsed := now.Sub(a.StartedAt)
		ttlDuration := time.Duration(a.TTL) * time.Second
		if elapsed >= ttlDuration {
			log.Info("TTL expired, stopping agent", "agent", a.Name, "ttl", a.TTL, "elapsed", elapsed)
			if err := s.manager.StopAgent(a.Name); err != nil {
				log.Warn("TTL stop failed", "agent", a.Name, "error", err)
				continue
			}
			s.publishEvent("agent.stopped", map[string]any{
				"name":   a.Name,
				"reason": "ttl_expired",
			})
		}
	}
}

func (s *AgentService) publishEvent(eventType string, data map[string]any) {
	if s.events != nil {
		s.events.Publish(eventType, data)
	}
}
