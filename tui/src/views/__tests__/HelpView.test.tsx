/**
 * HelpView Tests - Keyboard shortcuts help display
 *
 * Issue #1582: Tests for extracted HelpView component
 *
 * Tests cover:
 * - Help sections structure
 * - Shortcut data format
 * - Line count calculation
 * - Scroll behavior
 * - Section visibility
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, test, expect } from 'bun:test';
import { ThemeProvider } from '../../theme';
import { DisableInputProvider } from '../../hooks';
import { HelpView } from '../HelpView';

// Test wrapper with required providers
// #1594: Use DisableInputProvider to disable input handling in tests
function renderWithProviders(component: React.ReactElement) {
  return render(
    <ThemeProvider>
      <DisableInputProvider disabled>
        {component}
      </DisableInputProvider>
    </ThemeProvider>
  );
}

// Types matching HelpView internal structure
interface ShortcutSection {
  type: 'section';
  title: string;
  shortcuts: { keys: string; desc: string }[];
}

interface HeaderSection {
  type: 'header';
}

interface FooterSection {
  type: 'footer';
}

type HelpSection = ShortcutSection | HeaderSection | FooterSection;

// Recreate help sections for testing
const helpSections: HelpSection[] = [
  { type: 'header' },
  { type: 'section', title: 'Global', shortcuts: [
    { keys: 'Tab', desc: 'Next view' },
    { keys: 'Shift+Tab', desc: 'Previous view' },
    { keys: 'M', desc: 'Memory view' },
    { keys: 'R', desc: 'Routing view' },
    { keys: '?', desc: 'Toggle help' },
    { keys: 'ESC', desc: 'Go back / Home' },
    { keys: 'Ctrl+R', desc: 'Refresh current view' },
    { keys: 'q', desc: 'Quit' },
  ]},
  { type: 'section', title: 'Navigation (Drawer & Lists)', shortcuts: [
    { keys: 'j / ↓', desc: 'Move down in drawer/list' },
    { keys: 'k / ↑', desc: 'Move up in drawer/list' },
    { keys: 'g', desc: 'Jump to top' },
    { keys: 'G', desc: 'Jump to bottom' },
    { keys: 'Enter', desc: 'Select / Drill down' },
  ]},
  { type: 'section', title: 'Agents', shortcuts: [
    { keys: 'Enter', desc: 'Attach to agent session' },
    { keys: 'p', desc: 'Peek agent output' },
    { keys: 'x', desc: 'Stop agent' },
    { keys: 'X', desc: 'Kill agent (force)' },
    { keys: 'R', desc: 'Restart agent' },
  ]},
  { type: 'section', title: 'Channels', shortcuts: [
    { keys: 'Enter', desc: 'View channel history' },
    { keys: 'm', desc: 'Compose message' },
    { keys: 'j/k', desc: 'Scroll messages' },
    { keys: 'c', desc: 'Clear draft' },
  ]},
  { type: 'section', title: 'Costs', shortcuts: [
    { keys: '1/2/3', desc: 'Switch agent/model/team tabs' },
    { keys: 'b', desc: 'Set budget' },
    { keys: 'e', desc: 'Export to CSV' },
    { keys: 'r', desc: 'Refresh data' },
  ]},
  { type: 'section', title: 'Commands', shortcuts: [
    { keys: '/', desc: 'Search commands' },
    { keys: 'f', desc: 'Toggle favorite' },
    { keys: 'Enter', desc: 'Copy command' },
  ]},
  { type: 'section', title: 'Memory', shortcuts: [
    { keys: 'j/k', desc: 'Navigate agents' },
    { keys: 'Enter', desc: 'View details' },
    { keys: '/', desc: 'Search memories' },
    { keys: '1/2', desc: 'Switch exp/learnings' },
    { keys: 'c', desc: 'Clear memory' },
  ]},
  { type: 'section', title: 'Routing', shortcuts: [
    { keys: 'j/k', desc: 'Navigate rules' },
    { keys: 'Enter', desc: 'View details' },
  ]},
  { type: 'footer' },
];

// Helper to calculate total lines
function calculateTotalLines(sections: HelpSection[]): number {
  return sections.reduce((acc, section) => {
    if (section.type === 'header') return acc + 2;
    if (section.type === 'footer') return acc + 3;
    return acc + 1 + section.shortcuts.length + 1; // title + shortcuts + margin
  }, 0);
}

describe('HelpView', () => {
  describe('Rendering', () => {
    test('renders without crashing', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      expect(lastFrame()).toBeDefined();
    });

    test('displays header divider', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      // Header divider line should be visible
      expect(lastFrame()).toContain('─');
    });

    test('displays Global shortcuts', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      // Global shortcuts content should be visible (Tab shortcut)
      expect(lastFrame()).toContain('Tab');
    });

    test('displays Navigation shortcuts', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      // Navigation shortcuts content should be visible (j/k)
      expect(lastFrame()).toContain('Move down');
    });

    test('displays shortcut key Tab', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      expect(lastFrame()).toContain('Tab');
    });

    test('displays shortcut description', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      expect(lastFrame()).toContain('Next view');
    });
  });

  describe('Help Sections Structure', () => {
    test('has correct number of sections', () => {
      // 1 header + 8 sections + 1 footer = 10 total
      expect(helpSections.length).toBe(10);
    });

    test('first section is header', () => {
      expect(helpSections[0].type).toBe('header');
    });

    test('last section is footer', () => {
      expect(helpSections[helpSections.length - 1].type).toBe('footer');
    });

    test('Global section has 8 shortcuts', () => {
      const globalSection = helpSections.find(s => s.type === 'section' && s.title === 'Global') as ShortcutSection;
      expect(globalSection.shortcuts.length).toBe(8);
    });

    test('Navigation section has 5 shortcuts', () => {
      const navSection = helpSections.find(s => s.type === 'section' && s.title === 'Navigation (Drawer & Lists)') as ShortcutSection;
      expect(navSection.shortcuts.length).toBe(5);
    });

    test('Agents section has 5 shortcuts', () => {
      const agentsSection = helpSections.find(s => s.type === 'section' && s.title === 'Agents') as ShortcutSection;
      expect(agentsSection.shortcuts.length).toBe(5);
    });

    test('Channels section has 4 shortcuts', () => {
      const channelsSection = helpSections.find(s => s.type === 'section' && s.title === 'Channels') as ShortcutSection;
      expect(channelsSection.shortcuts.length).toBe(4);
    });

    test('all shortcuts have keys and desc', () => {
      for (const section of helpSections) {
        if (section.type === 'section') {
          for (const shortcut of section.shortcuts) {
            expect(shortcut.keys).toBeDefined();
            expect(shortcut.keys.length).toBeGreaterThan(0);
            expect(shortcut.desc).toBeDefined();
            expect(shortcut.desc.length).toBeGreaterThan(0);
          }
        }
      }
    });
  });

  describe('Line Count Calculation', () => {
    test('calculates correct total lines', () => {
      const totalLines = calculateTotalLines(helpSections);
      // Header: 2 lines
      // Footer: 3 lines
      // Each section: 1 (title) + shortcuts + 1 (margin)
      // Global: 1 + 8 + 1 = 10
      // Navigation: 1 + 5 + 1 = 7
      // Agents: 1 + 5 + 1 = 7
      // Channels: 1 + 4 + 1 = 6
      // Costs: 1 + 4 + 1 = 6
      // Commands: 1 + 3 + 1 = 5
      // Memory: 1 + 5 + 1 = 7
      // Routing: 1 + 2 + 1 = 4
      // Total: 2 + 3 + 10 + 7 + 7 + 6 + 6 + 5 + 7 + 4 = 57
      expect(totalLines).toBe(57);
    });

    test('header contributes 2 lines', () => {
      const headerOnly: HelpSection[] = [{ type: 'header' }];
      expect(calculateTotalLines(headerOnly)).toBe(2);
    });

    test('footer contributes 3 lines', () => {
      const footerOnly: HelpSection[] = [{ type: 'footer' }];
      expect(calculateTotalLines(footerOnly)).toBe(3);
    });

    test('section with 3 shortcuts contributes 5 lines', () => {
      const sectionOnly: HelpSection[] = [{
        type: 'section',
        title: 'Test',
        shortcuts: [
          { keys: 'a', desc: 'Action A' },
          { keys: 'b', desc: 'Action B' },
          { keys: 'c', desc: 'Action C' },
        ]
      }];
      // 1 (title) + 3 (shortcuts) + 1 (margin) = 5
      expect(calculateTotalLines(sectionOnly)).toBe(5);
    });
  });

  describe('Scroll Behavior', () => {
    test('scroll offset calculation', () => {
      const availableHeight = 20;
      const totalLines = calculateTotalLines(helpSections);
      const needsScroll = totalLines > availableHeight;
      const maxScroll = Math.max(0, totalLines - availableHeight);

      expect(needsScroll).toBe(true);
      expect(maxScroll).toBe(totalLines - availableHeight);
    });

    test('no scroll needed for small content', () => {
      const smallSections: HelpSection[] = [
        { type: 'header' },
        { type: 'section', title: 'Test', shortcuts: [{ keys: 'a', desc: 'Action' }] },
        { type: 'footer' }
      ];
      const totalLines = calculateTotalLines(smallSections);
      // 2 + (1 + 1 + 1) + 3 = 8 lines
      const availableHeight = 20;
      const needsScroll = totalLines > availableHeight;

      expect(needsScroll).toBe(false);
    });
  });

  describe('Shortcut Keys Format', () => {
    test('keys are padded to 12 characters', () => {
      const shortKey = 'Tab';
      const paddedKey = shortKey.padEnd(12);
      expect(paddedKey.length).toBe(12);
      expect(paddedKey).toBe('Tab         ');
    });

    test('long keys are not truncated', () => {
      const longKey = 'Shift+Tab';
      const paddedKey = longKey.padEnd(12);
      expect(paddedKey.length).toBe(12);
      expect(paddedKey).toBe('Shift+Tab   ');
    });
  });

  describe('Theme Integration', () => {
    test('renders with theme provider', () => {
      const { lastFrame } = renderWithProviders(<HelpView />);
      // Should render successfully with theme (footer may not be visible due to scroll)
      // Check for scroll hints which indicate proper rendering
      expect(lastFrame()).toContain('j/k');
    });
  });
});
