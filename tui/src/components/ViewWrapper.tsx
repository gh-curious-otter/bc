/**
 * ViewWrapper - Consistent wrapper for all TUI views
 * Issue #1419: TUI Production Polish - Standardize view layout
 *
 * Provides:
 * - Consistent padding and layout structure
 * - Optional Panel border with title
 * - Responsive layout context
 * - Standard loading and error states
 * - Footer with keybinding hints
 */

import React, { memo } from 'react';
import { Box, Text } from 'ink';
import { Panel } from './Panel';
import { Footer, type HintItem } from './Footer';
import { LoadingIndicator } from './LoadingIndicator';
import { ErrorDisplay } from './ErrorDisplay';
import { useResponsiveLayout, type ResponsiveLayoutState } from '../hooks/useResponsiveLayout';

export interface ViewWrapperProps {
  /** View children to render */
  children: React.ReactNode;
  /** Optional panel title (if set, wraps content in Panel) */
  title?: string;
  /** Whether to wrap content in a bordered Panel */
  usePanel?: boolean;
  /** Panel border color */
  borderColor?: string;
  /** Whether this view is focused */
  focused?: boolean;
  /** Loading state - shows LoadingIndicator */
  loading?: boolean;
  /** Loading message */
  loadingMessage?: string;
  /** Error state - shows ErrorDisplay */
  error?: string | null;
  /** Error retry callback */
  onRetry?: () => void;
  /** Footer keybinding hints */
  hints?: HintItem[];
  /** Custom footer content (overrides hints) */
  footer?: React.ReactNode;
  /** Hide footer entirely */
  hideFooter?: boolean;
  /** Padding around content (default: 1) */
  padding?: number;
  /** Additional responsive layout render prop */
  renderWithLayout?: (layout: ResponsiveLayoutState) => React.ReactNode;
}

/**
 * ViewWrapper - Standardized wrapper for TUI views
 *
 * @example Basic usage with title
 * ```tsx
 * <ViewWrapper title="Processes" loading={isLoading} hints={[{ key: 'j/k', label: 'nav' }]}>
 *   <ProcessList />
 * </ViewWrapper>
 * ```
 *
 * @example With Panel border
 * ```tsx
 * <ViewWrapper usePanel title="Agent Details" borderColor="cyan">
 *   <AgentInfo />
 * </ViewWrapper>
 * ```
 *
 * @example With responsive layout access
 * ```tsx
 * <ViewWrapper
 *   title="Dashboard"
 *   renderWithLayout={({ canMultiColumn }) => (
 *     canMultiColumn ? <TwoColumnLayout /> : <SingleColumnLayout />
 *   )}
 * />
 * ```
 */
export const ViewWrapper = memo(function ViewWrapper({
  children,
  title,
  usePanel = false,
  borderColor = 'gray',
  focused = false,
  loading = false,
  loadingMessage = 'Loading...',
  error,
  onRetry,
  hints,
  footer,
  hideFooter = false,
  padding = 1,
  renderWithLayout,
}: ViewWrapperProps): React.ReactElement {
  const layout = useResponsiveLayout();

  // Error state takes precedence
  if (error) {
    return (
      <Box flexDirection="column" padding={padding}>
        <ErrorDisplay error={error} onRetry={onRetry} />
      </Box>
    );
  }

  // Loading state (only when no content yet)
  if (loading && !children && !renderWithLayout) {
    return (
      <Box flexDirection="column" padding={padding}>
        <LoadingIndicator message={loadingMessage} />
      </Box>
    );
  }

  // Determine content to render
  const content = renderWithLayout ? renderWithLayout(layout) : children;

  // Wrap in Panel if requested
  const wrappedContent = usePanel ? (
    <Panel
      title={title}
      borderColor={borderColor}
      focused={focused}
    >
      {content}
    </Panel>
  ) : (
    <>
      {title && (
        <Box marginBottom={1}>
          <Text bold>{title}</Text>
          {loading && <Text dimColor> (refreshing...)</Text>}
        </Box>
      )}
      {content}
    </>
  );

  return (
    <Box flexDirection="column" padding={padding} flexGrow={1}>
      {/* Main content */}
      <Box flexDirection="column" flexGrow={1}>
        {wrappedContent}
      </Box>

      {/* Footer with hints */}
      {!hideFooter && (footer || hints) && (
        footer ?? <Footer hints={hints ?? []} />
      )}
    </Box>
  );
});

/**
 * Hook to access responsive layout within ViewWrapper
 * Re-exported for convenience
 */
export { useResponsiveLayout };

export default ViewWrapper;
