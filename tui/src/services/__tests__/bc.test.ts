/**
 * Tests for bc service - CLI command execution layer
 * Validates that service properly executes bc commands and parses responses
 *
 * #1066: Uses dependency injection via _setSpawnForTesting() for proper test isolation.
 * This avoids mock.module() conflicts when running full test suite in parallel.
 */

import { describe, it, expect, beforeEach, afterEach, mock } from 'bun:test';
import type { ChildProcess } from 'child_process';
import {
  execBc,
  execBcJson,
  getStatus,
  getChannels,
  getChannelHistory,
  sendChannelMessage,
  getCostSummary,
  reportState,
  getDemons,
  getTeams,
  clearCache,
  _setSpawnForTesting,
} from '../bc';

// Mock process factory - creates a mock ChildProcess-like object
const mockFn = () => mock(() => {});
const mockProcessorFactory = () => ({
  stdout: { on: mockFn() },
  stderr: { on: mockFn() },
  on: mockFn(),
  kill: mockFn(),
});

// Tracking for spawn mock
let mockSpawnImpl = mockFn();
let restoreSpawn: (() => void) | null = null;

describe('execBc - Basic command execution', () => {
  beforeEach(() => {
    clearCache(); // #1005: Clear command cache between tests
    mockSpawnImpl = mock(() => mockProcessorFactory());
    // Inject mock spawn before each test
    restoreSpawn = _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);
  });

  afterEach(() => {
    // Restore original spawn after each test
    if (restoreSpawn) {
      restoreSpawn();
      restoreSpawn = null;
    }
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
      _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

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
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

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
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await execBc(['status', '--json']);
    const callArgs = mockSpawnImpl.mock.calls[0][1] as string[];
    const jsonCount = callArgs.filter((arg: string) => arg === '--json').length;
    expect(jsonCount).toBe(1);
  });
});

describe('execBcJson - JSON parsing', () => {
  beforeEach(() => {
    clearCache();
    mockSpawnImpl = mock(() => mockProcessorFactory());
    restoreSpawn = _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);
  });

  afterEach(() => {
    if (restoreSpawn) {
      restoreSpawn();
      restoreSpawn = null;
    }
  });

  it('parses valid JSON response', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    const testData = { foo: 'bar', num: 123 };

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

    const result = await execBcJson<typeof testData>(['test']);
    expect(result).toEqual(testData);
  });

  it('throws on malformed JSON', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    setTimeout(() => {
      const stdoutCalls = (mockProc.stdout.on as ReturnType<typeof mock>).mock.calls;
      stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
        if (event === 'data') handler(Buffer.from('not json'));
      });
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    expect(execBcJson(['test'])).rejects.toThrow(/Failed to parse bc output as JSON/);
  });

  it('throws on empty response', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    expect(execBcJson(['test'])).rejects.toThrow(/Failed to parse bc output as JSON/);
  });
});

describe('Command wrapper functions - Status and channels', () => {
  beforeEach(() => {
    clearCache();
    mockSpawnImpl = mock(() => mockProcessorFactory());
    restoreSpawn = _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);
  });

  afterEach(() => {
    if (restoreSpawn) {
      restoreSpawn();
      restoreSpawn = null;
    }
  });

  it('getStatus fetches agent status', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

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
    expect(result.agents).toEqual(statusData.agents);
  });

  it('getChannels fetches channel list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

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
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

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

describe('Command wrapper functions - Cost and teams', () => {
  beforeEach(() => {
    clearCache();
    mockSpawnImpl = mock(() => mockProcessorFactory());
    restoreSpawn = _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);
  });

  afterEach(() => {
    if (restoreSpawn) {
      restoreSpawn();
      restoreSpawn = null;
    }
  });

  it('getCostSummary returns cost data', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    const costData = {
      total_cost: 1.23,
      total_input_tokens: 1000,
      total_output_tokens: 500,
      by_agent: {},
      by_team: {},
      by_model: {},
    };

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
    expect(result.total_cost).toBe(1.23);
  });

  it('getCostSummary returns empty object on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getCostSummary();
    expect(result.total_cost).toBe(0);
  });

  it('getTeams fetches team list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    const teamsData = { teams: [{ name: 'frontend', members: ['eng-01', 'eng-02'] }] };

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
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    setTimeout(() => {
      const onCalls = (mockProc.on as ReturnType<typeof mock>).mock.calls;
      onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getTeams();
    expect(result).toEqual({ teams: [] });
  });
});

describe('Demon operations', () => {
  beforeEach(() => {
    clearCache();
    mockSpawnImpl = mock(() => mockProcessorFactory());
    restoreSpawn = _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);
  });

  afterEach(() => {
    if (restoreSpawn) {
      restoreSpawn();
      restoreSpawn = null;
    }
  });

  it('getDemons returns demon list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawnImpl = mock(() => mockProc);
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    const demonData = [{ name: 'daily-backup', schedule: '0 0 * * *' }];

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
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

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

/**
 * Command cache stress tests (#1016)
 * Validates caching behavior under load
 */
describe('Command cache stress testing (#1016)', () => {
  beforeEach(() => {
    clearCache();
    mockSpawnImpl = mock(() => mockProcessorFactory());
    restoreSpawn = _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);
  });

  afterEach(() => {
    if (restoreSpawn) {
      restoreSpawn();
      restoreSpawn = null;
    }
  });

  it('cache reduces subprocess calls on repeated status checks', async () => {
    let spawnCallCount = 0;
    const statusData = { agents: [{ name: 'test-agent', state: 'idle' }] };

    mockSpawnImpl = mock(() => {
      spawnCallCount++;
      const newProc = mockProcessorFactory();
      setTimeout(() => {
        const stdoutCalls = (newProc.stdout.on as ReturnType<typeof mock>).mock.calls;
        stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
          if (event === 'data') handler(Buffer.from(JSON.stringify(statusData)));
        });
        const onCalls = (newProc.on as ReturnType<typeof mock>).mock.calls;
        onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
          if (event === 'close') handler(0);
        });
      }, 5);
      return newProc;
    });
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    // First call should spawn
    const result1 = await getStatus();
    expect(result1.agents).toEqual(statusData.agents);

    // Second call within TTL should use cache (status TTL is 1000ms)
    const result2 = await getStatus();
    expect(result2.agents).toEqual(statusData.agents);

    // Should have only spawned once due to caching
    expect(spawnCallCount).toBe(1);
  });

  it('clearCache invalidates all cached results', async () => {
    let spawnCallCount = 0;
    const statusData = { agents: [{ name: 'test-agent', state: 'idle' }] };

    mockSpawnImpl = mock(() => {
      spawnCallCount++;
      const newProc = mockProcessorFactory();
      setTimeout(() => {
        const stdoutCalls = (newProc.stdout.on as ReturnType<typeof mock>).mock.calls;
        stdoutCalls.forEach(([event, handler]: [string, (data: Buffer) => void]) => {
          if (event === 'data') handler(Buffer.from(JSON.stringify(statusData)));
        });
        const onCalls = (newProc.on as ReturnType<typeof mock>).mock.calls;
        onCalls.forEach(([event, handler]: [string, (code: number) => void]) => {
          if (event === 'close') handler(0);
        });
      }, 5);
      return newProc;
    });
    _setSpawnForTesting(mockSpawnImpl as unknown as Parameters<typeof _setSpawnForTesting>[0]);

    // First call
    await getStatus();
    const afterFirstCall = spawnCallCount;

    // Clear cache
    clearCache();

    // Second call should spawn again (cache cleared)
    await getStatus();

    // Should have spawned again after cache clear
    expect(spawnCallCount).toBe(afterFirstCall + 1);
  });
});
