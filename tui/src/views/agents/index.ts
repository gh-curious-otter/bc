/**
 * Agent view components - Split from AgentsView.tsx (#1592)
 *
 * Original 614 lines split into focused components:
 * - AgentCard: Individual agent row
 * - AgentGroupHeader: Role group header
 * - AgentList: List/table renderer
 * - AgentActions: Action bar
 * - AgentPeekPanel: Peek output panel
 * - AgentConfirmDialog: Confirmation dialog
 * - AgentSearchOverlay: Search input UI
 */

export { AgentCard } from './AgentCard';
export type { AgentCardProps } from './AgentCard';

export { AgentGroupHeader } from './AgentGroupHeader';
export type { AgentGroupHeaderProps } from './AgentGroupHeader';

export { AgentList } from './AgentList';
export type { AgentListProps } from './AgentList';

export { AgentActions } from './AgentActions';
export type { AgentActionsProps } from './AgentActions';

export { AgentPeekPanel } from './AgentPeekPanel';
export type { AgentPeekPanelProps } from './AgentPeekPanel';

export { AgentConfirmDialog } from './AgentConfirmDialog';
export type { AgentConfirmDialogProps, AgentAction } from './AgentConfirmDialog';

export { AgentSearchOverlay } from './AgentSearchOverlay';
export type { AgentSearchOverlayProps } from './AgentSearchOverlay';
