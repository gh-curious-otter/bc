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

/**
 * Terminal fallback dimensions
 * Used when stdout dimensions are unavailable
 */
export const TERMINAL_DEFAULTS = {
  /** Default terminal rows (standard 80x24) */
  ROWS: 24,
  /** Default terminal columns (standard 80x24) */
  COLS: 80,
  /** Minimum usable height for views */
  MIN_VIEW_HEIGHT: 10,
  /** Reserved lines for header/footer/hints */
  RESERVED_LINES: 6,
} as const;

/**
 * UI element dimensions
 * For dividers, separators, and decorative elements
 */
export const UI_ELEMENTS = {
  /** Standard divider width */
  DIVIDER_WIDTH: 40,
  /** Narrow divider width (for compact views) */
  DIVIDER_WIDTH_NARROW: 30,
  /** Wide divider width (for full-width views) */
  DIVIDER_WIDTH_WIDE: 50,
  /** Command palette width */
  COMMAND_PALETTE_WIDTH: 60,
  /** Command palette minimum margin */
  COMMAND_PALETTE_MIN_MARGIN: 4,
} as const;

/**
 * Data fetch limits
 * For tail, pagination, and batch sizes
 */
export const DATA_LIMITS = {
  /** Default log tail limit */
  LOG_TAIL: 100,
  /** Activity feed tail limit */
  ACTIVITY_TAIL: 50,
  /** Process output lines */
  PROCESS_LINES: 50,
  /** Maximum file size to preview (100KB) */
  MAX_PREVIEW_SIZE: 100 * 1024,
} as const;
