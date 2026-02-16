/**
 * CommandsView - Browse and search all bc commands
 * Displays commands organized by category with search/filter capability
 * Supports execution of read-only commands directly from TUI
 */

import React, { useState, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import { COMMAND_REGISTRY, searchCommands } from '../types/commands';
import type { BcCommand } from '../types/commands';
import { useFocus } from '../navigation/FocusContext';
import { execBc } from '../services/bc';

interface CommandsViewProps {
  onBack?: () => void;
  disableInput?: boolean;
}

export const CommandsView: React.FC<CommandsViewProps> = ({
  onBack,
  disableInput = false,
}) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [searchMode, setSearchMode] = useState(false);
  const { setFocus, returnFocus } = useFocus();

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

  // Get filtered commands
  const filteredCommands = searchQuery.length > 0
    ? searchCommands(searchQuery)
    : COMMAND_REGISTRY.flatMap(cat => cat.commands);

  // Clamp selectedIndex to valid range whenever filteredCommands changes
  const validatedIndex = Math.min(selectedIndex, Math.max(0, filteredCommands.length - 1));
  const selectedCommand = filteredCommands[validatedIndex] as typeof filteredCommands[number] | undefined;

  // Reset selection when search results change
  React.useEffect(() => {
    setSelectedIndex(0);
  }, [searchQuery]);

  // Sync focus state with search mode
  React.useEffect(() => {
    if (searchMode) {
      setFocus('input');
    } else {
      returnFocus();
    }
  }, [searchMode, setFocus, returnFocus]);

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
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- defensive check for empty list
      } else if (key.return && selectedCommand) {
        // Execute read-only commands, show warning for others
        if (selectedCommand.readOnly) {
          void executeCommand(selectedCommand);
        } else {
          setCommandError(`"${selectedCommand.name}" modifies state - use CLI directly`);
          setCommandOutput(null);
        }
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
          onBack?.();
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
            <Text bold color={commandError ? 'red' : 'green'}>
              {isExecuting ? '⟳ Running' : commandError ? '✗ Error' : '✓ Output'}
            </Text>
            {lastExecutedCommand && (
              <Text dimColor> — {lastExecutedCommand}</Text>
            )}
          </Box>
          {isExecuting ? (
            <Text dimColor>Executing command...</Text>
          ) : commandError ? (
            <Text color="red">{commandError}</Text>
          ) : commandOutput ? (
            <Box flexDirection="column">
              {commandOutput.split('\n').slice(0, 15).map((line, idx) => (
                <Text key={idx} dimColor>{line}</Text>
              ))}
              {commandOutput.split('\n').length > 15 && (
                <Text dimColor>... ({commandOutput.split('\n').length - 15} more lines)</Text>
              )}
            </Box>
          ) : null}
          <Box marginTop={1}>
            <Text dimColor>Press c to clear, Esc to close</Text>
          </Box>
        </Box>
      )}

      {/* Command preview */}
      {selectedCommand !== undefined && filteredCommands.length > 0 && !commandOutput && !commandError && !isExecuting && (
        <Box flexDirection="column" marginBottom={1} paddingX={1} borderStyle="single" borderColor="gray">
          <Text bold color="cyan">{selectedCommand.name}</Text>
          <Text dimColor>{selectedCommand.description}</Text>
          <Box marginTop={1}>
            <Text dimColor>Usage: {selectedCommand.usage}</Text>
          </Box>
          {selectedCommand.flags && (
            <Text dimColor>Flags: {selectedCommand.flags.join(', ')}</Text>
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
            : 'j/k: navigate | /: search | Enter: run | q: back'}
        </Text>
      </Box>
    </Box>
  );
};

interface CommandRowProps {
  command: BcCommand;
  selected: boolean;
}

function CommandRow({ command, selected }: CommandRowProps): React.ReactElement {
  return (
    <Box marginBottom={1}>
      <Text color={selected ? 'cyan' : undefined} bold={selected}>
        {selected ? '▸ ' : '  '}
        {command.name}
      </Text>
      <Text dimColor> — {command.description}</Text>
    </Box>
  );
}

export default CommandsView;
