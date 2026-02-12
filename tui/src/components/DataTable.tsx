import React, { memo, useMemo } from 'react';
import { Box, Text } from 'ink';

export interface Column<T> {
  key: keyof T;
  header: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
  render?: (value: T[keyof T], row: T) => React.ReactNode;
}

export interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  selectedIndex?: number;
  onSelect?: (row: T, index: number) => void;
  emptyMessage?: string;
  showHeader?: boolean;
  maxVisibleRows?: number;
  scrollOffset?: number;
}

/**
 * DataTable - Flexible table component for displaying structured data
 * Shared component with performance optimizations:
 * - Memoized row components to prevent unnecessary re-renders
 * - Optional virtualization via maxVisibleRows/scrollOffset
 * Issue #559 - Performance optimization
 */
export function DataTable<T extends Record<string, unknown>>({
  columns,
  data,
  selectedIndex,
  emptyMessage = 'No data',
  showHeader = true,
  maxVisibleRows,
  scrollOffset = 0,
}: DataTableProps<T>) {
  if (data.length === 0) {
    return <Text dimColor>{emptyMessage}</Text>;
  }

  // Apply virtualization if maxVisibleRows is specified
  const visibleData = useMemo(() => {
    if (maxVisibleRows && data.length > maxVisibleRows) {
      return data.slice(scrollOffset, scrollOffset + maxVisibleRows);
    }
    return data;
  }, [data, maxVisibleRows, scrollOffset]);

  // Adjust selected index for virtualized view
  const adjustedSelectedIndex = useMemo(() => {
    if (selectedIndex === undefined) return undefined;
    if (maxVisibleRows) {
      return selectedIndex - scrollOffset;
    }
    return selectedIndex;
  }, [selectedIndex, maxVisibleRows, scrollOffset]);

  return (
    <Box flexDirection="column">
      {/* Header row */}
      {showHeader && (
        <Box>
          {columns.map((col) => (
            <Box key={String(col.key)} width={col.width}>
              <Text bold dimColor>
                {col.header}
              </Text>
            </Box>
          ))}
        </Box>
      )}

      {/* Data rows - using memoized row component */}
      {visibleData.map((row, rowIndex) => (
        <DataTableRow
          key={rowIndex}
          row={row}
          columns={columns}
          isSelected={adjustedSelectedIndex === rowIndex}
        />
      ))}
    </Box>
  );
}

/**
 * Memoized table row component - only re-renders when row data or selection changes
 */
interface DataTableRowProps<T> {
  row: T;
  columns: Column<T>[];
  isSelected: boolean;
}

const DataTableRow = memo(function DataTableRow<T extends Record<string, unknown>>({
  row,
  columns,
  isSelected,
}: DataTableRowProps<T>) {
  return (
    <Box>
      <Text color={isSelected ? 'cyan' : undefined}>{isSelected ? '▸ ' : '  '}</Text>
      {columns.map((col) => {
        const value = row[col.key];
        const content = col.render
          ? col.render(value, row)
          : String(value ?? '-');

        return (
          <Box key={String(col.key)} width={col.width}>
            {typeof content === 'string' ? (
              <Text color={isSelected ? 'cyan' : undefined} bold={isSelected}>
                {truncate(content, col.width)}
              </Text>
            ) : (
              content
            )}
          </Box>
        );
      })}
    </Box>
  );
}) as <T extends Record<string, unknown>>(props: DataTableRowProps<T>) => React.ReactElement;

function truncate(str: string, maxWidth?: number): string {
  if (!maxWidth || str.length <= maxWidth - 1) return str;
  return str.slice(0, maxWidth - 4) + '...';
}

export default DataTable;
