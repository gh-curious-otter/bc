import { memo, useState } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { Panel } from '../components/Panel.js';
import { MetricCard } from '../components/MetricCard.js';
import { Footer } from '../components/Footer.js';
import { LoadingIndicator } from '../components/LoadingIndicator.js';
import { ErrorDisplay } from '../components/ErrorDisplay.js';
import { ActivityFeed } from '../components/ActivityFeed.js';
import { PerformanceDebugPanel } from '../components/PerformanceDebugPanel.js';
import { PulseText } from '../components/AnimatedText.js';
import { useDashboard } from '../hooks/useDashboard.js';
import { useNavigation } from '../navigation/NavigationContext.js';
import { useResponsiveLayout } from '../hooks/useResponsiveLayout.js';
import { STATUS_COLORS, HEALTH_COLORS } from '../theme/StatusColors.js';

interface DashboardProps {
  /** @deprecated Use navigation context instead */
  onNavigate?: (view: string) => void;
}

/**
 * Dashboard view - main overview of bc workspace
 * Issues #543 (layout), #544 (stats components), #931 (shortcuts fix)
 */
export function Dashboard({ onNavigate: _onNavigate }: DashboardProps) {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;
  const { navigate } = useNavigation();
  const { canMultiColumn, isMedium, isWide } = useResponsiveLayout();
  const [showDebugPanel, setShowDebugPanel] = useState(false);

  const {
    summary,
    agents,
    // channels removed from dashboard - use Channels tab [3]
    agentStats,
    isLoading,
    error,
    refresh,
    lastRefresh,
  } = useDashboard();

  // Keyboard navigation - Dashboard-specific shortcuts
  // Global shortcuts (1-8, Tab, ESC, q) are handled by useKeyboardNavigation
  useInput((input, key) => {
    // Quick navigation shortcuts
    if (input === 'a') {
      navigate('agents');
    }
    if (input === 'c') {
      navigate('channels');
    }
    if (input === '$') {
      navigate('costs');
    }
    // Refresh is Dashboard-specific (global Ctrl+R handled elsewhere)
    if (input === 'r') {
      void refresh();
    }
    // Performance overlay toggle - Ctrl+P (Phase 3 integration #1032)
    if (key.ctrl && input === 'p') {
      setShowDebugPanel(!showDebugPanel);
    }
    // Note: q and ESC are handled by global useKeyboardNavigation
  });

  if (error) {
    return <ErrorDisplay error={error.message} onRetry={() => { void refresh(); }} />;
  }

  if (isLoading && !agents.data) {
    return <LoadingIndicator message="Loading workspace data..." />;
  }

  return (
    <Box flexDirection="column" padding={1} width={terminalWidth}>
      {/* Header with activity indicator */}
      <Header
        workspaceName={summary.workspaceName}
        isLoading={isLoading}
        lastRefresh={lastRefresh}
      />

      {/* Summary Cards - Agent counts */}
      <SummaryCards
        total={summary.total}
        active={summary.active}
        working={summary.working}
        idle={summary.idle}
        stuck={summary.stuck}
        errorCount={summary.error}
      />

      {/* Metrics panels - Optimized for all terminal sizes (Issue #1041) */}
      {/* On narrow terminals: 2x2 grid at top, on wide: side-by-side with activity feed */}
      {!canMultiColumn && (
        <Box marginTop={1} flexDirection="row" width="100%">
          {/* Left column: System Health + Cost */}
          <Box flexDirection="column" flexGrow={1} marginRight={1}>
            <SystemHealthPanel
              working={summary.working}
              idle={summary.idle}
              stuck={summary.stuck}
              errorCount={summary.error}
              total={summary.total}
            />
            <CostPanel
              totalCostUSD={summary.totalCostUSD}
              inputTokens={summary.inputTokens}
              outputTokens={summary.outputTokens}
            />
          </Box>

          {/* Right column: Agent Distribution */}
          <Box flexDirection="column" width={Math.max(20, Math.floor(terminalWidth * 0.35))}>
            <AgentStatsPanel stats={agentStats} />
          </Box>
        </Box>
      )}

      {/* Main Content - Uses responsive layout for flexible column arrangement */}
      <Box marginTop={1} flexDirection={canMultiColumn ? 'row' : 'column'}>
        {/* Activity Feed - primary focus */}
        <Box flexDirection="column" flexGrow={1} marginRight={canMultiColumn ? 1 : 0}>
          <ActivityFeed maxEntries={isMedium || isWide ? 15 : 8} compact={!isWide} showFilterHints={canMultiColumn} />
        </Box>

        {/* Stats & Health panels - side column when space allows (medium+) */}
        {canMultiColumn && (
          <Box flexDirection="column" width={Math.min(32, Math.max(26, Math.floor((terminalWidth - 4) * 0.28)))}>
            {/* System Health - Agent states */}
            <SystemHealthPanel
              working={summary.working}
              idle={summary.idle}
              stuck={summary.stuck}
              errorCount={summary.error}
              total={summary.total}
            />

            {/* Cost Panel with budget progress */}
            <CostPanel
              totalCostUSD={summary.totalCostUSD}
              inputTokens={summary.inputTokens}
              outputTokens={summary.outputTokens}
            />

            {/* Agent Distribution by Role */}
            <AgentStatsPanel stats={agentStats} />
          </Box>
        )}
      </Box>

      {/* Performance Debug Panel - toggled with Ctrl+P or F12 (Phase 3) */}
      {showDebugPanel && <PerformanceDebugPanel compact={!isWide} forceShow />}

      {/* Footer with keyboard hints */}
      <Footer
        hints={[
          { key: 'a', label: 'agents' },
          { key: 'c', label: 'channels' },
          { key: '$', label: 'costs' },
          { key: 'r', label: 'refresh' },
          ...(showDebugPanel ? [{ key: 'Ctrl+P', label: 'hide perf' }] : [{ key: 'Ctrl+P', label: 'perf' }]),
          { key: 'q', label: 'quit' },
        ]}
      />
    </Box>
  );
}

interface HeaderProps {
  workspaceName: string;
  isLoading: boolean;
  lastRefresh: Date | null;
}

/**
 * Memoized header - only re-renders when props change
 */
const Header = memo(function Header({ workspaceName, isLoading, lastRefresh }: HeaderProps) {
  const refreshText = lastRefresh
    ? `Updated ${formatRelativeTime(lastRefresh)}`
    : '';

  return (
    <Box marginBottom={1}>
      <Text bold color="blue">
        bc
      </Text>
      <Text> · </Text>
      <Text>{workspaceName}</Text>
      <Box flexGrow={1} />
      {isLoading ? (
        <Text color="yellow">↻ refreshing...</Text>
      ) : (
        <Text dimColor>{refreshText}</Text>
      )}
    </Box>
  );
});

interface SummaryCardsProps {
  total: number;
  active: number;
  working: number;
  idle: number;
  stuck: number;
  errorCount: number;
}

/**
 * Memoized summary cards - only re-renders when counts change
 * Wraps to multiple lines on narrow terminals
 */
const SummaryCards = memo(function SummaryCards({
  total,
  active,
  working,
  idle,
  stuck,
  errorCount,
}: SummaryCardsProps) {
  return (
    <Box flexWrap="wrap">
      <MetricCard value={total} label="Total" />
      <MetricCard value={active} label="Active" color="green" />
      <MetricCard value={working} label="Working" color="cyan" />
      <MetricCard value={idle} label="Idle" color="gray" />
      {stuck > 0 && <MetricCard value={stuck} label="Stuck" color="yellow" />}
      {errorCount > 0 && (
        <MetricCard value={errorCount} label="Error" color="red" />
      )}
    </Box>
  );
});

interface SystemHealthPanelProps {
  working: number;
  idle: number;
  stuck: number;
  errorCount: number;
  total: number;
}

/**
 * System Health panel - shows agent state distribution
 */
const SystemHealthPanel = memo(function SystemHealthPanel({
  working,
  idle,
  stuck,
  errorCount,
  total,
}: SystemHealthPanelProps) {
  const healthyCount = working + idle;
  const unhealthyCount = stuck + errorCount;
  const healthPercent = total > 0 ? Math.round((healthyCount / total) * 100) : 100;
  const healthColor = healthPercent >= 80 ? HEALTH_COLORS.healthy : healthPercent >= 50 ? HEALTH_COLORS.warning : HEALTH_COLORS.critical;

  // #1181 fix: Use Box for inline layout instead of nested Text with wrap="truncate"
  // Nested Text inside Text with wrap="truncate" causes garbling (e.g., "50% healthyth")
  return (
    <Panel title="System Health">
      <Box flexDirection="column">
        {/* Health percentage - Box layout to prevent truncation garbling */}
        <Box>
          <Text color={healthColor} bold>{healthPercent}%</Text>
          <Text dimColor> healthy</Text>
        </Box>
        <Box marginTop={1} flexDirection="column">
          {/* Working agents with pulse animation (Phase 3) - consistent colors */}
          <Box>
            <PulseText color={STATUS_COLORS.working} enabled={working > 0} interval={1500}>●</PulseText>
            <Text> Working: {working}</Text>
          </Box>
          <Box>
            <Text color={STATUS_COLORS.idle}>●</Text>
            <Text> Idle: {idle}</Text>
          </Box>
          {stuck > 0 && (
            <Box>
              <Text color={STATUS_COLORS.warning}>●</Text>
              <Text> Stuck: {stuck}</Text>
            </Box>
          )}
          {errorCount > 0 && (
            <Box>
              <Text color={STATUS_COLORS.error}>●</Text>
              <Text> Error: {errorCount}</Text>
            </Box>
          )}
        </Box>
        {unhealthyCount > 0 && (
          <Text color="yellow" dimColor>
            ⚠ {unhealthyCount} agent{unhealthyCount > 1 ? 's' : ''} need attention
          </Text>
        )}
      </Box>
    </Panel>
  );
});

interface CostPanelProps {
  totalCostUSD: number;
  inputTokens: number;
  outputTokens: number;
  budgetUSD?: number;
}

/**
 * Cost panel with budget progress bar (responsive width)
 */
const CostPanel = memo(function CostPanel({
  totalCostUSD,
  inputTokens,
  outputTokens,
  budgetUSD = 10.0,
}: CostPanelProps) {
  const totalTokens = inputTokens + outputTokens;
  const budgetPercent = Math.min(100, Math.round((totalCostUSD / budgetUSD) * 100));
  // Responsive bar width: smaller on narrow terminals
  const barWidth = 15;
  const filledWidth = Math.round((budgetPercent / 100) * barWidth);
  const emptyWidth = barWidth - filledWidth;

  const barColor = budgetPercent >= 90 ? 'red' : budgetPercent >= 75 ? 'yellow' : 'green';

  return (
    <Panel title="Cost">
      <Box flexDirection="column">
        <Box>
          <Text bold color="yellow">${totalCostUSD.toFixed(2)}</Text>
          <Text dimColor> / ${budgetUSD.toFixed(2)}</Text>
        </Box>
        <Box marginTop={1}>
          <Text color={barColor}>{'█'.repeat(filledWidth)}</Text>
          <Text dimColor>{'░'.repeat(emptyWidth)}</Text>
          <Text> {budgetPercent}%</Text>
        </Box>
        <Box marginTop={1}>
          <Text dimColor>
            {formatNumber(totalTokens)} tokens
          </Text>
        </Box>
      </Box>
    </Panel>
  );
});

interface AgentStatsPanelProps {
  stats: {
    byState: Record<string, number>;
    byRole: Record<string, number>;
  };
}

/**
 * Memoized agent stats panel - only re-renders when stats change
 * Fixed: Use proper Box layout to prevent text overlap (#1065)
 * #1181 fix: Use Box for inline layout instead of nested Text with wrap="truncate"
 */
const AgentStatsPanel = memo(function AgentStatsPanel({ stats }: AgentStatsPanelProps) {
  const hasRoles = Object.keys(stats.byRole).length > 0;

  if (!hasRoles) return null;

  const roleEntries = Object.entries(stats.byRole);

  return (
    <Panel title="Agent Distribution">
      <Box flexDirection="column">
        <Text dimColor>By Role:</Text>
        <Box flexDirection="column" marginTop={1}>
          {roleEntries.map(([role, count]) => (
            <Box key={role}>
              <Text color="cyan">{role}</Text>
              <Text>: {count}</Text>
            </Box>
          ))}
        </Box>
      </Box>
    </Panel>
  );
});


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

/**
 * Format date to relative time string
 */
function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);

  if (diffSecs < 5) return 'just now';
  if (diffSecs < 60) return `${String(diffSecs)}s ago`;

  const diffMins = Math.floor(diffSecs / 60);
  if (diffMins < 60) return `${String(diffMins)}m ago`;

  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

export default Dashboard;
