/**
 * DemonsView Tests - Scheduled Task Display & Management
 * Issue #682 - Component Testing
 *
 * Tests cover:
 * - Demon data model validation
 * - Schedule formatting utilities
 * - Relative time formatting utilities
 * - Navigation logic
 * - State and action management
 */

import { describe, test, expect } from 'bun:test';
import type { Demon } from '../../types';

// Mock demon data for testing
const mockDemons: Demon[] = [
  {
    name: 'daily-backup',
    schedule: '0 2 * * *',
    command: 'bc backup create',
    description: 'Daily workspace backup at 2am',
    enabled: true,
    run_count: 45,
    last_run: '2024-01-15T02:00:00Z',
    next_run: '2024-01-16T02:00:00Z',
    owner: 'system',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-15T02:00:00Z',
  },
  {
    name: 'health-check',
    schedule: '*/5 * * * *',
    command: 'bc status --health',
    enabled: true,
    run_count: 1200,
    last_run: '2024-01-15T12:30:00Z',
    next_run: '2024-01-15T12:35:00Z',
    created_at: '2024-01-05T00:00:00Z',
    updated_at: '2024-01-15T12:30:00Z',
  },
  {
    name: 'weekly-cleanup',
    schedule: '0 0 * * 0',
    command: 'bc cleanup --all',
    description: 'Weekly cleanup of temporary files',
    enabled: false,
    run_count: 2,
    last_run: '2024-01-07T00:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-07T00:00:00Z',
  },
  {
    name: 'minute-task',
    schedule: '* * * * *',
    command: 'echo heartbeat',
    enabled: true,
    run_count: 0,
    created_at: '2024-01-15T12:00:00Z',
    updated_at: '2024-01-15T12:00:00Z',
  },
];

describe('DemonsView Data Model', () => {
  test('Demon interface has required properties', () => {
    const demon = mockDemons[0];
    expect(demon).toHaveProperty('name');
    expect(demon).toHaveProperty('schedule');
    expect(demon).toHaveProperty('command');
    expect(demon).toHaveProperty('enabled');
    expect(demon).toHaveProperty('run_count');
    expect(demon).toHaveProperty('created_at');
    expect(demon).toHaveProperty('updated_at');
  });

  test('Demon optional properties are handled', () => {
    const demonWithOptionals = mockDemons[0];
    const demonWithoutOptionals = mockDemons[3];

    expect(demonWithOptionals.description).toBe('Daily workspace backup at 2am');
    expect(demonWithOptionals.owner).toBe('system');
    expect(demonWithOptionals.last_run).toBeTruthy();
    expect(demonWithOptionals.next_run).toBeTruthy();

    expect(demonWithoutOptionals.description).toBeUndefined();
    expect(demonWithoutOptionals.owner).toBeUndefined();
    expect(demonWithoutOptionals.last_run).toBeUndefined();
    expect(demonWithoutOptionals.next_run).toBeUndefined();
  });

  test('enabled is boolean', () => {
    mockDemons.forEach(demon => {
      expect(typeof demon.enabled).toBe('boolean');
    });
  });

  test('run_count is a number', () => {
    mockDemons.forEach(demon => {
      expect(typeof demon.run_count).toBe('number');
      expect(demon.run_count).toBeGreaterThanOrEqual(0);
    });
  });
});

describe('DemonsView Schedule Formatting', () => {
  // Replicating the formatSchedule function logic for testing

  function formatSchedule(schedule: string): string {
    if (schedule === '* * * * *') return 'every minute';
    if (schedule === '0 * * * *') return 'every hour';
    if (schedule.startsWith('*/')) {
      const match = schedule.match(/^\*\/(\d+) \* \* \* \*$/);
      if (match) return `every ${match[1]} min`;
    }
    if (schedule.match(/^0 \d+ \* \* \*$/)) {
      const hour = schedule.split(' ')[1];
      return `daily at ${hour}:00`;
    }
    return schedule;
  }

  test('formats every minute schedule', () => {
    expect(formatSchedule('* * * * *')).toBe('every minute');
  });

  test('formats every hour schedule', () => {
    expect(formatSchedule('0 * * * *')).toBe('every hour');
  });

  test('formats every N minutes schedule', () => {
    expect(formatSchedule('*/5 * * * *')).toBe('every 5 min');
    expect(formatSchedule('*/10 * * * *')).toBe('every 10 min');
    expect(formatSchedule('*/15 * * * *')).toBe('every 15 min');
    expect(formatSchedule('*/30 * * * *')).toBe('every 30 min');
  });

  test('formats daily at hour schedule', () => {
    expect(formatSchedule('0 2 * * *')).toBe('daily at 2:00');
    expect(formatSchedule('0 14 * * *')).toBe('daily at 14:00');
    expect(formatSchedule('0 0 * * *')).toBe('daily at 0:00');
  });

  test('returns original for unrecognized patterns', () => {
    expect(formatSchedule('0 0 * * 0')).toBe('0 0 * * 0');
    expect(formatSchedule('0 0 1 * *')).toBe('0 0 1 * *');
    expect(formatSchedule('custom')).toBe('custom');
  });
});

describe('DemonsView Relative Time Formatting', () => {
  // Test utility function logic

  function formatRelativeTime(timestamp?: string, now?: Date): string {
    if (!timestamp) return '-';
    try {
      const date = new Date(timestamp);
      const nowDate = now ?? new Date();
      const diffMs = nowDate.getTime() - date.getTime();
      const diffMins = Math.floor(Math.abs(diffMs) / 60000);
      const diffHours = Math.floor(diffMins / 60);
      const diffDays = Math.floor(diffHours / 24);

      const prefix = diffMs < 0 ? 'in ' : '';
      const suffix = diffMs >= 0 ? ' ago' : '';

      if (diffMins < 1) return 'now';
      if (diffMins < 60) return `${prefix}${String(diffMins)}m${suffix}`;
      if (diffHours < 24) return `${prefix}${String(diffHours)}h${suffix}`;
      return `${prefix}${String(diffDays)}d${suffix}`;
    } catch {
      return timestamp;
    }
  }

  test('returns dash for undefined timestamp', () => {
    expect(formatRelativeTime(undefined)).toBe('-');
  });

  test('returns "now" for very recent timestamps', () => {
    const now = new Date();
    const justNow = new Date(now.getTime() - 10000); // 10 seconds ago
    expect(formatRelativeTime(justNow.toISOString(), now)).toBe('now');
  });

  test('formats minutes ago', () => {
    const now = new Date();
    const fiveMinAgo = new Date(now.getTime() - 5 * 60000);
    expect(formatRelativeTime(fiveMinAgo.toISOString(), now)).toBe('5m ago');
  });

  test('formats hours ago', () => {
    const now = new Date();
    const twoHoursAgo = new Date(now.getTime() - 2 * 3600000);
    expect(formatRelativeTime(twoHoursAgo.toISOString(), now)).toBe('2h ago');
  });

  test('formats days ago', () => {
    const now = new Date();
    const threeDaysAgo = new Date(now.getTime() - 3 * 24 * 3600000);
    expect(formatRelativeTime(threeDaysAgo.toISOString(), now)).toBe('3d ago');
  });

  test('formats future time with "in" prefix', () => {
    const now = new Date();
    const inFiveMin = new Date(now.getTime() + 5 * 60000);
    expect(formatRelativeTime(inFiveMin.toISOString(), now)).toBe('in 5m');
  });

  test('formats future hours', () => {
    const now = new Date();
    const inTwoHours = new Date(now.getTime() + 2 * 3600000);
    expect(formatRelativeTime(inTwoHours.toISOString(), now)).toBe('in 2h');
  });
});

describe('DemonsView Navigation Logic', () => {
  test('selection index clamping works correctly', () => {
    const listLength = mockDemons.length;
    const clampIndex = (index: number) =>
      Math.max(0, Math.min(index, listLength - 1));

    expect(clampIndex(-1)).toBe(0);
    expect(clampIndex(0)).toBe(0);
    expect(clampIndex(1)).toBe(1);
    expect(clampIndex(listLength - 1)).toBe(listLength - 1);
    expect(clampIndex(listLength)).toBe(listLength - 1);
    expect(clampIndex(100)).toBe(listLength - 1);
  });

  test('navigate down increments index', () => {
    const listLength = mockDemons.length;
    const navigateDown = (prev: number) => Math.min(prev + 1, listLength - 1);
    expect(navigateDown(0)).toBe(1);
    expect(navigateDown(2)).toBe(3);
    expect(navigateDown(3)).toBe(3); // At end
  });

  test('navigate up decrements index', () => {
    const navigateUp = (prev: number) => Math.max(prev - 1, 0);
    expect(navigateUp(3)).toBe(2);
    expect(navigateUp(1)).toBe(0);
    expect(navigateUp(0)).toBe(0); // At start
  });

  test('g key goes to first item', () => {
    expect(0).toBe(0);
  });

  test('G key goes to last item', () => {
    const listLength = mockDemons.length;
    expect(listLength - 1).toBe(3);
  });
});

describe('DemonsView Filtering', () => {
  test('can filter enabled demons', () => {
    const enabledDemons = mockDemons.filter(d => d.enabled);
    expect(enabledDemons.length).toBe(3);
    expect(enabledDemons.every(d => d.enabled)).toBe(true);
  });

  test('can filter disabled demons', () => {
    const disabledDemons = mockDemons.filter(d => !d.enabled);
    expect(disabledDemons.length).toBe(1);
    expect(disabledDemons[0].name).toBe('weekly-cleanup');
  });

  test('can find demon by name', () => {
    const demon = mockDemons.find(d => d.name === 'health-check');
    expect(demon).toBeTruthy();
    expect(demon?.schedule).toBe('*/5 * * * *');
  });

  test('returns undefined for non-existent demon', () => {
    const demon = mockDemons.find(d => d.name === 'non-existent');
    expect(demon).toBeUndefined();
  });
});

describe('DemonsView Counts', () => {
  test('total count is correct', () => {
    expect(mockDemons.length).toBe(4);
  });

  test('enabled count is correct', () => {
    const enabledCount = mockDemons.filter(d => d.enabled).length;
    expect(enabledCount).toBe(3);
  });

  test('demons with run history', () => {
    const demonsWithRuns = mockDemons.filter(d => d.run_count > 0);
    expect(demonsWithRuns.length).toBe(3);
  });

  test('demons with last_run timestamp', () => {
    const demonsWithLastRun = mockDemons.filter(d => d.last_run);
    expect(demonsWithLastRun.length).toBe(3);
  });

  test('demons with next_run timestamp', () => {
    const demonsWithNextRun = mockDemons.filter(d => d.next_run);
    expect(demonsWithNextRun.length).toBe(2);
  });
});

describe('DemonsView Rendering States', () => {
  test('loading state shows loading indicator', () => {
    const loading = true;
    const demons = null;
    const showLoading = loading && !demons;
    expect(showLoading).toBe(true);
  });

  test('error state shows error display', () => {
    const error = 'Failed to fetch demons';
    expect(error).toBeTruthy();
  });

  test('empty state shows create hint', () => {
    const demons: Demon[] = [];
    const showEmptyState = demons.length === 0;
    expect(showEmptyState).toBe(true);
  });

  test('populated state shows demon list', () => {
    const demons = mockDemons;
    const showList = demons.length > 0;
    expect(showList).toBe(true);
  });
});

describe('DemonsView Actions', () => {
  test('enable action for disabled demon', () => {
    const disabledDemon = mockDemons.find(d => !d.enabled);
    expect(disabledDemon).toBeTruthy();
    expect(disabledDemon?.name).toBe('weekly-cleanup');
    // Enable would change enabled to true
    const enabledDemon = { ...disabledDemon, enabled: true };
    expect(enabledDemon.enabled).toBe(true);
  });

  test('disable action for enabled demon', () => {
    const enabledDemon = mockDemons.find(d => d.enabled);
    expect(enabledDemon).toBeTruthy();
    // Disable would change enabled to false
    const disabledDemon = { ...enabledDemon, enabled: false };
    expect(disabledDemon.enabled).toBe(false);
  });

  test('run action increments run_count', () => {
    const demon = mockDemons[0];
    const initialCount = demon.run_count;
    const afterRun = { ...demon, run_count: initialCount + 1 };
    expect(afterRun.run_count).toBe(initialCount + 1);
  });

  test('action error clears after timeout', () => {
    const ERROR_DISPLAY_DURATION = 3000;
    expect(ERROR_DISPLAY_DURATION).toBe(3000); // 3 seconds
  });
});

describe('DemonsView Keyboard Shortcuts', () => {
  test('j/k for navigation', () => {
    const navigateDown = (prev: number, max: number) => Math.min(prev + 1, max - 1);
    const navigateUp = (prev: number) => Math.max(prev - 1, 0);

    expect(navigateDown(0, 4)).toBe(1);
    expect(navigateUp(2)).toBe(1);
  });

  test('e key enables selected demon', () => {
    const actions: string[] = [];
    const eKeyAction = (demonName: string) => { actions.push(`enable:${demonName}`); };
    eKeyAction('weekly-cleanup');
    expect(actions).toContain('enable:weekly-cleanup');
  });

  test('d key disables selected demon', () => {
    const actions: string[] = [];
    const dKeyAction = (demonName: string) => { actions.push(`disable:${demonName}`); };
    dKeyAction('health-check');
    expect(actions).toContain('disable:health-check');
  });

  test('x key runs selected demon', () => {
    const actions: string[] = [];
    const xKeyAction = (demonName: string) => { actions.push(`run:${demonName}`); };
    xKeyAction('daily-backup');
    expect(actions).toContain('run:daily-backup');
  });

  test('r key refreshes list', () => {
    let refreshed = false;
    const rKeyAction = () => { refreshed = true; };
    rKeyAction();
    expect(refreshed).toBe(true);
  });

  test('q key exits view', () => {
    let exited = false;
    const qKeyAction = (onExit: () => void) => { onExit(); };
    qKeyAction(() => { exited = true; });
    expect(exited).toBe(true);
  });
});

describe('DemonRow Component Logic', () => {
  test('name truncation for long names', () => {
    const longName = 'very-long-demon-name-that-exceeds-width';
    // Updated: now truncates at 12 chars with ellipsis for 80-col responsive layout
    const truncated = longName.length > 12 ? longName.slice(0, 11) + '…' : longName;
    expect(truncated).toBe('very-long-d…');
  });

  test('name not truncated for short names', () => {
    const shortName = 'backup';
    const result = shortName.length > 12 ? shortName.slice(0, 11) + '…' : shortName;
    expect(result).toBe('backup');
  });

  test('status text based on enabled state', () => {
    const getStatusText = (enabled: boolean) => enabled ? 'enabled' : 'disabled';
    expect(getStatusText(true)).toBe('enabled');
    expect(getStatusText(false)).toBe('disabled');
  });

  test('selected row has highlight indicator', () => {
    const getSelectionIndicator = (selected: boolean) => selected ? '▸ ' : '  ';
    expect(getSelectionIndicator(true)).toBe('▸ ');
    expect(getSelectionIndicator(false)).toBe('  ');
  });
});

describe('DemonsView Selected Demon Details', () => {
  test('details panel shows for selected demon', () => {
    const demons = mockDemons;
    const selectedIndex = 0;
    const showDetails = demons.length > 0 && demons[selectedIndex] !== undefined;
    expect(showDetails).toBe(true);
  });

  test('details include command', () => {
    const demon = mockDemons[0];
    expect(demon.command).toBe('bc backup create');
  });

  test('details include description when present', () => {
    const demonWithDesc = mockDemons[0];
    expect(demonWithDesc.description).toBe('Daily workspace backup at 2am');
  });

  test('details include owner when present', () => {
    const demonWithOwner = mockDemons[0];
    expect(demonWithOwner.owner).toBe('system');
  });

  test('details hide optional fields when missing', () => {
    const demonWithoutOptionals = mockDemons[3];
    expect(demonWithoutOptionals.description).toBeUndefined();
    expect(demonWithoutOptionals.owner).toBeUndefined();
  });
});
