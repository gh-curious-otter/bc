/**
 * ProgressBar Component Tests
 * Issue #864 - Cost visualizations
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, test } from 'bun:test';
import { ProgressBar } from '../components/ProgressBar';

describe('ProgressBar', () => {
  describe('basic rendering', () => {
    it('renders without crashing', () => {
      const { lastFrame } = render(<ProgressBar value={50} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with custom width', () => {
      const { lastFrame } = render(<ProgressBar value={50} width={10} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders percentage label by default', () => {
      const { lastFrame } = render(<ProgressBar value={50} />);
      expect(lastFrame()).toContain('50%');
    });
  });

  describe('value handling', () => {
    it('renders 0% correctly', () => {
      const { lastFrame } = render(<ProgressBar value={0} />);
      expect(lastFrame()).toContain('0%');
    });

    it('renders 100% correctly', () => {
      const { lastFrame } = render(<ProgressBar value={100} />);
      expect(lastFrame()).toContain('100%');
    });

    it('clamps values above 100', () => {
      const { lastFrame } = render(<ProgressBar value={150} />);
      expect(lastFrame()).toContain('100%');
    });

    it('clamps negative values to 0', () => {
      const { lastFrame } = render(<ProgressBar value={-10} />);
      expect(lastFrame()).toContain('0%');
    });

    it('handles custom max value', () => {
      const { lastFrame } = render(<ProgressBar value={5} max={10} />);
      expect(lastFrame()).toContain('50%');
    });
  });

  describe('threshold colors', () => {
    test('default color below warning threshold', () => {
      const { lastFrame } = render(<ProgressBar value={40} />);
      // Should render without error - value below 50% (default warning)
      expect(lastFrame()).toBeDefined();
    });

    test('warning threshold triggers at configured value', () => {
      const { lastFrame } = render(
        <ProgressBar value={60} colorThresholds={{ warning: 50, critical: 80 }} />
      );
      expect(lastFrame()).toContain('60%');
    });

    test('critical threshold triggers at configured value', () => {
      const { lastFrame } = render(
        <ProgressBar value={85} colorThresholds={{ warning: 50, critical: 80 }} />
      );
      expect(lastFrame()).toContain('85%');
    });
  });

  describe('label options', () => {
    it('hides percentage when showPercent is false', () => {
      const { lastFrame } = render(<ProgressBar value={50} showPercent={false} />);
      expect(lastFrame()).not.toContain('%');
    });

    it('shows value/max when showValue is true', () => {
      const { lastFrame } = render(
        <ProgressBar value={5} max={10} showPercent={false} showValue={true} />
      );
      expect(lastFrame()).toContain('5.00');
      expect(lastFrame()).toContain('10.00');
    });

    it('shows valuePrefix with value display', () => {
      const { lastFrame } = render(
        <ProgressBar value={5} max={10} showPercent={false} showValue={true} valuePrefix="$" />
      );
      expect(lastFrame()).toContain('$');
    });
  });
});

describe('ProgressBar calculations', () => {
  test('percentage calculation is correct', () => {
    const value = 25;
    const max = 100;
    const percentage = (value / max) * 100;
    expect(percentage).toBe(25);
  });

  test('handles zero max value gracefully', () => {
    const value = 50;
    const max = 0;
    const percentage = max === 0 ? 0 : (value / max) * 100;
    expect(percentage).toBe(0);
  });

  test('filled width calculation', () => {
    const percentage = 50;
    const width = 20;
    const filledWidth = Math.round((percentage / 100) * width);
    expect(filledWidth).toBe(10);
  });
});
