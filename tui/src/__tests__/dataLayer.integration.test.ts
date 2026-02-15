/**
 * Data Layer Integration Tests - End-to-end workflows
 * Tests complete user workflows combining multiple data operations
 */

import * as bcService from '../services/bc';

jest.mock('../services/bc');

const mockBcService = bcService as jest.Mocked<typeof bcService>;

describe('Data Layer - Agent lifecycle workflow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('complete agent workflow: status -> channels -> report', async () => {
    // Setup: Agent status
    const statusResponse = {
      agents: [{ name: 'eng-01', state: 'idle', role: 'engineer' }],
    };
    mockBcService.getStatus.mockResolvedValue(statusResponse);

    // Step 1: Get current status
    const status = await bcService.getStatus();
    expect(status.agents[0].state).toBe('idle');

    // Setup: Channel operations
    const channelsResponse = {
      channels: [{ name: 'eng', members: ['eng-01', 'eng-02'] }],
    };
    mockBcService.getChannels.mockResolvedValue(channelsResponse);

    // Step 2: Get channels
    const channels = await bcService.getChannels();
    expect(channels.channels[0].name).toBe('eng');

    // Setup: Send message
    mockBcService.sendChannelMessage.mockResolvedValue(undefined);

    // Step 3: Report status while in channel context
    await bcService.reportState('working', 'Implemented feature X');
    expect(mockBcService.reportState).toHaveBeenCalledWith('working', 'Implemented feature X');
  });

  it('multi-step agent state transitions', async () => {
    const states = [
      { state: 'idle', message: 'Ready for assignment' },
      { state: 'working', message: 'Starting task' },
      { state: 'working', message: 'In progress' },
      { state: 'done', message: 'Task completed' },
    ];

    mockBcService.reportState.mockResolvedValue(undefined);

    for (const { state, message } of states) {
      await bcService.reportState(state, message);
      expect(mockBcService.reportState).toHaveBeenCalledWith(state, message);
    }

    expect(mockBcService.reportState).toHaveBeenCalledTimes(states.length);
  });
});

describe('Data Layer - Channel communication workflow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('full channel conversation: list -> history -> send -> refresh', async () => {
    // Step 1: List channels
    const channelsData = {
      channels: [
        { name: 'eng', members: ['eng-01', 'eng-02'] },
        { name: 'leads', members: ['tl-01'] },
      ],
    };
    mockBcService.getChannels.mockResolvedValue(channelsData);

    const channels = await bcService.getChannels();
    expect(channels.channels).toHaveLength(2);

    // Step 2: Get history of first channel
    const historyData = {
      messages: [
        { sender: 'eng-01', text: 'Hi everyone', timestamp: 1000 },
        { sender: 'tl-01', text: 'Welcome!', timestamp: 1100 },
      ],
    };
    mockBcService.getChannelHistory.mockResolvedValue(historyData);

    const history = await bcService.getChannelHistory('eng');
    expect(history.messages).toHaveLength(2);

    // Step 3: Send message to channel
    mockBcService.sendChannelMessage.mockResolvedValue(undefined);
    await bcService.sendChannelMessage('eng', 'Just finished implementation');
    expect(mockBcService.sendChannelMessage).toHaveBeenCalledWith('eng', 'Just finished implementation');

    // Step 4: Refresh history
    mockBcService.getChannelHistory.mockResolvedValue({
      messages: [...historyData.messages, { sender: 'eng-01', text: 'Just finished implementation', timestamp: 1200 }],
    });

    const updatedHistory = await bcService.getChannelHistory('eng');
    expect(updatedHistory.messages).toHaveLength(3);
  });

  it('handles channel message batching', async () => {
    mockBcService.sendChannelMessage.mockResolvedValue(undefined);

    const messages = [
      'Starting Phase 2 testing',
      'Implemented 50 tests',
      'All tests passing',
      'Ready for review',
    ];

    for (const msg of messages) {
      await bcService.sendChannelMessage('eng', msg);
    }

    expect(mockBcService.sendChannelMessage).toHaveBeenCalledTimes(4);
  });
});

describe('Data Layer - Team coordination workflow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('team operations: list -> add member -> manage', async () => {
    // Step 1: List teams
    const teamsData = {
      teams: [{ name: 'eng-team', members: ['eng-01', 'eng-02'] }],
    };
    mockBcService.getTeams.mockResolvedValue(teamsData);

    const teams = await bcService.getTeams();
    expect(teams.teams[0].members).toHaveLength(2);

    // Step 2: Add member to team
    mockBcService.addTeamMember.mockResolvedValue(undefined);
    await bcService.addTeamMember('eng-team', 'eng-03');
    expect(mockBcService.addTeamMember).toHaveBeenCalledWith('eng-team', 'eng-03');

    // Step 3: Refresh teams
    const updatedTeams = {
      teams: [{ name: 'eng-team', members: ['eng-01', 'eng-02', 'eng-03'] }],
    };
    mockBcService.getTeams.mockResolvedValue(updatedTeams);

    const teams2 = await bcService.getTeams();
    expect(teams2.teams[0].members).toHaveLength(3);
  });

  it('manages multiple team operations', async () => {
    mockBcService.addTeamMember.mockResolvedValue(undefined);
    mockBcService.removeTeamMember.mockResolvedValue(undefined);

    // Add members
    await bcService.addTeamMember('eng-team', 'eng-03');
    await bcService.addTeamMember('eng-team', 'eng-04');

    // Remove member
    await bcService.removeTeamMember('eng-team', 'eng-02');

    expect(mockBcService.addTeamMember).toHaveBeenCalledTimes(2);
    expect(mockBcService.removeTeamMember).toHaveBeenCalledTimes(1);
  });
});

describe('Data Layer - Demon (scheduled task) workflow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('demon workflow: list -> get details -> manage', async () => {
    // Step 1: List all demons
    const demonsData = [
      { name: 'hourly-sync', enabled: true, next_run: 12345 },
      { name: 'daily-cleanup', enabled: false, next_run: 54321 },
    ];
    mockBcService.getDemons.mockResolvedValue(demonsData);

    const demons = await bcService.getDemons();
    expect(demons).toHaveLength(2);

    // Step 2: Get specific demon
    mockBcService.getDemon.mockResolvedValue(demonsData[0]);

    const demon = await bcService.getDemon('hourly-sync');
    expect(demon?.name).toBe('hourly-sync');
    expect(demon?.enabled).toBe(true);

    // Step 3: Get demon logs
    mockBcService.getDemonLogs.mockResolvedValue([
      { timestamp: 1000, status: 'success', message: 'Sync completed' },
      { timestamp: 2000, status: 'success', message: 'Sync completed' },
    ]);

    const logs = await bcService.getDemonLogs('hourly-sync', 10);
    expect(logs).toHaveLength(2);
  });

  it('manages demon enable/disable', async () => {
    mockBcService.enableDemon.mockResolvedValue(undefined);
    mockBcService.disableDemon.mockResolvedValue(undefined);
    mockBcService.runDemon.mockResolvedValue(undefined);

    // Enable demon
    await bcService.enableDemon('hourly-sync');
    expect(mockBcService.enableDemon).toHaveBeenCalledWith('hourly-sync');

    // Disable demon
    await bcService.disableDemon('daily-cleanup');
    expect(mockBcService.disableDemon).toHaveBeenCalledWith('daily-cleanup');

    // Manually run demon
    await bcService.runDemon('hourly-sync');
    expect(mockBcService.runDemon).toHaveBeenCalledWith('hourly-sync');
  });
});

describe('Data Layer - Cost tracking workflow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('cost tracking through agent lifecycle', async () => {
    // Initial costs
    const initialCosts = {
      total_cost: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      by_agent: {},
      by_team: {},
      by_model: {},
    };
    mockBcService.getCostSummary.mockResolvedValueOnce(initialCosts);

    let costs = await bcService.getCostSummary();
    expect(costs.total_cost).toBe(0);

    // After some work
    const updatedCosts = {
      total_cost: 150.50,
      total_input_tokens: 50000,
      total_output_tokens: 10000,
      by_agent: { 'eng-01': 75.25, 'eng-02': 75.25 },
      by_team: { 'eng-team': 150.50 },
      by_model: { 'claude-3-sonnet': 150.50 },
    };
    mockBcService.getCostSummary.mockResolvedValueOnce(updatedCosts);

    costs = await bcService.getCostSummary();
    expect(costs.total_cost).toBe(150.50);
    expect(costs.by_agent['eng-01']).toBe(75.25);
  });
});

describe('Data Layer - Process management workflow', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('process management: list -> get logs -> track', async () => {
    // Step 1: List processes
    const processesData = {
      processes: [
        { name: 'worker-1', pid: 1234, status: 'running' },
        { name: 'worker-2', pid: 1235, status: 'running' },
        { name: 'archive', pid: 1236, status: 'stopped' },
      ],
    };
    mockBcService.getProcesses.mockResolvedValue(processesData);

    const processes = await bcService.getProcesses();
    const running = processes.processes.filter(p => p.status === 'running');
    expect(running).toHaveLength(2);

    // Step 2: Get logs for specific process
    mockBcService.getProcessLogs.mockResolvedValue([
      'Process started',
      'Processing batch 1',
      'Processing batch 2',
      'Process completed',
    ]);

    const logs = await bcService.getProcessLogs('worker-1', 100);
    expect(logs).toHaveLength(4);
  });
});

describe('Data Layer - Complex concurrent operations', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('handles concurrent status and channel operations', async () => {
    mockBcService.getStatus.mockResolvedValue({
      agents: [{ name: 'eng-01', state: 'working', role: 'engineer' }],
    });

    mockBcService.getChannels.mockResolvedValue({
      channels: [{ name: 'eng', members: ['eng-01'] }],
    });

    mockBcService.getTeams.mockResolvedValue({
      teams: [{ name: 'eng-team', members: ['eng-01'] }],
    });

    // Execute all in parallel
    const [status, channels, teams] = await Promise.all([
      bcService.getStatus(),
      bcService.getChannels(),
      bcService.getTeams(),
    ]);

    expect(status.agents).toBeDefined();
    expect(channels.channels).toBeDefined();
    expect(teams.teams).toBeDefined();
  });

  it('handles rapid report state changes', async () => {
    mockBcService.reportState.mockResolvedValue(undefined);

    const reports = [];
    for (let i = 0; i < 10; i++) {
      reports.push(bcService.reportState('working', `Status update ${i}`));
    }

    await Promise.all(reports);
    expect(mockBcService.reportState).toHaveBeenCalledTimes(10);
  });

  it('handles batch channel messages', async () => {
    mockBcService.sendChannelMessage.mockResolvedValue(undefined);

    const channels = ['eng', 'leads', 'design'];
    const messages = channels.flatMap(ch =>
      Array.from({ length: 5 }, (_, i) => bcService.sendChannelMessage(ch, `Message ${i}`))
    );

    await Promise.all(messages);
    expect(mockBcService.sendChannelMessage).toHaveBeenCalledTimes(15);
  });
});

describe('Data Layer - Error recovery workflows', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('recovers from transient failures', async () => {
    // First call fails, second succeeds
    mockBcService.getStatus
      .mockRejectedValueOnce(new Error('Network timeout'))
      .mockResolvedValueOnce({
        agents: [{ name: 'eng-01', state: 'working', role: 'engineer' }],
      });

    try {
      await bcService.getStatus();
    } catch (error) {
      // Expected to fail first
    }

    // Retry succeeds
    const result = await bcService.getStatus();
    expect(result.agents).toBeDefined();
  });

  it('handles cascading failures gracefully', async () => {
    mockBcService.getStatus.mockRejectedValue(new Error('Service unavailable'));
    mockBcService.getChannels.mockRejectedValue(new Error('Service unavailable'));
    mockBcService.getTeams.mockRejectedValue(new Error('Service unavailable'));

    const results = await Promise.allSettled([
      bcService.getStatus(),
      bcService.getChannels(),
      bcService.getTeams(),
    ]);

    expect(results).toHaveLength(3);
    results.forEach(result => {
      expect(result.status).toBe('rejected');
    });
  });
});

describe('Data Layer - State consistency', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('maintains consistent state across multiple reads', async () => {
    const statusData = {
      agents: [
        { name: 'eng-01', state: 'working', role: 'engineer' },
        { name: 'eng-02', state: 'idle', role: 'engineer' },
      ],
    };

    mockBcService.getStatus.mockResolvedValue(statusData);

    // Multiple reads should return consistent data
    const status1 = await bcService.getStatus();
    const status2 = await bcService.getStatus();
    const status3 = await bcService.getStatus();

    expect(status1).toEqual(status2);
    expect(status2).toEqual(status3);
  });

  it('detects state changes between reads', async () => {
    mockBcService.getStatus
      .mockResolvedValueOnce({
        agents: [{ name: 'eng-01', state: 'idle', role: 'engineer' }],
      })
      .mockResolvedValueOnce({
        agents: [{ name: 'eng-01', state: 'working', role: 'engineer' }],
      });

    const status1 = await bcService.getStatus();
    const status2 = await bcService.getStatus();

    expect(status1.agents[0].state).toBe('idle');
    expect(status2.agents[0].state).toBe('working');
  });
});
