import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel.js';
import { MetricCard } from '../components/MetricCard.js';
import { DataTable } from '../components/DataTable.js';
import { Footer } from '../components/Footer.js';
import { LoadingIndicator } from '../components/LoadingIndicator.js';
import { ErrorDisplay } from '../components/ErrorDisplay.js';
import { useCosts } from '../hooks';

interface CostDashboardProps {
  onBack?: () => void;
}

/**
 * CostDashboard - Comprehensive cost overview view
 * Issue #553 - Cost dashboard view
 */
export function CostDashboard({ onBack }: CostDashboardProps) {
  const { data: costs, loading, error, refresh } = useCosts();

  // Keyboard navigation
  useInput((input, key) => {
    if (input === 'q' || key.escape) {
      onBack?.();
    }
    if (input === 'r') {
      refresh();
    }
  });

  if (error) {
    return <ErrorDisplay error={error} onRetry={refresh} />;
  }

  if (loading && !costs) {
    return <LoadingIndicator message="Loading cost data..." />;
  }

  const totalCost = costs?.total_cost ?? 0;
  const inputTokens = costs?.total_input_tokens ?? 0;
  const outputTokens = costs?.total_output_tokens ?? 0;
  const totalTokens = inputTokens + outputTokens;

  // Convert agent breakdown to table data
  const agentData = Object.entries(costs?.by_agent ?? {})
    .map(([agent, cost]) => ({
      agent,
      cost,
      pct: totalCost > 0 ? (cost / totalCost) * 100 : 0,
    }))
    .sort((a, b) => b.cost - a.cost);

  // Convert model breakdown to table data
  const modelData = Object.entries(costs?.by_model ?? {})
    .map(([model, cost]) => ({
      model,
      cost,
      pct: totalCost > 0 ? (cost / totalCost) * 100 : 0,
    }))
    .sort((a, b) => b.cost - a.cost);

  // Convert team breakdown to table data
  const teamData = Object.entries(costs?.by_team ?? {})
    .map(([team, cost]) => ({
      team,
      cost,
      pct: totalCost > 0 ? (cost / totalCost) * 100 : 0,
    }))
    .sort((a, b) => b.cost - a.cost);

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="yellow">
          Cost Dashboard
        </Text>
        {loading && <Text color="cyan"> (refreshing...)</Text>}
      </Box>

      {/* Summary Metrics */}
      <Box marginBottom={1}>
        <MetricCard
          value={totalCost.toFixed(4)}
          label="Total Cost"
          prefix="$"
          color="yellow"
        />
        <MetricCard
          value={formatNumber(inputTokens)}
          label="Input Tokens"
          color="cyan"
        />
        <MetricCard
          value={formatNumber(outputTokens)}
          label="Output Tokens"
          color="cyan"
        />
        <MetricCard value={formatNumber(totalTokens)} label="Total Tokens" />
      </Box>

      {/* Agent Breakdown */}
      <Panel title="By Agent">
        {agentData.length === 0 ? (
          <Text dimColor>No agent costs recorded</Text>
        ) : (
          <DataTable
            columns={[
              { key: 'agent', header: 'AGENT', width: 20 },
              {
                key: 'cost',
                header: 'COST',
                width: 12,
                render: (value) => (
                  <Text color="yellow">${(value as number).toFixed(4)}</Text>
                ),
              },
              {
                key: 'pct',
                header: '% SHARE',
                width: 10,
                render: (value) => <Text>{(value as number).toFixed(1)}%</Text>,
              },
            ]}
            data={agentData.slice(0, 8)}
          />
        )}
        {agentData.length > 8 && (
          <Text dimColor>... and {agentData.length - 8} more agents</Text>
        )}
      </Panel>

      {/* Model Breakdown */}
      <Panel title="By Model">
        {modelData.length === 0 ? (
          <Text dimColor>No model costs recorded</Text>
        ) : (
          <DataTable
            columns={[
              { key: 'model', header: 'MODEL', width: 25 },
              {
                key: 'cost',
                header: 'COST',
                width: 12,
                render: (value) => (
                  <Text color="magenta">${(value as number).toFixed(4)}</Text>
                ),
              },
              {
                key: 'pct',
                header: '% SHARE',
                width: 10,
                render: (value) => <Text>{(value as number).toFixed(1)}%</Text>,
              },
            ]}
            data={modelData}
          />
        )}
      </Panel>

      {/* Team Breakdown (if data exists) */}
      {teamData.length > 0 && (
        <Panel title="By Team">
          <DataTable
            columns={[
              { key: 'team', header: 'TEAM', width: 20 },
              {
                key: 'cost',
                header: 'COST',
                width: 12,
                render: (value) => (
                  <Text color="blue">${(value as number).toFixed(4)}</Text>
                ),
              },
              {
                key: 'pct',
                header: '% SHARE',
                width: 10,
                render: (value) => <Text>{(value as number).toFixed(1)}%</Text>,
              },
            ]}
            data={teamData}
          />
        </Panel>
      )}

      {/* Footer with keyboard hints */}
      <Footer
        hints={[
          { key: 'r', label: 'refresh' },
          { key: 'q', label: 'back' },
        ]}
      />
    </Box>
  );
}

/**
 * Format large numbers with K/M suffixes
 */
function formatNumber(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`;
  }
  return n.toString();
}

export default CostDashboard;
