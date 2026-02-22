/**
 * CommandPalette - Quick command search and execution (Ctrl+K)
 * Issue #1098: Implement command palette for quick navigation
 */

import React, { useState, useMemo, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import { searchCommands, getAllCommands, type BcCommand } from '../types/commands';

export interface CommandPaletteProps {
  /** Whether the palette is visible */
  isOpen: boolean;
  /** Called when palette should close */
  onClose: () => void;
  /** Called when a command is selected */
  onSelect?: (command: BcCommand) => void;
  /** Recently used commands (stored externally) */
  recentCommands?: string[];
  /** Maximum results to show */
  maxResults?: number;
  /** Disable input handling (for testing) */
  disableInput?: boolean;
}

/**
 * Command palette with fuzzy search
 */
export function CommandPalette({
  isOpen,
  onClose,
  onSelect,
  recentCommands = [],
  maxResults = 8,
  disableInput = false,
}: CommandPaletteProps): React.ReactElement | null {
  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);

  // Get filtered commands based on query
  const results = useMemo(() => {
    if (!query.trim()) {
      // Show recent commands first, then all commands
      const allCommands = getAllCommands();
      const recent = recentCommands
        .map(name => allCommands.find(c => c.name === name))
        .filter((c): c is BcCommand => c !== undefined);

      // Dedupe and limit
      const seen = new Set(recent.map(c => c.name));
      const rest = allCommands.filter(c => !seen.has(c.name));
      return [...recent, ...rest].slice(0, maxResults);
    }
    return searchCommands(query).slice(0, maxResults);
  }, [query, recentCommands, maxResults]);

  // Reset selection when results change
  const handleQueryChange = useCallback((newQuery: string) => {
    setQuery(newQuery);
    setSelectedIndex(0);
  }, []);

  // #1596: Memoize keyboard input handler
  const handleKeyboardInput = useCallback(
    (input: string, key: { escape: boolean; upArrow: boolean; downArrow: boolean; return: boolean; backspace: boolean; delete: boolean; ctrl: boolean; meta: boolean }) => {
      if (!isOpen) return;

      // Close on Escape
      if (key.escape) {
        setQuery('');
        setSelectedIndex(0);
        onClose();
        return;
      }

      // Navigate results
      if (key.upArrow) {
        setSelectedIndex(i => (i > 0 ? i - 1 : results.length - 1));
        return;
      }
      if (key.downArrow) {
        setSelectedIndex(i => (i < results.length - 1 ? i + 1 : 0));
        return;
      }

      // Select command
      if (key.return && results[selectedIndex]) {
        const selected = results[selectedIndex];
        setQuery('');
        setSelectedIndex(0);
        onSelect?.(selected);
        onClose();
        return;
      }

      // Backspace
      if (key.backspace || key.delete) {
        handleQueryChange(query.slice(0, -1));
        return;
      }

      // Type character
      if (input && !key.ctrl && !key.meta) {
        handleQueryChange(query + input);
      }
    },
    [isOpen, onClose, results, selectedIndex, onSelect, handleQueryChange, query]
  );

  // Handle keyboard input
  useInput(handleKeyboardInput, { isActive: isOpen && !disableInput });

  if (!isOpen) {
    return null;
  }

  return (
    <Box
      flexDirection="column"
      borderStyle="round"
      borderColor="cyan"
      paddingX={1}
      paddingY={0}
    >
      {/* Search input */}
      <Box>
        <Text color="cyan" bold>{'> '}</Text>
        <Text>{query}</Text>
        <Text color="cyan">|</Text>
      </Box>

      {/* Divider */}
      <Box>
        <Text dimColor>{'─'.repeat(40)}</Text>
      </Box>

      {/* Results */}
      {results.length === 0 ? (
        <Box>
          <Text dimColor>No commands found</Text>
        </Box>
      ) : (
        results.map((cmd, index) => (
          <CommandRow
            key={cmd.name}
            command={cmd}
            isSelected={index === selectedIndex}
            isRecent={recentCommands.includes(cmd.name)}
          />
        ))
      )}

      {/* Footer hints */}
      <Box marginTop={1}>
        <Text dimColor>
          ↑/↓: navigate  Enter: select  Esc: close
        </Text>
      </Box>
    </Box>
  );
}

interface CommandRowProps {
  command: BcCommand;
  isSelected: boolean;
  isRecent: boolean;
}

/** #1596: Memoized command row to prevent re-renders when unrelated state changes */
const CommandRow = React.memo(function CommandRow({ command, isSelected, isRecent }: CommandRowProps): React.ReactElement {
  return (
    <Box>
      <Text
        color={isSelected ? 'cyan' : undefined}
        bold={isSelected}
        inverse={isSelected}
      >
        {isSelected ? '> ' : '  '}
        {isRecent && <Text color="yellow">* </Text>}
        <Text bold={isSelected}>{command.name}</Text>
        <Text dimColor> - {command.description.slice(0, 35)}</Text>
      </Text>
    </Box>
  );
});

export default CommandPalette;
