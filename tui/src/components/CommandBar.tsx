/**
 * CommandBar - k9s-style :command navigation
 *
 * Activated by pressing ':'. Shows a text input at the bottom
 * with fuzzy-matched view suggestions above it.
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import type { View } from '../navigation/NavigationContext';
import { searchCommands, resolveCommand, resolveAction, type MatchedCommand } from '../navigation/viewCommands';

interface CommandBarProps {
  onSelect: (view: View) => void;
  onClose: () => void;
  /** #1871: Recently used command names for LRU ordering */
  recentCommands?: string[];
  /** #1871: Callback when a command is selected (for LRU tracking) */
  onCommandUsed?: (command: string) => void;
}

const MAX_SUGGESTIONS = 10;

export function CommandBar({ onSelect, onClose, recentCommands = [], onCommandUsed }: CommandBarProps): React.ReactElement {
  const [input, setInput] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);

  const matches = searchCommands(input, recentCommands).slice(0, MAX_SUGGESTIONS);

  useInput((char, key) => {
    if (key.escape) {
      onClose();
      return;
    }

    if (key.return) {
      // #1836: Check action commands first (:q, :q!, :quit)
      const action = resolveAction(input);
      if (action === 'quit' || action === 'force-quit') {
        process.exit(0);
        return;
      }

      // Try exact view resolve first, then use selected suggestion
      const resolved = resolveCommand(input);
      if (resolved) {
        onCommandUsed?.(input.toLowerCase().trim());
        onSelect(resolved);
      } else if (matches.length > 0) {
        onCommandUsed?.(matches[selectedIndex].command.command);
        onSelect(matches[selectedIndex].command.view);
      }
      return;
    }

    if (key.tab) {
      // Auto-complete with selected suggestion
      if (matches.length > 0) {
        setInput(matches[selectedIndex].command.command);
      }
      return;
    }

    if (key.upArrow) {
      setSelectedIndex(prev => Math.max(0, prev - 1));
      return;
    }

    if (key.downArrow) {
      setSelectedIndex(prev => Math.min(matches.length - 1, prev + 1));
      return;
    }

    if (key.backspace || key.delete) {
      setInput(prev => prev.slice(0, -1));
      setSelectedIndex(0);
      return;
    }

    // Regular character input
    if (char && !key.ctrl && !key.meta) {
      setInput(prev => prev + char);
      setSelectedIndex(0);
    }
  });

  return (
    <Box flexDirection="column">
      {/* Suggestions dropdown */}
      {matches.map((match: MatchedCommand, idx: number) => (
        <Box key={match.command.command}>
          <Text color={idx === selectedIndex ? 'cyan' : undefined}>
            {idx === selectedIndex ? '> ' : '  '}
          </Text>
          <Text color={idx === selectedIndex ? 'cyan' : 'white'} bold={idx === selectedIndex}>
            {match.command.aliases[0] ?? match.command.command}
          </Text>
          <Text>{'  '}</Text>
          <Text dimColor={idx !== selectedIndex}>
            {match.command.command}
          </Text>
          <Text>{'  '}</Text>
          <Text dimColor color="gray">
            {match.command.section}
          </Text>
        </Box>
      ))}

      {/* Input line */}
      <Box>
        <Text color="cyan" bold>: </Text>
        <Text>{input}</Text>
        <Text color="gray">|</Text>
        <Text dimColor>{'  [Tab] complete  [Esc] cancel'}</Text>
      </Box>
    </Box>
  );
}
