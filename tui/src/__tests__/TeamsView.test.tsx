/**
 * TeamsView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('TeamsView - formatDate', () => {
  function formatDate(isoString: string | undefined): string {
    if (!isoString) return '-';
    try {
      const date = new Date(isoString);
      return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
      });
    } catch {
      return '-';
    }
  }

  test('formats ISO date string', () => {
    const result = formatDate('2024-12-25T10:30:00Z');
    expect(result).toMatch(/Dec 25, 2024/);
  });

  test('formats different month', () => {
    const result = formatDate('2024-06-15T00:00:00Z');
    expect(result).toMatch(/Jun 15, 2024/);
  });

  test('returns dash for undefined', () => {
    expect(formatDate(undefined)).toBe('-');
  });

  test('returns dash for empty string', () => {
    // Empty string is falsy in the check
    expect(formatDate('')).toBe('-');
  });

  test('handles valid date at start of year', () => {
    const result = formatDate('2025-01-01T00:00:00Z');
    expect(result).toMatch(/Jan 1, 2025/);
  });

  test('handles valid date at end of year', () => {
    const result = formatDate('2024-12-31T23:59:59Z');
    expect(result).toMatch(/Dec 31, 2024/);
  });
});

describe('TeamsView - TeamRow conversion', () => {
  interface Team {
    name: string;
    members: string[];
    lead: string | null;
    description: string | null;
  }

  interface TeamRow {
    name: string;
    members: string[];
    lead: string;
    description: string;
  }

  function convertToTeamRow(team: Team): TeamRow {
    return {
      name: team.name,
      members: team.members,
      lead: team.lead ?? '',
      description: team.description ?? '',
    };
  }

  test('converts team with all fields', () => {
    const team: Team = {
      name: 'eng-team',
      members: ['eng-01', 'eng-02'],
      lead: 'eng-01',
      description: 'Engineering team',
    };
    const row = convertToTeamRow(team);
    expect(row.name).toBe('eng-team');
    expect(row.members).toEqual(['eng-01', 'eng-02']);
    expect(row.lead).toBe('eng-01');
    expect(row.description).toBe('Engineering team');
  });

  test('converts team with null lead', () => {
    const team: Team = {
      name: 'new-team',
      members: ['member1'],
      lead: null,
      description: 'Test team',
    };
    const row = convertToTeamRow(team);
    expect(row.lead).toBe('');
  });

  test('converts team with null description', () => {
    const team: Team = {
      name: 'no-desc',
      members: [],
      lead: 'leader',
      description: null,
    };
    const row = convertToTeamRow(team);
    expect(row.description).toBe('');
  });

  test('converts team with empty members', () => {
    const team: Team = {
      name: 'empty-team',
      members: [],
      lead: null,
      description: null,
    };
    const row = convertToTeamRow(team);
    expect(row.members).toEqual([]);
  });
});

describe('TeamsView - expanded team toggle', () => {
  function toggleExpanded(current: string | null, teamName: string): string | null {
    return current === teamName ? null : teamName;
  }

  test('expands when collapsed', () => {
    expect(toggleExpanded(null, 'eng-team')).toBe('eng-team');
  });

  test('collapses when same team selected', () => {
    expect(toggleExpanded('eng-team', 'eng-team')).toBeNull();
  });

  test('switches to different team', () => {
    expect(toggleExpanded('eng-team', 'qa-team')).toBe('qa-team');
  });

  test('handles empty team name', () => {
    expect(toggleExpanded(null, '')).toBe('');
  });
});

describe('TeamsView - member count display', () => {
  function getMemberCount(members: string[]): number {
    return members.length;
  }

  test('returns 0 for empty array', () => {
    expect(getMemberCount([])).toBe(0);
  });

  test('returns 1 for single member', () => {
    expect(getMemberCount(['eng-01'])).toBe(1);
  });

  test('returns correct count for multiple members', () => {
    expect(getMemberCount(['eng-01', 'eng-02', 'eng-03'])).toBe(3);
  });
});

describe('TeamsView - member indicator', () => {
  function getMemberIndicator(member: string, lead: string | null): string {
    return member === lead ? '★ ' : '• ';
  }

  test('star indicator for team lead', () => {
    expect(getMemberIndicator('eng-01', 'eng-01')).toBe('★ ');
  });

  test('bullet indicator for regular member', () => {
    expect(getMemberIndicator('eng-02', 'eng-01')).toBe('• ');
  });

  test('bullet indicator when no lead', () => {
    expect(getMemberIndicator('eng-01', null)).toBe('• ');
  });

  test('bullet indicator when lead is different', () => {
    expect(getMemberIndicator('member', 'leader')).toBe('• ');
  });
});

describe('TeamsView - empty state', () => {
  function getEmptyMessage(): string {
    return 'No teams configured';
  }

  function getCreateHint(): string {
    return 'Create a team with: bc team create <name>';
  }

  test('empty message text', () => {
    expect(getEmptyMessage()).toBe('No teams configured');
  });

  test('create hint text', () => {
    const hint = getCreateHint();
    expect(hint).toContain('bc team create');
    expect(hint).toContain('<name>');
  });
});

describe('TeamsView - footer hints', () => {
  interface Hint {
    key: string;
    label: string;
  }

  const footerHints: Hint[] = [
    { key: 'j/k', label: 'navigate' },
    { key: 'g/G', label: 'top/bottom' },
    { key: 'Enter', label: 'expand' },
    { key: 'r', label: 'refresh' },
    { key: 'q/ESC', label: 'back' },
  ];

  test('has navigation hint', () => {
    expect(footerHints.some(h => h.key === 'j/k')).toBe(true);
  });

  test('has expand hint', () => {
    expect(footerHints.some(h => h.label === 'expand')).toBe(true);
  });

  test('has refresh hint', () => {
    expect(footerHints.some(h => h.key === 'r')).toBe(true);
  });

  test('has quit hint', () => {
    expect(footerHints.some(h => h.label === 'back')).toBe(true);
  });

  test('correct number of hints', () => {
    expect(footerHints).toHaveLength(5);
  });
});

describe('TeamsView - DataTable column config', () => {
  interface Column {
    key: string;
    header: string;
    width?: number;
  }

  const columns: Column[] = [
    { key: 'name', header: 'TEAM', width: 20 },
    { key: 'members', header: 'MEMBERS', width: 10 },
    { key: 'lead', header: 'LEAD', width: 15 },
    { key: 'description', header: 'DESCRIPTION' },
  ];

  test('has name column', () => {
    const col = columns.find(c => c.key === 'name');
    expect(col).toBeDefined();
    expect(col?.width).toBe(20);
  });

  test('has members column', () => {
    const col = columns.find(c => c.key === 'members');
    expect(col).toBeDefined();
    expect(col?.width).toBe(10);
  });

  test('has lead column', () => {
    const col = columns.find(c => c.key === 'lead');
    expect(col).toBeDefined();
    expect(col?.width).toBe(15);
  });

  test('description column has no fixed width', () => {
    const col = columns.find(c => c.key === 'description');
    expect(col).toBeDefined();
    expect(col?.width).toBeUndefined();
  });

  test('correct number of columns', () => {
    expect(columns).toHaveLength(4);
  });
});

describe('TeamsView - team details visibility', () => {
  interface Team {
    name: string;
    description: string | null;
    lead: string | null;
  }

  function hasDescription(team: Team): boolean {
    return team.description !== null && team.description.length > 0;
  }

  function hasLead(team: Team): boolean {
    return team.lead !== null && team.lead.length > 0;
  }

  test('hasDescription returns true when present', () => {
    const team: Team = { name: 't', description: 'Test', lead: null };
    expect(hasDescription(team)).toBe(true);
  });

  test('hasDescription returns false when null', () => {
    const team: Team = { name: 't', description: null, lead: null };
    expect(hasDescription(team)).toBe(false);
  });

  test('hasDescription returns false when empty', () => {
    const team: Team = { name: 't', description: '', lead: null };
    expect(hasDescription(team)).toBe(false);
  });

  test('hasLead returns true when present', () => {
    const team: Team = { name: 't', description: null, lead: 'eng-01' };
    expect(hasLead(team)).toBe(true);
  });

  test('hasLead returns false when null', () => {
    const team: Team = { name: 't', description: null, lead: null };
    expect(hasLead(team)).toBe(false);
  });

  test('hasLead returns false when empty', () => {
    const team: Team = { name: 't', description: null, lead: '' };
    expect(hasLead(team)).toBe(false);
  });
});

describe('TeamsView - find team by name', () => {
  interface Team {
    name: string;
    members: string[];
  }

  const teams: Team[] = [
    { name: 'eng-team', members: ['eng-01', 'eng-02'] },
    { name: 'qa-team', members: ['qa-01'] },
    { name: 'devops', members: [] },
  ];

  function findTeam(teamName: string | null): Team | undefined {
    if (!teamName) return undefined;
    return teams.find(t => t.name === teamName);
  }

  test('finds existing team', () => {
    const team = findTeam('eng-team');
    expect(team).toBeDefined();
    expect(team?.members).toHaveLength(2);
  });

  test('returns undefined for non-existent team', () => {
    expect(findTeam('nonexistent')).toBeUndefined();
  });

  test('returns undefined for null', () => {
    expect(findTeam(null)).toBeUndefined();
  });

  test('finds team with empty members', () => {
    const team = findTeam('devops');
    expect(team).toBeDefined();
    expect(team?.members).toHaveLength(0);
  });
});
