/**
 * DetailPane - Toggleable right-side detail panel
 *
 * A 30-character fixed-width right panel that shows:
 * - Detailed info for the currently selected item
 * - Toggle visibility with 'i' key
 * - Automatically hidden at 80x24 resolution
 *
 * Issue #1310: Add toggleable detail pane for selected items
 */

import React from 'react';
import { Box, Text } from 'ink';
import type { View } from '../navigation';

/** Fixed width for detail pane */
export const DETAIL_PANE_WIDTH = 30;

/** Minimum terminal width to show detail pane */
export const MIN_WIDTH_FOR_DETAIL = 100;

/** Minimum terminal width/height to hide detail pane automatically */
export const COMPACT_WIDTH = 80;
export const COMPACT_HEIGHT = 24;

export interface DetailPaneProps {
  /** Currently active view */
  view: View;
  /** Selected item data to display */
  selectedItem?: DetailItem | null;
  /** Terminal width for responsive behavior */
  terminalWidth: number;
  /** Terminal height for responsive behavior */
  terminalHeight: number;
}

/** Generic interface for detail item data */
export interface DetailItem {
  /** Display title */
  title: string;
  /** Item type (e.g., 'agent', 'channel', 'process') */
  type: string;
  /** Key-value pairs to display */
  fields: DetailField[];
  /** Optional description text */
  description?: string;
}

export interface DetailField {
  /** Field label */
  label: string;
  /** Field value */
  value: string;
  /** Optional color for the value */
  color?: string;
}

/**
 * Determines if detail pane should be visible based on terminal size
 */
export function shouldShowDetailPane(
  width: number,
  height: number,
  isVisible: boolean
): boolean {
  // Always hidden at compact terminal size (80x24)
  if (width <= COMPACT_WIDTH && height <= COMPACT_HEIGHT) {
    return false;
  }
  // Hidden when terminal too narrow
  if (width < MIN_WIDTH_FOR_DETAIL) {
    return false;
  }
  return isVisible;
}

/**
 * DetailPane component - shows detailed info for selected items
 * Note: view, terminalWidth, terminalHeight reserved for future view-specific detail rendering
 */
export function DetailPane({
  view: _view,
  selectedItem,
  terminalWidth: _terminalWidth,
  terminalHeight: _terminalHeight,
}: DetailPaneProps): React.ReactElement | null {
  // No item selected - show placeholder
  if (!selectedItem) {
    return (
      <Box
        flexDirection="column"
        width={DETAIL_PANE_WIDTH}
        borderStyle="single"
        borderLeft
        borderTop={false}
        borderBottom={false}
        borderRight={false}
        paddingLeft={1}
      >
        <Box marginBottom={1}>
          <Text bold color="cyan">Details</Text>
        </Box>
        <Box>
          <Text dimColor>{'─'.repeat(DETAIL_PANE_WIDTH - 3)}</Text>
        </Box>
        <Box marginTop={1} flexDirection="column">
          <Text dimColor>Select an item</Text>
          <Text dimColor>to view details</Text>
        </Box>
        <Box flexGrow={1} />
        <Box>
          <Text dimColor>[i] toggle pane</Text>
        </Box>
      </Box>
    );
  }

  return (
    <Box
      flexDirection="column"
      width={DETAIL_PANE_WIDTH}
      borderStyle="single"
      borderLeft
      borderTop={false}
      borderBottom={false}
      borderRight={false}
      paddingLeft={1}
    >
      {/* Header with type badge */}
      <Box marginBottom={1}>
        <Text bold color="cyan">Details</Text>
        <Text dimColor> [{selectedItem.type}]</Text>
      </Box>
      <Box>
        <Text dimColor>{'─'.repeat(DETAIL_PANE_WIDTH - 3)}</Text>
      </Box>

      {/* Title */}
      <Box marginTop={1}>
        <Text bold wrap="truncate">
          {truncateText(selectedItem.title, DETAIL_PANE_WIDTH - 4)}
        </Text>
      </Box>

      {/* Description if available */}
      {selectedItem.description && (
        <Box marginTop={1}>
          <Text dimColor wrap="wrap">
            {truncateText(selectedItem.description, (DETAIL_PANE_WIDTH - 4) * 2)}
          </Text>
        </Box>
      )}

      {/* Fields */}
      <Box marginTop={1} flexDirection="column">
        {selectedItem.fields.map((field, idx) => (
          <DetailFieldRow key={idx} field={field} maxWidth={DETAIL_PANE_WIDTH - 4} />
        ))}
      </Box>

      {/* Spacer and footer hint */}
      <Box flexGrow={1} />
      <Box>
        <Text dimColor>[i] toggle pane</Text>
      </Box>
    </Box>
  );
}

interface DetailFieldRowProps {
  field: DetailField;
  maxWidth: number;
}

function DetailFieldRow({ field, maxWidth }: DetailFieldRowProps): React.ReactElement {
  const labelWidth = Math.min(field.label.length, 10);
  const valueWidth = maxWidth - labelWidth - 2; // -2 for ": "

  return (
    <Box>
      <Text dimColor>{field.label.substring(0, 10).padEnd(10)}: </Text>
      <Text color={field.color as 'green' | 'red' | 'yellow' | 'cyan' | undefined}>
        {truncateText(field.value, valueWidth)}
      </Text>
    </Box>
  );
}

/**
 * Truncate text to max length with ellipsis
 */
function truncateText(text: string, maxLen: number): string {
  if (text.length <= maxLen) {
    return text;
  }
  return text.substring(0, maxLen - 1) + '…';
}

export default DetailPane;
