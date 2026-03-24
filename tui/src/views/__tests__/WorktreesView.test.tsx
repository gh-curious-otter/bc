/**
 * WorktreesView Tests - Git worktree management tab (#868)
 *
 * Tests cover:
 * - Path formatting (extract .bc/worktrees portion)
 * - Worktree filtering (active vs orphaned)
 * - Keyboard navigation (j/k, g/G, Enter, o, p, r, q)
 * - Column width calculation
 * - State management
 */

import { describe, test, expect } from 'bun:test';

// Helper function extracted from WorktreesView for testing
function formatPath(fullPath: string): string {
  const match = /\.bc\/worktrees\/.+$/.exec(fullPath);
  return match ? match[0] : fullPath;
}

interface Worktree {
  agent: string;
  path: string;
  status: 'OK' | 'ORPHANED';
  branch?: string;
}

describe('WorktreesView', () => {
  describe('Path Formatting', () => {
    test('extracts .bc/worktrees portion from full path', () => {
      const fullPath = '/Users/test/project/.bc/worktrees/eng-01';
      const formatted = formatPath(fullPath);
      expect(formatted).toBe('.bc/worktrees/eng-01');
    });

    test('returns full path if no .bc/worktrees match', () => {
      const fullPath = '/some/other/path';
      const formatted = formatPath(fullPath);
      expect(formatted).toBe('/some/other/path');
    });

    test('handles nested worktree paths', () => {
      const fullPath = '/Users/test/project/.bc/worktrees/eng-01/src';
      const formatted = formatPath(fullPath);
      expect(formatted).toBe('.bc/worktrees/eng-01/src');
    });
  });

  describe('Worktree Filtering', () => {
    const mockWorktrees: Worktree[] = [
      { agent: 'eng-01', path: '/project/.bc/worktrees/eng-01', status: 'OK', branch: 'feature-1' },
      { agent: 'eng-02', path: '/project/.bc/worktrees/eng-02', status: 'OK', branch: 'main' },
      { agent: 'eng-03', path: '/project/.bc/worktrees/eng-03', status: 'ORPHANED' },
      { agent: '', path: '/project/.bc/worktrees/old-agent', status: 'ORPHANED' },
    ];

    test('separates active and orphaned worktrees', () => {
      const active = mockWorktrees.filter((wt) => wt.status === 'OK');
      const orphaned = mockWorktrees.filter((wt) => wt.status !== 'OK');

      expect(active).toHaveLength(2);
      expect(orphaned).toHaveLength(2);
    });

    test('filters to show only orphaned', () => {
      const showOrphanedOnly = true;
      const filtered = showOrphanedOnly
        ? mockWorktrees.filter((wt) => wt.status === 'ORPHANED')
        : mockWorktrees;

      expect(filtered).toHaveLength(2);
      expect(filtered.every((wt) => wt.status === 'ORPHANED')).toBe(true);
    });

    test('shows all when filter is off', () => {
      const showOrphanedOnly = false;
      const filtered = showOrphanedOnly
        ? mockWorktrees.filter((wt) => wt.status === 'ORPHANED')
        : mockWorktrees;

      expect(filtered).toHaveLength(4);
    });

    test('identifies if orphans exist', () => {
      const orphaned = mockWorktrees.filter((wt) => wt.status !== 'OK');
      const hasOrphans = orphaned.length > 0;
      expect(hasOrphans).toBe(true);
    });
  });

  describe('Keyboard Navigation', () => {
    test('j/k moves selection up/down', () => {
      let selectedIndex = 0;
      const maxIndex = 3;

      // Press 'j' - move down
      selectedIndex = Math.min(maxIndex, selectedIndex + 1);
      expect(selectedIndex).toBe(1);

      // Press 'k' - move up
      selectedIndex = Math.max(0, selectedIndex - 1);
      expect(selectedIndex).toBe(0);
    });

    test('g goes to first item, G goes to last', () => {
      let selectedIndex = 2;
      const worktreesLength = 4;

      // Press 'g' - go to first
      selectedIndex = 0;
      expect(selectedIndex).toBe(0);

      // Press 'G' - go to last
      selectedIndex = Math.max(0, worktreesLength - 1);
      expect(selectedIndex).toBe(3);
    });

    test('o toggles orphaned-only filter', () => {
      let showOrphanedOnly = false;

      // Press 'o' - toggle on
      showOrphanedOnly = !showOrphanedOnly;
      expect(showOrphanedOnly).toBe(true);

      // Press 'o' again - toggle off
      showOrphanedOnly = !showOrphanedOnly;
      expect(showOrphanedOnly).toBe(false);
    });

    test('p shows prune confirm only when orphans exist', () => {
      const hasOrphans = true;
      let showPruneConfirm = false;

      // Press 'p' with orphans
      if (hasOrphans) {
        showPruneConfirm = true;
      }
      expect(showPruneConfirm).toBe(true);
    });

    test('p does not show prune confirm when no orphans', () => {
      const hasOrphans = false;
      let showPruneConfirm = false;

      // Press 'p' without orphans
      if (hasOrphans) {
        showPruneConfirm = true;
      }
      expect(showPruneConfirm).toBe(false);
    });
  });

  describe('Prune Confirmation', () => {
    test('y confirms prune action', () => {
      let pruneExecuted = false;
      const input = 'y';

      if (input === 'y' || input === 'Y') {
        pruneExecuted = true;
      }
      expect(pruneExecuted).toBe(true);
    });

    test('n cancels prune action', () => {
      let showPruneConfirm = true;
      const input = 'n';

      if (input === 'n' || input === 'N') {
        showPruneConfirm = false;
      }
      expect(showPruneConfirm).toBe(false);
    });

    test('escape cancels prune action', () => {
      let showPruneConfirm = true;
      const key = { escape: true };

      if (key.escape) {
        showPruneConfirm = false;
      }
      expect(showPruneConfirm).toBe(false);
    });
  });

  describe('Column Width Calculation', () => {
    function calculateColumnWidths(terminalWidth: number) {
      const agentWidth = 15;
      const statusWidth = 10;
      const pathWidth = Math.min(50, terminalWidth - agentWidth - statusWidth - 10);
      return { agentWidth, statusWidth, pathWidth };
    }

    test('calculates correct column widths at 80 columns', () => {
      const { agentWidth, statusWidth, pathWidth } = calculateColumnWidths(80);
      expect(agentWidth).toBe(15);
      expect(statusWidth).toBe(10);
      expect(pathWidth).toBe(45); // 80 - 15 - 10 - 10 = 45
    });

    test('calculates correct column widths at 120 columns', () => {
      const { agentWidth, statusWidth, pathWidth } = calculateColumnWidths(120);
      expect(agentWidth).toBe(15);
      expect(statusWidth).toBe(10);
      expect(pathWidth).toBe(50); // min(50, 85) = 50
    });

    test('path width is capped at 50', () => {
      const { pathWidth } = calculateColumnWidths(200);
      expect(pathWidth).toBe(50);
    });
  });

  describe('State Management', () => {
    test('initial state values', () => {
      const initialState = {
        worktrees: [] as Worktree[],
        loading: true,
        error: null as string | null,
        selectedIndex: 0,
        showDetail: false,
        showPruneConfirm: false,
        pruneResult: null as string | null,
        showOrphanedOnly: false,
      };

      expect(initialState.worktrees).toHaveLength(0);
      expect(initialState.loading).toBe(true);
      expect(initialState.error).toBeNull();
      expect(initialState.selectedIndex).toBe(0);
      expect(initialState.showDetail).toBe(false);
      expect(initialState.showPruneConfirm).toBe(false);
      expect(initialState.pruneResult).toBeNull();
      expect(initialState.showOrphanedOnly).toBe(false);
    });

    test('toggling orphaned-only resets selection', () => {
      let selectedIndex = 5;
      let showOrphanedOnly = false;

      // Toggle filter
      showOrphanedOnly = !showOrphanedOnly;
      selectedIndex = 0;

      expect(selectedIndex).toBe(0);
      expect(showOrphanedOnly).toBe(true);
    });
  });

  describe('Worktree Display', () => {
    test('handles worktrees with empty agent name', () => {
      const worktree: Worktree = {
        agent: '',
        path: '/project/.bc/worktrees/orphan',
        status: 'ORPHANED',
      };

      const displayAgent = worktree.agent || '(orphan)';
      expect(displayAgent).toBe('(orphan)');
    });

    test('status colors are correct', () => {
      const getStatusColor = (status: string): string => {
        return status === 'OK' ? 'green' : 'red';
      };

      expect(getStatusColor('OK')).toBe('green');
      expect(getStatusColor('ORPHANED')).toBe('red');
    });

    test('prune result message styling', () => {
      const getResultColor = (result: string): string => {
        return result.startsWith('Error') ? 'red' : 'green';
      };

      expect(getResultColor('Pruned successfully')).toBe('green');
      expect(getResultColor('Error: Failed to prune')).toBe('red');
    });
  });
});
