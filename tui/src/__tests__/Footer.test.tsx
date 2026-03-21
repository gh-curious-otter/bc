import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { ThemeProvider } from '../theme/ThemeContext';
import { Footer, KeyHint } from '../components/Footer';

const renderWithTheme = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);

describe('KeyHint', () => {
  test('renders key and label', () => {
    const { lastFrame } = renderWithTheme(<KeyHint keyChar="q" label="quit" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('[');
    expect(output).toContain('q');
    expect(output).toContain(']');
    expect(output).toContain('quit');
  });

  test('renders with special keys', () => {
    const { lastFrame } = renderWithTheme(<KeyHint keyChar="?" label="help" />);
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
    const { lastFrame } = renderWithTheme(<Footer hints={hints} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('j');
    expect(output).toContain('down');
    expect(output).toContain('k');
    expect(output).toContain('up');
    expect(output).toContain('q');
    expect(output).toContain('quit');
  });

  test('renders empty hints array', () => {
    const { lastFrame } = renderWithTheme(<Footer hints={[]} />);
    expect(lastFrame()).toBeDefined();
  });

  test('renders single hint', () => {
    const hints = [{ key: 'r', label: 'refresh' }];
    const { lastFrame } = renderWithTheme(<Footer hints={hints} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('r');
    expect(output).toContain('refresh');
  });
});
