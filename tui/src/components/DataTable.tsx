import React, { memo, useMemo } from 'react';
import { Box, Text, useStdout } from 'ink';

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
  /** Field to use as stable row key (default: 'id') */
  rowKey?: keyof T;
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
  rowKey,
  selectedIndex,
  emptyMessage = 'No data',
  showHeader = true,
  maxVisibleRows,
  scrollOffset = 0,
}: DataTableProps<T>) {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns;

  // Apply virtualization if maxVisibleRows is specified
  // Note: useMemo must be called before any early returns
  const visibleData = useMemo(() => {
    if (data.length === 0) return data;
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

  // Calculate available width accounting for border (2 chars) and padding (2 chars)
  // #1618: Ensure table doesn't overflow at narrow terminals
  const minWidth = 40;
  const tableWidth = Math.max(minWidth, Math.min(terminalWidth - 4, terminalWidth * 0.95));

  // #1618: Generate stable row key from rowKey prop or fall back to stringified row
  const getRowKey = (row: T, index: number): string => {
    if (rowKey && row[rowKey] !== undefined && row[rowKey] !== null) {
      return String(row[rowKey]);
    }
    // Try common id fields
    if ('id' in row && row.id !== undefined) return String(row.id);
    if ('name' in row && row.name !== undefined) return String(row.name);
    // Fall back to index as last resort
    return `row-${String(index)}`;
  };

  if (data.length === 0) {
    return <Text dimColor>{emptyMessage}</Text>;
  }

  return (
    <Box flexDirection="column" borderStyle="single" borderColor="gray" paddingX={1} width={tableWidth}>
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

      {/* Data rows - using memoized row component with stable keys (#1618) */}
      {visibleData.map((row, rowIndex) => (
        <DataTableRow
          key={getRowKey(row, rowIndex + scrollOffset)}
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
