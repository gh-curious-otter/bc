import React from 'react';
import { Box, Text } from 'ink';

export interface PanelProps {
  title?: string;
  children: React.ReactNode;
  borderColor?: string;
  focused?: boolean;
  width?: number | string;
  height?: number | string;
}

/**
 * Panel - Bordered container with optional title
 * Shared component for all views
 */
export function Panel({
  title,
  children,
  borderColor = 'gray',
  focused = false,
  width,
  height,
}: PanelProps) {
  return (
    <Box
      flexDirection="column"
      borderStyle="single"
      borderColor={focused ? 'blue' : borderColor}
      width={width}
      height={height}
      paddingX={1}
      marginBottom={1}
    >
      {title && (
        <Box marginBottom={1}>
          <Text bold>{title}</Text>
        </Box>
      )}
      {children}
    </Box>
  );
}

export default Panel;
