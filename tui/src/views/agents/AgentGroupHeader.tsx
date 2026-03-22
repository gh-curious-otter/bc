import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';
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
  const { theme } = useTheme();
  return (
    <Box>
      <Text color={isSelected ? theme.colors.primary : theme.colors.text} bold>
        {isSelected ? '▸ ' : '  '}
        {isCollapsed ? '▶' : '▼'}{' '}
      </Text>
      <Text bold color={isSelected ? theme.colors.primary : theme.colors.text}>
        {group.role.toUpperCase()} ({group.agents.length})
      </Text>
      {group.working > 0 && <Text color={theme.colors.secondary}> ● {group.working}</Text>}
      {group.stuck > 0 && <Text color={theme.colors.warning}> ⚠ {group.stuck}</Text>}
    </Box>
  );
}

export default AgentGroupHeader;
