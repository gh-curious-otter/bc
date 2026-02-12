import { describe, expect, test } from 'bun:test';
import { render } from 'ink-testing-library';
import React from 'react';
import { MetricCard } from '../components/MetricCard';

describe('MetricCard', () => {
  test('renders label and value', () => {
    const { lastFrame } = render(
      <MetricCard label="Total" value={42} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Total');
    expect(output).toContain('42');
  });

  test('renders with zero value', () => {
    const { lastFrame } = render(
      <MetricCard label="Errors" value={0} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Errors');
    expect(output).toContain('0');
  });

  test('renders with string value', () => {
    const { lastFrame } = render(
      <MetricCard label="Status" value="OK" />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Status');
    expect(output).toContain('OK');
  });

  test('renders with custom color', () => {
    const { lastFrame } = render(
      <MetricCard label="Active" value={5} color="green" />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Active');
    expect(output).toContain('5');
  });

  test('renders large numbers', () => {
    const { lastFrame } = render(
      <MetricCard label="Tokens" value={1234567} />
    );
    const output = lastFrame() ?? '';
    expect(output).toContain('Tokens');
    expect(output).toContain('1234567');
  });
});
