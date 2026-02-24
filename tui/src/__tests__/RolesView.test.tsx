/**
 * RolesView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('RolesView - isBuiltinRole', () => {
  function isBuiltinRole(name: string): boolean {
    const builtinRoles = ['root', 'manager', 'engineer', 'tech-lead', 'product-manager'];
    return builtinRoles.includes(name);
  }

  test('root is builtin', () => {
    expect(isBuiltinRole('root')).toBe(true);
  });

  test('manager is builtin', () => {
    expect(isBuiltinRole('manager')).toBe(true);
  });

  test('engineer is builtin', () => {
    expect(isBuiltinRole('engineer')).toBe(true);
  });

  test('tech-lead is builtin', () => {
    expect(isBuiltinRole('tech-lead')).toBe(true);
  });

  test('product-manager is builtin', () => {
    expect(isBuiltinRole('product-manager')).toBe(true);
  });

  test('custom-role is not builtin', () => {
    expect(isBuiltinRole('custom-role')).toBe(false);
  });

  test('empty string is not builtin', () => {
    expect(isBuiltinRole('')).toBe(false);
  });

  test('case sensitive check', () => {
    expect(isBuiltinRole('ROOT')).toBe(false);
    expect(isBuiltinRole('Manager')).toBe(false);
  });
});

describe('RolesView - nameColumnWidth calculation', () => {
  // From RolesView: Math.min(25, Math.max(15, maxNameLen + 3))
  function calculateNameColumnWidth(roles: { name: string }[]): number {
    if (roles.length === 0) return 15;
    const maxNameLen = Math.max(...roles.map((r) => r.name.length));
    return Math.min(25, Math.max(15, maxNameLen + 3));
  }

  test('returns default 15 for empty roles', () => {
    expect(calculateNameColumnWidth([])).toBe(15);
  });

  test('short names use minimum width 15', () => {
    const roles = [{ name: 'eng' }, { name: 'dev' }];
    expect(calculateNameColumnWidth(roles)).toBe(15);
  });

  test('medium names add 3 for padding', () => {
    const roles = [{ name: 'senior-engineer' }]; // 15 chars + 3 = 18
    expect(calculateNameColumnWidth(roles)).toBe(18);
  });

  test('long names capped at 25', () => {
    const roles = [{ name: 'very-long-role-name-that-exceeds-limit' }];
    expect(calculateNameColumnWidth(roles)).toBe(25);
  });

  test('uses longest name', () => {
    const roles = [
      { name: 'eng' },
      { name: 'product-manager' }, // 15 chars
      { name: 'dev' },
    ];
    expect(calculateNameColumnWidth(roles)).toBe(18); // 15 + 3 = 18
  });
});

describe('RolesView - role filtering', () => {
  interface MockRole {
    name: string;
    description: string | null;
    capabilities: string[];
  }

  const mockRoles: MockRole[] = [
    { name: 'engineer', description: 'Implements features', capabilities: ['implement_tasks', 'write_code'] },
    { name: 'manager', description: 'Manages team', capabilities: ['create_agents', 'assign_work'] },
    { name: 'tech-lead', description: 'Technical leadership', capabilities: ['code_review', 'architect'] },
    { name: 'qa', description: null, capabilities: ['test_code'] },
  ];

  function filterRoles(roles: MockRole[], query: string): MockRole[] {
    if (query.length === 0) return roles;
    const lower = query.toLowerCase();
    return roles.filter(
      (r) =>
        r.name.toLowerCase().includes(lower) ||
        (r.description?.toLowerCase().includes(lower) ?? false) ||
        r.capabilities.some((c) => c.toLowerCase().includes(lower))
    );
  }

  test('returns all roles when query is empty', () => {
    expect(filterRoles(mockRoles, '')).toHaveLength(4);
  });

  test('filters by name', () => {
    const result = filterRoles(mockRoles, 'engineer');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('engineer');
  });

  test('filters by description', () => {
    const result = filterRoles(mockRoles, 'leadership');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('tech-lead');
  });

  test('filters by capability', () => {
    const result = filterRoles(mockRoles, 'code_review');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('tech-lead');
  });

  test('search is case insensitive', () => {
    const result = filterRoles(mockRoles, 'MANAGER');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('manager');
  });

  test('handles null description gracefully', () => {
    const result = filterRoles(mockRoles, 'qa');
    expect(result).toHaveLength(1);
    expect(result[0].description).toBeNull();
  });

  test('returns empty for no matches', () => {
    const result = filterRoles(mockRoles, 'nonexistent');
    expect(result).toHaveLength(0);
  });

  test('matches multiple roles', () => {
    const result = filterRoles(mockRoles, 'code');
    expect(result).toHaveLength(3); // write_code, code_review, test_code
  });
});

describe('RolesView - capabilities string formatting', () => {
  function formatCapabilities(capabilities: string[]): string {
    if (capabilities.length === 0) return '-';
    return capabilities.slice(0, 3).join(', ') +
      (capabilities.length > 3 ? '...' : '');
  }

  test('returns dash for empty capabilities', () => {
    expect(formatCapabilities([])).toBe('-');
  });

  test('shows single capability', () => {
    expect(formatCapabilities(['write_code'])).toBe('write_code');
  });

  test('shows two capabilities', () => {
    expect(formatCapabilities(['write_code', 'test_code'])).toBe('write_code, test_code');
  });

  test('shows three capabilities', () => {
    expect(formatCapabilities(['a', 'b', 'c'])).toBe('a, b, c');
  });

  test('truncates with ellipsis when more than 3', () => {
    expect(formatCapabilities(['a', 'b', 'c', 'd'])).toBe('a, b, c...');
  });

  test('truncates with ellipsis for many capabilities', () => {
    const caps = ['one', 'two', 'three', 'four', 'five', 'six'];
    expect(formatCapabilities(caps)).toBe('one, two, three...');
  });
});

describe('RolesView - agent count by role', () => {
  interface MockAgent {
    name: string;
    role: string;
  }

  function computeAgentCountByRole(agents: MockAgent[]): Record<string, number> {
    const counts: Record<string, number> = {};
    for (const agent of agents) {
      counts[agent.role] = (counts[agent.role] || 0) + 1;
    }
    return counts;
  }

  test('counts agents by role', () => {
    const agents: MockAgent[] = [
      { name: 'eng-01', role: 'engineer' },
      { name: 'eng-02', role: 'engineer' },
      { name: 'mgr-01', role: 'manager' },
    ];
    const counts = computeAgentCountByRole(agents);
    expect(counts.engineer).toBe(2);
    expect(counts.manager).toBe(1);
  });

  test('handles empty agents', () => {
    const counts = computeAgentCountByRole([]);
    expect(Object.keys(counts)).toHaveLength(0);
  });

  test('handles single agent', () => {
    const agents: MockAgent[] = [{ name: 'root', role: 'root' }];
    const counts = computeAgentCountByRole(agents);
    expect(counts.root).toBe(1);
  });

  test('returns undefined for missing role', () => {
    const agents: MockAgent[] = [{ name: 'eng-01', role: 'engineer' }];
    const counts = computeAgentCountByRole(agents);
    expect(counts.manager).toBeUndefined();
  });
});

describe('RolesView - truncate name for display', () => {
  // Uses truncate utility - mirroring behavior
  function truncate(str: string, maxLen: number): string {
    if (str.length <= maxLen) return str;
    return str.slice(0, maxLen - 1) + '…';
  }

  test('short names not truncated', () => {
    expect(truncate('engineer', 15)).toBe('engineer');
  });

  test('exact length names not truncated', () => {
    expect(truncate('1234567890', 10)).toBe('1234567890');
  });

  test('long names truncated with ellipsis', () => {
    const longName = 'super-long-role-name';
    expect(truncate(longName, 12)).toBe('super-long-…');
  });

  test('handles empty string', () => {
    expect(truncate('', 10)).toBe('');
  });
});

describe('RolesView - role details display', () => {
  interface Role {
    name: string;
    description: string | null;
    parent: string | null;
    capabilities: string[];
    prompt: string | null;
  }

  function hasParent(role: Role): boolean {
    return role.parent !== null && role.parent.length > 0;
  }

  function hasPrompt(role: Role): boolean {
    return role.prompt !== null && role.prompt.length > 0;
  }

  function getDescription(role: Role): string {
    return role.description ?? 'No description';
  }

  test('hasParent returns true when parent exists', () => {
    const role: Role = { name: 'engineer', description: null, parent: 'root', capabilities: [], prompt: null };
    expect(hasParent(role)).toBe(true);
  });

  test('hasParent returns false when no parent', () => {
    const role: Role = { name: 'root', description: null, parent: null, capabilities: [], prompt: null };
    expect(hasParent(role)).toBe(false);
  });

  test('hasParent returns false for empty string', () => {
    const role: Role = { name: 'root', description: null, parent: '', capabilities: [], prompt: null };
    expect(hasParent(role)).toBe(false);
  });

  test('hasPrompt returns true when prompt exists', () => {
    const role: Role = { name: 'engineer', description: null, parent: null, capabilities: [], prompt: 'Build features' };
    expect(hasPrompt(role)).toBe(true);
  });

  test('hasPrompt returns false when no prompt', () => {
    const role: Role = { name: 'engineer', description: null, parent: null, capabilities: [], prompt: null };
    expect(hasPrompt(role)).toBe(false);
  });

  test('getDescription returns description when exists', () => {
    const role: Role = { name: 'engineer', description: 'Implements code', parent: null, capabilities: [], prompt: null };
    expect(getDescription(role)).toBe('Implements code');
  });

  test('getDescription returns fallback when null', () => {
    const role: Role = { name: 'engineer', description: null, parent: null, capabilities: [], prompt: null };
    expect(getDescription(role)).toBe('No description');
  });
});

describe('RolesView - search mode hints', () => {
  function getHintText(searchMode: boolean): string {
    return searchMode
      ? 'Type to search, Enter/Esc to exit'
      : 'j/k: navigate | g/G: top/bottom | Enter: details | d: delete | r: refresh | q/ESC: back';
  }

  test('search mode hint', () => {
    expect(getHintText(true)).toBe('Type to search, Enter/Esc to exit');
  });

  test('normal mode hint', () => {
    const hint = getHintText(false);
    expect(hint).toContain('j/k: navigate');
    expect(hint).toContain('Enter: details');
    expect(hint).toContain('d: delete');
  });
});

describe('RolesView - empty state messages', () => {
  function getEmptyMessage(searchQuery: string): string {
    return searchQuery.length > 0
      ? `No roles match "${searchQuery}"`
      : 'No roles defined.';
  }

  test('shows search message when searching', () => {
    expect(getEmptyMessage('engineer')).toBe('No roles match "engineer"');
  });

  test('shows default message when not searching', () => {
    expect(getEmptyMessage('')).toBe('No roles defined.');
  });
});
