/**
 * WorkspaceSelectorView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('WorkspaceSelectorView - formatPath', () => {
  // Mock HOME for testing
  const HOME = '/Users/testuser';

  function formatPath(fullPath: string, home = HOME): string {
    if (home && fullPath.startsWith(home)) {
      return '~' + fullPath.slice(home.length);
    }
    return fullPath;
  }

  test('replaces home directory with ~', () => {
    expect(formatPath('/Users/testuser/projects/myapp', HOME)).toBe('~/projects/myapp');
  });

  test('returns path unchanged if not under home', () => {
    expect(formatPath('/var/lib/app', HOME)).toBe('/var/lib/app');
  });

  test('handles root path', () => {
    expect(formatPath('/', HOME)).toBe('/');
  });

  test('handles home directory exactly', () => {
    expect(formatPath('/Users/testuser', HOME)).toBe('~');
  });

  test('handles nested paths', () => {
    expect(formatPath('/Users/testuser/a/b/c/d', HOME)).toBe('~/a/b/c/d');
  });

  test('handles empty home', () => {
    expect(formatPath('/some/path', '')).toBe('/some/path');
  });
});

describe('WorkspaceSelectorView - v2 filter', () => {
  interface Workspace {
    name: string;
    path: string;
    is_v2: boolean;
    from_cache: boolean;
  }

  const mockWorkspaces: Workspace[] = [
    { name: 'project-a', path: '/a', is_v2: true, from_cache: true },
    { name: 'project-b', path: '/b', is_v2: false, from_cache: true },
    { name: 'project-c', path: '/c', is_v2: true, from_cache: false },
    { name: 'project-d', path: '/d', is_v2: false, from_cache: false },
  ];

  function filterV2(workspaces: Workspace[], v2Only: boolean): Workspace[] {
    if (!v2Only) return workspaces;
    return workspaces.filter((ws) => ws.is_v2);
  }

  test('returns all when v2Only is false', () => {
    expect(filterV2(mockWorkspaces, false)).toHaveLength(4);
  });

  test('returns only v2 when v2Only is true', () => {
    const result = filterV2(mockWorkspaces, true);
    expect(result).toHaveLength(2);
    expect(result.every(ws => ws.is_v2)).toBe(true);
  });

  test('handles empty array', () => {
    expect(filterV2([], true)).toHaveLength(0);
  });
});

describe('WorkspaceSelectorView - registered vs discovered separation', () => {
  interface Workspace {
    name: string;
    from_cache: boolean;
  }

  const mockWorkspaces: Workspace[] = [
    { name: 'registered-1', from_cache: true },
    { name: 'registered-2', from_cache: true },
    { name: 'discovered-1', from_cache: false },
    { name: 'discovered-2', from_cache: false },
    { name: 'discovered-3', from_cache: false },
  ];

  function getRegistered(workspaces: Workspace[]): Workspace[] {
    return workspaces.filter((ws) => ws.from_cache);
  }

  function getDiscovered(workspaces: Workspace[]): Workspace[] {
    return workspaces.filter((ws) => !ws.from_cache);
  }

  test('separates registered workspaces', () => {
    const registered = getRegistered(mockWorkspaces);
    expect(registered).toHaveLength(2);
    expect(registered.every(ws => ws.from_cache)).toBe(true);
  });

  test('separates discovered workspaces', () => {
    const discovered = getDiscovered(mockWorkspaces);
    expect(discovered).toHaveLength(3);
    expect(discovered.every(ws => !ws.from_cache)).toBe(true);
  });

  test('handles all registered', () => {
    const allRegistered = mockWorkspaces.filter(ws => ws.from_cache);
    expect(getDiscovered(allRegistered)).toHaveLength(0);
  });

  test('handles all discovered', () => {
    const allDiscovered = mockWorkspaces.filter(ws => !ws.from_cache);
    expect(getRegistered(allDiscovered)).toHaveLength(0);
  });
});

describe('WorkspaceSelectorView - column width calculation', () => {
  const NAME_WIDTH = 20;
  const TYPE_WIDTH = 8;

  function calculatePathWidth(terminalWidth: number): number {
    return Math.min(50, terminalWidth - NAME_WIDTH - TYPE_WIDTH - 10);
  }

  test('wide terminal uses max path width', () => {
    expect(calculatePathWidth(120)).toBe(50);
  });

  test('standard terminal calculates remaining space', () => {
    expect(calculatePathWidth(80)).toBe(42); // 80 - 20 - 8 - 10 = 42
  });

  test('narrow terminal uses remaining space', () => {
    expect(calculatePathWidth(60)).toBe(22); // 60 - 20 - 8 - 10 = 22
  });
});

describe('WorkspaceSelectorView - name truncation', () => {
  const NAME_WIDTH = 20;

  function truncateName(name: string): string {
    return name.slice(0, NAME_WIDTH - 1).padEnd(NAME_WIDTH);
  }

  test('short name is padded', () => {
    const result = truncateName('project');
    expect(result.length).toBe(20);
    expect(result).toBe('project             ');
  });

  test('exact length name is padded by 1', () => {
    const result = truncateName('1234567890123456789'); // 19 chars
    expect(result.length).toBe(20);
  });

  test('long name is truncated', () => {
    const longName = 'very-long-project-name-here';
    const result = truncateName(longName);
    expect(result.length).toBe(20);
    expect(result).toBe('very-long-project-n ');
  });
});

describe('WorkspaceSelectorView - type formatting', () => {
  const TYPE_WIDTH = 8;

  function formatType(isV2: boolean): string {
    return (isV2 ? 'v2' : 'v1').padEnd(TYPE_WIDTH);
  }

  test('v2 is padded', () => {
    const result = formatType(true);
    expect(result).toBe('v2      ');
    expect(result.length).toBe(8);
  });

  test('v1 is padded', () => {
    const result = formatType(false);
    expect(result).toBe('v1      ');
    expect(result.length).toBe(8);
  });
});

describe('WorkspaceSelectorView - v2 count', () => {
  interface Workspace {
    is_v2: boolean;
  }

  function countV2(workspaces: Workspace[]): number {
    return workspaces.filter((ws) => ws.is_v2).length;
  }

  test('counts v2 workspaces', () => {
    const workspaces: Workspace[] = [
      { is_v2: true },
      { is_v2: false },
      { is_v2: true },
    ];
    expect(countV2(workspaces)).toBe(2);
  });

  test('returns 0 for all v1', () => {
    const workspaces: Workspace[] = [{ is_v2: false }, { is_v2: false }];
    expect(countV2(workspaces)).toBe(0);
  });

  test('returns 0 for empty', () => {
    expect(countV2([])).toBe(0);
  });
});

describe('WorkspaceSelectorView - footer hints', () => {
  function getFooterHint(hasOnSelect: boolean, filterV2Only: boolean): string {
    const selectAction = hasOnSelect ? 'select' : 'details';
    const filterAction = filterV2Only ? 'show all' : 'v2 only';
    return `j/k: nav | g/G: top/bottom | Enter: ${selectAction} | v: ${filterAction} | r: refresh | q/ESC: back`;
  }

  test('shows "select" when onSelect provided', () => {
    const hint = getFooterHint(true, false);
    expect(hint).toContain('Enter: select');
  });

  test('shows "details" when no onSelect', () => {
    const hint = getFooterHint(false, false);
    expect(hint).toContain('Enter: details');
  });

  test('shows "show all" when filter active', () => {
    const hint = getFooterHint(false, true);
    expect(hint).toContain('v: show all');
  });

  test('shows "v2 only" when filter inactive', () => {
    const hint = getFooterHint(false, false);
    expect(hint).toContain('v: v2 only');
  });
});

describe('WorkspaceSelectorView - separator display', () => {
  function shouldShowSeparator(registeredCount: number, discoveredCount: number): boolean {
    return registeredCount > 0 && discoveredCount > 0;
  }

  test('shows separator when both exist', () => {
    expect(shouldShowSeparator(2, 3)).toBe(true);
  });

  test('hides separator when only registered', () => {
    expect(shouldShowSeparator(2, 0)).toBe(false);
  });

  test('hides separator when only discovered', () => {
    expect(shouldShowSeparator(0, 3)).toBe(false);
  });

  test('hides separator when empty', () => {
    expect(shouldShowSeparator(0, 0)).toBe(false);
  });
});

describe('WorkspaceSelectorView - type color', () => {
  function getTypeColor(isV2: boolean): string {
    return isV2 ? 'green' : 'yellow';
  }

  test('v2 is green', () => {
    expect(getTypeColor(true)).toBe('green');
  });

  test('v1 is yellow', () => {
    expect(getTypeColor(false)).toBe('yellow');
  });
});

describe('WorkspaceSelectorView - source color', () => {
  function getSourceColor(fromCache: boolean): string {
    return fromCache ? 'blue' : 'gray';
  }

  test('registered is blue', () => {
    expect(getSourceColor(true)).toBe('blue');
  });

  test('discovered is gray', () => {
    expect(getSourceColor(false)).toBe('gray');
  });
});

describe('WorkspaceSelectorView - header counts', () => {
  interface Workspace {
    from_cache: boolean;
  }

  function getHeaderCounts(workspaces: Workspace[]): { registered: number; discovered: number } {
    const registered = workspaces.filter(ws => ws.from_cache).length;
    const discovered = workspaces.filter(ws => !ws.from_cache).length;
    return { registered, discovered };
  }

  test('counts both types', () => {
    const workspaces: Workspace[] = [
      { from_cache: true },
      { from_cache: true },
      { from_cache: false },
    ];
    const counts = getHeaderCounts(workspaces);
    expect(counts.registered).toBe(2);
    expect(counts.discovered).toBe(1);
  });
});

describe('WorkspaceSelectorView - empty state', () => {
  function getEmptyMessage(): string {
    return 'No workspaces found';
  }

  test('empty message text', () => {
    expect(getEmptyMessage()).toBe('No workspaces found');
  });
});
