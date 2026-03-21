import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { Text, Box } from 'ink';
import { ThemeProvider } from '../theme/ThemeContext';
import { FocusProvider } from '../navigation/FocusContext';
import { useListNavigation } from '../hooks/useListNavigation';

const renderWithProviders = (ui: React.ReactElement) => render(
  <ThemeProvider><FocusProvider>{ui}</FocusProvider></ThemeProvider>
);

// Test component that uses the hook
function TestList({
  items,
  onSelect,
}: {
  items: string[];
  onSelect?: (item: string, index: number) => void;
}): React.ReactElement {
  const { selectedIndex, isSelected } = useListNavigation({
    items,
    onSelect,
  });

  return (
    <Box flexDirection="column">
      {items.map((item, index) => (
        <Text key={item} color={isSelected(index) ? 'green' : undefined}>
          {isSelected(index) ? '> ' : '  '}
          {item}
        </Text>
      ))}
      <Text>Selected: {selectedIndex}</Text>
    </Box>
  );
}

describe('useListNavigation', () => {
  test('renders list with first item selected by default', () => {
    const { lastFrame } = renderWithProviders(<TestList items={['Item 1', 'Item 2', 'Item 3']} />);
    const output = lastFrame() ?? '';

    expect(output).toContain('> Item 1');
    expect(output).toContain('Selected: 0');
  });

  test('handles empty list gracefully', () => {
    const { lastFrame } = renderWithProviders(<TestList items={[]} />);
    const output = lastFrame() ?? '';

    expect(output).toContain('Selected: 0');
  });

  test('shows selection indicator', () => {
    const { lastFrame } = renderWithProviders(<TestList items={['A', 'B', 'C']} />);
    const output = lastFrame() ?? '';

    // First item should have selection indicator
    expect(output).toContain('> A');
    // Other items should not
    expect(output).toContain('  B');
    expect(output).toContain('  C');
  });
});
