import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';

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
  const { theme } = useTheme();
  return (
    <Box flexDirection="column" padding={1}>
      <Text bold>Search Agents</Text>
      <Box
        marginTop={1}
        borderStyle={isNarrow ? undefined : 'single'}
        borderColor={theme.colors.primary}
        paddingX={1}
      >
        <Text color={theme.colors.primary}>{'> '}</Text>
        <Text>{searchQuery}</Text>
        <Text color={theme.colors.primary}>|</Text>
      </Box>
      <Box marginTop={1}>
        <Text dimColor>Enter to confirm, Esc to cancel</Text>
      </Box>
    </Box>
  );
}

export default AgentSearchOverlay;
