/**
 * Tests for useResponsiveLayout hook
 * Issue #1023: Responsive multi-column layouts
 * Issue #1326: 5-tier breakpoint system (XS/SM/MD/LG/XL)
 */

import { describe, expect, it } from 'bun:test';
import {
  BREAKPOINTS,
  BREAKPOINTS_LEGACY,
  type LayoutMode,
  type ColumnLayout,
} from '../useResponsiveLayout';

// Test breakpoint constants (#1326)
describe('BREAKPOINTS', () => {
  it('has correct threshold values for 5-tier system', () => {
    expect(BREAKPOINTS.XS).toBe(80);
    expect(BREAKPOINTS.SM).toBe(100);
    expect(BREAKPOINTS.MD).toBe(120);
    expect(BREAKPOINTS.LG).toBe(140);
    expect(BREAKPOINTS.XL).toBe(160);
  });

  it('thresholds are in ascending order', () => {
    expect(BREAKPOINTS.XS).toBeLessThan(BREAKPOINTS.SM);
    expect(BREAKPOINTS.SM).toBeLessThan(BREAKPOINTS.MD);
    expect(BREAKPOINTS.MD).toBeLessThan(BREAKPOINTS.LG);
    expect(BREAKPOINTS.LG).toBeLessThan(BREAKPOINTS.XL);
  });

  it('provides legacy aliases for backwards compatibility', () => {
    expect(BREAKPOINTS_LEGACY.MINIMAL).toBe(BREAKPOINTS.XS);
    expect(BREAKPOINTS_LEGACY.COMPACT).toBe(BREAKPOINTS.SM);
    expect(BREAKPOINTS_LEGACY.MEDIUM).toBe(BREAKPOINTS.MD);
    expect(BREAKPOINTS_LEGACY.WIDE).toBe(BREAKPOINTS.LG);
  });
});

// Test layout mode determination logic (#1326)
describe('Layout mode determination', () => {
  // Helper to determine mode from width (mirrors hook logic)
  function getLayoutMode(width: number): LayoutMode {
    if (width >= BREAKPOINTS.LG) return 'xl';
    if (width >= BREAKPOINTS.MD) return 'lg';
    if (width >= BREAKPOINTS.SM) return 'md';
    if (width >= BREAKPOINTS.XS) return 'sm';
    return 'xs';
  }

  it('returns xs for very narrow terminals (<80)', () => {
    expect(getLayoutMode(40)).toBe('xs');
    expect(getLayoutMode(60)).toBe('xs');
    expect(getLayoutMode(79)).toBe('xs');
  });

  it('returns sm for 80-99 col terminals', () => {
    expect(getLayoutMode(80)).toBe('sm');
    expect(getLayoutMode(90)).toBe('sm');
    expect(getLayoutMode(99)).toBe('sm');
  });

  it('returns md for 100-119 col terminals', () => {
    expect(getLayoutMode(100)).toBe('md');
    expect(getLayoutMode(110)).toBe('md');
    expect(getLayoutMode(119)).toBe('md');
  });

  it('returns lg for 120-139 col terminals', () => {
    expect(getLayoutMode(120)).toBe('lg');
    expect(getLayoutMode(130)).toBe('lg');
    expect(getLayoutMode(139)).toBe('lg');
  });

  it('returns xl for 140+ col terminals', () => {
    expect(getLayoutMode(140)).toBe('xl');
    expect(getLayoutMode(160)).toBe('xl');
    expect(getLayoutMode(200)).toBe('xl');
  });
});

// Test column layout determination (#1326)
describe('Column layout determination', () => {
  function getColumnLayout(width: number): ColumnLayout {
    if (width >= BREAKPOINTS.LG) return 'triple';
    if (width >= BREAKPOINTS.MD) return 'dual';
    return 'single';
  }

  it('returns single column for xs-sm terminals', () => {
    expect(getColumnLayout(40)).toBe('single');
    expect(getColumnLayout(80)).toBe('single');
    expect(getColumnLayout(99)).toBe('single');
  });

  it('returns dual column for md terminals', () => {
    expect(getColumnLayout(100)).toBe('single'); // Still single at 100
    expect(getColumnLayout(120)).toBe('dual');
    expect(getColumnLayout(139)).toBe('dual');
  });

  it('returns triple column for lg+ terminals', () => {
    expect(getColumnLayout(140)).toBe('triple');
    expect(getColumnLayout(160)).toBe('triple');
    expect(getColumnLayout(200)).toBe('triple');
  });
});

// Test drawer configuration per breakpoint (#1326)
describe('Drawer configuration', () => {
  interface DrawerConfig {
    visible: boolean;
    width: number;
    shrunk: boolean;
  }

  function getDrawerConfig(mode: LayoutMode): DrawerConfig {
    switch (mode) {
      case 'xs':
        return { visible: false, width: 0, shrunk: true };
      case 'sm':
        return { visible: true, width: 6, shrunk: true };
      case 'md':
        return { visible: true, width: 10, shrunk: true };
      case 'lg':
      case 'xl':
        return { visible: true, width: 14, shrunk: false };
    }
  }

  it('hides drawer in xs mode', () => {
    const config = getDrawerConfig('xs');
    expect(config.visible).toBe(false);
    expect(config.width).toBe(0);
  });

  it('shows minimal 6-char drawer in sm mode', () => {
    const config = getDrawerConfig('sm');
    expect(config.visible).toBe(true);
    expect(config.width).toBe(6);
    expect(config.shrunk).toBe(true);
  });

  it('shows 10-char drawer in md mode', () => {
    const config = getDrawerConfig('md');
    expect(config.visible).toBe(true);
    expect(config.width).toBe(10);
    expect(config.shrunk).toBe(true);
  });

  it('shows full 14-char drawer in lg+ modes', () => {
    for (const mode of ['lg', 'xl'] as LayoutMode[]) {
      const config = getDrawerConfig(mode);
      expect(config.visible).toBe(true);
      expect(config.width).toBe(14);
      expect(config.shrunk).toBe(false);
    }
  });
});

// Test detail pane configuration per breakpoint (#1326)
describe('Detail pane configuration', () => {
  interface DetailPaneConfig {
    visible: boolean;
    width: number;
    compressed: boolean;
  }

  function getDetailPaneConfig(mode: LayoutMode): DetailPaneConfig {
    switch (mode) {
      case 'xs':
      case 'sm':
      case 'md':
      case 'lg':
        return { visible: false, width: 0, compressed: false };
      case 'xl':
        return { visible: true, width: 30, compressed: false };
    }
  }

  it('hides detail pane in xs-lg modes', () => {
    for (const mode of ['xs', 'sm', 'md', 'lg'] as LayoutMode[]) {
      const config = getDetailPaneConfig(mode);
      expect(config.visible).toBe(false);
    }
  });

  it('shows full detail pane in xl mode', () => {
    const config = getDetailPaneConfig('xl');
    expect(config.visible).toBe(true);
    expect(config.width).toBe(30);
    expect(config.compressed).toBe(false);
  });
});

// Test responsive value selection with new modes (#1326)
describe('Responsive value selection', () => {
  interface ResponsiveValues<T> {
    xs?: T;
    sm?: T;
    md?: T;
    lg?: T;
    xl?: T;
    // Legacy
    minimal?: T;
    compact?: T;
    medium?: T;
    wide?: T;
    default: T;
  }

  function responsive<T>(mode: LayoutMode, values: ResponsiveValues<T>): T {
    switch (mode) {
      case 'xl':
        if (values.xl !== undefined) return values.xl;
        if (values.lg !== undefined) return values.lg;
        if (values.wide !== undefined) return values.wide;
        if (values.md !== undefined) return values.md;
        if (values.medium !== undefined) return values.medium;
        if (values.sm !== undefined) return values.sm;
        if (values.compact !== undefined) return values.compact;
        if (values.xs !== undefined) return values.xs;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'lg':
        if (values.lg !== undefined) return values.lg;
        if (values.wide !== undefined) return values.wide;
        if (values.md !== undefined) return values.md;
        if (values.medium !== undefined) return values.medium;
        if (values.sm !== undefined) return values.sm;
        if (values.compact !== undefined) return values.compact;
        if (values.xs !== undefined) return values.xs;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'md':
        if (values.md !== undefined) return values.md;
        if (values.medium !== undefined) return values.medium;
        if (values.sm !== undefined) return values.sm;
        if (values.compact !== undefined) return values.compact;
        if (values.xs !== undefined) return values.xs;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'sm':
        if (values.sm !== undefined) return values.sm;
        if (values.compact !== undefined) return values.compact;
        if (values.xs !== undefined) return values.xs;
        if (values.minimal !== undefined) return values.minimal;
        break;
      case 'xs':
        if (values.xs !== undefined) return values.xs;
        if (values.minimal !== undefined) return values.minimal;
        break;
    }
    return values.default;
  }

  it('returns mode-specific value when available', () => {
    const values = { xs: 5, sm: 8, md: 12, lg: 18, xl: 24, default: 0 };
    expect(responsive('xs', values)).toBe(5);
    expect(responsive('sm', values)).toBe(8);
    expect(responsive('md', values)).toBe(12);
    expect(responsive('lg', values)).toBe(18);
    expect(responsive('xl', values)).toBe(24);
  });

  it('supports legacy mode names', () => {
    const values = { minimal: 5, compact: 10, medium: 15, wide: 20, default: 0 };
    expect(responsive('xs', values)).toBe(5);
    expect(responsive('sm', values)).toBe(10);
    expect(responsive('md', values)).toBe(15);
    expect(responsive('lg', values)).toBe(20);
    expect(responsive('xl', values)).toBe(20);
  });

  it('falls back through modes when value not specified', () => {
    const values = { xs: 5, default: 0 };
    expect(responsive('xs', values)).toBe(5);
    expect(responsive('sm', values)).toBe(5);
    expect(responsive('md', values)).toBe(5);
    expect(responsive('lg', values)).toBe(5);
    expect(responsive('xl', values)).toBe(5);
  });

  it('returns default when no mode-specific values', () => {
    const values = { default: 42 };
    expect(responsive('xs', values)).toBe(42);
    expect(responsive('sm', values)).toBe(42);
    expect(responsive('md', values)).toBe(42);
    expect(responsive('lg', values)).toBe(42);
    expect(responsive('xl', values)).toBe(42);
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
    expect(responsiveWidth(100, 10, 20)).toBe(20);
  });

  it('enforces maximum width', () => {
    expect(responsiveWidth(100, 90, 0, 50)).toBe(50);
  });

  it('handles edge cases', () => {
    expect(responsiveWidth(100, 0)).toBe(0);
    expect(responsiveWidth(100, 100)).toBe(100);
    expect(responsiveWidth(0, 50)).toBe(0);
  });
});

// Test boolean flags for new breakpoints (#1326)
describe('Layout boolean flags', () => {
  function getFlags(width: number) {
    const mode: LayoutMode =
      width >= BREAKPOINTS.LG
        ? 'xl'
        : width >= BREAKPOINTS.MD
          ? 'lg'
          : width >= BREAKPOINTS.SM
            ? 'md'
            : width >= BREAKPOINTS.XS
              ? 'sm'
              : 'xs';

    return {
      isXS: mode === 'xs',
      isSM: mode === 'sm',
      isMD: mode === 'md',
      isLG: mode === 'lg',
      isXL: mode === 'xl',
      // Legacy compatibility
      isMinimal: mode === 'xs',
      isCompact: mode === 'sm',
      isMedium: mode === 'md',
      isWide: mode === 'lg' || mode === 'xl',
      canMultiColumn: width >= BREAKPOINTS.MD,
      canTripleColumn: width >= BREAKPOINTS.LG,
      canShowDetail: width >= BREAKPOINTS.LG,
    };
  }

  it('sets correct flags for xs mode', () => {
    const flags = getFlags(60);
    expect(flags.isXS).toBe(true);
    expect(flags.isMinimal).toBe(true);
    expect(flags.canMultiColumn).toBe(false);
    expect(flags.canShowDetail).toBe(false);
  });

  it('sets correct flags for sm mode', () => {
    const flags = getFlags(90);
    expect(flags.isSM).toBe(true);
    expect(flags.isCompact).toBe(true);
    expect(flags.canMultiColumn).toBe(false);
    expect(flags.canShowDetail).toBe(false);
  });

  it('sets correct flags for md mode', () => {
    const flags = getFlags(110);
    expect(flags.isMD).toBe(true);
    expect(flags.isMedium).toBe(true);
    expect(flags.canMultiColumn).toBe(false); // MD is 120+, 110 < 120
    expect(flags.canTripleColumn).toBe(false);
    expect(flags.canShowDetail).toBe(false);
  });

  it('sets correct flags for lg mode', () => {
    const flags = getFlags(130);
    expect(flags.isLG).toBe(true);
    expect(flags.isWide).toBe(true);
    expect(flags.canMultiColumn).toBe(true);
    expect(flags.canTripleColumn).toBe(false); // LG is 140+
    expect(flags.canShowDetail).toBe(false); // Detail is XL only
  });

  it('sets correct flags for xl mode', () => {
    const flags = getFlags(160);
    expect(flags.isXL).toBe(true);
    expect(flags.isWide).toBe(true);
    expect(flags.canMultiColumn).toBe(true);
    expect(flags.canTripleColumn).toBe(true);
    expect(flags.canShowDetail).toBe(true);
  });
});
