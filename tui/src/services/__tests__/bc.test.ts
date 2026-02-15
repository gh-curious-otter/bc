/**
 * Tests for bc service - CLI command execution layer
 * Validates that service properly executes bc commands and parses responses
 */

import { execBc, execBcJson, getStatus, getChannels, getChannelHistory, sendChannelMessage, getCostSummary, reportState, getDemons, getTeams } from '../bc';
import { spawn } from 'child_process';

jest.mock('child_process');

// Test fixtures
const mockProcessorFactory = () => ({
  stdout: { on: jest.fn() },
  stderr: { on: jest.fn() },
  on: jest.fn(),
  kill: jest.fn(),
});

const mockSpawn = spawn as jest.MockedFunction<typeof spawn>;

describe('execBc - Basic command execution', () => {
  beforeEach(() => {
    jest.clearAllMocks();
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

  testCases.forEach(({ name, args, shouldJson, output, expectedCode }) => {
    it(name, async () => {
      const mockProc = mockProcessorFactory();
      mockSpawn.mockReturnValue(mockProc as any);

      setTimeout(() => {
        // Simulate stdout data
        mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
          if (event === 'data') handler(Buffer.from(output));
        });
        // Simulate close event
        mockProc.on.mock.calls.forEach(([event, handler]) => {
          if (event === 'close') handler(expectedCode);
        });
      }, 5);

      const result = await execBc(args);
      expect(result).toBe(output);
    });
  });

  it('automatically adds --json flag for supported commands', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await execBc(['status']);
    const callArgs = mockSpawn.mock.calls[0][1] as string[];
    expect(callArgs).toContain('--json');
  });

  it('does not duplicate --json flag if already present', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await execBc(['status', '--json']);
    const callArgs = mockSpawn.mock.calls[0][1] as string[];
    const jsonCount = callArgs.filter(arg => arg === '--json').length;
    expect(jsonCount).toBe(1);
  });
});

describe('execBc - Error handling', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  const errorCases = [
    {
      name: 'rejects with stderr on non-zero exit',
      setupError: 'agent not found',
      exitCode: 1,
      expectError: /agent not found/,
    },
    {
      name: 'handles spawn process errors',
      setupError: null,
      processError: new Error('ENOENT: command not found'),
      expectError: /Failed to spawn bc/,
    },
  ];

  errorCases.forEach(({ name, setupError, exitCode, processError, expectError }) => {
    it(name, async () => {
      const mockProc = mockProcessorFactory();
      mockSpawn.mockReturnValue(mockProc as any);

      setTimeout(() => {
        if (setupError) {
          mockProc.stderr.on.mock.calls.forEach(([event, handler]) => {
            if (event === 'data') handler(Buffer.from(setupError));
          });
          mockProc.on.mock.calls.forEach(([event, handler]) => {
            if (event === 'close') handler(exitCode || 1);
          });
        }
        if (processError) {
          mockProc.on.mock.calls.forEach(([event, handler]) => {
            if (event === 'error') handler(processError);
          });
        }
      }, 5);

      await expect(execBc(['invalid'])).rejects.toThrow(expectError);
    });
  });
});

describe('execBcJson - JSON parsing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('parses valid JSON response', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const testData = { agents: [{ name: 'eng-01', state: 'working' }] };

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(testData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await execBcJson(['status']);
    expect(result).toEqual(testData);
  });

  it('throws on malformed JSON', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from('{invalid json'));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await expect(execBcJson(['status'])).rejects.toThrow('Failed to parse bc output as JSON');
  });

  it('throws on empty response', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await expect(execBcJson(['status'])).rejects.toThrow('Failed to parse bc output as JSON');
  });
});

describe('Command wrapper functions - Status and channels', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('getStatus fetches agent status', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const statusData = { agents: [{ name: 'eng-01', state: 'working' }] };

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(statusData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getStatus();
    expect(result).toEqual(statusData);
    expect(mockSpawn).toHaveBeenCalledWith('bc', expect.arrayContaining(['status']), expect.any(Object));
  });

  it('getChannels fetches channel list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const channelsData = { channels: [{ name: 'eng', members: ['eng-01'] }] };

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(channelsData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getChannels();
    expect(result).toEqual(channelsData);
    expect(mockSpawn).toHaveBeenCalledWith('bc', expect.arrayContaining(['channel', 'list']), expect.any(Object));
  });

  it('getChannelHistory fetches message history', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const historyData = { messages: [{ sender: 'eng-01', text: 'Hello', timestamp: 123456 }] };

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(historyData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getChannelHistory('eng');
    expect(result).toEqual(historyData);
    expect(mockSpawn).toHaveBeenCalledWith('bc', expect.arrayContaining(['channel', 'history', 'eng']), expect.any(Object));
  });
});

describe('Command wrapper functions - Actions', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('sendChannelMessage sends message to channel', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await sendChannelMessage('eng', 'Test message');
    expect(mockSpawn).toHaveBeenCalledWith(
      'bc',
      expect.arrayContaining(['channel', 'send', 'eng', 'Test message']),
      expect.any(Object)
    );
  });

  it('reportState reports agent state', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    await reportState('working', 'Implementing feature');
    expect(mockSpawn).toHaveBeenCalledWith(
      'bc',
      expect.arrayContaining(['report', 'working', 'Implementing feature']),
      expect.any(Object)
    );
  });
});

describe('Command wrapper functions - Cost and teams', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('getCostSummary returns cost data', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const costData = { total_cost: 100, by_agent: { 'eng-01': 50 }, by_team: {}, by_model: {} };

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(costData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getCostSummary();
    expect(result.total_cost).toBe(100);
  });

  it('getCostSummary returns empty object on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(1); // Simulate failure
      });
    }, 5);

    const result = await getCostSummary();
    expect(result.total_cost).toBe(0);
    expect(result.by_agent).toEqual({});
  });

  it('getTeams fetches team list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const teamsData = { teams: [{ name: 'eng-team', members: ['eng-01', 'eng-02'] }] };

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(teamsData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getTeams();
    expect(result).toEqual(teamsData);
  });

  it('getTeams returns empty array on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getTeams();
    expect(result.teams).toEqual([]);
  });
});

describe('Demon operations', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('getDemons returns demon list', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    const demonData = [{ name: 'hourly-sync', enabled: true, next_run: 12345 }];

    setTimeout(() => {
      mockProc.stdout.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'data') handler(Buffer.from(JSON.stringify(demonData)));
      });
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(0);
      });
    }, 5);

    const result = await getDemons();
    expect(result).toEqual(demonData);
  });

  it('getDemons returns empty array on failure', async () => {
    const mockProc = mockProcessorFactory();
    mockSpawn.mockReturnValue(mockProc as any);

    setTimeout(() => {
      mockProc.on.mock.calls.forEach(([event, handler]) => {
        if (event === 'close') handler(1);
      });
    }, 5);

    const result = await getDemons();
    expect(result).toEqual([]);
  });
});
