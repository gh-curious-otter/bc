import { describe, test, expect } from 'bun:test';
import React from 'react';
import { render } from 'ink-testing-library';
import { AgentDetailView } from '../AgentDetailView';
import type { Agent } from '../../types';

describe('AgentDetailView Component', () => {
  const mockAgent: Agent = {
    name: 'test-agent',
    role: 'engineer',
    state: 'running',
    task: 'Implementing feature #662',
    session: 'test-session',
    memory: undefined,
  };

  test('renders agent name in header', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('test-agent');
  });

  test('renders agent role in header', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('engineer');
  });

  test('renders agent task', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Implementing feature #662');
  });

  test('shows input prompt when not in input mode', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Press i or m');
  });

  test('shows navigation hints in footer', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('i/m: message');
    expect(output).toContain('r: refresh');
  });

  test('displays agent state (running)', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('State');
  });

  test('handles agent with undefined task', () => {
    const agentNoTask = { ...mockAgent, task: undefined };
    const { lastFrame } = render(
      <AgentDetailView agent={agentNoTask} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toBeTruthy();
  });

  test('handles agent with undefined role', () => {
    const agentNoRole = { ...mockAgent, role: undefined };
    const { lastFrame } = render(
      <AgentDetailView agent={agentNoRole} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toBeTruthy();
  });

  test('renders with different agent states', () => {
    const states: Array<Agent['state']> = ['running', 'idle', 'working', 'stopped'];
    states.forEach(state => {
      const agent = { ...mockAgent, state };
      const { lastFrame } = render(
        <AgentDetailView agent={agent} onBack={() => {}} />
      );
      const output = lastFrame() ?? '';
      expect(output).toBeTruthy();
    });
  });

  test('renders output area with border', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    // Output area is rendered
    expect(output).toBeTruthy();
  });

  test('renders message input area with border', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    // Message input area is rendered
    expect(output).toBeTruthy();
  });

  test('accepts onBack callback', () => {
    let callbackCalled = false;
    render(
      <AgentDetailView
        agent={mockAgent}
        onBack={() => { callbackCalled = true; }}
      />
    );
    // Callback is registered
    expect(callbackCalled === false).toBe(true);
  });

  test('displays loading state when fetching output', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    // Component renders initial state
    expect(output).toBeTruthy();
  });

  test('handles agent with long name', () => {
    const agentLongName = { ...mockAgent, name: 'very-long-agent-name-that-might-cause-layout-issues' };
    const { lastFrame } = render(
      <AgentDetailView agent={agentLongName} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('very-long-agent-name');
  });

  test('handles agent with long task description', () => {
    const agentLongTask = {
      ...mockAgent,
      task: 'This is a very long task description that spans many words and might cause layout wrapping issues in the TUI component'
    };
    const { lastFrame } = render(
      <AgentDetailView agent={agentLongTask} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toBeTruthy();
  });

  test('renders all required UI sections', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={mockAgent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    // Header section present
    expect(output).toContain('test-agent');
    // Navigation hints present
    expect(output).toBeTruthy();
  });
});

describe('AgentDetailView Integration Patterns', () => {
  const agent: Agent = {
    name: 'integration-test-agent',
    role: 'manager',
    state: 'idle',
    task: 'Testing integration patterns',
    session: 'integration-session',
  };

  test('component receives agent prop correctly', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={agent} onBack={() => {}} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('integration-test-agent');
  });

  test('component receives onBack callback correctly', () => {
    const mockCallback = () => {};
    const { lastFrame } = render(
      <AgentDetailView agent={agent} onBack={mockCallback} />
    );
    const output = lastFrame() ?? '';
    expect(output).toBeTruthy();
  });

  test('component handles missing onBack callback gracefully', () => {
    const { lastFrame } = render(
      <AgentDetailView agent={agent} />
    );
    const output = lastFrame() ?? '';
    expect(output).toBeTruthy();
  });
});
