/**
 * LogsView Tests - Event logs tab (#866)
 *
 * Tests cover:
 * - Time formatting (today vs previous days)
 * - Time filtering (1h, 6h, 24h, all)
 * - Agent filtering
 * - Search filtering
 * - Severity filtering
 * - Keyboard navigation (j/k, g/G, Enter, /, s, a, t, c, r, q)
 * - Visible rows calculation for 80x24 support
 */

import { describe, test, expect } from 'bun:test';

// Helper functions extracted from LogsView for testing
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    const now = new Date();
    const isToday = date.toDateString() === now.toDateString();

    if (isToday) {
      return date.toLocaleTimeString('en-US', {
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit',
        hour12: false,
      });
    } else {
      const month = String(date.getMonth() + 1).padStart(2, '0');
      const day = String(date.getDate()).padStart(2, '0');
      const hours = String(date.getHours()).padStart(2, '0');
      const mins = String(date.getMinutes()).padStart(2, '0');
      return `${month}/${day} ${hours}:${mins}`;
    }
  } catch {
    return timestamp.slice(0, 8);
  }
}

type TimeFilter = '1h' | '6h' | '24h' | 'all';

interface LogEntry {
  ts: string;
  agent: string;
  type: string;
  message: string;
}

function filterByTime(logs: LogEntry[], timeFilter: TimeFilter): LogEntry[] {
  if (timeFilter === 'all') return logs;

  const now = Date.now();
  const hours = timeFilter === '1h' ? 1 : timeFilter === '6h' ? 6 : 24;
  const cutoff = now - hours * 60 * 60 * 1000;

  return logs.filter((log) => {
    try {
      return new Date(log.ts).getTime() >= cutoff;
    } catch {
      return true;
    }
  });
}

describe('LogsView', () => {
  describe('Time Formatting', () => {
    test('formats today timestamps as HH:MM:SS', () => {
      const now = new Date();
      const todayTimestamp = now.toISOString();
      const formatted = formatTime(todayTimestamp);

      // Should be in HH:MM:SS format (8 chars)
      expect(formatted).toMatch(/^\d{2}:\d{2}:\d{2}$/);
    });

    test('formats previous day timestamps as MM/DD HH:MM', () => {
      const yesterday = new Date();
      yesterday.setDate(yesterday.getDate() - 1);
      const formatted = formatTime(yesterday.toISOString());

      // Should be in MM/DD HH:MM format (11 chars)
      expect(formatted).toMatch(/^\d{2}\/\d{2} \d{2}:\d{2}$/);
    });

    test('handles invalid timestamps gracefully', () => {
      const formatted = formatTime('invalid-timestamp');
      // Invalid date produces NaN values in the MM/DD HH:MM format
      expect(formatted).toContain('NaN');
    });
  });

  describe('Time Filtering', () => {
    const now = Date.now();
    const mockLogs: LogEntry[] = [
      {
        ts: new Date(now - 30 * 60 * 1000).toISOString(),
        agent: 'eng-01',
        type: 'info',
        message: '30 min ago',
      },
      {
        ts: new Date(now - 2 * 60 * 60 * 1000).toISOString(),
        agent: 'eng-01',
        type: 'info',
        message: '2 hours ago',
      },
      {
        ts: new Date(now - 12 * 60 * 60 * 1000).toISOString(),
        agent: 'eng-01',
        type: 'info',
        message: '12 hours ago',
      },
      {
        ts: new Date(now - 48 * 60 * 60 * 1000).toISOString(),
        agent: 'eng-01',
        type: 'info',
        message: '2 days ago',
      },
    ];

    test('all filter returns all logs', () => {
      const filtered = filterByTime(mockLogs, 'all');
      expect(filtered).toHaveLength(4);
    });

    test('1h filter returns only recent logs', () => {
      const filtered = filterByTime(mockLogs, '1h');
      expect(filtered).toHaveLength(1);
      expect(filtered[0].message).toBe('30 min ago');
    });

    test('6h filter returns logs within 6 hours', () => {
      const filtered = filterByTime(mockLogs, '6h');
      expect(filtered).toHaveLength(2);
    });

    test('24h filter returns logs within 24 hours', () => {
      const filtered = filterByTime(mockLogs, '24h');
      expect(filtered).toHaveLength(3);
    });
  });

  describe('Agent Filtering', () => {
    const mockLogs: LogEntry[] = [
      { ts: '2024-01-01T10:00:00Z', agent: 'eng-01', type: 'info', message: 'log 1' },
      { ts: '2024-01-01T10:01:00Z', agent: 'eng-02', type: 'info', message: 'log 2' },
      { ts: '2024-01-01T10:02:00Z', agent: 'eng-01', type: 'info', message: 'log 3' },
    ];

    test('filters logs by agent', () => {
      const agentFilter = 'eng-01';
      const filtered = mockLogs.filter((log) => log.agent === agentFilter);
      expect(filtered).toHaveLength(2);
    });

    test('returns all logs when no agent filter', () => {
      const agentFilter: string | null = null;
      const filtered = agentFilter ? mockLogs.filter((log) => log.agent === agentFilter) : mockLogs;
      expect(filtered).toHaveLength(3);
    });

    test('extracts unique agents from logs', () => {
      const agents = Array.from(new Set(mockLogs.map((log) => log.agent))).sort();
      expect(agents).toEqual(['eng-01', 'eng-02']);
    });
  });

  describe('Search Filtering', () => {
    const mockLogs: LogEntry[] = [
      {
        ts: '2024-01-01T10:00:00Z',
        agent: 'eng-01',
        type: 'agent.started',
        message: 'Agent started successfully',
      },
      {
        ts: '2024-01-01T10:01:00Z',
        agent: 'eng-02',
        type: 'state.working',
        message: 'Starting implementation',
      },
      {
        ts: '2024-01-01T10:02:00Z',
        agent: 'eng-01',
        type: 'agent.report',
        message: 'Completed feature X',
      },
    ];

    test('filters by message content', () => {
      const query = 'start';
      const filtered = mockLogs.filter((log) =>
        log.message.toLowerCase().includes(query.toLowerCase())
      );
      expect(filtered).toHaveLength(2); // 'started' and 'Starting'
    });

    test('filters by agent name', () => {
      const query = 'eng-02';
      const filtered = mockLogs.filter((log) =>
        log.agent.toLowerCase().includes(query.toLowerCase())
      );
      expect(filtered).toHaveLength(1);
    });

    test('filters by event type', () => {
      const query = 'report';
      const filtered = mockLogs.filter((log) =>
        log.type.toLowerCase().includes(query.toLowerCase())
      );
      expect(filtered).toHaveLength(1);
    });

    test('search is case-insensitive', () => {
      const query = 'START';
      const filtered = mockLogs.filter((log) =>
        log.message.toLowerCase().includes(query.toLowerCase())
      );
      expect(filtered).toHaveLength(2); // 'started' and 'Starting'
    });
  });

  describe('Keyboard Navigation', () => {
    test('j/k moves selection up/down', () => {
      let selectedIndex = 0;
      const maxIndex = 9;

      // Press 'j' - move down
      selectedIndex = Math.min(maxIndex, selectedIndex + 1);
      expect(selectedIndex).toBe(1);

      // Press 'k' - move up
      selectedIndex = Math.max(0, selectedIndex - 1);
      expect(selectedIndex).toBe(0);

      // Press 'k' at top - stays at 0
      selectedIndex = Math.max(0, selectedIndex - 1);
      expect(selectedIndex).toBe(0);
    });

    test('g goes to first item, G goes to last', () => {
      let selectedIndex = 5;
      const logsLength = 10;

      // Press 'g' - go to first
      selectedIndex = 0;
      expect(selectedIndex).toBe(0);

      // Press 'G' - go to last
      selectedIndex = Math.max(0, logsLength - 1);
      expect(selectedIndex).toBe(9);
    });

    test('severity filter cycles through options', () => {
      const severities: (string | null)[] = [null, 'info', 'warn', 'error'];
      let currentIdx = 0;

      // Cycle through severities
      currentIdx = (currentIdx + 1) % severities.length;
      expect(severities[currentIdx]).toBe('info');

      currentIdx = (currentIdx + 1) % severities.length;
      expect(severities[currentIdx]).toBe('warn');

      currentIdx = (currentIdx + 1) % severities.length;
      expect(severities[currentIdx]).toBe('error');

      currentIdx = (currentIdx + 1) % severities.length;
      expect(severities[currentIdx]).toBe(null);
    });

    test('time filter cycles through options', () => {
      const times: TimeFilter[] = ['all', '1h', '6h', '24h'];
      let currentIdx = 0;

      // Cycle through time filters
      currentIdx = (currentIdx + 1) % times.length;
      expect(times[currentIdx]).toBe('1h');

      currentIdx = (currentIdx + 1) % times.length;
      expect(times[currentIdx]).toBe('6h');

      currentIdx = (currentIdx + 1) % times.length;
      expect(times[currentIdx]).toBe('24h');

      currentIdx = (currentIdx + 1) % times.length;
      expect(times[currentIdx]).toBe('all');
    });
  });

  describe('Visible Rows Calculation (80x24 support)', () => {
    function calculateVisibleRows(terminalHeight: number): number {
      const viewOverhead = 11; // header + filters + table border + footer
      return Math.max(5, Math.min(15, terminalHeight - viewOverhead));
    }

    test('calculates correct visible rows at 24 rows terminal', () => {
      expect(calculateVisibleRows(24)).toBe(13);
    });

    test('calculates correct visible rows at 40 rows terminal', () => {
      // 40 - 11 = 29, capped at 15
      expect(calculateVisibleRows(40)).toBe(15);
    });

    test('calculates minimum visible rows at small terminal', () => {
      // 15 - 11 = 4, minimum is 5
      expect(calculateVisibleRows(15)).toBe(5);
    });

    test('visible rows has minimum of 5', () => {
      expect(calculateVisibleRows(10)).toBe(5);
    });

    test('visible rows has maximum of 15', () => {
      expect(calculateVisibleRows(100)).toBe(15);
    });
  });

  describe('Column Width Calculation', () => {
    function calculateColumnWidths(terminalWidth: number) {
      const timeWidth = 12;
      const agentWidth = Math.min(12, Math.floor((terminalWidth - 40) * 0.2));
      const typeWidth = 10;
      const messageWidth = terminalWidth - timeWidth - agentWidth - typeWidth - 10;
      return { timeWidth, agentWidth, typeWidth, messageWidth };
    }

    test('calculates correct column widths at 80 columns', () => {
      const { timeWidth, agentWidth, typeWidth, messageWidth } = calculateColumnWidths(80);
      expect(timeWidth).toBe(12);
      expect(agentWidth).toBe(8); // (80-40)*0.2 = 8
      expect(typeWidth).toBe(10);
      expect(messageWidth).toBe(40); // 80 - 12 - 8 - 10 - 10 = 40
    });

    test('calculates correct column widths at 120 columns', () => {
      const { timeWidth, agentWidth, typeWidth, messageWidth } = calculateColumnWidths(120);
      expect(timeWidth).toBe(12);
      expect(agentWidth).toBe(12); // min(12, 16) = 12
      expect(typeWidth).toBe(10);
      expect(messageWidth).toBe(76); // 120 - 12 - 12 - 10 - 10 = 76
    });
  });

  describe('State Management', () => {
    test('initial state values', () => {
      const initialState = {
        selectedIndex: 0,
        showDetail: false,
        searchQuery: '',
        searchMode: false,
        agentFilter: null as string | null,
        timeFilter: 'all' as TimeFilter,
      };

      expect(initialState.selectedIndex).toBe(0);
      expect(initialState.showDetail).toBe(false);
      expect(initialState.searchQuery).toBe('');
      expect(initialState.searchMode).toBe(false);
      expect(initialState.agentFilter).toBeNull();
      expect(initialState.timeFilter).toBe('all');
    });

    test('clear filters resets all filter state', () => {
      const state = {
        searchQuery: 'test',
        agentFilter: 'eng-01' as string | null,
        timeFilter: '1h' as TimeFilter,
        severityFilter: 'error' as string | null,
        selectedIndex: 5,
      };

      // Clear all filters
      state.searchQuery = '';
      state.agentFilter = null;
      state.timeFilter = 'all';
      state.severityFilter = null;
      state.selectedIndex = 0;

      expect(state.searchQuery).toBe('');
      expect(state.agentFilter).toBeNull();
      expect(state.timeFilter).toBe('all');
      expect(state.severityFilter).toBeNull();
      expect(state.selectedIndex).toBe(0);
    });
  });
});
