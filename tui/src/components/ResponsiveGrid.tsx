/**
 * ResponsiveGrid - Responsive multi-column grid layout component
 * Issue #1023: Responsive multi-column layouts
 *
 * Provides a flexible grid system that adapts to terminal width:
 * - Single column on narrow terminals (<100 cols)
 * - Two columns on medium terminals (100-149 cols)
 * - Three columns on wide terminals (150+ cols)
 *
 * Uses flexbox for layout with automatic wrapping and spacing.
 */

import React, { memo, useMemo } from 'react';
import { Box } from 'ink';
import { useResponsiveLayout, type LayoutMode } from '../hooks/useResponsiveLayout';

export interface ResponsiveGridProps {
  /** Child components to arrange in grid */
  children: React.ReactNode;
  /** Gap between grid items (default: 1) */
  gap?: number;
  /** Minimum column width for auto-sizing (default: 30) */
  minColumnWidth?: number;
  /** Maximum columns to display (default: 3) */
  maxColumns?: number;
  /** Force a specific number of columns (overrides responsive) */
  columns?: number;
  /** Override terminal width (for testing) */
  terminalWidth?: number;
  /** Vertical alignment of items */
  alignItems?: 'flex-start' | 'center' | 'flex-end' | 'stretch';
  /** Full width of container */
  fullWidth?: boolean;
}

/**
 * Calculate optimal column count based on width and constraints
 * Updated for 5-tier breakpoint system (#1326)
 */
function calculateColumns(
  width: number,
  minColumnWidth: number,
  maxColumns: number,
  mode: LayoutMode
): number {
  // Force single column on narrow terminals (XS, SM, MD)
  if (mode === 'xs' || mode === 'sm' || mode === 'md') {
    return 1;
  }

  // Calculate how many columns fit
  const possibleColumns = Math.floor(width / minColumnWidth);
  return Math.min(maxColumns, Math.max(1, possibleColumns));
}

/**
 * ResponsiveGrid component for multi-column layouts
 *
 * @example
 * ```tsx
 * <ResponsiveGrid gap={2} maxColumns={2}>
 *   <Panel title="Panel 1">Content 1</Panel>
 *   <Panel title="Panel 2">Content 2</Panel>
 *   <Panel title="Panel 3">Content 3</Panel>
 * </ResponsiveGrid>
 * ```
 */
export const ResponsiveGrid = memo<ResponsiveGridProps>(function ResponsiveGrid({
  children,
  gap = 1,
  minColumnWidth = 30,
  maxColumns = 3,
  columns: forcedColumns,
  terminalWidth,
  alignItems = 'flex-start',
  fullWidth = true,
}: ResponsiveGridProps) {
  const { width, mode } = useResponsiveLayout({ terminalWidth });

  const columnCount = useMemo(() => {
    if (forcedColumns !== undefined) {
      return forcedColumns;
    }
    return calculateColumns(width, minColumnWidth, maxColumns, mode);
  }, [forcedColumns, width, minColumnWidth, maxColumns, mode]);

  // Calculate column width based on available space
  const columnWidth = useMemo(() => {
    const totalGap = gap * (columnCount - 1);
    const availableWidth = width - totalGap - 2; // Account for padding
    return Math.floor(availableWidth / columnCount);
  }, [width, gap, columnCount]);

  // Convert children to array for mapping
  const childArray = React.Children.toArray(children);

  // For single column, just stack vertically
  if (columnCount === 1) {
    return (
      <Box
        flexDirection="column"
        width={fullWidth ? '100%' : undefined}
        gap={gap}
      >
        {children}
      </Box>
    );
  }

  // Multi-column: wrap children in width-constrained boxes
  return (
    <Box
      flexDirection="row"
      flexWrap="wrap"
      width={fullWidth ? '100%' : undefined}
      alignItems={alignItems}
    >
      {childArray.map((child, index) => (
        <Box
          key={index}
          width={columnWidth}
          marginRight={index % columnCount < columnCount - 1 ? gap : 0}
          marginBottom={index < childArray.length - columnCount ? gap : 0}
        >
          {child}
        </Box>
      ))}
    </Box>
  );
});

export interface ResponsiveSidebarLayoutProps {
  /** Main content area */
  children: React.ReactNode;
  /** Sidebar content (rendered on right side when space available) */
  sidebar?: React.ReactNode;
  /** Gap between main and sidebar (default: 1) */
  gap?: number;
  /** Sidebar width in columns (default: auto-calculated) */
  sidebarWidth?: number;
  /** Override terminal width (for testing) */
  terminalWidth?: number;
  /** Whether to show sidebar below main content on narrow terminals */
  stackOnNarrow?: boolean;
}

/**
 * ResponsiveSidebarLayout - Main content with optional sidebar
 *
 * On wide terminals: [Main Content] [Sidebar]
 * On narrow terminals: [Main Content] (sidebar hidden or stacked below)
 *
 * @example
 * ```tsx
 * <ResponsiveSidebarLayout
 *   sidebar={<StatsPanel />}
 *   stackOnNarrow
 * >
 *   <MainContent />
 * </ResponsiveSidebarLayout>
 * ```
 */
export const ResponsiveSidebarLayout = memo<ResponsiveSidebarLayoutProps>(
  function ResponsiveSidebarLayout({
    children,
    sidebar,
    gap = 1,
    sidebarWidth: forcedSidebarWidth,
    terminalWidth,
    stackOnNarrow = false,
  }: ResponsiveSidebarLayoutProps) {
    const {
      width,
      canMultiColumn,
      sidebarWidth: autoSidebarWidth,
    } = useResponsiveLayout({ terminalWidth });

    const actualSidebarWidth = forcedSidebarWidth ?? autoSidebarWidth;

    // No sidebar provided or can't display
    if (!sidebar) {
      return (
        <Box flexDirection="column" width="100%">
          {children}
        </Box>
      );
    }

    // Wide terminal: side-by-side layout
    if (canMultiColumn) {
      const mainWidth = width - actualSidebarWidth - gap - 2;

      return (
        <Box flexDirection="row" width="100%">
          <Box flexDirection="column" width={mainWidth} marginRight={gap}>
            {children}
          </Box>
          <Box flexDirection="column" width={actualSidebarWidth}>
            {sidebar}
          </Box>
        </Box>
      );
    }

    // Narrow terminal: stack or hide sidebar
    if (stackOnNarrow) {
      return (
        <Box flexDirection="column" width="100%">
          {children}
          <Box marginTop={gap}>{sidebar}</Box>
        </Box>
      );
    }

    // Hide sidebar on narrow
    return (
      <Box flexDirection="column" width="100%">
        {children}
      </Box>
    );
  }
);

export interface ResponsiveColumnsProps {
  /** Child components to arrange in columns */
  children: React.ReactNode;
  /** Gap between columns (default: 1) */
  gap?: number;
  /** Column width ratios (e.g., [2, 1] for 2:1 ratio) */
  ratio?: number[];
  /** Override terminal width (for testing) */
  terminalWidth?: number;
  /** Whether to stack columns on narrow terminals (default: true) */
  stackOnNarrow?: boolean;
}

/**
 * ResponsiveColumns - Flexible column layout with ratios
 *
 * @example
 * ```tsx
 * <ResponsiveColumns ratio={[2, 1]} gap={2}>
 *   <MainContent />  {/* Takes 2/3 of width *\/}
 *   <Sidebar />      {/* Takes 1/3 of width *\/}
 * </ResponsiveColumns>
 * ```
 */
export const ResponsiveColumns = memo<ResponsiveColumnsProps>(
  function ResponsiveColumns({
    children,
    gap = 1,
    ratio,
    terminalWidth,
    stackOnNarrow = true,
  }: ResponsiveColumnsProps) {
    const { width, canMultiColumn } = useResponsiveLayout({ terminalWidth });

    const childArray = React.Children.toArray(children);
    const columnCount = childArray.length;

    // Calculate column widths based on ratio (must be before early return for hooks rules)
    const columnWidths = useMemo((): number[] => {
      const totalGap = gap * (columnCount - 1);
      const availableWidth = width - totalGap - 2;

      if (ratio && ratio.length === columnCount) {
        const totalRatio = ratio.reduce((a, b) => a + b, 0);
        return ratio.map((r) => Math.floor((availableWidth * r) / totalRatio));
      }

      // Equal widths if no ratio
      const equalWidth = Math.floor(availableWidth / columnCount);
      return Array.from({ length: columnCount }, () => equalWidth);
    }, [width, gap, columnCount, ratio]);

    // Stack on narrow terminals
    if (!canMultiColumn && stackOnNarrow) {
      return (
        <Box flexDirection="column" width="100%">
          {childArray.map((child, index) => (
            <Box key={index} marginBottom={index < columnCount - 1 ? gap : 0}>
              {child}
            </Box>
          ))}
        </Box>
      );
    }

    return (
      <Box flexDirection="row" width="100%">
        {childArray.map((child, index) => (
          <Box
            key={index}
            width={columnWidths[index]}
            marginRight={index < columnCount - 1 ? gap : 0}
          >
            {child}
          </Box>
        ))}
      </Box>
    );
  }
);

export default ResponsiveGrid;
