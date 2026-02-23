/**
 * CommandsView - Browse and search all bc commands
 * Displays commands organized by category with search/filter capability
 * Supports execution of read-only commands directly from TUI
 * Supports favorites with persistence
 * Issue #1727: Migrated to useListNavigation hook
 */

import React, { useState, useCallback, useEffect, useMemo } from 'react';
import { Box, Text, useInput, useStdout } from 'ink';
import { COMMAND_REGISTRY } from '../types/commands';
import type { BcCommand } from '../types/commands';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { useDisableInput, useListNavigation } from '../hooks';
import { execBc } from '../services/bc';
import * as fs from 'fs';
import * as path from 'path';
import * as os from 'os';

// Favorites storage path
const FAVORITES_FILE = path.join(os.homedir(), '.bc', 'command-favorites.json');

/**
 * Load favorites from disk
 */
function loadFavorites(): Set<string> {
  try {
    if (fs.existsSync(FAVORITES_FILE)) {
      const data = fs.readFileSync(FAVORITES_FILE, 'utf-8');
      const parsed = JSON.parse(data) as string[];
      return new Set(parsed);
    }
  } catch {
    // Ignore errors, return empty set
  }
  return new Set();
}

/**
 * Save favorites to disk
 */
function saveFavorites(favorites: Set<string>): void {
  try {
    const dir = path.dirname(FAVORITES_FILE);
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }
    fs.writeFileSync(FAVORITES_FILE, JSON.stringify([...favorites], null, 2));
  } catch {
    // Ignore save errors
  }
}

// #1594: Using empty interface for future extensibility, props removed
// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface CommandsViewProps {}

// Get all category names from registry
const CATEGORY_NAMES = ['All', ...COMMAND_REGISTRY.map(cat => cat.name)];

export const CommandsView: React.FC<CommandsViewProps> = (_props = {}) => {
  // #1594: Use context instead of prop drilling
  const { isDisabled: disableInput } = useDisableInput();
  const [searchQuery, setSearchQuery] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('All');
  const { setFocus } = useFocus();
  const { goHome } = useNavigation();
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;
  const terminalHeight = stdout.rows;

  // #1460: Calculate visible command count to prevent overflow
  // Reserve space for: header(2) + category(2) + search(3) + preview(8) + footer(2) = 17 lines
  const visibleCommandCount = Math.max(3, terminalHeight - 17);

  // Favorites state - persisted to disk
  const [favorites, setFavorites] = useState<Set<string>>(() => loadFavorites());

  // Save favorites when they change
  useEffect(() => {
    saveFavorites(favorites);
  }, [favorites]);

  // Toggle favorite for a command
  const toggleFavorite = useCallback((commandName: string) => {
    setFavorites(prev => {
      const next = new Set(prev);
      if (next.has(commandName)) {
        next.delete(commandName);
      } else {
        next.add(commandName);
      }
      return next;
    });
  }, []);

  // Command execution state
  const [isExecuting, setIsExecuting] = useState(false);
  const [commandOutput, setCommandOutput] = useState<string | null>(null);
  const [commandError, setCommandError] = useState<string | null>(null);
  const [lastExecutedCommand, setLastExecutedCommand] = useState<string | null>(null);

  /**
   * Execute a read-only bc command and capture output
   */
  const executeCommand = useCallback(async (command: BcCommand) => {
    if (!command.readOnly) {
      setCommandError('Only read-only commands can be executed from TUI');
      return;
    }

    setIsExecuting(true);
    setCommandOutput(null);
    setCommandError(null);
    setLastExecutedCommand(command.name);

    try {
      // Parse command name into args (e.g., "agent list" -> ["agent", "list"])
      const args = command.name.split(' ');
      const output = await execBc(args);
      setCommandOutput(output);
    } catch (err) {
      setCommandError(err instanceof Error ? err.message : 'Command failed');
    } finally {
      setIsExecuting(false);
    }
  }, []);

  // Get filtered commands by category and search, with favorites first
  const filteredCommands = useMemo(() => {
    let commands = categoryFilter === 'All'
      ? COMMAND_REGISTRY.flatMap(cat => cat.commands)
      : COMMAND_REGISTRY.find(cat => cat.name === categoryFilter)?.commands ?? [];

    if (searchQuery.length > 0) {
      const lowerQuery = searchQuery.toLowerCase();
      commands = commands.filter(cmd =>
        cmd.name.toLowerCase().includes(lowerQuery) ||
        cmd.description.toLowerCase().includes(lowerQuery)
      );
    }

    // Sort favorites to the top
    return [...commands].sort((a, b) => {
      const aFav = favorites.has(a.name) ? 0 : 1;
      const bFav = favorites.has(b.name) ? 0 : 1;
      return aFav - bFav;
    });
  }, [categoryFilter, searchQuery, favorites]);

  // Count favorites for display
  const favoriteCount = favorites.size;

  // Callbacks for list navigation
  const handleSelect = useCallback((command: BcCommand) => {
    // Execute read-only commands, show warning for others
    if (command.readOnly) {
      void executeCommand(command);
    } else {
      setCommandError(`"${command.name}" modifies state - use CLI directly`);
      setCommandOutput(null);
    }
  }, [executeCommand]);

  const handleCycleCategory = useCallback(() => {
    const currentIdx = CATEGORY_NAMES.indexOf(categoryFilter);
    const nextIdx = (currentIdx + 1) % CATEGORY_NAMES.length;
    setCategoryFilter(CATEGORY_NAMES[nextIdx] ?? 'All');
  }, [categoryFilter]);

  const handleClearOutput = useCallback(() => {
    if (commandOutput !== null || commandError !== null) {
      setCommandOutput(null);
      setCommandError(null);
      setLastExecutedCommand(null);
    }
  }, [commandOutput, commandError]);

  // #1727: Use useListNavigation hook for vim-style navigation
  const {
    selectedIndex,
    selectedItem: selectedCommand,
    search,
    setSelectedIndex,
  } = useListNavigation({
    items: filteredCommands,
    enableSearch: true,
    onSearchChange: setSearchQuery,
    onSelect: handleSelect,
    onBack: () => {
      if (commandOutput !== null || commandError !== null) {
        // First press clears output
        handleClearOutput();
      } else {
        // Navigate to home/dashboard
        goHome();
      }
    },
    customKeys: {
      'f': () => { if (selectedCommand) toggleFavorite(selectedCommand.name); },
      'c': handleClearOutput,
    },
    isActive: !disableInput,
  });

  // Reset selection when category changes
  useEffect(() => {
    setSelectedIndex(0);
  }, [categoryFilter, setSelectedIndex]);

  // Handle Tab key for category cycling (not handled by useListNavigation)
  useInput((_input, key) => {
    if (!search.isActive && key.tab) {
      handleCycleCategory();
    }
  }, { isActive: !disableInput });

  // Sync focus state with search mode
  // Use setFocus('view') to enable local q/ESC handling via !isFocused('view') guard
  // This prevents global q-key handler from quitting while in CommandsView
  useEffect(() => {
    if (search.isActive) {
      setFocus('input');
    } else {
      setFocus('view');
    }
  }, [search.isActive, setFocus]);

  return (
    <Box flexDirection="column" width="100%">
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="cyan">
          Commands
        </Text>
        <Text dimColor> ({filteredCommands.length} available)</Text>
        {favoriteCount > 0 && (
          <Text color="yellow"> ★ {favoriteCount} favorites</Text>
        )}
      </Box>

      {/* Category filter bar */}
      <Box marginBottom={1} paddingX={1}>
        <Text dimColor>Category: </Text>
        <Text color="cyan" bold>{categoryFilter}</Text>
        <Text dimColor> (Tab to cycle)</Text>
      </Box>

      {/* Search bar */}
      <Box marginBottom={1} paddingX={1} borderStyle="single" borderColor={search.isActive ? 'cyan' : 'gray'}>
        {search.isActive ? (
          <Box>
            <Text color="cyan">{'/ '}</Text>
            <Text>{search.query}</Text>
            <Text color="cyan">▌</Text>
          </Box>
        ) : (
          <Text dimColor>Press / to search, j/k to navigate, q to back</Text>
        )}
      </Box>

      {/* Command list - #1460: Windowed to prevent overflow */}
      <Box flexDirection="column" marginBottom={1} paddingX={1}>
        {filteredCommands.length === 0 ? (
          <Box flexDirection="column">
            <Text dimColor>No commands match &quot;{search.query}&quot;</Text>
            {search.query.length > 0 && (
              <Box marginTop={1}>
                <Text dimColor>Try a different search or press Esc to clear</Text>
              </Box>
            )}
          </Box>
        ) : (
          (() => {
            // #1460: Window the visible commands around selection
            const start = Math.max(0, Math.min(
              selectedIndex - Math.floor(visibleCommandCount / 2),
              filteredCommands.length - visibleCommandCount
            ));
            const visibleCommands = filteredCommands.slice(start, start + visibleCommandCount);

            return (
              <>
                {start > 0 && <Text dimColor>↑ {start} more above</Text>}
                {visibleCommands.map((cmd, idx) => (
                  <CommandRow
                    key={`${cmd.category}-${cmd.name}`}
                    command={cmd}
                    selected={start + idx === selectedIndex}
                    isFavorite={favorites.has(cmd.name)}
                  />
                ))}
                {start + visibleCommandCount < filteredCommands.length && (
                  <Text dimColor>↓ {filteredCommands.length - start - visibleCommandCount} more below</Text>
                )}
              </>
            );
          })()
        )}
      </Box>

      {/* Command output panel */}
      {(isExecuting || commandOutput !== null || commandError !== null) && (
        <Box
          flexDirection="column"
          marginBottom={1}
          paddingX={1}
          borderStyle="single"
          borderColor={commandError ? 'red' : 'green'}
        >
          <Box marginBottom={1}>
            <Text bold color={commandError ? 'red' : 'green'} wrap="truncate">
              {isExecuting ? '⟳ Running' : commandError ? '✗ Error' : '✓ Output'}
            </Text>
            {lastExecutedCommand && (
              <Text dimColor wrap="truncate"> — {lastExecutedCommand}</Text>
            )}
          </Box>
          {isExecuting ? (
            <Text dimColor wrap="truncate">Executing command...</Text>
          ) : commandError ? (
            <Text color="red" wrap="truncate">{commandError}</Text>
          ) : commandOutput ? (
            <Box flexDirection="column">
              {commandOutput.split('\n').slice(0, 15).map((line, idx) => (
                <Text key={idx} dimColor wrap="truncate">{line}</Text>
              ))}
              {commandOutput.split('\n').length > 15 && (
                <Text dimColor wrap="truncate">... ({commandOutput.split('\n').length - 15} more lines)</Text>
              )}
            </Box>
          ) : null}
          <Box marginTop={1}>
            <Text dimColor>Press c to clear, Esc to close</Text>
          </Box>
        </Box>
      )}

      {/* Command preview - #1366: Dynamic width constraint + wrap=truncate to prevent text corruption */}
      {selectedCommand !== undefined && filteredCommands.length > 0 && !commandOutput && !commandError && !isExecuting && (
        <Box flexDirection="column" marginBottom={1} paddingX={1} borderStyle="single" borderColor="gray" width={Math.min(terminalWidth - 4, 100)}>
          <Text bold color="cyan" wrap="truncate">{selectedCommand.name}</Text>
          <Text dimColor wrap="truncate">{selectedCommand.description}</Text>
          <Box marginTop={1}>
            <Text dimColor wrap="truncate">Usage: {selectedCommand.usage}</Text>
          </Box>
          {selectedCommand.flags && (
            <Text dimColor wrap="truncate">Flags: {selectedCommand.flags.join(', ')}</Text>
          )}
          <Box marginTop={1}>
            <Text dimColor wrap="truncate">
              {selectedCommand.readOnly ? '✓ Read-only - Enter to run' : '⚠ Modifying - use CLI'}
            </Text>
          </Box>
        </Box>
      )}

      {/* Footer */}
      <Box>
        <Text dimColor>
          {search.isActive
            ? 'Type to search, Enter/Esc to exit'
            : commandOutput !== null || commandError !== null
            ? 'c: clear output | Esc: close | q: back'
            : filteredCommands.length === 0
            ? 'No commands found | /: search | q: back'
            : 'j/k: navigate | g/G: top/bottom | /: search | Enter: run | f: favorite | q/ESC: back'}
        </Text>
      </Box>
    </Box>
  );
};

interface CommandRowProps {
  command: BcCommand;
  selected: boolean;
  isFavorite: boolean;
}

function CommandRow({ command, selected, isFavorite }: CommandRowProps): React.ReactElement {
  // #1366: Explicit text slicing prevents corruption at 120x40
  // wrap='truncate' needs width constraints to work properly
  const displayName = command.name.length > 25 ? command.name.slice(0, 24) + '…' : command.name;
  const displayDesc = command.description.length > 45 ? command.description.slice(0, 44) + '…' : command.description;

  return (
    <Box marginBottom={1} flexWrap="nowrap">
      <Text color="yellow">{isFavorite ? '★ ' : '  '}</Text>
      <Text color={selected ? 'cyan' : undefined} bold={selected}>
        {selected ? '▸ ' : '  '}
        {displayName}
      </Text>
      <Text dimColor> — {displayDesc}</Text>
    </Box>
  );
}

export default CommandsView;
