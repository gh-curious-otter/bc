/**
 * Dimension constants for layout, sizing, and breakpoints
 *
 * Issue #1597: Centralize magic numbers and constants
 */

/**
 * Terminal width breakpoints for responsive layouts
 */
export const BREAKPOINTS = {
  /** Extra small terminals */
  XS: 60,
  /** Small terminals (minimal mode) */
  SMALL: 80,
  /** Compact mode threshold */
  COMPACT: 100,
  /** Medium terminals (short labels) */
  MEDIUM: 120,
  /** Large terminals (full layout) */
  LARGE: 160,
} as const;

/**
 * Input field dimensions
 */
export const INPUT_DIMENSIONS = {
  /** Minimum input height in lines */
  MIN_HEIGHT: 3,
  /** Maximum input height in lines */
  MAX_HEIGHT: 10,
  /** Default input width */
  DEFAULT_WIDTH: 60,
} as const;

/**
 * Pane and panel dimensions
 */
export const PANE_DIMENSIONS = {
  /** Detail pane width */
  DETAIL_PANE_WIDTH: 30,
  /** Minimum detail pane width */
  DETAIL_PANE_MIN_WIDTH: 25,
  /** Maximum detail pane width */
  DETAIL_PANE_MAX_WIDTH: 50,
  /** Drawer width - full mode */
  DRAWER_FULL_WIDTH: 20,
  /** Drawer width - shrunk mode */
  DRAWER_SHRUNK_WIDTH: 4,
} as const;

/**
 * Message bubble dimensions
 */
export const BUBBLE_DIMENSIONS = {
  /** Minimum bubble width */
  MIN_WIDTH: 50,
  /** Maximum bubble width */
  MAX_WIDTH: 140,
  /** Bubble width as percentage of terminal */
  WIDTH_PERCENTAGE: 0.8,
  /** Maximum lines before truncation */
  MAX_LINES: 8,
} as const;

/**
 * Activity feed column widths
 */
export const ACTIVITY_FEED_WIDTHS = {
  /** Timestamp column (HH:MM:SS + space) */
  TIMESTAMP: 9,
  /** Agent name column */
  AGENT: 11,
  /** Icon column (icon + space) */
  ICON: 2,
  /** Event type column */
  EVENT: 13,
  /** Count column ((x99) + space) */
  COUNT: 6,
  /** Minimum message width */
  MIN_MESSAGE: 20,
} as const;

/**
 * Table and list dimensions
 */
export const TABLE_DIMENSIONS = {
  /** Default row height */
  ROW_HEIGHT: 1,
  /** Header height */
  HEADER_HEIGHT: 2,
  /** Minimum column width */
  MIN_COLUMN_WIDTH: 8,
  /** Maximum items visible without scroll */
  MAX_VISIBLE_ITEMS: 20,
} as const;

/**
 * Margin and padding values
 */
export const SPACING = {
  /** Small spacing */
  XS: 1,
  /** Medium spacing */
  SM: 2,
  /** Default spacing */
  MD: 4,
  /** Large spacing */
  LG: 8,
} as const;
