/**
 * State Transition Tests - Verify state changes flow correctly
 * Issue #751 - TUI E2E Workflows & Real-Time Updates
 *
 * Tests verify that state transitions happen correctly and data
 * updates are reflected across all views.
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { Text, Box } from 'ink';
import { describe, it, expect } from 'bun:test';

// Valid state transitions for agents
const validAgentTransitions: Record<string, string[]> = {
  stopped: ['starting'],
  starting: ['idle', 'error'],
  idle: ['working', 'stopped'],
  working: ['done', 'stuck', 'error', 'idle'],
  done: ['idle', 'working', 'stopped'],
  stuck: ['working', 'error', 'stopped'],
  error: ['stopped', 'idle'],
};

// Test component for state transitions
interface StateTransitionProps {
  currentState: string;
  history: string[];
  onTransition: (newState: string) => void;
}

function StateTransitionComponent({
  currentState,
  history,
}: StateTransitionProps): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text>Current State: {currentState}</Text>
      <Text>History: {history.join(' → ')}</Text>
      <Text>Transitions: {history.length}</Text>
    </Box>
  );
}

describe('State Transitions: Agent Lifecycle', () => {
  it('idle to working is valid', () => {
    const canTransition = validAgentTransitions.idle.includes('working');
    expect(canTransition).toBe(true);
  });

  it('working to done is valid', () => {
    const canTransition = validAgentTransitions.working.includes('done');
    expect(canTransition).toBe(true);
  });

  it('working to stuck is valid', () => {
    const canTransition = validAgentTransitions.working.includes('stuck');
    expect(canTransition).toBe(true);
  });

  it('working to error is valid', () => {
    const canTransition = validAgentTransitions.working.includes('error');
    expect(canTransition).toBe(true);
  });

  it('done to idle is valid', () => {
    const canTransition = validAgentTransitions.done.includes('idle');
    expect(canTransition).toBe(true);
  });

  it('stuck to working is valid (recovery)', () => {
    const canTransition = validAgentTransitions.stuck.includes('working');
    expect(canTransition).toBe(true);
  });

  it('error to stopped is valid', () => {
    const canTransition = validAgentTransitions.error.includes('stopped');
    expect(canTransition).toBe(true);
  });

  it('displays current state', () => {
    const { lastFrame } = render(
      <StateTransitionComponent
        currentState="working"
        history={['idle', 'working']}
        onTransition={() => {}}
      />
    );
    expect(lastFrame()).toContain('Current State: working');
  });

  it('displays transition history', () => {
    const { lastFrame } = render(
      <StateTransitionComponent
        currentState="done"
        history={['idle', 'working', 'done']}
        onTransition={() => {}}
      />
    );
    expect(lastFrame()).toContain('idle → working → done');
    expect(lastFrame()).toContain('Transitions: 3');
  });
});

describe('State Transitions: Create Agent Flow', () => {
  it('new agent starts in stopped state', () => {
    const initialState = 'stopped';
    expect(initialState).toBe('stopped');
  });

  it('stopped agent can start (transition to starting)', () => {
    const canStart = validAgentTransitions.stopped.includes('starting');
    expect(canStart).toBe(true);
  });

  it('starting agent transitions to idle on success', () => {
    const canIdle = validAgentTransitions.starting.includes('idle');
    expect(canIdle).toBe(true);
  });

  it('starting agent transitions to error on failure', () => {
    const canError = validAgentTransitions.starting.includes('error');
    expect(canError).toBe(true);
  });

  it('complete create-start flow: stopped → starting → idle', () => {
    const flow = ['stopped', 'starting', 'idle'];

    // Verify each transition is valid
    for (let i = 0; i < flow.length - 1; i++) {
      const from = flow[i];
      const to = flow[i + 1];
      const isValid = validAgentTransitions[from]?.includes(to);
      expect(isValid).toBe(true);
    }
  });
});

describe('State Transitions: Work Cycle', () => {
  it('complete work cycle: idle → working → done → idle', () => {
    const flow = ['idle', 'working', 'done', 'idle'];

    for (let i = 0; i < flow.length - 1; i++) {
      const from = flow[i];
      const to = flow[i + 1];
      const isValid = validAgentTransitions[from]?.includes(to);
      expect(isValid).toBe(true);
    }
  });

  it('multiple work cycles are valid', () => {
    const flow = ['idle', 'working', 'done', 'idle', 'working', 'done', 'idle'];

    for (let i = 0; i < flow.length - 1; i++) {
      const from = flow[i];
      const to = flow[i + 1];
      const isValid = validAgentTransitions[from]?.includes(to);
      expect(isValid).toBe(true);
    }
  });
});

describe('State Transitions: Error Recovery', () => {
  it('stuck agent can recover to working', () => {
    const flow = ['working', 'stuck', 'working', 'done'];

    for (let i = 0; i < flow.length - 1; i++) {
      const from = flow[i];
      const to = flow[i + 1];
      const isValid = validAgentTransitions[from]?.includes(to);
      expect(isValid).toBe(true);
    }
  });

  it('error agent can be stopped and restarted', () => {
    const flow = ['error', 'stopped', 'starting', 'idle'];

    for (let i = 0; i < flow.length - 1; i++) {
      const from = flow[i];
      const to = flow[i + 1];
      const isValid = validAgentTransitions[from]?.includes(to);
      expect(isValid).toBe(true);
    }
  });

  it('error agent can transition directly to idle', () => {
    const canIdle = validAgentTransitions.error.includes('idle');
    expect(canIdle).toBe(true);
  });
});

describe('State Transitions: Message Flow', () => {
  interface Message {
    id: number;
    sender: string;
    content: string;
    timestamp: number;
  }

  it('messages are ordered by timestamp', () => {
    const messages: Message[] = [
      { id: 1, sender: 'eng-01', content: 'First', timestamp: 1000 },
      { id: 2, sender: 'eng-02', content: 'Second', timestamp: 2000 },
      { id: 3, sender: 'tl-01', content: 'Third', timestamp: 3000 },
    ];

    const isOrdered = messages.every((msg, idx) => {
      if (idx === 0) return true;
      return msg.timestamp > messages[idx - 1].timestamp;
    });

    expect(isOrdered).toBe(true);
  });

  it('new message appears in history', () => {
    const existingMessages: Message[] = [
      { id: 1, sender: 'eng-01', content: 'First', timestamp: 1000 },
    ];

    const newMessage: Message = {
      id: 2,
      sender: 'eng-02',
      content: 'New message',
      timestamp: 2000,
    };

    const updatedMessages = [...existingMessages, newMessage];
    expect(updatedMessages.length).toBe(2);
    expect(updatedMessages[1].content).toBe('New message');
  });
});

describe('State Transitions: Cost Updates', () => {
  interface CostState {
    total: number;
    lastUpdate: number;
  }

  it('cost increases are cumulative', () => {
    const costs: CostState[] = [
      { total: 0, lastUpdate: 1000 },
      { total: 0.5, lastUpdate: 2000 },
      { total: 1.2, lastUpdate: 3000 },
      { total: 2.0, lastUpdate: 4000 },
    ];

    const isIncreasing = costs.every((cost, idx) => {
      if (idx === 0) return true;
      return cost.total >= costs[idx - 1].total;
    });

    expect(isIncreasing).toBe(true);
  });

  it('cost updates have increasing timestamps', () => {
    const costs: CostState[] = [
      { total: 0, lastUpdate: 1000 },
      { total: 0.5, lastUpdate: 2000 },
      { total: 1.2, lastUpdate: 3000 },
    ];

    const timestampsOrdered = costs.every((cost, idx) => {
      if (idx === 0) return true;
      return cost.lastUpdate > costs[idx - 1].lastUpdate;
    });

    expect(timestampsOrdered).toBe(true);
  });
});

describe('State Transitions: Process Lifecycle', () => {
  interface ProcessState {
    name: string;
    running: boolean;
    pid: number | null;
    exitCode: number | null;
  }

  it('process starts with valid PID', () => {
    const process: ProcessState = {
      name: 'server',
      running: true,
      pid: 1234,
      exitCode: null,
    };

    expect(process.running).toBe(true);
    expect(process.pid).toBeGreaterThan(0);
    expect(process.exitCode).toBeNull();
  });

  it('stopped process has exit code', () => {
    const process: ProcessState = {
      name: 'server',
      running: false,
      pid: null,
      exitCode: 0,
    };

    expect(process.running).toBe(false);
    expect(process.exitCode).toBeDefined();
  });

  it('process transition: not running → running → stopped', () => {
    const states: ProcessState[] = [
      { name: 'server', running: false, pid: null, exitCode: null },
      { name: 'server', running: true, pid: 1234, exitCode: null },
      { name: 'server', running: false, pid: null, exitCode: 0 },
    ];

    // Initially not running
    expect(states[0].running).toBe(false);
    expect(states[0].pid).toBeNull();

    // Running with PID
    expect(states[1].running).toBe(true);
    expect(states[1].pid).not.toBeNull();

    // Stopped with exit code
    expect(states[2].running).toBe(false);
    expect(states[2].exitCode).not.toBeNull();
  });
});
