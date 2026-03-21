/**
 * MentionAutocomplete - Dropdown for @mention suggestions
 *
 * Shows filtered list of agent names when typing @
 */

import React from 'react';
import { Box, Text } from 'ink';
import { useTheme } from '../theme';

/** Suggestion item for @mention autocomplete */
export interface MentionSuggestion {
  name: string;
  role?: string;
  state?: string;
}

export interface MentionAutocompleteProps {
  /** List of filtered suggestions */
  suggestions: MentionSuggestion[];
  /** Currently selected index */
  selectedIndex: number;
  /** Whether the autocomplete is visible */
  visible: boolean;
  /** The query being typed (for highlighting) */
  query?: string;
}

/**
 * Get color for agent state
 */
function getStateColor(state?: string): string | undefined {
  switch (state) {
    case 'working':
      return 'green';
    case 'idle':
      return 'yellow';
    case 'stuck':
      return 'red';
    case 'error':
      return 'red';
    default:
      return undefined;
  }
}

/**
 * Get icon for role
 */
function getRoleIcon(role?: string): string {
  switch (role) {
    case 'broadcast':
      return '@';
    case 'root':
      return '#';
    case 'manager':
      return '*';
    case 'tech-lead':
      return '+';
    case 'engineer':
      return '-';
    default:
      return ' ';
  }
}

export const MentionAutocomplete: React.FC<MentionAutocompleteProps> = ({
  suggestions,
  selectedIndex,
  visible,
  query = '',
}) => {
  const { theme } = useTheme();

  if (!visible || suggestions.length === 0) {
    return null;
  }

  return (
    <Box
      flexDirection="column"
      borderStyle="single"
      borderColor={theme.colors.primary}
      paddingX={1}
      marginBottom={1}
    >
      <Box marginBottom={1}>
        <Text color={theme.colors.primary} bold>
          @{query}
        </Text>
        <Text color={theme.colors.textMuted}> - Tab to complete</Text>
      </Box>

      {suggestions.map((suggestion, index) => {
        const isSelected = index === selectedIndex;
        const icon = getRoleIcon(suggestion.role);
        const stateColor = getStateColor(suggestion.state);

        return (
          <Box key={suggestion.name}>
            <Text
              color={isSelected ? theme.colors.primary : undefined}
              bold={isSelected}
              inverse={isSelected}
            >
              {isSelected ? '> ' : '  '}
              <Text>{icon} </Text>
              <Text bold={isSelected}>@{suggestion.name}</Text>
              {suggestion.role && suggestion.role !== 'broadcast' && (
                <Text color={theme.colors.textMuted}> ({suggestion.role})</Text>
              )}
              {suggestion.state && (
                <Text color={stateColor}> [{suggestion.state}]</Text>
              )}
            </Text>
          </Box>
        );
      })}

      <Box marginTop={1}>
        <Text color={theme.colors.textMuted} dimColor>
          ↑/↓: select | Tab: complete | Esc: close
        </Text>
      </Box>
    </Box>
  );
};

export default MentionAutocomplete;
