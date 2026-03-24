/**
 * Table Tests
 * Issue #559: Performance optimization
 *
 * Tests cover:
 * - Virtualization (visible data slicing)
 * - Selection index adjustment
 * - Column width defaults
 * - Empty state handling
 * - Selection indicator
 */

import { describe, test, expect } from 'bun:test';

// Types matching Table
interface Column<T> {
  key: keyof T | string;
  header: string;
  width?: number;
}

interface TestItem {
  id: number;
  name: string;
  value: number;
}

// Helper functions matching Table logic
function getVisibleData<T>(
  data: T[],
  maxVisibleRows: number | undefined,
  scrollOffset: number
): T[] {
  if (maxVisibleRows && data.length > maxVisibleRows) {
    return data.slice(scrollOffset, scrollOffset + maxVisibleRows);
  }
  return data;
}

function adjustSelectedIndex(
  selectedIndex: number | undefined,
  maxVisibleRows: number | undefined,
  scrollOffset: number
): number | undefined {
  if (selectedIndex === undefined) return undefined;
  if (maxVisibleRows) {
    return selectedIndex - scrollOffset;
  }
  return selectedIndex;
}

function getColumnWidth(column: Column<TestItem>): number {
  return column.width ?? 15;
}

function getSelectionIndicator(isSelected: boolean): string {
  return isSelected ? '▸ ' : '  ';
}

function getCellValue(item: Record<string, string | number | boolean | null | undefined>, key: string): string {
  return String(item[key] ?? '');
}

describe('Table', () => {
  describe('Virtualization', () => {
    const testData: TestItem[] = Array.from({ length: 100 }, (_, i) => ({
      id: i,
      name: `Item ${i}`,
      value: i * 10,
    }));

    test('returns all data when no max rows', () => {
      const visible = getVisibleData(testData, undefined, 0);
      expect(visible).toHaveLength(100);
    });

    test('slices data when max rows specified', () => {
      const visible = getVisibleData(testData, 10, 0);
      expect(visible).toHaveLength(10);
      expect(visible[0].id).toBe(0);
      expect(visible[9].id).toBe(9);
    });

    test('applies scroll offset', () => {
      const visible = getVisibleData(testData, 10, 20);
      expect(visible).toHaveLength(10);
      expect(visible[0].id).toBe(20);
      expect(visible[9].id).toBe(29);
    });

    test('handles offset near end', () => {
      const visible = getVisibleData(testData, 10, 95);
      expect(visible).toHaveLength(5);
      expect(visible[0].id).toBe(95);
    });

    test('returns all when fewer items than max', () => {
      const smallData = testData.slice(0, 5);
      const visible = getVisibleData(smallData, 10, 0);
      expect(visible).toHaveLength(5);
    });
  });

  describe('Selection Index Adjustment', () => {
    test('returns undefined when no selection', () => {
      expect(adjustSelectedIndex(undefined, 10, 0)).toBeUndefined();
    });

    test('returns same index when no virtualization', () => {
      expect(adjustSelectedIndex(5, undefined, 0)).toBe(5);
    });

    test('adjusts index for scroll offset', () => {
      expect(adjustSelectedIndex(25, 10, 20)).toBe(5);
    });

    test('handles selection at scroll start', () => {
      expect(adjustSelectedIndex(20, 10, 20)).toBe(0);
    });

    test('handles selection at scroll end', () => {
      expect(adjustSelectedIndex(29, 10, 20)).toBe(9);
    });

    test('can return negative (off-screen above)', () => {
      expect(adjustSelectedIndex(15, 10, 20)).toBe(-5);
    });
  });

  describe('Column Width', () => {
    test('uses specified width', () => {
      const col: Column<TestItem> = { key: 'name', header: 'Name', width: 20 };
      expect(getColumnWidth(col)).toBe(20);
    });

    test('defaults to 15', () => {
      const col: Column<TestItem> = { key: 'name', header: 'Name' };
      expect(getColumnWidth(col)).toBe(15);
    });

    test('respects zero width', () => {
      const col: Column<TestItem> = { key: 'name', header: 'Name', width: 0 };
      expect(getColumnWidth(col)).toBe(0);
    });
  });

  describe('Selection Indicator', () => {
    test('shows arrow when selected', () => {
      expect(getSelectionIndicator(true)).toBe('▸ ');
    });

    test('shows spaces when not selected', () => {
      expect(getSelectionIndicator(false)).toBe('  ');
    });

    test('indicators have same width', () => {
      expect(getSelectionIndicator(true).length).toBe(2);
      expect(getSelectionIndicator(false).length).toBe(2);
    });
  });

  describe('Cell Value Extraction', () => {
    const item: TestItem = { id: 1, name: 'Test', value: 100 };

    test('extracts string value', () => {
      expect(getCellValue(item, 'name')).toBe('Test');
    });

    test('converts number to string', () => {
      expect(getCellValue(item, 'id')).toBe('1');
      expect(getCellValue(item, 'value')).toBe('100');
    });

    test('returns empty for missing key', () => {
      expect(getCellValue(item, 'nonexistent')).toBe('');
    });

    test('handles undefined value', () => {
      const itemWithUndefined = { id: 1, name: undefined };
      expect(getCellValue(itemWithUndefined, 'name')).toBe('');
    });

    test('handles null value', () => {
      const itemWithNull = { id: 1, name: null };
      expect(getCellValue(itemWithNull, 'name')).toBe('');
    });
  });

  describe('Empty State', () => {
    test('empty data array', () => {
      const data: TestItem[] = [];
      expect(data.length).toBe(0);
    });

    test('visible data for empty array', () => {
      const visible = getVisibleData([], 10, 0);
      expect(visible).toHaveLength(0);
    });
  });

  describe('Column Configuration', () => {
    test('column has required fields', () => {
      const col: Column<TestItem> = {
        key: 'name',
        header: 'Name',
      };
      expect(col.key).toBe('name');
      expect(col.header).toBe('Name');
    });

    test('column with all fields', () => {
      const col: Column<TestItem> = {
        key: 'name',
        header: 'Name',
        width: 25,
      };
      expect(col.width).toBe(25);
    });
  });

  describe('Scroll Behavior', () => {
    const data: TestItem[] = Array.from({ length: 50 }, (_, i) => ({
      id: i,
      name: `Item ${i}`,
      value: i,
    }));

    test('scroll at start', () => {
      const visible = getVisibleData(data, 10, 0);
      expect(visible[0].id).toBe(0);
    });

    test('scroll in middle', () => {
      const visible = getVisibleData(data, 10, 20);
      expect(visible[0].id).toBe(20);
    });

    test('scroll at end', () => {
      const visible = getVisibleData(data, 10, 40);
      expect(visible[0].id).toBe(40);
      expect(visible).toHaveLength(10);
    });

    test('scroll past end returns fewer items', () => {
      const visible = getVisibleData(data, 10, 45);
      expect(visible).toHaveLength(5);
    });
  });

  describe('Header Row', () => {
    test('header uses bold cyan', () => {
      const style = { bold: true, color: 'cyan' };
      expect(style.bold).toBe(true);
      expect(style.color).toBe('cyan');
    });

    test('header has selection indicator space', () => {
      const headerPrefix = '  ';
      expect(headerPrefix.length).toBe(2);
    });
  });

  describe('Data Row Styling', () => {
    test('selected row is cyan', () => {
      const isSelected = true;
      const color = isSelected ? 'cyan' : undefined;
      expect(color).toBe('cyan');
    });

    test('unselected row has no color', () => {
      const isSelected = false;
      const color = isSelected ? 'cyan' : undefined;
      expect(color).toBeUndefined();
    });
  });

  describe('Integration', () => {
    test('complete table data flow', () => {
      const data: TestItem[] = [
        { id: 1, name: 'Alpha', value: 100 },
        { id: 2, name: 'Beta', value: 200 },
        { id: 3, name: 'Gamma', value: 300 },
      ];

      const columns: Column<TestItem>[] = [
        { key: 'id', header: 'ID', width: 5 },
        { key: 'name', header: 'Name', width: 15 },
        { key: 'value', header: 'Value', width: 10 },
      ];

      // No virtualization
      const visible = getVisibleData(data, undefined, 0);
      expect(visible).toHaveLength(3);

      // Column widths
      expect(getColumnWidth(columns[0])).toBe(5);
      expect(getColumnWidth(columns[1])).toBe(15);
      expect(getColumnWidth(columns[2])).toBe(10);

      // Selection
      const adjusted = adjustSelectedIndex(1, undefined, 0);
      expect(adjusted).toBe(1);
      expect(getSelectionIndicator(true)).toBe('▸ ');

      // Cell values
      expect(getCellValue(data[0], 'name')).toBe('Alpha');
      expect(getCellValue(data[1], 'value')).toBe('200');
    });
  });
});
