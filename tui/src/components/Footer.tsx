import { memo } from 'react';
import { Box, Text } from 'ink';

export interface KeyHintProps {
  keyChar: string;
  label: string;
}

/**
 * KeyHint - Single keyboard shortcut hint
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 */
export const KeyHint = memo(function KeyHint({ keyChar, label }: KeyHintProps) {
  return (
    <Box marginRight={2}>
      <Text>[</Text>
      <Text bold color="blue">{keyChar}</Text>
      <Text>] {label}</Text>
    </Box>
  );
});

/** Type for keybinding hint items */
export type HintItem = { key: string; label: string };

export interface FooterProps {
  hints: HintItem[];
}

/**
 * Footer - Keyboard shortcut hints bar
 * Shared component
 *
 * Memoized for performance - Issue #1003 Phase 3 optimization.
 * Issue #1362: Use flexWrap to prevent truncation of keybindings
 */
export const Footer = memo(function Footer({ hints }: FooterProps) {
  return (
    <Box
      borderStyle="single"
      borderTop
      borderBottom={false}
      borderLeft={false}
      borderRight={false}
      paddingX={1}
      marginTop={1}
      flexWrap="wrap"
    >
      {hints.map((h) => (
        <KeyHint key={h.key} keyChar={h.key} label={h.label} />
      ))}
    </Box>
  );
});

export default Footer;
