/**
 * Color constants for consistent theming
 *
 * Issue #1598: Remove duplicated code
 */

/**
 * Role-based colors for agent/user identification
 * Used in chat messages, agent lists, status displays
 */
export const ROLE_COLORS: Record<string, string> = {
  // System roles
  root: 'magenta',
  cli: 'gray',
  system: 'gray',

  // Engineering roles
  engineer: 'green',
  'tech-lead': 'cyan',

  // Management roles
  manager: 'yellow',
  pm: 'yellow',

  // Specialist roles
  ux: 'blue',
  qa: 'red',

  // Default
  default: 'white',
} as const;

/**
 * Role prefix patterns for matching sender names
 * Maps name prefixes to role keys
 */
export const ROLE_PREFIXES: { prefix: string; role: string }[] = [
  { prefix: 'root', role: 'root' },
  { prefix: 'tech-lead', role: 'tech-lead' },
  { prefix: 'tl-', role: 'tech-lead' },
  { prefix: 'eng-', role: 'engineer' },
  { prefix: 'mgr-', role: 'manager' },
  { prefix: 'pm-', role: 'pm' },
  { prefix: 'ux-', role: 'ux' },
  { prefix: 'qa-', role: 'qa' },
  { prefix: 'cli', role: 'cli' },
  { prefix: 'system', role: 'system' },
];

/**
 * Role emoji prefixes for visual distinction
 */
export const ROLE_EMOJIS: Record<string, string> = {
  root: '⚙ ',
  'tech-lead': '🔧 ',
  engineer: '💻 ',
  manager: '📋 ',
  pm: '📊 ',
  ux: '🎨 ',
  qa: '🧪 ',
  cli: '⌨ ',
  system: '',
  default: '',
} as const;

/**
 * Get color for a sender/agent name based on role prefix matching
 *
 * @param name - Sender or agent name (e.g., "eng-01", "root")
 * @returns Color string for the role
 */
export function getColorForName(name: string): string {
  // Check exact matches first
  if (ROLE_COLORS[name]) {
    return ROLE_COLORS[name];
  }

  // Check prefix patterns
  for (const { prefix, role } of ROLE_PREFIXES) {
    if (name === prefix || name.startsWith(prefix)) {
      return ROLE_COLORS[role] || ROLE_COLORS.default;
    }
  }

  return ROLE_COLORS.default;
}

/**
 * Get emoji prefix for a sender/agent name
 *
 * @param name - Sender or agent name
 * @returns Emoji prefix string (may be empty)
 */
export function getEmojiForName(name: string): string {
  // Check exact matches first
  if (ROLE_EMOJIS[name]) {
    return ROLE_EMOJIS[name];
  }

  // Check prefix patterns
  for (const { prefix, role } of ROLE_PREFIXES) {
    if (name === prefix || name.startsWith(prefix)) {
      return ROLE_EMOJIS[role] || ROLE_EMOJIS.default;
    }
  }

  return ROLE_EMOJIS.default;
}

/**
 * Get role key for a sender/agent name
 *
 * @param name - Sender or agent name
 * @returns Role key (e.g., "engineer", "manager")
 */
export function getRoleFromName(name: string): string {
  for (const { prefix, role } of ROLE_PREFIXES) {
    if (name === prefix || name.startsWith(prefix)) {
      return role;
    }
  }
  return 'default';
}
