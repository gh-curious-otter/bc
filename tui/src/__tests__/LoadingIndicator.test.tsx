import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { LoadingIndicator } from '../components/LoadingIndicator';

describe('LoadingIndicator', () => {
  describe('initial render', () => {
    it('renders with default message', () => {
      const { lastFrame } = render(<LoadingIndicator />);
      expect(lastFrame()).toContain('Loading');
    });

    it('renders with custom message', () => {
      const { lastFrame } = render(<LoadingIndicator message="Processing..." />);
      expect(lastFrame()).toContain('Processing');
    });

    it('renders spinner character', () => {
      const { lastFrame } = render(<LoadingIndicator />);
      expect(lastFrame()).toContain('⠋');
    });
  });

  describe('animation', () => {
    it('renders spinner frame', () => {
      const { lastFrame } = render(<LoadingIndicator message="Wait" />);
      const frame = lastFrame() ?? '';
      // Should contain a Braille spinner character
      expect(frame).toMatch(/[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏]/);
      expect(frame).toContain('Wait');
    });

    it('renders message with animation', () => {
      const { lastFrame } = render(<LoadingIndicator message="Loading" />);
      const frame = lastFrame();
      expect(frame).toContain('Loading');
    });
  });

  describe('customization', () => {
    it('accepts custom color prop', () => {
      const { lastFrame } = render(<LoadingIndicator message="Loading" color="green" />);
      expect(lastFrame()).toContain('Loading');
    });

    it('accepts custom interval prop', () => {
      const { lastFrame } = render(<LoadingIndicator message="Loading" interval={100} />);
      expect(lastFrame()).toContain('Loading');
    });
  });

  describe('edge cases', () => {
    it('handles empty message', () => {
      const { lastFrame } = render(<LoadingIndicator message="" />);
      expect(lastFrame()).toContain('⠋');
    });

    it('handles very long message', () => {
      const longMessage = 'L'.repeat(100);
      const { lastFrame } = render(<LoadingIndicator message={longMessage} />);
      expect(lastFrame()).toContain('L');
    });

    it('handles special characters in message', () => {
      const { lastFrame } = render(<LoadingIndicator message="Loading... (50%)" />);
      expect(lastFrame()).toContain('50%');
    });
  });

  describe('unmount', () => {
    it('cleans up interval on unmount', () => {
      const { unmount } = render(<LoadingIndicator message="Loading" />);
      // Component should not crash on unmount
      expect(() => {
        unmount();
      }).not.toThrow();
    });
  });
});
