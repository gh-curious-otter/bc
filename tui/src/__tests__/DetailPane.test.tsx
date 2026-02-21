/**
 * DetailPane component tests
 * Issue #1310: Add toggleable detail pane for selected items
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { DetailPane, shouldShowDetailPane, type DetailItem } from '../components/DetailPane';

describe('DetailPane', () => {
  const mockItem: DetailItem = {
    title: 'test-agent',
    type: 'agent',
    fields: [
      { label: 'Status', value: 'running', color: 'green' },
      { label: 'Role', value: 'engineer' },
      { label: 'PID', value: '12345' },
    ],
    description: 'Test agent description',
  };

  describe('rendering', () => {
    it('renders placeholder when no item selected', () => {
      const { lastFrame } = render(
        <DetailPane
          view="agents"
          selectedItem={null}
          terminalWidth={120}
          terminalHeight={40}
        />
      );

      expect(lastFrame()).toContain('Details');
      expect(lastFrame()).toContain('Select an item');
    });

    it('renders item details when item provided', () => {
      const { lastFrame } = render(
        <DetailPane
          view="agents"
          selectedItem={mockItem}
          terminalWidth={120}
          terminalHeight={40}
        />
      );

      expect(lastFrame()).toContain('Details');
      expect(lastFrame()).toContain('[agent]');
      expect(lastFrame()).toContain('test-agent');
      expect(lastFrame()).toContain('Status');
      expect(lastFrame()).toContain('running');
    });

    it('shows toggle hint', () => {
      const { lastFrame } = render(
        <DetailPane
          view="agents"
          selectedItem={null}
          terminalWidth={120}
          terminalHeight={40}
        />
      );

      expect(lastFrame()).toContain('[i] toggle pane');
    });

    it('renders item description', () => {
      const { lastFrame } = render(
        <DetailPane
          view="agents"
          selectedItem={mockItem}
          terminalWidth={120}
          terminalHeight={40}
        />
      );

      expect(lastFrame()).toContain('Test agent description');
    });

    it('renders all fields', () => {
      const { lastFrame } = render(
        <DetailPane
          view="agents"
          selectedItem={mockItem}
          terminalWidth={120}
          terminalHeight={40}
        />
      );

      expect(lastFrame()).toContain('Status');
      expect(lastFrame()).toContain('Role');
      expect(lastFrame()).toContain('PID');
      expect(lastFrame()).toContain('12345');
    });

    it('truncates long titles', () => {
      const longItem: DetailItem = {
        title: 'this-is-a-very-long-agent-name-that-should-truncate',
        type: 'agent',
        fields: [],
      };

      const { lastFrame } = render(
        <DetailPane
          view="agents"
          selectedItem={longItem}
          terminalWidth={120}
          terminalHeight={40}
        />
      );

      // Should truncate and end with ellipsis (26 chars max for title)
      expect(lastFrame()).toContain('this-is-a-very-long-agent…');
    });
  });

  describe('shouldShowDetailPane', () => {
    it('returns false at compact terminal size (80x24)', () => {
      expect(shouldShowDetailPane(80, 24, true)).toBe(false);
    });

    it('returns false when terminal too narrow', () => {
      expect(shouldShowDetailPane(90, 40, true)).toBe(false);
    });

    it('returns false when user toggled off', () => {
      expect(shouldShowDetailPane(120, 40, false)).toBe(false);
    });

    it('returns true when terminal wide enough and toggled on', () => {
      expect(shouldShowDetailPane(120, 40, true)).toBe(true);
    });

    it('returns true at minimum width', () => {
      expect(shouldShowDetailPane(100, 40, true)).toBe(true);
    });

    it('handles edge case: exactly 80x24 returns false', () => {
      expect(shouldShowDetailPane(80, 24, true)).toBe(false);
    });

    it('handles wide terminal with short height', () => {
      // Width okay (120) but height is 24 - should show because width > 80
      expect(shouldShowDetailPane(120, 24, true)).toBe(true);
    });

    it('handles narrow terminal with tall height', () => {
      // Height okay (40) but width < 100 - should hide
      expect(shouldShowDetailPane(80, 40, true)).toBe(false);
    });
  });
});
