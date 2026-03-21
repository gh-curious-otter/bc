/**
 * Metrics tab for AgentDetailView
 * Shows cost breakdown, activity timeline, and session info
 */

import React from 'react';
import { Box, Text } from 'ink';
import type { Agent } from '../../types';
import type { AgentCostDetails, AgentActivity } from '../../hooks/useAgentDetails';
import { MetricCard } from '../../components/MetricCard';
import { DetailRow, formatDate, formatTime, formatNumber, truncateMessage, formatUptime } from './types';

interface AgentMetricsTabProps {
  agent: Agent;
  cost: AgentCostDetails | null;
  activity: AgentActivity[];
}

export function AgentMetricsTab({ agent, cost, activity }: AgentMetricsTabProps): React.ReactElement {
  return (
    <Box flexDirection="column" paddingX={1}>
      {/* Cost Metrics */}
      <Box marginBottom={1}>
        <Text bold color="white">Cost Breakdown</Text>
      </Box>
      <Box flexDirection="row" marginBottom={1}>
        <MetricCard
          label="Total Cost"
          value={cost ? `$${cost.totalCost.toFixed(4)}` : '$0.00'}
          color="green"
        />
        <MetricCard
          label="Input Tokens"
          value={cost ? formatNumber(cost.inputTokens) : '0'}
          color="cyan"
        />
        <MetricCard
          label="Output Tokens"
          value={cost ? formatNumber(cost.outputTokens) : '0'}
          color="cyan"
        />
      </Box>

      {/* Activity Timeline */}
      <Box marginY={1}>
        <Text bold color="white">Recent Activity</Text>
      </Box>
      <Box flexDirection="column" paddingX={1} borderStyle="single" borderColor="gray" minHeight={6}>
        {activity.length === 0 ? (
          <Text dimColor>No recent activity</Text>
        ) : (
          activity.slice(0, 8).map((event, idx) => (
            <Box key={idx}>
              <Text dimColor wrap="truncate">{formatTime(event.timestamp)}</Text>
              <Text color="cyan" wrap="truncate"> [{event.type.split('.').pop()}] </Text>
              <Text wrap="truncate">{truncateMessage(event.message, 40)}</Text>
            </Box>
          ))
        )}
      </Box>

      {/* Performance Summary */}
      <Box marginY={1}>
        <Text bold color="white">Session Info</Text>
      </Box>
      <DetailRow label="Uptime" value={formatUptime(agent.started_at)} />
      <DetailRow label="Last Update" value={formatDate(agent.updated_at)} />
      <DetailRow label="Events" value={String(activity.length)} />
    </Box>
  );
}
