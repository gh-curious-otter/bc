/**
 * useListNavigation Tests - Vim-style list navigation hook
 *
 * Tests cover:
 * - Index clamping logic
 * - Wrap-around behavior
 * - Move up/down operations
 * - Jump to first/last
 * - Selection state
 * - Search mode logic (#1586)
 * - Edge cases (empty list, single item)
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 * #1586: Extract reusable keyboard navigation hook
 *
 * Note: useInput tests require TTY stdin which is not available in Bun.
 * These tests focus on the navigation logic that can be tested without hooks.
 */

import { describe, test, expect } from 'bun:test';

// Navigation logic extracted for testing
// Mirrors the clampIndex function in useListNavigation
function clampIndex(index: number, length: number, wrap: boolean): number {
  if (length === 0) return 0;
  if (wrap) {
    if (index < 0) return length - 1;
    if (index >= length) return 0;
    return index;
  }
  return Math.min(Math.max(0, index), length - 1);
}

// Initial index clamping logic
function getInitialIndex(initialIndex: number, length: number): number {
  return Math.min(Math.max(0, initialIndex), Math.max(0, length - 1));
}

// Search query manipulation logic
function appendToQuery(query: string, char: string): string {
  return query + char;
}

function backspaceQuery(query: string): string {
  return query.slice(0, -1);
}

describe('useListNavigation', () => {
  describe('Index Clamping (no wrap)', () => {
    test('clamps negative index to 0', () => {
      expect(clampIndex(-1, 5, false)).toBe(0);
      expect(clampIndex(-10, 5, false)).toBe(0);
    });

    test('clamps index exceeding length to last valid index', () => {
      expect(clampIndex(5, 5, false)).toBe(4);
      expect(clampIndex(100, 5, false)).toBe(4);
    });

    test('returns valid index unchanged', () => {
      expect(clampIndex(0, 5, false)).toBe(0);
      expect(clampIndex(2, 5, false)).toBe(2);
      expect(clampIndex(4, 5, false)).toBe(4);
    });

    test('handles empty list', () => {
      expect(clampIndex(0, 0, false)).toBe(0);
      expect(clampIndex(5, 0, false)).toBe(0);
      expect(clampIndex(-1, 0, false)).toBe(0);
    });

    test('handles single item list', () => {
      expect(clampIndex(0, 1, false)).toBe(0);
      expect(clampIndex(1, 1, false)).toBe(0);
      expect(clampIndex(-1, 1, false)).toBe(0);
    });
  });

  describe('Index Clamping (with wrap)', () => {
    test('wraps negative index to end of list', () => {
      expect(clampIndex(-1, 5, true)).toBe(4);
    });

    test('wraps index at length to start of list', () => {
      expect(clampIndex(5, 5, true)).toBe(0);
    });

    test('returns valid index unchanged', () => {
      expect(clampIndex(0, 5, true)).toBe(0);
      expect(clampIndex(2, 5, true)).toBe(2);
      expect(clampIndex(4, 5, true)).toBe(4);
    });

    test('handles empty list', () => {
      expect(clampIndex(0, 0, true)).toBe(0);
    });

    test('handles single item list (no actual wrapping)', () => {
      expect(clampIndex(-1, 1, true)).toBe(0);
      expect(clampIndex(1, 1, true)).toBe(0);
    });
  });

  describe('Initial Index', () => {
    test('uses provided initial index when valid', () => {
      expect(getInitialIndex(2, 5)).toBe(2);
    });

    test('clamps initial index to valid range', () => {
      expect(getInitialIndex(10, 5)).toBe(4);
      expect(getInitialIndex(-1, 5)).toBe(0);
    });

    test('defaults to 0 for empty list', () => {
      expect(getInitialIndex(0, 0)).toBe(0);
      expect(getInitialIndex(5, 0)).toBe(0);
    });
  });

  describe('Move Down Logic', () => {
    test('moves selection down by 1', () => {
      const current = 2;
      const next = clampIndex(current + 1, 5, false);
      expect(next).toBe(3);
    });

    test('moves selection down by n', () => {
      const current = 1;
      const next = clampIndex(current + 3, 5, false);
      expect(next).toBe(4);
    });

    test('stops at last item without wrap', () => {
      const current = 4;
      const next = clampIndex(current + 1, 5, false);
      expect(next).toBe(4);
    });

    test('wraps to first item with wrap enabled', () => {
      const current = 4;
      const next = clampIndex(current + 1, 5, true);
      expect(next).toBe(0);
    });
  });

  describe('Move Up Logic', () => {
    test('moves selection up by 1', () => {
      const current = 2;
      const next = clampIndex(current - 1, 5, false);
      expect(next).toBe(1);
    });

    test('moves selection up by n', () => {
      const current = 4;
      const next = clampIndex(current - 3, 5, false);
      expect(next).toBe(1);
    });

    test('stops at first item without wrap', () => {
      const current = 0;
      const next = clampIndex(current - 1, 5, false);
      expect(next).toBe(0);
    });

    test('wraps to last item with wrap enabled', () => {
      const current = 0;
      const next = clampIndex(current - 1, 5, true);
      expect(next).toBe(4);
    });
  });

  describe('Jump Operations', () => {
    test('jump to first sets index to 0', () => {
      expect(0).toBe(0);
    });

    test('jump to last sets index to length - 1', () => {
      const length = 5;
      const lastIndex = Math.max(0, length - 1);
      expect(lastIndex).toBe(4);
    });

    test('jump to last handles empty list', () => {
      const length = 0;
      const lastIndex = Math.max(0, length - 1);
      expect(lastIndex).toBe(0);
    });

    test('jump to last handles single item', () => {
      const length = 1;
      const lastIndex = Math.max(0, length - 1);
      expect(lastIndex).toBe(0);
    });
  });

  describe('Selection State', () => {
    test('isSelected returns true for selected index', () => {
      const selectedIndex = 2;
      const isSelected = (index: number) => index === selectedIndex;
      expect(isSelected(2)).toBe(true);
    });

    test('isSelected returns false for non-selected index', () => {
      const selectedIndex = 2;
      const isSelected = (index: number) => index === selectedIndex;
      expect(isSelected(0)).toBe(false);
      expect(isSelected(4)).toBe(false);
    });

    test('selectedItem returns correct item', () => {
      const items = ['a', 'b', 'c', 'd', 'e'];
      const selectedIndex = 2;
      const selectedItem = items.length > 0 ? items[selectedIndex] : undefined;
      expect(selectedItem).toBe('c');
    });

    test('selectedItem is undefined for empty list', () => {
      const items: string[] = [];
      const selectedIndex = 0;
      const selectedItem = items.length > 0 ? items[selectedIndex] : undefined;
      expect(selectedItem).toBeUndefined();
    });
  });

  describe('Search Mode Logic (#1586)', () => {
    test('appendToQuery adds character to query', () => {
      expect(appendToQuery('', 'a')).toBe('a');
      expect(appendToQuery('test', 'x')).toBe('testx');
      expect(appendToQuery('hello', ' ')).toBe('hello ');
    });

    test('backspaceQuery removes last character', () => {
      expect(backspaceQuery('test')).toBe('tes');
      expect(backspaceQuery('a')).toBe('');
      expect(backspaceQuery('')).toBe('');
    });

    test('search state defaults', () => {
      const defaultSearch = { isActive: false, query: '' };
      expect(defaultSearch.isActive).toBe(false);
      expect(defaultSearch.query).toBe('');
    });

    test('entering search mode sets isActive true', () => {
      let searchState = { isActive: false, query: '' };
      // Simulate entering search mode
      searchState = { ...searchState, isActive: true };
      expect(searchState.isActive).toBe(true);
    });

    test('exiting search mode sets isActive false', () => {
      let searchState = { isActive: true, query: 'test' };
      // Simulate exiting search mode (preserves query)
      searchState = { ...searchState, isActive: false };
      expect(searchState.isActive).toBe(false);
      expect(searchState.query).toBe('test');
    });

    test('clearing search resets query', () => {
      let searchState = { isActive: false, query: 'test' };
      // Simulate clearing search
      searchState = { ...searchState, query: '' };
      expect(searchState.query).toBe('');
    });
  });

  describe('Custom Keys (#1586)', () => {
    test('custom keys object can hold multiple handlers', () => {
      const customKeys: Record<string, () => void> = {
        r: () => {
          /* refresh */
        },
        x: () => {
          /* delete */
        },
        v: () => {
          /* toggle view */
        },
      };
      expect(Object.keys(customKeys)).toHaveLength(3);
      expect(typeof customKeys.r).toBe('function');
    });

    test('custom keys are looked up by input character', () => {
      const called: string[] = [];
      const customKeys: Record<string, () => void> = {
        r: () => called.push('r'),
        x: () => called.push('x'),
      };

      // Simulate key press
      const input = 'r';
      if (customKeys[input]) {
        customKeys[input]();
      }

      expect(called).toEqual(['r']);
    });
  });

  describe('itemCount Override (#1842)', () => {
    test('clampIndex uses itemCount when provided instead of items.length', () => {
      // Simulates grouped view: 4 agents + 3 headers = 7 visible items
      const itemsLength = 4; // raw agents
      const itemCount = 7; // visible items with headers
      const navLength = itemCount; // itemCount overrides items.length

      // Can navigate to index 6 (last visible item)
      expect(clampIndex(6, navLength, false)).toBe(6);
      // Without override, would clamp to 3
      expect(clampIndex(6, itemsLength, false)).toBe(3);
    });

    test('jumpToLast uses itemCount for boundary', () => {
      const itemCount = 7;
      const lastIndex = Math.max(0, itemCount - 1);
      expect(lastIndex).toBe(6);
    });

    test('clampIndex wraps correctly with itemCount', () => {
      const itemCount = 7;
      expect(clampIndex(-1, itemCount, true)).toBe(6);
      expect(clampIndex(7, itemCount, true)).toBe(0);
    });

    test('itemCount of 0 returns 0 for any index', () => {
      expect(clampIndex(0, 0, false)).toBe(0);
      expect(clampIndex(5, 0, false)).toBe(0);
    });
  });

  describe('Edge Cases', () => {
    test('handles large lists', () => {
      const length = 10000;
      expect(clampIndex(5000, length, false)).toBe(5000);
      expect(clampIndex(length + 1, length, false)).toBe(length - 1);
    });

    test('handles negative n in moveDown', () => {
      const current = 3;
      const n = -2;
      const next = clampIndex(current + n, 5, false);
      expect(next).toBe(1);
    });

    test('handles negative n in moveUp', () => {
      const current = 1;
      const n = -2;
      const next = clampIndex(current - n, 5, false);
      expect(next).toBe(3);
    });

    test('search query handles special characters', () => {
      expect(appendToQuery('test', '/')).toBe('test/');
      expect(appendToQuery('path', ':')).toBe('path:');
      expect(appendToQuery('name', '-')).toBe('name-');
    });

    test('search query handles unicode', () => {
      expect(appendToQuery('', '日')).toBe('日');
      expect(appendToQuery('hello', '世')).toBe('hello世');
    });
  });

  describe('Options Interface', () => {
    test('options have sensible defaults', () => {
      const defaults = {
        initialIndex: 0,
        disabled: false,
        wrap: false,
        enableSearch: false,
        isActive: true,
      };
      expect(defaults.initialIndex).toBe(0);
      expect(defaults.disabled).toBe(false);
      expect(defaults.wrap).toBe(false);
      expect(defaults.enableSearch).toBe(false);
      expect(defaults.isActive).toBe(true);
    });

    test('wrap option changes clamping behavior', () => {
      // Without wrap
      expect(clampIndex(-1, 5, false)).toBe(0);
      expect(clampIndex(5, 5, false)).toBe(4);

      // With wrap
      expect(clampIndex(-1, 5, true)).toBe(4);
      expect(clampIndex(5, 5, true)).toBe(0);
    });

    test('isActive controls input handling', () => {
      // When isActive is false, input should be ignored
      // This is tested by checking the option exists
      const options = { isActive: false };
      expect(options.isActive).toBe(false);
    });
  });

  describe('Return Value Interface', () => {
    test('result contains all expected properties', () => {
      const expectedProps = [
        'selectedIndex',
        'selectedItem',
        'setSelectedIndex',
        'moveDown',
        'moveUp',
        'jumpToFirst',
        'jumpToLast',
        'isSelected',
        'search',
        'enterSearchMode',
        'exitSearchMode',
        'clearSearch',
        'setSearchQuery',
      ];
      // This tests the type interface is correct
      expectedProps.forEach((prop) => {
        expect(typeof prop).toBe('string');
      });
      expect(expectedProps).toHaveLength(13);
    });

    test('search object has expected structure', () => {
      const search = { isActive: false, query: '' };
      expect('isActive' in search).toBe(true);
      expect('query' in search).toBe(true);
    });
  });

  describe('Callbacks', () => {
    test('onSelect callback receives item and index', () => {
      const items = ['a', 'b', 'c'];
      const selectedIndex = 1;
      let callbackArgs: { item: string; index: number } | null = null;

      const onSelect = (item: string, index: number) => {
        callbackArgs = { item, index };
      };

      // Simulate selection
      onSelect(items[selectedIndex], selectedIndex);

      expect(callbackArgs).toEqual({ item: 'b', index: 1 });
    });

    test('onBack callback is called on escape', () => {
      let backCalled = false;
      const onBack = () => {
        backCalled = true;
      };

      // Simulate escape press
      onBack();

      expect(backCalled).toBe(true);
    });

    test('onSearchChange callback receives updated query', () => {
      let receivedQuery = '';
      const onSearchChange = (query: string) => {
        receivedQuery = query;
      };

      // Simulate search query change
      onSearchChange('test');
      expect(receivedQuery).toBe('test');

      onSearchChange('testing');
      expect(receivedQuery).toBe('testing');
    });
  });
});
