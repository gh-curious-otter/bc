import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { App } from '../app.js';

describe('App', () => {
  test('renders without crashing', () => {
    const { lastFrame } = render(<App />);
    expect(lastFrame()).toBeDefined();
  });

  test('shows bc header', () => {
    const { lastFrame } = render(<App />);
    const output = lastFrame() ?? '';
    expect(output).toContain('bc');
  });

  test('shows navigation tabs', () => {
    const { lastFrame } = render(<App />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Dashboard');
    expect(output).toContain('Agents');
    expect(output).toContain('Channels');
    expect(output).toContain('Costs');
  });

  test('shows help hint in footer', () => {
    const { lastFrame } = render(<App />);
    const output = lastFrame() ?? '';
    expect(output).toContain('[?] for help');
    expect(output).toContain('[q] to quit');
  });

  test('starts on dashboard view', () => {
    const { lastFrame } = render(<App />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Dashboard');
  });
});
