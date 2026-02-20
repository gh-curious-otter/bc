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

// Chatroom components (eng-04 #570)
export { Reaction, ReactionBar } from './Reaction';
export type { ReactionProps, ReactionBarProps, ReactionType } from './Reaction';

export { MentionText } from './MentionText';
export type { MentionTextProps } from './MentionText';

export { ChatMessage } from './ChatMessage';
export type { ChatMessageProps } from './ChatMessage';

export { MentionAutocomplete } from './MentionAutocomplete';
export type { MentionAutocompleteProps } from './MentionAutocomplete';

// Activity feed (eng-01 #796)
export { ActivityFeed } from './ActivityFeed';
export type { ActivityFeedProps } from './ActivityFeed';

// Members panel (eng-01 #847)
export { MembersPanel, MemberCountBadge } from './MembersPanel';
export type { MembersPanelProps, MemberInfo, MemberCountBadgeProps } from './MembersPanel';

// Progress bar (eng-03 #864)
export { ProgressBar, InlineProgressBar } from './ProgressBar';
export type { ProgressBarProps } from './ProgressBar';

// Sparkline (eng-03 #864, enhanced #974)
export { Sparkline, TrendSparkline, MiniSparkline } from './Sparkline';
export type { SparklineProps, TrendSparklineProps, MiniSparklineProps } from './Sparkline';

// Inline editor (eng-01 #858)
export { InlineEditor, EditorModal } from './InlineEditor';
export type { InlineEditorProps, EditorModalProps } from './InlineEditor';

// Responsive layout (eng-03 #1023)
export { ResponsiveGrid, ResponsiveSidebarLayout, ResponsiveColumns } from './ResponsiveGrid';
export type {
  ResponsiveGridProps,
  ResponsiveSidebarLayoutProps,
  ResponsiveColumnsProps,
} from './ResponsiveGrid';

// Animation components (eng-01 #1024)
export {
  FadeText,
  PulseText,
  TypewriterText,
  BlinkText,
  StatusTransition,
  NotificationText,
} from './AnimatedText';
export type {
  FadeTextProps,
  PulseTextProps,
  TypewriterTextProps,
  BlinkTextProps,
  StatusTransitionProps,
  NotificationTextProps,
} from './AnimatedText';

// Performance overlay (eng-04 #1025)
export { PerformanceOverlay } from './PerformanceOverlay';
export type { PerformanceOverlayProps } from './PerformanceOverlay';

// Data visualization components (eng-04 #1046)
export { Timeline, AgentTimeline, TimelineLegend } from './Timeline';
export type { TimelineProps, TimelineSegment, AgentTimelineProps } from './Timeline';

export { BarChart, MiniBarChart, DistributionChart } from './BarChart';
export type {
  BarChartProps,
  BarChartItem,
  MiniBarChartProps,
  DistributionChartProps,
} from './BarChart';
