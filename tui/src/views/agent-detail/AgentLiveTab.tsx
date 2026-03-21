/**
 * Live output tab for AgentDetailView
 * Shows real-time agent output with scroll and follow controls
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../../theme';
import { colorizeOutputLine } from '../../utils';

interface AgentLiveTabProps {
  liveLines: string[];
  scrollOffset: number;
  outputHeight: number;
  isFollowing: boolean;
}

export function AgentLiveTab({
  liveLines,
  scrollOffset,
  outputHeight,
  isFollowing,
}: AgentLiveTabProps): React.ReactElement {
  const { theme } = useTheme();
  return (
    <Box
      flexDirection="column"
      flexGrow={1}
      marginBottom={1}
      paddingX={1}
      borderStyle="single"
      borderColor={theme.colors.primary}
    >
      <Box marginBottom={1}>
        <Text color={theme.colors.primary} bold>LIVE OUTPUT</Text>
        <Text dimColor> | </Text>
        {isFollowing ? (
          <><Text color={theme.colors.success}>FOLLOWING</Text><Text dimColor> (2.5s)</Text></>
        ) : (
          <><Text color={theme.colors.warning}>PAUSED</Text><Text dimColor> (r: refresh)</Text></>
        )}
        <Text dimColor> | f: toggle</Text>
      </Box>
      <Box flexDirection="column" height={outputHeight + 2} overflow="hidden">
        {liveLines.length === 0 ? (
          <Text dimColor>Waiting for output...</Text>
        ) : (
          liveLines.slice(scrollOffset, scrollOffset + outputHeight).map((line, idx) => (
            <Text key={idx} wrap="truncate">
              {colorizeOutputLine(line)}
            </Text>
          ))
        )}
      </Box>
      {liveLines.length > outputHeight && (
        <Box marginTop={1}>
          <Text dimColor>
            Lines {scrollOffset + 1}-{Math.min(scrollOffset + outputHeight, liveLines.length)} of {liveLines.length}
            {isFollowing && ' (following)'}
          </Text>
        </Box>
      )}
    </Box>
  );
}
