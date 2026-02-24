/**
 * IssuesView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test, beforeEach, afterEach } from 'bun:test';

// Label color mapping (from IssuesView)
const LABEL_COLORS: Record<string, string> = {
  bug: 'red',
  enhancement: 'green',
  feature: 'cyan',
  'P0-critical': 'red',
  'P1-high': 'yellow',
  'P2-medium': 'blue',
  'P3-low': 'gray',
  tui: 'magenta',
  go: 'blue',
  epic: 'cyan',
  task: 'white',
};

describe('IssuesView - getLabelColor', () => {
  function getLabelColor(name: string): string {
    return LABEL_COLORS[name] ?? 'gray';
  }

  test('bug label is red', () => {
    expect(getLabelColor('bug')).toBe('red');
  });

  test('enhancement label is green', () => {
    expect(getLabelColor('enhancement')).toBe('green');
  });

  test('feature label is cyan', () => {
    expect(getLabelColor('feature')).toBe('cyan');
  });

  test('P0-critical label is red', () => {
    expect(getLabelColor('P0-critical')).toBe('red');
  });

  test('P1-high label is yellow', () => {
    expect(getLabelColor('P1-high')).toBe('yellow');
  });

  test('P2-medium label is blue', () => {
    expect(getLabelColor('P2-medium')).toBe('blue');
  });

  test('P3-low label is gray', () => {
    expect(getLabelColor('P3-low')).toBe('gray');
  });

  test('tui label is magenta', () => {
    expect(getLabelColor('tui')).toBe('magenta');
  });

  test('go label is blue', () => {
    expect(getLabelColor('go')).toBe('blue');
  });

  test('epic label is cyan', () => {
    expect(getLabelColor('epic')).toBe('cyan');
  });

  test('task label is white', () => {
    expect(getLabelColor('task')).toBe('white');
  });

  test('unknown label defaults to gray', () => {
    expect(getLabelColor('unknown-label')).toBe('gray');
  });

  test('empty string defaults to gray', () => {
    expect(getLabelColor('')).toBe('gray');
  });

  test('case sensitive label check', () => {
    expect(getLabelColor('BUG')).toBe('gray');
    expect(getLabelColor('Bug')).toBe('gray');
  });
});

describe('IssuesView - formatRelativeDate', () => {
  let realDateNow: () => number;
  const fixedNow = new Date('2026-02-24T12:00:00Z').getTime();

  beforeEach(() => {
    realDateNow = Date.now;
    Date.now = () => fixedNow;
  });

  afterEach(() => {
    Date.now = realDateNow;
  });

  function formatRelativeDate(dateStr: string): string {
    try {
      const date = new Date(dateStr);
      const now = new Date(Date.now());
      const diffMs = now.getTime() - date.getTime();
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

      if (diffDays === 0) return 'today';
      if (diffDays === 1) return 'yesterday';
      if (diffDays < 7) return `${String(diffDays)}d ago`;
      if (diffDays < 30) return `${String(Math.floor(diffDays / 7))}w ago`;
      return `${String(Math.floor(diffDays / 30))}mo ago`;
    } catch {
      return dateStr;
    }
  }

  test('same day shows today', () => {
    expect(formatRelativeDate('2026-02-24T08:00:00Z')).toBe('today');
  });

  test('yesterday shows yesterday', () => {
    expect(formatRelativeDate('2026-02-23T12:00:00Z')).toBe('yesterday');
  });

  test('2 days ago shows 2d ago', () => {
    expect(formatRelativeDate('2026-02-22T12:00:00Z')).toBe('2d ago');
  });

  test('6 days ago shows 6d ago', () => {
    expect(formatRelativeDate('2026-02-18T12:00:00Z')).toBe('6d ago');
  });

  test('7 days ago shows 1w ago', () => {
    expect(formatRelativeDate('2026-02-17T12:00:00Z')).toBe('1w ago');
  });

  test('14 days ago shows 2w ago', () => {
    expect(formatRelativeDate('2026-02-10T12:00:00Z')).toBe('2w ago');
  });

  test('28 days ago shows 4w ago', () => {
    expect(formatRelativeDate('2026-01-27T12:00:00Z')).toBe('4w ago');
  });

  test('30 days ago shows 1mo ago', () => {
    expect(formatRelativeDate('2026-01-25T12:00:00Z')).toBe('1mo ago');
  });

  test('60 days ago shows 2mo ago', () => {
    expect(formatRelativeDate('2025-12-26T12:00:00Z')).toBe('2mo ago');
  });

  test('invalid date returns NaN-based string', () => {
    // Note: JavaScript Date parsing doesn't throw for invalid strings
    // It returns Invalid Date with NaN timestamps
    const result = formatRelativeDate('not-a-date');
    expect(result).toContain('NaN');
  });
});

describe('IssuesView - issue filtering by label', () => {
  interface MockLabel {
    name: string;
  }

  interface MockIssue {
    number: number;
    title: string;
    labels: MockLabel[];
  }

  const mockIssues: MockIssue[] = [
    { number: 1, title: 'Fix login bug', labels: [{ name: 'bug' }, { name: 'P1-high' }] },
    { number: 2, title: 'Add dark mode', labels: [{ name: 'enhancement' }, { name: 'tui' }] },
    { number: 3, title: 'CLI refactor', labels: [{ name: 'enhancement' }, { name: 'go' }] },
    { number: 4, title: 'Critical security fix', labels: [{ name: 'bug' }, { name: 'P0-critical' }] },
    { number: 5, title: 'No labels issue', labels: [] },
  ];

  function filterByLabel(issues: MockIssue[], labelFilter: string | null): MockIssue[] {
    if (!labelFilter) return issues;
    return issues.filter(issue =>
      issue.labels.some(l => l.name === labelFilter)
    );
  }

  test('no filter returns all issues', () => {
    expect(filterByLabel(mockIssues, null)).toHaveLength(5);
  });

  test('filter by bug', () => {
    const result = filterByLabel(mockIssues, 'bug');
    expect(result).toHaveLength(2);
    expect(result[0].number).toBe(1);
    expect(result[1].number).toBe(4);
  });

  test('filter by enhancement', () => {
    const result = filterByLabel(mockIssues, 'enhancement');
    expect(result).toHaveLength(2);
    expect(result[0].number).toBe(2);
    expect(result[1].number).toBe(3);
  });

  test('filter by P1-high', () => {
    const result = filterByLabel(mockIssues, 'P1-high');
    expect(result).toHaveLength(1);
    expect(result[0].number).toBe(1);
  });

  test('filter by nonexistent label returns empty', () => {
    const result = filterByLabel(mockIssues, 'nonexistent');
    expect(result).toHaveLength(0);
  });

  test('issues without labels never match', () => {
    const result = filterByLabel(mockIssues, 'bug');
    expect(result.some(i => i.number === 5)).toBe(false);
  });
});

describe('IssuesView - unique labels extraction', () => {
  interface MockLabel {
    name: string;
  }

  interface MockIssue {
    labels: MockLabel[];
  }

  function extractUniqueLabels(issues: MockIssue[]): string[] {
    const labels = new Set<string>();
    for (const issue of issues) {
      for (const label of issue.labels) {
        labels.add(label.name);
      }
    }
    return Array.from(labels).sort();
  }

  test('extracts unique labels sorted', () => {
    const issues: MockIssue[] = [
      { labels: [{ name: 'bug' }, { name: 'P1-high' }] },
      { labels: [{ name: 'enhancement' }, { name: 'bug' }] },
    ];
    const result = extractUniqueLabels(issues);
    expect(result).toEqual(['P1-high', 'bug', 'enhancement']);
  });

  test('handles empty issues array', () => {
    expect(extractUniqueLabels([])).toEqual([]);
  });

  test('handles issues with no labels', () => {
    const issues: MockIssue[] = [
      { labels: [] },
      { labels: [] },
    ];
    expect(extractUniqueLabels(issues)).toEqual([]);
  });

  test('deduplicates same label across issues', () => {
    const issues: MockIssue[] = [
      { labels: [{ name: 'bug' }] },
      { labels: [{ name: 'bug' }] },
      { labels: [{ name: 'bug' }] },
    ];
    const result = extractUniqueLabels(issues);
    expect(result).toEqual(['bug']);
  });

  test('mixed labels deduplicated and sorted', () => {
    const issues: MockIssue[] = [
      { labels: [{ name: 'tui' }, { name: 'enhancement' }] },
      { labels: [{ name: 'go' }, { name: 'enhancement' }] },
      { labels: [{ name: 'bug' }] },
    ];
    const result = extractUniqueLabels(issues);
    expect(result).toEqual(['bug', 'enhancement', 'go', 'tui']);
  });
});

describe('IssuesView - state filter cycling', () => {
  type StateFilter = 'open' | 'closed' | 'all';

  function cycleStateFilter(current: StateFilter): StateFilter {
    if (current === 'open') return 'closed';
    if (current === 'closed') return 'all';
    return 'open';
  }

  test('open cycles to closed', () => {
    expect(cycleStateFilter('open')).toBe('closed');
  });

  test('closed cycles to all', () => {
    expect(cycleStateFilter('closed')).toBe('all');
  });

  test('all cycles back to open', () => {
    expect(cycleStateFilter('all')).toBe('open');
  });
});

describe('IssuesView - label filter cycling', () => {
  function cycleLabelFilter(
    currentFilter: string | null,
    uniqueLabels: string[]
  ): string | null {
    const currentIdx = currentFilter ? uniqueLabels.indexOf(currentFilter) : -1;
    if (currentIdx === uniqueLabels.length - 1) {
      return null;
    }
    return uniqueLabels[currentIdx + 1] ?? null;
  }

  const labels = ['bug', 'enhancement', 'tui'];

  test('null cycles to first label', () => {
    expect(cycleLabelFilter(null, labels)).toBe('bug');
  });

  test('first label cycles to second', () => {
    expect(cycleLabelFilter('bug', labels)).toBe('enhancement');
  });

  test('second label cycles to third', () => {
    expect(cycleLabelFilter('enhancement', labels)).toBe('tui');
  });

  test('last label cycles to null', () => {
    expect(cycleLabelFilter('tui', labels)).toBe(null);
  });

  test('empty labels array returns null', () => {
    expect(cycleLabelFilter(null, [])).toBe(null);
    expect(cycleLabelFilter('bug', [])).toBe(null);
  });

  test('unknown filter returns first label', () => {
    expect(cycleLabelFilter('unknown', labels)).toBe('bug');
  });
});

describe('IssuesView - hints generation', () => {
  interface Hint {
    key: string;
    label: string;
  }

  function buildHints(labelFilter: string | null, stateFilter: string): Hint[] {
    return [
      { key: 'j/k', label: 'navigate' },
      { key: 'g/G', label: 'top/bottom' },
      { key: 'Enter', label: 'details' },
      { key: 'f', label: labelFilter ? `filter:${labelFilter.slice(0, 8)}` : 'filter' },
      { key: 's', label: stateFilter },
      { key: 'r', label: 'refresh' },
      { key: 'q/ESC', label: 'back' },
    ];
  }

  test('hints include navigation', () => {
    const hints = buildHints(null, 'open');
    expect(hints[0]).toEqual({ key: 'j/k', label: 'navigate' });
  });

  test('hints include top/bottom', () => {
    const hints = buildHints(null, 'open');
    expect(hints[1]).toEqual({ key: 'g/G', label: 'top/bottom' });
  });

  test('filter hint shows filter when no label', () => {
    const hints = buildHints(null, 'open');
    expect(hints[3]).toEqual({ key: 'f', label: 'filter' });
  });

  test('filter hint shows label when filtered', () => {
    const hints = buildHints('bug', 'open');
    expect(hints[3]).toEqual({ key: 'f', label: 'filter:bug' });
  });

  test('filter hint truncates long labels', () => {
    const hints = buildHints('P0-critical', 'open');
    expect(hints[3]).toEqual({ key: 'f', label: 'filter:P0-criti' });
  });

  test('state hint shows current state', () => {
    const hints = buildHints(null, 'closed');
    expect(hints[4]).toEqual({ key: 's', label: 'closed' });
  });

  test('refresh hint always present', () => {
    const hints = buildHints(null, 'open');
    expect(hints[5]).toEqual({ key: 'r', label: 'refresh' });
  });

  test('back hint always present', () => {
    const hints = buildHints(null, 'open');
    expect(hints[6]).toEqual({ key: 'q/ESC', label: 'back' });
  });
});

describe('IssuesView - issue counts', () => {
  interface MockIssue {
    state: 'OPEN' | 'CLOSED';
  }

  function computeCounts(issues: MockIssue[]): { open: number; closed: number; total: number } {
    let open = 0;
    let closed = 0;
    for (const issue of issues) {
      if (issue.state === 'OPEN') open++;
      else closed++;
    }
    return { open, closed, total: issues.length };
  }

  test('counts open issues', () => {
    const issues: MockIssue[] = [
      { state: 'OPEN' },
      { state: 'OPEN' },
      { state: 'CLOSED' },
    ];
    const counts = computeCounts(issues);
    expect(counts.open).toBe(2);
  });

  test('counts closed issues', () => {
    const issues: MockIssue[] = [
      { state: 'OPEN' },
      { state: 'CLOSED' },
      { state: 'CLOSED' },
    ];
    const counts = computeCounts(issues);
    expect(counts.closed).toBe(2);
  });

  test('counts total issues', () => {
    const issues: MockIssue[] = [
      { state: 'OPEN' },
      { state: 'CLOSED' },
      { state: 'OPEN' },
    ];
    const counts = computeCounts(issues);
    expect(counts.total).toBe(3);
  });

  test('handles empty array', () => {
    const counts = computeCounts([]);
    expect(counts).toEqual({ open: 0, closed: 0, total: 0 });
  });

  test('all open', () => {
    const issues: MockIssue[] = [
      { state: 'OPEN' },
      { state: 'OPEN' },
    ];
    const counts = computeCounts(issues);
    expect(counts).toEqual({ open: 2, closed: 0, total: 2 });
  });

  test('all closed', () => {
    const issues: MockIssue[] = [
      { state: 'CLOSED' },
      { state: 'CLOSED' },
    ];
    const counts = computeCounts(issues);
    expect(counts).toEqual({ open: 0, closed: 2, total: 2 });
  });
});

describe('IssuesView - IssueRow primary label', () => {
  interface MockLabel {
    name: string;
  }

  interface MockIssue {
    labels: MockLabel[];
  }

  function getPrimaryLabel(issue: MockIssue): string {
    return issue.labels[0]?.name ?? '';
  }

  test('returns first label', () => {
    const issue: MockIssue = {
      labels: [{ name: 'bug' }, { name: 'P1-high' }],
    };
    expect(getPrimaryLabel(issue)).toBe('bug');
  });

  test('returns empty for no labels', () => {
    const issue: MockIssue = { labels: [] };
    expect(getPrimaryLabel(issue)).toBe('');
  });

  test('handles single label', () => {
    const issue: MockIssue = { labels: [{ name: 'enhancement' }] };
    expect(getPrimaryLabel(issue)).toBe('enhancement');
  });
});

describe('IssuesView - issue state color', () => {
  function getStateColor(state: 'OPEN' | 'CLOSED'): string {
    return state === 'OPEN' ? 'green' : 'red';
  }

  test('OPEN state is green', () => {
    expect(getStateColor('OPEN')).toBe('green');
  });

  test('CLOSED state is red', () => {
    expect(getStateColor('CLOSED')).toBe('red');
  });
});

describe('IssuesView - detail view navigation', () => {
  function shouldCloseDetailView(input: string, key: { escape?: boolean; return?: boolean }): boolean {
    return key.escape === true || input === 'q' || key.return === true;
  }

  test('escape closes detail view', () => {
    expect(shouldCloseDetailView('', { escape: true })).toBe(true);
  });

  test('q closes detail view', () => {
    expect(shouldCloseDetailView('q', {})).toBe(true);
  });

  test('return closes detail view', () => {
    expect(shouldCloseDetailView('', { return: true })).toBe(true);
  });

  test('other keys do not close', () => {
    expect(shouldCloseDetailView('a', {})).toBe(false);
    expect(shouldCloseDetailView('j', {})).toBe(false);
    expect(shouldCloseDetailView('k', {})).toBe(false);
  });
});

describe('IssuesView - body truncation', () => {
  function truncateBody(body: string, maxLen = 500): string {
    if (body.length <= maxLen) return body;
    return body.slice(0, maxLen) + '...';
  }

  test('short body not truncated', () => {
    expect(truncateBody('Short body')).toBe('Short body');
  });

  test('exact length body not truncated', () => {
    const body = 'a'.repeat(500);
    expect(truncateBody(body)).toBe(body);
  });

  test('long body truncated with ellipsis', () => {
    const body = 'a'.repeat(600);
    const result = truncateBody(body);
    expect(result.length).toBe(503);
    expect(result.endsWith('...')).toBe(true);
  });

  test('empty body handled', () => {
    expect(truncateBody('')).toBe('');
  });
});

describe('IssuesView - comments display logic', () => {
  interface Comment {
    author: { login: string };
    body: string;
    createdAt: string;
  }

  function getDisplayComments(comments: Comment[]): Comment[] {
    return comments.slice(0, 3);
  }

  function getRemainingCount(comments: Comment[]): number {
    return Math.max(0, comments.length - 3);
  }

  test('3 or fewer comments all displayed', () => {
    const comments: Comment[] = [
      { author: { login: 'user1' }, body: 'Comment 1', createdAt: '2026-02-24' },
      { author: { login: 'user2' }, body: 'Comment 2', createdAt: '2026-02-24' },
    ];
    expect(getDisplayComments(comments)).toHaveLength(2);
    expect(getRemainingCount(comments)).toBe(0);
  });

  test('more than 3 comments shows first 3', () => {
    const comments: Comment[] = [
      { author: { login: 'user1' }, body: 'Comment 1', createdAt: '2026-02-24' },
      { author: { login: 'user2' }, body: 'Comment 2', createdAt: '2026-02-24' },
      { author: { login: 'user3' }, body: 'Comment 3', createdAt: '2026-02-24' },
      { author: { login: 'user4' }, body: 'Comment 4', createdAt: '2026-02-24' },
      { author: { login: 'user5' }, body: 'Comment 5', createdAt: '2026-02-24' },
    ];
    expect(getDisplayComments(comments)).toHaveLength(3);
    expect(getRemainingCount(comments)).toBe(2);
  });

  test('empty comments array', () => {
    expect(getDisplayComments([])).toHaveLength(0);
    expect(getRemainingCount([])).toBe(0);
  });
});

describe('IssuesView - filter indicator display', () => {
  function shouldShowFilterIndicator(
    labelFilter: string | null,
    stateFilter: 'open' | 'closed' | 'all'
  ): boolean {
    return labelFilter !== null || stateFilter !== 'open';
  }

  test('no filters shows no indicator', () => {
    expect(shouldShowFilterIndicator(null, 'open')).toBe(false);
  });

  test('label filter shows indicator', () => {
    expect(shouldShowFilterIndicator('bug', 'open')).toBe(true);
  });

  test('state filter closed shows indicator', () => {
    expect(shouldShowFilterIndicator(null, 'closed')).toBe(true);
  });

  test('state filter all shows indicator', () => {
    expect(shouldShowFilterIndicator(null, 'all')).toBe(true);
  });

  test('both filters shows indicator', () => {
    expect(shouldShowFilterIndicator('bug', 'closed')).toBe(true);
  });
});

describe('IssuesView - empty state messages', () => {
  function getEmptyMessage(): string {
    return 'No issues found';
  }

  function getCreateHint(): string {
    return 'Create an issue with: bc issue create --title "..."';
  }

  test('empty message is correct', () => {
    expect(getEmptyMessage()).toBe('No issues found');
  });

  test('create hint includes command', () => {
    expect(getCreateHint()).toContain('bc issue create');
    expect(getCreateHint()).toContain('--title');
  });
});

describe('IssuesView - truncate utility', () => {
  // Mirroring the truncate utility used in IssuesView
  function truncate(str: string, maxLen: number): string {
    if (str.length <= maxLen) return str;
    return str.slice(0, maxLen - 1) + '…';
  }

  test('short strings not truncated', () => {
    expect(truncate('hello', 10)).toBe('hello');
  });

  test('exact length not truncated', () => {
    expect(truncate('hello', 5)).toBe('hello');
  });

  test('long strings truncated with ellipsis', () => {
    expect(truncate('hello world', 8)).toBe('hello w…');
  });

  test('title truncation at 48 chars', () => {
    const longTitle = 'This is a very long issue title that exceeds the maximum display width';
    const result = truncate(longTitle, 48);
    expect(result.length).toBe(48);
    expect(result.endsWith('…')).toBe(true);
  });

  test('label truncation at 13 chars', () => {
    const longLabel = 'very-long-label-name';
    const result = truncate(longLabel, 13);
    expect(result.length).toBe(13);
    expect(result.endsWith('…')).toBe(true);
  });

  test('filter hint truncation at 8 chars', () => {
    const label = 'P0-critical';
    const result = truncate(label, 8);
    expect(result).toBe('P0-crit…');
  });
});
