import React from 'react';
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
}

export function Table<T>({
  data,
  columns,
  selectedIndex,
}: TableProps<T>) {
  return (
    <Box flexDirection="column">
      {/* Header Row */}
      <Box borderStyle="single" borderBottom={false}>
        {columns.map((col, i) => (
          <Box key={i} width={col.width || 15} paddingRight={1}>
            <Text bold color="cyan">
              {col.header}
            </Text>
          </Box>
        ))}
      </Box>

      {/* Data Rows */}
      {data.map((item, rowIndex) => (
        <Box
          key={rowIndex}
          borderStyle={selectedIndex === rowIndex ? 'double' : undefined}
        >
          {columns.map((col, colIndex) => (
            <Box key={colIndex} width={col.width || 15} paddingRight={1}>
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

export default Table;
