/**
 * BarChart Tests
 * Issue #1046: Data visualization components
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import {
  BarChart,
  MiniBarChart,
  DistributionChart,
} from '../components/BarChart';
import type { BarChartItem } from '../components/BarChart';

describe('BarChart', () => {
  describe('empty state', () => {
    it('renders empty message when no data', () => {
      const { lastFrame } = render(<BarChart data={[]} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('No data');
    });

    it('shows title even with no data', () => {
      const { lastFrame } = render(<BarChart data={[]} title="Test Chart" />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Test Chart');
      expect(output).toContain('No data');
    });
  });

  describe('rendering', () => {
    const testData: BarChartItem[] = [
      { label: 'Item A', value: 50 },
      { label: 'Item B', value: 100 },
      { label: 'Item C', value: 25 },
    ];

    it('renders all items', () => {
      const { lastFrame } = render(<BarChart data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Item A');
      expect(output).toContain('Item B');
      expect(output).toContain('Item C');
    });

    it('shows values by default', () => {
      const { lastFrame } = render(<BarChart data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('(50)');
      expect(output).toContain('(100)');
      expect(output).toContain('(25)');
    });

    it('hides values when showValues=false', () => {
      const { lastFrame } = render(<BarChart data={testData} showValues={false} />);
      const output = lastFrame() ?? '';
      expect(output).not.toContain('(50)');
      expect(output).not.toContain('(100)');
    });

    it('shows percentages when showPercent=true', () => {
      const { lastFrame } = render(<BarChart data={testData} showPercent />);
      const output = lastFrame() ?? '';
      expect(output).toContain('%');
    });

    it('renders bar characters', () => {
      const { lastFrame } = render(<BarChart data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('█');
    });

    it('renders empty bar characters', () => {
      const data: BarChartItem[] = [
        { label: 'Small', value: 10 },
        { label: 'Large', value: 100 },
      ];
      const { lastFrame } = render(<BarChart data={data} barWidth={20} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('░');
    });
  });

  describe('bar scaling', () => {
    it('scales bars relative to max value', () => {
      const data: BarChartItem[] = [
        { label: 'Half', value: 50 },
        { label: 'Full', value: 100 },
      ];
      const { lastFrame } = render(<BarChart data={data} barWidth={20} />);
      const output = lastFrame() ?? '';
      const lines = output.split('\n').filter((l) => l.includes('█'));

      // "Full" should have more filled chars than "Half"
      const halfLine = lines.find((l) => l.includes('Half')) ?? '';
      const fullLine = lines.find((l) => l.includes('Full')) ?? '';

      const halfBars = (halfLine.match(/█/g) ?? []).length;
      const fullBars = (fullLine.match(/█/g) ?? []).length;

      expect(fullBars).toBeGreaterThan(halfBars);
    });
  });

  describe('title', () => {
    it('renders title when provided', () => {
      const data: BarChartItem[] = [{ label: 'Test', value: 50 }];
      const { lastFrame } = render(<BarChart data={data} title="My Chart" />);
      const output = lastFrame() ?? '';
      expect(output).toContain('My Chart');
    });
  });

  describe('custom colors', () => {
    it('uses item color when provided', () => {
      const data: BarChartItem[] = [
        { label: 'Red', value: 50, color: 'red' },
      ];
      const { lastFrame } = render(<BarChart data={data} />);
      const output = lastFrame() ?? '';
      // Output should render (color applied via Text color prop)
      expect(output).toContain('Red');
      expect(output).toContain('█');
    });
  });

  describe('custom characters', () => {
    it('uses custom bar character', () => {
      const data: BarChartItem[] = [{ label: 'Test', value: 100 }];
      const { lastFrame } = render(<BarChart data={data} barChar="=" />);
      const output = lastFrame() ?? '';
      expect(output).toContain('=');
    });

    it('uses custom empty character', () => {
      const data: BarChartItem[] = [
        { label: 'Small', value: 10 },
        { label: 'Large', value: 100 },
      ];
      const { lastFrame } = render(<BarChart data={data} emptyChar="-" barWidth={20} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('-');
    });
  });
});

describe('MiniBarChart', () => {
  it('renders empty placeholder when no data', () => {
    const { lastFrame } = render(<MiniBarChart data={[]} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('─');
  });

  it('renders bar characters for data', () => {
    const { lastFrame } = render(<MiniBarChart data={[50, 100, 25]} />);
    const output = lastFrame() ?? '';
    // Should contain sparkline-like characters
    expect(output.length).toBeGreaterThan(0);
  });

  it('shows total when showTotal=true', () => {
    const { lastFrame } = render(<MiniBarChart data={[10, 20, 30]} showTotal />);
    const output = lastFrame() ?? '';
    expect(output).toContain('(60)');
  });

  it('respects width parameter', () => {
    const { lastFrame } = render(<MiniBarChart data={[50, 100]} width={8} />);
    const output = lastFrame() ?? '';
    // Output should be compact
    expect(output.length).toBeLessThan(20);
  });
});

describe('DistributionChart', () => {
  it('renders empty message when no data', () => {
    const { lastFrame } = render(<DistributionChart data={[]} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('No data');
  });

  it('renders labels and symbols', () => {
    const data = [
      { label: 'engineer', count: 4 },
      { label: 'manager', count: 2 },
    ];
    const { lastFrame } = render(<DistributionChart data={data} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('engineer');
    expect(output).toContain('manager');
    expect(output).toContain('⊙');
  });

  it('shows counts by default', () => {
    const data = [{ label: 'test', count: 5 }];
    const { lastFrame } = render(<DistributionChart data={data} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('(5)');
  });

  it('hides counts when showCounts=false', () => {
    const data = [{ label: 'test', count: 5 }];
    const { lastFrame } = render(<DistributionChart data={data} showCounts={false} />);
    const output = lastFrame() ?? '';
    expect(output).not.toContain('(5)');
  });

  it('uses custom symbol', () => {
    const data = [{ label: 'test', count: 3 }];
    const { lastFrame } = render(<DistributionChart data={data} symbol="●" />);
    const output = lastFrame() ?? '';
    expect(output).toContain('●');
  });

  it('scales symbols to maxSymbols', () => {
    const data = [
      { label: 'small', count: 1 },
      { label: 'large', count: 100 },
    ];
    const { lastFrame } = render(
      <DistributionChart data={data} maxSymbols={8} />
    );
    const output = lastFrame() ?? '';
    const lines = output.split('\n');
    const largeLine = lines.find((l) => l.includes('large')) ?? '';
    const symbols = (largeLine.match(/⊙/g) ?? []).length;
    // Should scale to maxSymbols
    expect(symbols).toBeLessThanOrEqual(8);
  });

  it('uses item color when provided', () => {
    const data = [{ label: 'red', count: 3, color: 'red' }];
    const { lastFrame } = render(<DistributionChart data={data} />);
    const output = lastFrame() ?? '';
    expect(output).toContain('red');
    expect(output).toContain('⊙');
  });
});
