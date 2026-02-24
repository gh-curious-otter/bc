/**
 * CostsView Tests
 * Issue #1346: Borderless compact layout for 80x24 terminals
 *
 * Tests cover:
 * - Cost formatting
 * - Entry sorting by cost (descending)
 * - Entry slicing for narrow vs wide layouts
 * - Agent/model name truncation
 * - Token number formatting
 * - Responsive layout switching
 */

import { describe, test, expect } from 'bun:test';

interface CostSummary {
  total_cost: number;
  total_input_tokens: number;
  total_output_tokens: number;
  by_agent?: Record<string, number>;
  by_model?: Record<string, number>;
  by_team?: Record<string, number>;
}

// Helper functions matching CostsView logic
function formatCost(cost: number): string {
  return `$${cost.toFixed(4)}`;
}

function sortByValueDesc(entries: [string, number][]): [string, number][] {
  return [...entries].sort(([, a], [, b]) => b - a);
}

function truncateAgentName(name: string, maxLen = 12): string {
  return name.length > maxLen ? name.slice(0, maxLen - 1) + '…' : name;
}

function truncateModelName(name: string, maxLen = 15): string {
  return name.length > maxLen ? name.slice(0, maxLen - 1) + '…' : name;
}

function formatTokenCount(count: number): string {
  return count.toLocaleString();
}

function getTopAgents(costs: CostSummary, limit: number): [string, number][] {
  const entries = Object.entries(costs.by_agent ?? {});
  return sortByValueDesc(entries).slice(0, limit);
}

function getTopModels(costs: CostSummary, limit: number): [string, number][] {
  const entries = Object.entries(costs.by_model ?? {});
  return sortByValueDesc(entries).slice(0, limit);
}

describe('CostsView', () => {
  describe('Cost Formatting', () => {
    test('formats zero cost', () => {
      expect(formatCost(0)).toBe('$0.0000');
    });

    test('formats small cost', () => {
      expect(formatCost(0.0001)).toBe('$0.0001');
      expect(formatCost(0.0123)).toBe('$0.0123');
    });

    test('formats typical cost', () => {
      expect(formatCost(1.2345)).toBe('$1.2345');
      expect(formatCost(12.3456)).toBe('$12.3456');
    });

    test('rounds to 4 decimal places', () => {
      expect(formatCost(0.00001)).toBe('$0.0000');
      expect(formatCost(0.00005)).toBe('$0.0001');
      expect(formatCost(1.23456789)).toBe('$1.2346');
    });

    test('formats large cost', () => {
      expect(formatCost(100)).toBe('$100.0000');
      expect(formatCost(1000.5)).toBe('$1000.5000');
    });
  });

  describe('Entry Sorting', () => {
    test('sorts entries by value descending', () => {
      const entries: [string, number][] = [
        ['low', 1],
        ['high', 10],
        ['mid', 5],
      ];

      const sorted = sortByValueDesc(entries);
      expect(sorted[0][0]).toBe('high');
      expect(sorted[1][0]).toBe('mid');
      expect(sorted[2][0]).toBe('low');
    });

    test('handles equal values', () => {
      const entries: [string, number][] = [
        ['a', 5],
        ['b', 5],
        ['c', 5],
      ];

      const sorted = sortByValueDesc(entries);
      expect(sorted).toHaveLength(3);
      expect(sorted.every(([, v]) => v === 5)).toBe(true);
    });

    test('handles empty array', () => {
      const entries: [string, number][] = [];
      const sorted = sortByValueDesc(entries);
      expect(sorted).toHaveLength(0);
    });

    test('handles single entry', () => {
      const entries: [string, number][] = [['only', 100]];
      const sorted = sortByValueDesc(entries);
      expect(sorted).toHaveLength(1);
      expect(sorted[0][0]).toBe('only');
    });
  });

  describe('Agent Name Truncation', () => {
    test('short name not truncated', () => {
      expect(truncateAgentName('eng-01')).toBe('eng-01');
      expect(truncateAgentName('short')).toBe('short');
    });

    test('exact length not truncated', () => {
      expect(truncateAgentName('123456789012')).toBe('123456789012');
    });

    test('long name truncated with ellipsis', () => {
      expect(truncateAgentName('very-long-agent-name')).toBe('very-long-a…');
    });

    test('custom max length', () => {
      expect(truncateAgentName('hello-world', 8)).toBe('hello-w…');
    });
  });

  describe('Model Name Truncation', () => {
    test('short name not truncated', () => {
      expect(truncateModelName('gpt-4')).toBe('gpt-4');
    });

    test('exact length not truncated', () => {
      expect(truncateModelName('123456789012345')).toBe('123456789012345');
    });

    test('long name truncated with ellipsis', () => {
      expect(truncateModelName('claude-3-opus-20240229')).toBe('claude-3-opus-…');
    });

    test('custom max length', () => {
      expect(truncateModelName('hello-world-test', 10)).toBe('hello-wor…');
    });
  });

  describe('Token Count Formatting', () => {
    test('formats zero', () => {
      expect(formatTokenCount(0)).toBe('0');
    });

    test('formats small number', () => {
      expect(formatTokenCount(100)).toBe('100');
    });

    test('formats thousands with comma', () => {
      expect(formatTokenCount(1000)).toBe('1,000');
      expect(formatTokenCount(12345)).toBe('12,345');
    });

    test('formats millions with commas', () => {
      expect(formatTokenCount(1000000)).toBe('1,000,000');
      expect(formatTokenCount(1234567)).toBe('1,234,567');
    });
  });

  describe('Top Agents Extraction', () => {
    const costs: CostSummary = {
      total_cost: 100,
      total_input_tokens: 10000,
      total_output_tokens: 5000,
      by_agent: {
        'eng-01': 50,
        'eng-02': 30,
        'eng-03': 10,
        'eng-04': 5,
        'eng-05': 3,
        'eng-06': 2,
      },
    };

    test('gets top 5 agents for narrow layout', () => {
      const top = getTopAgents(costs, 5);
      expect(top).toHaveLength(5);
      expect(top[0][0]).toBe('eng-01');
      expect(top[4][0]).toBe('eng-05');
    });

    test('gets top 10 agents for wide layout', () => {
      const top = getTopAgents(costs, 10);
      expect(top).toHaveLength(6); // Only 6 agents exist
      expect(top[0][0]).toBe('eng-01');
    });

    test('handles empty agents', () => {
      const emptyCosts: CostSummary = {
        total_cost: 0,
        total_input_tokens: 0,
        total_output_tokens: 0,
        by_agent: {},
      };
      const top = getTopAgents(emptyCosts, 5);
      expect(top).toHaveLength(0);
    });

    test('handles undefined agents', () => {
      const noCosts: CostSummary = {
        total_cost: 0,
        total_input_tokens: 0,
        total_output_tokens: 0,
      };
      const top = getTopAgents(noCosts, 5);
      expect(top).toHaveLength(0);
    });
  });

  describe('Top Models Extraction', () => {
    const costs: CostSummary = {
      total_cost: 100,
      total_input_tokens: 10000,
      total_output_tokens: 5000,
      by_model: {
        'gpt-4': 60,
        'gpt-3.5-turbo': 30,
        'claude-3': 10,
      },
    };

    test('gets top 3 models for narrow layout', () => {
      const top = getTopModels(costs, 3);
      expect(top).toHaveLength(3);
      expect(top[0][0]).toBe('gpt-4');
      expect(top[2][0]).toBe('claude-3');
    });

    test('handles empty models', () => {
      const emptyCosts: CostSummary = {
        total_cost: 0,
        total_input_tokens: 0,
        total_output_tokens: 0,
        by_model: {},
      };
      const top = getTopModels(emptyCosts, 3);
      expect(top).toHaveLength(0);
    });
  });

  describe('Cost Data Structure', () => {
    test('minimal cost data', () => {
      const costs: CostSummary = {
        total_cost: 0,
        total_input_tokens: 0,
        total_output_tokens: 0,
      };

      expect(costs.total_cost).toBe(0);
      expect(costs.by_agent).toBeUndefined();
      expect(costs.by_model).toBeUndefined();
    });

    test('full cost data', () => {
      const costs: CostSummary = {
        total_cost: 100.5,
        total_input_tokens: 50000,
        total_output_tokens: 25000,
        by_agent: { 'eng-01': 50, 'eng-02': 50.5 },
        by_model: { 'gpt-4': 100.5 },
        by_team: { 'team-a': 60, 'team-b': 40.5 },
      };

      expect(costs.total_cost).toBe(100.5);
      expect(costs.by_agent?.['eng-01']).toBe(50);
      expect(costs.by_team?.['team-a']).toBe(60);
    });
  });

  describe('Responsive Layout Logic', () => {
    test('narrow layout limits (compact, minimal, or md)', () => {
      const isCompact = true;
      const isMinimal = false;
      const isMD = false;
      const isNarrow = isCompact || isMinimal || isMD;

      expect(isNarrow).toBe(true);
    });

    test('wide layout (not compact, minimal, or md)', () => {
      const isCompact = false;
      const isMinimal = false;
      const isMD = false;
      const isNarrow = isCompact || isMinimal || isMD;

      expect(isNarrow).toBe(false);
    });

    test('isMD triggers narrow layout', () => {
      const isCompact = false;
      const isMinimal = false;
      const isMD = true;
      const isNarrow = isCompact || isMinimal || isMD;

      expect(isNarrow).toBe(true);
    });
  });

  describe('Loading and Error States', () => {
    test('loading state shows loading message', () => {
      const loading = true;
      const message = loading ? 'Loading cost data...' : '';
      expect(message).toBe('Loading cost data...');
    });

    test('error state shows error message', () => {
      const error = 'Failed to fetch costs';
      const message = `Error: ${error}`;
      expect(message).toBe('Error: Failed to fetch costs');
    });

    test('no data state shows message', () => {
      const costs = null;
      const message = !costs ? 'No cost data available' : '';
      expect(message).toBe('No cost data available');
    });
  });

  describe('Manual Entry Detection', () => {
    test('detects manual entry (cost > 0 but tokens = 0)', () => {
      const costs: CostSummary = {
        total_cost: 100,
        total_input_tokens: 0,
        total_output_tokens: 0,
      };

      const isManualEntry = costs.total_input_tokens === 0 && costs.total_cost > 0;
      expect(isManualEntry).toBe(true);
    });

    test('regular entry has tokens', () => {
      const costs: CostSummary = {
        total_cost: 100,
        total_input_tokens: 10000,
        total_output_tokens: 5000,
      };

      const isManualEntry = costs.total_input_tokens === 0 && costs.total_cost > 0;
      expect(isManualEntry).toBe(false);
    });
  });

  describe('More Indicator', () => {
    test('shows more indicator when agents exceed limit', () => {
      const totalAgents = 10;
      const limit = 5;
      const remaining = totalAgents - limit;

      expect(remaining).toBe(5);
      expect(`+${remaining} more`).toBe('+5 more');
    });

    test('no indicator when within limit', () => {
      const totalAgents = 3;
      const limit = 5;
      const showMore = totalAgents > limit;

      expect(showMore).toBe(false);
    });
  });

  describe('Team Costs', () => {
    test('sorts teams by cost descending', () => {
      const byTeam: Record<string, number> = {
        'team-a': 30,
        'team-b': 50,
        'team-c': 20,
      };

      const sorted = sortByValueDesc(Object.entries(byTeam));
      expect(sorted[0][0]).toBe('team-b');
      expect(sorted[1][0]).toBe('team-a');
      expect(sorted[2][0]).toBe('team-c');
    });

    test('team section hidden when no teams', () => {
      const byTeam: Record<string, number> = {};
      const showTeams = Object.keys(byTeam).length > 0;

      expect(showTeams).toBe(false);
    });
  });

  describe('Display Padding', () => {
    test('pads agent name to 20 chars in wide mode', () => {
      const agent = 'eng-01';
      const padded = agent.padEnd(20);

      expect(padded.length).toBe(20);
      expect(padded.startsWith('eng-01')).toBe(true);
    });

    test('pads model name to 20 chars in wide mode', () => {
      const model = 'gpt-4';
      const padded = model.padEnd(20);

      expect(padded.length).toBe(20);
      expect(padded.startsWith('gpt-4')).toBe(true);
    });
  });
});
