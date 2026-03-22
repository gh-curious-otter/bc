/**
 * Tests for useHintsContext hook
 * Issue #1461: Fix duplicate keyboard hints
 */
import { describe, expect, test } from 'bun:test';
import { render, cleanup } from 'ink-testing-library';
import React, { useEffect } from 'react';
import { Text, Box } from 'ink';
import { HintsProvider, useHintsContext, useViewHints } from '../hooks/useHintsContext';
import type { HintItem } from '../components/Footer';

// Helper to wait for React state updates
const wait = (ms: number): Promise<void> => new Promise((r) => setTimeout(r, ms));

// Test component that displays current hints
function HintsDisplay(): React.ReactElement {
  const { viewHints } = useHintsContext();
  return (
    <Box flexDirection="column">
      <Text>Hints: {viewHints.length}</Text>
      {viewHints.map((hint) => (
        <Text key={hint.key}>
          [{hint.key}] {hint.label}
        </Text>
      ))}
    </Box>
  );
}

// Test component that sets hints
function HintsSetter({ hints }: { hints: HintItem[] }): React.ReactElement {
  const { setViewHints } = useHintsContext();

  useEffect(() => {
    setViewHints(hints);
  }, [hints, setViewHints]);

  return <Text>Setter active</Text>;
}

// Test component using useViewHints hook
function ViewWithHints({ hints }: { hints: HintItem[] }): React.ReactElement {
  useViewHints(hints);
  return <Text>View active</Text>;
}

describe('HintsContext', () => {
  test('renders with empty hints by default', () => {
    const { lastFrame } = render(
      <HintsProvider>
        <HintsDisplay />
      </HintsProvider>
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Hints: 0');
    cleanup();
  });

  test('sets view hints via context', async () => {
    const hints: HintItem[] = [
      { key: 'j/k', label: 'navigate' },
      { key: 'Enter', label: 'select' },
    ];

    const { lastFrame } = render(
      <HintsProvider>
        <HintsSetter hints={hints} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    const output = lastFrame() ?? '';

    expect(output).toContain('Hints: 2');
    expect(output).toContain('[j/k] navigate');
    expect(output).toContain('[Enter] select');
    cleanup();
  });

  test('clears hints when setViewHints called with empty array', async () => {
    const { lastFrame, rerender } = render(
      <HintsProvider>
        <HintsSetter hints={[{ key: 'a', label: 'action' }]} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    let output = lastFrame() ?? '';
    expect(output).toContain('Hints: 1');

    rerender(
      <HintsProvider>
        <HintsSetter hints={[]} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    output = lastFrame() ?? '';
    expect(output).toContain('Hints: 0');
    cleanup();
  });

  test('works without provider (returns default noop context)', () => {
    // Should not throw when provider is not present
    const { lastFrame } = render(<HintsDisplay />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Hints: 0');
    cleanup();
  });
});

describe('useViewHints', () => {
  test('sets hints on mount', async () => {
    const hints: HintItem[] = [{ key: 'q', label: 'quit' }];

    const { lastFrame } = render(
      <HintsProvider>
        <ViewWithHints hints={hints} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    const output = lastFrame() ?? '';

    expect(output).toContain('[q] quit');
    cleanup();
  });

  test('clears hints on unmount', async () => {
    const hints: HintItem[] = [{ key: 'x', label: 'delete' }];

    const { lastFrame, rerender } = render(
      <HintsProvider>
        <ViewWithHints hints={hints} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    let output = lastFrame() ?? '';
    expect(output).toContain('[x] delete');

    // Remove ViewWithHints to trigger unmount
    rerender(
      <HintsProvider>
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    output = lastFrame() ?? '';
    expect(output).toContain('Hints: 0');
    cleanup();
  });

  test('updates hints when prop changes', async () => {
    const { lastFrame, rerender } = render(
      <HintsProvider>
        <ViewWithHints hints={[{ key: 'a', label: 'first' }]} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    let output = lastFrame() ?? '';
    expect(output).toContain('[a] first');

    rerender(
      <HintsProvider>
        <ViewWithHints hints={[{ key: 'b', label: 'second' }]} />
        <HintsDisplay />
      </HintsProvider>
    );

    await wait(50);
    output = lastFrame() ?? '';
    expect(output).toContain('[b] second');
    cleanup();
  });
});
