/**
 * Tests for useListNavigation hook - Vim-style list navigation
 * Validates navigation logic and type exports
 *
 * #1081 Q1 Cleanup: TUI hook test coverage
 *
 * Note: React hook testing with useInput requires stdin which is not available in Bun/Ink.
 * These tests focus on navigation logic and type validation.
 */

import { describe, it, expect } from 'bun:test';
import type {
  UseListNavigationOptions,
  UseListNavigationResult,
} from '../useListNavigation';

describe('useListNavigation - Index Clamping Logic', () => {
  // Test the clamping logic that the hook uses internally

  describe('standard clamping (no wrap)', () => {
    function clampIndex(index: number, length: number): number {
      if (length === 0) return 0;
      return Math.min(Math.max(0, index), length - 1);
    }

    it('clamps negative indices to 0', () => {
      expect(clampIndex(-1, 5)).toBe(0);
      expect(clampIndex(-100, 5)).toBe(0);
    });

    it('clamps indices beyond array length', () => {
      expect(clampIndex(10, 5)).toBe(4);
      expect(clampIndex(100, 5)).toBe(4);
    });

    it('preserves valid indices', () => {
      expect(clampIndex(0, 5)).toBe(0);
      expect(clampIndex(2, 5)).toBe(2);
      expect(clampIndex(4, 5)).toBe(4);
    });

    it('handles empty list', () => {
      expect(clampIndex(0, 0)).toBe(0);
      expect(clampIndex(5, 0)).toBe(0);
    });

    it('handles single-item list', () => {
      expect(clampIndex(0, 1)).toBe(0);
      expect(clampIndex(1, 1)).toBe(0);
      expect(clampIndex(-1, 1)).toBe(0);
    });
  });

  describe('wrap-around clamping', () => {
    function clampIndexWrap(index: number, length: number): number {
      if (length === 0) return 0;
      if (index < 0) return length - 1;
      if (index >= length) return 0;
      return index;
    }

    it('wraps negative indices to last item', () => {
      expect(clampIndexWrap(-1, 5)).toBe(4);
    });

    it('wraps indices beyond array to first item', () => {
      expect(clampIndexWrap(5, 5)).toBe(0);
    });

    it('preserves valid indices', () => {
      expect(clampIndexWrap(0, 5)).toBe(0);
      expect(clampIndexWrap(2, 5)).toBe(2);
      expect(clampIndexWrap(4, 5)).toBe(4);
    });

    it('handles empty list', () => {
      expect(clampIndexWrap(0, 0)).toBe(0);
    });

    it('handles single-item list wrap', () => {
      expect(clampIndexWrap(0, 1)).toBe(0);
      expect(clampIndexWrap(1, 1)).toBe(0);
      expect(clampIndexWrap(-1, 1)).toBe(0);
    });
  });
});

describe('useListNavigation - Initial Index Logic', () => {
  function computeInitialIndex(initialIndex: number, itemsLength: number): number {
    return Math.min(Math.max(0, initialIndex), Math.max(0, itemsLength - 1));
  }

  it('uses initial index when valid', () => {
    expect(computeInitialIndex(2, 5)).toBe(2);
    expect(computeInitialIndex(0, 5)).toBe(0);
    expect(computeInitialIndex(4, 5)).toBe(4);
  });

  it('clamps initial index to valid range', () => {
    expect(computeInitialIndex(10, 5)).toBe(4);
    expect(computeInitialIndex(-5, 5)).toBe(0);
  });

  it('handles empty list', () => {
    expect(computeInitialIndex(0, 0)).toBe(0);
    expect(computeInitialIndex(5, 0)).toBe(0);
  });

  it('defaults to 0 when initialIndex is undefined', () => {
    const defaultIndex = 0;
    expect(computeInitialIndex(defaultIndex, 5)).toBe(0);
  });
});

describe('useListNavigation - Type Exports', () => {
  it('exports UseListNavigationOptions interface', () => {
    const options: UseListNavigationOptions<string> = {
      items: ['a', 'b', 'c'],
      onSelect: (item, index) => { /* noop */ },
      initialIndex: 0,
      disabled: false,
      wrap: true,
    };

    expect(options.items).toHaveLength(3);
    expect(options.initialIndex).toBe(0);
    expect(options.disabled).toBe(false);
    expect(options.wrap).toBe(true);
  });

  it('allows partial UseListNavigationOptions', () => {
    const minimalOptions: UseListNavigationOptions<number> = {
      items: [1, 2, 3],
    };

    expect(minimalOptions.items).toHaveLength(3);
    expect(minimalOptions.onSelect).toBeUndefined();
    expect(minimalOptions.disabled).toBeUndefined();
  });

  it('supports generic type parameter', () => {
    interface CustomItem {
      id: number;
      name: string;
    }

    const options: UseListNavigationOptions<CustomItem> = {
      items: [
        { id: 1, name: 'First' },
        { id: 2, name: 'Second' },
      ],
      onSelect: (item) => {
        // Type check - should have id and name
        expect(item.id).toBeDefined();
        expect(item.name).toBeDefined();
      },
    };

    expect(options.items[0].id).toBe(1);
    expect(options.items[1].name).toBe('Second');
  });
});

describe('useListNavigation - Navigation Simulation', () => {
  // Simulate navigation without using the actual hook

  class ListNavigator<T> {
    private items: T[];
    private index: number;
    private wrap: boolean;

    constructor(items: T[], initialIndex = 0, wrap = false) {
      this.items = items;
      this.wrap = wrap;
      this.index = this.clamp(initialIndex);
    }

    private clamp(idx: number): number {
      if (this.items.length === 0) return 0;
      if (this.wrap) {
        if (idx < 0) return this.items.length - 1;
        if (idx >= this.items.length) return 0;
        return idx;
      }
      return Math.min(Math.max(0, idx), this.items.length - 1);
    }

    get selectedIndex(): number {
      return this.index;
    }

    get selectedItem(): T | undefined {
      return this.items[this.index];
    }

    moveDown(n = 1): void {
      this.index = this.clamp(this.index + n);
    }

    moveUp(n = 1): void {
      this.index = this.clamp(this.index - n);
    }

    jumpToFirst(): void {
      this.index = 0;
    }

    jumpToLast(): void {
      this.index = Math.max(0, this.items.length - 1);
    }

    isSelected(idx: number): boolean {
      return idx === this.index;
    }
  }

  describe('basic navigation', () => {
    it('starts at initial index', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 1);
      expect(nav.selectedIndex).toBe(1);
      expect(nav.selectedItem).toBe('b');
    });

    it('moves down correctly', () => {
      const nav = new ListNavigator(['a', 'b', 'c']);
      nav.moveDown();
      expect(nav.selectedIndex).toBe(1);
      nav.moveDown();
      expect(nav.selectedIndex).toBe(2);
    });

    it('moves up correctly', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 2);
      nav.moveUp();
      expect(nav.selectedIndex).toBe(1);
      nav.moveUp();
      expect(nav.selectedIndex).toBe(0);
    });

    it('jumps to first', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 2);
      nav.jumpToFirst();
      expect(nav.selectedIndex).toBe(0);
    });

    it('jumps to last', () => {
      const nav = new ListNavigator(['a', 'b', 'c']);
      nav.jumpToLast();
      expect(nav.selectedIndex).toBe(2);
    });
  });

  describe('boundary behavior without wrap', () => {
    it('stops at end when moving down', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 2);
      nav.moveDown();
      expect(nav.selectedIndex).toBe(2);
    });

    it('stops at start when moving up', () => {
      const nav = new ListNavigator(['a', 'b', 'c']);
      nav.moveUp();
      expect(nav.selectedIndex).toBe(0);
    });
  });

  describe('wrap behavior', () => {
    it('wraps to start when moving past end', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 2, true);
      nav.moveDown();
      expect(nav.selectedIndex).toBe(0);
    });

    it('wraps to end when moving before start', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 0, true);
      nav.moveUp();
      expect(nav.selectedIndex).toBe(2);
    });
  });

  describe('isSelected', () => {
    it('returns true for selected index', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 1);
      expect(nav.isSelected(1)).toBe(true);
    });

    it('returns false for non-selected indices', () => {
      const nav = new ListNavigator(['a', 'b', 'c'], 1);
      expect(nav.isSelected(0)).toBe(false);
      expect(nav.isSelected(2)).toBe(false);
    });
  });

  describe('empty list handling', () => {
    it('handles empty list gracefully', () => {
      const nav = new ListNavigator<string>([]);
      expect(nav.selectedIndex).toBe(0);
      expect(nav.selectedItem).toBeUndefined();
    });

    it('navigation on empty list is no-op', () => {
      const nav = new ListNavigator<string>([]);
      nav.moveDown();
      nav.moveUp();
      nav.jumpToFirst();
      nav.jumpToLast();
      expect(nav.selectedIndex).toBe(0);
    });
  });

  describe('multi-step navigation', () => {
    it('supports moving multiple steps', () => {
      const nav = new ListNavigator(['a', 'b', 'c', 'd', 'e']);
      nav.moveDown(2);
      expect(nav.selectedIndex).toBe(2);
      nav.moveUp(2);
      expect(nav.selectedIndex).toBe(0);
    });

    it('clamps multi-step moves', () => {
      const nav = new ListNavigator(['a', 'b', 'c']);
      nav.moveDown(10);
      expect(nav.selectedIndex).toBe(2);
    });
  });
});

describe('useListNavigation - Function Import', () => {
  it('useListNavigation function is importable', async () => {
    const module = await import('../useListNavigation');
    expect(typeof module.useListNavigation).toBe('function');
    expect(typeof module.default).toBe('function');
  });
});
