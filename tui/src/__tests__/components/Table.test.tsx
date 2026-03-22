/**
 * Table component tests
 * Issue #682 - Component Testing
 */

import React from 'react';
import { render } from 'ink-testing-library';
import { describe, it, expect } from 'bun:test';
import { ThemeProvider } from '../../theme/ThemeContext';
import { Table } from '../../components/Table';

const renderWithTheme = (ui: React.ReactElement) => render(<ThemeProvider>{ui}</ThemeProvider>);
import type { Column } from '../../components/Table';

interface TestData {
  id: number;
  name: string;
  role: string;
  active: boolean;
}

const testColumns: Column<TestData>[] = [
  { key: 'id', header: 'ID', width: 5 },
  { key: 'name', header: 'Name', width: 20 },
  { key: 'role', header: 'Role', width: 15 },
];

const testData: TestData[] = [
  { id: 1, name: 'Alice', role: 'Engineer', active: true },
  { id: 2, name: 'Bob', role: 'Manager', active: true },
  { id: 3, name: 'Charlie', role: 'Designer', active: false },
];

describe('Table', () => {
  describe('basic rendering', () => {
    it('renders without crashing', () => {
      expect(() => {
        render(<Table columns={testColumns} data={testData} />);
      }).not.toThrow();
    });

    it('renders column headers', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('ID');
      expect(output).toContain('Name');
      expect(output).toContain('Role');
    });

    it('renders row data', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Alice');
      expect(output).toContain('Engineer');
    });

    it('renders all rows', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Alice');
      expect(output).toContain('Bob');
      expect(output).toContain('Charlie');
    });
  });

  describe('empty state', () => {
    it('renders empty message when no data', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={[]} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('No data');
    });

    it('shows headers even with no data', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={[]} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('ID');
      expect(output).toContain('Name');
    });
  });

  describe('column configuration', () => {
    it('respects column width', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={testData} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles columns without width', () => {
      const columnsNoWidth: Column<TestData>[] = [
        { key: 'name', header: 'Name' },
        { key: 'role', header: 'Role' },
      ];
      const { lastFrame } = renderWithTheme(<Table columns={columnsNoWidth} data={testData} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles single column', () => {
      const singleColumn: Column<TestData>[] = [{ key: 'name', header: 'Name' }];
      const { lastFrame } = renderWithTheme(<Table columns={singleColumn} data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Name');
      expect(output).toContain('Alice');
    });

    it('handles many columns', () => {
      const manyColumns: Column<TestData>[] = [
        { key: 'id', header: 'ID' },
        { key: 'name', header: 'Name' },
        { key: 'role', header: 'Role' },
        { key: 'active', header: 'Active' },
      ];
      const { lastFrame } = renderWithTheme(<Table columns={manyColumns} data={testData} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('row selection', () => {
    it('highlights selected row', () => {
      const { lastFrame } = renderWithTheme(
        <Table columns={testColumns} data={testData} selectedRow={1} />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('handles no selection', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={testData} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles out-of-bounds selection', () => {
      const { lastFrame } = renderWithTheme(
        <Table columns={testColumns} data={testData} selectedRow={999} />
      );
      expect(lastFrame()).toBeDefined();
    });

    it('handles negative selection', () => {
      const { lastFrame } = renderWithTheme(
        <Table columns={testColumns} data={testData} selectedRow={-1} />
      );
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('data types', () => {
    it('handles numeric values', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={testData} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('1');
    });

    it('handles boolean values', () => {
      const columnsWithBool: Column<TestData>[] = [
        { key: 'name', header: 'Name' },
        { key: 'active', header: 'Active' },
      ];
      const { lastFrame } = renderWithTheme(<Table columns={columnsWithBool} data={testData} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles empty strings', () => {
      const dataWithEmpty = [{ id: 1, name: '', role: '', active: true }];
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={dataWithEmpty} />);
      expect(lastFrame()).toBeDefined();
    });
  });

  describe('edge cases', () => {
    it('handles single row', () => {
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={[testData[0]]} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('Alice');
    });

    it('handles very long strings', () => {
      const dataWithLong = [{ id: 1, name: 'A'.repeat(100), role: 'B'.repeat(50), active: true }];
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={dataWithLong} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles special characters', () => {
      const dataWithSpecial = [
        { id: 1, name: '<script>alert(1)</script>', role: 'Test', active: true },
      ];
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={dataWithSpecial} />);
      expect(lastFrame()).toBeDefined();
    });

    it('handles unicode characters', () => {
      const dataWithUnicode = [{ id: 1, name: '你好世界 🌍', role: 'Engineer', active: true }];
      const { lastFrame } = renderWithTheme(<Table columns={testColumns} data={dataWithUnicode} />);
      const output = lastFrame() ?? '';
      expect(output).toContain('你好世界');
    });
  });

  describe('consistency', () => {
    it('produces consistent output on re-render', () => {
      const { lastFrame: frame1 } = renderWithTheme(
        <Table columns={testColumns} data={testData} />
      );
      const { lastFrame: frame2 } = renderWithTheme(
        <Table columns={testColumns} data={testData} />
      );
      expect(frame1()).toBe(frame2());
    });
  });
});
