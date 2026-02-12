// Shared components for bc TUI
// Merged from eng-04 (#561) and eng-03 (#562)

// eng-04's Table component
export { Table } from './Table';
export type { Column } from './Table';

// eng-03's components
export { Panel } from './Panel.js';
export type { PanelProps } from './Panel.js';

export { MetricCard } from './MetricCard.js';
export type { MetricCardProps } from './MetricCard.js';

export { StatusBadge } from './StatusBadge';
export type { StatusBadgeProps } from './StatusBadge';

export { DataTable } from './DataTable.js';
export type { DataTableProps, Column as DataTableColumn } from './DataTable.js';

export { Footer, KeyHint } from './Footer.js';
export type { FooterProps, KeyHintProps } from './Footer.js';

export { LoadingIndicator } from './LoadingIndicator.js';
export type { LoadingIndicatorProps } from './LoadingIndicator.js';

export { ErrorDisplay } from './ErrorDisplay.js';
export type { ErrorDisplayProps } from './ErrorDisplay.js';

// eng-04's MessageInput component
export { MessageInput } from './MessageInput';
