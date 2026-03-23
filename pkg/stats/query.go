package stats

import (
	"context"
	"fmt"
)

// QuerySystem returns system metrics aggregated by time bucket.
func (s *Store) QuerySystem(ctx context.Context, tr TimeRange) ([]SystemMetric, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        AVG(cpu_percent), AVG(mem_bytes)::BIGINT, AVG(mem_percent),
		        AVG(disk_bytes)::BIGINT, AVG(goroutines)::INT, MAX(hostname)
		 FROM system_metrics
		 WHERE time >= $2 AND time < $3
		 GROUP BY bucket
		 ORDER BY bucket`,
		tr.PGInterval(), tr.From, tr.To,
	)
	if err != nil {
		return nil, fmt.Errorf("query system metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is informational

	var result []SystemMetric
	for rows.Next() {
		var m SystemMetric
		if err := rows.Scan(&m.Time, &m.CPUPercent, &m.MemBytes, &m.MemPercent,
			&m.DiskBytes, &m.Goroutines, &m.Hostname); err != nil {
			return nil, fmt.Errorf("scan system metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryAgent returns metrics for a specific agent aggregated by time bucket.
func (s *Store) QueryAgent(ctx context.Context, agentName string, tr TimeRange) ([]AgentMetric, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        agent_name, MAX(agent_id), MAX(role), MAX(state),
		        AVG(cpu_pct), AVG(mem_bytes)::BIGINT, MAX(uptime_sec)
		 FROM agent_metrics
		 WHERE agent_name = $2 AND time >= $3 AND time < $4
		 GROUP BY bucket, agent_name
		 ORDER BY bucket`,
		tr.PGInterval(), agentName, tr.From, tr.To,
	)
	if err != nil {
		return nil, fmt.Errorf("query agent metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is informational

	var result []AgentMetric
	for rows.Next() {
		var m AgentMetric
		if err := rows.Scan(&m.Time, &m.AgentName, &m.AgentID, &m.Role, &m.State,
			&m.CPUPct, &m.MemBytes, &m.UptimeSec); err != nil {
			return nil, fmt.Errorf("scan agent metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryTokens returns aggregated token metrics by time bucket.
func (s *Store) QueryTokens(ctx context.Context, tr TimeRange) ([]TokenMetric, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        '' AS agent_id, '' AS agent_name, '' AS provider, '' AS model,
		        SUM(input_tokens), SUM(output_tokens), SUM(cost_usd)
		 FROM token_metrics
		 WHERE time >= $2 AND time < $3
		 GROUP BY bucket
		 ORDER BY bucket`,
		tr.PGInterval(), tr.From, tr.To,
	)
	if err != nil {
		return nil, fmt.Errorf("query token metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is informational

	var result []TokenMetric
	for rows.Next() {
		var m TokenMetric
		if err := rows.Scan(&m.Time, &m.AgentID, &m.AgentName, &m.Provider, &m.Model,
			&m.InputTokens, &m.OutputTokens, &m.CostUSD); err != nil {
			return nil, fmt.Errorf("scan token metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryTokensByAgent returns token metrics for a specific agent by time bucket.
func (s *Store) QueryTokensByAgent(ctx context.Context, agentID string, tr TimeRange) ([]TokenMetric, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        agent_id, MAX(agent_name), MAX(provider), MAX(model),
		        SUM(input_tokens), SUM(output_tokens), SUM(cost_usd)
		 FROM token_metrics
		 WHERE agent_id = $2 AND time >= $3 AND time < $4
		 GROUP BY bucket, agent_id
		 ORDER BY bucket`,
		tr.PGInterval(), agentID, tr.From, tr.To,
	)
	if err != nil {
		return nil, fmt.Errorf("query tokens by agent: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is informational

	var result []TokenMetric
	for rows.Next() {
		var m TokenMetric
		if err := rows.Scan(&m.Time, &m.AgentID, &m.AgentName, &m.Provider, &m.Model,
			&m.InputTokens, &m.OutputTokens, &m.CostUSD); err != nil {
			return nil, fmt.Errorf("scan token metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryChannels returns channel metrics aggregated by time bucket.
func (s *Store) QueryChannels(ctx context.Context, tr TimeRange) ([]ChannelMetric, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        channel_name, SUM(messages_sent), SUM(messages_read), MAX(participants)
		 FROM channel_metrics
		 WHERE time >= $2 AND time < $3
		 GROUP BY bucket, channel_name
		 ORDER BY bucket`,
		tr.PGInterval(), tr.From, tr.To,
	)
	if err != nil {
		return nil, fmt.Errorf("query channel metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is informational

	var result []ChannelMetric
	for rows.Next() {
		var m ChannelMetric
		if err := rows.Scan(&m.Time, &m.ChannelName, &m.MessagesSent, &m.MessagesRead,
			&m.Participants); err != nil {
			return nil, fmt.Errorf("scan channel metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryDaemons returns daemon metrics aggregated by time bucket.
func (s *Store) QueryDaemons(ctx context.Context, tr TimeRange) ([]DaemonMetric, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT time_bucket($1::interval, time) AS bucket,
		        daemon_name, MAX(state), MAX(pid), AVG(cpu_pct),
		        AVG(mem_bytes)::BIGINT, MAX(restarts)
		 FROM daemon_metrics
		 WHERE time >= $2 AND time < $3
		 GROUP BY bucket, daemon_name
		 ORDER BY bucket`,
		tr.PGInterval(), tr.From, tr.To,
	)
	if err != nil {
		return nil, fmt.Errorf("query daemon metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck // rows.Close error is informational

	var result []DaemonMetric
	for rows.Next() {
		var m DaemonMetric
		if err := rows.Scan(&m.Time, &m.DaemonName, &m.State, &m.PID, &m.CPUPct,
			&m.MemBytes, &m.Restarts); err != nil {
			return nil, fmt.Errorf("scan daemon metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// Summary returns current aggregate totals across all metric tables.
func (s *Store) Summary(ctx context.Context) (*StatsSummary, error) {
	var summary StatsSummary

	// Total distinct agents
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(DISTINCT agent_name) FROM agent_metrics`,
	).Scan(&summary.TotalAgents)
	if err != nil {
		return nil, fmt.Errorf("summary agents: %w", err)
	}

	// Total tokens and cost
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(input_tokens + output_tokens), 0),
		        COALESCE(SUM(cost_usd), 0)
		 FROM token_metrics`,
	).Scan(&summary.TotalTokens, &summary.TotalCostUSD)
	if err != nil {
		return nil, fmt.Errorf("summary tokens: %w", err)
	}

	// Total messages
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(messages_sent), 0) FROM channel_metrics`,
	).Scan(&summary.TotalMessages)
	if err != nil {
		return nil, fmt.Errorf("summary messages: %w", err)
	}

	// Latest system metrics for current CPU/mem
	err = s.db.QueryRowContext(ctx,
		`SELECT COALESCE(cpu_percent, 0), COALESCE(mem_bytes, 0)
		 FROM system_metrics
		 ORDER BY time DESC
		 LIMIT 1`,
	).Scan(&summary.CPUPercent, &summary.MemBytes)
	if err != nil {
		// No system metrics yet is not an error
		summary.CPUPercent = 0
		summary.MemBytes = 0
	}

	return &summary, nil
}
