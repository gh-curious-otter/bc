/**
 * WorktreesView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

/**
 * formatPath - Extract .bc/worktrees/... portion from full path
 * Mirrors the implementation in WorktreesView.tsx
 */
function formatPath(fullPath: string): string {
  const match = fullPath.match(/\.bc\/worktrees\/.+$/);
  return match ? match[0] : fullPath;
}

describe('WorktreesView - formatPath', () => {
  test('extracts .bc/worktrees portion', () => {
    const path = '/Users/user/project/.bc/worktrees/eng-01/main';
    expect(formatPath(path)).toBe('.bc/worktrees/eng-01/main');
  });

  test('handles nested directories', () => {
    const path = '/home/dev/projects/myapp/.bc/worktrees/agent-02/feature-branch';
    expect(formatPath(path)).toBe('.bc/worktrees/agent-02/feature-branch');
  });

  test('returns full path when no .bc/worktrees match', () => {
    const path = '/Users/user/some/other/path';
    expect(formatPath(path)).toBe('/Users/user/some/other/path');
  });

  test('handles root worktree path', () => {
    const path = '/.bc/worktrees/eng-01';
    expect(formatPath(path)).toBe('.bc/worktrees/eng-01');
  });

  test('handles Windows-style paths', () => {
    // Note: regex uses forward slashes, Windows paths may need conversion
    const path = 'C:/Users/dev/project/.bc/worktrees/eng-01';
    expect(formatPath(path)).toBe('.bc/worktrees/eng-01');
  });

  test('handles empty path', () => {
    expect(formatPath('')).toBe('');
  });
});

describe('WorktreesView - worktree filtering', () => {
  interface MockWorktree {
    agent: string;
    path: string;
    status: 'OK' | 'ORPHANED';
    branch?: string;
  }

  const mockWorktrees: MockWorktree[] = [
    { agent: 'eng-01', path: '/project/.bc/worktrees/eng-01', status: 'OK', branch: 'main' },
    { agent: 'eng-02', path: '/project/.bc/worktrees/eng-02', status: 'OK', branch: 'feature' },
    { agent: 'old-agent', path: '/project/.bc/worktrees/old-agent', status: 'ORPHANED' },
    { agent: 'eng-03', path: '/project/.bc/worktrees/eng-03', status: 'OK' },
    { agent: 'deleted-agent', path: '/project/.bc/worktrees/deleted', status: 'ORPHANED' },
  ];

  test('filters active worktrees', () => {
    const active = mockWorktrees.filter((wt) => wt.status === 'OK');
    expect(active).toHaveLength(3);
    expect(active.map((w) => w.agent)).toEqual(['eng-01', 'eng-02', 'eng-03']);
  });

  test('filters orphaned worktrees', () => {
    const orphaned = mockWorktrees.filter((wt) => wt.status !== 'OK');
    expect(orphaned).toHaveLength(2);
    expect(orphaned.map((w) => w.agent)).toEqual(['old-agent', 'deleted-agent']);
  });

  test('filters to show orphaned only', () => {
    const showOrphanedOnly = true;
    const filtered = showOrphanedOnly
      ? mockWorktrees.filter((wt) => wt.status === 'ORPHANED')
      : mockWorktrees;
    expect(filtered).toHaveLength(2);
  });

  test('shows all when filter is off', () => {
    const showOrphanedOnly = false;
    const filtered = showOrphanedOnly
      ? mockWorktrees.filter((wt) => wt.status === 'ORPHANED')
      : mockWorktrees;
    expect(filtered).toHaveLength(5);
  });

  test('hasOrphans check', () => {
    const orphaned = mockWorktrees.filter((wt) => wt.status !== 'OK');
    const hasOrphans = orphaned.length > 0;
    expect(hasOrphans).toBe(true);
  });

  test('hasOrphans is false when no orphans', () => {
    const activeOnly = mockWorktrees.filter((wt) => wt.status === 'OK');
    const orphaned = activeOnly.filter((wt) => wt.status !== 'OK');
    const hasOrphans = orphaned.length > 0;
    expect(hasOrphans).toBe(false);
  });
});

describe('WorktreesView - column width calculations', () => {
  const agentWidth = 15;
  const statusWidth = 10;

  test('calculates path width for standard terminal', () => {
    const terminalWidth = 80;
    const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);
    expect(pathWidth).toBe(45);
  });

  test('calculates path width for wide terminal', () => {
    const terminalWidth = 120;
    const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);
    expect(pathWidth).toBe(50); // Capped at 50
  });

  test('calculates path width for narrow terminal', () => {
    const terminalWidth = 60;
    const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);
    expect(pathWidth).toBe(25);
  });

  test('handles very narrow terminal', () => {
    const terminalWidth = 40;
    const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);
    expect(pathWidth).toBe(5);
  });
});

describe('WorktreesView - agent name truncation', () => {
  const agentWidth = 15;

  function truncateAgent(agent: string): string {
    return agent.slice(0, agentWidth - 3).padEnd(agentWidth - 2);
  }

  test('short names are padded', () => {
    const result = truncateAgent('eng-01');
    expect(result).toBe('eng-01       '); // 6 chars + 7 spaces = 13
  });

  test('long names are truncated', () => {
    const result = truncateAgent('very-long-agent-name');
    expect(result).toBe('very-long-ag '); // 12 chars + 1 space = 13
  });

  test('exact length names', () => {
    const result = truncateAgent('exactly12chr');
    expect(result).toBe('exactly12chr '); // 12 chars + 1 space = 13
  });
});

describe('WorktreesView - subtitle generation', () => {
  test('generates subtitle when orphans exist', () => {
    const orphanedCount = 3;
    const subtitle = orphanedCount > 0 ? `${String(orphanedCount)} orphaned` : undefined;
    expect(subtitle).toBe('3 orphaned');
  });

  test('returns undefined when no orphans', () => {
    const orphanedCount = 0;
    const subtitle = orphanedCount > 0 ? `${String(orphanedCount)} orphaned` : undefined;
    expect(subtitle).toBeUndefined();
  });

  test('handles single orphan', () => {
    const orphanedCount = 1;
    const subtitle = orphanedCount > 0 ? `${String(orphanedCount)} orphaned` : undefined;
    expect(subtitle).toBe('1 orphaned');
  });
});

describe('WorktreesView - prune confirmation', () => {
  test('shows first 5 worktrees in confirmation', () => {
    const orphaned = [
      { agent: 'agent-1' },
      { agent: 'agent-2' },
      { agent: 'agent-3' },
      { agent: 'agent-4' },
      { agent: 'agent-5' },
      { agent: 'agent-6' },
      { agent: 'agent-7' },
    ];

    const displayed = orphaned.slice(0, 5);
    const remaining = orphaned.length - 5;

    expect(displayed).toHaveLength(5);
    expect(remaining).toBe(2);
  });

  test('shows all when 5 or fewer', () => {
    const orphaned = [{ agent: 'agent-1' }, { agent: 'agent-2' }, { agent: 'agent-3' }];

    const displayed = orphaned.slice(0, 5);
    const remaining = orphaned.length - 5;

    expect(displayed).toHaveLength(3);
    expect(remaining).toBe(-2); // No "and X more" shown
  });
});

describe('WorktreesView - separator display', () => {
  test('separator shown when both active and orphaned exist', () => {
    const activeCount = 3;
    const orphanedCount = 2;
    const showOrphanedOnly = false;

    const showSeparator = activeCount > 0 && orphanedCount > 0 && !showOrphanedOnly;
    expect(showSeparator).toBe(true);
  });

  test('separator hidden when only active', () => {
    const activeCount = 3;
    const orphanedCount = 0;
    const showOrphanedOnly = false;

    const showSeparator = activeCount > 0 && orphanedCount > 0 && !showOrphanedOnly;
    expect(showSeparator).toBe(false);
  });

  test('separator hidden when only orphaned', () => {
    const activeCount = 0;
    const orphanedCount = 2;
    const showOrphanedOnly = false;

    const showSeparator = activeCount > 0 && orphanedCount > 0 && !showOrphanedOnly;
    expect(showSeparator).toBe(false);
  });

  test('separator hidden in orphaned-only mode', () => {
    const activeCount = 3;
    const orphanedCount = 2;
    const showOrphanedOnly = true;

    const showSeparator = activeCount > 0 && orphanedCount > 0 && !showOrphanedOnly;
    expect(showSeparator).toBe(false);
  });
});
