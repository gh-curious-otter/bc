/**
 * BC CLI service wrapper
 * Executes bc commands and parses JSON responses
 */

import { spawn } from 'child_process';
import type {
  StatusResponse,
  ChannelsResponse,
  ChannelHistory,
  CostSummary,
  Demon,
  DemonRunLog,
  ProcessListResponse,
  TeamsResponse,
} from '../types';

/**
 * Execute a bc command and return the raw output
 * @param args - Command arguments (e.g., ['status', '--json'])
 * @returns Promise resolving to stdout string
 * @throws Error if command fails
 */
export async function execBc(args: string[]): Promise<string> {
  return new Promise((resolve, reject) => {
    // Always add --json flag if not present and command supports it
    const jsonCommands = ['status', 'stats', 'channel', 'cost', 'logs', 'agent', 'process', 'demon', 'team'];
    const hasJsonFlag = args.includes('--json');
    const command = args[0];

    const finalArgs = [...args];
    if (!hasJsonFlag && jsonCommands.includes(command)) {
      finalArgs.push('--json');
    }

    // Use BC_BIN if set, otherwise fall back to 'bc' in PATH
    const bcBin = process.env.BC_BIN || 'bc';
    const bcRoot = process.env.BC_ROOT || process.cwd();

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
        proc.kill();
        reject(new Error(`bc command timed out: ${args.join(' ')}`));
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
        reject(new Error(stderr || `bc command failed with code ${code}`));
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

// Convenience methods for common commands

/**
 * Get current agent status
 */
export async function getStatus(): Promise<StatusResponse> {
  return execBcJson<StatusResponse>(['status']);
}

/**
 * Get list of channels
 * Note: bc channel list --json now returns {channels: [...]} format (PR #589)
 */
export async function getChannels(): Promise<ChannelsResponse> {
  return execBcJson<ChannelsResponse>(['channel', 'list']);
}

/**
 * Get channel message history
 * @param channelName - Name of channel
 * @param _limit - Unused (CLI doesn't support --limit flag)
 */
export async function getChannelHistory(
  channelName: string,
  _limit?: number
): Promise<ChannelHistory> {
  // Note: bc channel history doesn't support --limit flag
  // It returns all messages, TUI should handle limiting display
  const args = ['channel', 'history', channelName];
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
}

/**
 * Get cost summary
 * Note: bc cost show returns text when empty, handle gracefully
 */
export async function getCostSummary(): Promise<CostSummary> {
  try {
    return await execBcJson<CostSummary>(['cost', 'show']);
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
    return await execBcJson<ProcessListResponse>(['process', 'list']);
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
    return await execBcJson<TeamsResponse>(['team', 'list']);
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
}
