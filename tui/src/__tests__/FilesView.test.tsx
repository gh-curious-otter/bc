/**
 * FilesView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('FilesView - getGitStatusIndicator', () => {
  type GitFileStatus = 'modified' | 'added' | 'deleted' | 'renamed' | 'untracked' | 'ignored';

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

  test('modified is yellow star', () => {
    const result = getGitStatusIndicator('modified');
    expect(result.icon).toBe('✱');
    expect(result.color).toBe('yellow');
  });

  test('added is green plus', () => {
    const result = getGitStatusIndicator('added');
    expect(result.icon).toBe('+');
    expect(result.color).toBe('green');
  });

  test('deleted is red minus', () => {
    const result = getGitStatusIndicator('deleted');
    expect(result.icon).toBe('−');
    expect(result.color).toBe('red');
  });

  test('renamed is blue arrow', () => {
    const result = getGitStatusIndicator('renamed');
    expect(result.icon).toBe('→');
    expect(result.color).toBe('blue');
  });

  test('untracked is gray question mark', () => {
    const result = getGitStatusIndicator('untracked');
    expect(result.icon).toBe('?');
    expect(result.color).toBe('gray');
  });

  test('ignored is gray exclamation', () => {
    const result = getGitStatusIndicator('ignored');
    expect(result.icon).toBe('!');
    expect(result.color).toBe('gray');
  });

  test('undefined is empty', () => {
    const result = getGitStatusIndicator(undefined);
    expect(result.icon).toBe(' ');
    expect(result.color).toBe('');
  });
});

describe('FilesView - flattenTree', () => {
  interface FileTreeEntry {
    name: string;
    path: string;
    isDirectory: boolean;
    expanded: boolean;
    children: FileTreeEntry[];
  }

  function flattenTree(entries: FileTreeEntry[], depth = 0): { entry: FileTreeEntry; depth: number }[] {
    const result: { entry: FileTreeEntry; depth: number }[] = [];
    for (const entry of entries) {
      result.push({ entry, depth });
      if (entry.isDirectory && entry.expanded && entry.children.length > 0) {
        result.push(...flattenTree(entry.children, depth + 1));
      }
    }
    return result;
  }

  test('flattens empty tree', () => {
    expect(flattenTree([])).toEqual([]);
  });

  test('flattens single file', () => {
    const entries: FileTreeEntry[] = [{
      name: 'file.txt',
      path: '/file.txt',
      isDirectory: false,
      expanded: false,
      children: [],
    }];
    const result = flattenTree(entries);
    expect(result).toHaveLength(1);
    expect(result[0].depth).toBe(0);
  });

  test('collapsed directory does not include children', () => {
    const entries: FileTreeEntry[] = [{
      name: 'src',
      path: '/src',
      isDirectory: true,
      expanded: false,
      children: [{
        name: 'index.ts',
        path: '/src/index.ts',
        isDirectory: false,
        expanded: false,
        children: [],
      }],
    }];
    const result = flattenTree(entries);
    expect(result).toHaveLength(1);
    expect(result[0].entry.name).toBe('src');
  });

  test('expanded directory includes children', () => {
    const entries: FileTreeEntry[] = [{
      name: 'src',
      path: '/src',
      isDirectory: true,
      expanded: true,
      children: [{
        name: 'index.ts',
        path: '/src/index.ts',
        isDirectory: false,
        expanded: false,
        children: [],
      }],
    }];
    const result = flattenTree(entries);
    expect(result).toHaveLength(2);
    expect(result[0].entry.name).toBe('src');
    expect(result[0].depth).toBe(0);
    expect(result[1].entry.name).toBe('index.ts');
    expect(result[1].depth).toBe(1);
  });

  test('nested expanded directories', () => {
    const entries: FileTreeEntry[] = [{
      name: 'src',
      path: '/src',
      isDirectory: true,
      expanded: true,
      children: [{
        name: 'components',
        path: '/src/components',
        isDirectory: true,
        expanded: true,
        children: [{
          name: 'Button.tsx',
          path: '/src/components/Button.tsx',
          isDirectory: false,
          expanded: false,
          children: [],
        }],
      }],
    }];
    const result = flattenTree(entries);
    expect(result).toHaveLength(3);
    expect(result[2].depth).toBe(2);
  });
});

describe('FilesView - uiReducer', () => {
  type FocusArea = 'worktree' | 'tree' | 'preview';

  interface UIState {
    worktreeIndex: number;
    worktreeSelectorOpen: boolean;
    selectedPath: string | null;
    treeIndex: number;
    focusArea: FocusArea;
  }

  const initialState: UIState = {
    worktreeIndex: 0,
    worktreeSelectorOpen: false,
    selectedPath: null,
    treeIndex: 0,
    focusArea: 'tree',
  };

  function cycleFocusForward(current: FocusArea): FocusArea {
    if (current === 'worktree') return 'tree';
    if (current === 'tree') return 'preview';
    return 'worktree';
  }

  function cycleFocusBackward(current: FocusArea): FocusArea {
    if (current === 'worktree') return 'preview';
    if (current === 'tree') return 'worktree';
    return 'tree';
  }

  test('cycle focus forward from tree', () => {
    expect(cycleFocusForward('tree')).toBe('preview');
  });

  test('cycle focus forward from preview', () => {
    expect(cycleFocusForward('preview')).toBe('worktree');
  });

  test('cycle focus forward from worktree', () => {
    expect(cycleFocusForward('worktree')).toBe('tree');
  });

  test('cycle focus backward from tree', () => {
    expect(cycleFocusBackward('tree')).toBe('worktree');
  });

  test('cycle focus backward from preview', () => {
    expect(cycleFocusBackward('preview')).toBe('tree');
  });

  test('cycle focus backward from worktree', () => {
    expect(cycleFocusBackward('worktree')).toBe('preview');
  });
});

describe('FilesView - path breadcrumb', () => {
  function getPathSegments(path: string): string[] {
    return path.split('/').filter(Boolean);
  }

  function shouldTruncatePath(path: string, maxWidth: number): boolean {
    const separator = ' › ';
    const segments = getPathSegments(path);
    const fullDisplay = segments.join(separator);
    return fullDisplay.length > maxWidth - 4;
  }

  function getTruncatedSegments(segments: string[]): string[] {
    if (segments.length <= 2) return segments;
    return [segments[0], '...', segments[segments.length - 1]];
  }

  test('extracts path segments', () => {
    expect(getPathSegments('src/components/Button.tsx')).toEqual(['src', 'components', 'Button.tsx']);
  });

  test('handles leading slash', () => {
    expect(getPathSegments('/src/index.ts')).toEqual(['src', 'index.ts']);
  });

  test('handles single segment', () => {
    expect(getPathSegments('file.txt')).toEqual(['file.txt']);
  });

  test('needs truncation for long path', () => {
    const longPath = 'very/long/path/to/deeply/nested/file.tsx';
    expect(shouldTruncatePath(longPath, 30)).toBe(true);
  });

  test('no truncation for short path', () => {
    expect(shouldTruncatePath('src/index.ts', 50)).toBe(false);
  });

  test('truncates to first and last segment', () => {
    const segments = ['src', 'components', 'ui', 'Button.tsx'];
    expect(getTruncatedSegments(segments)).toEqual(['src', '...', 'Button.tsx']);
  });

  test('does not truncate 2 segments', () => {
    const segments = ['src', 'index.ts'];
    expect(getTruncatedSegments(segments)).toEqual(['src', 'index.ts']);
  });
});

describe('FilesView - responsive treeWidth', () => {
  type Breakpoint = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

  const TREE_WIDTHS: Record<Breakpoint, number> = {
    xs: 16,
    sm: 20,
    md: 25,
    lg: 30,
    xl: 35,
  };

  function getTreeWidth(breakpoint: Breakpoint): number {
    return TREE_WIDTHS[breakpoint];
  }

  test('xs is 16 cols', () => {
    expect(getTreeWidth('xs')).toBe(16);
  });

  test('sm is 20 cols', () => {
    expect(getTreeWidth('sm')).toBe(20);
  });

  test('md is 25 cols', () => {
    expect(getTreeWidth('md')).toBe(25);
  });

  test('lg is 30 cols', () => {
    expect(getTreeWidth('lg')).toBe(30);
  });

  test('xl is 35 cols', () => {
    expect(getTreeWidth('xl')).toBe(35);
  });
});

describe('FilesView - preview width calculation', () => {
  function calculatePreviewWidth(terminalWidth: number, treeWidth: number): number {
    return terminalWidth - treeWidth - 4;
  }

  test('standard terminal (80 cols)', () => {
    expect(calculatePreviewWidth(80, 20)).toBe(56);
  });

  test('wide terminal (120 cols)', () => {
    expect(calculatePreviewWidth(120, 30)).toBe(86);
  });

  test('narrow terminal (60 cols)', () => {
    expect(calculatePreviewWidth(60, 16)).toBe(40);
  });
});

describe('FilesView - visible window calculation', () => {
  function calculateVisibleWindow(
    selectedIndex: number,
    totalItems: number,
    maxHeight: number
  ): { start: number; count: number } {
    const visibleCount = Math.max(1, maxHeight - 2);
    const start = Math.max(0, Math.min(
      selectedIndex - Math.floor(visibleCount / 2),
      totalItems - visibleCount
    ));
    return { start, count: visibleCount };
  }

  test('centers on selected item', () => {
    const { start, count } = calculateVisibleWindow(10, 50, 12);
    expect(count).toBe(10);
    expect(start).toBe(5); // selectedIndex(10) - floor(10/2) = 5
  });

  test('window at start', () => {
    const { start } = calculateVisibleWindow(2, 50, 12);
    expect(start).toBe(0);
  });

  test('window at end', () => {
    const { start, count } = calculateVisibleWindow(48, 50, 12);
    expect(start).toBe(40); // 50 - 10 = 40
  });

  test('minimum count is 1', () => {
    const { count } = calculateVisibleWindow(0, 10, 2);
    expect(count).toBe(1);
  });
});

describe('FilesView - git summary', () => {
  interface GitSummary {
    modified: number;
    added: number;
    deleted: number;
    untracked: number;
    total: number;
  }

  function calculateGitSummary(files: { status: string }[]): GitSummary {
    const summary: GitSummary = { modified: 0, added: 0, deleted: 0, untracked: 0, total: 0 };
    for (const file of files) {
      summary.total++;
      switch (file.status) {
        case 'modified':
          summary.modified++;
          break;
        case 'added':
          summary.added++;
          break;
        case 'deleted':
          summary.deleted++;
          break;
        case 'untracked':
          summary.untracked++;
          break;
      }
    }
    return summary;
  }

  test('calculates summary correctly', () => {
    const files = [
      { status: 'modified' },
      { status: 'modified' },
      { status: 'added' },
      { status: 'deleted' },
      { status: 'untracked' },
    ];
    const summary = calculateGitSummary(files);
    expect(summary.modified).toBe(2);
    expect(summary.added).toBe(1);
    expect(summary.deleted).toBe(1);
    expect(summary.untracked).toBe(1);
    expect(summary.total).toBe(5);
  });

  test('empty files', () => {
    const summary = calculateGitSummary([]);
    expect(summary.total).toBe(0);
  });
});

describe('FilesView - directory icon', () => {
  function getDirectoryIcon(expanded: boolean): string {
    return expanded ? '[-]' : '[+]';
  }

  function getFileIcon(): string {
    return '   ';
  }

  test('expanded directory is [-]', () => {
    expect(getDirectoryIcon(true)).toBe('[-]');
  });

  test('collapsed directory is [+]', () => {
    expect(getDirectoryIcon(false)).toBe('[+]');
  });

  test('file icon is spaces', () => {
    expect(getFileIcon()).toBe('   ');
  });
});

describe('FilesView - worktree filtering', () => {
  interface Worktree {
    agent: string;
    path: string;
    status: 'OK' | 'ORPHANED';
  }

  function filterActiveWorktrees(worktrees: Worktree[]): Worktree[] {
    return worktrees.filter(w => w.status === 'OK');
  }

  test('filters to only OK worktrees', () => {
    const worktrees: Worktree[] = [
      { agent: 'eng-01', path: '/path/1', status: 'OK' },
      { agent: 'eng-02', path: '/path/2', status: 'ORPHANED' },
      { agent: 'eng-03', path: '/path/3', status: 'OK' },
    ];
    const active = filterActiveWorktrees(worktrees);
    expect(active).toHaveLength(2);
    expect(active[0].agent).toBe('eng-01');
    expect(active[1].agent).toBe('eng-03');
  });

  test('returns empty for all orphaned', () => {
    const worktrees: Worktree[] = [
      { agent: 'old-1', path: '/path/1', status: 'ORPHANED' },
    ];
    expect(filterActiveWorktrees(worktrees)).toHaveLength(0);
  });
});

describe('FilesView - file size limit', () => {
  const MAX_FILE_SIZE = 100 * 1024; // 100KB

  function isFileTooLarge(sizeBytes: number): boolean {
    return sizeBytes > MAX_FILE_SIZE;
  }

  test('small file is not too large', () => {
    expect(isFileTooLarge(1000)).toBe(false);
  });

  test('100KB is not too large', () => {
    expect(isFileTooLarge(100 * 1024)).toBe(false);
  });

  test('100KB + 1 byte is too large', () => {
    expect(isFileTooLarge(100 * 1024 + 1)).toBe(true);
  });

  test('1MB is too large', () => {
    expect(isFileTooLarge(1024 * 1024)).toBe(true);
  });
});

describe('FilesView - responsive hints', () => {
  type Breakpoint = 'xs' | 'sm' | 'default';

  const HINTS: Record<Breakpoint, string> = {
    xs: 'j/k nav · Enter sel · w tree · Esc',
    sm: 'j/k nav | Enter expand | w worktree | Esc back',
    default: 'j/k: nav | Enter: expand/select | w: worktree | Tab: focus | Esc: back',
  };

  function getHintText(breakpoint: Breakpoint): string {
    return HINTS[breakpoint];
  }

  test('xs hints are shortest', () => {
    const hint = getHintText('xs');
    expect(hint).toContain('j/k nav');
    expect(hint.length).toBeLessThan(50);
  });

  test('sm hints are medium', () => {
    const hint = getHintText('sm');
    expect(hint).toContain('Enter expand');
  });

  test('default hints are most verbose', () => {
    const hint = getHintText('default');
    expect(hint).toContain('Tab: focus');
  });
});
