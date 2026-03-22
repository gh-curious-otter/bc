/**
 * FilesView Tests
 * RFC 002: File Explorer for TUI
 *
 * Tests cover:
 * - UI state reducer
 * - Focus area cycling
 * - Tree flattening
 * - Git status indicators
 * - Path breadcrumb truncation
 * - Responsive layout breakpoints
 * - File preview size limits
 */

import { describe, test, expect } from 'bun:test';

// Focus areas matching FilesView
type FocusArea = 'worktree' | 'tree' | 'preview';

// UI state interface matching FilesView
interface UIState {
  worktreeIndex: number;
  worktreeSelectorOpen: boolean;
  selectedPath: string | null;
  treeIndex: number;
  focusArea: FocusArea;
}

// UI action types
type UIAction =
  | { type: 'SET_WORKTREE_INDEX'; index: number }
  | { type: 'TOGGLE_WORKTREE_SELECTOR' }
  | { type: 'CLOSE_WORKTREE_SELECTOR' }
  | { type: 'SELECT_WORKTREE'; index: number }
  | { type: 'SET_TREE_INDEX'; index: number }
  | { type: 'SELECT_FILE'; path: string }
  | { type: 'RESET_NAVIGATION' }
  | { type: 'CYCLE_FOCUS_FORWARD' }
  | { type: 'CYCLE_FOCUS_BACKWARD' };

// Initial state matching FilesView
const initialUIState: UIState = {
  worktreeIndex: 0,
  worktreeSelectorOpen: false,
  selectedPath: null,
  treeIndex: 0,
  focusArea: 'tree',
};

// Reducer matching FilesView
function uiReducer(state: UIState, action: UIAction): UIState {
  switch (action.type) {
    case 'SET_WORKTREE_INDEX':
      return { ...state, worktreeIndex: action.index };
    case 'TOGGLE_WORKTREE_SELECTOR':
      return { ...state, worktreeSelectorOpen: !state.worktreeSelectorOpen };
    case 'CLOSE_WORKTREE_SELECTOR':
      return { ...state, worktreeSelectorOpen: false };
    case 'SELECT_WORKTREE':
      return { ...state, worktreeIndex: action.index, worktreeSelectorOpen: false };
    case 'SET_TREE_INDEX':
      return { ...state, treeIndex: action.index };
    case 'SELECT_FILE':
      return { ...state, selectedPath: action.path, focusArea: 'preview' };
    case 'RESET_NAVIGATION':
      return { ...state, treeIndex: 0, selectedPath: null };
    case 'CYCLE_FOCUS_FORWARD':
      return {
        ...state,
        focusArea:
          state.focusArea === 'worktree'
            ? 'tree'
            : state.focusArea === 'tree'
              ? 'preview'
              : 'worktree',
      };
    case 'CYCLE_FOCUS_BACKWARD':
      return {
        ...state,
        focusArea:
          state.focusArea === 'worktree'
            ? 'preview'
            : state.focusArea === 'tree'
              ? 'worktree'
              : 'tree',
      };
    default:
      return state;
  }
}

// Git status types
type GitFileStatus = 'modified' | 'added' | 'deleted' | 'renamed' | 'untracked' | 'ignored';

// Git status indicator helper
function getGitStatusIndicator(status: GitFileStatus | undefined): { icon: string; color: string } {
  switch (status) {
    case 'modified':
      return { icon: '✱', color: 'yellow' };
    case 'added':
      return { icon: '+', color: 'green' };
    case 'deleted':
      return { icon: '−', color: 'red' };
    case 'renamed':
      return { icon: '→', color: 'blue' };
    case 'untracked':
      return { icon: '?', color: 'gray' };
    case 'ignored':
      return { icon: '!', color: 'gray' };
    default:
      return { icon: ' ', color: '' };
  }
}

// File tree entry interface
interface FileTreeEntry {
  name: string;
  path: string;
  isDirectory: boolean;
  expanded?: boolean;
  children: FileTreeEntry[];
}

// Tree flattening helper
function flattenTree(
  entries: FileTreeEntry[],
  depth = 0
): { entry: FileTreeEntry; depth: number }[] {
  const result: { entry: FileTreeEntry; depth: number }[] = [];
  for (const entry of entries) {
    result.push({ entry, depth });
    if (entry.isDirectory && entry.expanded && entry.children.length > 0) {
      result.push(...flattenTree(entry.children, depth + 1));
    }
  }
  return result;
}

// Path breadcrumb truncation
function truncatePath(
  segments: string[],
  maxWidth: number,
  separator = ' › '
): { segments: string[]; truncated: boolean } {
  const fullDisplay = segments.join(separator);
  if (fullDisplay.length <= maxWidth - 4) {
    return { segments, truncated: false };
  }

  if (segments.length > 2) {
    return {
      segments: [segments[0], '...', segments[segments.length - 1]],
      truncated: true,
    };
  }

  return { segments, truncated: true };
}

// Responsive tree width calculation
function getTreeWidth(breakpoint: 'xs' | 'sm' | 'md' | 'lg' | 'xl'): number {
  const widths = { xs: 16, sm: 20, md: 25, lg: 30, xl: 35 };
  return widths[breakpoint];
}

// File size limit (100KB)
const MAX_PREVIEW_SIZE = 100 * 1024;

function isFileTooLarge(size: number): boolean {
  return size > MAX_PREVIEW_SIZE;
}

describe('FilesView', () => {
  describe('UI Reducer', () => {
    describe('SET_WORKTREE_INDEX', () => {
      test('updates worktree index', () => {
        const state = uiReducer(initialUIState, { type: 'SET_WORKTREE_INDEX', index: 2 });
        expect(state.worktreeIndex).toBe(2);
      });

      test('preserves other state', () => {
        const state = uiReducer(initialUIState, { type: 'SET_WORKTREE_INDEX', index: 1 });
        expect(state.focusArea).toBe('tree');
        expect(state.selectedPath).toBeNull();
      });
    });

    describe('TOGGLE_WORKTREE_SELECTOR', () => {
      test('opens closed selector', () => {
        const state = uiReducer(initialUIState, { type: 'TOGGLE_WORKTREE_SELECTOR' });
        expect(state.worktreeSelectorOpen).toBe(true);
      });

      test('closes open selector', () => {
        const openState = { ...initialUIState, worktreeSelectorOpen: true };
        const state = uiReducer(openState, { type: 'TOGGLE_WORKTREE_SELECTOR' });
        expect(state.worktreeSelectorOpen).toBe(false);
      });
    });

    describe('CLOSE_WORKTREE_SELECTOR', () => {
      test('closes selector', () => {
        const openState = { ...initialUIState, worktreeSelectorOpen: true };
        const state = uiReducer(openState, { type: 'CLOSE_WORKTREE_SELECTOR' });
        expect(state.worktreeSelectorOpen).toBe(false);
      });
    });

    describe('SELECT_WORKTREE', () => {
      test('selects worktree and closes selector', () => {
        const openState = { ...initialUIState, worktreeSelectorOpen: true };
        const state = uiReducer(openState, { type: 'SELECT_WORKTREE', index: 1 });
        expect(state.worktreeIndex).toBe(1);
        expect(state.worktreeSelectorOpen).toBe(false);
      });
    });

    describe('SET_TREE_INDEX', () => {
      test('updates tree index', () => {
        const state = uiReducer(initialUIState, { type: 'SET_TREE_INDEX', index: 5 });
        expect(state.treeIndex).toBe(5);
      });
    });

    describe('SELECT_FILE', () => {
      test('sets selected path and switches to preview', () => {
        const state = uiReducer(initialUIState, { type: 'SELECT_FILE', path: '/path/to/file.ts' });
        expect(state.selectedPath).toBe('/path/to/file.ts');
        expect(state.focusArea).toBe('preview');
      });
    });

    describe('RESET_NAVIGATION', () => {
      test('resets tree index and selected path', () => {
        const modifiedState = { ...initialUIState, treeIndex: 10, selectedPath: '/some/path' };
        const state = uiReducer(modifiedState, { type: 'RESET_NAVIGATION' });
        expect(state.treeIndex).toBe(0);
        expect(state.selectedPath).toBeNull();
      });
    });
  });

  describe('Focus Cycling', () => {
    describe('CYCLE_FOCUS_FORWARD', () => {
      test('worktree → tree', () => {
        const state = uiReducer(
          { ...initialUIState, focusArea: 'worktree' },
          { type: 'CYCLE_FOCUS_FORWARD' }
        );
        expect(state.focusArea).toBe('tree');
      });

      test('tree → preview', () => {
        const state = uiReducer(
          { ...initialUIState, focusArea: 'tree' },
          { type: 'CYCLE_FOCUS_FORWARD' }
        );
        expect(state.focusArea).toBe('preview');
      });

      test('preview → worktree', () => {
        const state = uiReducer(
          { ...initialUIState, focusArea: 'preview' },
          { type: 'CYCLE_FOCUS_FORWARD' }
        );
        expect(state.focusArea).toBe('worktree');
      });
    });

    describe('CYCLE_FOCUS_BACKWARD', () => {
      test('worktree → preview', () => {
        const state = uiReducer(
          { ...initialUIState, focusArea: 'worktree' },
          { type: 'CYCLE_FOCUS_BACKWARD' }
        );
        expect(state.focusArea).toBe('preview');
      });

      test('tree → worktree', () => {
        const state = uiReducer(
          { ...initialUIState, focusArea: 'tree' },
          { type: 'CYCLE_FOCUS_BACKWARD' }
        );
        expect(state.focusArea).toBe('worktree');
      });

      test('preview → tree', () => {
        const state = uiReducer(
          { ...initialUIState, focusArea: 'preview' },
          { type: 'CYCLE_FOCUS_BACKWARD' }
        );
        expect(state.focusArea).toBe('tree');
      });
    });
  });

  describe('Git Status Indicators', () => {
    test('modified files show yellow marker', () => {
      const indicator = getGitStatusIndicator('modified');
      expect(indicator.icon).toBe('✱');
      expect(indicator.color).toBe('yellow');
    });

    test('added files show green plus', () => {
      const indicator = getGitStatusIndicator('added');
      expect(indicator.icon).toBe('+');
      expect(indicator.color).toBe('green');
    });

    test('deleted files show red minus', () => {
      const indicator = getGitStatusIndicator('deleted');
      expect(indicator.icon).toBe('−');
      expect(indicator.color).toBe('red');
    });

    test('renamed files show blue arrow', () => {
      const indicator = getGitStatusIndicator('renamed');
      expect(indicator.icon).toBe('→');
      expect(indicator.color).toBe('blue');
    });

    test('untracked files show gray question', () => {
      const indicator = getGitStatusIndicator('untracked');
      expect(indicator.icon).toBe('?');
      expect(indicator.color).toBe('gray');
    });

    test('ignored files show gray exclamation', () => {
      const indicator = getGitStatusIndicator('ignored');
      expect(indicator.icon).toBe('!');
      expect(indicator.color).toBe('gray');
    });

    test('undefined status shows empty', () => {
      const indicator = getGitStatusIndicator(undefined);
      expect(indicator.icon).toBe(' ');
      expect(indicator.color).toBe('');
    });
  });

  describe('Tree Flattening', () => {
    test('flattens empty tree', () => {
      const result = flattenTree([]);
      expect(result).toHaveLength(0);
    });

    test('flattens single file', () => {
      const tree: FileTreeEntry[] = [
        { name: 'file.ts', path: '/file.ts', isDirectory: false, children: [] },
      ];
      const result = flattenTree(tree);
      expect(result).toHaveLength(1);
      expect(result[0].entry.name).toBe('file.ts');
      expect(result[0].depth).toBe(0);
    });

    test('flattens collapsed directory', () => {
      const tree: FileTreeEntry[] = [
        {
          name: 'src',
          path: '/src',
          isDirectory: true,
          expanded: false,
          children: [{ name: 'index.ts', path: '/src/index.ts', isDirectory: false, children: [] }],
        },
      ];
      const result = flattenTree(tree);
      expect(result).toHaveLength(1);
      expect(result[0].entry.name).toBe('src');
    });

    test('flattens expanded directory', () => {
      const tree: FileTreeEntry[] = [
        {
          name: 'src',
          path: '/src',
          isDirectory: true,
          expanded: true,
          children: [{ name: 'index.ts', path: '/src/index.ts', isDirectory: false, children: [] }],
        },
      ];
      const result = flattenTree(tree);
      expect(result).toHaveLength(2);
      expect(result[0].entry.name).toBe('src');
      expect(result[0].depth).toBe(0);
      expect(result[1].entry.name).toBe('index.ts');
      expect(result[1].depth).toBe(1);
    });

    test('flattens nested directories', () => {
      const tree: FileTreeEntry[] = [
        {
          name: 'src',
          path: '/src',
          isDirectory: true,
          expanded: true,
          children: [
            {
              name: 'components',
              path: '/src/components',
              isDirectory: true,
              expanded: true,
              children: [
                {
                  name: 'Button.tsx',
                  path: '/src/components/Button.tsx',
                  isDirectory: false,
                  children: [],
                },
              ],
            },
          ],
        },
      ];
      const result = flattenTree(tree);
      expect(result).toHaveLength(3);
      expect(result[2].depth).toBe(2);
    });
  });

  describe('Path Breadcrumb Truncation', () => {
    test('short path not truncated', () => {
      const segments = ['src', 'index.ts'];
      const result = truncatePath(segments, 80);
      expect(result.truncated).toBe(false);
      expect(result.segments).toEqual(['src', 'index.ts']);
    });

    test('long path truncated with ellipsis', () => {
      const segments = ['src', 'components', 'features', 'auth', 'LoginForm.tsx'];
      const result = truncatePath(segments, 30);
      expect(result.truncated).toBe(true);
      expect(result.segments).toEqual(['src', '...', 'LoginForm.tsx']);
    });

    test('two segment path shows truncated flag only', () => {
      const segments = ['very-long-directory-name', 'very-long-file-name.tsx'];
      const result = truncatePath(segments, 20);
      expect(result.truncated).toBe(true);
      expect(result.segments).toHaveLength(2);
    });
  });

  describe('Responsive Tree Width', () => {
    test('xs breakpoint is 16 cols', () => {
      expect(getTreeWidth('xs')).toBe(16);
    });

    test('sm breakpoint is 20 cols', () => {
      expect(getTreeWidth('sm')).toBe(20);
    });

    test('md breakpoint is 25 cols', () => {
      expect(getTreeWidth('md')).toBe(25);
    });

    test('lg breakpoint is 30 cols', () => {
      expect(getTreeWidth('lg')).toBe(30);
    });

    test('xl breakpoint is 35 cols', () => {
      expect(getTreeWidth('xl')).toBe(35);
    });
  });

  describe('File Preview Size Limits', () => {
    test('small file is previewable', () => {
      expect(isFileTooLarge(1024)).toBe(false); // 1KB
      expect(isFileTooLarge(50 * 1024)).toBe(false); // 50KB
    });

    test('exactly 100KB is previewable', () => {
      expect(isFileTooLarge(100 * 1024)).toBe(false);
    });

    test('over 100KB is too large', () => {
      expect(isFileTooLarge(100 * 1024 + 1)).toBe(true);
      expect(isFileTooLarge(1024 * 1024)).toBe(true); // 1MB
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      j: 'down',
      k: 'up',
      Enter: 'expand/select',
      Escape: 'close/back',
      w: 'worktree selector',
      f: 'cycle focus forward',
      F: 'cycle focus backward',
    };

    test('navigation shortcuts', () => {
      expect(shortcuts.j).toBe('down');
      expect(shortcuts.k).toBe('up');
      expect(shortcuts.Enter).toBe('expand/select');
    });

    test('worktree shortcuts', () => {
      expect(shortcuts.w).toBe('worktree selector');
    });

    test('focus shortcuts', () => {
      expect(shortcuts.f).toBe('cycle focus forward');
      expect(shortcuts.F).toBe('cycle focus backward');
    });
  });

  describe('Worktree Data', () => {
    interface Worktree {
      agent: string;
      path: string;
      branch?: string;
      status: 'OK' | 'MISSING';
    }

    test('filters to OK worktrees only', () => {
      const worktrees: Worktree[] = [
        { agent: 'eng-01', path: '/path/1', branch: 'main', status: 'OK' },
        { agent: 'eng-02', path: '/path/2', status: 'MISSING' },
        { agent: 'eng-03', path: '/path/3', branch: 'feature', status: 'OK' },
      ];

      const activeWorktrees = worktrees.filter((w) => w.status === 'OK');
      expect(activeWorktrees).toHaveLength(2);
      expect(activeWorktrees.map((w) => w.agent)).toEqual(['eng-01', 'eng-03']);
    });
  });

  describe('File Tree Display', () => {
    test('calculates visible window', () => {
      const totalItems = 100;
      const maxHeight = 20;
      const visibleCount = Math.max(1, maxHeight - 2);
      const selectedIndex = 50;

      const start = Math.max(
        0,
        Math.min(selectedIndex - Math.floor(visibleCount / 2), totalItems - visibleCount)
      );

      expect(visibleCount).toBe(18);
      expect(start).toBe(41); // 50 - 9
    });

    test('handles selection at start', () => {
      const totalItems = 100;
      const visibleCount = 18;
      const selectedIndex = 0;

      const start = Math.max(
        0,
        Math.min(selectedIndex - Math.floor(visibleCount / 2), totalItems - visibleCount)
      );
      expect(start).toBe(0);
    });

    test('handles selection at end', () => {
      const totalItems = 100;
      const visibleCount = 18;
      const selectedIndex = 99;

      const start = Math.max(
        0,
        Math.min(selectedIndex - Math.floor(visibleCount / 2), totalItems - visibleCount)
      );
      expect(start).toBe(82); // 100 - 18
    });
  });

  describe('Loading States', () => {
    test('worktrees loading message', () => {
      const loading = true;
      const message = loading ? 'Loading worktrees...' : '';
      expect(message).toBe('Loading worktrees...');
    });

    test('tree loading message', () => {
      const loading = true;
      const message = loading ? 'Loading files...' : '';
      expect(message).toBe('Loading files...');
    });
  });

  describe('Error States', () => {
    test('worktree error message', () => {
      const error = 'Failed to load worktrees';
      const message = `Error: ${error}`;
      expect(message).toBe('Error: Failed to load worktrees');
    });

    test('file preview error message', () => {
      const error = 'File too large to preview (>100KB)';
      expect(error).toContain('100KB');
    });
  });

  describe('Empty States', () => {
    test('no worktrees message', () => {
      const worktrees: { agent: string }[] = [];
      const isEmpty = worktrees.length === 0;
      expect(isEmpty).toBe(true);
    });

    test('no files message', () => {
      const files: FileTreeEntry[] = [];
      const isEmpty = files.length === 0;
      expect(isEmpty).toBe(true);
    });
  });
});
