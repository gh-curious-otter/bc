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
  useDemons,
  useDemonLogs,
  type UseDemonsOptions,
  type UseDemonsResult,
  type UseDemonLogsOptions,
  type UseDemonLogsResult,
} from './useDemons';

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

export {
  useProcesses,
  useProcessLogs,
  type UseProcessesOptions,
  type UseProcessesResult,
  type UseProcessLogsOptions,
  type UseProcessLogsResult,
} from './useProcesses';

export {
  useMentionAutocomplete,
  type MentionSuggestion,
  type UseMentionAutocompleteOptions,
  type UseMentionAutocompleteResult,
} from './useMentionAutocomplete';

export {
  useTeams,
  type UseTeamsOptions,
  type UseTeamsResult,
} from './useTeams';

export {
  UnreadProvider,
  useUnread,
  type UnreadProviderProps,
} from './UnreadContext';

export {
  useLogs,
  getSeverityColor,
  getSeverityIcon,
  type UseLogsOptions,
  type UseLogsResult,
  type LogSeverity,
} from './useLogs';

export {
  usePerformanceMetrics,
  createPerformanceTracker,
  globalPerformanceTracker,
  type PerformanceMetric,
  type PerformanceMetrics,
} from './usePerformanceMetrics';

export {
  PerformanceProvider,
  usePerformance,
  usePerformanceOptional,
} from './PerformanceContext';

export {
  useAdaptivePolling,
  useAdaptiveAgentPolling,
  type PollingMode,
  type AdaptivePollingState,
  type UseAdaptivePollingOptions,
  type UseAdaptivePollingResult,
} from './useAdaptivePolling';

export {
  useResponsiveLayout,
  useTerminalSize,
  BREAKPOINTS,
  BREAKPOINTS_LEGACY,
  type LayoutMode,
  type LegacyLayoutMode,
  type ColumnLayout,
  type DrawerConfig,
  type DetailPaneConfig,
  type ResponsiveLayoutState,
  type ResponsiveValues,
  type UseResponsiveLayoutOptions,
  type UseResponsiveLayoutResult,
} from './useResponsiveLayout';

export {
  useAnimation,
  usePulse,
  useBlink,
  useTypewriter,
  useFade,
  easings,
  type EasingFunction,
  type AnimationState,
  type UseAnimationOptions,
  type UseAnimationResult,
  type UsePulseOptions,
  type UsePulseResult,
  type UseBlinkOptions,
  type UseBlinkResult,
  type UseTypewriterOptions,
  type UseTypewriterResult,
  type FadeDirection,
  type UseFadeOptions,
  type UseFadeResult,
} from './useAnimation';

export {
  useKeybindingHints,
  getStatusBarHints,
  formatHintsForStatusBar,
  getViewForKey,
  matchesKey,
  DEFAULT_VIEW_SHORTCUTS,
  DEFAULT_VIEW_NUMBERS,
  type KeyHint,
  type Keybinding,
  type GlobalBindings,
  type ViewBindings,
  type ContextBindings,
  type KeybindingConfig,
} from './useKeybindings';

export {
  useFileTree,
  type FileTreeEntry,
  type UseFileTreeOptions,
  type UseFileTreeResult,
} from './useFileTree';

export {
  useGitStatus,
  type GitFileStatus,
  type GitStatusEntry,
  type GitStatusSummary,
  type UseGitStatusOptions,
  type UseGitStatusResult,
} from './useGitStatus';

export {
  HintsProvider,
  useHintsContext,
  useViewHints,
  type HintsProviderProps,
} from './useHintsContext';
