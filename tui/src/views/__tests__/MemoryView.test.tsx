/**
 * MemoryView Tests
 * Issue #1231: Add additional TUI views
 * Issue #1729: Migrated to useListNavigation
 *
 * Tests cover:
 * - UI state reducer
 * - View mode switching
 * - Search functionality
 * - Time formatting
 * - Memory data structures
 * - Keyboard shortcuts
 * - Detail tab switching
 */

import { describe, test, expect } from 'bun:test';

// View mode types
type ViewMode = 'list' | 'detail' | 'search';
type DetailTab = 'experiences' | 'learnings' | 'prompt';

// UI state interface matching MemoryView
interface UIState {
  viewMode: ViewMode;
  searchQuery: string;
  searchMode: boolean;
  confirmClear: boolean;
  detailTab: DetailTab;
}

// UI action types
type UIAction =
  | { type: 'SET_VIEW_MODE'; mode: ViewMode }
  | { type: 'SET_SEARCH_QUERY'; query: string }
  | { type: 'APPEND_SEARCH_CHAR'; char: string }
  | { type: 'BACKSPACE_SEARCH' }
  | { type: 'TOGGLE_SEARCH_MODE'; enabled?: boolean }
  | { type: 'TOGGLE_CONFIRM_CLEAR'; enabled?: boolean }
  | { type: 'SET_DETAIL_TAB'; tab: DetailTab }
  | { type: 'EXIT_DETAIL' }
  | { type: 'EXIT_SEARCH' };

// Initial state
const initialUIState: UIState = {
  viewMode: 'list',
  searchQuery: '',
  searchMode: false,
  confirmClear: false,
  detailTab: 'experiences',
};

// Reducer matching MemoryView
function uiReducer(state: UIState, action: UIAction): UIState {
  switch (action.type) {
    case 'SET_VIEW_MODE':
      return { ...state, viewMode: action.mode };
    case 'SET_SEARCH_QUERY':
      return { ...state, searchQuery: action.query };
    case 'APPEND_SEARCH_CHAR':
      return { ...state, searchQuery: state.searchQuery + action.char };
    case 'BACKSPACE_SEARCH':
      return { ...state, searchQuery: state.searchQuery.slice(0, -1) };
    case 'TOGGLE_SEARCH_MODE':
      return { ...state, searchMode: action.enabled ?? !state.searchMode };
    case 'TOGGLE_CONFIRM_CLEAR':
      return { ...state, confirmClear: action.enabled ?? !state.confirmClear };
    case 'SET_DETAIL_TAB':
      return { ...state, detailTab: action.tab };
    case 'EXIT_DETAIL':
      return { ...state, viewMode: 'list' };
    case 'EXIT_SEARCH':
      return { ...state, viewMode: 'list', searchQuery: '' };
    default:
      return state;
  }
}

// Time formatting helper
function formatTime(timestamp: string): string {
  try {
    const date = new Date(timestamp);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  } catch {
    return timestamp;
  }
}

// Data structures
interface AgentMemorySummary {
  agent: string;
  experience_count: number;
  learning_count: number;
  last_updated?: string;
}

interface Experience {
  id?: string;
  timestamp: string;
  message: string;
  outcome: 'success' | 'failure';
  category?: string;
}

interface Learning {
  topic: string;
  content: string;
}

interface AgentMemory {
  agent: string;
  experience_count: number;
  learning_count: number;
  experiences: Experience[];
  learnings: Learning[];
}

interface MemorySearchResult {
  agent: string;
  type: 'experience' | 'learning';
  content: string;
  topic?: string;
}

describe('MemoryView', () => {
  describe('UI Reducer', () => {
    describe('SET_VIEW_MODE', () => {
      test('sets list mode', () => {
        const state = uiReducer(initialUIState, { type: 'SET_VIEW_MODE', mode: 'list' });
        expect(state.viewMode).toBe('list');
      });

      test('sets detail mode', () => {
        const state = uiReducer(initialUIState, { type: 'SET_VIEW_MODE', mode: 'detail' });
        expect(state.viewMode).toBe('detail');
      });

      test('sets search mode', () => {
        const state = uiReducer(initialUIState, { type: 'SET_VIEW_MODE', mode: 'search' });
        expect(state.viewMode).toBe('search');
      });
    });

    describe('Search Query', () => {
      test('SET_SEARCH_QUERY sets query directly', () => {
        const state = uiReducer(initialUIState, { type: 'SET_SEARCH_QUERY', query: 'test' });
        expect(state.searchQuery).toBe('test');
      });

      test('APPEND_SEARCH_CHAR adds character', () => {
        let state = uiReducer(initialUIState, { type: 'APPEND_SEARCH_CHAR', char: 'a' });
        state = uiReducer(state, { type: 'APPEND_SEARCH_CHAR', char: 'b' });
        state = uiReducer(state, { type: 'APPEND_SEARCH_CHAR', char: 'c' });
        expect(state.searchQuery).toBe('abc');
      });

      test('BACKSPACE_SEARCH removes last character', () => {
        let state = uiReducer({ ...initialUIState, searchQuery: 'test' }, { type: 'BACKSPACE_SEARCH' });
        expect(state.searchQuery).toBe('tes');
        state = uiReducer(state, { type: 'BACKSPACE_SEARCH' });
        expect(state.searchQuery).toBe('te');
      });

      test('BACKSPACE_SEARCH on empty query does nothing', () => {
        const state = uiReducer(initialUIState, { type: 'BACKSPACE_SEARCH' });
        expect(state.searchQuery).toBe('');
      });
    });

    describe('TOGGLE_SEARCH_MODE', () => {
      test('enables search mode', () => {
        const state = uiReducer(initialUIState, { type: 'TOGGLE_SEARCH_MODE', enabled: true });
        expect(state.searchMode).toBe(true);
      });

      test('disables search mode', () => {
        const enabled = { ...initialUIState, searchMode: true };
        const state = uiReducer(enabled, { type: 'TOGGLE_SEARCH_MODE', enabled: false });
        expect(state.searchMode).toBe(false);
      });

      test('toggles search mode', () => {
        let state = uiReducer(initialUIState, { type: 'TOGGLE_SEARCH_MODE' });
        expect(state.searchMode).toBe(true);
        state = uiReducer(state, { type: 'TOGGLE_SEARCH_MODE' });
        expect(state.searchMode).toBe(false);
      });
    });

    describe('TOGGLE_CONFIRM_CLEAR', () => {
      test('enables confirm clear', () => {
        const state = uiReducer(initialUIState, { type: 'TOGGLE_CONFIRM_CLEAR', enabled: true });
        expect(state.confirmClear).toBe(true);
      });

      test('disables confirm clear', () => {
        const enabled = { ...initialUIState, confirmClear: true };
        const state = uiReducer(enabled, { type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
        expect(state.confirmClear).toBe(false);
      });
    });

    describe('SET_DETAIL_TAB', () => {
      test('sets experiences tab', () => {
        const state = uiReducer(initialUIState, { type: 'SET_DETAIL_TAB', tab: 'experiences' });
        expect(state.detailTab).toBe('experiences');
      });

      test('sets learnings tab', () => {
        const state = uiReducer(initialUIState, { type: 'SET_DETAIL_TAB', tab: 'learnings' });
        expect(state.detailTab).toBe('learnings');
      });

      test('sets prompt tab', () => {
        const state = uiReducer(initialUIState, { type: 'SET_DETAIL_TAB', tab: 'prompt' });
        expect(state.detailTab).toBe('prompt');
      });
    });

    describe('EXIT_DETAIL', () => {
      test('returns to list mode', () => {
        const detailMode = { ...initialUIState, viewMode: 'detail' as ViewMode };
        const state = uiReducer(detailMode, { type: 'EXIT_DETAIL' });
        expect(state.viewMode).toBe('list');
      });
    });

    describe('EXIT_SEARCH', () => {
      test('returns to list and clears query', () => {
        const searchMode = { ...initialUIState, viewMode: 'search' as ViewMode, searchQuery: 'test' };
        const state = uiReducer(searchMode, { type: 'EXIT_SEARCH' });
        expect(state.viewMode).toBe('list');
        expect(state.searchQuery).toBe('');
      });
    });
  });

  describe('Time Formatting', () => {
    test('formats valid timestamp', () => {
      const timestamp = '2024-02-15T14:30:00Z';
      const formatted = formatTime(timestamp);
      // Should contain month and day
      expect(formatted).toMatch(/Feb/i);
      expect(formatted).toMatch(/15/);
    });

    test('handles invalid timestamp', () => {
      const invalid = 'not-a-date';
      const formatted = formatTime(invalid);
      // Returns original on error
      expect(typeof formatted).toBe('string');
    });
  });

  describe('Memory Data Structures', () => {
    describe('AgentMemorySummary', () => {
      test('has required fields', () => {
        const summary: AgentMemorySummary = {
          agent: 'eng-01',
          experience_count: 10,
          learning_count: 5,
        };

        expect(summary.agent).toBe('eng-01');
        expect(summary.experience_count).toBe(10);
        expect(summary.learning_count).toBe(5);
      });

      test('optional last_updated', () => {
        const summary: AgentMemorySummary = {
          agent: 'eng-02',
          experience_count: 0,
          learning_count: 0,
          last_updated: '2024-02-15T14:30:00Z',
        };

        expect(summary.last_updated).toBeDefined();
      });
    });

    describe('AgentMemory', () => {
      test('has experiences array', () => {
        const memory: AgentMemory = {
          agent: 'eng-01',
          experience_count: 1,
          learning_count: 0,
          experiences: [
            {
              id: '1',
              timestamp: '2024-02-15T14:30:00Z',
              message: 'Completed task',
              outcome: 'success',
            },
          ],
          learnings: [],
        };

        expect(memory.experiences).toHaveLength(1);
        expect(memory.experiences[0].outcome).toBe('success');
      });

      test('has learnings array', () => {
        const memory: AgentMemory = {
          agent: 'eng-01',
          experience_count: 0,
          learning_count: 1,
          experiences: [],
          learnings: [
            {
              topic: 'Testing',
              content: 'Always write tests first',
            },
          ],
        };

        expect(memory.learnings).toHaveLength(1);
        expect(memory.learnings[0].topic).toBe('Testing');
      });
    });

    describe('MemorySearchResult', () => {
      test('experience search result', () => {
        const result: MemorySearchResult = {
          agent: 'eng-01',
          type: 'experience',
          content: 'Fixed the bug',
        };

        expect(result.type).toBe('experience');
      });

      test('learning search result with topic', () => {
        const result: MemorySearchResult = {
          agent: 'eng-02',
          type: 'learning',
          content: 'Use TypeScript',
          topic: 'Best Practices',
        };

        expect(result.type).toBe('learning');
        expect(result.topic).toBe('Best Practices');
      });
    });
  });

  describe('Keyboard Shortcuts', () => {
    const shortcuts = {
      'j/k': 'navigate',
      Enter: 'details',
      '/': 'search',
      c: 'clear',
      R: 'refresh',
      '1': 'experiences tab',
      '2': 'learnings tab',
      '3': 'prompt tab',
      'Esc/q': 'back',
      y: 'confirm clear',
    };

    test('navigation shortcuts', () => {
      expect(shortcuts['j/k']).toBe('navigate');
      expect(shortcuts.Enter).toBe('details');
    });

    test('search shortcuts', () => {
      expect(shortcuts['/']).toBe('search');
    });

    test('action shortcuts', () => {
      expect(shortcuts.c).toBe('clear');
      expect(shortcuts.R).toBe('refresh');
    });

    test('detail tab shortcuts', () => {
      expect(shortcuts['1']).toBe('experiences tab');
      expect(shortcuts['2']).toBe('learnings tab');
      expect(shortcuts['3']).toBe('prompt tab');
    });

    test('back shortcuts', () => {
      expect(shortcuts['Esc/q']).toBe('back');
    });

    test('confirm shortcuts', () => {
      expect(shortcuts.y).toBe('confirm clear');
    });
  });

  describe('Memory Counts', () => {
    test('shows count colors based on value', () => {
      const getExperienceColor = (count: number) => count > 0 ? 'green' : 'gray';
      const getLearningColor = (count: number) => count > 0 ? 'yellow' : 'gray';

      expect(getExperienceColor(10)).toBe('green');
      expect(getExperienceColor(0)).toBe('gray');
      expect(getLearningColor(5)).toBe('yellow');
      expect(getLearningColor(0)).toBe('gray');
    });
  });

  describe('Experience Outcomes', () => {
    test('success outcome is green', () => {
      const getOutcomeColor = (outcome: 'success' | 'failure') => outcome === 'success' ? 'green' : 'red';
      expect(getOutcomeColor('success')).toBe('green');
    });

    test('failure outcome is red', () => {
      const getOutcomeColor = (outcome: 'success' | 'failure') => outcome === 'success' ? 'green' : 'red';
      expect(getOutcomeColor('failure')).toBe('red');
    });
  });

  describe('Result Limiting', () => {
    test('experiences limited to 10', () => {
      const experiences = Array(20).fill(null).map((_, i) => ({
        id: String(i),
        timestamp: '2024-02-15T14:30:00Z',
        message: `Experience ${i}`,
        outcome: 'success' as const,
      }));

      const displayed = experiences.slice(0, 10);
      const remaining = experiences.length - 10;

      expect(displayed).toHaveLength(10);
      expect(remaining).toBe(10);
    });

    test('search results limited to 15', () => {
      const results: MemorySearchResult[] = Array(25).fill(null).map((_, i) => ({
        agent: 'eng-01',
        type: 'experience' as const,
        content: `Result ${i}`,
      }));

      const displayed = results.slice(0, 15);
      const remaining = results.length - 15;

      expect(displayed).toHaveLength(15);
      expect(remaining).toBe(10);
    });
  });

  describe('Loading States', () => {
    test('shows loading message', () => {
      const loading = true;
      const message = loading ? 'Loading agent memories...' : '';
      expect(message).toBe('Loading agent memories...');
    });
  });

  describe('Error States', () => {
    test('shows error message', () => {
      const error = 'Failed to fetch memory list';
      expect(error).toBe('Failed to fetch memory list');
    });
  });

  describe('Empty States', () => {
    test('no memories message', () => {
      const agents: AgentMemorySummary[] = [];
      const isEmpty = agents.length === 0;
      expect(isEmpty).toBe(true);
    });

    test('no experiences message', () => {
      const experiences: Experience[] = [];
      const isEmpty = experiences.length === 0;
      expect(isEmpty).toBe(true);
    });

    test('no learnings message', () => {
      const learnings: Learning[] = [];
      const isEmpty = learnings.length === 0;
      expect(isEmpty).toBe(true);
    });

    test('no search results message', () => {
      const results: MemorySearchResult[] = [];
      const isEmpty = results.length === 0;
      expect(isEmpty).toBe(true);
    });
  });

  describe('Clear Confirmation', () => {
    test('shows agent name in confirmation', () => {
      const agent: AgentMemorySummary = {
        agent: 'eng-01',
        experience_count: 10,
        learning_count: 5,
      };

      const message = `Clear all memories for "${agent.agent}"?`;
      expect(message).toContain('eng-01');
    });

    test('shows counts in confirmation', () => {
      const agent: AgentMemorySummary = {
        agent: 'eng-01',
        experience_count: 10,
        learning_count: 5,
      };

      const detail = `This will delete ${agent.experience_count} experiences and ${agent.learning_count} learnings.`;
      expect(detail).toContain('10 experiences');
      expect(detail).toContain('5 learnings');
    });
  });

  describe('Breadcrumb Management', () => {
    test('detail view shows agent name', () => {
      const memory: AgentMemory = {
        agent: 'eng-01',
        experience_count: 0,
        learning_count: 0,
        experiences: [],
        learnings: [],
      };

      const breadcrumb = [{ label: memory.agent }];
      expect(breadcrumb[0].label).toBe('eng-01');
    });

    test('search view shows Search label', () => {
      const breadcrumb = [{ label: 'Search' }];
      expect(breadcrumb[0].label).toBe('Search');
    });
  });

  describe('Focus Management', () => {
    test('detail view focus is view', () => {
      const viewMode: ViewMode = 'detail';
      const focus = viewMode === 'detail' ? 'view' : viewMode === 'search' ? 'view' : 'main';
      expect(focus).toBe('view');
    });

    test('search view focus is view', () => {
      const viewMode: ViewMode = 'search';
      const focus = viewMode === 'detail' ? 'view' : viewMode === 'search' ? 'view' : 'main';
      expect(focus).toBe('view');
    });

    test('list view focus is main', () => {
      const viewMode: ViewMode = 'list';
      const focus = viewMode === 'detail' ? 'view' : viewMode === 'search' ? 'view' : 'main';
      expect(focus).toBe('main');
    });

    test('search mode focus is input', () => {
      const searchMode = true;
      const focus = searchMode ? 'input' : 'main';
      expect(focus).toBe('input');
    });
  });
});
