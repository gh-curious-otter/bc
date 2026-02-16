import React, { memo, useMemo } from 'react';
import { Box, Text } from 'ink';

export interface Column<T> {
  key: keyof T | string;
  header: string;
  width?: number;
  render?: (item: T, index: number) => React.ReactNode;
}

interface TableProps<T> {
  data: T[];
  columns: Column<T>[];
  selectedIndex?: number;
  onSelect?: (item: T, index: number) => void;
  maxVisibleRows?: number;
  scrollOffset?: number;
}

/**
 * Table - Memoized table component with optional virtualization
 * Issue #559 - Performance optimization
 */
export function Table<T>({
  data,
  columns,
  selectedIndex,
  maxVisibleRows,
  scrollOffset = 0,
}: TableProps<T>) {
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
      {/* Header Row */}
      <Box borderStyle="single" borderBottom={false}>
        {columns.map((col, i) => (
          <Box key={i} width={col.width ?? 15} paddingRight={1}>
            <Text bold color="cyan">
              {col.header}
            </Text>
          </Box>
        ))}
      </Box>

      {/* Data Rows - using memoized row component */}
      {visibleData.map((item, rowIndex) => (
        <TableRow
          key={rowIndex}
          item={item}
          columns={columns}
          rowIndex={rowIndex}
          isSelected={adjustedSelectedIndex === rowIndex}
        />
      ))}

      {/* Empty State */}
      {data.length === 0 && (
        <Box padding={1}>
          <Text color="gray">No data</Text>
        </Box>
      )}
    </Box>
  );
}

/**
 * Memoized table row - only re-renders when data or selection changes
 */
interface TableRowProps<T> {
  item: T;
  columns: Column<T>[];
  rowIndex: number;
  isSelected: boolean;
}

const TableRow = memo(function TableRow<T>({
  item,
  columns,
  rowIndex,
  isSelected,
}: TableRowProps<T>) {
  return (
    <Box borderStyle={isSelected ? 'double' : undefined}>
      {columns.map((col, colIndex) => (
        <Box key={colIndex} width={col.width ?? 15} paddingRight={1}>
          {col.render ? (
            col.render(item, rowIndex)
          ) : (
            <Text>
              {String((item as Record<string, unknown>)[col.key as string] ?? '')}
            </Text>
          )}
        </Box>
      ))}
    </Box>
  );
}) as <T>(props: TableRowProps<T>) => React.ReactElement;

export default Table;
