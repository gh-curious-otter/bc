import React from 'react';
import { Box, Text } from 'ink';
import { StatusBadge } from '../../components/StatusBadge';
import { normalizeTask } from '../../hooks/useAgentGroups';
import type { Agent } from '../../types';

export interface AgentCardProps {
  agent: Agent;
  isSelected: boolean;
  /** Indent for grouped view */
  indent?: boolean;
}

/**
 * AgentCard - Individual agent row display
 * Shows agent name, state badge, and current task.
 * Extracted from AgentsView (#1592).
 */
export function AgentCard({
  agent,
  isSelected,
  indent = false,
}: AgentCardProps): React.ReactElement {
  const displayName = agent.name.length > 12
    ? agent.name.slice(0, 11) + '…'
    : agent.name.padEnd(12);

  return (
    <Box marginLeft={indent ? 2 : 0}>
      <Text color={isSelected ? 'cyan' : undefined}>
        {isSelected ? '▸ ' : '  '}
      </Text>
      <Text color={isSelected ? 'cyan' : undefined}>
        {displayName}
      </Text>
      <Text> </Text>
      <StatusBadge state={agent.state} />
      <Text> </Text>
      <Text wrap="truncate" dimColor>
        {normalizeTask(agent.task).slice(0, 30)}
      </Text>
    </Box>
  );
}

export default AgentCard;
