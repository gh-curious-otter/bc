/**
 * TUI Hooks - React hooks for bc CLI integration
 */

export {
  useAgents,
  useAgentsByState,
  useAgentsByRole,
  useAgent,
  type UseAgentsOptions,
  type UseAgentsResult,
} from './useAgents';

export {
  useStatus,
  useWorkspaceHealth,
  useUtilization,
  type UseStatusOptions,
  type UseStatusResult,
  type WorkspaceStatus,
} from './useStatus';

export {
  useChannels,
  useChannelHistory,
  useUnreadCount,
  useChannelsWithUnread,
  type UseChannelsOptions,
  type UseChannelsResult,
  type UseChannelHistoryOptions,
  type UseChannelHistoryResult,
} from './useChannels';

export {
  useCosts,
  type UseCostsOptions,
  type UseCostsResult,
} from './useCosts';

export { useDashboard } from './useDashboard';
