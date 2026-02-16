import React, { memo, useMemo } from 'react';
import { Box, Text, useStdout } from 'ink';

export interface Column<T> {
  key: keyof T | string;
  header: string;
  width?: number;
  /** Minimum width for flex columns (default: 10) */
  minWidth?: number;
  /** If true, this column will flex to fill remaining space */
  flex?: boolean;
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
 * Issue #794 - Responsive layout
 */
export function Table<T>({
  data,
  columns,
  selectedIndex,
  maxVisibleRows,
  scrollOffset = 0,
}: TableProps<T>) {
  const { stdout } = useStdout();
  const terminalWidth = stdout.columns ?? 80;

  // Calculate responsive column widths
  const responsiveColumns = useMemo(() => {
    // Reserve space for borders (2), padding (2 per column), selection indicator (2)
    const reservedWidth = 4 + columns.length * 2;
    const availableWidth = Math.max(40, terminalWidth - reservedWidth);

    // Calculate total fixed width and count flex columns
    let fixedWidth = 0;
    let flexCount = 0;

    for (const col of columns) {
      if (col.flex) {
        flexCount++;
      } else {
        fixedWidth += col.width ?? 15;
      }
    }

    // Calculate flex column width
    const remainingWidth = Math.max(10, availableWidth - fixedWidth);
    const flexWidth = flexCount > 0 ? Math.floor(remainingWidth / flexCount) : 0;

    // Return columns with calculated widths
    return columns.map((col) => ({
      ...col,
      calculatedWidth: col.flex
        ? Math.max(col.minWidth ?? 10, flexWidth)
        : Math.min(col.width ?? 15, availableWidth),
    }));
  }, [columns, terminalWidth]);

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
    <Box flexDirection="column" width={terminalWidth - 2}>
      {/* Header Row */}
      <Box borderStyle="single" borderBottom={false}>
        {responsiveColumns.map((col, i) => (
          <Box key={i} width={col.calculatedWidth} paddingRight={1}>
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
          columns={responsiveColumns}
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
interface ResponsiveColumn<T> extends Column<T> {
  calculatedWidth: number;
}

interface TableRowProps<T> {
  item: T;
  columns: ResponsiveColumn<T>[];
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
        <Box key={colIndex} width={col.calculatedWidth} paddingRight={1}>
          {col.render ? (
            col.render(item, rowIndex)
          ) : (
            <Text wrap="truncate">
              {truncateText(
                String((item as Record<string, unknown>)[col.key as string] ?? ''),
                col.calculatedWidth - 1
              )}
            </Text>
          )}
        </Box>
      ))}
    </Box>
  );
}) as <T>(props: TableRowProps<T>) => React.ReactElement;

/**
 * Truncate text to fit within a given width
 */
function truncateText(text: string, maxWidth: number): string {
  if (text.length <= maxWidth) return text;
  if (maxWidth <= 3) return text.slice(0, maxWidth);
  return text.slice(0, maxWidth - 3) + '...';
}

export default Table;
