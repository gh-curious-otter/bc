/**
 * useListNavigation - Hook for vim-style list navigation
 *
 * Provides keyboard navigation for lists with:
 * - j/↓: Move selection down
 * - k/↑: Move selection up
 * - g/Home: Jump to first item
 * - G/End: Jump to last item
 * - Enter/Space: Select current item
 */

import { useState, useCallback, useMemo } from 'react';
import { useInput } from 'ink';

export interface UseListNavigationOptions<T> {
  /** The list of items to navigate */
  items: T[];
  /** Callback when an item is selected (Enter/Space) */
  onSelect?: (item: T, index: number) => void;
  /** Initial selected index (defaults to 0) */
  initialIndex?: number;
  /** Disable keyboard handling */
  disabled?: boolean;
  /** Wrap around when reaching start/end of list */
  wrap?: boolean;
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
}

export function useListNavigation<T>(
  options: UseListNavigationOptions<T>
): UseListNavigationResult<T> {
  const {
    items,
    onSelect,
    initialIndex = 0,
    disabled = false,
    wrap = false,
  } = options;

  const [selectedIndex, setSelectedIndex] = useState(() =>
    Math.min(Math.max(0, initialIndex), Math.max(0, items.length - 1))
  );

  const clampIndex = useCallback(
    (index: number): number => {
      if (items.length === 0) return 0;
      if (wrap) {
        // Wrap around
        if (index < 0) return items.length - 1;
        if (index >= items.length) return 0;
        return index;
      }
      // Clamp to valid range
      return Math.min(Math.max(0, index), items.length - 1);
    },
    [items.length, wrap]
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
    setSelectedIndex(Math.max(0, items.length - 1));
  }, [items.length]);

  const isSelected = useCallback(
    (index: number): boolean => index === selectedIndex,
    [selectedIndex]
  );

  const selectedItem = useMemo(
    () => (items.length > 0 ? items[selectedIndex] : undefined),
    [items, selectedIndex]
  );

  // Handle keyboard input
  useInput(
    (input, key) => {
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

      // g or Home: jump to first
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
    },
    { isActive: !disabled && items.length > 0 }
  );

  return {
    selectedIndex,
    selectedItem,
    setSelectedIndex: (index) => setSelectedIndex(clampIndex(index)),
    moveDown,
    moveUp,
    jumpToFirst,
    jumpToLast,
    isSelected,
  };
}

export default useListNavigation;
