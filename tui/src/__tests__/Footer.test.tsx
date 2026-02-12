import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { Footer, KeyHint } from '../components/Footer';

describe('KeyHint', () => {
  test('renders key and label', () => {
    const { lastFrame } = render(<KeyHint keyChar="q" label="quit" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('[');
    expect(output).toContain('q');
    expect(output).toContain(']');
    expect(output).toContain('quit');
  });

  test('renders with special keys', () => {
    const { lastFrame } = render(<KeyHint keyChar="?" label="help" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('?');
    expect(output).toContain('help');
  });
});

describe('Footer', () => {
  test('renders multiple hints', () => {
    const hints = [
      { key: 'j', label: 'down' },
      { key: 'k', label: 'up' },
      { key: 'q', label: 'quit' },
    ];
    const { lastFrame } = render(<Footer hints={hints} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('j');
    expect(output).toContain('down');
    expect(output).toContain('k');
    expect(output).toContain('up');
    expect(output).toContain('q');
    expect(output).toContain('quit');
  });

  test('renders empty hints array', () => {
    const { lastFrame } = render(<Footer hints={[]} />);
    expect(lastFrame()).toBeDefined();
  });

  test('renders single hint', () => {
    const hints = [{ key: 'r', label: 'refresh' }];
    const { lastFrame } = render(<Footer hints={hints} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('r');
    expect(output).toContain('refresh');
  });
});
