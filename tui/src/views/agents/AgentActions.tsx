import React from 'react';
import { Box, Text } from 'ink';
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
  return (
    <Box marginTop={1} paddingX={1}>
      <Text dimColor>Actions: </Text>
      <Text color="cyan">[p]</Text>
      <Text dimColor> peek </Text>
      {agent.state !== 'stopped' && agent.state !== 'error' && (
        <>
          <Text color="yellow">[x]</Text>
          <Text dimColor> stop </Text>
        </>
      )}
      {agent.state !== 'stopped' && (
        <>
          <Text color="red">[X]</Text>
          <Text dimColor> kill </Text>
        </>
      )}
      <Text color="green">[R]</Text>
      <Text dimColor> restart </Text>
      <Text color="cyan">[Enter]</Text>
      <Text dimColor> attach</Text>
    </Box>
  );
}

export default AgentActions;
