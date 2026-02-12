/**
 * useMentionAutocomplete - Hook for @mention autocomplete
 *
 * Provides:
 * - Detection of @ trigger in input
 * - Filtered list of matching agent names
 * - Navigation through suggestions
 */

import { useState, useEffect, useMemo, useCallback } from 'react';
import { useAgents } from './useAgents';

export interface MentionSuggestion {
  name: string;
  role?: string;
  state?: string;
}

export interface UseMentionAutocompleteOptions {
  /** Current input text */
  input: string;
  /** Cursor position in input */
  cursorPosition?: number;
  /** Maximum suggestions to show */
  maxSuggestions?: number;
}

export interface UseMentionAutocompleteResult {
  /** Whether autocomplete should be shown */
  isActive: boolean;
  /** Filtered suggestions */
  suggestions: MentionSuggestion[];
  /** Currently selected suggestion index */
  selectedIndex: number;
  /** The partial mention being typed (without @) */
  query: string;
  /** Start position of the mention in input */
  mentionStart: number;
  /** Move selection up */
  moveUp: () => void;
  /** Move selection down */
  moveDown: () => void;
  /** Get selected suggestion */
  getSelected: () => MentionSuggestion | null;
  /** Complete the mention with selected suggestion */
  complete: () => string;
  /** Reset autocomplete state */
  reset: () => void;
}

/**
 * Extract the partial mention being typed at cursor position
 */
function extractMentionQuery(
  input: string,
  cursorPosition: number
): { query: string; start: number } | null {
  // Look backwards from cursor for @
  const textBeforeCursor = input.slice(0, cursorPosition);
  const lastAtIndex = textBeforeCursor.lastIndexOf('@');

  if (lastAtIndex === -1) {
    return null;
  }

  // Check if there's a space between @ and cursor (mention complete)
  const textAfterAt = textBeforeCursor.slice(lastAtIndex + 1);
  if (textAfterAt.includes(' ')) {
    return null;
  }

  // @ at start or after space
  if (lastAtIndex === 0 || input[lastAtIndex - 1] === ' ') {
    return {
      query: textAfterAt.toLowerCase(),
      start: lastAtIndex,
    };
  }

  return null;
}

export function useMentionAutocomplete(
  options: UseMentionAutocompleteOptions
): UseMentionAutocompleteResult {
  const { input, cursorPosition = input.length, maxSuggestions = 5 } = options;

  const { data: agents } = useAgents();
  const [selectedIndex, setSelectedIndex] = useState(0);

  // Extract mention query from input
  const mentionData = useMemo(
    () => extractMentionQuery(input, cursorPosition),
    [input, cursorPosition]
  );

  // Get all available names (agents + special mentions)
  const allNames = useMemo(() => {
    const names: MentionSuggestion[] = [
      { name: 'all', role: 'broadcast' },
      { name: 'everyone', role: 'broadcast' },
    ];

    if (agents) {
      for (const agent of agents) {
        names.push({
          name: agent.name,
          role: agent.role,
          state: agent.state,
        });
      }
    }

    return names;
  }, [agents]);

  // Filter suggestions based on query
  const suggestions = useMemo(() => {
    if (!mentionData) return [];

    const { query } = mentionData;
    if (query === '') {
      return allNames.slice(0, maxSuggestions);
    }

    return allNames
      .filter((s) => s.name.toLowerCase().startsWith(query))
      .slice(0, maxSuggestions);
  }, [mentionData, allNames, maxSuggestions]);

  // Reset selection when suggestions change
  useEffect(() => {
    setSelectedIndex(0);
  }, [suggestions.length]);

  const moveUp = useCallback(() => {
    setSelectedIndex((i) => (i > 0 ? i - 1 : suggestions.length - 1));
  }, [suggestions.length]);

  const moveDown = useCallback(() => {
    setSelectedIndex((i) => (i < suggestions.length - 1 ? i + 1 : 0));
  }, [suggestions.length]);

  const getSelected = useCallback((): MentionSuggestion | null => {
    return suggestions[selectedIndex] || null;
  }, [suggestions, selectedIndex]);

  const complete = useCallback((): string => {
    if (!mentionData || suggestions.length === 0) {
      return input;
    }

    const selected = suggestions[selectedIndex];
    if (!selected) return input;

    const { start } = mentionData;
    const before = input.slice(0, start);
    const after = input.slice(cursorPosition);

    return `${before}@${selected.name} ${after}`;
  }, [mentionData, suggestions, selectedIndex, input, cursorPosition]);

  const reset = useCallback(() => {
    setSelectedIndex(0);
  }, []);

  return {
    isActive: mentionData !== null && suggestions.length > 0,
    suggestions,
    selectedIndex,
    query: mentionData?.query || '',
    mentionStart: mentionData?.start || 0,
    moveUp,
    moveDown,
    getSelected,
    complete,
    reset,
  };
}

export default useMentionAutocomplete;
