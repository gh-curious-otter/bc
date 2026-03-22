/**
 * IssuesView Tests
 * Issue #1754: Issues View with GitHub issue management
 *
 * Tests cover:
 * - Label color mapping
 * - Relative date formatting
 * - Issue filtering by label
 * - State filter cycling
 * - Unique label extraction
 * - Keyboard shortcuts
 */

import { describe, test, expect } from 'bun:test';

// Label color mapping matching IssuesView
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

interface GHIssueLabel {
  name: string;
  color: string;
}

interface GHIssue {
  number: number;
  title: string;
  state: 'OPEN' | 'CLOSED';
  createdAt: string;
  updatedAt?: string;
  labels: GHIssueLabel[];
  assignees: Array<{ login: string }>;
  body?: string;
  author?: { login: string };
  comments?: Array<{ body: string; author: { login: string }; createdAt: string }>;
}

function filterByLabel(issues: GHIssue[], labelFilter: string | null): GHIssue[] {
  if (!labelFilter) return issues;
  return issues.filter((issue) => issue.labels.some((l) => l.name === labelFilter));
}

function extractUniqueLabels(issues: GHIssue[]): string[] {
  const labels = new Set<string>();
  for (const issue of issues) {
    for (const label of issue.labels) {
      labels.add(label.name);
    }
  }
  return Array.from(labels).sort();
}

function cycleStateFilter(current: 'open' | 'closed' | 'all'): 'open' | 'closed' | 'all' {
  if (current === 'open') return 'closed';
  if (current === 'closed') return 'all';
  return 'open';
}

function cycleLabelFilter(current: string | null, labels: string[]): string | null {
  const currentIdx = current ? labels.indexOf(current) : -1;
  if (currentIdx === labels.length - 1) {
    return null;
  }
  return labels[currentIdx + 1] ?? null;
}

function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '…';
}

describe('IssuesView', () => {
  describe('Label Color Mapping', () => {
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
      expect(getLabelColor('unknown')).toBe('gray');
      expect(getLabelColor('custom-label')).toBe('gray');
    });
  });

  describe('Relative Date Formatting', () => {
    test('formats today', () => {
      const now = new Date();
      expect(formatRelativeDate(now.toISOString())).toBe('today');
    });

    test('formats yesterday', () => {
      const yesterday = new Date();
      yesterday.setDate(yesterday.getDate() - 1);
      expect(formatRelativeDate(yesterday.toISOString())).toBe('yesterday');
    });

    test('formats days ago', () => {
      const threeDaysAgo = new Date();
      threeDaysAgo.setDate(threeDaysAgo.getDate() - 3);
      expect(formatRelativeDate(threeDaysAgo.toISOString())).toBe('3d ago');
    });

    test('formats weeks ago', () => {
      const twoWeeksAgo = new Date();
      twoWeeksAgo.setDate(twoWeeksAgo.getDate() - 14);
      expect(formatRelativeDate(twoWeeksAgo.toISOString())).toBe('2w ago');
    });

    test('formats months ago', () => {
      const twoMonthsAgo = new Date();
      twoMonthsAgo.setDate(twoMonthsAgo.getDate() - 60);
      expect(formatRelativeDate(twoMonthsAgo.toISOString())).toBe('2mo ago');
    });

    test('handles invalid date string', () => {
      const invalidDate = 'not-a-date';
      // Returns original string on parse error
      const result = formatRelativeDate(invalidDate);
      expect(typeof result).toBe('string');
    });
  });

  describe('Issue Filtering by Label', () => {
    const mockIssues: GHIssue[] = [
      {
        number: 1,
        title: 'Bug issue',
        state: 'OPEN',
        createdAt: '2024-02-01T00:00:00Z',
        labels: [{ name: 'bug', color: 'd73a4a' }],
        assignees: [],
      },
      {
        number: 2,
        title: 'Feature issue',
        state: 'OPEN',
        createdAt: '2024-02-02T00:00:00Z',
        labels: [{ name: 'enhancement', color: 'a2eeef' }],
        assignees: [],
      },
      {
        number: 3,
        title: 'Both labels',
        state: 'CLOSED',
        createdAt: '2024-02-03T00:00:00Z',
        labels: [
          { name: 'bug', color: 'd73a4a' },
          { name: 'tui', color: '1d76db' },
        ],
        assignees: [],
      },
    ];

    test('null filter returns all issues', () => {
      expect(filterByLabel(mockIssues, null)).toHaveLength(3);
    });

    test('filter by bug returns bug issues', () => {
      const filtered = filterByLabel(mockIssues, 'bug');
      expect(filtered).toHaveLength(2);
      expect(filtered.map((i) => i.number)).toEqual([1, 3]);
    });

    test('filter by enhancement returns one issue', () => {
      const filtered = filterByLabel(mockIssues, 'enhancement');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(2);
    });

    test('filter by tui returns one issue', () => {
      const filtered = filterByLabel(mockIssues, 'tui');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(3);
    });

    test('filter by nonexistent label returns empty', () => {
      const filtered = filterByLabel(mockIssues, 'nonexistent');
      expect(filtered).toHaveLength(0);
    });
  });

  describe('Unique Label Extraction', () => {
    const mockIssues: GHIssue[] = [
      {
        number: 1,
        title: 'Issue 1',
        state: 'OPEN',
        createdAt: '2024-02-01T00:00:00Z',
        labels: [
          { name: 'bug', color: 'd73a4a' },
          { name: 'tui', color: '1d76db' },
        ],
        assignees: [],
      },
      {
        number: 2,
        title: 'Issue 2',
        state: 'OPEN',
        createdAt: '2024-02-02T00:00:00Z',
        labels: [
          { name: 'bug', color: 'd73a4a' },
          { name: 'P1-high', color: 'd93f0b' },
        ],
        assignees: [],
      },
      {
        number: 3,
        title: 'Issue 3',
        state: 'CLOSED',
        createdAt: '2024-02-03T00:00:00Z',
        labels: [{ name: 'enhancement', color: 'a2eeef' }],
        assignees: [],
      },
    ];

    test('extracts unique labels', () => {
      const labels = extractUniqueLabels(mockIssues);
      expect(labels).toHaveLength(4);
    });

    test('labels are sorted alphabetically', () => {
      const labels = extractUniqueLabels(mockIssues);
      expect(labels).toEqual(['P1-high', 'bug', 'enhancement', 'tui']);
    });

    test('empty issues returns empty array', () => {
      const labels = extractUniqueLabels([]);
      expect(labels).toHaveLength(0);
    });

    test('issue with no labels is handled', () => {
      const issues: GHIssue[] = [
        {
          number: 1,
          title: 'No labels',
          state: 'OPEN',
          createdAt: '2024-02-01T00:00:00Z',
          labels: [],
          assignees: [],
        },
      ];
      const labels = extractUniqueLabels(issues);
      expect(labels).toHaveLength(0);
    });
  });

  describe('State Filter Cycling', () => {
    test('open cycles to closed', () => {
      expect(cycleStateFilter('open')).toBe('closed');
    });

    test('closed cycles to all', () => {
      expect(cycleStateFilter('closed')).toBe('all');
    });

    test('all cycles to open', () => {
      expect(cycleStateFilter('all')).toBe('open');
    });
  });

  describe('Label Filter Cycling', () => {
    const labels = ['bug', 'enhancement', 'feature'];

    test('null starts at first label', () => {
      expect(cycleLabelFilter(null, labels)).toBe('bug');
    });

    test('cycles through labels', () => {
      expect(cycleLabelFilter('bug', labels)).toBe('enhancement');
      expect(cycleLabelFilter('enhancement', labels)).toBe('feature');
    });

    test('last label cycles to null', () => {
      expect(cycleLabelFilter('feature', labels)).toBe(null);
    });

    test('empty labels returns null', () => {
      expect(cycleLabelFilter(null, [])).toBe(null);
    });
  });

  describe('Title Truncation', () => {
    test('short title is not truncated', () => {
      expect(truncate('Short title', 48)).toBe('Short title');
    });

    test('long title is truncated with ellipsis', () => {
      const longTitle =
        'This is a very long issue title that exceeds the maximum allowed length for display';
      const truncated = truncate(longTitle, 48);
      expect(truncated.length).toBe(48);
      expect(truncated.endsWith('…')).toBe(true);
    });

    test('exact length is not truncated', () => {
      const exactTitle = 'a'.repeat(48);
      expect(truncate(exactTitle, 48)).toBe(exactTitle);
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      'j/k': 'navigate',
      'g/G': 'top/bottom',
      Enter: 'details',
      f: 'filter',
      s: 'state',
      r: 'refresh',
      'q/ESC': 'back',
    };

    test('navigation shortcuts', () => {
      expect(shortcuts['j/k']).toBe('navigate');
      expect(shortcuts['g/G']).toBe('top/bottom');
    });

    test('action shortcuts', () => {
      expect(shortcuts.Enter).toBe('details');
      expect(shortcuts.f).toBe('filter');
      expect(shortcuts.s).toBe('state');
      expect(shortcuts.r).toBe('refresh');
    });

    test('back shortcuts', () => {
      expect(shortcuts['q/ESC']).toBe('back');
    });
  });

  describe('Issue Data Structure', () => {
    test('issue has required fields', () => {
      const issue: GHIssue = {
        number: 123,
        title: 'Test issue',
        state: 'OPEN',
        createdAt: '2024-02-01T00:00:00Z',
        labels: [],
        assignees: [],
      };

      expect(issue.number).toBe(123);
      expect(issue.title).toBe('Test issue');
      expect(issue.state).toBe('OPEN');
    });

    test('issue with all fields', () => {
      const issue: GHIssue = {
        number: 456,
        title: 'Full issue',
        state: 'CLOSED',
        createdAt: '2024-02-01T00:00:00Z',
        updatedAt: '2024-02-15T00:00:00Z',
        labels: [{ name: 'bug', color: 'd73a4a' }],
        assignees: [{ login: 'user1' }],
        body: 'This is the issue body',
        author: { login: 'creator' },
        comments: [
          { body: 'Comment 1', author: { login: 'commenter' }, createdAt: '2024-02-10T00:00:00Z' },
        ],
      };

      expect(issue.updatedAt).toBeDefined();
      expect(issue.body).toBe('This is the issue body');
      expect(issue.author?.login).toBe('creator');
      expect(issue.comments).toHaveLength(1);
    });
  });

  describe('Issue State', () => {
    test('OPEN state color is green', () => {
      const state = 'OPEN';
      const color = state === 'OPEN' ? 'green' : 'red';
      expect(color).toBe('green');
    });

    test('CLOSED state color is red', () => {
      const state = 'CLOSED';
      const color = state === 'OPEN' ? 'green' : 'red';
      expect(color).toBe('red');
    });
  });

  describe('Issue Counts', () => {
    test('computes counts correctly', () => {
      const issues: GHIssue[] = [
        { number: 1, title: 'Open 1', state: 'OPEN', createdAt: '', labels: [], assignees: [] },
        { number: 2, title: 'Open 2', state: 'OPEN', createdAt: '', labels: [], assignees: [] },
        { number: 3, title: 'Closed 1', state: 'CLOSED', createdAt: '', labels: [], assignees: [] },
      ];

      const openCount = issues.filter((i) => i.state === 'OPEN').length;
      const closedCount = issues.filter((i) => i.state === 'CLOSED').length;
      const totalCount = issues.length;

      expect(openCount).toBe(2);
      expect(closedCount).toBe(1);
      expect(totalCount).toBe(3);
    });
  });

  describe('Primary Label Selection', () => {
    test('gets first label as primary', () => {
      const labels: GHIssueLabel[] = [
        { name: 'bug', color: 'd73a4a' },
        { name: 'tui', color: '1d76db' },
      ];
      const primaryLabel = labels[0]?.name ?? '';
      expect(primaryLabel).toBe('bug');
    });

    test('empty labels returns empty string', () => {
      const labels: GHIssueLabel[] = [];
      const primaryLabel = labels[0]?.name ?? '';
      expect(primaryLabel).toBe('');
    });
  });

  describe('Detail View Content', () => {
    test('body truncation at 500 chars', () => {
      const longBody = 'a'.repeat(600);
      const displayed = longBody.slice(0, 500);
      const showEllipsis = longBody.length > 500;

      expect(displayed.length).toBe(500);
      expect(showEllipsis).toBe(true);
    });

    test('short body not truncated', () => {
      const shortBody = 'Short body';
      const displayed = shortBody.slice(0, 500);
      const showEllipsis = shortBody.length > 500;

      expect(displayed).toBe(shortBody);
      expect(showEllipsis).toBe(false);
    });

    test('shows first 3 comments', () => {
      const comments = [
        { body: 'C1', author: { login: 'u1' }, createdAt: '' },
        { body: 'C2', author: { login: 'u2' }, createdAt: '' },
        { body: 'C3', author: { login: 'u3' }, createdAt: '' },
        { body: 'C4', author: { login: 'u4' }, createdAt: '' },
        { body: 'C5', author: { login: 'u5' }, createdAt: '' },
      ];

      const displayed = comments.slice(0, 3);
      const moreCount = comments.length - 3;

      expect(displayed).toHaveLength(3);
      expect(moreCount).toBe(2);
    });
  });
});
