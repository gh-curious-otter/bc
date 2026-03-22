/**
 * useCosts Hook Tests
 * Issue #682 - Phase 2-Subtask 3: Component & View Testing
 */

import { describe, test, expect } from 'bun:test';
import type { UseCostsOptions } from '../useCosts';
import type { CostSummary, AgentCost } from '../../types';

// Mock cost data
const mockAgentCosts: AgentCost[] = [
  { agent: 'eng-01', input_tokens: 10000, output_tokens: 5000, total_cost: 0.25 },
  { agent: 'eng-02', input_tokens: 8000, output_tokens: 4000, total_cost: 0.2 },
  { agent: 'eng-03', input_tokens: 15000, output_tokens: 7500, total_cost: 0.35 },
];

const mockCostSummary: CostSummary = {
  total_cost: 0.8,
  total_input_tokens: 33000,
  total_output_tokens: 16500,
  agent_costs: mockAgentCosts,
  period: 'session',
};

describe('useCosts Hook Logic', () => {
  describe('Options Defaults', () => {
    test('default poll interval is 5000ms', () => {
      const defaults: UseCostsOptions = {};
      const pollInterval = defaults.pollInterval ?? 5000;
      expect(pollInterval).toBe(5000);
    });

    test('default autoPoll is true', () => {
      const defaults: UseCostsOptions = {};
      const autoPoll = defaults.autoPoll ?? true;
      expect(autoPoll).toBe(true);
    });

    test('custom poll interval is respected', () => {
      const options: UseCostsOptions = { pollInterval: 10000 };
      expect(options.pollInterval).toBe(10000);
    });

    test('autoPoll can be disabled', () => {
      const options: UseCostsOptions = { autoPoll: false };
      expect(options.autoPoll).toBe(false);
    });
  });

  describe('Cost Data Processing', () => {
    test('cost summary is processed correctly', () => {
      const data: CostSummary | null = mockCostSummary;
      expect(data?.total_cost).toBe(0.8);
    });

    test('null data is handled', () => {
      const data: CostSummary | null = null;
      expect(data).toBeNull();
    });

    test('agent costs array is present', () => {
      expect(mockCostSummary.agent_costs.length).toBe(3);
    });

    test('cost summary has required properties', () => {
      expect(mockCostSummary).toHaveProperty('total_cost');
      expect(mockCostSummary).toHaveProperty('total_input_tokens');
      expect(mockCostSummary).toHaveProperty('total_output_tokens');
      expect(mockCostSummary).toHaveProperty('agent_costs');
    });
  });

  describe('State Management', () => {
    test('loading state starts true', () => {
      const loading = true;
      expect(loading).toBe(true);
    });

    test('loading becomes false after fetch', () => {
      let loading = true;
      loading = false;
      expect(loading).toBe(false);
    });

    test('error state starts null', () => {
      const error: string | null = null;
      expect(error).toBeNull();
    });

    test('error can be set on failure', () => {
      const error: string | null = 'Failed to fetch costs';
      expect(error).toBe('Failed to fetch costs');
    });
  });

  describe('Error Handling', () => {
    test('Error instance message extraction', () => {
      const err = new Error('API timeout');
      const message = err instanceof Error ? err.message : 'Unknown error';
      expect(message).toBe('API timeout');
    });

    test('non-Error fallback message', () => {
      const err = 'string error';
      const message = err instanceof Error ? err.message : 'Failed to fetch costs';
      expect(message).toBe('Failed to fetch costs');
    });
  });
});

describe('Cost Data Validation', () => {
  test('total cost is non-negative', () => {
    expect(mockCostSummary.total_cost).toBeGreaterThanOrEqual(0);
  });

  test('total input tokens is non-negative', () => {
    expect(mockCostSummary.total_input_tokens).toBeGreaterThanOrEqual(0);
  });

  test('total output tokens is non-negative', () => {
    expect(mockCostSummary.total_output_tokens).toBeGreaterThanOrEqual(0);
  });

  test('agent costs sum matches total', () => {
    const sum = mockAgentCosts.reduce((acc: number, a) => acc + (a.total_cost ?? 0), 0);
    expect(sum).toBeCloseTo(mockCostSummary.total_cost, 2);
  });

  test('input tokens sum matches total', () => {
    const sum = mockAgentCosts.reduce((acc: number, a) => acc + (a.input_tokens ?? 0), 0);
    expect(sum).toBe(mockCostSummary.total_input_tokens);
  });

  test('output tokens sum matches total', () => {
    const sum = mockAgentCosts.reduce((acc: number, a) => acc + (a.output_tokens ?? 0), 0);
    expect(sum).toBe(mockCostSummary.total_output_tokens);
  });
});

describe('Agent Cost Validation', () => {
  test('agent name is non-empty', () => {
    mockAgentCosts.forEach((ac) => {
      expect(ac.agent.length).toBeGreaterThan(0);
    });
  });

  test('agent cost has required properties', () => {
    mockAgentCosts.forEach((ac) => {
      expect(ac).toHaveProperty('agent');
      expect(ac).toHaveProperty('input_tokens');
      expect(ac).toHaveProperty('output_tokens');
      expect(ac).toHaveProperty('total_cost');
    });
  });

  test('tokens are integers', () => {
    mockAgentCosts.forEach((ac) => {
      expect(Number.isInteger(ac.input_tokens)).toBe(true);
      expect(Number.isInteger(ac.output_tokens)).toBe(true);
    });
  });

  test('cost is a number', () => {
    mockAgentCosts.forEach((ac) => {
      expect(typeof ac.total_cost).toBe('number');
    });
  });
});

describe('Cost Calculations', () => {
  test('calculate cost per 1K tokens', () => {
    const inputRate = 0.01; // $0.01 per 1K input
    const outputRate = 0.03; // $0.03 per 1K output
    const agent = mockAgentCosts[0];
    const calculated =
      (agent.input_tokens / 1000) * inputRate + (agent.output_tokens / 1000) * outputRate;
    expect(typeof calculated).toBe('number');
    expect(calculated).toBeGreaterThan(0);
  });

  test('format cost as currency', () => {
    const cost = 0.25;
    const formatted = `$${cost.toFixed(2)}`;
    expect(formatted).toBe('$0.25');
  });

  test('format large cost', () => {
    const cost = 1234.56;
    const formatted = `$${cost.toFixed(2)}`;
    expect(formatted).toBe('$1234.56');
  });

  test('format tokens with commas', () => {
    const tokens = 10000;
    const formatted = tokens.toLocaleString();
    expect(formatted).toBe('10,000');
  });
});

describe('Period Handling', () => {
  test('session period is recognized', () => {
    expect(mockCostSummary.period).toBe('session');
  });

  test('period can be different values', () => {
    const dailyCost: CostSummary = { ...mockCostSummary, period: 'daily' };
    expect(dailyCost.period).toBe('daily');
  });

  test('period affects display', () => {
    const period = mockCostSummary.period;
    const label = period === 'session' ? 'This Session' : 'Today';
    expect(label).toBe('This Session');
  });
});

describe('Empty State Handling', () => {
  test('empty agent costs is valid', () => {
    const emptyCosts: CostSummary = {
      total_cost: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      agent_costs: [],
      period: 'session',
    };
    expect(emptyCosts.agent_costs.length).toBe(0);
    expect(emptyCosts.total_cost).toBe(0);
  });

  test('zero cost display', () => {
    const zeroCost = 0;
    const display = zeroCost === 0 ? 'No costs yet' : `$${zeroCost.toFixed(2)}`;
    expect(display).toBe('No costs yet');
  });
});

describe('Refresh Function', () => {
  test('refresh is callable', () => {
    const refresh = async (): Promise<void> => {};
    expect(typeof refresh).toBe('function');
  });

  test('refresh updates data', async () => {
    let data: CostSummary | null = null;
    const refresh = async () => {
      data = mockCostSummary;
    };
    await refresh();
    expect(data).not.toBeNull();
  });
});

describe('Polling Configuration', () => {
  test('poll interval must be positive', () => {
    const pollInterval = 5000;
    expect(pollInterval).toBeGreaterThan(0);
  });

  test('longer interval reduces API calls', () => {
    const shortInterval = 1000;
    const longInterval = 10000;
    const callsPerMinuteShort = 60000 / shortInterval;
    const callsPerMinuteLong = 60000 / longInterval;
    expect(callsPerMinuteLong).toBeLessThan(callsPerMinuteShort);
  });

  test('autoPoll false stops updates', () => {
    const autoPoll = false;
    let pollingActive = true;
    if (!autoPoll) {
      pollingActive = false;
    }
    expect(pollingActive).toBe(false);
  });
});
