/**
 * StatusBadge component tests
 * Issue #682 - Component Testing
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { StatusBadge } from '../../components/StatusBadge';

describe('StatusBadge', () => {
  describe('rendering states', () => {
    it('renders idle state', () => {
      const { lastFrame } = render(<StatusBadge state="idle" />);
      const output = lastFrame();
      expect(output).toContain('idle');
    });

    it('renders working state', () => {
      const { lastFrame } = render(<StatusBadge state="working" />);
      const output = lastFrame();
      expect(output).toContain('working');
    });

    it('renders done state', () => {
      const { lastFrame } = render(<StatusBadge state="done" />);
      const output = lastFrame();
      expect(output).toContain('done');
    });

    it('renders stuck state', () => {
      const { lastFrame } = render(<StatusBadge state="stuck" />);
      const output = lastFrame();
      expect(output).toContain('stuck');
    });

    it('renders error state', () => {
      const { lastFrame } = render(<StatusBadge state="error" />);
      const output = lastFrame();
      expect(output).toContain('error');
    });

    it('renders stopped state', () => {
      const { lastFrame } = render(<StatusBadge state="stopped" />);
      const output = lastFrame();
      expect(output).toContain('stopped');
    });

    it('renders starting state', () => {
      const { lastFrame } = render(<StatusBadge state="starting" />);
      const output = lastFrame();
      expect(output).toContain('starting');
    });

    it('handles unknown state gracefully', () => {
      const { lastFrame } = render(<StatusBadge state="unknown" />);
      const output = lastFrame();
      expect(output).toBeDefined();
    });
  });

  describe('visual properties', () => {
    it('does not throw on render', () => {
      expect(() => {
        render(<StatusBadge state="working" />);
      }).not.toThrow();
    });

    it('produces consistent output', () => {
      const { lastFrame: frame1 } = render(<StatusBadge state="idle" />);
      const { lastFrame: frame2 } = render(<StatusBadge state="idle" />);
      expect(frame1()).toBe(frame2());
    });
  });

  describe('edge cases', () => {
    it('handles empty string state', () => {
      const { lastFrame } = render(<StatusBadge state="" />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles state with extra whitespace', () => {
      const { lastFrame } = render(<StatusBadge state="  working  " />);
      expect(lastFrame()).toBeDefined();
    });
  });
});
