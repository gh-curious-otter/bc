/**
 * Tests for bc service - CLI command execution layer
 * Validates that service properly executes bc commands and parses responses
 *
 * NOTE: These tests use bun:test mock.module() for child_process mocking.
 * The mock is set up at the top level before imports.
 */

import { describe, it, expect, beforeEach, mock } from 'bun:test';

// Mock child_process before importing the service
const mockFn = () => mock(() => {});
const mockProcessorFactory = () => ({
  stdout: { on: mockFn() },
  stderr: { on: mockFn() },
  on: mockFn(),
  kill: mockFn(),
});

let mockSpawnImpl = mockFn();

mock.module('child_process', () => ({
  spawn: (...args: unknown[]) => mockSpawnImpl(...args),
  spawnSync: () => ({ stdout: Buffer.from(''), stderr: Buffer.from(''), status: 0, signal: null }),
}));

// Now import the service (after mocking)
const { execBc, execBcJson, getStatus, getChannels, getChannelHistory, sendChannelMessage, getCostSummary, reportState, getDemons, getTeams } = await import('../bc');

describe('execBc - Basic command execution', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  const testCases = [
    {
      name: 'executes status command',
      args: ['status'],
      shouldJson: true,
      output: '{"agents":[]}',
      expectedCode: 0,
    },
    {
      name: 'executes channel list command',
      args: ['channel', 'list'],
      shouldJson: true,
      output: '{"channels":[]}',
      expectedCode: 0,
    },
    {
      name: 'handles non-JSON commands',
      args: ['version'],
      shouldJson: false,
      output: '1.0.0',
      expectedCode: 0,
    },
  ];

  testCases.forEach(({ name, args, output, expectedCode }) => {
    it(name, async () => {
      const mockProc = mockProcessorFactory();
      mockSpawnImpl = mock(() => mockProc);

      setTimeout(() => {
        // Simulate stdout data
        const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
        stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
          if (event === 'data') handler(Buffer.from(output));
        });
        // Simulate close event
        const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
        onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
          if (event === 'close') handler(expectedCode);
        });
      }, 5);

      const result = await execBc(args);
      expect(result).toBe(output);
    });
  });

  it('automatically adds --json flag for supported commands', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await execBc(['status']);
    const callArgs = mockSpawnImpl.mock.calls[0][1] as string[];
    expect(callArgs).toContain('--json');
  });

  it('does not duplicate --json flag if already present', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await execBc(['status', '--json']);
    const callArgs = mockSpawnImpl.mock.calls[0][1] as string[];
    const jsonCount = callArgs.filter(arg => arg === '--json').length;
    expect(jsonCount).toBe(1);
  });
});

describe('execBc - Error handling', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  it('rejects with stderr on non-zero exit', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const stderrCalls = (mockProc.stderr.on as ReturnType<typeof mock>).mock.calls;
      stderrCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from('agent not found'));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    // eslint-disable-next-line @typescript-eslint/await-thenable -- Jest/Bun requires await
    await expect(execBc(['invalid'])).rejects.toThrow(/agent not found/);
  });

  it('handles spawn process errors', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (err: Error) => void]) => {
        if (event === 'error') handler(new Error('ENOENT: command not found'));
      });
    }, 5);

    // eslint-disable-next-line @typescript-eslint/await-thenable -- Jest/Bun requires await
    await expect(execBc(['invalid'])).rejects.toThrow(/Failed to spawn bc/);
  });
});

describe('execBcJson - JSON parsing', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  it('parses valid JSON response', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const testData = { agents: [{ name: 'eng-01', state: 'working' }] };

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(testData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await execBcJson(['status']);
    expect(result).toEqual(testData);
  });

  it('throws on malformed JSON', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from('{invalid json'));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    // eslint-disable-next-line @typescript-eslint/await-thenable -- Jest/Bun requires await
    await expect(execBcJson(['status'])).rejects.toThrow('Failed to parse bc output as JSON');
  });

  it('throws on empty response', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    // eslint-disable-next-line @typescript-eslint/await-thenable -- Jest/Bun requires await
    await expect(execBcJson(['status'])).rejects.toThrow('Failed to parse bc output as JSON');
  });
});

describe('Command wrapper functions - Status and channels', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  it('getStatus fetches agent status', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const statusData = { agents: [{ name: 'eng-01', state: 'working' }] };

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(statusData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getStatus();
    expect(result).toEqual(statusData);
    expect(mockSpawnImpl).toHaveBeenCalled();
  });

  it('getChannels fetches channel list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const channelsData = { channels: [{ name: 'eng', members: ['eng-01'] }] };

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(channelsData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getChannels();
    expect(result).toEqual(channelsData);
  });

  it('getChannelHistory fetches message history', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const historyData = { messages: [{ sender: 'eng-01', text: 'Hello', timestamp: 123456 }] };

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(historyData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getChannelHistory('eng');
    expect(result).toEqual(historyData);
  });
});

describe('Command wrapper functions - Actions', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  it('sendChannelMessage sends message to channel', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await sendChannelMessage('eng', 'Test message');
    expect(mockSpawnImpl).toHaveBeenCalled();
  });

  it('reportState reports agent state', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await reportState('working', 'Implementing feature');
    expect(mockSpawnImpl).toHaveBeenCalled();
  });
});

describe('Command wrapper functions - Cost and teams', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  it('getCostSummary returns cost data', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const costData = { total_cost: 100, by_agent: { 'eng-01': 50 }, by_team: {}, by_model: {} };

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(costData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getCostSummary();
    expect(result.total_cost).toBe(100);
  });

  it('getCostSummary returns empty object on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getCostSummary();
    expect(result.total_cost).toBe(0);
    expect(result.by_agent).toEqual({});
  });

  it('getTeams fetches team list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const teamsData = { teams: [{ name: 'eng-team', members: ['eng-01', 'eng-02'] }] };

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(teamsData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getTeams();
    expect(result).toEqual(teamsData);
  });

  it('getTeams returns empty array on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getTeams();
    expect(result.teams).toEqual([]);
  });
});

describe('Demon operations', () => {
  beforeEach(() => {
    mockSpawnImpl = mock(() => mockProcessorFactory());
  });

  it('getDemons returns demon list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    const demonData = [{ name: 'hourly-sync', enabled: true, next_run: 12345 }];

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(demonData)));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getDemons();
    expect(result).toEqual(demonData);
  });

  it('getDemons returns empty array on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getDemons();
    expect(result).toEqual([]);
  });
});
