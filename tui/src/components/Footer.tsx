import React from 'react';
import { Box, Text } from 'ink';

export interface KeyHintProps {
  keyChar: string;
  label: string;
}

/**
 * KeyHint - Single keyboard shortcut hint
 */
export function KeyHint({ keyChar, label }: KeyHintProps) {
  return (
    <Box marginRight={2}>
      <Text>[</Text>
      <Text bold color="blue">{keyChar}</Text>
      <Text>] {label}</Text>
    </Box>
  );
}

export interface FooterProps {
  hints: Array<{ key: string; label: string }>;
}

/**
 * Footer - Keyboard shortcut hints bar
 * Shared component
 */
export function Footer({ hints }: FooterProps) {
  return (
    <Box
      borderStyle="single"
      borderTop
      borderBottom={false}
      borderLeft={false}
      borderRight={false}
      paddingX={1}
      marginTop={1}
    >
      {hints.map((h) => (
        <KeyHint key={h.key} keyChar={h.key} label={h.label} />
      ))}
    </Box>
  );
}

export default Footer;
