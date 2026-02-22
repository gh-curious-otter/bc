import React from 'react';
import { Box, Text } from 'ink';
import type { Agent } from '../../types';

export interface AgentPeekPanelProps {
  agent: Agent;
  output: string[];
  loading: boolean;
  isNarrow: boolean;
}

/**
 * AgentPeekPanel - Shows recent output from an agent
 * Displays the last few lines of agent output.
 * Extracted from AgentsView (#1592).
 */
export function AgentPeekPanel({
  agent,
  output,
  loading,
  isNarrow,
}: AgentPeekPanelProps): React.ReactElement {
  return (
    <Box
      marginBottom={1}
      paddingX={isNarrow ? 0 : 1}
      borderStyle={isNarrow ? undefined : 'single'}
      borderColor="cyan"
      flexDirection="column"
    >
      <Box marginBottom={1}>
        <Text bold color="cyan">Peek: {agent.name}</Text>
        <Text dimColor> (press p to close)</Text>
      </Box>
      {loading ? (
        <Text dimColor>Loading...</Text>
      ) : (
        output.map((line, idx) => (
          <Text key={idx} wrap="truncate" dimColor>{line}</Text>
        ))
      )}
    </Box>
  );
}

export default AgentPeekPanel;
