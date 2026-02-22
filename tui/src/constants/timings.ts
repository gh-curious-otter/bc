/**
 * Timing constants for intervals, durations, and delays
 *
 * Issue #1597: Centralize magic numbers and constants
 */

/**
 * Polling intervals for data refresh (in milliseconds)
 */
export const POLL_INTERVALS = {
  /** Status polling - fast for real-time updates */
  STATUS: 1000,
  /** Agent list polling */
  AGENTS: 2000,
  /** Channel messages polling */
  CHANNELS: 3000,
  /** Cost data polling - less frequent */
  COSTS: 5000,
  /** Logs polling */
  LOGS: 2000,
  /** Git status polling */
  GIT_STATUS: 5000,
  /** Process list polling */
  PROCESSES: 2000,
  /** Demons list polling */
  DEMONS: 5000,
  /** Default polling interval */
  DEFAULT: 3000,
} as const;

/**
 * UI feedback durations (in milliseconds)
 */
export const DURATIONS = {
  /** How long to show send errors before auto-clearing */
  SEND_ERROR_DISPLAY: 3000,
  /** Toast notification display time */
  TOAST_DISPLAY: 3000,
  /** Animation transition duration */
  ANIMATION: 150,
  /** Debounce delay for search input */
  SEARCH_DEBOUNCE: 300,
  /** Delay before showing loading indicator */
  LOADING_DELAY: 200,
} as const;

/**
 * Performance and rendering constants
 */
export const PERFORMANCE = {
  /** Target frames per second */
  TARGET_FPS: 24,
  /** Target frame time in milliseconds */
  TARGET_FRAME_TIME_MS: 1000 / 24,
  /** Maximum items to render without virtualization */
  VIRTUALIZATION_THRESHOLD: 100,
} as const;

/**
 * Timeout values for operations (in milliseconds)
 */
export const TIMEOUTS = {
  /** Command execution timeout */
  COMMAND: 30000,
  /** Network request timeout */
  REQUEST: 10000,
  /** Short operation timeout */
  SHORT: 5000,
} as const;
