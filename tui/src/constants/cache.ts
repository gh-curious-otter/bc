/**
 * Cache configuration constants
 *
 * Issue #1597: Centralize magic numbers and constants
 */

/**
 * Cache TTL values by data type (in milliseconds)
 * Shorter TTLs for frequently-changing data, longer for stable data
 */
export const CACHE_TTLS = {
  /** Status data - very short, changes frequently */
  STATUS: 1000,
  /** Agent list - short */
  AGENTS: 2000,
  /** Agent state - short */
  AGENT_STATE: 2000,
  /** Channel list - medium */
  CHANNELS: 5000,
  /** Channel messages - short for real-time feel */
  CHANNEL_MESSAGES: 2000,
  /** Cost data - medium, doesn't change rapidly */
  COSTS: 10000,
  /** Logs - short */
  LOGS: 2000,
  /** Roles - long, rarely changes */
  ROLES: 30000,
  /** Commands - long, rarely changes */
  COMMANDS: 60000,
  /** Config - long, rarely changes */
  CONFIG: 60000,
  /** Git status - medium */
  GIT_STATUS: 2000,
  /** Processes - short */
  PROCESSES: 2000,
  /** Demons - medium */
  DEMONS: 5000,
  /** Routing rules - long */
  ROUTING: 30000,
  /** Teams - medium */
  TEAMS: 10000,
  /** Worktrees - long, rarely changes */
  WORKTREES: 30000,
  /** Default TTL for unspecified types */
  DEFAULT: 5000,
} as const;

/**
 * Cache size limits
 */
export const CACHE_LIMITS = {
  /** Maximum cache entries */
  MAX_ENTRIES: 100,
  /** Maximum cache size in bytes (approximate) */
  MAX_SIZE_BYTES: 10 * 1024 * 1024, // 10MB
  /** Maximum age before forced cleanup (in ms) */
  MAX_AGE: 5 * 60 * 1000, // 5 minutes
} as const;

/**
 * Cache keys prefix constants
 */
export const CACHE_KEYS = {
  STATUS: 'status',
  AGENTS: 'agents',
  AGENT: 'agent',
  CHANNELS: 'channels',
  CHANNEL: 'channel',
  COSTS: 'costs',
  LOGS: 'logs',
  ROLES: 'roles',
  COMMANDS: 'commands',
  CONFIG: 'config',
  PROCESSES: 'processes',
  DEMONS: 'demons',
  ROUTING: 'routing',
  TEAMS: 'teams',
  WORKTREES: 'worktrees',
} as const;
