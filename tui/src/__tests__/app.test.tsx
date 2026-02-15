import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { App } from '../app.js';

describe('App', () => {
  // Use disableInput to avoid stdin.ref issues in test environment
  // However, some components still require TTY for proper rendering
  test('renders without crashing', () => {
    const { lastFrame } = render(<App disableInput />);
    expect(lastFrame()).toBeDefined();
  });

  // Note: The following tests require TTY stdin which is not available in CI
  // They validate App rendering but useInput hook initialization needs proper stdin
  test.skip('shows bc header', () => {
    const { lastFrame } = render(<App disableInput />);
    const output = lastFrame() ?? '';
    expect(output).toContain('bc');
  });

  test.skip('shows navigation tabs', () => {
    const { lastFrame } = render(<App disableInput />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Dashboard');
    expect(output).toContain('Agents');
    expect(output).toContain('Channels');
    expect(output).toContain('Costs');
  });

  test.skip('shows help hint in footer', () => {
    const { lastFrame } = render(<App disableInput />);
    const output = lastFrame() ?? '';
    expect(output).toContain('[?] for help');
    expect(output).toContain('[q] to quit');
  });

  test.skip('starts on dashboard view', () => {
    const { lastFrame } = render(<App disableInput />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Dashboard');
  });
});
