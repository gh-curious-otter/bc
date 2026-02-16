import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { useAgents } from '../hooks';
import { Table } from '../components/Table';
import type { Column } from '../components/Table';
import { StatusBadge } from '../components/StatusBadge';
import { AgentDetailView } from './AgentDetailView';
import type { Agent } from '../types';

interface AgentsViewProps {
  onBack?: () => void;
}

export const AgentsView: React.FC<AgentsViewProps> = ({
  onBack,
}) => {
  const { data: agents, loading, error, refresh } = useAgents();
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showDetail, setShowDetail] = useState(false);
  const agentList = agents ?? [];
  const selectedAgent = agentList[selectedIndex];

  // Keyboard navigation
  useInput((input, key) => {
    if (showDetail) {
      // Detail view handles its own keybinds via AgentDetailView
      return;
    }

    // List view navigation
    if (key.upArrow || input === 'k') {
      setSelectedIndex((i) => Math.max(0, i - 1));
    } else if (key.downArrow || input === 'j') {
      setSelectedIndex((i) => Math.min(agentList.length - 1, i + 1));
    } else if (key.return) {
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
      if (selectedAgent) {
        setShowDetail(true);
      }
    } else if (input === 'r') {
      void refresh();
    } else if (input === 'q' || key.escape) {
      onBack?.();
    }
  });

  // If showing detail view, render AgentDetailView instead
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
  if (showDetail && selectedAgent) {
    return (
      <AgentDetailView
        agent={selectedAgent}
        onBack={() => { setShowDetail(false); }}
      />
    );
  }

  const columns: Column<Agent>[] = [
    {
      key: 'name',
      header: 'Name',
      width: 18,
    },
    {
      key: 'role',
      header: 'Role',
      width: 12,
    },
    {
      key: 'state',
      header: 'State',
      width: 12,
      render: (agent) => <StatusBadge state={agent.state} />,
    },
    {
      key: 'task',
      header: 'Task',
      width: 40,
      render: (agent) => (
        <Text wrap="truncate">
          {agent.task ? agent.task.slice(0, 38) : '-'}
        </Text>
      ),
    },
  ];

  if (loading && agentList.length === 0) {
    return (
      <Box padding={1}>
        <Text color="yellow">Loading agents...</Text>
      </Box>
    );
  }

  if (error) {
    return (
      <Box padding={1}>
        <Text color="red">Error: {error}</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="green">
          Agents ({agentList.length})
        </Text>
        {loading && <Text color="gray"> (refreshing...)</Text>}
      </Box>

      {/* Agents Table */}
      <Table
        data={agentList}
        columns={columns}
        selectedIndex={selectedIndex}
      />

      {/* Footer with keybindings */}
      <Box marginTop={1}>
        <Text color="gray">
          j/k: navigate | Enter: attach | r: refresh | q: back
        </Text>
      </Box>
    </Box>
  );
};

export default AgentsView;
