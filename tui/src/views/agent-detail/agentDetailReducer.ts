/**
 * Reducer for AgentDetailView state management
 * Consolidates 9 useStates into a single useReducer
 */

import type { AgentDetailState, AgentDetailAction } from './types';

export const initialState: AgentDetailState = {
  outputLines: [],
  loading: true,
  error: null,
  inputMode: false,
  messageBuffer: '',
  sendStatus: null,
  activeTab: 'output',
  liveLines: [],
  scrollOffset: 0,
  isFollowing: true,
};

export function agentDetailReducer(
  state: AgentDetailState,
  action: AgentDetailAction
): AgentDetailState {
  switch (action.type) {
    case 'SET_OUTPUT':
      return { ...state, outputLines: action.lines };
    case 'SET_LOADING':
      return { ...state, loading: action.loading };
    case 'SET_ERROR':
      return { ...state, error: action.error };
    case 'SET_TAB':
      return { ...state, activeTab: action.tab };
    case 'TOGGLE_INPUT_MODE':
      return { ...state, inputMode: action.enabled };
    case 'SET_MESSAGE_BUFFER':
      return { ...state, messageBuffer: action.buffer };
    case 'SET_SEND_STATUS':
      return { ...state, sendStatus: action.status };
    case 'SET_LIVE_LINES':
      return {
        ...state,
        liveLines: action.lines,
        ...(action.scrollOffset !== undefined ? { scrollOffset: action.scrollOffset } : {}),
      };
    case 'SET_SCROLL_OFFSET':
      return { ...state, scrollOffset: action.offset };
    case 'SET_IS_FOLLOWING':
      return { ...state, isFollowing: action.following };
    case 'RESET_INPUT':
      return { ...state, inputMode: false, messageBuffer: '' };
    default:
      return state;
  }
}
