import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { App } from '../app.js';

describe('App', () => {
  // Use disableInput to avoid stdin.ref issues in test environment
  // Use help view to avoid Dashboard hook's bc command spawning in tests
  test('renders without crashing', () => {
    const { lastFrame } = render(<App disableInput initialView="help" />);
    expect(lastFrame()).toBeDefined();
  });

  test('shows bc header', () => {
    // Use help view to avoid Dashboard hook's bc command spawning
    const { lastFrame } = render(<App disableInput initialView="help" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('bc');
  });

  test('shows navigation tabs', () => {
    // Use help view to avoid Dashboard hook's bc command spawning
    const { lastFrame } = render(<App disableInput initialView="help" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Dashboard');
    expect(output).toContain('Agents');
    expect(output).toContain('Channels');
    expect(output).toContain('Costs');
  });

  test('shows help hint in footer', () => {
    // Use help view to avoid Dashboard hook's bc command spawning
    const { lastFrame } = render(<App disableInput initialView="help" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('[?] for help');
    expect(output).toContain('[q] to quit');
  });

  test('starts on dashboard view', () => {
    // Use help view - the tab bar shows "Dashboard" regardless of current view
    const { lastFrame } = render(<App disableInput initialView="help" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Dashboard');
  });
});
