/**
 * useResponsiveLayout - Centralized responsive layout system for TUI
 * Issue #1023: Responsive multi-column layouts
 * Issue #1326: Comprehensive 5-tier breakpoint system
 *
 * Provides:
 * - Standardized breakpoints across all components (XS/SM/MD/LG/XL)
 * - Layout mode detection with graceful degradation
 * - Responsive value helpers
 * - Terminal dimension hooks
 * - Drawer/detail pane configuration per breakpoint
 *
 * Breakpoints (Issue #1326):
 * - XS: <80 cols (navigation overlay, single pane)
 * - SM: 80-99 cols (minimal 6-char drawer, inline stats)
 * - MD: 100-119 cols (10-char drawer, single column)
 * - LG: 120-139 cols (14-char drawer, two columns)
 * - XL: >=140 cols (three columns with detail pane)
 */

import { useMemo } from 'react';
import { useStdout } from 'ink';

/** Terminal width breakpoint thresholds (#1326) */
export const BREAKPOINTS = {
  /** XS: Very narrow terminals - single pane, navigation overlay */
  XS: 80,
  /** SM: Standard 80-col terminal - minimal drawer, inline stats */
  SM: 100,
  /** MD: Medium width - 10-char drawer, single column */
  MD: 120,
  /** LG: Large - 14-char drawer, two columns */
  LG: 140,
  /** XL: Extra large - three columns with detail pane */
  XL: 160,
} as const;

/** Backwards compatibility aliases */
export const BREAKPOINTS_LEGACY = {
  MINIMAL: BREAKPOINTS.XS,
  COMPACT: BREAKPOINTS.SM,
  MEDIUM: BREAKPOINTS.MD,
  WIDE: BREAKPOINTS.LG,
} as const;

/** Layout mode based on terminal width (#1326) */
export type LayoutMode = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

/** Legacy layout mode for backwards compatibility */
export type LegacyLayoutMode = 'minimal' | 'compact' | 'medium' | 'wide';

/** Column layout configuration */
export type ColumnLayout = 'single' | 'dual' | 'triple';

/** Drawer configuration per breakpoint (#1326) */
export interface DrawerConfig {
  /** Whether drawer is visible */
  visible: boolean;
  /** Drawer width in characters */
  width: number;
  /** Whether to show short labels */
  shrunk: boolean;
}

/** Detail pane configuration per breakpoint (#1326) */
export interface DetailPaneConfig {
  /** Whether detail pane is visible */
  visible: boolean;
  /** Detail pane width in characters */
  width: number;
  /** Whether detail pane is compressed */
  compressed: boolean;
}

/** Responsive layout state */
export interface ResponsiveLayoutState {
  /** Current terminal width in columns */
  width: number;
  /** Current terminal height in rows */
  height: number;
  /** Current layout mode based on width (#1326) */
  mode: LayoutMode;
  /** Legacy mode for backwards compatibility */
  legacyMode: LegacyLayoutMode;
  /** Recommended column layout */
  columnLayout: ColumnLayout;
  /** Drawer configuration for current breakpoint */
  drawer: DrawerConfig;
  /** Detail pane configuration for current breakpoint */
  detailPane: DetailPaneConfig;
  /** Whether in XS mode (<80 cols) */
  isXS: boolean;
  /** Whether in SM mode (80-99 cols) */
  isSM: boolean;
  /** Whether in MD mode (100-119 cols) */
  isMD: boolean;
  /** Whether in LG mode (120-139 cols) */
  isLG: boolean;
  /** Whether in XL mode (>=140 cols) */
  isXL: boolean;
  // Legacy compatibility flags
  /** @deprecated Use isXS */
  isMinimal: boolean;
  /** @deprecated Use isSM */
  isCompact: boolean;
  /** @deprecated Use isMD */
  isMedium: boolean;
  /** @deprecated Use isLG || isXL */
  isWide: boolean;
  /** Whether multi-column layout is available (LG+) */
  canMultiColumn: boolean;
  /** Whether triple column layout is available (XL+) */
  canTripleColumn: boolean;
  /** Whether detail pane can be shown (XL+) */
  canShowDetail: boolean;
  /** Available content width (after drawer) */
  contentWidth: number;
}

/** Responsive value options for different breakpoints */
export interface ResponsiveValues<T> {
  /** Value for XS mode (<80 cols) */
  xs?: T;
  /** Value for SM mode (80-99 cols) */
  sm?: T;
  /** Value for MD mode (100-119 cols) */
  md?: T;
  /** Value for LG mode (120-139 cols) */
  lg?: T;
  /** Value for XL mode (>=140 cols) */
  xl?: T;
  // Legacy support
  /** @deprecated Use xs */
  minimal?: T;
  /** @deprecated Use sm */
  compact?: T;
  /** @deprecated Use md */
  medium?: T;
  /** @deprecated Use lg/xl */
  wide?: T;
  /** Default value if mode-specific not provided */
  default: T;
}

export interface UseResponsiveLayoutOptions {
  /** Override terminal width (for testing) */
  terminalWidth?: number;
  /** Override terminal height (for testing) */
  terminalHeight?: number;
}

export interface UseResponsiveLayoutResult extends ResponsiveLayoutState {
  /**
   * Get a value based on current layout mode
   * Falls back through modes: xs -> sm -> md -> lg -> xl -> default
   */
  responsive: <T>(values: ResponsiveValues<T>) => T;
  /**
   * Calculate responsive width as percentage of terminal
   * @param percent Percentage of terminal width (0-100)
   * @param min Minimum width in columns
   * @param max Maximum width in columns
   */
  responsiveWidth: (percent: number, min?: number, max?: number) => number;
  /**
   * Get flex direction based on layout mode
   * Returns 'row' for multi-column capable modes, 'column' otherwise
   */
  flexDirection: 'row' | 'column';
  /**
   * @deprecated Use drawer.width
   */
  sidebarWidth: number;
  /**
   * @deprecated Use contentWidth
   */
  mainContentWidth: number;
}

/**
 * Determine layout mode from terminal width (#1326)
 */
function getLayoutMode(width: number): LayoutMode {
  if (width >= BREAKPOINTS.LG) return 'xl';
  if (width >= BREAKPOINTS.MD) return 'lg';
  if (width >= BREAKPOINTS.SM) return 'md';
  if (width >= BREAKPOINTS.XS) return 'sm';
  return 'xs';
}

/**
 * Map new mode to legacy mode for backwards compatibility
 */
function getLegacyMode(mode: LayoutMode): LegacyLayoutMode {
  switch (mode) {
    case 'xs': return 'minimal';
    case 'sm': return 'compact';
    case 'md': return 'medium';
    case 'lg':
    case 'xl': return 'wide';
  }
}

/**
 * Determine column layout from terminal width (#1326)
 */
function getColumnLayout(width: number): ColumnLayout {
  if (width >= BREAKPOINTS.LG) return 'triple';
  if (width >= BREAKPOINTS.MD) return 'dual';
  return 'single';
}

/**
 * Get drawer configuration for current breakpoint (#1326)
 * Per ux-01 spec:
 * - XS: Hidden (overlay nav)
 * - SM: 6-char minimal drawer
 * - MD: 10-char drawer
 * - LG/XL: 14-char full drawer
 */
function getDrawerConfig(mode: LayoutMode): DrawerConfig {
  switch (mode) {
    case 'xs':
      // XS: Drawer hidden, use navigation overlay
      return { visible: false, width: 0, shrunk: true };
    case 'sm':
      // SM: Minimal drawer with 6-char width
      return { visible: true, width: 6, shrunk: true };
    case 'md':
      // MD: 10-char drawer
      return { visible: true, width: 10, shrunk: true };
    case 'lg':
    case 'xl':
      // LG/XL: Full 14-char drawer
      return { visible: true, width: 14, shrunk: false };
  }
}

/**
 * Get detail pane configuration for current breakpoint (#1326)
 * Per ux-01 spec:
 * - XS/SM/MD/LG: No detail pane
 * - XL: Full detail pane (30 chars)
 */
function getDetailPaneConfig(mode: LayoutMode): DetailPaneConfig {
  switch (mode) {
    case 'xs':
    case 'sm':
    case 'md':
    case 'lg':
      // XS-LG: No detail pane
      return { visible: false, width: 0, compressed: false };
    case 'xl':
      // XL: Full detail pane
      return { visible: true, width: 30, compressed: false };
  }
}

/**
 * Hook for responsive layout management (#1326)
 *
 * @example
 * ```tsx
 * const { mode, isXL, responsive, drawer, detailPane } = useResponsiveLayout();
 *
 * // Conditional rendering based on breakpoint
 * {drawer.visible && <Drawer shrunk={drawer.shrunk} width={drawer.width} />}
 * {detailPane.visible && <DetailPane />}
 *
 * // Responsive values
 * const maxItems = responsive({ xs: 5, sm: 8, md: 12, lg: 18, xl: 24, default: 12 });
 *
 * // Legacy support still works
 * const count = responsive({ minimal: 5, compact: 10, wide: 20, default: 15 });
 * ```
 */
export function useResponsiveLayout(
  options: UseResponsiveLayoutOptions = {}
): UseResponsiveLayoutResult {
  const { stdout } = useStdout();

  // Use override values for testing, otherwise use actual terminal dimensions
  const width = options.terminalWidth ?? stdout.columns;
  const height = options.terminalHeight ?? stdout.rows;

  // Calculate all responsive state in one memoized block
  const state = useMemo<ResponsiveLayoutState>(() => {
    const mode = getLayoutMode(width);
    const legacyMode = getLegacyMode(mode);
    const columnLayout = getColumnLayout(width);
    const drawer = getDrawerConfig(mode);
    const detailPane = getDetailPaneConfig(mode);

    // Calculate available content width
    const appPadding = 2; // 1 char padding on each side
    const contentPadding = drawer.visible ? 1 : 0;
    const contentWidth = width - appPadding - drawer.width - contentPadding - detailPane.width;

    return {
      width,
      height,
      mode,
      legacyMode,
      columnLayout,
      drawer,
      detailPane,
      // New breakpoint flags
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
      // Feature flags - per #1326 spec
      canMultiColumn: width >= BREAKPOINTS.MD, // LG+ for multi-column
      canTripleColumn: width >= BREAKPOINTS.LG, // XL+ for triple column
      canShowDetail: width >= BREAKPOINTS.LG, // XL+ for detail pane
      contentWidth,
    };
  }, [width, height]);

  // Responsive value selector with fallback chain
  const responsive = useMemo(() => {
    return <T>(values: ResponsiveValues<T>): T => {
      // Try new mode-specific values first, then legacy, then fall back
      switch (state.mode) {
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
    };
  }, [state.mode]);

  // Responsive width calculator
  const responsiveWidth = useMemo(() => {
    return (percent: number, min = 0, max = width): number => {
      const calculated = Math.floor((width * percent) / 100);
      return Math.min(max, Math.max(min, calculated));
    };
  }, [width]);

  // Legacy compatibility values
  const sidebarWidth = state.drawer.width;
  const mainContentWidth = state.contentWidth;
  const flexDirection = state.canMultiColumn ? 'row' : 'column';

  return {
    ...state,
    responsive,
    responsiveWidth,
    flexDirection,
    sidebarWidth,
    mainContentWidth,
  };
}

/**
 * Simple hook to just get terminal dimensions
 * Use when you only need width/height without layout calculations
 */
export function useTerminalSize(options: UseResponsiveLayoutOptions = {}): {
  width: number;
  height: number;
} {
  const { stdout } = useStdout();

  return useMemo(() => ({
    width: options.terminalWidth ?? stdout.columns,
    height: options.terminalHeight ?? stdout.rows,
  }), [options.terminalWidth, options.terminalHeight, stdout.columns, stdout.rows]);
}

export default useResponsiveLayout;
