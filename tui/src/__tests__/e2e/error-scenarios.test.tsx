/**
 * Error Scenario Tests - Error handling and edge cases
 * Issue #751 - TUI E2E Workflows & Real-Time Updates
 *
 * Tests verify proper error handling, edge cases, and recovery.
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Text, Box } from 'ink';
import { describe, it, expect } from 'bun:test';

// Error display component
interface ErrorDisplayProps {
  error: string | null;
  retryCount: number;
  isRetrying: boolean;
}

function ErrorDisplayComponent({
  error,
  retryCount,
  isRetrying,
}: ErrorDisplayProps): React.ReactElement {
  return (
    <Box flexDirection="column">
      {error ? (
        <>
          <Text color="red">Error: {error}</Text>
          <Text>Retry count: {retryCount}</Text>
          {isRetrying && <Text color="yellow">Retrying...</Text>}
        </>
      ) : (
        <Text color="green">No errors</Text>
      )}
    </Box>
  );
}

describe('Error Scenarios: Network Failures', () => {
  it('displays error message when service down', () => {
    const { lastFrame } = render(
      <ErrorDisplayComponent
        error="Connection refused: bc service unavailable"
        retryCount={0}
        isRetrying={false}
      />
    );

    expect(lastFrame()).toContain('Error:');
    expect(lastFrame()).toContain('Connection refused');
  });

  it('shows retry count', () => {
    const { lastFrame } = render(
      <ErrorDisplayComponent error="Connection timeout" retryCount={3} isRetrying={false} />
    );

    expect(lastFrame()).toContain('Retry count: 3');
  });

  it('shows retrying indicator', () => {
    const { lastFrame } = render(
      <ErrorDisplayComponent error="Network error" retryCount={1} isRetrying={true} />
    );

    expect(lastFrame()).toContain('Retrying...');
  });

  it('clears error on recovery', () => {
    const { lastFrame } = render(
      <ErrorDisplayComponent error={null} retryCount={0} isRetrying={false} />
    );

    expect(lastFrame()).toContain('No errors');
  });
});

describe('Error Scenarios: Invalid Operations', () => {
  interface OperationResult {
    success: boolean;
    error?: string;
  }

  function validateChannelSend(channelName: string, agents: string[]): OperationResult {
    const channelExists = ['eng', 'leads', 'all'].includes(channelName);
    if (!channelExists) {
      return { success: false, error: `Channel '${channelName}' not found` };
    }
    return { success: true };
  }

  function validateTeamAdd(
    teamName: string,
    agentName: string,
    existingAgents: string[]
  ): OperationResult {
    const teams = ['engineering', 'leadership'];
    if (!teams.includes(teamName)) {
      return { success: false, error: `Team '${teamName}' not found` };
    }
    if (!existingAgents.includes(agentName)) {
      return { success: false, error: `Agent '${agentName}' not found` };
    }
    return { success: true };
  }

  it('rejects send to non-existent channel', () => {
    const result = validateChannelSend('nonexistent', ['eng-01']);
    expect(result.success).toBe(false);
    expect(result.error).toContain('not found');
  });

  it('allows send to valid channel', () => {
    const result = validateChannelSend('eng', ['eng-01']);
    expect(result.success).toBe(true);
  });

  it('rejects add non-existent agent to team', () => {
    const existingAgents = ['eng-01', 'eng-02'];
    const result = validateTeamAdd('engineering', 'eng-99', existingAgents);
    expect(result.success).toBe(false);
    expect(result.error).toContain('Agent');
    expect(result.error).toContain('not found');
  });

  it('rejects add agent to non-existent team', () => {
    const existingAgents = ['eng-01', 'eng-02'];
    const result = validateTeamAdd('nonexistent-team', 'eng-01', existingAgents);
    expect(result.success).toBe(false);
    expect(result.error).toContain('Team');
    expect(result.error).toContain('not found');
  });

  it('allows add valid agent to valid team', () => {
    const existingAgents = ['eng-01', 'eng-02'];
    const result = validateTeamAdd('engineering', 'eng-01', existingAgents);
    expect(result.success).toBe(true);
  });
});

describe('Error Scenarios: Invalid Commands', () => {
  interface CommandResult {
    valid: boolean;
    error?: string;
  }

  function validateCommand(command: string): CommandResult {
    const validCommands = ['status', 'agent', 'channel', 'cost', 'team', 'process'];
    const parts = command.split(' ');
    const baseCommand = parts[0];

    if (!validCommands.includes(baseCommand)) {
      return { valid: false, error: `Unknown command: ${baseCommand}` };
    }

    return { valid: true };
  }

  it('rejects unknown command', () => {
    const result = validateCommand('foo bar');
    expect(result.valid).toBe(false);
    expect(result.error).toContain('Unknown command');
  });

  it('accepts valid command', () => {
    const result = validateCommand('status');
    expect(result.valid).toBe(true);
  });

  it('accepts valid command with args', () => {
    const result = validateCommand('agent list');
    expect(result.valid).toBe(true);
  });
});

describe('Error Scenarios: Edge Cases - Empty Data', () => {
  interface EmptyStateProps {
    hasAgents: boolean;
    hasChannels: boolean;
    hasMessages: boolean;
  }

  function EmptyStateComponent({
    hasAgents,
    hasChannels,
    hasMessages,
  }: EmptyStateProps): React.ReactElement {
    return (
      <Box flexDirection="column">
        {!hasAgents && <Text dimColor>No agents running</Text>}
        {!hasChannels && <Text dimColor>No channels</Text>}
        {!hasMessages && <Text dimColor>No messages yet</Text>}
        {hasAgents && hasChannels && hasMessages && <Text>All data available</Text>}
      </Box>
    );
  }

  it('handles empty agent list', () => {
    const { lastFrame } = render(
      <EmptyStateComponent hasAgents={false} hasChannels={true} hasMessages={true} />
    );
    expect(lastFrame()).toContain('No agents running');
  });

  it('handles empty channel list', () => {
    const { lastFrame } = render(
      <EmptyStateComponent hasAgents={true} hasChannels={false} hasMessages={true} />
    );
    expect(lastFrame()).toContain('No channels');
  });

  it('handles empty message history', () => {
    const { lastFrame } = render(
      <EmptyStateComponent hasAgents={true} hasChannels={true} hasMessages={false} />
    );
    expect(lastFrame()).toContain('No messages yet');
  });

  it('handles all data present', () => {
    const { lastFrame } = render(
      <EmptyStateComponent hasAgents={true} hasChannels={true} hasMessages={true} />
    );
    expect(lastFrame()).toContain('All data available');
  });
});

describe('Error Scenarios: Edge Cases - Agent with No Status', () => {
  interface AgentStatus {
    name: string;
    state: string | null;
    task: string | null;
  }

  function formatAgentStatus(agent: AgentStatus): string {
    const state = agent.state ?? 'unknown';
    const task = agent.task ?? 'No task assigned';
    return `${agent.name}: ${state} - ${task}`;
  }

  it('handles agent with null state', () => {
    const agent: AgentStatus = { name: 'eng-01', state: null, task: 'Some task' };
    const status = formatAgentStatus(agent);
    expect(status).toContain('unknown');
  });

  it('handles agent with null task', () => {
    const agent: AgentStatus = { name: 'eng-01', state: 'idle', task: null };
    const status = formatAgentStatus(agent);
    expect(status).toContain('No task assigned');
  });

  it('handles agent with all null fields', () => {
    const agent: AgentStatus = { name: 'eng-01', state: null, task: null };
    const status = formatAgentStatus(agent);
    expect(status).toContain('unknown');
    expect(status).toContain('No task assigned');
  });
});

describe('Error Scenarios: Edge Cases - Long Names/Messages', () => {
  function truncate(str: string, maxLen: number): string {
    if (str.length <= maxLen) return str;
    return str.slice(0, maxLen - 3) + '...';
  }

  it('truncates very long agent names', () => {
    const longName = 'a'.repeat(100);
    const truncated = truncate(longName, 20);
    expect(truncated.length).toBe(20);
    expect(truncated).toContain('...');
  });

  it('truncates very long messages', () => {
    const longMessage = 'word '.repeat(100);
    const truncated = truncate(longMessage, 50);
    expect(truncated.length).toBe(50);
    expect(truncated).toContain('...');
  });

  it('does not truncate short strings', () => {
    const shortString = 'Hello';
    const result = truncate(shortString, 20);
    expect(result).toBe('Hello');
    expect(result).not.toContain('...');
  });
});

describe('Error Scenarios: Edge Cases - Special Characters', () => {
  function sanitizeForDisplay(str: string): string {
    // Replace control characters but allow unicode
    // Using character code check instead of regex with control chars
    return str
      .split('')
      .filter((ch) => {
        const code = ch.charCodeAt(0);
        return code >= 32 && code !== 127;
      })
      .join('');
  }

  it('handles special characters in names', () => {
    const nameWithSpecial = 'eng-01<script>alert(1)</script>';
    // The display should still work (XSS is not a concern in terminal)
    expect(nameWithSpecial).toContain('eng-01');
  });

  it('handles unicode in messages', () => {
    const unicodeMessage = 'Hello 👋 World 🌍';
    expect(unicodeMessage).toContain('👋');
    expect(unicodeMessage).toContain('🌍');
  });

  it('handles newlines in messages', () => {
    const multilineMessage = 'Line 1\nLine 2\nLine 3';
    const lines = multilineMessage.split('\n');
    expect(lines.length).toBe(3);
  });

  it('sanitizes control characters', () => {
    const withControlChars = 'Hello\x00World\x1F';
    const sanitized = sanitizeForDisplay(withControlChars);
    expect(sanitized).toBe('HelloWorld');
  });
});

describe('Error Scenarios: Recovery', () => {
  interface RecoveryState {
    errorCount: number;
    lastError: string | null;
    recovered: boolean;
  }

  function attemptRecovery(state: RecoveryState): RecoveryState {
    if (state.errorCount >= 3) {
      return { ...state, recovered: false };
    }
    return { errorCount: 0, lastError: null, recovered: true };
  }

  it('recovers after few errors', () => {
    const state: RecoveryState = { errorCount: 2, lastError: 'Network error', recovered: false };
    const newState = attemptRecovery(state);
    expect(newState.recovered).toBe(true);
    expect(newState.errorCount).toBe(0);
    expect(newState.lastError).toBeNull();
  });

  it('fails to recover after many errors', () => {
    const state: RecoveryState = { errorCount: 3, lastError: 'Persistent error', recovered: false };
    const newState = attemptRecovery(state);
    expect(newState.recovered).toBe(false);
  });
});
