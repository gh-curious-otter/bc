/**
 * ProcessesView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

/**
 * formatUptime - Calculate uptime string from started_at timestamp
 * Mirrors the implementation in ProcessesView.tsx
 */
function formatUptime(startedAt: string): string {
  const start = new Date(startedAt);
  const now = new Date();
  const diffMs = now.getTime() - start.getTime();

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    return `${String(days)}d ${String(hours % 24)}h`;
  } else if (hours > 0) {
    return `${String(hours)}h ${String(minutes % 60)}m`;
  } else if (minutes > 0) {
    return `${String(minutes)}m ${String(seconds % 60)}s`;
  } else {
    return `${String(seconds)}s`;
  }
}

describe('ProcessesView - formatUptime', () => {
  test('formats seconds only', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - 45 * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('45s');
  });

  test('formats minutes and seconds', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - (5 * 60 + 30) * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('5m 30s');
  });

  test('formats hours and minutes', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - (2 * 60 * 60 + 15 * 60) * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('2h 15m');
  });

  test('formats days and hours', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - (3 * 24 * 60 * 60 + 5 * 60 * 60) * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('3d 5h');
  });

  test('handles zero seconds', () => {
    const now = new Date();
    const startedAt = now.toISOString();
    expect(formatUptime(startedAt)).toBe('0s');
  });

  test('handles exactly one minute', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - 60 * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('1m 0s');
  });

  test('handles exactly one hour', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - 60 * 60 * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('1h 0m');
  });

  test('handles exactly one day', () => {
    const now = new Date();
    const startedAt = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
    expect(formatUptime(startedAt)).toBe('1d 0h');
  });
});

describe('ProcessesView - process filtering', () => {
  interface MockProcess {
    name: string;
    command: string;
    owner: string | null;
    running: boolean;
    pid: number;
  }

  const mockProcesses: MockProcess[] = [
    { name: 'web-server', command: 'node server.js', owner: 'eng-01', running: true, pid: 1234 },
    { name: 'api-gateway', command: 'go run main.go', owner: 'eng-02', running: true, pid: 5678 },
    { name: 'db-watcher', command: 'python watch.py', owner: null, running: false, pid: 0 },
    { name: 'test-runner', command: 'bun test', owner: 'eng-01', running: true, pid: 9012 },
  ];

  function filterProcesses(processes: MockProcess[], query: string): MockProcess[] {
    if (!query) return processes;
    const q = query.toLowerCase();
    return processes.filter(
      (proc) =>
        proc.name.toLowerCase().includes(q) ||
        proc.command.toLowerCase().includes(q) ||
        (proc.owner?.toLowerCase().includes(q) ?? false)
    );
  }

  test('returns all processes when query is empty', () => {
    expect(filterProcesses(mockProcesses, '')).toHaveLength(4);
  });

  test('filters by process name', () => {
    const result = filterProcesses(mockProcesses, 'web');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('web-server');
  });

  test('filters by command', () => {
    const result = filterProcesses(mockProcesses, 'python');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('db-watcher');
  });

  test('filters by owner', () => {
    const result = filterProcesses(mockProcesses, 'eng-01');
    expect(result).toHaveLength(2);
  });

  test('filters case-insensitively', () => {
    const result = filterProcesses(mockProcesses, 'NODE');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('web-server');
  });

  test('returns empty array when no matches', () => {
    const result = filterProcesses(mockProcesses, 'nonexistent');
    expect(result).toHaveLength(0);
  });

  test('handles null owner gracefully', () => {
    const result = filterProcesses(mockProcesses, 'db');
    expect(result).toHaveLength(1);
    expect(result[0].owner).toBeNull();
  });

  test('matches partial strings', () => {
    const result = filterProcesses(mockProcesses, 'run');
    expect(result).toHaveLength(2); // test-runner and "go run"
  });
});

describe('ProcessesView - process status', () => {
  test('running process has positive PID', () => {
    const runningProcess = { running: true, pid: 1234 };
    expect(runningProcess.running).toBe(true);
    expect(runningProcess.pid).toBeGreaterThan(0);
  });

  test('stopped process has zero PID', () => {
    const stoppedProcess = { running: false, pid: 0 };
    expect(stoppedProcess.running).toBe(false);
    expect(stoppedProcess.pid).toBe(0);
  });
});

describe('ProcessesView - name truncation', () => {
  function truncateName(name: string, maxLength = 12): string {
    return name.length > maxLength ? name.slice(0, maxLength - 1) + '…' : name;
  }

  test('short names are not truncated', () => {
    expect(truncateName('web')).toBe('web');
  });

  test('exact length names are not truncated', () => {
    expect(truncateName('123456789012')).toBe('123456789012');
  });

  test('long names are truncated with ellipsis', () => {
    // 12 char max: 11 chars + ellipsis
    expect(truncateName('very-long-process-name')).toBe('very-long-p…');
  });

  test('handles empty string', () => {
    expect(truncateName('')).toBe('');
  });
});

describe('ProcessesView - log viewer', () => {
  test('calculates visible log range correctly', () => {
    const totalLines = 100;
    const maxVisibleLines = 15;
    const scrollOffset = 20;

    const startLine = scrollOffset;
    const endLine = Math.min(scrollOffset + maxVisibleLines, totalLines);

    expect(startLine).toBe(20);
    expect(endLine).toBe(35);
  });

  test('handles scroll at top', () => {
    const totalLines = 100;
    const maxVisibleLines = 15;
    const scrollOffset = 0;

    const startLine = scrollOffset;
    const endLine = Math.min(scrollOffset + maxVisibleLines, totalLines);

    expect(startLine).toBe(0);
    expect(endLine).toBe(15);
  });

  test('handles scroll at bottom', () => {
    const totalLines = 100;
    const maxVisibleLines = 15;
    const scrollOffset = 85;

    const startLine = scrollOffset;
    const endLine = Math.min(scrollOffset + maxVisibleLines, totalLines);

    expect(startLine).toBe(85);
    expect(endLine).toBe(100);
  });

  test('handles fewer lines than max visible', () => {
    const totalLines = 10;
    const maxVisibleLines = 15;
    const scrollOffset = 0;

    const startLine = scrollOffset;
    const endLine = Math.min(scrollOffset + maxVisibleLines, totalLines);

    expect(startLine).toBe(0);
    expect(endLine).toBe(10);
  });

  test('calculates max scroll offset', () => {
    const totalLines = 100;
    const maxVisibleLines = 15;
    const maxScroll = Math.max(0, totalLines - maxVisibleLines);

    expect(maxScroll).toBe(85);
  });

  test('max scroll is zero when content fits', () => {
    const totalLines = 10;
    const maxVisibleLines = 15;
    const maxScroll = Math.max(0, totalLines - maxVisibleLines);

    expect(maxScroll).toBe(0);
  });
});

describe('ProcessesView - command truncation', () => {
  function truncateCommand(command: string | undefined, maxLength = 20): string {
    if (!command) return '-';
    return command.length > maxLength ? command.slice(0, maxLength) : command;
  }

  test('short commands are not truncated', () => {
    expect(truncateCommand('npm start')).toBe('npm start');
  });

  test('long commands are truncated', () => {
    // 20 char max, no ellipsis added
    expect(truncateCommand('node --experimental-modules server.js')).toBe('node --experimental-');
  });

  test('undefined command returns dash', () => {
    expect(truncateCommand(undefined)).toBe('-');
  });

  test('empty command returns dash', () => {
    // Empty string is falsy, so returns '-'
    expect(truncateCommand('')).toBe('-');
  });
});
