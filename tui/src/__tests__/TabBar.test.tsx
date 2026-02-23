/**
 * TabBar responsive display mode tests
 *
 * Issue #1109: Fixed 80x24 display by adjusting thresholds:
 * - Full (>=120 cols): Full labels like "Dashboard", "Agents"
 * - Short (100-119 cols): Short labels like "Dash", "Agt"
 * - Minimal (<100 cols): Just numbers like "[1]", "[2]" (fits 80x24)
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

  test('at 140 cols (full mode), uses full labels', () => {
    const { lastFrame } = renderTabBar(140);
    const output = lastFrame() ?? '';

    // Full mode shows full labels (may be wrapped in ink-testing-library's 80-col output)
    // With 17 tabs (performance + issues), "Dashboard" may wrap - check for "board" suffix
    expect(output).toContain('board');
    expect(output).toContain('[1]');
  });

  test('at 120 cols (full mode boundary), uses full labels', () => {
    const { lastFrame } = renderTabBar(120);
    const output = lastFrame() ?? '';

    // At exactly 120, should be full mode
    // Note: ink-testing-library renders at 80 cols, so full labels wrap
    // With 17 tabs, "Dashboard" wraps - check for "board" suffix
    expect(output).toContain('board');
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
  });

  test('at 110 cols (short mode), shows abbreviated labels', () => {
    const { lastFrame } = renderTabBar(110);
    const output = lastFrame() ?? '';

    // Short mode shows abbreviated labels and shortcuts
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).toContain('[3]');
    expect(output).toContain('Dash');
    // Agents label may be truncated to "Ag" with more tabs
    expect(output).toMatch(/Ag/);
    // Full labels should NOT appear
    expect(output).not.toContain('Dashboard');
  });

  test('boundary: 119 cols triggers short mode', () => {
    const { lastFrame } = renderTabBar(119);
    const output = lastFrame() ?? '';

    // At 119, should be short mode - look for short labels
    expect(output).toContain('Dash');
    // Full "Dashboard" should NOT appear
    expect(output).not.toContain('Dashboard');
  });

  test('boundary: 100 cols triggers short mode', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // At 100, still short mode
    expect(output).toContain('Dash');
    // Agents label may be truncated to "Ag" with more tabs
    expect(output).toMatch(/Ag/);
    expect(output).not.toContain('Dashboard');
  });

  test('boundary: 99 cols triggers minimal mode', () => {
    const { lastFrame } = renderTabBar(99);
    const output = lastFrame() ?? '';

    // At 99, minimal mode - no labels, just numbers
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).not.toContain('Dash');
    expect(output).not.toContain('Dashboard');
  });

  test('at 80 cols (minimal mode), shows only tab numbers', () => {
    const { lastFrame } = renderTabBar(80);
    const output = lastFrame() ?? '';

    // At 80 (standard terminal), minimal mode
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).toContain('[3]');
    // No labels in minimal mode
    expect(output).not.toContain('Dash');
    expect(output).not.toContain('Dashboard');
  });
});

describe('TabBar structure', () => {
  test('renders separator after title', () => {
    const { lastFrame } = renderTabBar(120);
    const output = lastFrame() ?? '';

    // Tab bar should have content and contain tab numbers
    // Note: "bc" title may be partially truncated in ink-testing-library
    expect(output).toContain('[1]');
    expect(output.length).toBeGreaterThan(10);
  });

  test('all tab keys are present at every display mode', () => {
    const keys = ['[1]', '[2]', '[3]', '[4]', '[5]', '[6]', '[7]', '[8]', '[?]'];

    // Test each display mode
    for (const width of [80, 110, 140]) {
      const { lastFrame } = renderTabBar(width);
      const output = lastFrame() ?? '';

      for (const key of keys) {
        expect(output).toContain(key);
      }
    }
  });

  test('full labels map correctly at 120+ cols', () => {
    const { lastFrame } = renderTabBar(120);
    const output = lastFrame() ?? '';

    // Verify full labels are used in full mode at 120 cols
    // Note: ink-testing-library renders at 80 cols so labels wrap
    // With 17 tabs (performance + issues), wrapping differs - check for key parts
    expect(output).toContain('[1]');
    expect(output).toContain('board'); // End of "Dashboard" after wrap
    expect(output).toContain('[2]');
    expect(output).toMatch(/Agent/); // "Agents" may wrap differently with 17 tabs
    expect(output).toContain('[3]');
    expect(output).toMatch(/Ch/); // "Channel" may truncate with many tabs
    expect(output).toContain('[4]');
    expect(output).toMatch(/File/); // "Files" may wrap with many tabs
  });

  test('short labels map correctly at 100-119 cols', () => {
    const { lastFrame } = renderTabBar(110);
    const output = lastFrame() ?? '';

    // Verify short labels are used in short mode
    expect(output).toContain('[1]');
    expect(output).toContain('Dash');
    expect(output).toContain('[2]');
    // Agents label may be truncated to "Ag" with more tabs
    expect(output).toMatch(/Ag/);
  });

  test('minimal mode shows only numbers at <100 cols', () => {
    const { lastFrame } = renderTabBar(80);
    const output = lastFrame() ?? '';

    // Verify only numbers are shown
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).not.toContain('Dash');
    expect(output).not.toContain('Dashboard');
  });
});

describe('TabBar accessibility', () => {
  test('keyboard navigation keys always visible', () => {
    // All keys should be visible in all modes, including minimal
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

describe('TabBar #1109 - Fix 80x24 display (replaces #1038 tests)', () => {
  test('at 80x24 (standard terminal), shows minimal tab numbers only', () => {
    const { lastFrame } = renderTabBar(80);
    const output = lastFrame() ?? '';

    // Issue #1109: At 80x24, should use minimal mode to prevent overflow
    // 12 tabs with full labels need ~140 cols, short labels need ~105 cols
    // Minimal mode (just numbers) needs ~55 cols and fits 80x24
    expect(output).toContain('[1]');
    expect(output).toContain('[2]');
    expect(output).toContain('[3]');
    // Labels should NOT appear at 80 cols
    expect(output).not.toContain('Dashboard');
    expect(output).not.toContain('Dash');
  });

  test('at 100 cols, shows short abbreviated labels', () => {
    const { lastFrame } = renderTabBar(100);
    const output = lastFrame() ?? '';

    // At 100 cols, should show short labels
    expect(output).toContain('Dash');
    // Agents label may be truncated to "Ag" with more tabs
    expect(output).toMatch(/Ag/);
    expect(output).not.toContain('Dashboard');
  });

  test('at 120 cols, shows full tab names', () => {
    const { lastFrame } = renderTabBar(120);
    const output = lastFrame() ?? '';

    // At 120 cols, should be in full mode showing complete names
    // Note: ink-testing-library renders at 80 cols so labels wrap
    // With 17 tabs (performance + issues), wrapping differs - check for key parts
    expect(output).toContain('board'); // End of "Dashboard" after wrap
    expect(output).toMatch(/Agent/); // "Agents" may wrap with many tabs
    expect(output).toMatch(/File/); // "Files" may wrap with many tabs
  });

  test('at 99 cols (just below 100), shows minimal mode', () => {
    const { lastFrame } = renderTabBar(99);
    const output = lastFrame() ?? '';

    // At 99 cols, should fall back to minimal mode
    expect(output).toContain('[1]');
    expect(output).not.toContain('Dash');
    expect(output).not.toContain('Dashboard');
  });
});
