import { memo, useState, useCallback } from 'react';
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
import { useResponsiveLayout } from '../hooks/useResponsiveLayout.js';
import { STATUS_COLORS, STATUS_SYMBOLS, HEALTH_COLORS, getCostIndicator, type CostStatus } from '../theme/StatusColors.js';

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

  // #1596: Memoize keyboard input handler
  const handleDashboardInput = useCallback((input: string, key: { ctrl: boolean }) => {
    // Refresh is Dashboard-specific (global Ctrl+R handled elsewhere)
    if (input === 'r') {
      void refresh();
    }
    // Performance overlay toggle - Ctrl+P (Phase 3 integration #1032)
    if (key.ctrl && input === 'p') {
      setShowDebugPanel(prev => !prev);
    }
    // Note: q and ESC are handled by global useKeyboardNavigation
  }, [refresh]);

  // Keyboard navigation - Dashboard-specific shortcuts
  // Global shortcuts (1-9, Tab, ESC, q) are handled by useKeyboardNavigation
  // Note: Letter shortcuts (a, c, $) removed to avoid confusion - use numbers [2] [3] [4]
  // See #1327 for comprehensive keybinding system design
  useInput(handleDashboardInput);

  if (error) {
    return <ErrorDisplay error={error.message} onRetry={() => { void refresh(); }} />;
  }

  // Progressive loading: show content structure while data loads (#1614)
  // Only block on initial load when no data exists yet
  const showInitialLoading = isLoading && !agents.data;

  // #1318: Don't set explicit width - parent flexGrow handles it
  // Setting width={terminalWidth} caused overflow when nested inside drawer layout
  return (
    <Box flexDirection="column" padding={1}>
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

      {/* #1614: Simplified layout - StatsPanels renders once with layout prop */}
      {/* Main Content - Uses responsive layout for flexible column arrangement */}
      <Box marginTop={1} flexDirection={canMultiColumn ? 'row' : 'column'}>
        {/* Stats panels at top when narrow */}
        {!canMultiColumn && (
          <StatsPanels
            summary={summary}
            agentStats={agentStats}
            showInitialLoading={showInitialLoading}
          />
        )}

        {/* Activity Feed - primary focus */}
        <Box flexDirection="column" flexGrow={1} marginRight={canMultiColumn ? 1 : 0}>
          {showInitialLoading ? (
            <LoadingIndicator message="Loading activity..." />
          ) : (
            <ActivityFeed maxEntries={isMedium || isWide ? 15 : 8} compact={!isWide} showFilterHints={canMultiColumn} />
          )}
        </Box>

        {/* Stats & Health panels - side column when space allows (medium+) */}
        {canMultiColumn && (
          <Box flexDirection="column" width={Math.min(32, Math.max(26, Math.floor((terminalWidth - 4) * 0.28)))}>
            <StatsPanels
              summary={summary}
              agentStats={agentStats}
              showInitialLoading={showInitialLoading}
            />
          </Box>
        )}
      </Box>

      {/* Performance Debug Panel - toggled with Ctrl+P or F12 (Phase 3) */}
      {showDebugPanel && <PerformanceDebugPanel compact={!isWide} forceShow />}

      {/* Footer with keyboard hints - Issue #1514: use drawer navigation (#1467) */}
      <Footer
        hints={[
          { key: 'Tab', label: 'views' },
          { key: 'j/k', label: 'drawer' },
          { key: 'Enter', label: 'select' },
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
 * #1352: Uses inline text at <100 cols to prevent border overlap
 */
const SummaryCards = memo(function SummaryCards({
  total,
  active,
  working,
  idle,
  stuck,
  errorCount,
}: SummaryCardsProps) {
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;

  // #1352: Inline text summary for narrow terminals to avoid border overlap
  // #1591: Added symbols alongside colors for accessibility
  if (isNarrow) {
    return (
      <Box marginBottom={1}>
        <Text>{total} agents</Text>
        <Text> · </Text>
        <Text color="cyan">{STATUS_SYMBOLS.working} {working} working</Text>
        <Text> · </Text>
        <Text color="gray">{STATUS_SYMBOLS.idle} {idle} idle</Text>
        {stuck > 0 && (
          <>
            <Text> · </Text>
            <Text color="yellow">{STATUS_SYMBOLS.warning} {stuck} stuck</Text>
          </>
        )}
        {errorCount > 0 && (
          <>
            <Text> · </Text>
            <Text color="red">{STATUS_SYMBOLS.error} {errorCount} error</Text>
          </>
        )}
      </Box>
    );
  }

  // Standard bordered MetricCards for wider terminals
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
 * #1352: Uses simple text header at <100 cols to prevent border overlap
 */
const SystemHealthPanel = memo(function SystemHealthPanel({
  working,
  idle,
  stuck,
  errorCount,
  total,
}: SystemHealthPanelProps) {
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;
  const healthyCount = working + idle;
  const unhealthyCount = stuck + errorCount;
  const healthPercent = total > 0 ? Math.round((healthyCount / total) * 100) : 100;
  const healthColor = healthPercent >= 80 ? HEALTH_COLORS.healthy : healthPercent >= 50 ? HEALTH_COLORS.warning : HEALTH_COLORS.critical;

  // #1352: Compact borderless layout for narrow terminals
  if (isNarrow) {
    return (
      <Box flexDirection="column" marginBottom={1}>
        <Box>
          <Text bold dimColor>Health: </Text>
          <Text color={healthColor} bold>{healthPercent}%</Text>
          <Text> · </Text>
          <PulseText color={STATUS_COLORS.working} enabled={working > 0} interval={1500}>●</PulseText>
          <Text>{working}</Text>
          <Text> · </Text>
          <Text color={STATUS_COLORS.idle}>●</Text>
          <Text>{idle}</Text>
          {stuck > 0 && (
            <>
              <Text> · </Text>
              <Text color={STATUS_COLORS.warning}>●</Text>
              <Text>{stuck}</Text>
            </>
          )}
        </Box>
      </Box>
    );
  }

  // Standard bordered Panel for wider terminals
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
 * #1220: Added symbols and text labels for colorblind accessibility
 * #1352: Uses compact inline layout at <100 cols
 */
const CostPanel = memo(function CostPanel({
  totalCostUSD,
  inputTokens,
  outputTokens,
  budgetUSD = 10.0,
}: CostPanelProps) {
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;
  const totalTokens = inputTokens + outputTokens;
  const budgetPercent = Math.min(100, Math.round((totalCostUSD / budgetUSD) * 100));
  // Responsive bar width: smaller on narrow terminals
  const barWidth = isNarrow ? 10 : 15;
  const filledWidth = Math.round((budgetPercent / 100) * barWidth);
  const emptyWidth = barWidth - filledWidth;

  // Determine cost status for symbol and label (#1220 colorblind support)
  const costStatus: CostStatus = budgetPercent >= 90 ? 'critical' : budgetPercent >= 75 ? 'warning' : 'normal';
  const { color: barColor, symbol: costSymbol } = getCostIndicator(costStatus);

  // #1352: Compact inline layout for narrow terminals
  if (isNarrow) {
    return (
      <Box marginBottom={1}>
        <Text bold dimColor>Cost: </Text>
        <Text bold color="yellow">${totalCostUSD.toFixed(2)}</Text>
        <Text dimColor>/${budgetUSD.toFixed(2)} </Text>
        <Text color={barColor}>{'█'.repeat(filledWidth)}</Text>
        <Text dimColor>{'░'.repeat(emptyWidth)}</Text>
        <Text> {budgetPercent}%</Text>
        <Text color={barColor}> {costSymbol}</Text>
      </Box>
    );
  }

  // Standard bordered Panel for wider terminals
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
          {/* #1220: Symbol indicator for colorblind users */}
          <Text color={barColor}> {costSymbol}</Text>
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
 * #1352: Uses compact inline layout at <100 cols
 */
const AgentStatsPanel = memo(function AgentStatsPanel({ stats }: AgentStatsPanelProps) {
  const { isCompact, isMinimal } = useResponsiveLayout();
  const isNarrow = isCompact || isMinimal;
  const hasRoles = Object.keys(stats.byRole).length > 0;

  if (!hasRoles) return null;

  const roleEntries = Object.entries(stats.byRole);

  // #1338: Truncate role names at narrow widths to prevent text corruption
  const MAX_ROLE_LEN = isNarrow ? 8 : 12;

  // #1352: Compact inline layout for narrow terminals
  if (isNarrow) {
    // Show top 3 roles inline: "eng: 5 · mgr: 2 · ux: 1"
    const topRoles = roleEntries.slice(0, 3);
    return (
      <Box marginBottom={1}>
        <Text bold dimColor>Roles: </Text>
        {topRoles.map(([role, count], idx) => {
          const displayRole = role.length > MAX_ROLE_LEN
            ? role.slice(0, MAX_ROLE_LEN - 1) + '…'
            : role;
          return (
            <Text key={role}>
              {idx > 0 && ' · '}
              {displayRole}: {count}
            </Text>
          );
        })}
        {roleEntries.length > 3 && (
          <Text dimColor> +{roleEntries.length - 3}</Text>
        )}
      </Box>
    );
  }

  // Standard bordered Panel for wider terminals
  return (
    <Panel title="Agent Distribution">
      <Box flexDirection="column">
        <Text dimColor>By Role:</Text>
        <Box flexDirection="column" marginTop={1}>
          {roleEntries.map(([role, count]) => {
            // Truncate long role names to prevent overflow at narrow widths
            const displayRole = role.length > MAX_ROLE_LEN
              ? role.slice(0, MAX_ROLE_LEN - 1) + '…'
              : role;
            // #1338: Use single Text with wrap="truncate" to prevent text corruption
            // Avoid nested Box which causes layout issues at 80x24
            return (
              <Text key={role} wrap="truncate">
                {displayRole}: {count}
              </Text>
            );
          })}
        </Box>
      </Box>
    </Panel>
  );
});

interface StatsPanelsProps {
  summary: {
    working: number;
    idle: number;
    stuck: number;
    error: number;
    total: number;
    totalCostUSD: number;
    inputTokens: number;
    outputTokens: number;
  };
  agentStats: {
    byState: Record<string, number>;
    byRole: Record<string, number>;
  };
  showInitialLoading: boolean;
}

/**
 * StatsPanels - Consolidated stats display (#1614)
 * Reduces layout complexity by rendering panels once
 */
const StatsPanels = memo(function StatsPanels({
  summary,
  agentStats,
  showInitialLoading,
}: StatsPanelsProps) {
  if (showInitialLoading) {
    return <LoadingIndicator message="Loading stats..." />;
  }

  return (
    <>
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
      <AgentStatsPanel stats={agentStats} />
    </>
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
