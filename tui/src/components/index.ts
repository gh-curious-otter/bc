// Shared components for bc TUI

export { Table } from './Table';
export type { Column } from './Table';

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

export { HeaderBar } from './HeaderBar';
export type { HeaderBarProps } from './HeaderBar';

export { MessageInput } from './MessageInput';

export { Reaction, ReactionBar } from './Reaction';
export type { ReactionProps, ReactionBarProps, ReactionType } from './Reaction';

export { MentionText } from './MentionText';
export type { MentionTextProps } from './MentionText';

export { ChatMessage } from './ChatMessage';
export type { ChatMessageProps } from './ChatMessage';

export { MentionAutocomplete } from './MentionAutocomplete';
export type { MentionAutocompleteProps } from './MentionAutocomplete';

export { ActivityFeed } from './ActivityFeed';
export type { ActivityFeedProps } from './ActivityFeed';

export { MembersPanel, MemberCountBadge } from './MembersPanel';
export type { MembersPanelProps, MemberInfo, MemberCountBadgeProps } from './MembersPanel';

export { ProgressBar, InlineProgressBar } from './ProgressBar';
export type { ProgressBarProps } from './ProgressBar';

export { InlineEditor, EditorModal } from './InlineEditor';
export type { InlineEditorProps, EditorModalProps } from './InlineEditor';

export { ViewWrapper } from './ViewWrapper';
export type { ViewWrapperProps } from './ViewWrapper';

export { ErrorBoundary, ViewErrorBoundary } from './ErrorBoundary';
export type { ErrorBoundaryProps, ViewErrorBoundaryProps } from './ErrorBoundary';
