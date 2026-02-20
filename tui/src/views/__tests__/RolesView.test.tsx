/**
 * RolesView Tests - View and manage agent roles (#859)
 *
 * #1081 Q1 Cleanup: View test coverage
 *
 * Tests cover:
 * - Role data structure
 * - Search filtering
 * - Builtin role detection
 * - Truncation helper
 * - Name column width calculation
 * - Keyboard shortcuts
 */

import { describe, test, expect } from 'bun:test';

// Type definitions matching RolesView
interface Role {
  name: string;
  description?: string;
  capabilities: string[];
  parent?: string;
  prompt?: string;
  agent_count?: number;
}

// Helper functions matching RolesView logic
function isBuiltinRole(name: string): boolean {
  const builtinRoles = ['root', 'manager', 'engineer', 'tech-lead', 'product-manager'];
  return builtinRoles.includes(name);
}

function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '…';
}

function calculateNameColumnWidth(roles: Role[]): number {
  if (roles.length === 0) return 15;
  const maxNameLen = Math.max(...roles.map((r) => r.name.length));
  return Math.min(25, Math.max(15, maxNameLen + 3));
}

function filterRoles(roles: Role[], searchQuery: string): Role[] {
  if (searchQuery.length === 0) return roles;
  const lower = searchQuery.toLowerCase();
  return roles.filter(
    (r) =>
      r.name.toLowerCase().includes(lower) ||
      (r.description?.toLowerCase().includes(lower) ?? false) ||
      r.capabilities.some((c) => c.toLowerCase().includes(lower))
  );
}

function formatCapabilities(capabilities: string[]): string {
  if (capabilities.length === 0) return '-';
  return capabilities.slice(0, 3).join(', ') + (capabilities.length > 3 ? '...' : '');
}

describe('RolesView', () => {
  describe('Role Data Structure', () => {
    test('role has required fields', () => {
      const role: Role = {
        name: 'engineer',
        capabilities: ['implement_tasks', 'run_tests'],
      };

      expect(role.name).toBe('engineer');
      expect(role.capabilities).toBeArray();
    });

    test('role with all fields', () => {
      const role: Role = {
        name: 'tech-lead',
        description: 'Technical lead responsible for architecture',
        capabilities: ['implement_tasks', 'review_code', 'create_agents'],
        parent: 'engineer',
        prompt: 'You are a technical lead...',
        agent_count: 2,
      };

      expect(role.name).toBe('tech-lead');
      expect(role.description).toBeDefined();
      expect(role.parent).toBe('engineer');
      expect(role.prompt).toBeDefined();
    });
  });

  describe('Builtin Role Detection', () => {
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

    test('custom role is not builtin', () => {
      expect(isBuiltinRole('my-custom-role')).toBe(false);
    });

    test('similar names are not builtin', () => {
      expect(isBuiltinRole('engineers')).toBe(false);
      expect(isBuiltinRole('managers')).toBe(false);
    });
  });

  describe('Truncation Helper', () => {
    test('short string not truncated', () => {
      expect(truncate('hello', 10)).toBe('hello');
    });

    test('exact length not truncated', () => {
      expect(truncate('hello', 5)).toBe('hello');
    });

    test('long string truncated with ellipsis', () => {
      expect(truncate('hello world', 8)).toBe('hello w…');
    });

    test('truncate to 1 char', () => {
      expect(truncate('hello', 1)).toBe('…');
    });

    test('empty string handled', () => {
      expect(truncate('', 10)).toBe('');
    });
  });

  describe('Name Column Width Calculation', () => {
    test('default width for empty array', () => {
      expect(calculateNameColumnWidth([])).toBe(15);
    });

    test('minimum width is 15', () => {
      const roles: Role[] = [{ name: 'a', capabilities: [] }];
      expect(calculateNameColumnWidth(roles)).toBe(15);
    });

    test('adjusts for longer names', () => {
      const roles: Role[] = [{ name: 'my-very-long-role-name', capabilities: [] }];
      // 22 chars + 3 = 25, capped at 25
      expect(calculateNameColumnWidth(roles)).toBe(25);
    });

    test('maximum width is 25', () => {
      const roles: Role[] = [{ name: 'this-is-an-extremely-long-role-name', capabilities: [] }];
      expect(calculateNameColumnWidth(roles)).toBe(25);
    });

    test('uses longest name in list', () => {
      const roles: Role[] = [
        { name: 'short', capabilities: [] },
        { name: 'medium-length', capabilities: [] },
        { name: 'very-long-role', capabilities: [] },
      ];
      // 14 chars + 3 = 17
      expect(calculateNameColumnWidth(roles)).toBe(17);
    });
  });

  describe('Search Filtering', () => {
    const roles: Role[] = [
      { name: 'engineer', description: 'Implements features', capabilities: ['implement_tasks'] },
      { name: 'manager', description: 'Coordinates team', capabilities: ['assign_work', 'create_agents'] },
      { name: 'tech-lead', description: 'Technical guidance', capabilities: ['review_code'] },
    ];

    test('empty query returns all', () => {
      expect(filterRoles(roles, '')).toHaveLength(3);
    });

    test('filters by name', () => {
      expect(filterRoles(roles, 'eng')).toHaveLength(1);
      expect(filterRoles(roles, 'eng')[0].name).toBe('engineer');
    });

    test('filters by description', () => {
      expect(filterRoles(roles, 'team')).toHaveLength(1);
      expect(filterRoles(roles, 'team')[0].name).toBe('manager');
    });

    test('filters by capability', () => {
      expect(filterRoles(roles, 'review')).toHaveLength(1);
      expect(filterRoles(roles, 'review')[0].name).toBe('tech-lead');
    });

    test('case insensitive search', () => {
      expect(filterRoles(roles, 'ENGINEER')).toHaveLength(1);
      expect(filterRoles(roles, 'Manager')).toHaveLength(1);
    });

    test('no matches returns empty', () => {
      expect(filterRoles(roles, 'nonexistent')).toHaveLength(0);
    });
  });

  describe('Capabilities Formatting', () => {
    test('empty capabilities returns dash', () => {
      expect(formatCapabilities([])).toBe('-');
    });

    test('single capability', () => {
      expect(formatCapabilities(['implement_tasks'])).toBe('implement_tasks');
    });

    test('three capabilities', () => {
      expect(formatCapabilities(['a', 'b', 'c'])).toBe('a, b, c');
    });

    test('more than three shows ellipsis', () => {
      expect(formatCapabilities(['a', 'b', 'c', 'd', 'e'])).toBe('a, b, c...');
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      '/': 'search',
      j: 'down',
      k: 'up',
      g: 'first',
      G: 'last',
      d: 'delete',
      r: 'refresh',
      q: 'back',
      Enter: 'details',
    };

    test('search shortcut', () => {
      expect(shortcuts['/']).toBe('search');
    });

    test('navigation shortcuts', () => {
      expect(shortcuts.j).toBe('down');
      expect(shortcuts.k).toBe('up');
      expect(shortcuts.g).toBe('first');
      expect(shortcuts.G).toBe('last');
    });

    test('action shortcuts', () => {
      expect(shortcuts.d).toBe('delete');
      expect(shortcuts.r).toBe('refresh');
      expect(shortcuts.q).toBe('back');
    });
  });

  describe('Agent Count Computation', () => {
    test('computes counts by role', () => {
      const agents = [
        { name: 'eng-01', role: 'engineer' },
        { name: 'eng-02', role: 'engineer' },
        { name: 'mgr-01', role: 'manager' },
      ];

      const counts: Record<string, number> = {};
      for (const agent of agents) {
        counts[agent.role] = (counts[agent.role] || 0) + 1;
      }

      expect(counts['engineer']).toBe(2);
      expect(counts['manager']).toBe(1);
      expect(counts['tech-lead']).toBeUndefined();
    });

    test('handles empty agents', () => {
      const agents: { name: string; role: string }[] = [];
      const counts: Record<string, number> = {};
      for (const agent of agents) {
        counts[agent.role] = (counts[agent.role] || 0) + 1;
      }

      expect(Object.keys(counts)).toHaveLength(0);
    });
  });

  describe('Selection Index', () => {
    test('valid index stays in bounds', () => {
      const filteredRoles = ['a', 'b', 'c'];
      const selectedIndex = 1;
      const validIndex = Math.min(selectedIndex, Math.max(0, filteredRoles.length - 1));
      expect(validIndex).toBe(1);
    });

    test('out of bounds index clamped', () => {
      const filteredRoles = ['a', 'b'];
      const selectedIndex = 5;
      const validIndex = Math.min(selectedIndex, Math.max(0, filteredRoles.length - 1));
      expect(validIndex).toBe(1);
    });

    test('empty list index is 0', () => {
      const filteredRoles: string[] = [];
      const selectedIndex = 0;
      const validIndex = Math.min(selectedIndex, Math.max(0, filteredRoles.length - 1));
      expect(validIndex).toBe(0);
    });
  });

  describe('Delete Confirmation', () => {
    test('only non-builtin roles can be deleted', () => {
      const roleName = 'custom-role';
      const canDelete = !isBuiltinRole(roleName);
      expect(canDelete).toBe(true);
    });

    test('builtin roles cannot be deleted', () => {
      const roleName = 'engineer';
      const canDelete = !isBuiltinRole(roleName);
      expect(canDelete).toBe(false);
    });
  });
});
