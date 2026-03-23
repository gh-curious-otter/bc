package stats

import (
	"context"
	"fmt"
)

// RecordSystem inserts a system metric sample.
func (s *Store) RecordSystem(ctx context.Context, m SystemMetric) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO system_metrics (time, cpu_percent, mem_bytes, mem_percent, disk_bytes, goroutines, hostname)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.Time, m.CPUPercent, m.MemBytes, m.MemPercent, m.DiskBytes, m.Goroutines, m.Hostname,
	)
	if err != nil {
		return fmt.Errorf("record system metric: %w", err)
	}
	return nil
}

// RecordAgent inserts an agent metric sample.
func (s *Store) RecordAgent(ctx context.Context, m AgentMetric) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO agent_metrics (time, agent_name, agent_id, role, state, cpu_pct, mem_bytes, uptime_sec)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		m.Time, m.AgentName, m.AgentID, m.Role, m.State, m.CPUPct, m.MemBytes, m.UptimeSec,
	)
	if err != nil {
		return fmt.Errorf("record agent metric: %w", err)
	}
	return nil
}

// RecordToken inserts a token usage sample.
func (s *Store) RecordToken(ctx context.Context, m TokenMetric) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO token_metrics (time, agent_id, agent_name, provider, model, input_tokens, output_tokens, cost_usd)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		m.Time, m.AgentID, m.AgentName, m.Provider, m.Model, m.InputTokens, m.OutputTokens, m.CostUSD,
	)
	if err != nil {
		return fmt.Errorf("record token metric: %w", err)
	}
	return nil
}

// RecordChannel inserts a channel activity sample.
func (s *Store) RecordChannel(ctx context.Context, m ChannelMetric) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO channel_metrics (time, channel_name, messages_sent, messages_read, participants)
		 VALUES ($1, $2, $3, $4, $5)`,
		m.Time, m.ChannelName, m.MessagesSent, m.MessagesRead, m.Participants,
	)
	if err != nil {
		return fmt.Errorf("record channel metric: %w", err)
	}
	return nil
}

// RecordDaemon inserts a daemon process sample.
func (s *Store) RecordDaemon(ctx context.Context, m DaemonMetric) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO daemon_metrics (time, daemon_name, state, pid, cpu_pct, mem_bytes, restarts)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		m.Time, m.DaemonName, m.State, m.PID, m.CPUPct, m.MemBytes, m.Restarts,
	)
	if err != nil {
		return fmt.Errorf("record daemon metric: %w", err)
	}
	return nil
}
