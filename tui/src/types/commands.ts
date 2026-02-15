/**
 * bc Command Registry and Types
 * All bc commands available in TUI with descriptions and categories
 */

export interface BcCommand {
  name: string;           // Full command name (e.g., 'agent list')
  category: string;       // Category for organization
  description: string;    // User-friendly description
  usage: string;          // Command usage (e.g., 'bc agent list')
  readOnly: boolean;      // Safe to execute in TUI
  flags?: string[];       // Common flags
}

export interface CommandCategory {
  name: string;
  commands: BcCommand[];
}

// All bc commands organized by category
export const COMMAND_REGISTRY: CommandCategory[] = [
  {
    name: 'Agent Management',
    commands: [
      {
        name: 'agent status',
        category: 'Agent Management',
        description: 'Show status of all agents in workspace',
        usage: 'bc agent status',
        readOnly: true,
        flags: ['--json', '--verbose'],
      },
      {
        name: 'agent list',
        category: 'Agent Management',
        description: 'List all agents in workspace',
        usage: 'bc agent list',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'agent peek',
        category: 'Agent Management',
        description: 'Show recent output from agent session',
        usage: 'bc agent peek <agent-name>',
        readOnly: true,
        flags: ['--tail', '--json'],
      },
      {
        name: 'agent send',
        category: 'Agent Management',
        description: 'Send message to agent',
        usage: 'bc agent send <agent-name> "<message>"',
        readOnly: false,
      },
    ],
  },
  {
    name: 'Communication',
    commands: [
      {
        name: 'channel list',
        category: 'Communication',
        description: 'List all communication channels',
        usage: 'bc channel list',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'channel send',
        category: 'Communication',
        description: 'Send message to channel',
        usage: 'bc channel send <channel> "<message>"',
        readOnly: false,
      },
      {
        name: 'channel history',
        category: 'Communication',
        description: 'Show message history for channel',
        usage: 'bc channel history <channel>',
        readOnly: true,
        flags: ['--limit', '--json'],
      },
      {
        name: 'channel join',
        category: 'Communication',
        description: 'Join a communication channel',
        usage: 'bc channel join <channel>',
        readOnly: false,
      },
    ],
  },
  {
    name: 'Tracking & Monitoring',
    commands: [
      {
        name: 'cost show',
        category: 'Tracking & Monitoring',
        description: 'Show current cost information',
        usage: 'bc cost show',
        readOnly: true,
        flags: ['--agent', '--json'],
      },
      {
        name: 'stats',
        category: 'Tracking & Monitoring',
        description: 'Show workspace statistics',
        usage: 'bc stats',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'logs',
        category: 'Tracking & Monitoring',
        description: 'Show event logs',
        usage: 'bc logs',
        readOnly: true,
        flags: ['--agent', '--tail', '--json'],
      },
      {
        name: 'status',
        category: 'Tracking & Monitoring',
        description: 'Show overall workspace status',
        usage: 'bc status',
        readOnly: true,
        flags: ['--json'],
      },
    ],
  },
  {
    name: 'Configuration',
    commands: [
      {
        name: 'config show',
        category: 'Configuration',
        description: 'Show workspace configuration',
        usage: 'bc config show',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'role list',
        category: 'Configuration',
        description: 'List available agent roles',
        usage: 'bc role list',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'role create',
        category: 'Configuration',
        description: 'Create a new agent role',
        usage: 'bc role create <role-name>',
        readOnly: false,
      },
      {
        name: 'team list',
        category: 'Configuration',
        description: 'List agent teams',
        usage: 'bc team list',
        readOnly: true,
        flags: ['--json'],
      },
    ],
  },
  {
    name: 'Process Management',
    commands: [
      {
        name: 'up',
        category: 'Process Management',
        description: 'Start bc agents',
        usage: 'bc up',
        readOnly: false,
      },
      {
        name: 'down',
        category: 'Process Management',
        description: 'Stop bc agents',
        usage: 'bc down',
        readOnly: false,
      },
      {
        name: 'process list',
        category: 'Process Management',
        description: 'List background processes',
        usage: 'bc process list',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'demon list',
        category: 'Process Management',
        description: 'List scheduled tasks (demons)',
        usage: 'bc demon list',
        readOnly: true,
        flags: ['--json'],
      },
    ],
  },
  {
    name: 'Memory & Learning',
    commands: [
      {
        name: 'memory show',
        category: 'Memory & Learning',
        description: 'Show agent learnings and experiences',
        usage: 'bc memory show',
        readOnly: true,
        flags: ['--agent', '--json'],
      },
      {
        name: 'memory search',
        category: 'Memory & Learning',
        description: 'Search agent memory',
        usage: 'bc memory search <keyword>',
        readOnly: true,
        flags: ['--agent'],
      },
      {
        name: 'memory learn',
        category: 'Memory & Learning',
        description: 'Record learning to agent memory',
        usage: 'bc memory learn <category> <content>',
        readOnly: false,
      },
    ],
  },
  {
    name: 'Utilities',
    commands: [
      {
        name: 'version',
        category: 'Utilities',
        description: 'Show bc version information',
        usage: 'bc version',
        readOnly: true,
        flags: ['--json'],
      },
      {
        name: 'help',
        category: 'Utilities',
        description: 'Show help for bc commands',
        usage: 'bc help [command]',
        readOnly: true,
      },
      {
        name: 'init',
        category: 'Utilities',
        description: 'Initialize a new bc workspace',
        usage: 'bc init <workspace-name>',
        readOnly: false,
      },
    ],
  },
];

/**
 * Get all commands (flattened)
 */
export function getAllCommands(): BcCommand[] {
  return COMMAND_REGISTRY.flatMap(cat => cat.commands);
}

/**
 * Filter commands by search query
 */
export function searchCommands(query: string): BcCommand[] {
  const lowerQuery = query.toLowerCase();
  return getAllCommands().filter(cmd =>
    cmd.name.toLowerCase().includes(lowerQuery) ||
    cmd.description.toLowerCase().includes(lowerQuery) ||
    cmd.category.toLowerCase().includes(lowerQuery)
  );
}

/**
 * Get commands by category
 */
export function getCommandsByCategory(category: string): BcCommand[] {
  return COMMAND_REGISTRY.find(cat => cat.name === category)?.commands || [];
}
