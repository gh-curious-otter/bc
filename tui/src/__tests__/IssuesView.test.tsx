/**
 * IssuesView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('IssuesView - getLabelColor', () => {
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

  function getLabelColor(name: string): string {
    return LABEL_COLORS[name] ?? 'gray';
  }

  test('bug is red', () => {
    expect(getLabelColor('bug')).toBe('red');
  });

  test('enhancement is green', () => {
    expect(getLabelColor('enhancement')).toBe('green');
  });

  test('feature is cyan', () => {
    expect(getLabelColor('feature')).toBe('cyan');
  });

  test('P0-critical is red', () => {
    expect(getLabelColor('P0-critical')).toBe('red');
  });

  test('P1-high is yellow', () => {
    expect(getLabelColor('P1-high')).toBe('yellow');
  });

  test('P2-medium is blue', () => {
    expect(getLabelColor('P2-medium')).toBe('blue');
  });

  test('P3-low is gray', () => {
    expect(getLabelColor('P3-low')).toBe('gray');
  });

  test('tui is magenta', () => {
    expect(getLabelColor('tui')).toBe('magenta');
  });

  test('go is blue', () => {
    expect(getLabelColor('go')).toBe('blue');
  });

  test('epic is cyan', () => {
    expect(getLabelColor('epic')).toBe('cyan');
  });

  test('task is white', () => {
    expect(getLabelColor('task')).toBe('white');
  });

  test('unknown label is gray', () => {
    expect(getLabelColor('unknown')).toBe('gray');
  });

  test('empty string is gray', () => {
    expect(getLabelColor('')).toBe('gray');
  });
});

describe('IssuesView - formatRelativeDate', () => {
  function formatRelativeDate(dateStr: string): string {
    try {
      const date = new Date(dateStr);
      const now = new Date();
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

  test('today', () => {
    const now = new Date().toISOString();
    expect(formatRelativeDate(now)).toBe('today');
  });

  test('yesterday', () => {
    const yesterday = new Date(Date.now() - 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(yesterday)).toBe('yesterday');
  });

  test('days ago', () => {
    const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(threeDaysAgo)).toBe('3d ago');
  });

  test('6 days is days', () => {
    const sixDaysAgo = new Date(Date.now() - 6 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(sixDaysAgo)).toBe('6d ago');
  });

  test('1 week ago', () => {
    const oneWeekAgo = new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(oneWeekAgo)).toBe('1w ago');
  });

  test('2 weeks ago', () => {
    const twoWeeksAgo = new Date(Date.now() - 14 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(twoWeeksAgo)).toBe('2w ago');
  });

  test('1 month ago', () => {
    const oneMonthAgo = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(oneMonthAgo)).toBe('1mo ago');
  });

  test('3 months ago', () => {
    const threeMonthsAgo = new Date(Date.now() - 90 * 24 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeDate(threeMonthsAgo)).toBe('3mo ago');
  });
});

describe('IssuesView - filter by label', () => {
  interface Label {
    name: string;
  }

  interface MockIssue {
    number: number;
    title: string;
    labels: Label[];
  }

  const mockIssues: MockIssue[] = [
    { number: 1, title: 'Bug fix', labels: [{ name: 'bug' }] },
    { number: 2, title: 'Feature', labels: [{ name: 'feature' }, { name: 'P1-high' }] },
    { number: 3, title: 'Enhancement', labels: [{ name: 'enhancement' }] },
    { number: 4, title: 'Another bug', labels: [{ name: 'bug' }, { name: 'P2-medium' }] },
    { number: 5, title: 'No labels', labels: [] },
  ];

  function filterByLabel(issues: MockIssue[], label: string | null): MockIssue[] {
    if (!label) return issues;
    return issues.filter(issue =>
      issue.labels.some(l => l.name === label)
    );
  }

  test('returns all when no filter', () => {
    expect(filterByLabel(mockIssues, null)).toHaveLength(5);
  });

  test('filters by bug label', () => {
    const result = filterByLabel(mockIssues, 'bug');
    expect(result).toHaveLength(2);
    expect(result[0].number).toBe(1);
    expect(result[1].number).toBe(4);
  });

  test('filters by feature label', () => {
    const result = filterByLabel(mockIssues, 'feature');
    expect(result).toHaveLength(1);
    expect(result[0].number).toBe(2);
  });

  test('returns empty for non-matching label', () => {
    expect(filterByLabel(mockIssues, 'nonexistent')).toHaveLength(0);
  });
});

describe('IssuesView - unique labels extraction', () => {
  interface Label {
    name: string;
  }

  interface MockIssue {
    labels: Label[];
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

  test('extracts unique labels', () => {
    const issues: MockIssue[] = [
      { labels: [{ name: 'bug' }, { name: 'P1-high' }] },
      { labels: [{ name: 'feature' }] },
      { labels: [{ name: 'bug' }] }, // duplicate
    ];
    const result = extractUniqueLabels(issues);
    expect(result).toEqual(['P1-high', 'bug', 'feature']);
  });

  test('returns empty for no issues', () => {
    expect(extractUniqueLabels([])).toEqual([]);
  });

  test('returns empty for issues with no labels', () => {
    const issues: MockIssue[] = [{ labels: [] }, { labels: [] }];
    expect(extractUniqueLabels(issues)).toEqual([]);
  });

  test('sorts alphabetically', () => {
    const issues: MockIssue[] = [
      { labels: [{ name: 'zebra' }] },
      { labels: [{ name: 'alpha' }] },
      { labels: [{ name: 'mango' }] },
    ];
    expect(extractUniqueLabels(issues)).toEqual(['alpha', 'mango', 'zebra']);
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
    current: string | null,
    labels: string[]
  ): string | null {
    const currentIdx = current ? labels.indexOf(current) : -1;
    if (currentIdx === labels.length - 1) {
      return null;
    }
    return labels[currentIdx + 1] ?? null;
  }

  test('null cycles to first label', () => {
    expect(cycleLabelFilter(null, ['bug', 'feature', 'epic'])).toBe('bug');
  });

  test('cycles to next label', () => {
    expect(cycleLabelFilter('bug', ['bug', 'feature', 'epic'])).toBe('feature');
  });

  test('last label cycles to null', () => {
    expect(cycleLabelFilter('epic', ['bug', 'feature', 'epic'])).toBeNull();
  });

  test('empty labels returns null', () => {
    expect(cycleLabelFilter(null, [])).toBeNull();
  });
});

describe('IssuesView - issue state color', () => {
  function getStateColor(state: string): string {
    return state === 'OPEN' ? 'green' : 'red';
  }

  test('OPEN is green', () => {
    expect(getStateColor('OPEN')).toBe('green');
  });

  test('CLOSED is red', () => {
    expect(getStateColor('CLOSED')).toBe('red');
  });

  test('other states are red', () => {
    expect(getStateColor('MERGED')).toBe('red');
  });
});

describe('IssuesView - primary label extraction', () => {
  interface Label {
    name: string;
  }

  function getPrimaryLabel(labels: Label[]): string {
    return labels[0]?.name ?? '';
  }

  test('gets first label', () => {
    const labels: Label[] = [{ name: 'bug' }, { name: 'P1-high' }];
    expect(getPrimaryLabel(labels)).toBe('bug');
  });

  test('returns empty for no labels', () => {
    expect(getPrimaryLabel([])).toBe('');
  });

  test('single label', () => {
    expect(getPrimaryLabel([{ name: 'feature' }])).toBe('feature');
  });
});

describe('IssuesView - body truncation', () => {
  function truncateBody(body: string, maxLen = 500): string {
    return body.length > maxLen
      ? body.slice(0, maxLen) + '...'
      : body;
  }

  test('short body not truncated', () => {
    const short = 'This is a short body';
    expect(truncateBody(short)).toBe(short);
  });

  test('long body truncated with ellipsis', () => {
    const long = 'a'.repeat(600);
    const result = truncateBody(long);
    expect(result.length).toBe(503);
    expect(result.endsWith('...')).toBe(true);
  });

  test('exact length not truncated', () => {
    const exact = 'a'.repeat(500);
    expect(truncateBody(exact)).toBe(exact);
  });
});

describe('IssuesView - comment display', () => {
  interface Comment {
    author: { login: string };
    body: string;
    createdAt: string;
  }

  function getDisplayedComments(comments: Comment[], limit = 3): Comment[] {
    return comments.slice(0, limit);
  }

  function getRemainingCount(comments: Comment[], limit = 3): number {
    return Math.max(0, comments.length - limit);
  }

  test('shows up to 3 comments', () => {
    const comments: Comment[] = Array.from({ length: 5 }, (_, i) => ({
      author: { login: `user${i}` },
      body: `Comment ${i}`,
      createdAt: new Date().toISOString(),
    }));
    expect(getDisplayedComments(comments)).toHaveLength(3);
  });

  test('shows all when fewer than limit', () => {
    const comments: Comment[] = [
      { author: { login: 'user1' }, body: 'Comment', createdAt: new Date().toISOString() },
    ];
    expect(getDisplayedComments(comments)).toHaveLength(1);
  });

  test('remaining count is correct', () => {
    const comments: Comment[] = Array.from({ length: 5 }, (_, i) => ({
      author: { login: `user${i}` },
      body: `Comment ${i}`,
      createdAt: new Date().toISOString(),
    }));
    expect(getRemainingCount(comments)).toBe(2);
  });

  test('remaining count is 0 when at limit', () => {
    const comments: Comment[] = Array.from({ length: 3 }, (_, i) => ({
      author: { login: `user${i}` },
      body: `Comment ${i}`,
      createdAt: new Date().toISOString(),
    }));
    expect(getRemainingCount(comments)).toBe(0);
  });
});

describe('IssuesView - footer hints', () => {
  interface Hint {
    key: string;
    label: string;
  }

  function getFooterHints(labelFilter: string | null, stateFilter: string): Hint[] {
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

  test('filter hint shows label when active', () => {
    const hints = getFooterHints('bug', 'open');
    const filterHint = hints.find(h => h.key === 'f');
    expect(filterHint?.label).toBe('filter:bug');
  });

  test('filter hint shows "filter" when inactive', () => {
    const hints = getFooterHints(null, 'open');
    const filterHint = hints.find(h => h.key === 'f');
    expect(filterHint?.label).toBe('filter');
  });

  test('state hint shows current state', () => {
    const hints = getFooterHints(null, 'closed');
    const stateHint = hints.find(h => h.key === 's');
    expect(stateHint?.label).toBe('closed');
  });

  test('truncates long label filter', () => {
    const hints = getFooterHints('P0-critical', 'open');
    const filterHint = hints.find(h => h.key === 'f');
    expect(filterHint?.label).toBe('filter:P0-criti');
  });
});

describe('IssuesView - stats bar', () => {
  interface IssueCounts {
    open: number;
    closed: number;
    total: number;
  }

  function getCounts(issues: { state: string }[]): IssueCounts {
    const open = issues.filter(i => i.state === 'OPEN').length;
    const closed = issues.filter(i => i.state === 'CLOSED').length;
    return { open, closed, total: issues.length };
  }

  test('counts issues correctly', () => {
    const issues = [
      { state: 'OPEN' },
      { state: 'OPEN' },
      { state: 'CLOSED' },
    ];
    const counts = getCounts(issues);
    expect(counts.open).toBe(2);
    expect(counts.closed).toBe(1);
    expect(counts.total).toBe(3);
  });

  test('handles empty issues', () => {
    const counts = getCounts([]);
    expect(counts.open).toBe(0);
    expect(counts.closed).toBe(0);
    expect(counts.total).toBe(0);
  });
});
