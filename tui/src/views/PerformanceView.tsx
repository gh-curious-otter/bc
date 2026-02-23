/**
 * PerformanceView - System observability and monitoring view
 * Issue #1759: Performance/Monitor tab for system observability
 *
 * Shows:
 * - Agent utilization and health metrics
 * - Cost tracking and budget status
 * - TUI performance metrics (render times, command latency)
 * - System uptime and activity summary
 */

import React, { memo, useCallback, useState, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { MetricCard } from '../components/MetricCard';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { ErrorDisplay } from '../components/ErrorDisplay';
import { Footer } from '../components/Footer';
import { PulseText } from '../components/AnimatedText';
import { useStatus, useUtilization, useWorkspaceHealth, useDisableInput } from '../hooks';
import { usePerformanceMetrics, globalPerformanceTracker } from '../hooks/usePerformanceMetrics';
import { useCosts } from '../hooks/useCosts';
import { useResponsiveLayout } from '../hooks/useResponsiveLayout';
import { STATUS_COLORS, HEALTH_COLORS } from '../theme/StatusColors';

// #1594: Using empty interface for future extensibility
// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface PerformanceViewProps {}

/**
 * PerformanceView - System monitoring and observability dashboard
 */
export function PerformanceView(_props: PerformanceViewProps = {}): React.ReactElement {
  const { isDisabled: disableInput } = useDisableInput();
  const { canMultiColumn, isWide } = useResponsiveLayout();

  // Data hooks
  const { data: status, loading: statusLoading, error: statusError, refresh: refreshStatus } = useStatus();
  const { utilization, loading: utilizationLoading } = useUtilization();
  const { healthy, stuckCount, errorCount } = useWorkspaceHealth();
  const { data: costData, loading: costLoading, refresh: refreshCost } = useCosts();
  const { getAllMetrics, getUptime, clearMetrics, toggleDebug, debugEnabled } = usePerformanceMetrics();

  // Auto-refresh TUI metrics from global tracker
  const [tuiMetrics, setTuiMetrics] = useState(globalPerformanceTracker.getAllMetrics());

  useEffect(() => {
    const interval = setInterval(() => {
      setTuiMetrics(globalPerformanceTracker.getAllMetrics());
    }, 1000);
    return () => { clearInterval(interval); };
  }, []);

  // Combined metrics from hook and global tracker
  const allMetrics = [...getAllMetrics(), ...tuiMetrics];

  // Refresh all data
  const refreshAll = useCallback(() => {
    void refreshStatus();
    void refreshCost();
  }, [refreshStatus, refreshCost]);

  // Keyboard navigation
  useInput((input, key) => {
    if (key.ctrl && input === 'p') {
      toggleDebug();
    } else if (input === 'r') {
      refreshAll();
    } else if (input === 'c') {
      clearMetrics();
      globalPerformanceTracker.clear();
      setTuiMetrics([]);
    }
  }, { isActive: !disableInput });

  const isLoading = statusLoading && !status;

  if (statusError) {
    return <ErrorDisplay error={statusError} onRetry={refreshAll} />;
  }

  if (isLoading) {
    return <LoadingIndicator message="Loading performance data..." />;
  }

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Header
        workspaceName={status?.workspace ?? 'unknown'}
        uptime={getUptime()}
        isLoading={statusLoading || costLoading || utilizationLoading}
      />

      {/* Summary cards */}
      <SummaryCards
        utilization={utilization}
        healthy={healthy}
        stuckCount={stuckCount}
        errorCount={errorCount}
        total={status?.total ?? 0}
        active={status?.active ?? 0}
        working={status?.working ?? 0}
      />

      {/* Main content */}
      <Box marginTop={1} flexDirection={canMultiColumn ? 'row' : 'column'}>
        {/* Left column: Health & Agents */}
        <Box flexDirection="column" flexGrow={1} marginRight={canMultiColumn ? 1 : 0}>
          <AgentHealthPanel
            total={status?.total ?? 0}
            active={status?.active ?? 0}
            working={status?.working ?? 0}
            idle={status?.idle ?? 0}
            stuck={status?.stuck ?? 0}
            error={status?.error ?? 0}
            stopped={status?.stopped ?? 0}
          />

          {costData && (
            <CostMetricsPanel
              totalCost={costData.total_cost}
              inputTokens={costData.total_input_tokens}
              outputTokens={costData.total_output_tokens}
            />
          )}
        </Box>

        {/* Right column: Performance metrics */}
        <Box flexDirection="column" width={canMultiColumn ? (isWide ? 45 : 35) : undefined}>
          <PerformanceMetricsPanel
            metrics={allMetrics}
            debugEnabled={debugEnabled}
          />
        </Box>
      </Box>

      {/* Footer */}
      <Footer
        hints={[
          { key: 'r', label: 'refresh' },
          { key: 'c', label: 'clear metrics' },
          { key: 'Ctrl+P', label: debugEnabled ? 'debug off' : 'debug on' },
          { key: 'q', label: 'back' },
        ]}
      />
    </Box>
  );
}

interface HeaderProps {
  workspaceName: string;
  uptime: number;
  isLoading: boolean;
}

const Header = memo(function Header({ workspaceName, uptime, isLoading }: HeaderProps) {
  const uptimeStr = formatUptime(uptime);

  return (
    <Box marginBottom={1}>
      <Text bold color="magenta">Performance Monitor</Text>
      <Text> · </Text>
      <Text>{workspaceName}</Text>
      <Box flexGrow={1} />
      {isLoading ? (
        <PulseText color="yellow">refreshing...</PulseText>
      ) : (
        <Text dimColor>Uptime: {uptimeStr}</Text>
      )}
    </Box>
  );
});

interface SummaryCardsProps {
  utilization: number;
  healthy: boolean;
  stuckCount: number;
  errorCount: number;
  total: number;
  active: number;
  working: number;
}

const SummaryCards = memo(function SummaryCards({
  utilization,
  healthy,
  stuckCount,
  errorCount,
  total,
  active,
  working,
}: SummaryCardsProps) {
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;

  const healthColor = healthy ? HEALTH_COLORS.healthy : errorCount > 0 ? HEALTH_COLORS.critical : HEALTH_COLORS.warning;
  const utilizationColor = utilization >= 80 ? 'green' : utilization >= 50 ? 'yellow' : 'gray';

  if (isNarrow) {
    return (
      <Box marginBottom={1}>
        <Text color={healthColor} bold>{healthy ? 'HEALTHY' : 'DEGRADED'}</Text>
        <Text> · </Text>
        <Text color={utilizationColor}>{utilization}% util</Text>
        <Text> · </Text>
        <Text>{working}/{active} working</Text>
        {stuckCount > 0 && (
          <>
            <Text> · </Text>
            <Text color="yellow">{stuckCount} stuck</Text>
          </>
        )}
        {errorCount > 0 && (
          <>
            <Text> · </Text>
            <Text color="red">{errorCount} error</Text>
          </>
        )}
      </Box>
    );
  }

  return (
    <Box flexWrap="wrap">
      <MetricCard
        value={healthy ? 'OK' : 'WARN'}
        label="Health"
        color={healthColor}
      />
      <MetricCard
        value={`${String(utilization)}%`}
        label="Utilization"
        color={utilizationColor}
      />
      <MetricCard value={total} label="Total" />
      <MetricCard value={active} label="Active" color="green" />
      <MetricCard value={working} label="Working" color="cyan" />
      {stuckCount > 0 && <MetricCard value={stuckCount} label="Stuck" color="yellow" />}
      {errorCount > 0 && <MetricCard value={errorCount} label="Error" color="red" />}
    </Box>
  );
});

interface AgentHealthPanelProps {
  total: number;
  active: number;
  working: number;
  idle: number;
  stuck: number;
  error: number;
  stopped: number;
}

const AgentHealthPanel = memo(function AgentHealthPanel({
  total,
  active,
  working,
  idle,
  stuck,
  error,
  stopped,
}: AgentHealthPanelProps) {
  const utilizationPercent = active > 0 ? Math.round((working / active) * 100) : 0;
  const healthyCount = working + idle;
  const unhealthyCount = stuck + error;
  const healthPercent = total > 0 ? Math.round((healthyCount / total) * 100) : 100;

  return (
    <Panel title="Agent Health" borderColor="cyan">
      <Box flexDirection="column">
        {/* Utilization bar */}
        <Box marginBottom={1}>
          <Text dimColor>Utilization: </Text>
          <ProgressBar percent={utilizationPercent} width={15} color="cyan" />
          <Text> {utilizationPercent}%</Text>
        </Box>

        {/* Health bar */}
        <Box marginBottom={1}>
          <Text dimColor>Health:      </Text>
          <ProgressBar
            percent={healthPercent}
            width={15}
            color={healthPercent >= 80 ? 'green' : healthPercent >= 50 ? 'yellow' : 'red'}
          />
          <Text> {healthPercent}%</Text>
        </Box>

        {/* State breakdown */}
        <Box flexDirection="column" marginTop={1}>
          <Box>
            <PulseText color={STATUS_COLORS.working} enabled={working > 0} interval={1500}>●</PulseText>
            <Text> Working: </Text>
            <Text bold>{working}</Text>
          </Box>
          <Box>
            <Text color={STATUS_COLORS.idle}>●</Text>
            <Text> Idle: </Text>
            <Text>{idle}</Text>
          </Box>
          {stuck > 0 && (
            <Box>
              <Text color={STATUS_COLORS.warning}>●</Text>
              <Text> Stuck: </Text>
              <Text color="yellow">{stuck}</Text>
            </Box>
          )}
          {error > 0 && (
            <Box>
              <Text color={STATUS_COLORS.error}>●</Text>
              <Text> Error: </Text>
              <Text color="red">{error}</Text>
            </Box>
          )}
          {stopped > 0 && (
            <Box>
              <Text color="gray">●</Text>
              <Text> Stopped: </Text>
              <Text dimColor>{stopped}</Text>
            </Box>
          )}
        </Box>

        {unhealthyCount > 0 && (
          <Box marginTop={1}>
            <Text color="yellow">
              ⚠ {unhealthyCount} agent{unhealthyCount > 1 ? 's' : ''} need attention
            </Text>
          </Box>
        )}
      </Box>
    </Panel>
  );
});

interface CostMetricsPanelProps {
  totalCost: number;
  inputTokens: number;
  outputTokens: number;
}

const CostMetricsPanel = memo(function CostMetricsPanel({
  totalCost,
  inputTokens,
  outputTokens,
}: CostMetricsPanelProps) {
  const totalTokens = inputTokens + outputTokens;

  return (
    <Panel title="Cost Summary" borderColor="yellow">
      <Box flexDirection="column">
        <Box>
          <Text dimColor>Total Cost: </Text>
          <Text bold color="yellow">${totalCost.toFixed(4)}</Text>
        </Box>
        <Box>
          <Text dimColor>Total Tokens: </Text>
          <Text>{formatNumber(totalTokens)}</Text>
        </Box>
        <Box>
          <Text dimColor>Input: </Text>
          <Text>{formatNumber(inputTokens)}</Text>
          <Text dimColor> · Output: </Text>
          <Text>{formatNumber(outputTokens)}</Text>
        </Box>
      </Box>
    </Panel>
  );
});

interface PerformanceMetric {
  name: string;
  value: number;
  average: number;
  min: number;
  max: number;
  count: number;
}

interface PerformanceMetricsPanelProps {
  metrics: PerformanceMetric[];
  debugEnabled: boolean;
}

const PerformanceMetricsPanel = memo(function PerformanceMetricsPanel({
  metrics,
  debugEnabled,
}: PerformanceMetricsPanelProps) {
  // Sort by name and take top metrics
  const sortedMetrics = [...metrics]
    .sort((a, b) => a.name.localeCompare(b.name))
    .slice(0, 10);

  return (
    <Panel title="TUI Performance" borderColor={debugEnabled ? 'green' : 'gray'}>
      <Box flexDirection="column">
        {sortedMetrics.length === 0 ? (
          <Text dimColor>No metrics recorded yet</Text>
        ) : (
          <>
            {/* Header */}
            <Box marginBottom={1}>
              <Box width={18}>
                <Text bold dimColor>METRIC</Text>
              </Box>
              <Box width={8}>
                <Text bold dimColor>AVG</Text>
              </Box>
              <Box width={8}>
                <Text bold dimColor>MAX</Text>
              </Box>
              <Box width={6}>
                <Text bold dimColor>N</Text>
              </Box>
            </Box>

            {/* Metrics rows */}
            {sortedMetrics.map((metric) => (
              <MetricRow key={metric.name} metric={metric} />
            ))}
          </>
        )}

        {debugEnabled && (
          <Box marginTop={1}>
            <Text color="green">Debug mode enabled</Text>
          </Box>
        )}
      </Box>
    </Panel>
  );
});

interface MetricRowProps {
  metric: PerformanceMetric;
}

const MetricRow = memo(function MetricRow({ metric }: MetricRowProps) {
  // Color based on average latency
  const avgColor = metric.average < 16 ? 'green' : metric.average < 50 ? 'yellow' : 'red';
  const maxColor = metric.max < 50 ? 'green' : metric.max < 100 ? 'yellow' : 'red';

  // Truncate long metric names
  const displayName = metric.name.length > 16
    ? metric.name.slice(0, 15) + '…'
    : metric.name;

  return (
    <Box>
      <Box width={18}>
        <Text>{displayName}</Text>
      </Box>
      <Box width={8}>
        <Text color={avgColor}>{metric.average.toFixed(1)}ms</Text>
      </Box>
      <Box width={8}>
        <Text color={maxColor}>{metric.max.toFixed(0)}ms</Text>
      </Box>
      <Box width={6}>
        <Text dimColor>{metric.count}</Text>
      </Box>
    </Box>
  );
});

interface ProgressBarProps {
  percent: number;
  width: number;
  color: string;
}

const ProgressBar = memo(function ProgressBar({ percent, width, color }: ProgressBarProps) {
  const filled = Math.round((percent / 100) * width);
  const empty = width - filled;

  return (
    <>
      <Text color={color}>{'█'.repeat(filled)}</Text>
      <Text dimColor>{'░'.repeat(empty)}</Text>
    </>
  );
});

/**
 * Format uptime in seconds to human-readable string
 */
function formatUptime(seconds: number): string {
  if (seconds < 60) {
    return `${String(Math.floor(seconds))}s`;
  }
  const mins = Math.floor(seconds / 60);
  if (mins < 60) {
    return `${String(mins)}m`;
  }
  const hours = Math.floor(mins / 60);
  const remainingMins = mins % 60;
  return `${String(hours)}h ${String(remainingMins)}m`;
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

export default PerformanceView;
