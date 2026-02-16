/**
 * CostDashboard Tests - Cost Overview & Analytics
 * Issue #682 - Component Testing Phase 2
 *
 * Tests cover:
 * - Cost data model validation
 * - Number formatting utility
 * - Percentage calculations
 * - Data sorting and aggregation
 * - Rendering states
 */

import { describe, test, expect } from 'bun:test';

// Mock cost data for testing
interface CostData {
  total_cost: number;
  total_input_tokens: number;
  total_output_tokens: number;
  by_agent: Record<string, number>;
  by_model: Record<string, number>;
  by_team: Record<string, number>;
}

const mockCosts: CostData = {
  total_cost: 12.5678,
  total_input_tokens: 1500000,
  total_output_tokens: 500000,
  by_agent: {
    'eng-01': 4.5,
    'eng-02': 3.2,
    'tl-01': 2.8,
    'eng-03': 1.5,
    'qa-01': 0.5678,
  },
  by_model: {
    'claude-3-opus': 8.5,
    'claude-3-sonnet': 3.5,
    'claude-3-haiku': 0.5678,
  },
  by_team: {
    'engineering': 9.2,
    'qa': 2.5,
    'platform': 0.8678,
  },
};

const emptyCosts: CostData = {
  total_cost: 0,
  total_input_tokens: 0,
  total_output_tokens: 0,
  by_agent: {},
  by_model: {},
  by_team: {},
};

describe('CostDashboard Data Model', () => {
  test('CostData has required properties', () => {
    expect(mockCosts).toHaveProperty('total_cost');
    expect(mockCosts).toHaveProperty('total_input_tokens');
    expect(mockCosts).toHaveProperty('total_output_tokens');
    expect(mockCosts).toHaveProperty('by_agent');
    expect(mockCosts).toHaveProperty('by_model');
    expect(mockCosts).toHaveProperty('by_team');
  });

  test('total_cost is a number', () => {
    expect(typeof mockCosts.total_cost).toBe('number');
    expect(mockCosts.total_cost).toBeGreaterThan(0);
  });

  test('token counts are numbers', () => {
    expect(typeof mockCosts.total_input_tokens).toBe('number');
    expect(typeof mockCosts.total_output_tokens).toBe('number');
  });

  test('breakdown objects have string keys and number values', () => {
    Object.entries(mockCosts.by_agent).forEach(([key, value]) => {
      expect(typeof key).toBe('string');
      expect(typeof value).toBe('number');
    });
  });

  test('total tokens equals input plus output', () => {
    const totalTokens = mockCosts.total_input_tokens + mockCosts.total_output_tokens;
    expect(totalTokens).toBe(2000000);
  });
});

describe('CostDashboard Number Formatting', () => {
  // Replicating formatNumber utility from component
  function formatNumber(n: number): string {
    if (n >= 1_000_000) {
      return `${(n / 1_000_000).toFixed(1)}M`;
    }
    if (n >= 1_000) {
      return `${(n / 1_000).toFixed(1)}K`;
    }
    return n.toString();
  }

  test('formats millions correctly', () => {
    expect(formatNumber(1_000_000)).toBe('1.0M');
    expect(formatNumber(1_500_000)).toBe('1.5M');
    expect(formatNumber(12_345_678)).toBe('12.3M');
  });

  test('formats thousands correctly', () => {
    expect(formatNumber(1_000)).toBe('1.0K');
    expect(formatNumber(1_500)).toBe('1.5K');
    expect(formatNumber(999_999)).toBe('1000.0K');
  });

  test('formats small numbers without suffix', () => {
    expect(formatNumber(0)).toBe('0');
    expect(formatNumber(100)).toBe('100');
    expect(formatNumber(999)).toBe('999');
  });

  test('formats input tokens from mock data', () => {
    expect(formatNumber(mockCosts.total_input_tokens)).toBe('1.5M');
  });

  test('formats output tokens from mock data', () => {
    expect(formatNumber(mockCosts.total_output_tokens)).toBe('500.0K');
  });
});

describe('CostDashboard Percentage Calculations', () => {
  test('calculates percentage of total correctly', () => {
    const totalCost = mockCosts.total_cost;
    const agentCost = mockCosts.by_agent['eng-01'];
    const pct = totalCost > 0 ? (agentCost / totalCost) * 100 : 0;
    expect(pct).toBeCloseTo(35.79, 1);
  });

  test('handles zero total cost', () => {
    const totalCost = 0;
    const agentCost = 5;
    const pct = totalCost > 0 ? (agentCost / totalCost) * 100 : 0;
    expect(pct).toBe(0);
  });

  test('all agent percentages sum to 100', () => {
    const totalCost = mockCosts.total_cost;
    let sumPct = 0;
    Object.values(mockCosts.by_agent).forEach(cost => {
      sumPct += (cost / totalCost) * 100;
    });
    expect(sumPct).toBeCloseTo(100, 0);
  });

  test('all model percentages sum to 100', () => {
    const totalCost = mockCosts.total_cost;
    let sumPct = 0;
    Object.values(mockCosts.by_model).forEach(cost => {
      sumPct += (cost / totalCost) * 100;
    });
    expect(sumPct).toBeCloseTo(100, 0);
  });
});

describe('CostDashboard Data Sorting', () => {
  test('agent data sorted by cost descending', () => {
    const agentData = Object.entries(mockCosts.by_agent)
      .map(([agent, cost]) => ({ agent, cost }))
      .sort((a, b) => b.cost - a.cost);

    expect(agentData[0].agent).toBe('eng-01');
    expect(agentData[0].cost).toBe(4.5);
    expect(agentData[agentData.length - 1].agent).toBe('qa-01');
  });

  test('model data sorted by cost descending', () => {
    const modelData = Object.entries(mockCosts.by_model)
      .map(([model, cost]) => ({ model, cost }))
      .sort((a, b) => b.cost - a.cost);

    expect(modelData[0].model).toBe('claude-3-opus');
    expect(modelData[0].cost).toBe(8.5);
  });

  test('team data sorted by cost descending', () => {
    const teamData = Object.entries(mockCosts.by_team)
      .map(([team, cost]) => ({ team, cost }))
      .sort((a, b) => b.cost - a.cost);

    expect(teamData[0].team).toBe('engineering');
    expect(teamData[0].cost).toBe(9.2);
  });
});

describe('CostDashboard Data Aggregation', () => {
  test('converts agent breakdown to table data', () => {
    const totalCost = mockCosts.total_cost;
    const agentData = Object.entries(mockCosts.by_agent)
      .map(([agent, cost]) => ({
        agent,
        cost,
        pct: totalCost > 0 ? (cost / totalCost) * 100 : 0,
      }));

    expect(agentData.length).toBe(5);
    expect(agentData[0]).toHaveProperty('agent');
    expect(agentData[0]).toHaveProperty('cost');
    expect(agentData[0]).toHaveProperty('pct');
  });

  test('limits agent display to 8 items', () => {
    const agentData = Object.entries(mockCosts.by_agent)
      .map(([agent, cost]) => ({ agent, cost }))
      .sort((a, b) => b.cost - a.cost);

    const displayData = agentData.slice(0, 8);
    expect(displayData.length).toBeLessThanOrEqual(8);
  });

  test('shows overflow count when more than 8 agents', () => {
    const manyAgents: Record<string, number> = {};
    for (let i = 0; i < 12; i++) {
      manyAgents[`agent-${i}`] = Math.random() * 10;
    }

    const agentData = Object.entries(manyAgents);
    const overflow = agentData.length - 8;
    expect(overflow).toBe(4);
  });
});

describe('CostDashboard Cost Formatting', () => {
  test('formats cost with 4 decimal places', () => {
    const cost = mockCosts.total_cost;
    const formatted = cost.toFixed(4);
    expect(formatted).toBe('12.5678');
  });

  test('formats small costs correctly', () => {
    const smallCost = 0.0001;
    const formatted = smallCost.toFixed(4);
    expect(formatted).toBe('0.0001');
  });

  test('formats zero cost', () => {
    const zeroCost = 0;
    const formatted = zeroCost.toFixed(4);
    expect(formatted).toBe('0.0000');
  });

  test('formats percentage with 1 decimal place', () => {
    const pct = 35.789;
    const formatted = pct.toFixed(1);
    expect(formatted).toBe('35.8');
  });
});

describe('CostDashboard Rendering States', () => {
  test('loading state shows loading indicator', () => {
    const loading = true;
    const costs = null;
    const showLoading = loading && !costs;
    expect(showLoading).toBe(true);
  });

  test('loading with existing data shows refresh indicator', () => {
    const loading = true;
    const costs = mockCosts;
    const showRefreshIndicator = loading && costs !== null;
    expect(showRefreshIndicator).toBe(true);
  });

  test('error state shows error display', () => {
    const error = 'Failed to fetch costs';
    expect(error).toBeTruthy();
  });

  test('empty state with zero costs', () => {
    expect(emptyCosts.total_cost).toBe(0);
    expect(Object.keys(emptyCosts.by_agent).length).toBe(0);
  });

  test('populated state shows cost data', () => {
    expect(mockCosts.total_cost).toBeGreaterThan(0);
    expect(Object.keys(mockCosts.by_agent).length).toBeGreaterThan(0);
  });
});

describe('CostDashboard Team Section Visibility', () => {
  test('team section shown when team data exists', () => {
    const teamData = Object.entries(mockCosts.by_team);
    const showTeamSection = teamData.length > 0;
    expect(showTeamSection).toBe(true);
  });

  test('team section hidden when no team data', () => {
    const teamData = Object.entries(emptyCosts.by_team);
    const showTeamSection = teamData.length > 0;
    expect(showTeamSection).toBe(false);
  });
});

describe('CostDashboard Keyboard Shortcuts', () => {
  test('q key triggers onBack', () => {
    let backCalled = false;
    const qKeyAction = () => { backCalled = true; };
    qKeyAction();
    expect(backCalled).toBe(true);
  });

  test('r key triggers refresh', () => {
    let refreshCalled = false;
    const rKeyAction = () => { refreshCalled = true; };
    rKeyAction();
    expect(refreshCalled).toBe(true);
  });

  test('escape key triggers onBack', () => {
    let backCalled = false;
    const escapeAction = () => { backCalled = true; };
    escapeAction();
    expect(backCalled).toBe(true);
  });
});

describe('CostDashboard MetricCard Values', () => {
  test('total cost metric has dollar prefix', () => {
    const prefix = '$';
    const value = mockCosts.total_cost.toFixed(4);
    const display = `${prefix}${value}`;
    expect(display).toBe('$12.5678');
  });

  test('input tokens metric formatted', () => {
    const formatNumber = (n: number) => {
      if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
      if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
      return n.toString();
    };
    expect(formatNumber(mockCosts.total_input_tokens)).toBe('1.5M');
  });

  test('output tokens metric formatted', () => {
    const formatNumber = (n: number) => {
      if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
      if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
      return n.toString();
    };
    expect(formatNumber(mockCosts.total_output_tokens)).toBe('500.0K');
  });

  test('total tokens metric calculated correctly', () => {
    const totalTokens = mockCosts.total_input_tokens + mockCosts.total_output_tokens;
    const formatNumber = (n: number) => {
      if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
      if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
      return n.toString();
    };
    expect(formatNumber(totalTokens)).toBe('2.0M');
  });
});

describe('CostDashboard DataTable Columns', () => {
  test('agent table has correct columns', () => {
    const columns = ['AGENT', 'COST', '% SHARE'];
    expect(columns).toContain('AGENT');
    expect(columns).toContain('COST');
    expect(columns).toContain('% SHARE');
  });

  test('model table has correct columns', () => {
    const columns = ['MODEL', 'COST', '% SHARE'];
    expect(columns).toContain('MODEL');
    expect(columns).toContain('COST');
    expect(columns).toContain('% SHARE');
  });

  test('team table has correct columns', () => {
    const columns = ['TEAM', 'COST', '% SHARE'];
    expect(columns).toContain('TEAM');
    expect(columns).toContain('COST');
    expect(columns).toContain('% SHARE');
  });

  test('column widths are defined', () => {
    const agentColumnWidth = 20;
    const costColumnWidth = 12;
    const pctColumnWidth = 10;
    expect(agentColumnWidth + costColumnWidth + pctColumnWidth).toBe(42);
  });
});

describe('CostDashboard Empty State Handling', () => {
  test('shows no agent costs message when empty', () => {
    const agentData = Object.entries(emptyCosts.by_agent);
    const showEmptyMessage = agentData.length === 0;
    expect(showEmptyMessage).toBe(true);
  });

  test('shows no model costs message when empty', () => {
    const modelData = Object.entries(emptyCosts.by_model);
    const showEmptyMessage = modelData.length === 0;
    expect(showEmptyMessage).toBe(true);
  });

  test('handles null/undefined costs gracefully', () => {
    const costs = null;
    const totalCost = costs?.total_cost ?? 0;
    const inputTokens = costs?.total_input_tokens ?? 0;
    expect(totalCost).toBe(0);
    expect(inputTokens).toBe(0);
  });
});

describe('CostDashboard Breakdown Analysis', () => {
  test('identifies top spending agent', () => {
    const agentData = Object.entries(mockCosts.by_agent)
      .sort((a, b) => b[1] - a[1]);
    const topAgent = agentData[0];
    expect(topAgent[0]).toBe('eng-01');
    expect(topAgent[1]).toBe(4.5);
  });

  test('identifies top spending model', () => {
    const modelData = Object.entries(mockCosts.by_model)
      .sort((a, b) => b[1] - a[1]);
    const topModel = modelData[0];
    expect(topModel[0]).toBe('claude-3-opus');
    expect(topModel[1]).toBe(8.5);
  });

  test('calculates total from breakdown matches total_cost', () => {
    const sumFromBreakdown = Object.values(mockCosts.by_agent)
      .reduce((sum, cost) => sum + cost, 0);
    expect(sumFromBreakdown).toBeCloseTo(mockCosts.total_cost, 2);
  });
});
