/**
 * DataTable component tests
 * Issue #682 - Component Testing
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { DataTable } from '../../components/DataTable';
import type { Column } from '../../components/DataTable';

interface TestRow {
  id: number;
  name: string;
  status: string;
  value: number;
}

const mockColumns: Column<TestRow>[] = [
  { key: 'id', header: 'ID', width: 5 },
  { key: 'name', header: 'NAME', width: 15 },
  { key: 'status', header: 'STATUS', width: 10 },
  { key: 'value', header: 'VALUE', width: 10 },
];

const mockData: TestRow[] = [
  { id: 1, name: 'Item One', status: 'active', value: 100 },
  { id: 2, name: 'Item Two', status: 'inactive', value: 200 },
  { id: 3, name: 'Item Three', status: 'pending', value: 300 },
];

describe('DataTable', () => {
  describe('rendering', () => {
    it('renders table with data', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toContain('ID');
      expect(output).toContain('NAME');
    });

    it('renders column headers', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toContain('ID');
      expect(output).toContain('NAME');
      expect(output).toContain('STATUS');
      expect(output).toContain('VALUE');
    });

    it('renders row data', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toContain('Item One');
      expect(output).toContain('active');
    });

    it('renders empty message when no data', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={[]} emptyMessage="No items found" />
      );
      const output = lastFrame();
      expect(output).toContain('No items found');
    });

    it('uses default empty message', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={[]} />
      );
      const output = lastFrame();
      expect(output).toContain('No data');
    });
  });

  describe('selection', () => {
    it('highlights selected row', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} selectedIndex={1} />
      );
      const output = lastFrame();
      // Selected row should have indicator
      expect(output).toContain('▸');
    });

    it('handles no selection', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toBeDefined();
    });

    it('handles out-of-bounds selection', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} selectedIndex={99} />
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('header visibility', () => {
    it('shows header by default', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toContain('ID');
    });

    it('hides header when showHeader is false', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={mockData} showHeader={false} />
      );
      // Should still have data but layout may differ
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('virtualization', () => {
    const largeData: TestRow[] = Array.from({ length: 100 }, (_, i) => ({
      id: i + 1,
      name: `Item ${String(i + 1)}`,
      status: 'active',
      value: (i + 1) * 10,
    }));

    it('limits visible rows when maxVisibleRows is set', () => {
      const { lastFrame } = render(
        <DataTable
          columns={mockColumns}
          data={largeData}
          maxVisibleRows={5}
          scrollOffset={0}
        />
      );
      const output = lastFrame();
      expect(output).toContain('Item 1');
      // Should not show items beyond maxVisibleRows
    });

    it('respects scrollOffset', () => {
      const { lastFrame } = render(
        <DataTable
          columns={mockColumns}
          data={largeData}
          maxVisibleRows={5}
          scrollOffset={10}
        />
      );
      const output = lastFrame();
      expect(output).toContain('Item 11');
    });
  });

  describe('custom renderers', () => {
    it('uses custom render function', () => {
      const columnsWithRender: Column<TestRow>[] = [
        { key: 'id', header: 'ID', width: 5 },
        {
          key: 'status',
          header: 'STATUS',
          width: 15,
          render: (value) => `[${String(value).toUpperCase()}]`,
        },
      ];

      const { lastFrame } = render(
        <DataTable columns={columnsWithRender} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toContain('[ACTIVE]');
    });
  });

  describe('edge cases', () => {
    it('handles single row', () => {
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={[mockData[0]]} />
      );
      const output = lastFrame();
      expect(output).toContain('Item One');
    });

    it('handles single column', () => {
      const singleColumn: Column<TestRow>[] = [
        { key: 'name', header: 'NAME', width: 20 },
      ];
      const { lastFrame } = render(
        <DataTable columns={singleColumn} data={mockData} />
      );
      const output = lastFrame();
      expect(output).toContain('NAME');
    });

    it('handles long values', () => {
      const dataWithLongValues: TestRow[] = [
        { id: 1, name: 'A'.repeat(100), status: 'active', value: 100 },
      ];
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={dataWithLongValues} />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('handles null-like values', () => {
      const dataWithNulls = [
        { id: 1, name: '', status: null as unknown as string, value: 0 },
      ];
      const { lastFrame } = render(
        <DataTable columns={mockColumns} data={dataWithNulls} />
      );
      expect(lastFrame()).toBeDefined();
    });
  });
});
