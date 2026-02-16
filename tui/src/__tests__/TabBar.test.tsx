/**
 * TabBar responsive display mode tests for #982
 *
 * Verifies:
 * - Display mode logic returns correct mode for terminal widths
 * - TabBar renders properly with terminalWidth prop controlling display mode
 */

import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { TabBar } from '../navigation/TabBar';
import { NavigationProvider } from '../navigation/NavigationContext';

/** Wrapper to provide navigation context */
function renderTabBar(terminalWidth: number) {
  return render(
    <NavigationProvider>
      <TabBar terminalWidth={terminalWidth} />
    </NavigationProvider>
  );
}

describe('TabBar display mode logic', () => {
  // Note: ink-testing-library renders at fixed 80 cols, so we test the logic
  // indirectly by checking that the terminalWidth prop affects what labels are used

  test('at 140 cols (full mode), uses full label "Dashboard"', () => {
    const { lastFrame } = renderTabBar(140);
    const output = lastFrame() ?? '';

    // Full mode shows full labels - but rendering may truncate at 80 cols
    // We verify by checking that Dashboard is attempted (even if truncated)
    expect(output).toContain('Dashboar'); // May be truncated by 80-col renderer
    expect(output).toContain('[1]');
  });

  test('at 100 cols (short mode), uses short labels', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // Short mode uses abbreviated labels
    expect(output).toContain('Dash');
    expect(output).toContain('Agt');
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
  });

  test('at 60 cols (minimal mode), shows keys only', () => {
    const { lastFrame } = renderTabBar(60);
    const output = lastFrame() ?? '';

    // Minimal mode shows only keys, no labels
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).toContain('[3]');
    expect(output).toContain('[?]');

    // Should NOT contain any labels (not even short ones)
    expect(output).not.toContain('Dash');
    expect(output).not.toContain('Agt');
  });

  test('boundary: 120 cols triggers full mode', () => {
    const { lastFrame } = renderTabBar(120);
    const output = lastFrame() ?? '';

    // At exactly 120, should be full mode
    // Check for "Dashboar" which indicates full "Dashboard" was attempted
    expect(output).toContain('Dashboar');
  });

  test('boundary: 119 cols triggers short mode', () => {
    const { lastFrame } = renderTabBar(119);
    const output = lastFrame() ?? '';

    // At 119, should be short mode - look for short labels
    expect(output).toContain('Dash');
    // Full "Dashboard" should NOT appear (even truncated would be different)
    expect(output).not.toContain('Dashboar');
  });

  test('boundary: 80 cols triggers short mode', () => {
    const { lastFrame } = renderTabBar(80);
    const output = lastFrame() ?? '';

    // At 80, still short mode
    expect(output).toContain('Dash');
  });

  test('boundary: 79 cols triggers minimal mode', () => {
    const { lastFrame } = renderTabBar(79);
    const output = lastFrame() ?? '';

    // At 79, minimal mode - no labels
    expect(output).toContain('[1]');
    expect(output).not.toContain('Dash');
  });
});

describe('TabBar structure', () => {
  test('renders title "bc" prefix', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // Title should be visible
    expect(output).toContain('bc');
    expect(output).toContain('|');
  });

  test('all tab keys are present at every display mode', () => {
    const keys = ['[1]', '[2]', '[3]', '[4]', '[5]', '[6]', '[7]', '[8]', '[?]'];

    // Test each display mode
    for (const width of [60, 100, 140]) {
      const { lastFrame } = renderTabBar(width);
      const output = lastFrame() ?? '';

      for (const key of keys) {
        expect(output).toContain(key);
      }
    }
  });

  test('short labels map correctly', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // Verify short labels are used in short mode
    expect(output).toContain('[1] Dash');
    expect(output).toContain('[2] Agt');
    expect(output).toContain('[3] Chan');
    expect(output).toContain('[4] Cost');
    expect(output).toContain('[5] Cmd');
    expect(output).toContain('[6] Role');
    expect(output).toContain('[7] Log');
    expect(output).toContain('[8] Tree');
  });
});

describe('TabBar accessibility', () => {
  test('keyboard navigation keys always visible', () => {
    // Even in minimal mode, all keys should be visible for keyboard navigation
    const { lastFrame } = renderTabBar(60);
    const output = lastFrame() ?? '';

    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).toContain('[3]');
    expect(output).toContain('[4]');
    expect(output).toContain('[5]');
    expect(output).toContain('[6]');
    expect(output).toContain('[7]');
    expect(output).toContain('[8]');
    expect(output).toContain('[?]');
  });
});
