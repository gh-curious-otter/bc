/**
 * useResponsiveLayout - Centralized responsive layout system for TUI
 * Issue #1023: Responsive multi-column layouts
 *
 * Provides:
 * - Standardized breakpoints across all components
 * - Layout mode detection (single/dual/wide column)
 * - Responsive value helpers
 * - Terminal dimension hooks
 *
 * Breakpoints align with TabBar display modes:
 * - minimal: <80 cols (very narrow terminals)
 * - compact: 80-99 cols (standard 80-col terminal)
 * - medium: 100-119 cols (slightly wider)
 * - wide: 120+ cols (full feature display)
 */

import { useMemo } from 'react';
import { useStdout } from 'ink';

/** Terminal width breakpoint thresholds */
export const BREAKPOINTS = {
  /** Very narrow terminals - single column, minimal UI */
  MINIMAL: 80,
  /** Standard 80-col terminal - compact single column */
  COMPACT: 100,
  /** Medium width - enables two-column layouts */
  MEDIUM: 120,
  /** Wide terminals - full feature display */
  WIDE: 150,
} as const;

/** Layout mode based on terminal width */
export type LayoutMode = 'minimal' | 'compact' | 'medium' | 'wide';

/** Column layout configuration */
export type ColumnLayout = 'single' | 'dual' | 'triple';

/** Responsive layout state */
export interface ResponsiveLayoutState {
  /** Current terminal width in columns */
  width: number;
  /** Current terminal height in rows */
  height: number;
  /** Current layout mode based on width */
  mode: LayoutMode;
  /** Recommended column layout */
  columnLayout: ColumnLayout;
  /** Whether in minimal/narrow mode */
  isMinimal: boolean;
  /** Whether in compact mode (80-99 cols) */
  isCompact: boolean;
  /** Whether in medium mode (100-119 cols) */
  isMedium: boolean;
  /** Whether in wide mode (120+ cols) */
  isWide: boolean;
  /** Whether multi-column layout is available */
  canMultiColumn: boolean;
  /** Whether triple column layout is available */
  canTripleColumn: boolean;
}

/** Responsive value options for different breakpoints */
export interface ResponsiveValues<T> {
  /** Value for minimal mode (<80 cols) */
  minimal?: T;
  /** Value for compact mode (80-99 cols) */
  compact?: T;
  /** Value for medium mode (100-119 cols) */
  medium?: T;
  /** Value for wide mode (120+ cols) */
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
   * Falls back through modes: minimal -> compact -> medium -> wide -> default
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
   * Get recommended sidebar/panel width
   */
  sidebarWidth: number;
  /**
   * Get recommended main content width (terminal width minus sidebar)
   */
  mainContentWidth: number;
}

/**
 * Determine layout mode from terminal width
 */
function getLayoutMode(width: number): LayoutMode {
  if (width >= BREAKPOINTS.MEDIUM) return 'wide';
  if (width >= BREAKPOINTS.COMPACT) return 'medium';
  if (width >= BREAKPOINTS.MINIMAL) return 'compact';
  return 'minimal';
}

/**
 * Determine column layout from terminal width
 */
function getColumnLayout(width: number): ColumnLayout {
  if (width >= BREAKPOINTS.WIDE) return 'triple';
  if (width >= BREAKPOINTS.COMPACT) return 'dual';
  return 'single';
}

/**
 * Calculate sidebar width based on terminal width
 * Uses responsive scaling with min/max bounds
 */
function calculateSidebarWidth(width: number, mode: LayoutMode): number {
  if (mode === 'minimal' || mode === 'compact') {
    return 0; // No sidebar in narrow modes
  }
  // 25-30% of width, bounded 24-40 cols
  const percent = mode === 'wide' ? 0.25 : 0.28;
  return Math.min(40, Math.max(24, Math.floor(width * percent)));
}

/**
 * Hook for responsive layout management
 *
 * @example
 * ```tsx
 * const { mode, isWide, responsive, flexDirection } = useResponsiveLayout();
 *
 * // Conditional rendering
 * {isWide && <SidePanel />}
 *
 * // Responsive values
 * const maxItems = responsive({ minimal: 5, compact: 10, wide: 20, default: 15 });
 *
 * // Responsive layout direction
 * <Box flexDirection={flexDirection}>
 *   <MainContent />
 *   {canMultiColumn && <Sidebar />}
 * </Box>
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
    const columnLayout = getColumnLayout(width);

    return {
      width,
      height,
      mode,
      columnLayout,
      isMinimal: mode === 'minimal',
      isCompact: mode === 'compact',
      isMedium: mode === 'medium',
      isWide: mode === 'wide',
      canMultiColumn: width >= BREAKPOINTS.COMPACT,
      canTripleColumn: width >= BREAKPOINTS.WIDE,
    };
  }, [width, height]);

  // Responsive value selector
  const responsive = useMemo(() => {
    return <T>(values: ResponsiveValues<T>): T => {
      // Try mode-specific value, then fall back through hierarchy
      switch (state.mode) {
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
    };
  }, [state.mode]);

  // Responsive width calculator
  const responsiveWidth = useMemo(() => {
    return (percent: number, min = 0, max = width): number => {
      const calculated = Math.floor((width * percent) / 100);
      return Math.min(max, Math.max(min, calculated));
    };
  }, [width]);

  // Calculated layout values
  const sidebarWidth = useMemo(
    () => calculateSidebarWidth(width, state.mode),
    [width, state.mode]
  );

  const mainContentWidth = useMemo(
    () => (state.canMultiColumn ? width - sidebarWidth - 2 : width),
    [width, sidebarWidth, state.canMultiColumn]
  );

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
