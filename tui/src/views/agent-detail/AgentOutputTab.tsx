/**
 * Output tab for AgentDetailView
 * Shows agent output with message input area
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';
import { LoadingIndicator } from '../../components/LoadingIndicator';
import { colorizeOutputLine } from '../../utils';

interface AgentOutputTabProps {
  outputLines: string[];
  loading: boolean;
  error: string | null;
  inputMode: boolean;
  messageBuffer: string;
  sendStatus: string | null;
  outputHeight: number;
}

export function AgentOutputTab({
  outputLines,
  loading,
  error,
  inputMode,
  messageBuffer,
  sendStatus,
  outputHeight,
}: AgentOutputTabProps): React.ReactElement {
  const { theme } = useTheme();
  return (
    <>
      {/* #1161: Output box with bottom-aligned content and preserved colors */}
      <Box
        flexDirection="column"
        flexGrow={1}
        marginBottom={1}
        paddingX={1}
        borderStyle="single"
        borderColor={theme.colors.textMuted}
        height={outputHeight}
        justifyContent="flex-end"
      >
        {loading && outputLines.length === 0 ? (
          <LoadingIndicator message="Loading agent output..." />
        ) : error ? (
          <Text color={theme.colors.error}>Error: {error}</Text>
        ) : outputLines.length === 0 ? (
          <Text dimColor>No output yet. Agent may be idle.</Text>
        ) : (
          outputLines.slice(-outputHeight + 2).map((line, idx) => (
            <Text key={idx} wrap="truncate">
              {colorizeOutputLine(line)}
            </Text>
          ))
        )}
      </Box>

      <Box
        flexDirection="column"
        height={4}
        marginBottom={1}
        paddingX={1}
        borderStyle="single"
        borderColor={inputMode ? theme.colors.primary : theme.colors.textMuted}
      >
        {inputMode ? (
          <Box>
            <Text color={theme.colors.primary}>{'> '}</Text>
            <Text>{messageBuffer}</Text>
            <Text color={theme.colors.primary}>|</Text>
          </Box>
        ) : (
          <Text dimColor>Press i or m to send message</Text>
        )}
        {sendStatus && (
          <Box marginTop={1}>
            <Text color={theme.colors.success}>{sendStatus}</Text>
          </Box>
        )}
      </Box>
    </>
  );
}
