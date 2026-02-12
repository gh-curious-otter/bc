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

export {
  useMessagePolling,
  useAgentPolling,
  useCoordinatedPolling,
  type UsePollingOptions,
  type UseMessagePollingOptions,
  type UseMessagePollingResult,
  type UseAgentPollingOptions,
  type UseAgentPollingResult,
  type AgentChange,
  type UseCoordinatedPollingOptions,
} from './usePolling';

export {
  useListNavigation,
  type UseListNavigationOptions,
  type UseListNavigationResult,
} from './useListNavigation';
