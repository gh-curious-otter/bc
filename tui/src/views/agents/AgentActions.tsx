import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';
import type { Agent } from '../../types';

export interface AgentActionsProps {
  agent: Agent;
}

/**
 * AgentActions - Inline action bar for selected agent
 * Shows available keyboard shortcuts for agent actions.
 * Extracted from AgentsView (#1592).
 */
export function AgentActions({ agent }: AgentActionsProps): React.ReactElement {
  const { theme } = useTheme();
  return (
    <Box marginTop={1} paddingX={1}>
      <Text dimColor>Actions: </Text>
      <Text color={theme.colors.primary}>[p]</Text>
      <Text dimColor> peek </Text>
      {agent.state !== 'stopped' && agent.state !== 'error' && (
        <>
          <Text color={theme.colors.warning}>[x]</Text>
          <Text dimColor> stop </Text>
        </>
      )}
      {agent.state !== 'stopped' && (
        <>
          <Text color={theme.colors.error}>[X]</Text>
          <Text dimColor> kill </Text>
        </>
      )}
      <Text color={theme.colors.success}>[R]</Text>
      <Text dimColor> start </Text>
      <Text color={theme.colors.primary}>[Enter]</Text>
      <Text dimColor> attach</Text>
    </Box>
  );
}

export default AgentActions;
