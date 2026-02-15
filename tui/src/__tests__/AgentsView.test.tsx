/* eslint-disable @typescript-eslint/no-unused-vars */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { StatusBadge } from '../components/StatusBadge';
import { Table } from '../components/Table';

// Test StatusBadge component (no useInput dependency)
describe('StatusBadge', () => {
  it('renders idle state', () => {
    const { lastFrame } = render(<StatusBadge state="idle" />);
    expect(lastFrame()).toContain('idle');
    expect(lastFrame()).toContain('○');
  });

  it('renders working state', () => {
    const { lastFrame } = render(<StatusBadge state="working" />);
    expect(lastFrame()).toContain('working');
    expect(lastFrame()).toContain('●');
  });

  it('renders stuck state', () => {
    const { lastFrame } = render(<StatusBadge state="stuck" />);
    expect(lastFrame()).toContain('stuck');
    expect(lastFrame()).toContain('!');
  });

  it('renders done state', () => {
    const { lastFrame } = render(<StatusBadge state="done" />);
    expect(lastFrame()).toContain('done');
    expect(lastFrame()).toContain('✓');
  });
});

// Test Table component
describe('Table', () => {
  const mockData = [
    { name: 'eng-01', role: 'engineer', state: 'working' },
    { name: 'eng-02', role: 'engineer', state: 'idle' },
  ];

  const columns = [
    { key: 'name', header: 'Name', width: 15 },
    { key: 'role', header: 'Role', width: 12 },
    { key: 'state', header: 'State', width: 10 },
  ];

  it('renders column headers', () => {
    const { lastFrame } = render(
      <Table data={mockData} columns={columns} />
    );
    expect(lastFrame()).toContain('Name');
    expect(lastFrame()).toContain('Role');
    expect(lastFrame()).toContain('State');
  });

  it('renders data rows', () => {
    const { lastFrame } = render(
      <Table data={mockData} columns={columns} />
    );
    expect(lastFrame()).toContain('eng-01');
    expect(lastFrame()).toContain('eng-02');
    expect(lastFrame()).toContain('engineer');
  });

  it('renders empty state when no data', () => {
    const { lastFrame } = render(
      <Table data={[]} columns={columns} />
    );
    expect(lastFrame()).toContain('No data');
  });
});
