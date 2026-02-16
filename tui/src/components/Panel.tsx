import React from 'react';
import { Box, Text } from 'ink';

export interface PanelProps {
  title?: string;
  children: React.ReactNode;
  borderColor?: string;
  focused?: boolean;
  width?: number | string;
  height?: number | string;
  /** Minimum height to prevent collapse at narrow widths */
  minHeight?: number;
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
  minHeight,
}: PanelProps) {
  // #984 fix: Calculate minimum height to prevent panel collapse at narrow widths
  // Default minHeight ensures title + at least 1 line of content is visible
  const effectiveMinHeight = minHeight ?? (title ? 4 : 3);

  return (
    <Box
      flexDirection="column"
      borderStyle="single"
      borderColor={focused ? 'blue' : borderColor}
      width={width}
      height={height}
      minHeight={effectiveMinHeight}
      paddingX={1}
      marginBottom={1}
      overflow="hidden"
    >
      {title && (
        <Text bold wrap="truncate">{title}</Text>
      )}
      {children}
    </Box>
  );
}

export default Panel;
