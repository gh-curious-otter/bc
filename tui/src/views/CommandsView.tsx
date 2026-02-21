/**
 * CommandsView - Browse and search all bc commands
 * Displays commands organized by category with search/filter capability
 * Supports execution of read-only commands directly from TUI
 * Supports favorites with persistence
 */

import React, { useState, useCallback, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { COMMAND_REGISTRY } from '../types/commands';
import type { BcCommand } from '../types/commands';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
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

interface CommandsViewProps {
  disableInput?: boolean;
}

// Get all category names from registry
const CATEGORY_NAMES = ['All', ...COMMAND_REGISTRY.map(cat => cat.name)];

export const CommandsView: React.FC<CommandsViewProps> = ({
  disableInput = false,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [searchMode, setSearchMode] = useState(false);
  const [categoryFilter, setCategoryFilter] = useState('All');
  const { setFocus } = useFocus();
  const { goHome } = useNavigation();

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
  const filteredCommands = React.useMemo(() => {
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

  // Clamp selectedIndex to valid range whenever filteredCommands changes
  const validatedIndex = Math.min(selectedIndex, Math.max(0, filteredCommands.length - 1));
  const selectedCommand = filteredCommands[validatedIndex] as typeof filteredCommands[number] | undefined;

  // Reset selection when search results or category change
  React.useEffect(() => {
    setSelectedIndex(0);
  }, [searchQuery, categoryFilter]);

  // Sync focus state with search mode
  // Use setFocus('view') to enable local q/ESC handling via !isFocused('view') guard
  // This prevents global q-key handler from quitting while in CommandsView
  React.useEffect(() => {
    if (searchMode) {
      setFocus('input');
    } else {
      setFocus('view');
    }
  }, [searchMode, setFocus]);

  // Keyboard navigation
  useInput((input, key) => {
    if (searchMode) {
      // Search mode: handle text input
      if (key.return) {
        setSearchMode(false);
      } else if (key.escape) {
        setSearchQuery('');
        setSearchMode(false);
      } else if (key.backspace || key.delete) {
        setSearchQuery(searchQuery.slice(0, -1));
      } else if (input && !key.ctrl && !key.meta && !key.tab) {
        // Add printable characters to search query, ignore tab
        setSearchQuery(searchQuery + input);
      }
    } else {
      // Navigation mode
      if (input === '/') {
        setSearchMode(true);
      } else if (key.tab) {
        // Cycle to next category
        const currentIdx = CATEGORY_NAMES.indexOf(categoryFilter);
        const nextIdx = (currentIdx + 1) % CATEGORY_NAMES.length;
        setCategoryFilter(CATEGORY_NAMES[nextIdx] ?? 'All');
      } else if (key.upArrow || input === 'k') {
        // Navigate up, clamped to valid range
        if (filteredCommands.length > 0) {
          setSelectedIndex(Math.max(0, validatedIndex - 1));
        }
      } else if (key.downArrow || input === 'j') {
        // Navigate down, clamped to valid range
        if (filteredCommands.length > 0) {
          setSelectedIndex(Math.min(filteredCommands.length - 1, validatedIndex + 1));
        }
      } else if (input === 'g') {
        // Go to top
        setSelectedIndex(0);
      } else if (input === 'G') {
        // Go to bottom
        if (filteredCommands.length > 0) {
          setSelectedIndex(filteredCommands.length - 1);
        }
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
      } else if (key.return && selectedCommand) {
        // Execute read-only commands, show warning for others
        if (selectedCommand.readOnly) {
          void executeCommand(selectedCommand);
        } else {
          setCommandError(`"${selectedCommand.name}" modifies state - use CLI directly`);
          setCommandOutput(null);
        }
      } else if (input === 'f' && selectedCommand) {
        // Toggle favorite
        toggleFavorite(selectedCommand.name);
      } else if (input === 'c' && (commandOutput !== null || commandError !== null)) {
        // Clear output panel
        setCommandOutput(null);
        setCommandError(null);
        setLastExecutedCommand(null);
      } else if (input === 'q' || key.escape) {
        if (commandOutput !== null || commandError !== null) {
          // First press clears output, second press goes back
          setCommandOutput(null);
          setCommandError(null);
          setLastExecutedCommand(null);
        } else {
          // Navigate to home/dashboard
          goHome();
        }
      }
    }
  }, { isActive: !disableInput });

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
      <Box marginBottom={1} paddingX={1} borderStyle="single" borderColor={searchMode ? 'cyan' : 'gray'}>
        {searchMode ? (
          <Box>
            <Text color="cyan">{'/ '}</Text>
            <Text>{searchQuery}</Text>
            <Text color="cyan">▌</Text>
          </Box>
        ) : (
          <Text dimColor>Press / to search, j/k to navigate, q to back</Text>
        )}
      </Box>

      {/* Command list */}
      <Box flexDirection="column" marginBottom={1} paddingX={1}>
        {filteredCommands.length === 0 ? (
          <Box flexDirection="column">
            <Text dimColor>No commands match &quot;{searchQuery}&quot;</Text>
            {searchQuery.length > 0 && (
              <Box marginTop={1}>
                <Text dimColor>Try a different search or press Esc to clear</Text>
              </Box>
            )}
          </Box>
        ) : (
          filteredCommands.map((cmd, idx) => (
            <CommandRow
              key={`${cmd.category}-${cmd.name}`}
              command={cmd}
              selected={idx === validatedIndex}
              isFavorite={favorites.has(cmd.name)}
            />
          ))
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

      {/* Command preview - #1366: Slice strings to prevent text corruption at 120x40 */}
      {selectedCommand !== undefined && filteredCommands.length > 0 && !commandOutput && !commandError && !isExecuting && (
        <Box flexDirection="column" marginBottom={1} paddingX={1} borderStyle="single" borderColor="gray">
          <Text bold color="cyan">{selectedCommand.name}</Text>
          <Text dimColor>{selectedCommand.description.slice(0, 70)}{selectedCommand.description.length > 70 ? '…' : ''}</Text>
          <Box marginTop={1}>
            <Text dimColor>Usage: {selectedCommand.usage.slice(0, 60)}{selectedCommand.usage.length > 60 ? '…' : ''}</Text>
          </Box>
          {selectedCommand.flags && (
            <Text dimColor>Flags: {selectedCommand.flags.join(', ').slice(0, 60)}</Text>
          )}
          <Box marginTop={1}>
            <Text dimColor>
              {selectedCommand.readOnly ? '✓ Safe (read-only) - Press Enter to run' : '⚠ Modifying command - use CLI'}
            </Text>
          </Box>
        </Box>
      )}

      {/* Footer */}
      <Box>
        <Text dimColor>
          {searchMode
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
