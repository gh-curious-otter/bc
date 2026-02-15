/**
 * CommandsView - Browse and search all bc commands
 * Displays commands organized by category with search/filter capability
 */

import React, { useState } from 'react';
import { Box, Text, useInput } from 'ink';
import { COMMAND_REGISTRY, searchCommands } from '../types/commands';
import type { BcCommand } from '../types/commands';
import { useFocus } from '../navigation/FocusContext';

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

  // Get filtered commands
  const filteredCommands = searchQuery.length > 0
    ? searchCommands(searchQuery)
    : COMMAND_REGISTRY.flatMap(cat => cat.commands);

  const selectedCommand = filteredCommands[selectedIndex];

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
      } else if (input && !key.ctrl && !key.meta) {
        setSearchQuery(searchQuery + input);
      }
    } else {
      // Navigation mode
      if (input === '/') {
        setSearchMode(true);
      } else if ((key.upArrow || input === 'k') && selectedIndex > 0) {
        setSelectedIndex(selectedIndex - 1);
      } else if ((key.downArrow || input === 'j') && selectedIndex < filteredCommands.length - 1) {
        setSelectedIndex(selectedIndex + 1);
      } else if (key.return && selectedCommand) {
        // TODO: Execute command or show confirmation
      } else if (input === 'q' || key.escape) {
        onBack?.();
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
          <Text dimColor>No commands match &quot;{searchQuery}&quot;</Text>
        ) : (
          filteredCommands.map((cmd, idx) => (
            <CommandRow
              key={`${cmd.category}-${cmd.name}`}
              command={cmd}
              selected={idx === selectedIndex}
            />
          ))
        )}
      </Box>

      {/* Command preview */}
      {selectedCommand != null && (
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
              {selectedCommand.readOnly ? '✓ Safe (read-only)' : '⚠ Modifying command'}
            </Text>
          </Box>
        </Box>
      )}

      {/* Footer */}
      <Box>
        <Text dimColor>
          {searchMode
            ? 'Type to search, Enter to confirm, Esc to cancel'
            : 'j/k: navigate | /: search | Enter: view | q: back'}
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
