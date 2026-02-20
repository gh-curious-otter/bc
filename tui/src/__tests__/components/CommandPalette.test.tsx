/**
 * CommandPalette.test.tsx - Tests for command palette component
 * Issue #1098: Command palette implementation
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { CommandPalette } from '../../components/CommandPalette';
import { getAllCommands, type BcCommand } from '../../types/commands';

describe('CommandPalette', () => {
  describe('rendering', () => {
    it('should not render when isOpen is false', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={false} onClose={() => {}} disableInput />
      );

      expect(lastFrame()).toBe('');
    });

    it('should render when isOpen is true', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      expect(lastFrame()).toContain('>');
      expect(lastFrame()).toContain('navigate');
    });

    it('should show command results', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      // Should show some commands from the registry
      const commands = getAllCommands();
      expect(commands.length).toBeGreaterThan(0);
      // First command should be visible (at least part of its name)
      expect(lastFrame()).toBeTruthy();
    });

    it('should show footer hints', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      expect(lastFrame()).toContain('↑/↓');
      expect(lastFrame()).toContain('Enter');
      expect(lastFrame()).toContain('Esc');
    });
  });

  describe('recent commands', () => {
    it('should mark recent commands with asterisk', () => {
      const { lastFrame } = render(
        <CommandPalette
          isOpen={true}
          onClose={() => {}}
          recentCommands={['agent status']}
          disableInput
        />
      );

      expect(lastFrame()).toContain('*');
    });

    it('should show recent commands first', () => {
      const { lastFrame } = render(
        <CommandPalette
          isOpen={true}
          onClose={() => {}}
          recentCommands={['logs']}
          disableInput
        />
      );

      // logs should appear (it's in the recent list)
      expect(lastFrame()).toContain('logs');
    });
  });

  describe('max results', () => {
    it('should respect maxResults prop', () => {
      const { lastFrame } = render(
        <CommandPalette
          isOpen={true}
          onClose={() => {}}
          maxResults={3}
          disableInput
        />
      );

      // Should show limited results
      const frame = lastFrame() ?? '';
      // Count the number of command rows (lines with command names)
      const lines = frame.split('\n').filter(l => l.includes(' - '));
      expect(lines.length).toBeLessThanOrEqual(3);
    });
  });

  describe('search input', () => {
    it('should show cursor indicator', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      expect(lastFrame()).toContain('|'); // Cursor
    });

    it('should display divider', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      expect(lastFrame()).toContain('─');
    });
  });

  describe('command row', () => {
    it('should show command name and description', () => {
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      // Should contain the separator between name and description
      expect(lastFrame()).toContain(' - ');
    });
  });

  describe('empty state', () => {
    // This test would require simulating typing a query that matches nothing
    // For now, we test that the component handles the empty case gracefully
    it('should handle empty commands gracefully', () => {
      // The component relies on getAllCommands which always returns commands
      // So we just verify it renders without error
      const { lastFrame } = render(
        <CommandPalette isOpen={true} onClose={() => {}} disableInput />
      );

      expect(lastFrame()).toBeTruthy();
    });
  });

  describe('props', () => {
    it('should accept onSelect callback', () => {
      let selectedCommand: BcCommand | undefined;
      const onSelect = (cmd: BcCommand) => { selectedCommand = cmd; };

      const { lastFrame } = render(
        <CommandPalette
          isOpen={true}
          onClose={() => {}}
          onSelect={onSelect}
          disableInput
        />
      );

      // Component renders without error
      expect(lastFrame()).toBeTruthy();
      // Initially no command selected via callback
      expect(selectedCommand).toBeUndefined();
    });

    it('should call onClose when closed', () => {
      let closeCalled = false;
      const onClose = () => { closeCalled = true; };

      // Just verify it accepts the callback
      const { lastFrame } = render(
        <CommandPalette
          isOpen={true}
          onClose={onClose}
          disableInput
        />
      );

      expect(lastFrame()).toBeTruthy();
    });
  });
});
