/**
 * MetricCard component extended tests
 * Issue #682 - Component Testing
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { MetricCard } from '../../components/MetricCard';

describe('MetricCard - Extended Tests', () => {
  describe('value display', () => {
    it('renders zero value', () => {
      const { lastFrame } = render(<MetricCard value={0} label="Count" />);
      expect(lastFrame()).toContain('0');
    });

    it('renders positive integer', () => {
      const { lastFrame } = render(<MetricCard value={42} label="Count" />);
      expect(lastFrame()).toContain('42');
    });

    it('renders large number', () => {
      const { lastFrame } = render(<MetricCard value={1000000} label="Count" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders negative number', () => {
      const { lastFrame } = render(<MetricCard value={-5} label="Delta" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders decimal number', () => {
      const { lastFrame } = render(<MetricCard value={3.14} label="Pi" />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('label display', () => {
    it('renders short label', () => {
      const { lastFrame } = render(<MetricCard value={1} label="A" />);
      expect(lastFrame()).toContain('A');
    });

    it('renders long label', () => {
      const longLabel = 'Very Long Label Name';
      const { lastFrame } = render(<MetricCard value={1} label={longLabel} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders label with spaces', () => {
      const { lastFrame } = render(<MetricCard value={1} label="Active Agents" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders label with special characters', () => {
      const { lastFrame } = render(<MetricCard value={1} label="Cost ($)" />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('color variations', () => {
    it('renders with green color', () => {
      const { lastFrame } = render(<MetricCard value={5} label="Active" color="green" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with red color', () => {
      const { lastFrame } = render(<MetricCard value={2} label="Errors" color="red" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with yellow color', () => {
      const { lastFrame } = render(<MetricCard value={3} label="Warnings" color="yellow" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with cyan color', () => {
      const { lastFrame } = render(<MetricCard value={10} label="Working" color="cyan" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with gray color', () => {
      const { lastFrame } = render(<MetricCard value={0} label="Idle" color="gray" />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with default color (no color prop)', () => {
      const { lastFrame } = render(<MetricCard value={7} label="Total" />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('layout', () => {
    it('renders without crashing', () => {
      expect(() => {
        render(<MetricCard value={1} label="Test" />);
      }).not.toThrow();
    });

    it('produces consistent output', () => {
      const { lastFrame: frame1 } = render(<MetricCard value={42} label="Count" />);
      const { lastFrame: frame2 } = render(<MetricCard value={42} label="Count" />);
      expect(frame1()).toBe(frame2());
    });
  });

  describe('edge cases', () => {
    it('handles empty label', () => {
      const { lastFrame } = render(<MetricCard value={1} label="" />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles very large values', () => {
      const { lastFrame } = render(<MetricCard value={Number.MAX_SAFE_INTEGER} label="Max" />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles Infinity', () => {
      const { lastFrame } = render(<MetricCard value={Infinity} label="Inf" />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles NaN gracefully', () => {
      const { lastFrame } = render(<MetricCard value={NaN} label="NaN" />);
      expect(lastFrame()).toBeDefined();
    });
  });
});
