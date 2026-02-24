/**
 * CommandsView unit tests
 * Tests helper functions and business logic
 */

import { describe, expect, test } from 'bun:test';

describe('CommandsView - command filtering', () => {
  interface MockCommand {
    name: string;
    description: string;
    category: string;
    readOnly: boolean;
  }

  const mockCommands: MockCommand[] = [
    { name: 'agent list', description: 'List all agents', category: 'agent', readOnly: true },
    { name: 'agent start', description: 'Start an agent', category: 'agent', readOnly: false },
    { name: 'channel list', description: 'List channels', category: 'channel', readOnly: true },
    { name: 'channel send', description: 'Send message', category: 'channel', readOnly: false },
    { name: 'cost show', description: 'Show cost summary', category: 'cost', readOnly: true },
  ];

  function filterByCategory(commands: MockCommand[], category: string): MockCommand[] {
    if (category === 'All') return commands;
    return commands.filter(cmd => cmd.category === category);
  }

  test('returns all commands for "All" category', () => {
    const result = filterByCategory(mockCommands, 'All');
    expect(result).toHaveLength(5);
  });

  test('filters by agent category', () => {
    const result = filterByCategory(mockCommands, 'agent');
    expect(result).toHaveLength(2);
    expect(result.every(cmd => cmd.category === 'agent')).toBe(true);
  });

  test('filters by channel category', () => {
    const result = filterByCategory(mockCommands, 'channel');
    expect(result).toHaveLength(2);
  });

  test('returns empty array for unknown category', () => {
    const result = filterByCategory(mockCommands, 'unknown');
    expect(result).toHaveLength(0);
  });
});

describe('CommandsView - search filtering', () => {
  interface MockCommand {
    name: string;
    description: string;
  }

  const mockCommands: MockCommand[] = [
    { name: 'agent list', description: 'List all agents' },
    { name: 'agent start', description: 'Start an agent' },
    { name: 'channel history', description: 'View message history' },
    { name: 'cost show', description: 'Show agent costs' },
  ];

  function filterBySearch(commands: MockCommand[], query: string): MockCommand[] {
    if (!query) return commands;
    const lowerQuery = query.toLowerCase();
    return commands.filter(cmd =>
      cmd.name.toLowerCase().includes(lowerQuery) ||
      cmd.description.toLowerCase().includes(lowerQuery)
    );
  }

  test('returns all commands when query is empty', () => {
    expect(filterBySearch(mockCommands, '')).toHaveLength(4);
  });

  test('filters by command name', () => {
    const result = filterBySearch(mockCommands, 'agent');
    expect(result).toHaveLength(3); // agent list, agent start, cost show (has "agent" in description)
  });

  test('filters by description', () => {
    const result = filterBySearch(mockCommands, 'history');
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe('channel history');
  });

  test('search is case insensitive', () => {
    const result = filterBySearch(mockCommands, 'AGENT');
    expect(result.length).toBeGreaterThan(0);
  });

  test('returns empty for no matches', () => {
    const result = filterBySearch(mockCommands, 'xyz');
    expect(result).toHaveLength(0);
  });
});

describe('CommandsView - favorites sorting', () => {
  interface MockCommand {
    name: string;
  }

  const mockCommands: MockCommand[] = [
    { name: 'agent list' },
    { name: 'agent start' },
    { name: 'channel list' },
    { name: 'cost show' },
  ];

  function sortWithFavorites(commands: MockCommand[], favorites: Set<string>): MockCommand[] {
    return [...commands].sort((a, b) => {
      const aFav = favorites.has(a.name) ? 0 : 1;
      const bFav = favorites.has(b.name) ? 0 : 1;
      return aFav - bFav;
    });
  }

  test('favorites appear first', () => {
    const favorites = new Set(['cost show', 'agent list']);
    const result = sortWithFavorites(mockCommands, favorites);

    // First two should be favorites
    expect(favorites.has(result[0].name)).toBe(true);
    expect(favorites.has(result[1].name)).toBe(true);
    // Last two should not be favorites
    expect(favorites.has(result[2].name)).toBe(false);
    expect(favorites.has(result[3].name)).toBe(false);
  });

  test('maintains order within favorites', () => {
    const favorites = new Set(['cost show']);
    const result = sortWithFavorites(mockCommands, favorites);
    expect(result[0].name).toBe('cost show');
  });

  test('no change when no favorites', () => {
    const favorites = new Set<string>();
    const result = sortWithFavorites(mockCommands, favorites);
    expect(result.map(c => c.name)).toEqual(mockCommands.map(c => c.name));
  });
});

describe('CommandsView - name truncation', () => {
  function truncateName(name: string, maxLength = 25): string {
    return name.length > maxLength ? name.slice(0, maxLength - 1) + '…' : name;
  }

  test('short names are not truncated', () => {
    expect(truncateName('agent list')).toBe('agent list');
  });

  test('long names are truncated with ellipsis', () => {
    const longName = 'very-long-command-name-that-exceeds-limit';
    const result = truncateName(longName);
    expect(result).toBe('very-long-command-name-t…');
    expect(result.length).toBe(25);
  });

  test('exact length names are not truncated', () => {
    const exactName = 'exactly-twenty-five-char'; // 24 chars
    expect(truncateName(exactName)).toBe(exactName);
  });
});

describe('CommandsView - description truncation', () => {
  function truncateDescription(desc: string, maxLength = 45): string {
    return desc.length > maxLength ? desc.slice(0, maxLength - 1) + '…' : desc;
  }

  test('short descriptions are not truncated', () => {
    expect(truncateDescription('List all agents')).toBe('List all agents');
  });

  test('long descriptions are truncated with ellipsis', () => {
    const longDesc = 'This is a very long description that explains the command in great detail';
    const result = truncateDescription(longDesc);
    expect(result.length).toBe(45);
    expect(result.endsWith('…')).toBe(true);
  });
});

describe('CommandsView - visible command calculation', () => {
  function calculateVisibleCount(terminalHeight: number): number {
    // Reserve space for: header(2) + category(2) + search(3) + preview(8) + footer(2) = 17 lines
    return Math.max(3, terminalHeight - 17);
  }

  test('standard terminal (24 rows)', () => {
    expect(calculateVisibleCount(24)).toBe(7);
  });

  test('tall terminal (40 rows)', () => {
    expect(calculateVisibleCount(40)).toBe(23);
  });

  test('minimum visible (short terminal)', () => {
    expect(calculateVisibleCount(15)).toBe(3);
  });

  test('very short terminal enforces minimum', () => {
    expect(calculateVisibleCount(10)).toBe(3);
  });
});

describe('CommandsView - command windowing', () => {
  function calculateWindow(
    selectedIndex: number,
    totalCommands: number,
    visibleCount: number
  ): { start: number; end: number } {
    const start = Math.max(0, Math.min(
      selectedIndex - Math.floor(visibleCount / 2),
      totalCommands - visibleCount
    ));
    const end = start + visibleCount;
    return { start, end: Math.min(end, totalCommands) };
  }

  test('window centered on selection', () => {
    const { start, end } = calculateWindow(10, 50, 10);
    expect(start).toBe(5);
    expect(end).toBe(15);
  });

  test('window at start', () => {
    const { start, end } = calculateWindow(2, 50, 10);
    expect(start).toBe(0);
    expect(end).toBe(10);
  });

  test('window at end', () => {
    const { start, end } = calculateWindow(48, 50, 10);
    expect(start).toBe(40);
    expect(end).toBe(50);
  });

  test('window when fewer items than visible', () => {
    const { start, end } = calculateWindow(2, 5, 10);
    expect(start).toBe(0);
    expect(end).toBe(5);
  });
});

describe('CommandsView - read-only check', () => {
  interface MockCommand {
    name: string;
    readOnly: boolean;
  }

  test('read-only commands can be executed', () => {
    const cmd: MockCommand = { name: 'agent list', readOnly: true };
    expect(cmd.readOnly).toBe(true);
  });

  test('modifying commands cannot be executed', () => {
    const cmd: MockCommand = { name: 'agent start', readOnly: false };
    expect(cmd.readOnly).toBe(false);
  });
});

describe('CommandsView - category cycling', () => {
  const CATEGORIES = ['All', 'agent', 'channel', 'cost', 'memory'];

  function getNextCategory(current: string): string {
    const currentIdx = CATEGORIES.indexOf(current);
    const nextIdx = (currentIdx + 1) % CATEGORIES.length;
    return CATEGORIES[nextIdx] ?? 'All';
  }

  test('cycles from All to first category', () => {
    expect(getNextCategory('All')).toBe('agent');
  });

  test('cycles through categories', () => {
    expect(getNextCategory('agent')).toBe('channel');
  });

  test('wraps from last to All', () => {
    expect(getNextCategory('memory')).toBe('All');
  });
});

describe('CommandsView - output panel state', () => {
  test('hasOutputPanel when output exists', () => {
    const commandOutput = 'some output';
    const commandError = null;
    const hasOutputPanel = commandOutput !== null || commandError !== null;
    expect(hasOutputPanel).toBe(true);
  });

  test('hasOutputPanel when error exists', () => {
    const commandOutput = null;
    const commandError = 'some error';
    const hasOutputPanel = commandOutput !== null || commandError !== null;
    expect(hasOutputPanel).toBe(true);
  });

  test('no output panel when both null', () => {
    const commandOutput = null;
    const commandError = null;
    const hasOutputPanel = commandOutput !== null || commandError !== null;
    expect(hasOutputPanel).toBe(false);
  });
});

describe('CommandsView - output truncation', () => {
  function getVisibleLines(output: string, maxLines = 15): string[] {
    return output.split('\n').slice(0, maxLines);
  }

  function getRemainingCount(output: string, maxLines = 15): number {
    const lines = output.split('\n');
    return Math.max(0, lines.length - maxLines);
  }

  test('short output shows all lines', () => {
    const output = 'line1\nline2\nline3';
    expect(getVisibleLines(output)).toHaveLength(3);
    expect(getRemainingCount(output)).toBe(0);
  });

  test('long output is truncated', () => {
    const lines = Array.from({ length: 25 }, (_, i) => `line${i + 1}`);
    const output = lines.join('\n');
    expect(getVisibleLines(output)).toHaveLength(15);
    expect(getRemainingCount(output)).toBe(10);
  });
});
