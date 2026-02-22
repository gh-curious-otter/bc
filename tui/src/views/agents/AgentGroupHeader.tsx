import React from 'react';
import { Box, Text } from 'ink';
import type { RoleGroup } from '../../hooks/useAgentGroups';

export interface AgentGroupHeaderProps {
  group: RoleGroup;
  isSelected: boolean;
  isCollapsed: boolean;
}

/**
 * AgentGroupHeader - Role group header row
 * Shows role name, count, and working/stuck indicators.
 * Extracted from AgentsView (#1592).
 */
export function AgentGroupHeader({
  group,
  isSelected,
  isCollapsed,
}: AgentGroupHeaderProps): React.ReactElement {
  return (
    <Box>
      <Text color={isSelected ? 'cyan' : 'white'} bold>
        {isSelected ? '▸ ' : '  '}
        {isCollapsed ? '▶' : '▼'}{' '}
      </Text>
      <Text bold color={isSelected ? 'cyan' : 'white'}>
        {group.role.toUpperCase()} ({group.agents.length})
      </Text>
      {group.working > 0 && (
        <Text color="blue"> ● {group.working}</Text>
      )}
      {group.stuck > 0 && (
        <Text color="yellow"> ⚠ {group.stuck}</Text>
      )}
    </Box>
  );
}

export default AgentGroupHeader;
