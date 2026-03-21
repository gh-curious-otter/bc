/**
 * Agent detail sub-components barrel export
 */

export { AgentOutputTab } from './AgentOutputTab';
export { AgentLiveTab } from './AgentLiveTab';
export { AgentDetailsTab } from './AgentDetailsTab';
export { AgentMetricsTab } from './AgentMetricsTab';
export { agentDetailReducer, initialState } from './agentDetailReducer';
export {
  TabButton,
  DetailRow,
  normalizeTask,
  formatDate,
  formatTime,
  formatNumber,
  truncateMessage,
  formatUptime,
  LABEL_WIDTH,
} from './types';
export type { AgentTab, AgentDetailState, AgentDetailAction } from './types';
