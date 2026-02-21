/**
 * HeaderBar component tests
 * Issue #1419: TUI Production Polish - Consistent headers
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { HeaderBar } from '../components/HeaderBar';

describe('HeaderBar', () => {
  it('renders title', () => {
    const { lastFrame } = render(<HeaderBar title="Test Title" />);
    expect(lastFrame()).toContain('Test Title');
  });

  it('renders count badge', () => {
    const { lastFrame } = render(<HeaderBar title="Items" count={42} />);
    expect(lastFrame()).toContain('Items');
    expect(lastFrame()).toContain('(42)');
  });

  it('renders subtitle', () => {
    const { lastFrame } = render(
      <HeaderBar title="Main" subtitle="Description here" />
    );
    expect(lastFrame()).toContain('Main');
    expect(lastFrame()).toContain('Description here');
  });

  it('renders keyboard hints', () => {
    const { lastFrame } = render(
      <HeaderBar title="View" hints="Press q to quit" />
    );
    expect(lastFrame()).toContain('Press q to quit');
  });

  it('renders loading indicator when loading', () => {
    const { lastFrame } = render(<HeaderBar title="Loading" loading />);
    // LoadingIndicator renders animated spinner
    expect(lastFrame()).toBeTruthy();
  });

  it('supports different colors', () => {
    const { lastFrame } = render(<HeaderBar title="Blue" color="blue" />);
    expect(lastFrame()).toContain('Blue');
  });

  it('renders all props together', () => {
    const { lastFrame } = render(
      <HeaderBar
        title="Agents"
        subtitle="Manage AI agents"
        count={5}
        hints="j/k to navigate"
        color="cyan"
      />
    );
    const frame = lastFrame();
    expect(frame).toContain('Agents');
    expect(frame).toContain('(5)');
    expect(frame).toContain('Manage AI agents');
    expect(frame).toContain('j/k to navigate');
  });
});
