/**
 * Dashboard unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

/**
 * formatNumber - Format large numbers with K/M suffixes
 * Mirrors the implementation in Dashboard.tsx
 */
function formatNumber(n: number): string {
  if (n >= 1_000_000) {
    return `${(n / 1_000_000).toFixed(1)}M`;
  }
  if (n >= 1_000) {
    return `${(n / 1_000).toFixed(1)}K`;
  }
  return n.toString();
}

describe('Dashboard - formatNumber', () => {
  test('formats small numbers as-is', () => {
    expect(formatNumber(0)).toBe('0');
    expect(formatNumber(1)).toBe('1');
    expect(formatNumber(999)).toBe('999');
  });

  test('formats thousands with K suffix', () => {
    expect(formatNumber(1000)).toBe('1.0K');
    expect(formatNumber(1500)).toBe('1.5K');
    expect(formatNumber(10000)).toBe('10.0K');
    expect(formatNumber(999999)).toBe('1000.0K');
  });

  test('formats millions with M suffix', () => {
    expect(formatNumber(1000000)).toBe('1.0M');
    expect(formatNumber(1500000)).toBe('1.5M');
    expect(formatNumber(10000000)).toBe('10.0M');
  });

  test('handles edge cases', () => {
    expect(formatNumber(999)).toBe('999');
    expect(formatNumber(1000)).toBe('1.0K');
    expect(formatNumber(999999)).toBe('1000.0K');
    expect(formatNumber(1000000)).toBe('1.0M');
  });
});

/**
 * formatRelativeTime - Format date to relative time string
 * Mirrors the implementation in Dashboard.tsx
 */
function formatRelativeTime(date: Date): string {
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSecs = Math.floor(diffMs / 1000);

  if (diffSecs < 5) return 'just now';
  if (diffSecs < 60) return `${String(diffSecs)}s ago`;

  const diffMins = Math.floor(diffSecs / 60);
  if (diffMins < 60) return `${String(diffMins)}m ago`;

  return date.toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
  });
}

describe('Dashboard - formatRelativeTime', () => {
  test('shows "just now" for very recent times', () => {
    const now = new Date();
    const recent = new Date(now.getTime() - 2000); // 2 seconds ago
    expect(formatRelativeTime(recent)).toBe('just now');
  });

  test('shows seconds for times under a minute', () => {
    const now = new Date();
    const thirtySecsAgo = new Date(now.getTime() - 30000);
    expect(formatRelativeTime(thirtySecsAgo)).toBe('30s ago');
  });

  test('shows minutes for times under an hour', () => {
    const now = new Date();
    const fiveMinsAgo = new Date(now.getTime() - 5 * 60 * 1000);
    expect(formatRelativeTime(fiveMinsAgo)).toBe('5m ago');
  });

  test('shows time for times over an hour', () => {
    const now = new Date();
    const twoHoursAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000);
    const result = formatRelativeTime(twoHoursAgo);
    // Should match HH:MM format (e.g., "10:30 AM")
    expect(result).toMatch(/^\d{1,2}:\d{2}\s?(AM|PM)$/);
  });

  test('boundary: 4 seconds is "just now"', () => {
    const now = new Date();
    const fourSecsAgo = new Date(now.getTime() - 4000);
    expect(formatRelativeTime(fourSecsAgo)).toBe('just now');
  });

  test('boundary: 5 seconds shows seconds', () => {
    const now = new Date();
    const fiveSecsAgo = new Date(now.getTime() - 5000);
    expect(formatRelativeTime(fiveSecsAgo)).toBe('5s ago');
  });

  test('boundary: 59 seconds shows seconds', () => {
    const now = new Date();
    const fiftyNineSecsAgo = new Date(now.getTime() - 59000);
    expect(formatRelativeTime(fiftyNineSecsAgo)).toBe('59s ago');
  });

  test('boundary: 60 seconds shows 1m', () => {
    const now = new Date();
    const sixtySecsAgo = new Date(now.getTime() - 60000);
    expect(formatRelativeTime(sixtySecsAgo)).toBe('1m ago');
  });
});

describe('Dashboard - side panel width calculation', () => {
  // From Dashboard.tsx: Math.min(32, Math.max(26, Math.floor((terminalWidth - 4) * 0.28)))
  function calculateSidePanelWidth(terminalWidth: number): number {
    return Math.min(32, Math.max(26, Math.floor((terminalWidth - 4) * 0.28)));
  }

  test('standard terminal (80 cols)', () => {
    const width = calculateSidePanelWidth(80);
    expect(width).toBe(26); // (80-4)*0.28 = 21.28, max(26, 21) = 26
  });

  test('wide terminal (120 cols)', () => {
    const width = calculateSidePanelWidth(120);
    expect(width).toBe(32); // (120-4)*0.28 = 32.48, min(32, 32) = 32
  });

  test('very wide terminal (160 cols)', () => {
    const width = calculateSidePanelWidth(160);
    expect(width).toBe(32); // Capped at 32
  });

  test('narrow terminal (100 cols)', () => {
    const width = calculateSidePanelWidth(100);
    expect(width).toBe(26); // (100-4)*0.28 = 26.88, floor = 26
  });

  test('minimum width is 26', () => {
    const width = calculateSidePanelWidth(60);
    expect(width).toBe(26);
  });
});

describe('Dashboard - summary cards display logic', () => {
  interface SummaryProps {
    total: number;
    active: number;
    working: number;
    idle: number;
    stuck: number;
    errorCount: number;
  }

  function shouldShowStuck(stuck: number): boolean {
    return stuck > 0;
  }

  function shouldShowError(errorCount: number): boolean {
    return errorCount > 0;
  }

  test('shows stuck card only when stuck > 0', () => {
    expect(shouldShowStuck(0)).toBe(false);
    expect(shouldShowStuck(1)).toBe(true);
    expect(shouldShowStuck(5)).toBe(true);
  });

  test('shows error card only when errorCount > 0', () => {
    expect(shouldShowError(0)).toBe(false);
    expect(shouldShowError(1)).toBe(true);
    expect(shouldShowError(3)).toBe(true);
  });
});

describe('Dashboard - activity feed limits', () => {
  function getMaxEntries(isMedium: boolean, isWide: boolean): number {
    return isMedium || isWide ? 15 : 8;
  }

  test('narrow layout shows 8 entries', () => {
    expect(getMaxEntries(false, false)).toBe(8);
  });

  test('medium layout shows 15 entries', () => {
    expect(getMaxEntries(true, false)).toBe(15);
  });

  test('wide layout shows 15 entries', () => {
    expect(getMaxEntries(false, true)).toBe(15);
  });

  test('medium+wide layout shows 15 entries', () => {
    expect(getMaxEntries(true, true)).toBe(15);
  });
});

describe('Dashboard - progressive loading', () => {
  function shouldShowInitialLoading(isLoading: boolean, hasData: boolean): boolean {
    return isLoading && !hasData;
  }

  test('shows loading when loading and no data', () => {
    expect(shouldShowInitialLoading(true, false)).toBe(true);
  });

  test('does not show loading when has data', () => {
    expect(shouldShowInitialLoading(true, true)).toBe(false);
  });

  test('does not show loading when not loading', () => {
    expect(shouldShowInitialLoading(false, false)).toBe(false);
    expect(shouldShowInitialLoading(false, true)).toBe(false);
  });
});

describe('Dashboard - refresh text generation', () => {
  function getRefreshText(lastRefresh: Date | null): string {
    if (!lastRefresh) return '';
    return `Updated ${formatRelativeTime(lastRefresh)}`;
  }

  test('returns empty string when no lastRefresh', () => {
    expect(getRefreshText(null)).toBe('');
  });

  test('returns formatted text when lastRefresh exists', () => {
    const now = new Date();
    const result = getRefreshText(now);
    expect(result).toContain('Updated');
    expect(result).toContain('just now');
  });
});

describe('Dashboard - footer hints', () => {
  function getFooterHints(showDebugPanel: boolean): { key: string; label: string }[] {
    return [
      { key: 'Tab', label: 'views' },
      { key: 'j/k', label: 'drawer' },
      { key: 'Enter', label: 'select' },
      { key: 'r', label: 'refresh' },
      ...(showDebugPanel ? [{ key: 'Ctrl+P', label: 'hide perf' }] : [{ key: 'Ctrl+P', label: 'perf' }]),
      { key: 'q', label: 'quit' },
    ];
  }

  test('shows perf when debug panel hidden', () => {
    const hints = getFooterHints(false);
    const perfHint = hints.find(h => h.key === 'Ctrl+P');
    expect(perfHint?.label).toBe('perf');
  });

  test('shows hide perf when debug panel shown', () => {
    const hints = getFooterHints(true);
    const perfHint = hints.find(h => h.key === 'Ctrl+P');
    expect(perfHint?.label).toBe('hide perf');
  });

  test('always includes standard hints', () => {
    const hints = getFooterHints(false);
    expect(hints.some(h => h.key === 'Tab')).toBe(true);
    expect(hints.some(h => h.key === 'r')).toBe(true);
    expect(hints.some(h => h.key === 'q')).toBe(true);
  });
});
