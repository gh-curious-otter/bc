package stats

import (
	"context"
	"fmt"
	"strings"
)

// QuerySystemCPU returns CPU metrics for system containers.
func (s *Store) QuerySystemCPU(ctx context.Context, systems []string, tr TimeRange) ([]SystemMetric, error) {
	return s.querySystem(ctx, systems, tr)
}

// QuerySystemMem returns memory metrics for system containers.
func (s *Store) QuerySystemMem(ctx context.Context, systems []string, tr TimeRange) ([]SystemMetric, error) {
	return s.querySystem(ctx, systems, tr)
}

// QuerySystemNet returns network metrics for system containers.
func (s *Store) QuerySystemNet(ctx context.Context, systems []string, tr TimeRange) ([]SystemMetric, error) {
	return s.querySystem(ctx, systems, tr)
}

// QuerySystemDisk returns disk metrics for system containers.
func (s *Store) QuerySystemDisk(ctx context.Context, systems []string, tr TimeRange) ([]SystemMetric, error) {
	return s.querySystem(ctx, systems, tr)
}

func (s *Store) querySystem(ctx context.Context, systems []string, tr TimeRange) ([]SystemMetric, error) {
	query := `SELECT time_bucket($1::interval, time) AS bucket,
		system_name,
		AVG(cpu_percent), AVG(mem_used_bytes)::BIGINT, AVG(mem_limit_bytes)::BIGINT, AVG(mem_percent),
		AVG(net_rx_bytes)::BIGINT, AVG(net_tx_bytes)::BIGINT,
		AVG(disk_read_bytes)::BIGINT, AVG(disk_write_bytes)::BIGINT
	FROM system_metrics
	WHERE time >= $2 AND time < $3`

	args := []any{tr.PGInterval(), tr.From, tr.To}

	if len(systems) > 0 {
		placeholders := make([]string, len(systems))
		for i, sys := range systems {
			args = append(args, sys)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		query += ` AND system_name IN (` + strings.Join(placeholders, ",") + `)`
	}

	query += ` GROUP BY bucket, system_name ORDER BY bucket`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query system metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []SystemMetric
	for rows.Next() {
		var m SystemMetric
		if err := rows.Scan(&m.Time, &m.SystemName, &m.CPUPercent, &m.MemUsedBytes, &m.MemLimitBytes, &m.MemPercent, &m.NetRxBytes, &m.NetTxBytes, &m.DiskReadBytes, &m.DiskWriteBytes); err != nil {
			return nil, fmt.Errorf("scan system metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// AgentFilter specifies which agents to query.
type AgentFilter struct {
	Agent   []string
	Role    string
	Tool    string
	Runtime string
}

// QueryAgentCPU returns CPU metrics for agents.
func (s *Store) QueryAgentCPU(ctx context.Context, f AgentFilter, tr TimeRange) ([]AgentMetric, error) {
	return s.queryAgent(ctx, f, tr)
}

// QueryAgentMem returns memory metrics for agents.
func (s *Store) QueryAgentMem(ctx context.Context, f AgentFilter, tr TimeRange) ([]AgentMetric, error) {
	return s.queryAgent(ctx, f, tr)
}

// QueryAgentNet returns network metrics for agents.
func (s *Store) QueryAgentNet(ctx context.Context, f AgentFilter, tr TimeRange) ([]AgentMetric, error) {
	return s.queryAgent(ctx, f, tr)
}

// QueryAgentDisk returns disk metrics for agents.
func (s *Store) QueryAgentDisk(ctx context.Context, f AgentFilter, tr TimeRange) ([]AgentMetric, error) {
	return s.queryAgent(ctx, f, tr)
}

func (s *Store) queryAgent(ctx context.Context, f AgentFilter, tr TimeRange) ([]AgentMetric, error) {
	query := `SELECT time_bucket($1::interval, time) AS bucket,
		agent_name, MAX(role), MAX(tool), MAX(runtime), MAX(state),
		AVG(cpu_percent), AVG(mem_used_bytes)::BIGINT, AVG(mem_limit_bytes)::BIGINT, AVG(mem_percent),
		AVG(net_rx_bytes)::BIGINT, AVG(net_tx_bytes)::BIGINT,
		AVG(disk_read_bytes)::BIGINT, AVG(disk_write_bytes)::BIGINT
	FROM agent_metrics
	WHERE time >= $2 AND time < $3`

	args := []any{tr.PGInterval(), tr.From, tr.To}

	if len(f.Agent) > 0 {
		ph := make([]string, len(f.Agent))
		for i, a := range f.Agent {
			args = append(args, a)
			ph[i] = fmt.Sprintf("$%d", len(args))
		}
		query += ` AND agent_name IN (` + strings.Join(ph, ",") + `)`
	}
	if f.Role != "" {
		args = append(args, f.Role)
		query += fmt.Sprintf(` AND role = $%d`, len(args))
	}
	if f.Tool != "" {
		args = append(args, f.Tool)
		query += fmt.Sprintf(` AND tool = $%d`, len(args))
	}
	if f.Runtime != "" {
		args = append(args, f.Runtime)
		query += fmt.Sprintf(` AND runtime = $%d`, len(args))
	}

	query += ` GROUP BY bucket, agent_name ORDER BY bucket`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query agent metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []AgentMetric
	for rows.Next() {
		var m AgentMetric
		if err := rows.Scan(&m.Time, &m.AgentName, &m.Role, &m.Tool, &m.Runtime, &m.State,
			&m.CPUPercent, &m.MemUsedBytes, &m.MemLimitBytes, &m.MemPercent,
			&m.NetRxBytes, &m.NetTxBytes, &m.DiskReadBytes, &m.DiskWriteBytes); err != nil {
			return nil, fmt.Errorf("scan agent metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryAgentTokens returns token usage metrics for agents.
func (s *Store) QueryAgentTokens(ctx context.Context, f AgentFilter, tr TimeRange) ([]TokenMetric, error) {
	query := `SELECT time_bucket($1::interval, time) AS bucket,
		agent_name, MAX(model),
		SUM(input_tokens), SUM(output_tokens), SUM(cache_read), SUM(cache_create), SUM(cost_usd)
	FROM token_metrics
	WHERE time >= $2 AND time < $3`

	args := []any{tr.PGInterval(), tr.From, tr.To}

	if len(f.Agent) > 0 {
		ph := make([]string, len(f.Agent))
		for i, a := range f.Agent {
			args = append(args, a)
			ph[i] = fmt.Sprintf("$%d", len(args))
		}
		query += ` AND agent_name IN (` + strings.Join(ph, ",") + `)`
	}

	query += ` GROUP BY bucket, agent_name ORDER BY bucket`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query token metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []TokenMetric
	for rows.Next() {
		var m TokenMetric
		if err := rows.Scan(&m.Time, &m.AgentName, &m.Model, &m.InputTokens, &m.OutputTokens, &m.CacheRead, &m.CacheCreate, &m.CostUSD); err != nil {
			return nil, fmt.Errorf("scan token metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

// QueryAgentCost returns cost metrics for agents.
func (s *Store) QueryAgentCost(ctx context.Context, f AgentFilter, tr TimeRange) ([]TokenMetric, error) {
	return s.QueryAgentTokens(ctx, f, tr) // same data, different view
}

// ChannelFilter specifies which channels to query.
type ChannelFilter struct {
	Channel []string
}

// QueryChannelMessages returns message count metrics.
func (s *Store) QueryChannelMessages(ctx context.Context, f ChannelFilter, tr TimeRange) ([]ChannelMetric, error) {
	return s.queryChannel(ctx, f, tr)
}

// QueryChannelMembers returns member count metrics.
func (s *Store) QueryChannelMembers(ctx context.Context, f ChannelFilter, tr TimeRange) ([]ChannelMetric, error) {
	return s.queryChannel(ctx, f, tr)
}

// QueryChannelReactions returns reaction count metrics.
func (s *Store) QueryChannelReactions(ctx context.Context, f ChannelFilter, tr TimeRange) ([]ChannelMetric, error) {
	return s.queryChannel(ctx, f, tr)
}

func (s *Store) queryChannel(ctx context.Context, f ChannelFilter, tr TimeRange) ([]ChannelMetric, error) {
	query := `SELECT time_bucket($1::interval, time) AS bucket,
		channel_name, SUM(message_count), MAX(member_count), SUM(reaction_count)
	FROM channel_metrics
	WHERE time >= $2 AND time < $3`

	args := []any{tr.PGInterval(), tr.From, tr.To}

	if len(f.Channel) > 0 {
		ph := make([]string, len(f.Channel))
		for i, ch := range f.Channel {
			args = append(args, ch)
			ph[i] = fmt.Sprintf("$%d", len(args))
		}
		query += ` AND channel_name IN (` + strings.Join(ph, ",") + `)`
	}

	query += ` GROUP BY bucket, channel_name ORDER BY bucket`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query channel metrics: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var result []ChannelMetric
	for rows.Next() {
		var m ChannelMetric
		if err := rows.Scan(&m.Time, &m.ChannelName, &m.MessageCount, &m.MemberCount, &m.ReactionCount); err != nil {
			return nil, fmt.Errorf("scan channel metric: %w", err)
		}
		result = append(result, m)
	}
	return result, rows.Err()
}
