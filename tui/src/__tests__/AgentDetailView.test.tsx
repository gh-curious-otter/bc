/**
 * AgentDetailView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test, beforeEach, afterEach } from 'bun:test';

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

  test('undefined returns (no task)', () => {
    expect(normalizeTask(undefined)).toBe('(no task)');
  });

  test('empty string returns empty', () => {
    expect(normalizeTask('')).toBe('(no task)');
  });

  test('Sautéed replaced with Working', () => {
    expect(normalizeTask('Sautéed tests')).toBe('Working tests');
  });

  test('Sauteed (ASCII) replaced with Working', () => {
    expect(normalizeTask('Sauteed code review')).toBe('Working code review');
  });

  test('Cooked replaced with Processed', () => {
    expect(normalizeTask('Cooked data files')).toBe('Processed data files');
  });

  test('Cogitated replaced with Thinking', () => {
    expect(normalizeTask('Cogitated about design')).toBe('Thinking about design');
  });

  test('Marinated replaced with Idle', () => {
    expect(normalizeTask('Marinated on task')).toBe('Idle on task');
  });

  test('Frolicking replaced with Active', () => {
    expect(normalizeTask('Frolicking in tests')).toBe('Active in tests');
  });

  test('normal task unchanged', () => {
    expect(normalizeTask('Running tests')).toBe('Running tests');
  });

  test('task with no cooking terms unchanged', () => {
    expect(normalizeTask('Implementing feature #123')).toBe('Implementing feature #123');
  });
});

describe('AgentDetailView - colorizeOutputLine patterns', () => {
  // Returns the color that would be applied based on line content
  function getLineColor(line: string): string | null {
    const trimmed = line.trim().toLowerCase();

    // Error patterns
    if (
      trimmed.includes('error') ||
      trimmed.includes('failed') ||
      trimmed.includes('exception') ||
      trimmed.startsWith('✗') ||
      trimmed.startsWith('x ')
    ) {
      return 'red';
    }

    // Warning patterns
    if (
      trimmed.includes('warning') ||
      trimmed.includes('warn') ||
      trimmed.includes('deprecated') ||
      trimmed.startsWith('⚠')
    ) {
      return 'yellow';
    }

    // Success patterns
    if (
      trimmed.includes('success') ||
      trimmed.includes('passed') ||
      trimmed.includes('complete') ||
      trimmed.startsWith('✓') ||
      trimmed.startsWith('✔')
    ) {
      return 'green';
    }

    // Tool/command patterns
    if (
      trimmed.startsWith('>') ||
      trimmed.startsWith('$') ||
      trimmed.includes('running') ||
      trimmed.includes('executing')
    ) {
      return 'cyan';
    }

    // File paths
    if (trimmed.match(/^[./~].*\.(tsx?|jsx?|go|py|md|json)$/)) {
      return 'white';
    }

    return null;
  }

  // Error patterns
  test('error keyword is red', () => {
    expect(getLineColor('Error: something went wrong')).toBe('red');
  });

  test('failed keyword is red', () => {
    expect(getLineColor('Test failed')).toBe('red');
  });

  test('exception keyword is red', () => {
    expect(getLineColor('Exception thrown')).toBe('red');
  });

  test('✗ symbol is red', () => {
    expect(getLineColor('✗ Test did not pass')).toBe('red');
  });

  test('x prefix is red', () => {
    expect(getLineColor('x test failed')).toBe('red');
  });

  // Warning patterns
  test('warning keyword is yellow', () => {
    expect(getLineColor('Warning: deprecated API')).toBe('yellow');
  });

  test('warn keyword is yellow', () => {
    expect(getLineColor('WARN: missing config')).toBe('yellow');
  });

  test('deprecated keyword is yellow', () => {
    expect(getLineColor('Function is deprecated')).toBe('yellow');
  });

  test('⚠ symbol is yellow', () => {
    expect(getLineColor('⚠ Check your config')).toBe('yellow');
  });

  // Success patterns
  test('success keyword is green', () => {
    expect(getLineColor('Build success')).toBe('green');
  });

  test('passed keyword is green', () => {
    expect(getLineColor('All tests passed')).toBe('green');
  });

  test('complete keyword is green', () => {
    expect(getLineColor('Task complete')).toBe('green');
  });

  test('✓ symbol is green', () => {
    expect(getLineColor('✓ Test passed')).toBe('green');
  });

  test('✔ symbol is green', () => {
    expect(getLineColor('✔ Done')).toBe('green');
  });

  // Tool/command patterns
  test('> prefix is cyan', () => {
    expect(getLineColor('> Running npm install')).toBe('cyan');
  });

  test('$ prefix is cyan', () => {
    expect(getLineColor('$ make build')).toBe('cyan');
  });

  test('running keyword is cyan', () => {
    expect(getLineColor('Running tests...')).toBe('cyan');
  });

  test('executing keyword is cyan', () => {
    expect(getLineColor('Executing command')).toBe('cyan');
  });

  // File path patterns
  test('ts file path is white', () => {
    expect(getLineColor('./src/index.ts')).toBe('white');
  });

  test('tsx file path is white', () => {
    expect(getLineColor('./components/App.tsx')).toBe('white');
  });

  test('go file path is white', () => {
    expect(getLineColor('./main.go')).toBe('white');
  });

  test('py file path is white', () => {
    expect(getLineColor('~/scripts/test.py')).toBe('white');
  });

  test('json file path is white', () => {
    expect(getLineColor('/config/settings.json')).toBe('white');
  });

  // Default (no special color)
  test('normal line has no special color', () => {
    expect(getLineColor('Just a normal line')).toBe(null);
  });

  test('empty line has no special color', () => {
    expect(getLineColor('')).toBe(null);
  });
});

describe('AgentDetailView - formatDate', () => {
  function formatDate(dateString: string | undefined): string {
    if (!dateString) return '-';
    try {
      const date = new Date(dateString);
      if (isNaN(date.getTime())) return dateString;
      return date.toLocaleString();
    } catch {
      return dateString;
    }
  }

  test('undefined returns dash', () => {
    expect(formatDate(undefined)).toBe('-');
  });

  test('empty string returns dash', () => {
    expect(formatDate('')).toBe('-');
  });

  test('valid ISO date formats correctly', () => {
    const result = formatDate('2026-02-24T12:30:00Z');
    // Result varies by locale, just check it contains date components
    expect(result).toContain('2026');
  });

  test('invalid date returns original string', () => {
    expect(formatDate('not-a-date')).toBe('not-a-date');
  });
});

describe('AgentDetailView - formatTime', () => {
  function formatTime(timestamp: string): string {
    try {
      const date = new Date(timestamp);
      if (isNaN(date.getTime())) return timestamp;
      return date.toLocaleTimeString();
    } catch {
      return timestamp;
    }
  }

  test('valid timestamp formats to time', () => {
    const result = formatTime('2026-02-24T14:30:45Z');
    // Result varies by locale, just check it contains colon for time format
    expect(result).toContain(':');
  });

  test('invalid timestamp returns original', () => {
    expect(formatTime('not-a-time')).toBe('not-a-time');
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

  test('small number unchanged', () => {
    expect(formatNumber(123)).toBe('123');
  });

  test('999 unchanged', () => {
    expect(formatNumber(999)).toBe('999');
  });

  test('1000 formats to K', () => {
    expect(formatNumber(1000)).toBe('1.0K');
  });

  test('1500 formats to K', () => {
    expect(formatNumber(1500)).toBe('1.5K');
  });

  test('45000 formats to K', () => {
    expect(formatNumber(45000)).toBe('45.0K');
  });

  test('1000000 formats to M', () => {
    expect(formatNumber(1000000)).toBe('1.0M');
  });

  test('2500000 formats to M', () => {
    expect(formatNumber(2500000)).toBe('2.5M');
  });

  test('zero returns string 0', () => {
    expect(formatNumber(0)).toBe('0');
  });
});

describe('AgentDetailView - truncateMessage', () => {
  function truncateMessage(message: string, maxLen: number): string {
    if (message.length <= maxLen) return message;
    return message.slice(0, maxLen - 3) + '...';
  }

  test('short message unchanged', () => {
    expect(truncateMessage('hello', 10)).toBe('hello');
  });

  test('exact length unchanged', () => {
    expect(truncateMessage('1234567890', 10)).toBe('1234567890');
  });

  test('long message truncated', () => {
    expect(truncateMessage('this is a long message', 15)).toBe('this is a lo...');
  });

  test('truncation at 40 chars', () => {
    const longMsg = 'This is a very long message that exceeds the maximum display length for activity events';
    const result = truncateMessage(longMsg, 40);
    expect(result.length).toBe(40);
    expect(result.endsWith('...')).toBe(true);
  });
});

describe('AgentDetailView - formatUptime', () => {
  let realDateNow: () => number;
  const fixedNow = new Date('2026-02-24T12:00:00Z').getTime();

  beforeEach(() => {
    realDateNow = Date.now;
    Date.now = () => fixedNow;
  });

  afterEach(() => {
    Date.now = realDateNow;
  });

  function formatUptime(startedAt: string | undefined): string {
    if (!startedAt) return '-';
    try {
      const started = new Date(startedAt);
      const now = new Date(Date.now());
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

  test('undefined returns dash', () => {
    expect(formatUptime(undefined)).toBe('-');
  });

  test('30 minutes ago shows 30m', () => {
    const started = new Date(fixedNow - 30 * 60 * 1000).toISOString();
    expect(formatUptime(started)).toBe('30m');
  });

  test('0 minutes shows 0m', () => {
    const started = new Date(fixedNow).toISOString();
    expect(formatUptime(started)).toBe('0m');
  });

  test('1 hour shows 1h 0m', () => {
    const started = new Date(fixedNow - 60 * 60 * 1000).toISOString();
    expect(formatUptime(started)).toBe('1h 0m');
  });

  test('90 minutes shows 1h 30m', () => {
    const started = new Date(fixedNow - 90 * 60 * 1000).toISOString();
    expect(formatUptime(started)).toBe('1h 30m');
  });

  test('3 hours 45 minutes shows correctly', () => {
    const started = new Date(fixedNow - (3 * 60 + 45) * 60 * 1000).toISOString();
    expect(formatUptime(started)).toBe('3h 45m');
  });
});

describe('AgentDetailView - tab switching', () => {
  type TabType = 'output' | 'live' | 'details' | 'metrics';

  function getTabForKey(key: string): TabType | null {
    if (key === '1') return 'output';
    if (key === '2') return 'live';
    if (key === '3') return 'details';
    if (key === '4') return 'metrics';
    return null;
  }

  test('1 switches to output', () => {
    expect(getTabForKey('1')).toBe('output');
  });

  test('2 switches to live', () => {
    expect(getTabForKey('2')).toBe('live');
  });

  test('3 switches to details', () => {
    expect(getTabForKey('3')).toBe('details');
  });

  test('4 switches to metrics', () => {
    expect(getTabForKey('4')).toBe('metrics');
  });

  test('other keys return null', () => {
    expect(getTabForKey('5')).toBe(null);
    expect(getTabForKey('a')).toBe(null);
  });
});

describe('AgentDetailView - input mode', () => {
  function shouldEnterInputMode(key: string): boolean {
    return key === 'i' || key === 'm';
  }

  function shouldExitInputMode(input: string, key: { return?: boolean; escape?: boolean }): boolean {
    return key.return === true || key.escape === true;
  }

  test('i key enters input mode', () => {
    expect(shouldEnterInputMode('i')).toBe(true);
  });

  test('m key enters input mode', () => {
    expect(shouldEnterInputMode('m')).toBe(true);
  });

  test('other keys do not enter input mode', () => {
    expect(shouldEnterInputMode('x')).toBe(false);
    expect(shouldEnterInputMode('q')).toBe(false);
  });

  test('return exits input mode', () => {
    expect(shouldExitInputMode('', { return: true })).toBe(true);
  });

  test('escape exits input mode', () => {
    expect(shouldExitInputMode('', { escape: true })).toBe(true);
  });

  test('other keys do not exit', () => {
    expect(shouldExitInputMode('a', {})).toBe(false);
  });
});

describe('AgentDetailView - live mode scrolling', () => {
  function calculateScrollDown(
    currentOffset: number,
    totalLines: number,
    viewportSize: number
  ): { newOffset: number; isFollowing: boolean } {
    const maxOffset = Math.max(0, totalLines - viewportSize);
    const newOffset = Math.min(currentOffset + 1, maxOffset);
    const isFollowing = newOffset >= maxOffset;
    return { newOffset, isFollowing };
  }

  function calculateScrollUp(currentOffset: number): { newOffset: number; isFollowing: false } {
    return {
      newOffset: Math.max(0, currentOffset - 1),
      isFollowing: false,
    };
  }

  test('scroll down increments offset', () => {
    const result = calculateScrollDown(5, 100, 20);
    expect(result.newOffset).toBe(6);
    expect(result.isFollowing).toBe(false);
  });

  test('scroll down at bottom enables following', () => {
    const result = calculateScrollDown(79, 100, 20);
    expect(result.newOffset).toBe(80);
    expect(result.isFollowing).toBe(true);
  });

  test('scroll down caps at max offset', () => {
    const result = calculateScrollDown(80, 100, 20);
    expect(result.newOffset).toBe(80);
    expect(result.isFollowing).toBe(true);
  });

  test('scroll up decrements offset', () => {
    const result = calculateScrollUp(5);
    expect(result.newOffset).toBe(4);
    expect(result.isFollowing).toBe(false);
  });

  test('scroll up at top stays at 0', () => {
    const result = calculateScrollUp(0);
    expect(result.newOffset).toBe(0);
    expect(result.isFollowing).toBe(false);
  });
});

describe('AgentDetailView - live mode jump navigation', () => {
  function jumpToTop(): { offset: number; isFollowing: false } {
    return { offset: 0, isFollowing: false };
  }

  function jumpToBottom(totalLines: number, viewportSize: number): { offset: number; isFollowing: true } {
    return {
      offset: Math.max(0, totalLines - viewportSize),
      isFollowing: true,
    };
  }

  test('g jumps to top', () => {
    const result = jumpToTop();
    expect(result.offset).toBe(0);
    expect(result.isFollowing).toBe(false);
  });

  test('G jumps to bottom', () => {
    const result = jumpToBottom(100, 20);
    expect(result.offset).toBe(80);
    expect(result.isFollowing).toBe(true);
  });

  test('G with few lines stays at 0', () => {
    const result = jumpToBottom(10, 20);
    expect(result.offset).toBe(0);
    expect(result.isFollowing).toBe(true);
  });
});

describe('AgentDetailView - follow toggle', () => {
  function toggleFollow(
    currentFollowing: boolean,
    totalLines: number,
    viewportSize: number
  ): { isFollowing: boolean; offset?: number } {
    if (currentFollowing) {
      return { isFollowing: false };
    }
    return {
      isFollowing: true,
      offset: Math.max(0, totalLines - viewportSize),
    };
  }

  test('toggling from following disables it', () => {
    const result = toggleFollow(true, 100, 20);
    expect(result.isFollowing).toBe(false);
    expect(result.offset).toBeUndefined();
  });

  test('toggling from not following enables and jumps to bottom', () => {
    const result = toggleFollow(false, 100, 20);
    expect(result.isFollowing).toBe(true);
    expect(result.offset).toBe(80);
  });
});

describe('AgentDetailView - back navigation', () => {
  function shouldGoBack(input: string, key: { escape?: boolean }): boolean {
    return input === 'q' || key.escape === true;
  }

  test('q goes back', () => {
    expect(shouldGoBack('q', {})).toBe(true);
  });

  test('escape goes back', () => {
    expect(shouldGoBack('', { escape: true })).toBe(true);
  });

  test('other keys do not go back', () => {
    expect(shouldGoBack('a', {})).toBe(false);
    expect(shouldGoBack('1', {})).toBe(false);
  });
});

describe('AgentDetailView - output slice', () => {
  function getVisibleLines(lines: string[], height: number): string[] {
    return lines.slice(-(height - 2));
  }

  test('fewer lines than height shows all', () => {
    const lines = ['a', 'b', 'c'];
    expect(getVisibleLines(lines, 10)).toEqual(['a', 'b', 'c']);
  });

  test('more lines than height shows bottom portion', () => {
    const lines = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j'];
    const result = getVisibleLines(lines, 5);
    expect(result).toEqual(['h', 'i', 'j']); // Last 3 lines (5-2=3)
  });
});

describe('AgentDetailView - live output slice', () => {
  function getLiveVisibleLines(
    lines: string[],
    scrollOffset: number,
    viewportSize: number
  ): string[] {
    return lines.slice(scrollOffset, scrollOffset + viewportSize);
  }

  test('slices from offset', () => {
    const lines = Array.from({ length: 100 }, (_, i) => `line${i}`);
    const result = getLiveVisibleLines(lines, 10, 20);
    expect(result.length).toBe(20);
    expect(result[0]).toBe('line10');
    expect(result[19]).toBe('line29');
  });

  test('handles offset at end', () => {
    const lines = Array.from({ length: 100 }, (_, i) => `line${i}`);
    const result = getLiveVisibleLines(lines, 90, 20);
    expect(result.length).toBe(10); // Only 10 lines remaining
    expect(result[0]).toBe('line90');
  });
});

describe('AgentDetailView - DetailRow label padding', () => {
  const LABEL_WIDTH = 12;

  function padLabel(label: string, width = LABEL_WIDTH): string {
    return label.padEnd(width);
  }

  test('short label padded', () => {
    expect(padLabel('Name')).toBe('Name        ');
    expect(padLabel('Name').length).toBe(12);
  });

  test('exact length label unchanged', () => {
    expect(padLabel('LongLabelHer')).toBe('LongLabelHer');
    expect(padLabel('LongLabelHer').length).toBe(12);
  });

  test('longer label not truncated', () => {
    expect(padLabel('VeryLongLabel')).toBe('VeryLongLabel');
    expect(padLabel('VeryLongLabel').length).toBe(13);
  });
});

describe('AgentDetailView - TabButton styling', () => {
  function getTabColor(active: boolean): string {
    return active ? 'cyan' : 'gray';
  }

  function getTabBold(active: boolean): boolean {
    return active;
  }

  test('active tab is cyan', () => {
    expect(getTabColor(true)).toBe('cyan');
  });

  test('inactive tab is gray', () => {
    expect(getTabColor(false)).toBe('gray');
  });

  test('active tab is bold', () => {
    expect(getTabBold(true)).toBe(true);
  });

  test('inactive tab is not bold', () => {
    expect(getTabBold(false)).toBe(false);
  });
});

describe('AgentDetailView - footer hints', () => {
  function getFooterText(inputMode: boolean, activeTab: string): string {
    if (inputMode) {
      return 'Enter: send | Esc: cancel';
    }
    if (activeTab === 'live') {
      return '1-4: tabs | j/k: scroll | g/G: top/bottom | f: follow | a: attach | q/ESC: back';
    }
    return '1-4: tabs | i: message | a: attach | r: refresh | q/ESC: back';
  }

  test('input mode shows send/cancel hints', () => {
    expect(getFooterText(true, 'output')).toBe('Enter: send | Esc: cancel');
  });

  test('live tab shows scroll hints', () => {
    const hint = getFooterText(false, 'live');
    expect(hint).toContain('j/k: scroll');
    expect(hint).toContain('f: follow');
  });

  test('output tab shows message hint', () => {
    const hint = getFooterText(false, 'output');
    expect(hint).toContain('i: message');
  });

  test('details tab shows standard hints', () => {
    const hint = getFooterText(false, 'details');
    expect(hint).toContain('a: attach');
    expect(hint).toContain('r: refresh');
  });
});

describe('AgentDetailView - message buffer handling', () => {
  function handleBackspace(buffer: string): string {
    return buffer.slice(0, -1);
  }

  function handleCharInput(buffer: string, char: string): string {
    return buffer + char;
  }

  test('backspace removes last char', () => {
    expect(handleBackspace('hello')).toBe('hell');
  });

  test('backspace on empty stays empty', () => {
    expect(handleBackspace('')).toBe('');
  });

  test('char input appends', () => {
    expect(handleCharInput('hello', ' ')).toBe('hello ');
    expect(handleCharInput('hello', 'w')).toBe('hellow');
  });
});

describe('AgentDetailView - focus state', () => {
  function getFocusState(inputMode: boolean): 'input' | 'view' {
    return inputMode ? 'input' : 'view';
  }

  test('input mode sets focus to input', () => {
    expect(getFocusState(true)).toBe('input');
  });

  test('non-input mode sets focus to view', () => {
    expect(getFocusState(false)).toBe('view');
  });
});

describe('AgentDetailView - line count display', () => {
  function getLineCountText(
    scrollOffset: number,
    viewportSize: number,
    totalLines: number
  ): string {
    const start = scrollOffset + 1;
    const end = Math.min(scrollOffset + viewportSize, totalLines);
    let text = `Lines ${String(start)}-${String(end)} of ${String(totalLines)}`;
    if (scrollOffset === 0) {
      text += ' (following)';
    }
    return text;
  }

  test('formats line range correctly', () => {
    expect(getLineCountText(0, 20, 100)).toBe('Lines 1-20 of 100 (following)');
  });

  test('middle of list no following indicator', () => {
    expect(getLineCountText(40, 20, 100)).toBe('Lines 41-60 of 100');
  });

  test('end of list formats correctly', () => {
    expect(getLineCountText(80, 20, 100)).toBe('Lines 81-100 of 100');
  });
});

describe('AgentDetailView - cost display', () => {
  function formatCost(cost: number): string {
    return `$${cost.toFixed(4)}`;
  }

  test('formats cost with 4 decimals', () => {
    expect(formatCost(0.0025)).toBe('$0.0025');
  });

  test('formats zero cost', () => {
    expect(formatCost(0)).toBe('$0.0000');
  });

  test('formats large cost', () => {
    expect(formatCost(12.3456)).toBe('$12.3456');
  });
});

describe('AgentDetailView - activity type extraction', () => {
  function extractActivityType(type: string): string {
    return type.split('.').pop() ?? type;
  }

  test('extracts last segment', () => {
    expect(extractActivityType('agent.task.completed')).toBe('completed');
  });

  test('handles single segment', () => {
    expect(extractActivityType('started')).toBe('started');
  });

  test('handles empty string', () => {
    expect(extractActivityType('')).toBe('');
  });
});

describe('AgentDetailView - activity slice limit', () => {
  function getVisibleActivities<T>(activities: T[]): T[] {
    return activities.slice(0, 8);
  }

  test('fewer than 8 shows all', () => {
    const activities = [1, 2, 3, 4, 5];
    expect(getVisibleActivities(activities)).toEqual([1, 2, 3, 4, 5]);
  });

  test('more than 8 shows first 8', () => {
    const activities = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
    expect(getVisibleActivities(activities)).toEqual([1, 2, 3, 4, 5, 6, 7, 8]);
  });
});

describe('AgentDetailView - empty state messages', () => {
  function getOutputEmptyMessage(): string {
    return 'No output yet. Agent may be idle.';
  }

  function getLiveEmptyMessage(): string {
    return 'Waiting for output...';
  }

  function getActivityEmptyMessage(): string {
    return 'No recent activity';
  }

  test('output empty message', () => {
    expect(getOutputEmptyMessage()).toContain('No output');
  });

  test('live empty message', () => {
    expect(getLiveEmptyMessage()).toContain('Waiting');
  });

  test('activity empty message', () => {
    expect(getActivityEmptyMessage()).toContain('No recent activity');
  });
});

describe('AgentDetailView - input box border color', () => {
  function getInputBorderColor(inputMode: boolean): string {
    return inputMode ? 'cyan' : 'gray';
  }

  test('input mode has cyan border', () => {
    expect(getInputBorderColor(true)).toBe('cyan');
  });

  test('normal mode has gray border', () => {
    expect(getInputBorderColor(false)).toBe('gray');
  });
});
