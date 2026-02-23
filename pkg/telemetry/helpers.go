package telemetry

import (
	"context"
	"time"
)

// Helper functions for emitting common telemetry events.
// These provide type-safe, convenient APIs for instrumentation.

// AgentSpawn emits an agent spawn event.
func AgentSpawn(ctx context.Context, agent, role string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}
	data["role"] = role
	Emit(ctx, Event{
		Type:    EventAgentSpawn,
		Agent:   agent,
		Message: "agent spawned",
		Data:    data,
	})
}

// AgentStop emits an agent stop event.
func AgentStop(ctx context.Context, agent string, reason string) {
	Emit(ctx, Event{
		Type:    EventAgentStop,
		Agent:   agent,
		Message: "agent stopped",
		Data:    map[string]any{"reason": reason},
	})
}

// AgentStateChange emits an agent state change event.
func AgentStateChange(ctx context.Context, agent, fromState, toState string) {
	Emit(ctx, Event{
		Type:    EventAgentStateChange,
		Agent:   agent,
		Message: "agent state changed",
		Data: map[string]any{
			"from_state": fromState,
			"to_state":   toState,
		},
	})
}

// AgentError emits an agent error event.
func AgentError(ctx context.Context, agent string, err error, operation string) {
	Emit(ctx, Event{
		Type:    EventAgentError,
		Agent:   agent,
		Message: "agent error",
		Error:   err,
		Data:    map[string]any{"operation": operation},
	})
}

// ChannelSend emits a channel send event.
func ChannelSend(ctx context.Context, channel, sender, message string) {
	Emit(ctx, Event{
		Type:    EventChannelSend,
		Agent:   sender,
		Message: "message sent",
		Data: map[string]any{
			"channel":        channel,
			"message_length": len(message),
		},
	})
}

// ChannelReceive emits a channel receive event.
func ChannelReceive(ctx context.Context, channel, receiver string, messageCount int) {
	Emit(ctx, Event{
		Type:    EventChannelReceive,
		Agent:   receiver,
		Message: "messages received",
		Data: map[string]any{
			"channel":       channel,
			"message_count": messageCount,
		},
	})
}

// CostRecord emits a cost record event.
func CostRecord(ctx context.Context, agent, model string, inputTokens, outputTokens int, costUSD float64) {
	Emit(ctx, Event{
		Type:    EventCostRecord,
		Agent:   agent,
		Message: "cost recorded",
		Data: map[string]any{
			"model":         model,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"cost_usd":      costUSD,
		},
	})
}

// CostAlert emits a cost budget alert event.
func CostAlert(ctx context.Context, scope string, currentSpend, budgetLimit float64, percentUsed float64) {
	Emit(ctx, Event{
		Type:    EventCostAlert,
		Message: "budget alert",
		Data: map[string]any{
			"scope":         scope,
			"current_spend": currentSpend,
			"budget_limit":  budgetLimit,
			"percent_used":  percentUsed,
		},
	})
}

// CostExceeded emits a cost budget exceeded event.
func CostExceeded(ctx context.Context, scope string, currentSpend, budgetLimit float64) {
	Emit(ctx, Event{
		Type:    EventCostExceeded,
		Message: "budget exceeded",
		Data: map[string]any{
			"scope":         scope,
			"current_spend": currentSpend,
			"budget_limit":  budgetLimit,
		},
	})
}

// HealthCheck emits a health check event.
func HealthCheck(ctx context.Context, agent, status string, details map[string]any) {
	if details == nil {
		details = make(map[string]any)
	}
	details["status"] = status
	Emit(ctx, Event{
		Type:    EventHealthCheck,
		Agent:   agent,
		Message: "health check",
		Data:    details,
	})
}

// HealthUnhealthy emits an unhealthy status event.
func HealthUnhealthy(ctx context.Context, agent, reason string) {
	Emit(ctx, Event{
		Type:    EventHealthUnhealthy,
		Agent:   agent,
		Message: "agent unhealthy",
		Data:    map[string]any{"reason": reason},
	})
}

// HealthRecovered emits a health recovered event.
func HealthRecovered(ctx context.Context, agent string, downtime time.Duration) {
	Emit(ctx, Event{
		Type:    EventHealthRecovered,
		Agent:   agent,
		Message: "agent recovered",
		Data:    map[string]any{"downtime_seconds": downtime.Seconds()},
	})
}

// WorkAssigned emits a work assigned event.
func WorkAssigned(ctx context.Context, agent, task string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}
	data["task"] = task
	Emit(ctx, Event{
		Type:    EventWorkAssigned,
		Agent:   agent,
		Message: "work assigned",
		Data:    data,
	})
}

// WorkStarted emits a work started event.
func WorkStarted(ctx context.Context, agent, task string) {
	Emit(ctx, Event{
		Type:    EventWorkStarted,
		Agent:   agent,
		Message: "work started",
		Data:    map[string]any{"task": task},
	})
}

// WorkCompleted emits a work completed event.
func WorkCompleted(ctx context.Context, agent, task string, duration time.Duration) {
	Emit(ctx, Event{
		Type:     EventWorkCompleted,
		Agent:    agent,
		Message:  "work completed",
		Duration: duration,
		Data:     map[string]any{"task": task},
	})
}

// WorkFailed emits a work failed event.
func WorkFailed(ctx context.Context, agent, task string, err error) {
	Emit(ctx, Event{
		Type:    EventWorkFailed,
		Agent:   agent,
		Message: "work failed",
		Error:   err,
		Data:    map[string]any{"task": task},
	})
}

// Span creates a simple timing span for operations.
// Usage:
//
//	done := telemetry.Span(ctx, "operation_name", agent)
//	defer done()
func Span(ctx context.Context, operation, agent string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		Emit(ctx, Event{
			Type:     EventType("span." + operation),
			Agent:    agent,
			Message:  operation + " completed",
			Duration: duration,
		})
	}
}
