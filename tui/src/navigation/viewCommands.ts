/**
 * View command registry for k9s-style :command navigation
 */

import type { View } from './NavigationContext';

export interface ViewCommand {
  command: string;
  aliases: string[];
  view: View;
  section: string;
}

export const VIEW_COMMANDS: ViewCommand[] = [
  { command: 'dashboard', aliases: ['dash', 'd'], view: 'dashboard', section: 'CORE' },
  { command: 'agents', aliases: ['ag', 'a'], view: 'agents', section: 'CORE' },
  { command: 'channels', aliases: ['ch', 'c'], view: 'channels', section: 'CORE' },
  { command: 'costs', aliases: ['co', 'cost'], view: 'costs', section: 'CORE' },
  { command: 'logs', aliases: ['log', 'l'], view: 'logs', section: 'CORE' },
  { command: 'roles', aliases: ['ro', 'r'], view: 'roles', section: 'SYSTEM' },
  { command: 'worktrees', aliases: ['wt', 'w'], view: 'worktrees', section: 'SYSTEM' },
  { command: 'help', aliases: ['?', 'h'], view: 'help', section: 'CORE' },
];

/**
 * Fuzzy match scoring - returns 0 for no match, higher = better match
 */
function fuzzyScore(query: string, target: string): number {
  const q = query.toLowerCase();
  const t = target.toLowerCase();

  // Exact match
  if (t === q) return 100;

  // Starts with
  if (t.startsWith(q)) return 80;

  // Contains
  if (t.includes(q)) return 60;

  // Fuzzy character match
  let qi = 0;
  let score = 0;
  for (let ti = 0; ti < t.length && qi < q.length; ti++) {
    if (t[ti] === q[qi]) {
      score += 10;
      qi++;
    }
  }
  return qi === q.length ? score : 0;
}

export interface MatchedCommand {
  command: ViewCommand;
  score: number;
}

/**
 * Search view commands with fuzzy matching
 */
export function searchCommands(query: string): MatchedCommand[] {
  if (!query) {
    return VIEW_COMMANDS.map(cmd => ({ command: cmd, score: 50 }));
  }

  const results: MatchedCommand[] = [];

  for (const cmd of VIEW_COMMANDS) {
    // Check command name
    let bestScore = fuzzyScore(query, cmd.command);

    // Check aliases
    for (const alias of cmd.aliases) {
      const aliasScore = fuzzyScore(query, alias);
      if (aliasScore > bestScore) {
        bestScore = aliasScore;
      }
    }

    if (bestScore > 0) {
      results.push({ command: cmd, score: bestScore });
    }
  }

  return results.sort((a, b) => b.score - a.score);
}

/**
 * Resolve a command string directly to a view (exact match or alias)
 */
export function resolveCommand(input: string): View | null {
  const q = input.toLowerCase().trim();
  for (const cmd of VIEW_COMMANDS) {
    if (cmd.command === q || cmd.aliases.includes(q)) {
      return cmd.view;
    }
  }
  return null;
}
