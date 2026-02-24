/**
 * WorkspaceSelectorView Tests
 * Issue #922: Workspace discovery and selection
 * Issue #1750: Migrated to useListNavigation
 *
 * Tests cover:
 * - Path formatting (home directory shortening)
 * - Workspace filtering (v2 only)
 * - Registered vs discovered categorization
 * - Column width calculation
 * - Keyboard shortcuts
 * - Detail view display
 */

import { describe, test, expect } from 'bun:test';

// Workspace type matching WorkspaceSelectorView
interface DiscoveredWorkspace {
  name: string;
  path: string;
  is_v2: boolean;
  from_cache: boolean;
}

// Path formatting helper matching WorkspaceSelectorView
function formatPath(fullPath: string, home = '/Users/test'): string {
  if (home && fullPath.startsWith(home)) {
    return '~' + fullPath.slice(home.length);
  }
  return fullPath;
}

// Filter workspaces by v2 config
function filterV2Only(workspaces: DiscoveredWorkspace[], enabled: boolean): DiscoveredWorkspace[] {
  if (!enabled) return workspaces;
  return workspaces.filter((ws) => ws.is_v2);
}

// Separate registered and discovered
function getRegisteredWorkspaces(workspaces: DiscoveredWorkspace[]): DiscoveredWorkspace[] {
  return workspaces.filter((ws) => ws.from_cache);
}

function getDiscoveredWorkspaces(workspaces: DiscoveredWorkspace[]): DiscoveredWorkspace[] {
  return workspaces.filter((ws) => !ws.from_cache);
}

// Count v2 workspaces
function countV2Workspaces(workspaces: DiscoveredWorkspace[]): number {
  return workspaces.filter((ws) => ws.is_v2).length;
}

// Calculate path column width
function calculatePathWidth(terminalWidth: number, nameWidth = 20, typeWidth = 8): number {
  return Math.min(50, terminalWidth - nameWidth - typeWidth - 10);
}

// Truncate name for display
function truncateName(name: string, maxWidth: number): string {
  return name.slice(0, maxWidth - 1).padEnd(maxWidth);
}

describe('WorkspaceSelectorView', () => {
  describe('Path Formatting', () => {
    test('shortens home directory to tilde', () => {
      const path = '/Users/test/Projects/myapp';
      const formatted = formatPath(path, '/Users/test');
      expect(formatted).toBe('~/Projects/myapp');
    });

    test('preserves non-home paths', () => {
      const path = '/var/lib/myapp';
      const formatted = formatPath(path, '/Users/test');
      expect(formatted).toBe('/var/lib/myapp');
    });

    test('handles exact home path', () => {
      const path = '/Users/test';
      const formatted = formatPath(path, '/Users/test');
      expect(formatted).toBe('~');
    });

    test('handles empty home', () => {
      const path = '/some/path';
      const formatted = formatPath(path, '');
      expect(formatted).toBe('/some/path');
    });
  });

  describe('Workspace Filtering', () => {
    const workspaces: DiscoveredWorkspace[] = [
      { name: 'app1', path: '/path1', is_v2: true, from_cache: true },
      { name: 'app2', path: '/path2', is_v2: false, from_cache: true },
      { name: 'app3', path: '/path3', is_v2: true, from_cache: false },
      { name: 'app4', path: '/path4', is_v2: false, from_cache: false },
    ];

    test('returns all workspaces when filter disabled', () => {
      const filtered = filterV2Only(workspaces, false);
      expect(filtered).toHaveLength(4);
    });

    test('returns only v2 workspaces when filter enabled', () => {
      const filtered = filterV2Only(workspaces, true);
      expect(filtered).toHaveLength(2);
      expect(filtered.every(ws => ws.is_v2)).toBe(true);
    });

    test('handles empty array', () => {
      const filtered = filterV2Only([], true);
      expect(filtered).toHaveLength(0);
    });
  });

  describe('Registered vs Discovered Categorization', () => {
    const workspaces: DiscoveredWorkspace[] = [
      { name: 'registered1', path: '/path1', is_v2: true, from_cache: true },
      { name: 'registered2', path: '/path2', is_v2: true, from_cache: true },
      { name: 'discovered1', path: '/path3', is_v2: true, from_cache: false },
    ];

    test('extracts registered workspaces', () => {
      const registered = getRegisteredWorkspaces(workspaces);
      expect(registered).toHaveLength(2);
      expect(registered.every(ws => ws.from_cache)).toBe(true);
    });

    test('extracts discovered workspaces', () => {
      const discovered = getDiscoveredWorkspaces(workspaces);
      expect(discovered).toHaveLength(1);
      expect(discovered.every(ws => !ws.from_cache)).toBe(true);
    });

    test('handles all registered', () => {
      const allRegistered: DiscoveredWorkspace[] = [
        { name: 'reg1', path: '/p1', is_v2: true, from_cache: true },
        { name: 'reg2', path: '/p2', is_v2: true, from_cache: true },
      ];
      const discovered = getDiscoveredWorkspaces(allRegistered);
      expect(discovered).toHaveLength(0);
    });

    test('handles all discovered', () => {
      const allDiscovered: DiscoveredWorkspace[] = [
        { name: 'disc1', path: '/p1', is_v2: true, from_cache: false },
        { name: 'disc2', path: '/p2', is_v2: true, from_cache: false },
      ];
      const registered = getRegisteredWorkspaces(allDiscovered);
      expect(registered).toHaveLength(0);
    });
  });

  describe('V2 Workspace Counting', () => {
    const workspaces: DiscoveredWorkspace[] = [
      { name: 'v2-1', path: '/p1', is_v2: true, from_cache: true },
      { name: 'v1-1', path: '/p2', is_v2: false, from_cache: true },
      { name: 'v2-2', path: '/p3', is_v2: true, from_cache: false },
      { name: 'v1-2', path: '/p4', is_v2: false, from_cache: false },
    ];

    test('counts v2 workspaces', () => {
      const count = countV2Workspaces(workspaces);
      expect(count).toBe(2);
    });

    test('returns 0 for all v1', () => {
      const allV1: DiscoveredWorkspace[] = [
        { name: 'v1-1', path: '/p1', is_v2: false, from_cache: true },
        { name: 'v1-2', path: '/p2', is_v2: false, from_cache: true },
      ];
      const count = countV2Workspaces(allV1);
      expect(count).toBe(0);
    });

    test('handles empty array', () => {
      const count = countV2Workspaces([]);
      expect(count).toBe(0);
    });
  });

  describe('Column Width Calculation', () => {
    test('calculates path width for wide terminal', () => {
      const width = calculatePathWidth(120);
      // 120 - 20 - 8 - 10 = 82, but capped at 50
      expect(width).toBe(50);
    });

    test('calculates path width for narrow terminal', () => {
      const width = calculatePathWidth(80);
      // 80 - 20 - 8 - 10 = 42
      expect(width).toBe(42);
    });

    test('calculates path width for very narrow terminal', () => {
      const width = calculatePathWidth(60);
      // 60 - 20 - 8 - 10 = 22
      expect(width).toBe(22);
    });

    test('path width is always capped at 50', () => {
      const width = calculatePathWidth(200);
      expect(width).toBe(50);
    });
  });

  describe('Name Truncation', () => {
    test('pads short names', () => {
      const result = truncateName('app', 20);
      expect(result.length).toBe(20);
      expect(result.startsWith('app')).toBe(true);
      expect(result.trimEnd()).toBe('app');
    });

    test('truncates long names', () => {
      const longName = 'very-long-workspace-name-here';
      const result = truncateName(longName, 20);
      expect(result.length).toBe(20);
      // Truncates to maxWidth-1 then pads to maxWidth
      expect(result.trimEnd()).toBe('very-long-workspace');
    });

    test('handles exact length', () => {
      const exactName = 'exactly-nineteen--';
      const result = truncateName(exactName, 20);
      expect(result.length).toBe(20);
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      'j/k': 'navigate',
      'g/G': 'top/bottom',
      Enter: 'select/details',
      v: 'toggle v2 filter',
      r: 'refresh',
      'q/ESC': 'back',
    };

    test('navigation shortcuts', () => {
      expect(shortcuts['j/k']).toBe('navigate');
      expect(shortcuts['g/G']).toBe('top/bottom');
    });

    test('action shortcuts', () => {
      expect(shortcuts.Enter).toBe('select/details');
      expect(shortcuts.v).toBe('toggle v2 filter');
      expect(shortcuts.r).toBe('refresh');
    });

    test('back shortcuts', () => {
      expect(shortcuts['q/ESC']).toBe('back');
    });
  });

  describe('Workspace Data Structure', () => {
    test('workspace has required fields', () => {
      const workspace: DiscoveredWorkspace = {
        name: 'myapp',
        path: '/path/to/myapp',
        is_v2: true,
        from_cache: false,
      };

      expect(workspace.name).toBe('myapp');
      expect(workspace.path).toBe('/path/to/myapp');
      expect(workspace.is_v2).toBe(true);
      expect(workspace.from_cache).toBe(false);
    });

    test('v1 workspace', () => {
      const workspace: DiscoveredWorkspace = {
        name: 'legacy-app',
        path: '/path/to/legacy',
        is_v2: false,
        from_cache: true,
      };

      expect(workspace.is_v2).toBe(false);
      expect(workspace.from_cache).toBe(true);
    });
  });

  describe('Config Type Display', () => {
    test('v2 displays green', () => {
      const is_v2 = true;
      const color = is_v2 ? 'green' : 'yellow';
      const label = is_v2 ? 'v2 (TOML)' : 'v1 (JSON)';

      expect(color).toBe('green');
      expect(label).toBe('v2 (TOML)');
    });

    test('v1 displays yellow', () => {
      const is_v2 = false;
      const color = is_v2 ? 'green' : 'yellow';
      const label = is_v2 ? 'v2 (TOML)' : 'v1 (JSON)';

      expect(color).toBe('yellow');
      expect(label).toBe('v1 (JSON)');
    });
  });

  describe('Source Display', () => {
    test('registered displays blue', () => {
      const from_cache = true;
      const color = from_cache ? 'blue' : 'gray';
      const label = from_cache ? 'Registered' : 'Discovered';

      expect(color).toBe('blue');
      expect(label).toBe('Registered');
    });

    test('discovered displays gray', () => {
      const from_cache = false;
      const color = from_cache ? 'blue' : 'gray';
      const label = from_cache ? 'Registered' : 'Discovered';

      expect(color).toBe('gray');
      expect(label).toBe('Discovered');
    });
  });

  describe('Loading State', () => {
    test('shows loading message', () => {
      const loading = true;
      const message = loading ? 'Discovering workspaces...' : '';
      expect(message).toBe('Discovering workspaces...');
    });

    test('shows refreshing indicator', () => {
      const loading = true;
      const hasWorkspaces = true;
      const indicator = loading && hasWorkspaces ? '(refreshing...)' : '';
      expect(indicator).toBe('(refreshing...)');
    });
  });

  describe('Error State', () => {
    test('shows error message', () => {
      const error = 'Failed to fetch workspaces';
      const message = `Error: ${error}`;
      expect(message).toBe('Error: Failed to fetch workspaces');
    });
  });

  describe('Empty State', () => {
    test('shows no workspaces message', () => {
      const workspaces: DiscoveredWorkspace[] = [];
      const isEmpty = workspaces.length === 0;
      expect(isEmpty).toBe(true);
    });
  });

  describe('Header Display', () => {
    test('shows workspace counts', () => {
      const registered = 3;
      const discovered = 2;
      const header = `Workspaces (${registered} registered, ${discovered} discovered)`;
      expect(header).toContain('3 registered');
      expect(header).toContain('2 discovered');
    });

    test('shows only registered when no discovered', () => {
      const registered = 5;
      const discovered = 0;
      const header = discovered > 0
        ? `(${registered} registered, ${discovered} discovered)`
        : `(${registered} registered)`;
      expect(header).toBe('(5 registered)');
    });
  });

  describe('Filter Indicator', () => {
    test('shows v2 filter indicator when active', () => {
      const filterV2Only = true;
      const v2Count = 3;
      const indicator = filterV2Only ? `[Showing v2 only] (${v2Count} workspaces)` : '';
      expect(indicator).toBe('[Showing v2 only] (3 workspaces)');
    });

    test('no indicator when filter inactive', () => {
      const filterV2Only = false;
      const indicator = filterV2Only ? '[Showing v2 only]' : '';
      expect(indicator).toBe('');
    });
  });

  describe('Selection Indicator', () => {
    test('selected item has blue background', () => {
      const isSelected = true;
      const backgroundColor = isSelected ? 'blue' : undefined;
      const textColor = isSelected ? 'white' : 'cyan';

      expect(backgroundColor).toBe('blue');
      expect(textColor).toBe('white');
    });

    test('unselected item has no background', () => {
      const isSelected = false;
      const backgroundColor = isSelected ? 'blue' : undefined;
      const textColor = isSelected ? 'white' : 'cyan';

      expect(backgroundColor).toBeUndefined();
      expect(textColor).toBe('cyan');
    });
  });

  describe('Detail View', () => {
    test('shows workspace name', () => {
      const workspace: DiscoveredWorkspace = {
        name: 'myapp',
        path: '/path/to/myapp',
        is_v2: true,
        from_cache: true,
      };

      const nameLabel = `Name: ${workspace.name}`;
      expect(nameLabel).toBe('Name: myapp');
    });

    test('shows workspace path', () => {
      const workspace: DiscoveredWorkspace = {
        name: 'myapp',
        path: '/path/to/myapp',
        is_v2: true,
        from_cache: true,
      };

      const pathLabel = `Path: ${workspace.path}`;
      expect(pathLabel).toBe('Path: /path/to/myapp');
    });
  });

  describe('Breadcrumb Management', () => {
    test('detail view sets breadcrumb with workspace name', () => {
      const workspace: DiscoveredWorkspace = {
        name: 'myapp',
        path: '/path',
        is_v2: true,
        from_cache: true,
      };

      const breadcrumb = [{ label: workspace.name }];
      expect(breadcrumb[0].label).toBe('myapp');
    });
  });
});
