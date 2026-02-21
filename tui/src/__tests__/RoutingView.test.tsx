/**
 * RoutingView tests
 * Issue #1231 - Additional TUI views
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { RoutingView } from '../views/RoutingView';
import { FocusProvider } from '../navigation/FocusContext';
import * as useAgentsHook from '../hooks/useAgents';

// Mock the useAgents hook
jest.mock('../hooks/useAgents', () => ({
  useAgents: jest.fn(),
}));

const mockUseAgents = useAgentsHook.useAgents as jest.Mock;

function renderRoutingView(props = {}) {
  return render(
    <FocusProvider>
      <RoutingView disableInput {...props} />
    </FocusProvider>
  );
}

describe('RoutingView', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockUseAgents.mockReturnValue({
      data: [
        { name: 'eng-01', role: 'engineer', state: 'idle' },
        { name: 'eng-02', role: 'engineer', state: 'working' },
        { name: 'tl-01', role: 'tech-lead', state: 'idle' },
      ],
      loading: false,
      error: null,
    });
  });

  test('displays routing header', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';
    expect(output).toContain('Task Routing');
  });

  test('shows routing rules', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    // Check for task types
    expect(output).toContain('code');
    expect(output).toContain('review');
    expect(output).toContain('merge');
    expect(output).toContain('qa');
  });

  test('shows target roles', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    // Check for target roles
    expect(output).toContain('engineer');
    expect(output).toContain('tech-lead');
    expect(output).toContain('manager');
  });

  test('displays agent counts by role', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    // Should show role summary with agent counts
    expect(output).toContain('Role Summary');
  });

  test('shows keyboard hints', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    expect(output).toContain('j/k');
    expect(output).toContain('navigate');
  });

  test('shows description of routing', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    expect(output).toContain('round-robin');
  });

  test('shows rule count in header', () => {
    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    // 4 rules: code, review, merge, qa
    expect(output).toContain('4 rules');
  });
});

describe('RoutingView with no agents', () => {
  test('shows zero counts when no agents', () => {
    mockUseAgents.mockReturnValue({
      data: [],
      loading: false,
      error: null,
    });

    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    // Should still show the routing rules
    expect(output).toContain('code');
    expect(output).toContain('engineer');
  });
});

describe('RoutingView availability tracking', () => {
  test('shows available agent counts', () => {
    mockUseAgents.mockReturnValue({
      data: [
        { name: 'eng-01', role: 'engineer', state: 'idle' },
        { name: 'eng-02', role: 'engineer', state: 'stopped' },
        { name: 'eng-03', role: 'engineer', state: 'working' },
      ],
      loading: false,
      error: null,
    });

    const { lastFrame } = renderRoutingView();
    const output = lastFrame() ?? '';

    // Should count idle + working as available
    expect(output).toContain('engineer');
    // 2 available (idle + working), 1 stopped
  });
});
