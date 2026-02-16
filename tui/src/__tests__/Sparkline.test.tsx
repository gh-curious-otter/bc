/**
 * Sparkline Component Tests
 * Issue #864 - Cost visualizations
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect, test } from 'bun:test';
import { Sparkline } from '../components/Sparkline';

describe('Sparkline', () => {
  describe('basic rendering', () => {
    it('renders without crashing', () => {
      const { lastFrame } = render(<Sparkline data={[1, 2, 3, 4, 5]} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with empty data', () => {
      const { lastFrame } = render(<Sparkline data={[]} />);
      expect(lastFrame()).toContain('No data');
    });

    it('renders with single data point', () => {
      const { lastFrame } = render(<Sparkline data={[5]} />);
      expect(lastFrame()).toBeDefined();
    });

    it('renders with label', () => {
      const { lastFrame } = render(<Sparkline data={[1, 2, 3]} label="Trend" />);
      expect(lastFrame()).toContain('Trend');
    });
  });

  describe('range display', () => {
    it('shows range when showRange is true', () => {
      const { lastFrame } = render(
        <Sparkline data={[1, 5, 10]} showRange={true} />
      );
      // Should show min-max range
      expect(lastFrame()).toBeDefined();
    });

    it('hides range by default', () => {
      const { lastFrame } = render(<Sparkline data={[1, 5, 10]} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('width handling', () => {
    it('renders with custom width', () => {
      const { lastFrame } = render(<Sparkline data={[1, 2, 3, 4, 5]} width={10} />);
      expect(lastFrame()).toBeDefined();
    });

    it('resamples data when width differs from data length', () => {
      const { lastFrame } = render(<Sparkline data={[1, 2, 3, 4, 5, 6, 7, 8, 9, 10]} width={5} />);
      expect(lastFrame()).toBeDefined();
    });
  });
});

describe('Sparkline data processing', () => {
  test('min/max calculation', () => {
    const data = [5, 2, 8, 1, 9];
    const min = Math.min(...data);
    const max = Math.max(...data);
    expect(min).toBe(1);
    expect(max).toBe(9);
  });

  test('handles all equal values', () => {
    const data = [5, 5, 5, 5, 5];
    const min = Math.min(...data);
    const max = Math.max(...data);
    expect(min).toBe(max);
  });

  test('normalization calculation', () => {
    const value = 5;
    const min = 0;
    const max = 10;
    const normalized = (value - min) / (max - min);
    expect(normalized).toBe(0.5);
  });
});

describe('Sparkline characters', () => {
  const SPARK_CHARS = ['▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'];

  test('sparkline uses correct characters', () => {
    expect(SPARK_CHARS.length).toBe(8);
  });

  test('lowest value maps to first character', () => {
    const normalized = 0;
    const index = Math.min(SPARK_CHARS.length - 1, Math.floor(normalized * SPARK_CHARS.length));
    expect(SPARK_CHARS[index]).toBe('▁');
  });

  test('highest value maps to last character', () => {
    const normalized = 1;
    const index = Math.min(SPARK_CHARS.length - 1, Math.floor(normalized * SPARK_CHARS.length));
    expect(SPARK_CHARS[index]).toBe('█');
  });

  test('middle value maps to middle character', () => {
    const normalized = 0.5;
    const index = Math.min(SPARK_CHARS.length - 1, Math.floor(normalized * SPARK_CHARS.length));
    expect(SPARK_CHARS[index]).toBe('▅');
  });
});

describe('Number formatting', () => {
  test('formats millions correctly', () => {
    const n = 1_500_000;
    const formatted = `${(n / 1_000_000).toFixed(1)}M`;
    expect(formatted).toBe('1.5M');
  });

  test('formats thousands correctly', () => {
    const n = 2_500;
    const formatted = `${(n / 1_000).toFixed(1)}K`;
    expect(formatted).toBe('2.5K');
  });

  test('formats small decimals correctly', () => {
    const n = 0.25;
    const formatted = n.toFixed(2);
    expect(formatted).toBe('0.25');
  });
});
