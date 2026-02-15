import { describe, expect, it } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { MetricCard } from '../components/MetricCard';

describe('MetricCard', () => {
  describe('basic rendering', () => {
    it('renders label and value', () => {
      const { lastFrame } = render(
        <MetricCard label="Total" value={42} />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('Total');
      expect(output).toContain('42');
    });

    it('renders with zero value', () => {
      const { lastFrame } = render(
        <MetricCard label="Errors" value={0} />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('Errors');
      expect(output).toContain('0');
    });

    it('renders with string value', () => {
      const { lastFrame } = render(
        <MetricCard label="Status" value="OK" />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('Status');
      expect(output).toContain('OK');
    });

    it('renders large numbers', () => {
      const { lastFrame } = render(
        <MetricCard label="Tokens" value={1234567} />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('Tokens');
      expect(output).toContain('1234567');
    });
  });

  describe('colors', () => {
    it('renders with custom color', () => {
      const { lastFrame } = render(
        <MetricCard label="Active" value={5} color="green" />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('Active');
      expect(output).toContain('5');
    });

    it('renders with red color', () => {
      const { lastFrame } = render(
        <MetricCard label="Failed" value={3} color="red" />
      );
      expect(lastFrame()).toContain('Failed');
    });

    it('renders with cyan color', () => {
      const { lastFrame } = render(
        <MetricCard label="Progress" value={50} color="cyan" />
      );
      expect(lastFrame()).toContain('Progress');
    });
  });

  describe('prefix and suffix', () => {
    it('renders with prefix', () => {
      const { lastFrame } = render(
        <MetricCard value={99} label="Cost" prefix="$" />
      );
      expect(lastFrame()).toContain('$99');
    });

    it('renders with suffix', () => {
      const { lastFrame } = render(
        <MetricCard value={75} label="Success" suffix="%" />
      );
      expect(lastFrame()).toContain('75%');
    });

    it('renders with both prefix and suffix', () => {
      const { lastFrame } = render(
        <MetricCard value={1000} label="Revenue" prefix="$" suffix=" USD" />
      );
      const output = lastFrame() ?? '';
      expect(output).toContain('$1000');
      expect(output).toContain('USD');
    });
  });

  describe('value types', () => {
    it('renders decimal numbers', () => {
      const { lastFrame } = render(
        <MetricCard value={3.14} label="Pi" />
      );
      expect(lastFrame()).toContain('3.14');
    });

    it('renders negative numbers', () => {
      const { lastFrame } = render(
        <MetricCard value={-5} label="Delta" />
      );
      expect(lastFrame()).toContain('-5');
    });

    it('renders very large numbers', () => {
      const { lastFrame } = render(
        <MetricCard value={999999999} label="Big" />
      );
      expect(lastFrame()).toContain('999999999');
    });
  });

  describe('edge cases', () => {
    it('renders with very long label', () => {
      const longLabel = 'A'.repeat(50);
      const { lastFrame } = render(
        <MetricCard value={1} label={longLabel} />
      );
      expect(lastFrame()).toContain('A');
    });

    it('renders with special characters', () => {
      const { lastFrame } = render(
        <MetricCard value={100} label="Test [#123]" />
      );
      expect(lastFrame()).toContain('#123');
    });
  });
});
