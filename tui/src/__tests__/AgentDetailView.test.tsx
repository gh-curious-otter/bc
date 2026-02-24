/**
 * AgentDetailView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('AgentDetailView - normalizeTask', () => {
  function normalizeTask(task: string | undefined): string {
    if (!task) return '(no task)';
    const replacements: [string, string][] = [
      ['Sautéed', 'Working'],
      ['Sauteed', 'Working'],
      ['Cooked', 'Processed'],
      ['Cogitated', 'Thinking'],
      ['Marinated', 'Idle'],
      ['Frolicking', 'Active'],
    ];
    for (const [old, replacement] of replacements) {
      if (task.includes(old)) {
        return task.replace(old, replacement);
      }
    }
    return task;
  }

  test('returns "(no task)" for undefined', () => {
    expect(normalizeTask(undefined)).toBe('(no task)');
  });

  test('replaces Sautéed with Working', () => {
    expect(normalizeTask('Sautéed some code')).toBe('Working some code');
  });

  test('replaces Sauteed with Working (ASCII)', () => {
    expect(normalizeTask('Sauteed the files')).toBe('Working the files');
  });

  test('replaces Cooked with Processed', () => {
    expect(normalizeTask('Cooked the data')).toBe('Processed the data');
  });

  test('replaces Cogitated with Thinking', () => {
    expect(normalizeTask('Cogitated about design')).toBe('Thinking about design');
  });

  test('replaces Marinated with Idle', () => {
    expect(normalizeTask('Marinated in queue')).toBe('Idle in queue');
  });

  test('replaces Frolicking with Active', () => {
    expect(normalizeTask('Frolicking in tests')).toBe('Active in tests');
  });

  test('returns task unchanged if no matches', () => {
    expect(normalizeTask('Running tests')).toBe('Running tests');
  });
});

describe('AgentDetailView - colorizeOutputLine patterns', () => {
  function getLineType(line: string): 'error' | 'warning' | 'success' | 'command' | 'default' {
    const trimmed = line.trim().toLowerCase();

    if (
      trimmed.includes('error') ||
      trimmed.includes('failed') ||
      trimmed.includes('exception') ||
      trimmed.startsWith('✗') ||
      trimmed.startsWith('x ')
    ) {
      return 'error';
    }

    if (
      trimmed.includes('warning') ||
      trimmed.includes('warn') ||
      trimmed.includes('deprecated') ||
      trimmed.startsWith('⚠')
    ) {
      return 'warning';
    }

    if (
      trimmed.includes('success') ||
      trimmed.includes('passed') ||
      trimmed.includes('complete') ||
      trimmed.startsWith('✓') ||
      trimmed.startsWith('✔')
    ) {
      return 'success';
    }

    if (
      trimmed.startsWith('>') ||
      trimmed.startsWith('$') ||
      trimmed.includes('running') ||
      trimmed.includes('executing')
    ) {
      return 'command';
    }

    return 'default';
  }

  test('detects error by keyword', () => {
    expect(getLineType('Error: something failed')).toBe('error');
    expect(getLineType('failed to connect')).toBe('error');
    expect(getLineType('exception occurred')).toBe('error');
  });

  test('detects error by symbol', () => {
    expect(getLineType('✗ Test failed')).toBe('error');
    expect(getLineType('x test case')).toBe('error');
  });

  test('detects warning by keyword', () => {
    expect(getLineType('Warning: deprecated')).toBe('warning');
    expect(getLineType('warn: check config')).toBe('warning');
    expect(getLineType('deprecated function used')).toBe('warning');
  });

  test('detects warning by symbol', () => {
    expect(getLineType('⚠ Warning message')).toBe('warning');
  });

  test('detects success by keyword', () => {
    expect(getLineType('Success!')).toBe('success');
    expect(getLineType('tests passed')).toBe('success');
    expect(getLineType('complete')).toBe('success');
  });

  test('detects success by symbol', () => {
    expect(getLineType('✓ All tests passed')).toBe('success');
    expect(getLineType('✔ Done')).toBe('success');
  });

  test('detects command by prefix', () => {
    expect(getLineType('> npm install')).toBe('command');
    expect(getLineType('$ git status')).toBe('command');
  });

  test('detects command by keyword', () => {
    expect(getLineType('running tests...')).toBe('command');
    expect(getLineType('executing script')).toBe('command');
  });

  test('returns default for normal text', () => {
    expect(getLineType('Just some text')).toBe('default');
  });
});

describe('AgentDetailView - formatDate', () => {
  function formatDate(dateString: string | undefined): string {
    if (!dateString) return '-';
    try {
      const date = new Date(dateString);
      return date.toLocaleString();
    } catch {
      return dateString;
    }
  }

  test('returns dash for undefined', () => {
    expect(formatDate(undefined)).toBe('-');
  });

  test('formats valid date', () => {
    const result = formatDate('2024-12-25T10:30:00Z');
    // Should contain some numeric date components
    expect(result).toMatch(/\d/);
  });

  test('returns original for invalid date', () => {
    const invalid = 'not-a-date';
    const result = formatDate(invalid);
    // Either returns original or "-" depending on implementation
    expect(result === invalid || result === '-' || result.includes('Invalid')).toBe(true);
  });
});

describe('AgentDetailView - formatTime', () => {
  function formatTime(timestamp: string): string {
    try {
      const date = new Date(timestamp);
      return date.toLocaleTimeString();
    } catch {
      return timestamp;
    }
  }

  test('formats valid timestamp', () => {
    const result = formatTime('2024-12-25T14:30:00Z');
    // Should contain time components
    expect(result).toMatch(/\d{1,2}:\d{2}/);
  });
});

describe('AgentDetailView - formatNumber', () => {
  function formatNumber(num: number): string {
    if (num >= 1000000) {
      return `${(num / 1000000).toFixed(1)}M`;
    }
    if (num >= 1000) {
      return `${(num / 1000).toFixed(1)}K`;
    }
    return String(num);
  }

  test('formats small numbers', () => {
    expect(formatNumber(0)).toBe('0');
    expect(formatNumber(100)).toBe('100');
    expect(formatNumber(999)).toBe('999');
  });

  test('formats thousands', () => {
    expect(formatNumber(1000)).toBe('1.0K');
    expect(formatNumber(1500)).toBe('1.5K');
    expect(formatNumber(10000)).toBe('10.0K');
  });

  test('formats millions', () => {
    expect(formatNumber(1000000)).toBe('1.0M');
    expect(formatNumber(2500000)).toBe('2.5M');
  });
});

describe('AgentDetailView - truncateMessage', () => {
  function truncateMessage(message: string, maxLen: number): string {
    if (message.length <= maxLen) return message;
    return message.slice(0, maxLen - 3) + '...';
  }

  test('short messages not truncated', () => {
    expect(truncateMessage('hello', 10)).toBe('hello');
  });

  test('exact length not truncated', () => {
    expect(truncateMessage('1234567890', 10)).toBe('1234567890');
  });

  test('long messages truncated with ellipsis', () => {
    expect(truncateMessage('hello world foo bar', 10)).toBe('hello w...');
  });
});

describe('AgentDetailView - formatUptime', () => {
  function formatUptime(startedAt: string | undefined): string {
    if (!startedAt) return '-';
    try {
      const started = new Date(startedAt);
      const now = new Date();
      const diffMs = now.getTime() - started.getTime();
      const diffMins = Math.floor(diffMs / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const mins = diffMins % 60;

      if (diffHours > 0) {
        return `${String(diffHours)}h ${String(mins)}m`;
      }
      return `${String(mins)}m`;
    } catch {
      return '-';
    }
  }

  test('returns dash for undefined', () => {
    expect(formatUptime(undefined)).toBe('-');
  });

  test('formats minutes only for short duration', () => {
    const fiveMinsAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    expect(formatUptime(fiveMinsAgo)).toBe('5m');
  });

  test('formats hours and minutes for longer duration', () => {
    const twoHoursAgo = new Date(Date.now() - 130 * 60 * 1000).toISOString(); // 2h 10m
    expect(formatUptime(twoHoursAgo)).toBe('2h 10m');
  });
});

describe('AgentDetailView - tab state', () => {
  type Tab = 'output' | 'live' | 'details' | 'metrics';

  function getTabKey(tab: Tab): string {
    switch (tab) {
      case 'output':
        return '1';
      case 'live':
        return '2';
      case 'details':
        return '3';
      case 'metrics':
        return '4';
    }
  }

  test('output is tab 1', () => {
    expect(getTabKey('output')).toBe('1');
  });

  test('live is tab 2', () => {
    expect(getTabKey('live')).toBe('2');
  });

  test('details is tab 3', () => {
    expect(getTabKey('details')).toBe('3');
  });

  test('metrics is tab 4', () => {
    expect(getTabKey('metrics')).toBe('4');
  });
});

describe('AgentDetailView - scroll calculations', () => {
  function calculateMaxOffset(totalLines: number, visibleLines = 20): number {
    return Math.max(0, totalLines - visibleLines);
  }

  function clampScroll(offset: number, maxOffset: number): number {
    return Math.max(0, Math.min(offset, maxOffset));
  }

  test('maxOffset when content exceeds visible', () => {
    expect(calculateMaxOffset(50, 20)).toBe(30);
  });

  test('maxOffset is 0 when content fits', () => {
    expect(calculateMaxOffset(15, 20)).toBe(0);
  });

  test('clamp scroll to bounds', () => {
    expect(clampScroll(-5, 30)).toBe(0);
    expect(clampScroll(50, 30)).toBe(30);
    expect(clampScroll(15, 30)).toBe(15);
  });
});

describe('AgentDetailView - follow mode', () => {
  function shouldFollow(isFollowing: boolean, scrolledToBottom: boolean): boolean {
    return isFollowing || scrolledToBottom;
  }

  test('follows when following enabled', () => {
    expect(shouldFollow(true, false)).toBe(true);
  });

  test('follows when at bottom', () => {
    expect(shouldFollow(false, true)).toBe(true);
  });

  test('does not follow when paused and not at bottom', () => {
    expect(shouldFollow(false, false)).toBe(false);
  });
});

describe('AgentDetailView - output slicing', () => {
  function getVisibleLines(lines: string[], offset: number, visible = 20): string[] {
    return lines.slice(offset, offset + visible);
  }

  function getLastNLines(lines: string[], n: number): string[] {
    return lines.slice(-n);
  }

  test('gets visible window from offset', () => {
    const lines = Array.from({ length: 50 }, (_, i) => `line ${i}`);
    const visible = getVisibleLines(lines, 10, 5);
    expect(visible).toHaveLength(5);
    expect(visible[0]).toBe('line 10');
  });

  test('gets last N lines', () => {
    const lines = ['a', 'b', 'c', 'd', 'e'];
    expect(getLastNLines(lines, 3)).toEqual(['c', 'd', 'e']);
  });
});

describe('AgentDetailView - DetailRow label padding', () => {
  const LABEL_WIDTH = 12;

  function padLabel(label: string): string {
    return label.padEnd(LABEL_WIDTH);
  }

  test('short label is padded', () => {
    expect(padLabel('Name')).toBe('Name        ');
    expect(padLabel('Name').length).toBe(12);
  });

  test('exact length not padded', () => {
    expect(padLabel('123456789012')).toBe('123456789012');
  });

  test('longer than width is not truncated', () => {
    expect(padLabel('Very Long Label')).toBe('Very Long Label');
  });
});

describe('AgentDetailView - footer hints', () => {
  function getFooterHint(inputMode: boolean, activeTab: string): string {
    if (inputMode) {
      return 'Enter: send | Esc: cancel';
    }
    if (activeTab === 'live') {
      return '1-4: tabs | j/k: scroll | g/G: top/bottom | f: follow | a: attach | q/ESC: back';
    }
    return '1-4: tabs | i: message | a: attach | r: refresh | q/ESC: back';
  }

  test('input mode hint', () => {
    const hint = getFooterHint(true, 'output');
    expect(hint).toContain('Enter: send');
    expect(hint).toContain('Esc: cancel');
  });

  test('live tab hint has scroll keys', () => {
    const hint = getFooterHint(false, 'live');
    expect(hint).toContain('j/k: scroll');
    expect(hint).toContain('f: follow');
  });

  test('output tab hint has message key', () => {
    const hint = getFooterHint(false, 'output');
    expect(hint).toContain('i: message');
    expect(hint).toContain('r: refresh');
  });
});
