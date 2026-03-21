/**
 * HeaderBar - Standardized header component for views
 * Issue #1419: TUI Production Polish - Consistent headers across views
 */

import React from 'react';
import { Box, Text } from 'ink';
import { LoadingIndicator } from './LoadingIndicator';

export interface HeaderBarProps {
  /** Main title text */
  title: string;
  /** Optional subtitle/description */
  subtitle?: string;
  /** Show loading indicator */
  loading?: boolean;
  /** Optional count badge (e.g., number of items) */
  count?: number;
  /** Title color (default: cyan) */
  color?: string;
  /** Keyboard hints to show below header */
  hints?: string;
}

/**
 * HeaderBar provides a consistent header pattern for TUI views:
 * - Bold colored title
 * - Optional count badge
 * - Optional loading indicator
 * - Optional subtitle
 * - Optional keyboard hints
 *
 * Usage:
 * ```tsx
 * <HeaderBar
 *   title="Agents"
 *   count={agents.length}
 *   loading={isLoading}
 *   hints="↑/↓ navigate, Enter select"
 * />
 * ```
 */
export function HeaderBar({
  title,
  subtitle,
  loading = false,
  count,
  color = 'cyan',
  hints,
}: HeaderBarProps): React.ReactElement {
  return (
    <Box flexDirection="column" marginBottom={1}>
      {/* Title row with count badge and loading indicator */}
      <Box>
        <Text bold color={color}>
          {title}
        </Text>
        {count !== undefined && (
          <Text dimColor> ({count})</Text>
        )}
        {loading && (
          <Box marginLeft={1}>
            <LoadingIndicator />
          </Box>
        )}
      </Box>

      {/* Subtitle if provided */}
      {subtitle && (
        <Text dimColor>{subtitle}</Text>
      )}

      {/* Keyboard hints if provided */}
      {hints && (
        <Text dimColor>{hints}</Text>
      )}
    </Box>
  );
}

export default HeaderBar;
