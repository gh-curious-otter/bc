import { memo, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { Panel } from '../components/Panel.js';
import { MetricCard } from '../components/MetricCard.js';
import { DataTable } from '../components/DataTable.js';
import { StatusBadge } from '../components/StatusBadge';
import { Footer } from '../components/Footer.js';
import { LoadingIndicator } from '../components/LoadingIndicator.js';
import { ErrorDisplay } from '../components/ErrorDisplay.js';
import { ActivityFeed } from '../components/ActivityFeed.js';
import { useDashboard } from '../hooks/useDashboard.js';

interface DashboardProps {
  onNavigate?: (view: string) => void;
}

/**
 * Dashboard view - main overview of bc workspace
 * Issues #543 (layout), #544 (stats components)
 */
export function Dashboard({ onNavigate }: DashboardProps) {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  const {
    summary,
    agents,
    channels,
    agentStats,
    isLoading,
    error,
    refresh,
    lastRefresh,
  } = useDashboard();

  // Keyboard navigation
  useInput((input, key) => {
    if (input === 'a') {
      onNavigate?.('agents');
    }
    if (input === 'c') {
      onNavigate?.('channels');
    }
    if (input === '$') {
      onNavigate?.('costs');
    }
    if (input === 'r') {
      void refresh();
    }
    if (input === 'q' || key.escape) {
      onNavigate?.('quit');
    }
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

      {/* Cost Summary */}
      <CostSummary
        totalCostUSD={summary.totalCostUSD}
        inputTokens={summary.inputTokens}
        outputTokens={summary.outputTokens}
      />

      {/* Main Content - Two column layout */}
      <Box marginTop={1}>
        {/* Left column - Main panels */}
        <Box flexDirection="column" flexGrow={1}>
          {/* Agent Stats by Role */}
          <AgentStatsPanel stats={agentStats} />

          {/* Agents Panel */}
          <AgentsPanel agents={agents.data ?? []} />

          {/* Channels Panel */}
          <ChannelsPanel channels={channels.data ?? []} />
        </Box>

        {/* Right column - Activity feed (compact) */}
        <Box flexDirection="column" width={45} marginLeft={1}>
          <ActivityFeed maxEntries={10} compact showFilterHints={false} />
        </Box>
      </Box>

      {/* Footer with keyboard hints */}
      <Footer
        hints={[
          { key: 'a', label: 'agents' },
          { key: 'c', label: 'channels' },
          { key: '$', label: 'costs' },
          { key: 'r', label: 'refresh' },
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
    <Box>
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

interface CostSummaryProps {
  totalCostUSD: number;
  inputTokens: number;
  outputTokens: number;
}

/**
 * Memoized cost summary - only re-renders when cost data changes
 */
const CostSummary = memo(function CostSummary({
  totalCostUSD,
  inputTokens,
  outputTokens,
}: CostSummaryProps) {
  const totalTokens = inputTokens + outputTokens;

  return (
    <Box marginTop={1}>
      <Text bold>Cost: </Text>
      <Text color="yellow">${totalCostUSD.toFixed(4)}</Text>
      <Text> · </Text>
      <Text dimColor>
        {formatNumber(totalTokens)} tokens ({formatNumber(inputTokens)} in /{' '}
        {formatNumber(outputTokens)} out)
      </Text>
    </Box>
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
 */
const AgentStatsPanel = memo(function AgentStatsPanel({ stats }: AgentStatsPanelProps) {
  const hasRoles = Object.keys(stats.byRole).length > 0;

  if (!hasRoles) return null;

  return (
    <Panel title="Agent Distribution">
      <Box>
        {/* By Role */}
        <Box marginRight={4}>
          <Text dimColor>By Role: </Text>
          {Object.entries(stats.byRole).map(([role, count], idx, arr) => (
            <Text key={role}>
              <Text color="cyan">{role}</Text>
              <Text>:{count}</Text>
              {idx < arr.length - 1 && <Text> · </Text>}
            </Text>
          ))}
        </Box>
      </Box>
    </Panel>
  );
});

interface Agent {
  name: string;
  role: string;
  state: string;
  startedAt: string;
  updatedAt: string;
  task: string;
  [key: string]: unknown;
}

interface AgentsPanelProps {
  agents: Agent[];
}

/**
 * Memoized agents panel - only re-renders when agents array changes
 */
const AgentsPanel = memo(function AgentsPanel({ agents }: AgentsPanelProps) {
  // Memoize displayed agents slice
  const displayAgents = useMemo(() => agents.slice(0, 5), [agents]);
  const hasMore = agents.length > 5;

  return (
    <Panel title="Agents">
      {agents.length === 0 ? (
        <Text dimColor>No agents running</Text>
      ) : (
        <>
          <DataTable
            columns={[
              { key: 'name', header: 'AGENT', width: 15 },
              { key: 'role', header: 'ROLE', width: 12 },
              {
                key: 'state',
                header: 'STATE',
                width: 10,
                render: (value) => <StatusBadge state={value as string} />,
              },
              { key: 'updatedAt', header: 'UPDATED', width: 10 },
              { key: 'task', header: 'TASK' },
            ]}
            data={displayAgents}
          />
          {hasMore && (
            <Text dimColor>
              ... and {agents.length - 5} more (press &apos;a&apos; to view all)
            </Text>
          )}
        </>
      )}
    </Panel>
  );
});

interface Channel {
  name: string;
  members: string[];
  messageCount?: number;
}

interface ChannelsPanelProps {
  channels: Channel[];
}

/**
 * Memoized channels panel - only re-renders when channels array changes
 */
const ChannelsPanel = memo(function ChannelsPanel({ channels }: ChannelsPanelProps) {
  // Memoize displayed channels slice
  const displayChannels = useMemo(() => channels.slice(0, 5), [channels]);

  return (
    <Panel title="Channels">
      {channels.length === 0 ? (
        <Text dimColor>No channels</Text>
      ) : (
        <Box flexDirection="column">
          {displayChannels.map((ch) => (
            <Box key={ch.name}>
              <Text color="cyan">#{ch.name}</Text>
              <Text> </Text>
              <Text dimColor>{ch.members.length} members</Text>
            </Box>
          ))}
          {channels.length > 5 && (
            <Text dimColor>
              ... and {channels.length - 5} more (press &apos;c&apos; to view all)
            </Text>
          )}
        </Box>
      )}
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
