import React from 'react';
import { Box } from 'ink';
import { Table } from '../../components/Table';
import type { Column } from '../../components/Table';
import { AgentCard } from './AgentCard';
import { AgentGroupHeader } from './AgentGroupHeader';
import type { GroupedItem } from '../../hooks/useAgentGroups';
import { abbreviateRole, normalizeTask } from '../../hooks/useAgentGroups';
import { Text } from 'ink';
import { StatusBadge } from '../../components/StatusBadge';
import type { Agent } from '../../types';

export interface AgentListProps {
  items: GroupedItem[];
  agents: Agent[];
  selectedIndex: number;
  groupedView: boolean;
  collapsedRoles: Set<string>;
}

/**
 * AgentList - Renders agent list in grouped or table mode
 * Handles both the grouped collapsible view and flat table view.
 * Extracted from AgentsView (#1592).
 */
export function AgentList({
  items,
  agents,
  selectedIndex,
  groupedView,
  collapsedRoles,
}: AgentListProps): React.ReactElement {
  // Table columns for flat view
  const columns: Column<Agent>[] = [
    {
      key: 'name',
      header: 'Name',
      width: 14,
      render: (agent) => (
        <Text>{agent.name.length > 12 ? agent.name.slice(0, 11) + '…' : agent.name}</Text>
      ),
    },
    {
      key: 'role',
      header: 'Role',
      width: 10,
      render: (agent) => (
        <Text>{abbreviateRole(agent.role)}</Text>
      ),
    },
    {
      key: 'state',
      header: 'State',
      width: 10,
      render: (agent) => <StatusBadge state={agent.state} />,
    },
    {
      key: 'task',
      header: 'Task',
      width: 32,
      render: (agent) => (
        <Text wrap="truncate">
          {normalizeTask(agent.task).slice(0, 30)}
        </Text>
      ),
    },
  ];

  if (!groupedView) {
    // Show helpful empty state message (#1607)
    if (agents.length === 0) {
      return (
        <Box flexDirection="column" padding={1}>
          <Text dimColor>No agents yet.</Text>
          <Text dimColor>Create one with: bc agent create --role &lt;role&gt;</Text>
        </Box>
      );
    }
    return (
      <Table
        data={agents}
        columns={columns}
        selectedIndex={selectedIndex}
      />
    );
  }

  // Show helpful empty state for grouped view (#1607)
  if (items.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text dimColor>No agents yet.</Text>
        <Text dimColor>Create one with: bc agent create --role &lt;role&gt;</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {items.map((item, idx) => {
        const isSelected = idx === selectedIndex;

        if (item.type === 'header') {
          return (
            <AgentGroupHeader
              key={`header-${item.role}`}
              group={item.group}
              isSelected={isSelected}
              isCollapsed={collapsedRoles.has(item.role)}
            />
          );
        }

        return (
          <AgentCard
            key={`agent-${item.agent.name}`}
            agent={item.agent}
            isSelected={isSelected}
            indent
          />
        );
      })}
    </Box>
  );
}

export default AgentList;
