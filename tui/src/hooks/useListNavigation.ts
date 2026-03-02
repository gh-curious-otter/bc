/**
 * useListNavigation - Hook for vim-style list navigation
 *
 * Provides keyboard navigation for lists with:
 * - j/↓: Move selection down
 * - k/↑: Move selection up
 * - g/Home: Jump to first item
 * - G/End: Jump to last item
 * - Enter/Space: Select current item
 * - /: Enter search mode (optional)
 * - Escape: Exit search mode or trigger onBack
 *
 * Enhanced for #1586: Consolidated keyboard navigation patterns
 */

import { useState, useCallback, useMemo } from 'react';
import { useInput } from 'ink';

export interface SearchState {
  /** Whether search mode is active */
  isActive: boolean;
  /** Current search query */
  query: string;
}

export interface UseListNavigationOptions<T> {
  /** The list of items to navigate */
  items: T[];
  /** Override item count for navigation boundaries (e.g., when visible list includes extra items like group headers) */
  itemCount?: number;
  /** Callback when an item is selected (Enter/Space) */
  onSelect?: (item: T, index: number) => void;
  /** Callback when Escape is pressed (and not in search mode) */
  onBack?: () => void;
  /** Initial selected index (defaults to 0) */
  initialIndex?: number;
  /** Disable keyboard handling */
  disabled?: boolean;
  /** Wrap around when reaching start/end of list */
  wrap?: boolean;
  /** Enable search mode with / key */
  enableSearch?: boolean;
  /** Callback when search query changes */
  onSearchChange?: (query: string) => void;
  /** Custom key handlers (processed after built-in handlers) */
  customKeys?: Record<string, () => void>;
  /** Additional condition for disabling input (e.g., when modal is open) */
  isActive?: boolean;
}

export interface UseListNavigationResult<T> {
  /** Currently selected index */
  selectedIndex: number;
  /** Currently selected item (or undefined if list is empty) */
  selectedItem: T | undefined;
  /** Set the selected index directly */
  setSelectedIndex: (index: number) => void;
  /** Move selection down by n items (default 1) */
  moveDown: (n?: number) => void;
  /** Move selection up by n items (default 1) */
  moveUp: (n?: number) => void;
  /** Jump to first item */
  jumpToFirst: () => void;
  /** Jump to last item */
  jumpToLast: () => void;
  /** Check if an index is the selected one */
  isSelected: (index: number) => boolean;
  /** Search state (if enableSearch is true) */
  search: SearchState;
  /** Enter search mode */
  enterSearchMode: () => void;
  /** Exit search mode */
  exitSearchMode: () => void;
  /** Clear search query */
  clearSearch: () => void;
  /** Set search query directly */
  setSearchQuery: (query: string) => void;
}

export function useListNavigation<T>(
  options: UseListNavigationOptions<T>
): UseListNavigationResult<T> {
  const {
    items,
    itemCount,
    onSelect,
    onBack,
    initialIndex = 0,
    disabled = false,
    wrap = false,
    enableSearch = false,
    onSearchChange,
    customKeys = {},
    isActive = true,
  } = options;

  // Use itemCount override when provided (e.g., visibleItems.length in grouped view)
  const navLength = itemCount !== undefined ? itemCount : items.length;

  const [selectedIndex, setSelectedIndex] = useState(() =>
    Math.min(Math.max(0, initialIndex), Math.max(0, navLength - 1))
  );

  const [searchMode, setSearchMode] = useState(false);
  const [searchQuery, setSearchQueryState] = useState('');

  const clampIndex = useCallback(
    (index: number): number => {
      if (navLength === 0) return 0;
      if (wrap) {
        // Wrap around
        if (index < 0) return navLength - 1;
        if (index >= navLength) return 0;
        return index;
      }
      // Clamp to valid range
      return Math.min(Math.max(0, index), navLength - 1);
    },
    [navLength, wrap]
  );

  const moveDown = useCallback(
    (n = 1) => {
      setSelectedIndex((prev) => clampIndex(prev + n));
    },
    [clampIndex]
  );

  const moveUp = useCallback(
    (n = 1) => {
      setSelectedIndex((prev) => clampIndex(prev - n));
    },
    [clampIndex]
  );

  const jumpToFirst = useCallback(() => {
    setSelectedIndex(0);
  }, []);

  const jumpToLast = useCallback(() => {
    setSelectedIndex(Math.max(0, navLength - 1));
  }, [navLength]);

  const isSelectedFn = useCallback(
    (index: number): boolean => index === selectedIndex,
    [selectedIndex]
  );

  const selectedItem = useMemo(
    () => (items.length > 0 ? items[selectedIndex] : undefined),
    [items, selectedIndex]
  );

  // Search mode functions
  const enterSearchMode = useCallback(() => {
    if (enableSearch) {
      setSearchMode(true);
    }
  }, [enableSearch]);

  const exitSearchMode = useCallback(() => {
    setSearchMode(false);
  }, []);

  const clearSearch = useCallback(() => {
    setSearchQueryState('');
    onSearchChange?.('');
  }, [onSearchChange]);

  const setSearchQuery = useCallback(
    (query: string) => {
      setSearchQueryState(query);
      onSearchChange?.(query);
    },
    [onSearchChange]
  );

  // Handle keyboard input
  useInput(
    (input, key) => {
      // Search mode input handling
      if (searchMode) {
        if (key.return || key.escape) {
          exitSearchMode();
          return;
        }
        if (key.backspace || key.delete) {
          const newQuery = searchQuery.slice(0, -1);
          setSearchQuery(newQuery);
          return;
        }
        // Append printable character to search
        if (input && !key.ctrl && !key.meta && !key.tab) {
          setSearchQuery(searchQuery + input);
        }
        return;
      }

      // j or down arrow: move down
      if (input === 'j' || key.downArrow) {
        moveDown();
        return;
      }

      // k or up arrow: move up
      if (input === 'k' || key.upArrow) {
        moveUp();
        return;
      }

      // g: jump to first
      if (input === 'g') {
        jumpToFirst();
        return;
      }

      // G (shift+g): jump to last
      if (input === 'G') {
        jumpToLast();
        return;
      }

      // Enter or Space: select current item
      if (key.return || input === ' ') {
        if (onSelect && selectedItem !== undefined) {
          onSelect(selectedItem, selectedIndex);
        }
        return;
      }

      // /: enter search mode
      if (input === '/' && enableSearch) {
        enterSearchMode();
        return;
      }

      // c: clear search (when search is active but not in search mode)
      if (input === 'c' && enableSearch && searchQuery) {
        clearSearch();
        return;
      }

      // Escape: back navigation (when not in search mode)
      if (key.escape) {
        onBack?.();
        return;
      }

      // Process custom key handlers
      if (input && input in customKeys) {
        customKeys[input]();
        return;
      }
    },
    { isActive: !disabled && isActive && navLength > 0 }
  );

  return {
    selectedIndex,
    selectedItem,
    setSelectedIndex: (index) => {
      setSelectedIndex(clampIndex(index));
    },
    moveDown,
    moveUp,
    jumpToFirst,
    jumpToLast,
    isSelected: isSelectedFn,
    search: {
      isActive: searchMode,
      query: searchQuery,
    },
    enterSearchMode,
    exitSearchMode,
    clearSearch,
    setSearchQuery,
  };
}

export default useListNavigation;
