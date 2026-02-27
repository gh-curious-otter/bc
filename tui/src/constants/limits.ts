/**
 * Display limits and truncation constants for UI elements
 *
 * Issue #1779: Extract hardcoded UI limits to configurable constants
 */

/**
 * Text truncation limits for display elements
 */
export const TRUNCATION = {
  /** Agent/process/demon name truncation length */
  NAME_SHORT: 12,
  /** Role/command name truncation length */
  NAME_MEDIUM: 22,
  /** Command name truncation length */
  COMMAND_NAME: 25,
  /** Description truncation length */
  DESCRIPTION: 45,
  /** Message/content truncation in lists */
  MESSAGE: 70,
  /** Long content preview truncation */
  PREVIEW: 100,
  /** Prompt preview truncation */
  PROMPT_PREVIEW: 200,
  /** Issue body preview truncation */
  ISSUE_BODY: 500,
} as const;

/**
 * Maximum items to display in lists before showing "and N more"
 */
export const DISPLAY_LIMITS = {
  /** Experiences shown in memory detail view */
  EXPERIENCES: 10,
  /** Search results shown in memory search */
  SEARCH_RESULTS: 15,
  /** Comments shown in issue detail */
  ISSUE_COMMENTS: 3,
  /** Capabilities shown in role row preview */
  CAPABILITIES_PREVIEW: 3,
  /** Top roles shown in dashboard */
  TOP_ROLES: 3,
  /** Recent activity events shown in agent detail */
  RECENT_ACTIVITY: 8,
  /** Orphaned worktrees shown in warning */
  ORPHANED_WORKTREES: 5,
  /** Performance metrics shown in dashboard */
  TOP_METRICS: 10,
} as const;

/**
 * Column width constants for table layouts
 */
export const COLUMN_WIDTHS = {
  /** Selection indicator width (▸ + space) */
  SELECTION: 3,
  /** Timestamp column width */
  TIMESTAMP: 9,
  /** Short timestamp (HH:MM:SS) */
  TIMESTAMP_SHORT: 8,
  /** Date-inclusive timestamp (MM/DD HH:MM) */
  TIMESTAMP_DATE: 12,
  /** Agent name column */
  AGENT_NAME: 12,
  /** Role name column */
  ROLE_NAME: 15,
  /** Status column */
  STATUS: 9,
  /** PID column */
  PID: 7,
  /** Port column */
  PORT: 6,
  /** Uptime column */
  UPTIME: 8,
  /** Short command preview */
  COMMAND_SHORT: 20,
  /** Full command preview */
  COMMAND_FULL: 22,
} as const;
