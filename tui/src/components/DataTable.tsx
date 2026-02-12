import React from 'react';
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
}

/**
 * DataTable - Flexible table component for displaying structured data
 * Shared component
 */
export function DataTable<T extends Record<string, unknown>>({
  columns,
  data,
  selectedIndex,
  emptyMessage = 'No data',
  showHeader = true,
}: DataTableProps<T>) {
  if (data.length === 0) {
    return <Text dimColor>{emptyMessage}</Text>;
  }

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

      {/* Data rows */}
      {data.map((row, rowIndex) => (
        <Box
          key={rowIndex}
          backgroundColor={selectedIndex === rowIndex ? 'blue' : undefined}
        >
          {columns.map((col) => {
            const value = row[col.key];
            const content = col.render
              ? col.render(value, row)
              : String(value ?? '-');

            return (
              <Box key={String(col.key)} width={col.width}>
                {typeof content === 'string' ? (
                  <Text>{truncate(content, col.width)}</Text>
                ) : (
                  content
                )}
              </Box>
            );
          })}
        </Box>
      ))}
    </Box>
  );
}

function truncate(str: string, maxWidth?: number): string {
  if (!maxWidth || str.length <= maxWidth - 1) return str;
  return str.slice(0, maxWidth - 4) + '...';
}

export default DataTable;
