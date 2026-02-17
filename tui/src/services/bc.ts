/**
 * BC CLI service wrapper
 * Executes bc commands and parses JSON responses
 *
 * #1005: Added command result caching with stale-while-revalidate pattern
 * to reduce subprocess overhead from polling operations.
 */

import { spawn, spawnSync } from 'child_process';
import type {
  StatusResponse,
  ChannelsResponse,
  ChannelHistory,
  CostSummary,
  Demon,
  DemonRunLog,
  ProcessListResponse,
  TeamsResponse,
  LogEntry,
  Role,
  RolesResponse,
  Worktree,
  WorkspacesResponse,
} from '../types';

// ============================================================================
// Command Result Caching (#1005 - Performance Epic Phase 3)
// ============================================================================

/**
 * Cache entry with timestamp and TTL
 */
interface CacheEntry<T> {
  data: T;
  timestamp: number;
  ttl: number;
}

/**
 * In-memory cache for command results
 */
const commandCache = new Map<string, CacheEntry<unknown>>();

/**
 * Default TTLs by command type (in milliseconds)
 * Shorter TTLs for frequently-changing data, longer for stable data
 */
const DEFAULT_TTLS: Record<string, number> = {
  status: 1000,      // 1s - agent status changes frequently
  'channel:list': 5000,   // 5s - channel list rarely changes
  'channel:history': 2000, // 2s - messages may arrive
  'cost:show': 10000,     // 10s - aggregated data
  'role:list': 30000,     // 30s - roles rarely change
  'team:list': 10000,     // 10s - team membership stable
  'process:list': 5000,   // 5s - processes may change
  'demon:list': 10000,    // 10s - demons stable
  'logs': 2000,           // 2s - logs update frequently
  'worktree:list': 30000, // 30s - worktrees stable
  'workspace:list': 60000, // 60s - workspaces very stable
};

/**
 * Generate cache key from command arguments
 */
function getCacheKey(args: string[]): string {
  return args.join(':');
}

/**
 * Get cached result if valid, otherwise return null
 */
function getCachedResult<T>(key: string): T | null {
  const entry = commandCache.get(key);
  if (!entry) return null;

  const now = Date.now();
  if (now - entry.timestamp < entry.ttl) {
    return entry.data as T;
  }

  // Expired - remove from cache
  commandCache.delete(key);
  return null;
}

/**
 * Store result in cache with TTL
 */
function setCachedResult<T>(key: string, data: T, ttl: number): void {
  commandCache.set(key, {
    data,
    timestamp: Date.now(),
    ttl,
  });
}

/**
 * Invalidate cache entries matching a prefix
 * Called after write operations to ensure fresh data
 */
export function invalidateCache(prefix?: string): void {
  if (!prefix) {
    commandCache.clear();
    return;
  }

  for (const key of commandCache.keys()) {
    if (key.startsWith(prefix)) {
      commandCache.delete(key);
    }
  }
}

/**
 * Clear all cached command results
 * Exported for testing purposes
 */
export function clearCache(): void {
  commandCache.clear();
}

/**
 * Get TTL for a command based on its type
 */
function getTtlForCommand(args: string[]): number {
  // Try specific command key first (e.g., "channel:list")
  const specificKey = args.slice(0, 2).join(':');
  if (DEFAULT_TTLS[specificKey]) {
    return DEFAULT_TTLS[specificKey];
  }

  // Fall back to command type (e.g., "status")
  const command = args[0];
  return DEFAULT_TTLS[command] ?? 5000; // Default 5s
}

// ============================================================================

/**
 * Execute a bc command and return the raw output
 * @param args - Command arguments (e.g., ['status', '--json'])
 * @returns Promise resolving to stdout string
 * @throws Error if command fails
 */
export async function execBc(args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    // Always add --json flag if not present and command supports it
    const jsonCommands = ['status', 'stats', 'channel', 'cost', 'logs', 'agent', 'process', 'demon', 'team', 'role', 'worktree'];
    const hasJsonFlag = args.includes('--json');
    const command = args[0];

    const finalArgs = [...args];
    if (!hasJsonFlag && jsonCommands.includes(command)) {
      finalArgs.push('--json');
    }

    // Use BC_BIN if set, otherwise fall back to 'bc' in PATH
    const bcBin = process.env.BC_BIN ?? 'bc';
    const bcRoot = process.env.BC_ROOT ?? process.cwd();

    const proc = spawn(bcBin, finalArgs, {
      stdio: ['ignore', 'pipe', 'pipe'],
      cwd: bcRoot,
    });

    let stdout = '';
    let stderr = '';
    let finished = false;

    // Timeout after 30 seconds
    const timeout = setTimeout(() => {
      if (!finished) {
        finished = true;
        // Kill the process forcefully to ensure cleanup
        proc.kill('SIGKILL');
        clearTimeout(timeout);
        reject(new Error(`bc command timed out after 30s: ${args.join(' ')}`));
      }
    }, 30000);

    proc.stdout.on('data', (data: Buffer) => {
      stdout += data.toString();
    });

    proc.stderr.on('data', (data: Buffer) => {
      stderr += data.toString();
    });

    proc.on('close', (code: number | null) => {
      if (finished) return;
      finished = true;
      clearTimeout(timeout);
      if (code === 0) {
        resolve(stdout.trim());
      } else {
        reject(new Error(stderr || `bc command failed with code ${String(code ?? 'unknown')}`));
      }
    });

    proc.on('error', (err: Error) => {
      if (finished) return;
      finished = true;
      clearTimeout(timeout);
      reject(new Error(`Failed to spawn bc: ${err.message}`));
    });
  });
}

/**
 * Execute bc command and parse JSON response
 * @param args - Command arguments
 * @returns Parsed JSON response
 */
export async function execBcJson<T>(args: string[]): Promise<T> {
  const output = await execBc(args);
  try {
    return JSON.parse(output) as T;
  } catch {
    throw new Error(`Failed to parse bc output as JSON: ${output.slice(0, 100)}`);
  }
}

/**
 * Execute bc command with caching (stale-while-revalidate pattern)
 * #1005: Reduces subprocess overhead for polling operations
 *
 * @param args - Command arguments
 * @param ttl - Optional TTL override (uses default based on command type)
 * @returns Cached or fresh JSON response
 */
export async function execBcJsonCached<T>(args: string[], ttl?: number): Promise<T> {
  const key = getCacheKey(args);

  // Check cache first
  const cached = getCachedResult<T>(key);
  if (cached !== null) {
    return cached;
  }

  // Cache miss - fetch fresh data
  const data = await execBcJson<T>(args);
  const effectiveTtl = ttl ?? getTtlForCommand(args);
  setCachedResult(key, data, effectiveTtl);

  return data;
}

// Convenience methods for common commands

/**
 * Get current agent status
 */
export async function getStatus(): Promise<StatusResponse> {
  // #1005: Use cached version to reduce polling overhead
  return execBcJsonCached<StatusResponse>(['status']);
}

/**
 * Get list of channels
 * Note: bc channel list --json now returns {channels: [...]} format (PR #589)
 */
export async function getChannels(): Promise<ChannelsResponse> {
  // #1005: Use cached version to reduce polling overhead
  return execBcJsonCached<ChannelsResponse>(['channel', 'list']);
}

/**
 * Get channel message history
 * @param channelName - Name of channel
 * @param limit - Maximum number of messages to return (default: 50)
 */
export async function getChannelHistory(
  channelName: string,
  limit?: number
): Promise<ChannelHistory> {
  const args = ['channel', 'history', channelName];
  if (limit !== undefined && limit > 0) {
    args.push('--limit', String(limit));
  }
  return execBcJson<ChannelHistory>(args);
}

/**
 * Send message to channel
 * @param channelName - Name of channel
 * @param message - Message to send
 */
export async function sendChannelMessage(
  channelName: string,
  message: string
): Promise<void> {
  await execBc(['channel', 'send', channelName, message]);
  // #1005: Invalidate channel cache after sending message
  invalidateCache('channel');
}

/**
 * Get cost summary
 * Note: bc cost show returns text when empty, handle gracefully
 */
export async function getCostSummary(): Promise<CostSummary> {
  try {
    // #1005: Use cached version to reduce polling overhead
    return await execBcJsonCached<CostSummary>(['cost', 'show']);
  } catch {
    // Return empty cost summary when no records exist
    return {
      total_cost: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      by_agent: {},
      by_team: {},
      by_model: {},
    };
  }
}

/**
 * Report agent state
 * @param state - New state (working, done, stuck, idle, error)
 * @param message - Status message
 */
export async function reportState(
  state: string,
  message: string
): Promise<void> {
  await execBc(['report', state, message]);
  // #1005: Invalidate status cache after state change
  invalidateCache('status');
}

/**
 * Get list of demons (scheduled tasks)
 */
export async function getDemons(): Promise<Demon[]> {
  try {
    return await execBcJson<Demon[]>(['demon', 'list']);
  } catch {
    // If no demons exist, bc returns text not JSON
    return [];
  }
}

/**
 * Get demon details
 * @param name - Demon name
 */
export async function getDemon(name: string): Promise<Demon | null> {
  try {
    return await execBcJson<Demon>(['demon', 'show', name]);
  } catch {
    return null;
  }
}

/**
 * Get demon run logs
 * @param name - Demon name
 * @param tail - Number of recent entries (optional)
 */
export async function getDemonLogs(
  name: string,
  tail?: number
): Promise<DemonRunLog[]> {
  try {
    const args = ['demon', 'logs', name];
    if (tail) {
      args.push('--tail', String(tail));
    }
    return await execBcJson<DemonRunLog[]>(args);
  } catch {
    return [];
  }
}

/**
 * Enable a demon
 * @param name - Demon name
 */
export async function enableDemon(name: string): Promise<void> {
  await execBc(['demon', 'enable', name]);
}

/**
 * Disable a demon
 * @param name - Demon name
 */
export async function disableDemon(name: string): Promise<void> {
  await execBc(['demon', 'disable', name]);
}

/**
 * Manually run a demon
 * @param name - Demon name
 */
export async function runDemon(name: string): Promise<void> {
  await execBc(['demon', 'run', name]);
}

/**
 * Get list of managed processes
 * Note: bc process list returns text when empty, handle gracefully
 */
export async function getProcesses(): Promise<ProcessListResponse> {
  try {
    // #1005: Use cached version to reduce polling overhead
    return await execBcJsonCached<ProcessListResponse>(['process', 'list']);
  } catch {
    return { processes: [] };
  }
}

/**
 * Get logs for a specific process
 * @param name - Process name
 * @param lines - Number of lines to return (optional)
 */
export async function getProcessLogs(
  name: string,
  lines?: number
): Promise<string[]> {
  const args = ['process', 'logs', name];
  if (lines) {
    args.push('--lines', String(lines));
  }
  const response = await execBcJson<{ name: string; lines: string[] }>(args);
  return response.lines;
}

/**
 * Get list of teams
 * Note: bc team list returns text when empty, handle gracefully
 */
export async function getTeams(): Promise<TeamsResponse> {
  try {
    // #1005: Use cached version to reduce polling overhead
    return await execBcJsonCached<TeamsResponse>(['team', 'list']);
  } catch {
    return { teams: [] };
  }
}

/**
 * Add a member to a team
 * @param teamName - Name of team
 * @param agentName - Name of agent to add
 */
export async function addTeamMember(
  teamName: string,
  agentName: string
): Promise<void> {
  await execBc(['team', 'add', teamName, agentName]);
  // #1005: Invalidate team cache after modification
  invalidateCache('team');
}

/**
 * Remove a member from a team
 * @param teamName - Name of team
 * @param agentName - Name of agent to remove
 */
export async function removeTeamMember(
  teamName: string,
  agentName: string
): Promise<void> {
  await execBc(['team', 'remove', teamName, agentName]);
  // #1005: Invalidate team cache after modification
  invalidateCache('team');
}

/**
 * Get event logs
 * @param tail - Number of recent entries (optional, default 50)
 * @param agent - Filter by agent name (optional)
 * @param eventType - Filter by event type (optional)
 */
export async function getLogs(
  tail?: number,
  agent?: string,
  eventType?: string
): Promise<LogEntry[]> {
  try {
    const args = ['logs'];
    if (tail) {
      args.push('--tail', String(tail));
    }
    if (agent) {
      args.push('--agent', agent);
    }
    if (eventType) {
      args.push('--type', eventType);
    }
    return await execBcJson<LogEntry[]>(args);
  } catch {
    return [];
  }
}

/**
 * Get list of worktrees
 * @param orphanedOnly - Only show orphaned worktrees
 */
export async function getWorktrees(orphanedOnly = false): Promise<Worktree[]> {
  try {
    const args = ['worktree', 'list'];
    if (orphanedOnly) {
      args.push('--orphaned');
    }
    return await execBcJson<Worktree[]>(args);
  } catch {
    return [];
  }
}

/**
 * Prune orphaned worktrees
 * @param force - Actually remove (vs dry run)
 */
export async function pruneWorktrees(force = false): Promise<string> {
  const args = ['worktree', 'prune'];
  if (force) {
    args.push('--force');
  }
  return execBc(args);
}

/**
 * Attach to an agent's tmux session
 * @param sessionName - Tmux session name for the agent
 * @throws Error if session doesn't exist or attachment fails
 */
export function attachToAgentSession(sessionName: string): void {
  // Use spawnSync to attach to tmux session with full stdio inheritance
  // This will replace the current process with the tmux session
  spawnSync('tmux', ['attach-session', '-t', sessionName], {
    stdio: 'inherit',
  });
  // Exit after attachment ends
  process.exit(0);
}

/**
 * Get list of roles
 */
export async function getRoles(): Promise<RolesResponse> {
  try {
    // #1005: Use cached version to reduce polling overhead
    return await execBcJsonCached<RolesResponse>(['role', 'list']);
  } catch {
    return { roles: [] };
  }
}

/**
 * Get role details
 * @param name - Role name
 */
export async function getRole(name: string): Promise<Role | null> {
  try {
    return await execBcJson<Role>(['role', 'show', name]);
  } catch {
    return null;
  }
}

/**
 * Delete a role
 * @param name - Role name
 */
export async function deleteRole(name: string): Promise<void> {
  await execBc(['role', 'delete', name]);
}

/**
 * Validate all role files
 */
export async function validateRoles(): Promise<string> {
  return await execBc(['role', 'validate']);
}

/**
 * Get list of discovered workspaces
 * @param scanPaths - Additional paths to scan
 */
export async function getWorkspaces(scanPaths?: string[]): Promise<WorkspacesResponse> {
  try {
    const args = ['workspace', 'list'];
    if (scanPaths) {
      for (const path of scanPaths) {
        args.push('--scan', path);
      }
    }
    return await execBcJson<WorkspacesResponse>(args);
  } catch {
    return { workspaces: [] };
  }
}
