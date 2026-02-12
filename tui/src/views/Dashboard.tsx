import { Box, Text } from 'ink';
import { Panel } from '../components/Panel.js';
import { MetricCard } from '../components/MetricCard.js';
import { DataTable } from '../components/DataTable.js';
import { StatusBadge } from '../components/StatusBadge';
import { Footer } from '../components/Footer.js';
import { LoadingIndicator } from '../components/LoadingIndicator.js';
import { ErrorDisplay } from '../components/ErrorDisplay.js';
import { useDashboard } from '../hooks/useDashboard.js';

/**
 * Dashboard view - main overview of bc workspace
 * Issue #543 - Dashboard layout
 */
export function Dashboard() {
  const { summary, agents, channels, isLoading, error, refresh } = useDashboard();

  if (error) {
    return <ErrorDisplay error={error} onRetry={refresh} />;
  }

  if (isLoading && !agents.data) {
    return <LoadingIndicator message="Loading workspace data..." />;
  }

  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Header workspaceName={summary.workspaceName} />

      {/* Summary Cards */}
      <SummaryCards
        total={summary.total}
        active={summary.active}
        working={summary.working}
        totalCostUSD={summary.totalCostUSD}
      />

      {/* Main Content */}
      <Box flexDirection="column" marginTop={1}>
        {/* Agents Panel */}
        <AgentsPanel agents={agents.data ?? []} />

        {/* Channels Panel */}
        <ChannelsPanel channels={channels.data ?? []} />
      </Box>

      {/* Footer with keyboard hints */}
      <Footer
        hints={[
          { key: 'a', label: 'agents' },
          { key: 'c', label: 'channels' },
          { key: 'r', label: 'refresh' },
          { key: 'q', label: 'quit' },
        ]}
      />
    </Box>
  );
}

interface HeaderProps {
  workspaceName: string;
}

function Header({ workspaceName }: HeaderProps) {
  return (
    <Box marginBottom={1}>
      <Text bold color="blue">bc</Text>
      <Text> · </Text>
      <Text>{workspaceName}</Text>
    </Box>
  );
}

interface SummaryCardsProps {
  total: number;
  active: number;
  working: number;
  totalCostUSD: number;
}

function SummaryCards({ total, active, working, totalCostUSD }: SummaryCardsProps) {
  return (
    <Box>
      <MetricCard value={total} label="Agents" />
      <MetricCard value={active} label="Active" color="green" />
      <MetricCard value={working} label="Working" color="cyan" />
      <MetricCard value={totalCostUSD.toFixed(2)} label="Cost" prefix="$" />
    </Box>
  );
}

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

function AgentsPanel({ agents }: AgentsPanelProps) {
  return (
    <Panel title="Agents">
      {agents.length === 0 ? (
        <Text dimColor>No agents running</Text>
      ) : (
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
            { key: 'startedAt', header: 'STARTED', width: 10 },
            { key: 'task', header: 'TASK' },
          ]}
          data={agents}
        />
      )}
    </Panel>
  );
}

interface Channel {
  name: string;
  members: string[];
  messageCount?: number;
}

interface ChannelsPanelProps {
  channels: Channel[];
}

function ChannelsPanel({ channels }: ChannelsPanelProps) {
  return (
    <Panel title="Channels">
      {channels.length === 0 ? (
        <Text dimColor>No channels</Text>
      ) : (
        <Box flexDirection="column">
          {channels.map((ch) => (
            <Box key={ch.name}>
              <Text color="cyan">#{ch.name}</Text>
              <Text> </Text>
              <Text dimColor>{ch.members.length} members</Text>
            </Box>
          ))}
        </Box>
      )}
    </Panel>
  );
}

export default Dashboard;
