import React from 'react';
import { Box, Text, useStdout } from 'ink';
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
 *
 * Issue #1689: Fixed text cutoff by using full terminal width
 * and proper text wrapping instead of truncation.
 */
export function AgentPeekPanel({
  agent,
  output,
  loading,
  isNarrow,
}: AgentPeekPanelProps): React.ReactElement {
  const { stdout } = useStdout();
  // Use full terminal width, accounting for padding and borders
  const terminalWidth = stdout.columns || 80;
  // Border (2) + paddingX (1 each side when not narrow) = 4
  const contentWidth = isNarrow ? terminalWidth : terminalWidth - 4;

  return (
    <Box
      marginBottom={1}
      paddingX={isNarrow ? 0 : 1}
      borderStyle={isNarrow ? undefined : 'single'}
      borderColor="cyan"
      flexDirection="column"
      width={terminalWidth}
    >
      <Box marginBottom={1}>
        <Text bold color="cyan">Peek: {agent.name}</Text>
        <Text dimColor> (press p to close)</Text>
      </Box>
      {loading ? (
        <Text dimColor>Loading...</Text>
      ) : (
        <Box flexDirection="column" width={contentWidth}>
          {output.map((line, idx) => (
            <Text key={idx} wrap="wrap" dimColor>{line}</Text>
          ))}
        </Box>
      )}
    </Box>
  );
}

export default AgentPeekPanel;
