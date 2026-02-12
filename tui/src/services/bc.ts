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
    const jsonCommands = ['status', 'stats', 'channel', 'cost', 'logs', 'agent'];
    const hasJsonFlag = args.includes('--json');
    const command = args[0];

    const finalArgs = [...args];
    if (!hasJsonFlag && jsonCommands.includes(command)) {
      finalArgs.push('--json');
    }

    const proc = spawn('bc', finalArgs, {
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';

    proc.stdout.on('data', (data: Buffer) => {
      stdout += data.toString();
    });

    proc.stderr.on('data', (data: Buffer) => {
      stderr += data.toString();
    });

    proc.on('close', (code: number | null) => {
      if (code === 0) {
        resolve(stdout.trim());
      } else {
        reject(new Error(stderr || `bc command failed with code ${code}`));
      }
    });

    proc.on('error', (err: Error) => {
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
 */
export async function getChannels(): Promise<ChannelsResponse> {
  return execBcJson<ChannelsResponse>(['channel', 'list']);
}

/**
 * Get channel message history
 * @param channelName - Name of channel
 * @param limit - Max messages to return (optional)
 */
export async function getChannelHistory(
  channelName: string,
  limit?: number
): Promise<ChannelHistory> {
  const args = ['channel', 'history', channelName];
  if (limit) {
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
}

/**
 * Get cost summary
 */
export async function getCostSummary(): Promise<CostSummary> {
  return execBcJson<CostSummary>(['cost', 'show']);
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
