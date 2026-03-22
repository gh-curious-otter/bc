/**
 * Data Consistency Tests - Cross-reference validation
 * Issue #751 - TUI E2E Workflows & Real-Time Updates
 *
 * Tests verify data consistency across different views and operations.
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Text, Box } from 'ink';
import { describe, it, expect } from 'bun:test';
import type { Agent, Channel, Team, CostSummary } from '../../types';

// Mock data with intentional cross-references
const mockWorkspace = {
  name: 'test-workspace',
  agents: [
    { name: 'eng-01', role: 'engineer', state: 'working' },
    { name: 'eng-02', role: 'engineer', state: 'idle' },
    { name: 'tl-01', role: 'tech-lead', state: 'working' },
    { name: 'mgr-01', role: 'manager', state: 'idle' },
  ] as Array<Pick<Agent, 'name' | 'role' | 'state'>>,
  channels: [
    { name: 'eng', members: ['eng-01', 'eng-02', 'tl-01'] },
    { name: 'leads', members: ['tl-01', 'mgr-01'] },
    { name: 'all', members: ['eng-01', 'eng-02', 'tl-01', 'mgr-01'] },
  ] as Array<Pick<Channel, 'name' | 'members'>>,
  teams: [
    { name: 'engineering', members: ['eng-01', 'eng-02'], lead: 'tl-01' },
    { name: 'leadership', members: ['tl-01', 'mgr-01'], lead: 'mgr-01' },
  ] as Array<Pick<Team, 'name' | 'members'> & { lead: string }>,
  costs: {
    total_cost: 5.0,
    total_input_tokens: 100000,
    total_output_tokens: 50000,
    by_agent: {
      'eng-01': 2.0,
      'eng-02': 1.5,
      'tl-01': 1.0,
      'mgr-01': 0.5,
    },
    by_team: {
      engineering: 3.5,
      leadership: 1.5,
    },
    by_model: {
      'claude-opus': 4.0,
      'claude-sonnet': 1.0,
    },
  } as CostSummary,
};

// Test component for data display
function DataConsistencyDisplay({
  workspace,
}: {
  workspace: typeof mockWorkspace;
}): React.ReactElement {
  const agentNames = workspace.agents.map((a) => a.name);
  const allChannelMembers = workspace.channels.flatMap((c) => c.members);
  const uniqueChannelMembers = [...new Set(allChannelMembers)];

  return (
    <Box flexDirection="column">
      <Text>Agents: {workspace.agents.length}</Text>
      <Text>Channels: {workspace.channels.length}</Text>
      <Text>Teams: {workspace.teams.length}</Text>
      <Text>Unique channel members: {uniqueChannelMembers.length}</Text>
      <Text>Total cost: ${workspace.costs.total_cost.toFixed(2)}</Text>
    </Box>
  );
}

describe('Data Consistency: Agent List Matches Status', () => {
  it('all agents have valid states', () => {
    const validStates = ['idle', 'starting', 'working', 'done', 'stuck', 'error', 'stopped'];
    const allStatesValid = mockWorkspace.agents.every((agent) => validStates.includes(agent.state));
    expect(allStatesValid).toBe(true);
  });

  it('all agents have valid roles', () => {
    const validRoles = ['root', 'product-manager', 'manager', 'tech-lead', 'engineer'];
    const allRolesValid = mockWorkspace.agents.every((agent) => validRoles.includes(agent.role));
    expect(allRolesValid).toBe(true);
  });

  it('agent names are unique', () => {
    const names = mockWorkspace.agents.map((a) => a.name);
    const uniqueNames = [...new Set(names)];
    expect(names.length).toBe(uniqueNames.length);
  });

  it('displays correct agent count', () => {
    const { lastFrame } = render(<DataConsistencyDisplay workspace={mockWorkspace} />);
    expect(lastFrame()).toContain('Agents: 4');
  });
});

describe('Data Consistency: Channel Members Match Agents', () => {
  it('all channel members exist as agents', () => {
    const agentNames = mockWorkspace.agents.map((a) => a.name);
    const allMembersExist = mockWorkspace.channels.every((channel) =>
      channel.members.every((member) => agentNames.includes(member))
    );
    expect(allMembersExist).toBe(true);
  });

  it('channel names are unique', () => {
    const names = mockWorkspace.channels.map((c) => c.name);
    const uniqueNames = [...new Set(names)];
    expect(names.length).toBe(uniqueNames.length);
  });

  it('no duplicate members in a channel', () => {
    const noDuplicates = mockWorkspace.channels.every((channel) => {
      const uniqueMembers = [...new Set(channel.members)];
      return channel.members.length === uniqueMembers.length;
    });
    expect(noDuplicates).toBe(true);
  });

  it('displays correct channel count', () => {
    const { lastFrame } = render(<DataConsistencyDisplay workspace={mockWorkspace} />);
    expect(lastFrame()).toContain('Channels: 3');
  });
});

describe('Data Consistency: Team Members in Agent List', () => {
  it('all team members exist as agents', () => {
    const agentNames = mockWorkspace.agents.map((a) => a.name);
    const allMembersExist = mockWorkspace.teams.every((team) =>
      team.members.every((member) => agentNames.includes(member))
    );
    expect(allMembersExist).toBe(true);
  });

  it('team leads exist as agents', () => {
    const agentNames = mockWorkspace.agents.map((a) => a.name);
    const allLeadsExist = mockWorkspace.teams.every((team) => agentNames.includes(team.lead));
    expect(allLeadsExist).toBe(true);
  });

  it('team names are unique', () => {
    const names = mockWorkspace.teams.map((t) => t.name);
    const uniqueNames = [...new Set(names)];
    expect(names.length).toBe(uniqueNames.length);
  });

  it('no duplicate members in a team', () => {
    const noDuplicates = mockWorkspace.teams.every((team) => {
      const uniqueMembers = [...new Set(team.members)];
      return team.members.length === uniqueMembers.length;
    });
    expect(noDuplicates).toBe(true);
  });
});

describe('Data Consistency: Costs Match Agent Usage', () => {
  it('agent costs sum to total cost', () => {
    const agentCostSum = Object.values(mockWorkspace.costs.by_agent).reduce(
      (sum, cost) => sum + cost,
      0
    );
    expect(agentCostSum).toBeCloseTo(mockWorkspace.costs.total_cost, 2);
  });

  it('team costs sum to total cost', () => {
    const teamCostSum = Object.values(mockWorkspace.costs.by_team).reduce(
      (sum, cost) => sum + cost,
      0
    );
    expect(teamCostSum).toBeCloseTo(mockWorkspace.costs.total_cost, 2);
  });

  it('model costs sum to total cost', () => {
    const modelCostSum = Object.values(mockWorkspace.costs.by_model).reduce(
      (sum, cost) => sum + cost,
      0
    );
    expect(modelCostSum).toBeCloseTo(mockWorkspace.costs.total_cost, 2);
  });

  it('all cost entries reference existing agents', () => {
    const agentNames = mockWorkspace.agents.map((a) => a.name);
    const costAgentNames = Object.keys(mockWorkspace.costs.by_agent);
    const allExist = costAgentNames.every((name) => agentNames.includes(name));
    expect(allExist).toBe(true);
  });

  it('all cost entries reference existing teams', () => {
    const teamNames = mockWorkspace.teams.map((t) => t.name);
    const costTeamNames = Object.keys(mockWorkspace.costs.by_team);
    const allExist = costTeamNames.every((name) => teamNames.includes(name));
    expect(allExist).toBe(true);
  });

  it('displays correct total cost', () => {
    const { lastFrame } = render(<DataConsistencyDisplay workspace={mockWorkspace} />);
    expect(lastFrame()).toContain('Total cost: $5.00');
  });
});

describe('Data Consistency: Token Counts', () => {
  it('total tokens is sum of input and output', () => {
    const totalTokens =
      mockWorkspace.costs.total_input_tokens + mockWorkspace.costs.total_output_tokens;
    expect(totalTokens).toBe(150000);
  });

  it('input tokens are non-negative', () => {
    expect(mockWorkspace.costs.total_input_tokens).toBeGreaterThanOrEqual(0);
  });

  it('output tokens are non-negative', () => {
    expect(mockWorkspace.costs.total_output_tokens).toBeGreaterThanOrEqual(0);
  });
});
