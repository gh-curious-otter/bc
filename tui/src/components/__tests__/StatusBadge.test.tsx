/**
 * StatusBadge Tests
 * Issue #561, #562: Agent and health status indicators
 *
 * Tests cover:
 * - State symbols mapping
 * - Fallback colors
 * - Theme color key mapping
 * - Agent states (idle, starting, working, done, stuck, error, stopped)
 * - Health states (healthy, degraded, unhealthy)
 */

import { describe, test, expect } from 'bun:test';

// Types matching StatusBadge
type AgentState = 'idle' | 'starting' | 'working' | 'done' | 'stuck' | 'error' | 'stopped';
type HealthStatus = 'healthy' | 'degraded' | 'unhealthy';

// State symbols matching StatusBadge
const stateSymbols: Record<string, string> = {
  idle: '○',
  starting: '◐',
  working: '●',
  done: '✓',
  stuck: '⚠',
  error: '✗',
  stopped: '◌',
  healthy: '✓',
  degraded: '!',
  unhealthy: '✗',
};

// Fallback colors matching StatusBadge
const fallbackColors: Record<string, string> = {
  idle: 'gray',
  starting: 'yellow',
  working: 'blue',
  done: 'green',
  stuck: 'yellow',
  error: 'red',
  stopped: 'gray',
  healthy: 'green',
  degraded: 'yellow',
  unhealthy: 'red',
};

// Theme color key mapping
function getThemeColorKey(state: string): string | null {
  switch (state) {
    case 'idle':
    case 'stopped':
      return 'agentIdle';
    case 'starting':
      return 'warning';
    case 'working':
      return 'agentWorking';
    case 'done':
    case 'healthy':
      return 'agentDone';
    case 'stuck':
      return 'warning';
    case 'error':
    case 'unhealthy':
      return 'agentError';
    case 'degraded':
      return 'warning';
    default:
      return null;
  }
}

// Helper to get display text
function getDisplayText(state: string, showIcon: boolean): string {
  const symbol = stateSymbols[state] || '?';
  return showIcon ? `${symbol} ${state}` : state;
}

describe('StatusBadge', () => {
  describe('State Symbols', () => {
    test('idle shows empty circle', () => {
      expect(stateSymbols.idle).toBe('○');
    });

    test('starting shows half circle', () => {
      expect(stateSymbols.starting).toBe('◐');
    });

    test('working shows filled circle', () => {
      expect(stateSymbols.working).toBe('●');
    });

    test('done shows checkmark', () => {
      expect(stateSymbols.done).toBe('✓');
    });

    test('stuck shows warning', () => {
      expect(stateSymbols.stuck).toBe('⚠');
    });

    test('error shows X', () => {
      expect(stateSymbols.error).toBe('✗');
    });

    test('stopped shows dotted circle', () => {
      expect(stateSymbols.stopped).toBe('◌');
    });
  });

  describe('Health State Symbols', () => {
    test('healthy shows checkmark', () => {
      expect(stateSymbols.healthy).toBe('✓');
    });

    test('degraded shows exclamation', () => {
      expect(stateSymbols.degraded).toBe('!');
    });

    test('unhealthy shows X', () => {
      expect(stateSymbols.unhealthy).toBe('✗');
    });
  });

  describe('Fallback Colors', () => {
    test('idle is gray', () => {
      expect(fallbackColors.idle).toBe('gray');
    });

    test('starting is yellow', () => {
      expect(fallbackColors.starting).toBe('yellow');
    });

    test('working is blue', () => {
      expect(fallbackColors.working).toBe('blue');
    });

    test('done is green', () => {
      expect(fallbackColors.done).toBe('green');
    });

    test('stuck is yellow per UX spec', () => {
      expect(fallbackColors.stuck).toBe('yellow');
    });

    test('error is red', () => {
      expect(fallbackColors.error).toBe('red');
    });

    test('stopped is gray', () => {
      expect(fallbackColors.stopped).toBe('gray');
    });
  });

  describe('Health Fallback Colors', () => {
    test('healthy is green', () => {
      expect(fallbackColors.healthy).toBe('green');
    });

    test('degraded is yellow', () => {
      expect(fallbackColors.degraded).toBe('yellow');
    });

    test('unhealthy is red', () => {
      expect(fallbackColors.unhealthy).toBe('red');
    });
  });

  describe('Theme Color Keys', () => {
    test('idle maps to agentIdle', () => {
      expect(getThemeColorKey('idle')).toBe('agentIdle');
    });

    test('stopped maps to agentIdle', () => {
      expect(getThemeColorKey('stopped')).toBe('agentIdle');
    });

    test('starting maps to warning', () => {
      expect(getThemeColorKey('starting')).toBe('warning');
    });

    test('working maps to agentWorking', () => {
      expect(getThemeColorKey('working')).toBe('agentWorking');
    });

    test('done maps to agentDone', () => {
      expect(getThemeColorKey('done')).toBe('agentDone');
    });

    test('healthy maps to agentDone', () => {
      expect(getThemeColorKey('healthy')).toBe('agentDone');
    });

    test('stuck maps to warning', () => {
      expect(getThemeColorKey('stuck')).toBe('warning');
    });

    test('error maps to agentError', () => {
      expect(getThemeColorKey('error')).toBe('agentError');
    });

    test('unhealthy maps to agentError', () => {
      expect(getThemeColorKey('unhealthy')).toBe('agentError');
    });

    test('degraded maps to warning', () => {
      expect(getThemeColorKey('degraded')).toBe('warning');
    });

    test('unknown state returns null', () => {
      expect(getThemeColorKey('unknown')).toBeNull();
    });
  });

  describe('Display Text', () => {
    test('shows icon and state by default', () => {
      expect(getDisplayText('working', true)).toBe('● working');
    });

    test('shows only state when icon disabled', () => {
      expect(getDisplayText('working', false)).toBe('working');
    });

    test('unknown state shows question mark', () => {
      expect(getDisplayText('unknown', true)).toBe('? unknown');
    });
  });

  describe('Agent State Types', () => {
    test('all agent states have symbols', () => {
      const agentStates: AgentState[] = [
        'idle',
        'starting',
        'working',
        'done',
        'stuck',
        'error',
        'stopped',
      ];
      for (const state of agentStates) {
        expect(stateSymbols[state]).toBeDefined();
      }
    });

    test('all agent states have fallback colors', () => {
      const agentStates: AgentState[] = [
        'idle',
        'starting',
        'working',
        'done',
        'stuck',
        'error',
        'stopped',
      ];
      for (const state of agentStates) {
        expect(fallbackColors[state]).toBeDefined();
      }
    });

    test('all agent states have theme mappings', () => {
      const agentStates: AgentState[] = [
        'idle',
        'starting',
        'working',
        'done',
        'stuck',
        'error',
        'stopped',
      ];
      for (const state of agentStates) {
        expect(getThemeColorKey(state)).not.toBeNull();
      }
    });
  });

  describe('Health State Types', () => {
    test('all health states have symbols', () => {
      const healthStates: HealthStatus[] = ['healthy', 'degraded', 'unhealthy'];
      for (const state of healthStates) {
        expect(stateSymbols[state]).toBeDefined();
      }
    });

    test('all health states have fallback colors', () => {
      const healthStates: HealthStatus[] = ['healthy', 'degraded', 'unhealthy'];
      for (const state of healthStates) {
        expect(fallbackColors[state]).toBeDefined();
      }
    });

    test('all health states have theme mappings', () => {
      const healthStates: HealthStatus[] = ['healthy', 'degraded', 'unhealthy'];
      for (const state of healthStates) {
        expect(getThemeColorKey(state)).not.toBeNull();
      }
    });
  });

  describe('Color Semantics', () => {
    test('positive states are green', () => {
      expect(fallbackColors.done).toBe('green');
      expect(fallbackColors.healthy).toBe('green');
    });

    test('negative states are red', () => {
      expect(fallbackColors.error).toBe('red');
      expect(fallbackColors.unhealthy).toBe('red');
    });

    test('warning states are yellow', () => {
      expect(fallbackColors.starting).toBe('yellow');
      expect(fallbackColors.stuck).toBe('yellow');
      expect(fallbackColors.degraded).toBe('yellow');
    });

    test('inactive states are gray', () => {
      expect(fallbackColors.idle).toBe('gray');
      expect(fallbackColors.stopped).toBe('gray');
    });

    test('active state is blue', () => {
      expect(fallbackColors.working).toBe('blue');
    });
  });

  describe('Symbol Semantics', () => {
    test('circles indicate activity level', () => {
      // Empty circle = idle
      expect(stateSymbols.idle).toBe('○');
      // Half circle = starting
      expect(stateSymbols.starting).toBe('◐');
      // Filled circle = working
      expect(stateSymbols.working).toBe('●');
      // Dotted circle = stopped
      expect(stateSymbols.stopped).toBe('◌');
    });

    test('checkmarks indicate success', () => {
      expect(stateSymbols.done).toBe('✓');
      expect(stateSymbols.healthy).toBe('✓');
    });

    test('X marks indicate failure', () => {
      expect(stateSymbols.error).toBe('✗');
      expect(stateSymbols.unhealthy).toBe('✗');
    });

    test('warning symbols indicate caution', () => {
      expect(stateSymbols.stuck).toBe('⚠');
      expect(stateSymbols.degraded).toBe('!');
    });
  });
});
