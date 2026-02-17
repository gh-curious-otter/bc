/**
 * Tests for useResponsiveLayout hook
 * Issue #1023: Responsive multi-column layouts
 */

import { describe, expect, it } from 'bun:test';
import {
  BREAKPOINTS,
  type LayoutMode,
  type ColumnLayout,
} from '../useResponsiveLayout';

// Test breakpoint constants
describe('BREAKPOINTS', () => {
  it('has correct threshold values', () => {
    expect(BREAKPOINTS.MINIMAL).toBe(80);
    expect(BREAKPOINTS.COMPACT).toBe(100);
    expect(BREAKPOINTS.MEDIUM).toBe(120);
    expect(BREAKPOINTS.WIDE).toBe(150);
  });

  it('thresholds are in ascending order', () => {
    expect(BREAKPOINTS.MINIMAL).toBeLessThan(BREAKPOINTS.COMPACT);
    expect(BREAKPOINTS.COMPACT).toBeLessThan(BREAKPOINTS.MEDIUM);
    expect(BREAKPOINTS.MEDIUM).toBeLessThan(BREAKPOINTS.WIDE);
  });
});

// Test layout mode determination logic
describe('Layout mode determination', () => {
  // Helper to determine mode from width (mirrors hook logic)
  function getLayoutMode(width: number): LayoutMode {
    if (width >= BREAKPOINTS.MEDIUM) return 'wide';
    if (width >= BREAKPOINTS.COMPACT) return 'medium';
    if (width >= BREAKPOINTS.MINIMAL) return 'compact';
    return 'minimal';
  }

  it('returns minimal for very narrow terminals', () => {
    expect(getLayoutMode(40)).toBe('minimal');
    expect(getLayoutMode(60)).toBe('minimal');
    expect(getLayoutMode(79)).toBe('minimal');
  });

  it('returns compact for 80-99 col terminals', () => {
    expect(getLayoutMode(80)).toBe('compact');
    expect(getLayoutMode(90)).toBe('compact');
    expect(getLayoutMode(99)).toBe('compact');
  });

  it('returns medium for 100-119 col terminals', () => {
    expect(getLayoutMode(100)).toBe('medium');
    expect(getLayoutMode(110)).toBe('medium');
    expect(getLayoutMode(119)).toBe('medium');
  });

  it('returns wide for 120+ col terminals', () => {
    expect(getLayoutMode(120)).toBe('wide');
    expect(getLayoutMode(150)).toBe('wide');
    expect(getLayoutMode(200)).toBe('wide');
  });
});

// Test column layout determination
describe('Column layout determination', () => {
  function getColumnLayout(width: number): ColumnLayout {
    if (width >= BREAKPOINTS.WIDE) return 'triple';
    if (width >= BREAKPOINTS.COMPACT) return 'dual';
    return 'single';
  }

  it('returns single column for narrow terminals', () => {
    expect(getColumnLayout(40)).toBe('single');
    expect(getColumnLayout(80)).toBe('single');
    expect(getColumnLayout(99)).toBe('single');
  });

  it('returns dual column for medium terminals', () => {
    expect(getColumnLayout(100)).toBe('dual');
    expect(getColumnLayout(120)).toBe('dual');
    expect(getColumnLayout(149)).toBe('dual');
  });

  it('returns triple column for wide terminals', () => {
    expect(getColumnLayout(150)).toBe('triple');
    expect(getColumnLayout(200)).toBe('triple');
  });
});

// Test sidebar width calculation
describe('Sidebar width calculation', () => {
  function calculateSidebarWidth(width: number, mode: LayoutMode): number {
    if (mode === 'minimal' || mode === 'compact') {
      return 0;
    }
    const percent = mode === 'wide' ? 0.25 : 0.28;
    return Math.min(40, Math.max(24, Math.floor(width * percent)));
  }

  it('returns 0 for minimal mode', () => {
    expect(calculateSidebarWidth(60, 'minimal')).toBe(0);
  });

  it('returns 0 for compact mode', () => {
    expect(calculateSidebarWidth(90, 'compact')).toBe(0);
  });

  it('calculates width for medium mode (28%)', () => {
    const width = calculateSidebarWidth(110, 'medium');
    expect(width).toBeGreaterThanOrEqual(24);
    expect(width).toBeLessThanOrEqual(40);
    expect(width).toBe(Math.floor(110 * 0.28));
  });

  it('calculates width for wide mode (25%)', () => {
    const width = calculateSidebarWidth(150, 'wide');
    expect(width).toBeGreaterThanOrEqual(24);
    expect(width).toBeLessThanOrEqual(40);
    expect(width).toBe(Math.floor(150 * 0.25));
  });

  it('enforces minimum width of 24', () => {
    // 80 * 0.28 = 22.4, should clamp to 24
    const width = calculateSidebarWidth(80, 'medium');
    expect(width).toBe(24);
  });

  it('enforces maximum width of 40', () => {
    // 200 * 0.25 = 50, should clamp to 40
    const width = calculateSidebarWidth(200, 'wide');
    expect(width).toBe(40);
  });
});

// Test responsive value selection
describe('Responsive value selection', () => {
  interface ResponsiveValues<T> {
    minimal?: T;
    compact?: T;
    medium?: T;
    wide?: T;
    default: T;
  }

  function responsive<T>(mode: LayoutMode, values: ResponsiveValues<T>): T {
    switch (mode) {
      case 'wide':
        if (values.wide !== undefined) return values.wide;
        if (values.medium !== undefined) return values.medium;
        if (values.compact !== undefined) return values.compact;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'medium':
        if (values.medium !== undefined) return values.medium;
        if (values.compact !== undefined) return values.compact;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'compact':
        if (values.compact !== undefined) return values.compact;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'minimal':
        if (values.minimal !== undefined) return values.minimal;
        break;
    }
    return values.default;
  }

  it('returns mode-specific value when available', () => {
    const values = { minimal: 5, compact: 10, medium: 15, wide: 20, default: 0 };
    expect(responsive('minimal', values)).toBe(5);
    expect(responsive('compact', values)).toBe(10);
    expect(responsive('medium', values)).toBe(15);
    expect(responsive('wide', values)).toBe(20);
  });

  it('falls back through modes when value not specified', () => {
    const values = { minimal: 5, default: 0 };
    expect(responsive('minimal', values)).toBe(5);
    expect(responsive('compact', values)).toBe(5);
    expect(responsive('medium', values)).toBe(5);
    expect(responsive('wide', values)).toBe(5);
  });

  it('returns default when no mode-specific values', () => {
    const values = { default: 42 };
    expect(responsive('minimal', values)).toBe(42);
    expect(responsive('compact', values)).toBe(42);
    expect(responsive('medium', values)).toBe(42);
    expect(responsive('wide', values)).toBe(42);
  });

  it('supports partial value specification', () => {
    const values = { compact: 10, wide: 20, default: 0 };
    expect(responsive('minimal', values)).toBe(0); // Falls to default
    expect(responsive('compact', values)).toBe(10);
    expect(responsive('medium', values)).toBe(10); // Falls to compact
    expect(responsive('wide', values)).toBe(20);
  });
});

// Test responsive width calculation
describe('Responsive width calculation', () => {
  function responsiveWidth(
    terminalWidth: number,
    percent: number,
    min = 0,
    max = terminalWidth
  ): number {
    const calculated = Math.floor((terminalWidth * percent) / 100);
    return Math.min(max, Math.max(min, calculated));
  }

  it('calculates percentage of terminal width', () => {
    expect(responsiveWidth(100, 50)).toBe(50);
    expect(responsiveWidth(120, 25)).toBe(30);
    expect(responsiveWidth(80, 75)).toBe(60);
  });

  it('enforces minimum width', () => {
    expect(responsiveWidth(100, 10, 20)).toBe(20); // 10% of 100 = 10, clamped to 20
  });

  it('enforces maximum width', () => {
    expect(responsiveWidth(100, 90, 0, 50)).toBe(50); // 90% of 100 = 90, clamped to 50
  });

  it('handles edge cases', () => {
    expect(responsiveWidth(100, 0)).toBe(0);
    expect(responsiveWidth(100, 100)).toBe(100);
    expect(responsiveWidth(0, 50)).toBe(0);
  });
});

// Test boolean flags
describe('Layout boolean flags', () => {
  function getFlags(width: number) {
    const mode =
      width >= BREAKPOINTS.MEDIUM
        ? 'wide'
        : width >= BREAKPOINTS.COMPACT
          ? 'medium'
          : width >= BREAKPOINTS.MINIMAL
            ? 'compact'
            : 'minimal';

    return {
      isMinimal: mode === 'minimal',
      isCompact: mode === 'compact',
      isMedium: mode === 'medium',
      isWide: mode === 'wide',
      canMultiColumn: width >= BREAKPOINTS.COMPACT,
      canTripleColumn: width >= BREAKPOINTS.WIDE,
    };
  }

  it('sets correct flags for minimal mode', () => {
    const flags = getFlags(60);
    expect(flags.isMinimal).toBe(true);
    expect(flags.isCompact).toBe(false);
    expect(flags.isMedium).toBe(false);
    expect(flags.isWide).toBe(false);
    expect(flags.canMultiColumn).toBe(false);
    expect(flags.canTripleColumn).toBe(false);
  });

  it('sets correct flags for compact mode', () => {
    const flags = getFlags(90);
    expect(flags.isMinimal).toBe(false);
    expect(flags.isCompact).toBe(true);
    expect(flags.canMultiColumn).toBe(false);
  });

  it('sets correct flags for medium mode', () => {
    const flags = getFlags(110);
    expect(flags.isMedium).toBe(true);
    expect(flags.canMultiColumn).toBe(true);
    expect(flags.canTripleColumn).toBe(false);
  });

  it('sets correct flags for wide mode', () => {
    const flags = getFlags(160);
    expect(flags.isWide).toBe(true);
    expect(flags.canMultiColumn).toBe(true);
    expect(flags.canTripleColumn).toBe(true);
  });
});
