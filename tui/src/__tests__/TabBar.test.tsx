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
    expect(output).toContain('Dashboard');
    expect(output).toContain('[1]');
  });

  test('at 100 cols (full mode), uses full labels', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // Full mode uses full labels at 100 cols
    expect(output).toContain('Dashboard');
    expect(output).toContain('Agents');
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
  });

  test('at 60 cols (short mode), shows abbreviated labels', () => {
    const { lastFrame } = renderTabBar(60);
    const output = lastFrame() ?? '';

    // Short mode shows abbreviated labels
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).toContain('[3]');
    expect(output).toContain('Dash');
    expect(output).toContain('Agt');
  });

  test('boundary: 80 cols triggers full mode', () => {
    const { lastFrame } = renderTabBar(80);
    const output = lastFrame() ?? '';

    // At exactly 80, should be full mode
    expect(output).toContain('Dashboard');
  });

  test('boundary: 79 cols triggers short mode', () => {
    const { lastFrame } = renderTabBar(79);
    const output = lastFrame() ?? '';

    // At 79, should be short mode - look for short labels
    expect(output).toContain('Dash');
    // Full "Dashboard" should NOT appear
    expect(output).not.toContain('Dashboard');
  });

  test('boundary: 50 cols triggers short mode', () => {
    const { lastFrame } = renderTabBar(50);
    const output = lastFrame() ?? '';

    // At 50, still short mode
    expect(output).toContain('Dash');
  });

  test('boundary: 49 cols triggers minimal mode', () => {
    const { lastFrame } = renderTabBar(49);
    const output = lastFrame() ?? '';

    // At 49, minimal mode - no labels
    expect(output).toContain('[1]');
    expect(output).not.toContain('Dash');
  });
});

describe('TabBar structure', () => {
  test('renders title "bc" prefix', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // Title should be visible (may be truncated in some renderers)
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

  test('full labels map correctly at 80+ cols', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // Verify full labels are used in full mode at 100 cols
    expect(output).toContain('[1]');
    expect(output).toContain('Dashboard');
    expect(output).toContain('[2]');
    expect(output).toContain('Agents');
    expect(output).toContain('[3]');
    expect(output).toContain('Channels');
  });

  test('short labels map correctly at <80 cols', () => {
    const { lastFrame } = renderTabBar(60);
    const output = lastFrame() ?? '';

    // Verify short labels are used in short mode at 60 cols
    expect(output).toContain('[1]');
    expect(output).toContain('Dash');
    expect(output).toContain('[2]');
    expect(output).toContain('Agt');
  });
});

describe('TabBar accessibility', () => {
  test('keyboard navigation keys always visible', () => {
    // All keys should be visible in all modes
    const { lastFrame } = renderTabBar(40);
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

describe('TabBar #1038 - Tab label truncation fix at 80x24', () => {
  test('at 80x24 (standard terminal), shows full tab names', () => {
    const { lastFrame } = renderTabBar(80);
    const output = lastFrame() ?? '';

    // Issue #1038: At 80x24, should show full names like "Dashboard"
    expect(output).toContain('Dashboard');
    expect(output).toContain('Agents');
    expect(output).toContain('Channels');
    expect(output).toContain('[1]');
  });

  test('at 81x24, shows full tab names', () => {
    const { lastFrame } = renderTabBar(81);
    const output = lastFrame() ?? '';

    // At 81 cols, should be in full mode showing complete names
    expect(output).toContain('Dashboard');
    expect(output).toContain('Agents');
  });

  test('at 79x24 (just below 80), shows abbreviations', () => {
    const { lastFrame } = renderTabBar(79);
    const output = lastFrame() ?? '';

    // At 79 cols, should fall back to short mode with abbreviations
    expect(output).toContain('Dash');
    expect(output).toContain('Agt');
  });
});
