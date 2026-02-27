/**
 * CommandPalette - Quick command search and execution (Ctrl+K)
 * Issue #1098: Implement command palette for quick navigation
 */

import React, { useState, useMemo, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import { searchCommands, getAllCommands, groupCommandsByCategory, type BcCommand } from '../types/commands';
import { UI_ELEMENTS } from '../constants';

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
  /** #1603: Show results grouped by category */
  showCategories?: boolean;
}

/** #1603: Render item in command list - can be command or category header */
interface RenderItem {
  type: 'command' | 'category';
  command?: BcCommand;
  category?: string;
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
  showCategories = true,
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

  // #1603: Build render items with category headers when showing categories
  const renderItems = useMemo((): RenderItem[] => {
    if (!showCategories || query.trim()) {
      // Don't show categories when searching
      return results.map(cmd => ({ type: 'command', command: cmd }));
    }

    // Group by category
    const grouped = groupCommandsByCategory(results);
    const items: RenderItem[] = [];

    for (const [category, commands] of grouped) {
      items.push({ type: 'category', category });
      for (const command of commands) {
        items.push({ type: 'command', command });
      }
    }

    return items;
  }, [results, showCategories, query]);

  // Get selectable indices (only commands, not category headers)
  const selectableIndices = useMemo(() => {
    return renderItems
      .map((item, idx) => (item.type === 'command' ? idx : -1))
      .filter(idx => idx >= 0);
  }, [renderItems]);

  // Map selectedIndex to actual render index
  const selectedRenderIndex = selectableIndices[selectedIndex] ?? 0;

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

      // Navigate selectable items (skip category headers)
      const maxIdx = selectableIndices.length - 1;
      if (key.upArrow) {
        setSelectedIndex(i => (i > 0 ? i - 1 : maxIdx));
        return;
      }
      if (key.downArrow) {
        setSelectedIndex(i => (i < maxIdx ? i + 1 : 0));
        return;
      }

      // Select command
      const selectedItem = renderItems[selectedRenderIndex] as RenderItem | undefined;
      if (key.return && selectedItem && selectedItem.type === 'command' && selectedItem.command) {
        const selected = selectedItem.command;
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
    [isOpen, onClose, selectableIndices, selectedRenderIndex, renderItems, onSelect, handleQueryChange, query]
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
        <Text dimColor>{'─'.repeat(UI_ELEMENTS.DIVIDER_WIDTH)}</Text>
      </Box>

      {/* Results */}
      {renderItems.length === 0 ? (
        <Box>
          <Text dimColor>No commands found</Text>
        </Box>
      ) : (
        renderItems.map((item, index) => {
          if (item.type === 'category') {
            return (
              <CategoryHeader key={`cat-${item.category ?? ''}`} name={item.category ?? ''} />
            );
          }
          const cmd = item.command;
          if (!cmd) return null;
          return (
            <CommandRow
              key={cmd.name}
              command={cmd}
              isSelected={index === selectedRenderIndex}
              isRecent={recentCommands.includes(cmd.name)}
            />
          );
        })
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

/** #1603: Category header for grouped results */
interface CategoryHeaderProps {
  name: string;
}

const CategoryHeader = React.memo(function CategoryHeader({ name }: CategoryHeaderProps): React.ReactElement {
  return (
    <Box marginTop={1}>
      <Text dimColor bold>{name}</Text>
    </Box>
  );
});

interface CommandRowProps {
  command: BcCommand;
  isSelected: boolean;
  isRecent: boolean;
}

/** #1596: Memoized command row to prevent re-renders when unrelated state changes */
/** #1603: Show keyboard shortcuts next to command name */
const CommandRow = React.memo(function CommandRow({ command, isSelected, isRecent }: CommandRowProps): React.ReactElement {
  // #1603: Truncate description to fit with shortcut
  const maxDescLen = command.shortcut ? 28 : 35;
  const desc = command.description.length > maxDescLen
    ? command.description.slice(0, maxDescLen - 1) + '…'
    : command.description;

  return (
    <Box>
      <Text
        color={isSelected ? 'cyan' : undefined}
        bold={isSelected}
        inverse={isSelected}
      >
        {isSelected ? '> ' : '  '}
        {isRecent && <Text color="yellow">★ </Text>}
        <Text bold={isSelected}>{command.name}</Text>
        {command.shortcut && (
          <Text color="magenta" dimColor={!isSelected}> [{command.shortcut}]</Text>
        )}
        <Text dimColor> — {desc}</Text>
      </Text>
    </Box>
  );
});

export default CommandPalette;
