/**
 * PerformanceDebugPanel - Debug overlay for performance metrics
 * Issue #965: Display performance metrics when BC_TUI_DEBUG=1
 */

import React from 'react';
import { Box, Text } from 'ink';
import { usePerformanceOptional } from '../hooks';

interface PerformanceDebugPanelProps {
  /** Maximum number of metrics to display */
  maxMetrics?: number;
  /** Show compact view (single line) */
  compact?: boolean;
  /** Force show panel regardless of debugEnabled state (for Ctrl+P toggle) */
  forceShow?: boolean;
}

/**
 * Debug panel showing performance metrics
 * Only renders when debug mode is enabled (BC_TUI_DEBUG=1)
 */
export function PerformanceDebugPanel({
  maxMetrics = 5,
  compact = false,
  forceShow = false,
}: PerformanceDebugPanelProps): React.ReactElement | null {
  const perf = usePerformanceOptional();

  // Don't render if no performance context available
  if (!perf) {
    return null;
  }

  // Don't render if not in debug mode (unless forceShow from Ctrl+P toggle)
  if (!forceShow && !perf.debugEnabled) {
    return null;
  }

  const metrics = perf.getAllMetrics().slice(0, maxMetrics);
  const uptime = perf.getUptime();

  if (compact) {
    // Single line compact view
    const summaryParts = metrics.map((m) => `${m.name}:${m.average.toFixed(0)}ms`);
    return (
      <Box>
        <Text dimColor>
          [PERF] {summaryParts.join(' | ')} | uptime:{Math.floor(uptime)}s | samples:{perf.totalMeasurements}
        </Text>
      </Box>
    );
  }

  // Full debug panel
  return (
    <Box
      flexDirection="column"
      borderStyle="single"
      borderColor="yellow"
      paddingX={1}
      marginTop={1}
    >
      <Box justifyContent="space-between">
        <Text bold color="yellow">PERFORMANCE DEBUG</Text>
        <Text dimColor>
          uptime: {Math.floor(uptime)}s | samples: {perf.totalMeasurements}
        </Text>
      </Box>

      <Box marginTop={0}>
        <Text dimColor>{'─'.repeat(50)}</Text>
      </Box>

      {/* Header */}
      <Box>
        <Text bold>
          {'METRIC'.padEnd(25)}
          {'AVG'.padStart(8)}
          {'MIN'.padStart(8)}
          {'MAX'.padStart(8)}
          {'CNT'.padStart(6)}
        </Text>
      </Box>

      {/* Metrics rows */}
      {metrics.map((metric) => (
        <MetricRow key={metric.name} metric={metric} />
      ))}

      {metrics.length === 0 && (
        <Text dimColor>No metrics recorded yet...</Text>
      )}

      <Box marginTop={0}>
        <Text dimColor>Press d to toggle debug | c to clear metrics</Text>
      </Box>
    </Box>
  );
}

interface MetricRowProps {
  metric: {
    name: string;
    average: number;
    min: number;
    max: number;
    count: number;
  };
}

function MetricRow({ metric }: MetricRowProps): React.ReactElement {
  // Color code based on average time
  const getColor = (avgMs: number): string => {
    if (avgMs < 50) return 'green';
    if (avgMs < 200) return 'yellow';
    return 'red';
  };

  const color = getColor(metric.average);

  return (
    <Box>
      <Text>
        {metric.name.slice(0, 24).padEnd(25)}
      </Text>
      <Text color={color}>
        {`${metric.average.toFixed(1)}ms`.padStart(8)}
      </Text>
      <Text dimColor>
        {`${metric.min.toFixed(1)}ms`.padStart(8)}
        {`${metric.max.toFixed(1)}ms`.padStart(8)}
        {String(metric.count).padStart(6)}
      </Text>
    </Box>
  );
}

/**
 * Compact footer-style metrics display
 * Shows key metrics in a single line
 */
export function PerformanceFooter(): React.ReactElement | null {
  const perf = usePerformanceOptional();

  if (!perf?.debugEnabled) {
    return null;
  }

  const metrics = perf.getAllMetrics();
  const pollMetrics = metrics.filter((m) => m.name.startsWith('poll:'));
  const cmdMetrics = metrics.filter((m) => m.name.startsWith('cmd:'));

  // Calculate averages
  const avgPoll = pollMetrics.length > 0
    ? pollMetrics.reduce((sum, m) => sum + m.average, 0) / pollMetrics.length
    : 0;
  const avgCmd = cmdMetrics.length > 0
    ? cmdMetrics.reduce((sum, m) => sum + m.average, 0) / cmdMetrics.length
    : 0;

  return (
    <Box marginTop={0}>
      <Text dimColor>
        [PERF] poll:{avgPoll.toFixed(0)}ms cmd:{avgCmd.toFixed(0)}ms samples:{perf.totalMeasurements}
      </Text>
    </Box>
  );
}

export default PerformanceDebugPanel;
