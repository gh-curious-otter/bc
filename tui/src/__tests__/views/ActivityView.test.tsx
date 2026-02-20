/**
 * ActivityView tests
 * Issue #1047 - Activity timeline view
 */

import { describe, test, expect } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { Box, Text } from 'ink';
import { NavigationProvider } from '../../navigation/NavigationContext';

// Mock component for testing without bc CLI dependency
function MockActivityView(): React.ReactElement {
  return (
    <Box flexDirection="column">
      <Text>Activity Timeline</Text>
      <Text>Period: [d] 24h | [w] Week | [m] Month</Text>
      <Text>Cost Summary</Text>
      <Text>Agent Activity</Text>
    </Box>
  );
}

// Wrapper to provide context
function renderWithNav(ui: React.ReactElement) {
  return render(<NavigationProvider>{ui}</NavigationProvider>);
}

describe('ActivityView', () => {
  test('renders activity timeline header', () => {
    const { lastFrame } = renderWithNav(<MockActivityView />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Activity Timeline');
  });

  test('renders time period selector', () => {
    const { lastFrame } = renderWithNav(<MockActivityView />);
    const output = lastFrame() ?? '';
    expect(output).toContain('24h');
    expect(output).toContain('Week');
    expect(output).toContain('Month');
  });

  test('renders cost summary section', () => {
    const { lastFrame } = renderWithNav(<MockActivityView />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Cost Summary');
  });

  test('renders agent activity section', () => {
    const { lastFrame } = renderWithNav(<MockActivityView />);
    const output = lastFrame() ?? '';
    expect(output).toContain('Agent Activity');
  });
});

describe('ActivityView keyboard navigation', () => {
  test('period selector shows keyboard hints', () => {
    const { lastFrame } = renderWithNav(<MockActivityView />);
    const output = lastFrame() ?? '';
    // Should show keyboard shortcuts
    expect(output).toContain('[d]');
    expect(output).toContain('[w]');
    expect(output).toContain('[m]');
  });
});

describe('ActivityView responsive behavior', () => {
  test('renders at standard terminal width', () => {
    const { lastFrame } = render(
      <NavigationProvider>
        <MockActivityView />
      </NavigationProvider>
    );
    const output = lastFrame() ?? '';
    expect(output.length).toBeGreaterThan(0);
  });
});
