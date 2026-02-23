/**
 * PerformanceView - System observability and performance monitoring
 * Issue #1759: Performance/Monitor tab for system health
 *
 * Displays:
 * - Agent health status (healthy, degraded, stuck, error)
 * - System stats (uptime, state distribution)
 * - Cost metrics
 */

import React, { useMemo, useCallback } from 'react';
import { Box, Text } from 'ink';
import { Panel } from '../components/Panel';
import { HeaderBar } from '../components/HeaderBar';
import { Footer } from '../components/Footer';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { useSystemHealth, useListNavigation, useResponsiveLayout } from '../hooks';
import type { AgentHealth } from '../types';

/**
 * Get status color based on health status
 */
function getStatusColor(status: string): string {
  switch (status) {
    case 'healthy':
      return 'green';
    case 'degraded':
      return 'yellow';
    case 'stuck':
      return 'red';
    case 'error':
      return 'red';
    default:
      return 'gray';
  }
}

/**
 * Get status icon based on health status
 */
function getStatusIcon(status: string): string {
  switch (status) {
    case 'healthy':
      return '●';
    case 'degraded':
      return '◐';
    case 'stuck':
      return '■';
    case 'error':
      return '✗';
    default:
      return '○';
  }
}

/**
 * Format uptime in human-readable form
 */
function formatUptime(nanoseconds: number): string {
  const seconds = Math.floor(nanoseconds / 1e9);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return `${String(days)}d ${String(hours % 24)}h`;
  } else if (hours > 0) {
    return `${String(hours)}h ${String(minutes % 60)}m`;
  } else if (minutes > 0) {
    return `${String(minutes)}m`;
  } else {
    return `${String(seconds)}s`;
  }
}

export function PerformanceView(): React.ReactElement {
  const { data, summary, loading, error, refresh, lastRefresh } = useSystemHealth();
  const { isCompact, isXS } = useResponsiveLayout();

  // Sort health by status (errors first, then stuck, then degraded, then healthy)
  const sortedHealth = useMemo((): AgentHealth[] => {
    const statusOrder: Record<string, number> = { error: 0, stuck: 1, degraded: 2, healthy: 3 };
    const healthList: AgentHealth[] = data.health;
    return [...healthList].sort((a: AgentHealth, b: AgentHealth) => {
      const orderA = statusOrder[a.status] ?? 4;
      const orderB = statusOrder[b.status] ?? 4;
      return orderA - orderB;
    });
  }, [data.health]);

  // Handle detail view for an agent
  const handleSelect = useCallback((_agent: AgentHealth) => {
    // Future: Show detailed agent health view
  }, []);

  // Custom keys for performance view
  const customKeys = useMemo(
    () => ({
      r: () => { void refresh(); },
    }),
    [refresh]
  );

  // #1759: Use useListNavigation for vim-style navigation
  const { selectedIndex } = useListNavigation({
    items: sortedHealth,
    onSelect: handleSelect,
    customKeys,
  });

  if (loading && sortedHealth.length === 0) {
    return <LoadingIndicator message="Loading performance data..." />;
  }

  if (error && sortedHealth.length === 0) {
    return <ErrorDisplay error={error} onRetry={() => { void refresh(); }} />;
  }

  // Compact layout for narrow terminals
  const isNarrow = isCompact || isXS;

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <HeaderBar
        title="Performance"
        count={summary.totalAgents}
        loading={loading}
        color="magenta"
      />

      {/* Summary Panel */}
      <Panel title="System Health">
        <Box flexDirection={isNarrow ? 'column' : 'row'} gap={isNarrow ? 0 : 2}>
          {/* Agent Health Summary */}
          <Box flexDirection="column">
            <Box>
              <Text color="green">● Healthy: </Text>
              <Text bold>{summary.healthyAgents}</Text>
            </Box>
            <Box>
              <Text color="yellow">◐ Degraded: </Text>
              <Text bold>{summary.degradedAgents}</Text>
            </Box>
            <Box>
              <Text color="red">■ Stuck: </Text>
              <Text bold>{summary.stuckAgents}</Text>
            </Box>
            <Box>
              <Text color="red">✗ Error: </Text>
              <Text bold>{summary.errorAgents}</Text>
            </Box>
          </Box>

          {/* Cost Summary */}
          {!isNarrow && (
            <Box flexDirection="column" marginLeft={4}>
              <Box>
                <Text dimColor>Total Cost: </Text>
                <Text bold color="yellow">${summary.totalCost.toFixed(4)}</Text>
              </Box>
              {lastRefresh && (
                <Box>
                  <Text dimColor>Last Update: </Text>
                  <Text>{lastRefresh.toLocaleTimeString()}</Text>
                </Box>
              )}
            </Box>
          )}
        </Box>
      </Panel>

      {/* Agent Health List */}
      <Panel title="Agent Health">
        {sortedHealth.length === 0 ? (
          <Text dimColor>No agents found</Text>
        ) : (
          <Box flexDirection="column">
            {/* Header row */}
            <Box marginBottom={1}>
              <Box width={3}><Text dimColor> </Text></Box>
              <Box width={15}><Text bold dimColor>AGENT</Text></Box>
              <Box width={10}><Text bold dimColor>STATUS</Text></Box>
              <Box width={12}><Text bold dimColor>ROLE</Text></Box>
              {!isNarrow && (
                <>
                  <Box width={6}><Text bold dimColor>TMUX</Text></Box>
                  <Box flexGrow={1}><Text bold dimColor>MESSAGE</Text></Box>
                </>
              )}
            </Box>

            {/* Agent rows */}
            {sortedHealth.map((agent, idx) => {
              const isSelected = idx === selectedIndex;
              const statusColor = getStatusColor(agent.status);
              const statusIcon = getStatusIcon(agent.status);

              return (
                <Box key={agent.name}>
                  <Box width={3}>
                    <Text color={isSelected ? 'cyan' : undefined}>
                      {isSelected ? '▸ ' : '  '}
                    </Text>
                  </Box>
                  <Box width={15}>
                    <Text color={isSelected ? 'cyan' : undefined} bold={isSelected}>
                      {agent.name.length > 13 ? agent.name.slice(0, 12) + '…' : agent.name}
                    </Text>
                  </Box>
                  <Box width={10}>
                    <Text color={statusColor}>
                      {statusIcon} {agent.status}
                    </Text>
                  </Box>
                  <Box width={12}>
                    <Text dimColor>{agent.role}</Text>
                  </Box>
                  {!isNarrow && (
                    <>
                      <Box width={6}>
                        <Text color={agent.tmux_alive ? 'green' : 'red'}>
                          {agent.tmux_alive ? '✓' : '✗'}
                        </Text>
                      </Box>
                      <Box flexGrow={1}>
                        <Text dimColor wrap="truncate">
                          {agent.error_message ?? agent.stuck_details ?? '-'}
                        </Text>
                      </Box>
                    </>
                  )}
                </Box>
              );
            })}
          </Box>
        )}
      </Panel>

      {/* Stats Panel - Uptime info */}
      {data.stats && (
        <Panel title="Agent Stats">
          <Box flexDirection="row" flexWrap="wrap" gap={2}>
            {data.stats.agents.agent_stats.slice(0, isNarrow ? 4 : 8).map((stat) => (
              <Box key={stat.name}>
                <Text color="cyan">{stat.name.slice(0, 8)}</Text>
                <Text dimColor>: </Text>
                <Text>{formatUptime(stat.uptime)}</Text>
              </Box>
            ))}
            {data.stats.agents.agent_stats.length > (isNarrow ? 4 : 8) && (
              <Text dimColor>+{data.stats.agents.agent_stats.length - (isNarrow ? 4 : 8)} more</Text>
            )}
          </Box>
        </Panel>
      )}

      {/* Footer */}
      <Footer
        hints={[
          { key: 'j/k', label: 'navigate' },
          { key: 'r', label: 'refresh' },
          { key: 'q/ESC', label: 'back' },
        ]}
      />
    </Box>
  );
}

export default PerformanceView;
