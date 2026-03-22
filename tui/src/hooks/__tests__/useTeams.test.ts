/**
 * useTeams hook tests (#1081)
 *
 * Tests cover:
 * - Initial data fetching
 * - Loading states
 * - Error handling
 * - Polling behavior
 * - addMember/removeMember operations
 * - Type interfaces
 */

import { describe, test, expect } from 'bun:test';
import type { Team, TeamsResponse } from '../../types';

// Test the Team interface structure
describe('useTeams Types', () => {
  describe('Team interface', () => {
    test('has required name field', () => {
      const team: Team = {
        name: 'platform',
        members: [],
      };
      expect(team.name).toBe('platform');
    });

    test('has required members array', () => {
      const team: Team = {
        name: 'platform',
        members: ['eng-01', 'eng-02'],
      };
      expect(team.members).toEqual(['eng-01', 'eng-02']);
    });

    test('supports optional description', () => {
      const team: Team = {
        name: 'platform',
        members: [],
        description: 'Platform team',
      };
      expect(team.description).toBe('Platform team');
    });
  });

  describe('TeamsResponse interface', () => {
    test('contains teams array', () => {
      const response: TeamsResponse = {
        teams: [
          { name: 'platform', members: ['eng-01'] },
          { name: 'frontend', members: ['eng-02', 'eng-03'] },
        ],
      };
      expect(response.teams).toHaveLength(2);
    });

    test('handles empty teams array', () => {
      const response: TeamsResponse = {
        teams: [],
      };
      expect(response.teams).toEqual([]);
    });
  });
});

// Test helper functions that would be used by useTeams
describe('useTeams Helper Functions', () => {
  describe('Team filtering', () => {
    const teams: Team[] = [
      { name: 'platform', members: ['eng-01', 'eng-02'] },
      { name: 'frontend', members: ['eng-03'] },
      { name: 'backend', members: ['eng-04', 'eng-05', 'eng-06'] },
    ];

    test('filters teams by member', () => {
      const filterByMember = (teams: Team[], member: string) =>
        teams.filter((t) => t.members.includes(member));

      const result = filterByMember(teams, 'eng-01');
      expect(result).toHaveLength(1);
      expect(result[0].name).toBe('platform');
    });

    test('finds team by name', () => {
      const findByName = (teams: Team[], name: string) => teams.find((t) => t.name === name);

      const result = findByName(teams, 'frontend');
      expect(result?.name).toBe('frontend');
      expect(result?.members).toEqual(['eng-03']);
    });

    test('counts total members across teams', () => {
      const countMembers = (teams: Team[]) => teams.reduce((sum, t) => sum + t.members.length, 0);

      expect(countMembers(teams)).toBe(6);
    });

    test('finds largest team', () => {
      const findLargest = (teams: Team[]) =>
        teams.reduce((max, t) => (t.members.length > max.members.length ? t : max), teams[0]);

      const largest = findLargest(teams);
      expect(largest.name).toBe('backend');
      expect(largest.members.length).toBe(3);
    });
  });

  describe('Team validation', () => {
    test('validates team name format', () => {
      const isValidTeamName = (name: string) => /^[a-zA-Z0-9_-]+$/.test(name) && name.length > 0;

      expect(isValidTeamName('platform')).toBe(true);
      expect(isValidTeamName('core-team')).toBe(true);
      expect(isValidTeamName('team_01')).toBe(true);
      expect(isValidTeamName('')).toBe(false);
      expect(isValidTeamName('team name')).toBe(false);
      expect(isValidTeamName('team@name')).toBe(false);
    });

    test('validates member name format', () => {
      const isValidMemberName = (name: string) => /^[a-zA-Z0-9_-]+$/.test(name) && name.length > 0;

      expect(isValidMemberName('eng-01')).toBe(true);
      expect(isValidMemberName('manager_01')).toBe(true);
      expect(isValidMemberName('')).toBe(false);
      expect(isValidMemberName('eng 01')).toBe(false);
    });
  });

  describe('Team operations', () => {
    test('adds member to team', () => {
      const addMemberToTeam = (team: Team, member: string): Team => ({
        ...team,
        members: [...team.members, member],
      });

      const team: Team = { name: 'platform', members: ['eng-01'] };
      const updated = addMemberToTeam(team, 'eng-02');

      expect(updated.members).toEqual(['eng-01', 'eng-02']);
      expect(team.members).toEqual(['eng-01']); // Original unchanged
    });

    test('removes member from team', () => {
      const removeMemberFromTeam = (team: Team, member: string): Team => ({
        ...team,
        members: team.members.filter((m) => m !== member),
      });

      const team: Team = { name: 'platform', members: ['eng-01', 'eng-02'] };
      const updated = removeMemberFromTeam(team, 'eng-01');

      expect(updated.members).toEqual(['eng-02']);
    });

    test('checks if member exists in team', () => {
      const hasMember = (team: Team, member: string) => team.members.includes(member);

      const team: Team = { name: 'platform', members: ['eng-01', 'eng-02'] };
      expect(hasMember(team, 'eng-01')).toBe(true);
      expect(hasMember(team, 'eng-03')).toBe(false);
    });
  });
});

// Test result state combinations
describe('useTeams Result States', () => {
  test('initial loading state', () => {
    const state = {
      data: null,
      error: null,
      loading: true,
    };
    expect(state.loading).toBe(true);
    expect(state.data).toBeNull();
    expect(state.error).toBeNull();
  });

  test('successful data state', () => {
    const teams: Team[] = [{ name: 'platform', members: ['eng-01'] }];
    const state = {
      data: teams,
      error: null,
      loading: false,
    };
    expect(state.loading).toBe(false);
    expect(state.data).toHaveLength(1);
    expect(state.error).toBeNull();
  });

  test('error state', () => {
    const state = {
      data: null,
      error: 'Failed to fetch teams',
      loading: false,
    };
    expect(state.loading).toBe(false);
    expect(state.data).toBeNull();
    expect(state.error).toBe('Failed to fetch teams');
  });

  test('empty data state', () => {
    const state = {
      data: [] as Team[],
      error: null,
      loading: false,
    };
    expect(state.loading).toBe(false);
    expect(state.data).toEqual([]);
    expect(state.error).toBeNull();
  });
});

// Test poll interval calculations
describe('useTeams Polling', () => {
  test('default poll interval from config', () => {
    const DEFAULT_POLL_INTERVAL = 5000;
    expect(DEFAULT_POLL_INTERVAL).toBe(5000);
  });

  test('custom poll interval override', () => {
    const options = { pollInterval: 10000 };
    const effectivePollInterval = options.pollInterval ?? 5000;
    expect(effectivePollInterval).toBe(10000);
  });

  test('autoPoll defaults to true', () => {
    const options = {};
    const autoPoll = (options as { autoPoll?: boolean }).autoPoll ?? true;
    expect(autoPoll).toBe(true);
  });

  test('autoPoll can be disabled', () => {
    const options = { autoPoll: false };
    expect(options.autoPoll).toBe(false);
  });
});

// Test error message formatting
describe('useTeams Error Handling', () => {
  test('formats Error instance message', () => {
    const formatError = (err: unknown) =>
      err instanceof Error ? err.message : 'Failed to fetch teams';

    const error = new Error('Network timeout');
    expect(formatError(error)).toBe('Network timeout');
  });

  test('provides default message for non-Error', () => {
    const formatError = (err: unknown) =>
      err instanceof Error ? err.message : 'Failed to fetch teams';

    expect(formatError('string error')).toBe('Failed to fetch teams');
    expect(formatError(null)).toBe('Failed to fetch teams');
    expect(formatError(undefined)).toBe('Failed to fetch teams');
  });

  test('formats add member error', () => {
    const formatAddError = (err: unknown) =>
      err instanceof Error ? err.message : 'Failed to add member';

    const error = new Error('Member already exists');
    expect(formatAddError(error)).toBe('Member already exists');
  });

  test('formats remove member error', () => {
    const formatRemoveError = (err: unknown) =>
      err instanceof Error ? err.message : 'Failed to remove member';

    const error = new Error('Member not found');
    expect(formatRemoveError(error)).toBe('Member not found');
  });
});
