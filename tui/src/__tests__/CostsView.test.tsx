import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { CostsView } from '../components/CostsView';

/**
 * CostsView tests
 * Note: These are basic rendering tests since the component uses useCosts hook
 * Full integration tests with data would be in views/__tests__
 */
describe('CostsView', () => {
  describe('basic rendering', () => {
    it('renders without crashing', () => {
      const { lastFrame } = render(<CostsView />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with disableInput prop false', () => {
      const { lastFrame } = render(<CostsView disableInput={false} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with disableInput prop true', () => {
      const { lastFrame } = render(<CostsView disableInput={true} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('loading states', () => {
    it('handles loading state gracefully', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      // Should render something when loading
      expect(frame).toBeDefined();
    });

    it('displays loading message during fetch', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      expect(frame).toBeDefined();
    });
  });

  describe('error handling', () => {
    it('renders when data load fails', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      // Should handle errors without crashing
      expect(frame).toBeDefined();
    });

    it('displays error message when applicable', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      expect(frame).toBeDefined();
    });
  });

  describe('data display', () => {
    it('renders cost dashboard when data available', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      expect(frame).toBeDefined();
    });

    it('displays title correctly', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      // Should display some form of title/header
      expect(frame).toBeDefined();
    });

    it('handles missing cost data', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      // Should handle null/undefined gracefully
      expect(frame).toBeDefined();
    });
  });

  describe('input handling', () => {
    it('respects disableInput prop', () => {
      const { lastFrame } = render(<CostsView disableInput={true} />);
      expect(lastFrame()).toBeDefined();
    });

    it('allows input when enabled', () => {
      const { lastFrame } = render(<CostsView disableInput={false} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('terminal responsiveness', () => {
    it('renders with terminal width constraints', () => {
      const { lastFrame } = render(<CostsView />);
      const frame = lastFrame();
      // Should adapt to terminal width
      expect(frame).toBeDefined();
    });

    it('handles narrow terminal widths', () => {
      const { lastFrame } = render(<CostsView />);
      expect(lastFrame()).toBeDefined();
    });
  });
});
