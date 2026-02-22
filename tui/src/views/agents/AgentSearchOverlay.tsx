import React from 'react';
import { Box, Text } from 'ink';

export interface AgentSearchOverlayProps {
  searchQuery: string;
  isNarrow: boolean;
}

/**
 * AgentSearchOverlay - Search mode input UI
 * Shows when user presses '/' to search agents.
 * Extracted from AgentsView (#1592).
 */
export function AgentSearchOverlay({
  searchQuery,
  isNarrow,
}: AgentSearchOverlayProps): React.ReactElement {
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold>Search Agents</Text>
      <Box
        marginTop={1}
        borderStyle={isNarrow ? undefined : 'single'}
        borderColor="cyan"
        paddingX={1}
      >
        <Text color="cyan">{'> '}</Text>
        <Text>{searchQuery}</Text>
        <Text color="cyan">|</Text>
      </Box>
      <Box marginTop={1}>
        <Text dimColor>Enter to confirm, Esc to cancel</Text>
      </Box>
    </Box>
  );
}

export default AgentSearchOverlay;
