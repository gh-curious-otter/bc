/**
 * MemoryView - View and manage agent memories
 * Issue #1231 - Add additional TUI views
 */

/**
 * #1729: Migrated to useListNavigation for consolidated keyboard patterns
 */
import React, { useState, useEffect, useCallback, useReducer, useMemo } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { HeaderBar } from '../components/HeaderBar';
import { ViewWrapper } from '../components/ViewWrapper';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { useDisableInput, useListNavigation } from '../hooks';
import { getMemoryList, getMemory, searchMemory, clearMemory } from '../services/bc';
import { truncate } from '../utils';
import { DISPLAY_LIMITS, TRUNCATION } from '../constants';
import type { AgentMemorySummary, AgentMemory, MemorySearchResult } from '../types';

// #1594: Using empty interface for future extensibility, props removed
// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface MemoryViewProps {}

type ViewMode = 'list' | 'detail' | 'search';
type DetailTab = 'experiences' | 'learnings';

/**
 * UI state for MemoryView - consolidated with useReducer (#1601)
 * #1729: Navigation moved to useListNavigation, reducer handles view-specific state
 */
interface UIState {
  viewMode: ViewMode;
  searchQuery: string;
  searchMode: boolean;
  confirmClear: boolean;
  detailTab: DetailTab;
}

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

const initialUIState: UIState = {
  viewMode: 'list',
  searchQuery: '',
  searchMode: false,
  confirmClear: false,
  detailTab: 'experiences',
};

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

/**
 * MemoryView - Display and manage agent memories
 */
export function MemoryView(_props: MemoryViewProps = {}): React.ReactElement {
  // #1594: Use context instead of prop drilling
  const { isDisabled: disableInput } = useDisableInput();

  // Data state (separate useState for async operations)
  const [agents, setAgents] = useState<AgentMemorySummary[]>([]);
  const [selectedMemory, setSelectedMemory] = useState<AgentMemory | null>(null);
  const [searchResults, setSearchResults] = useState<MemorySearchResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // UI state - consolidated with useReducer (#1601)
  // #1729: Navigation moved to useListNavigation
  const [ui, dispatch] = useReducer(uiReducer, initialUIState);
  const { viewMode, searchQuery, searchMode, confirmClear, detailTab } = ui;
  const { setFocus } = useFocus();
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();

  // Manage focus and breadcrumbs for nested views (#1604)
  // When in search mode, set focus='input' to allow typing special chars (#1692)
  useEffect(() => {
    if (viewMode === 'detail' && selectedMemory) {
      setFocus('view');
      setBreadcrumbs([{ label: selectedMemory.agent }]);
    } else if (viewMode === 'search') {
      setFocus('view');
      setBreadcrumbs([{ label: 'Search' }]);
    } else if (searchMode) {
      setFocus('input');
      clearBreadcrumbs();
    } else {
      setFocus('main');
      clearBreadcrumbs();
    }
  }, [viewMode, searchMode, selectedMemory, setFocus, setBreadcrumbs, clearBreadcrumbs]);

  // Fetch agent memory list
  const fetchMemoryList = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getMemoryList();
      setAgents(response.agents);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch memory list');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void fetchMemoryList();
  }, [fetchMemoryList]);

  // Fetch detailed memory for selected agent
  const fetchMemoryDetail = useCallback(async (agentName: string) => {
    try {
      const memory = await getMemory(agentName);
      if (memory) {
        setSelectedMemory(memory);
        dispatch({ type: 'SET_VIEW_MODE', mode: 'detail' });
      }
    } catch {
      setError('Failed to fetch memory details');
    }
  }, []);

  // Search memories
  const performSearch = useCallback(async (query: string) => {
    if (query.length === 0) return;
    try {
      const results = await searchMemory(query);
      setSearchResults(results);
      dispatch({ type: 'SET_VIEW_MODE', mode: 'search' });
    } catch {
      setError('Search failed');
    }
  }, []);

  // Custom key handlers for view-specific actions (#1729)
  const customKeys = useMemo(
    () => ({
      '/': () => { dispatch({ type: 'TOGGLE_SEARCH_MODE', enabled: true }); },
      'c': () => { dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: true }); },
      'R': () => { void fetchMemoryList(); },
    }),
    [fetchMemoryList]
  );

  // #1729: useListNavigation for consolidated keyboard patterns
  const { selectedIndex, selectedItem: currentAgent } = useListNavigation({
    items: agents,
    onSelect: (agent) => { void fetchMemoryDetail(agent.agent); },
    disabled: disableInput || viewMode !== 'list' || searchMode || confirmClear,
    customKeys,
  });

  // Handle clear confirmation
  const handleClear = useCallback(async () => {
    const agentToDelete = agents[selectedIndex] as AgentMemorySummary | undefined;
    if (agentToDelete === undefined) return;
    try {
      await clearMemory(agentToDelete.agent);
      dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
      await fetchMemoryList();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear memory');
      dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
    }
  }, [agents, selectedIndex, fetchMemoryList]);

  // Keyboard handling for modal states (confirm, detail, search)
  useInput(
    (input, key) => {
      // Confirm clear mode
      if (confirmClear) {
        if (input === 'y' || input === 'Y') {
          void handleClear();
        } else {
          dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
        }
        return;
      }

      // Detail view mode
      if (viewMode === 'detail') {
        if (key.escape || input === 'q') {
          dispatch({ type: 'EXIT_DETAIL' });
          setSelectedMemory(null);
        } else if (input === '1') {
          dispatch({ type: 'SET_DETAIL_TAB', tab: 'experiences' });
        } else if (input === '2') {
          dispatch({ type: 'SET_DETAIL_TAB', tab: 'learnings' });
        }
        return;
      }

      // Search results view
      if (viewMode === 'search') {
        if (key.escape || input === 'q') {
          dispatch({ type: 'EXIT_SEARCH' });
          setSearchResults([]);
        }
        return;
      }

      // Search input mode
      if (searchMode) {
        if (key.return) {
          void performSearch(searchQuery);
          dispatch({ type: 'TOGGLE_SEARCH_MODE', enabled: false });
        } else if (key.escape) {
          dispatch({ type: 'SET_SEARCH_QUERY', query: '' });
          dispatch({ type: 'TOGGLE_SEARCH_MODE', enabled: false });
        } else if (key.backspace || key.delete) {
          dispatch({ type: 'BACKSPACE_SEARCH' });
        } else if (input && !key.ctrl && !key.meta && !key.tab) {
          dispatch({ type: 'APPEND_SEARCH_CHAR', char: input });
        }
      }
    },
    { isActive: confirmClear || viewMode !== 'list' || searchMode }
  );

  // Loading/error states handled by ViewWrapper for initial load
  if ((loading || error) && agents.length === 0) {
    return (
      <ViewWrapper
        title="Agent Memories"
        loading={loading}
        loadingMessage="Loading agent memories..."
        error={error}
        onRetry={() => { void fetchMemoryList(); }}
        hints={[
          { key: 'j/k', label: 'navigate' },
          { key: 'Enter', label: 'details' },
        ]}
      >
        {null}
      </ViewWrapper>
    );
  }

  // Clear confirmation modal
  if (confirmClear && currentAgent !== undefined) {
    return (
      <Box flexDirection="column" padding={1}>
        <Panel title="Confirm Clear Memory" borderColor="red">
          <Box flexDirection="column">
            <Text color="red">Clear all memories for &quot;{currentAgent.agent}&quot;?</Text>
            <Text dimColor>This will delete {currentAgent.experience_count} experiences and {currentAgent.learning_count} learnings.</Text>
            <Box marginTop={1}>
              <Text>Press </Text>
              <Text color="red" bold>y</Text>
              <Text> to confirm, any other key to cancel</Text>
            </Box>
          </Box>
        </Panel>
      </Box>
    );
  }

  // Detail view
  if (viewMode === 'detail' && selectedMemory) {
    return (
      <MemoryDetailView
        memory={selectedMemory}
        activeTab={detailTab}
      />
    );
  }

  // Search results view
  if (viewMode === 'search') {
    return (
      <SearchResultsView
        query={searchQuery}
        results={searchResults}
      />
    );
  }

  // Main list view
  return (
    <ViewWrapper
      loading={loading}
      error={error}
      onRetry={() => { void fetchMemoryList(); }}
      hints={[
        { key: 'j/k', label: 'navigate' },
        { key: 'Enter', label: 'details' },
        { key: '/', label: 'search' },
        { key: 'c', label: 'clear' },
        { key: 'R', label: 'refresh' },
      ]}
    >
      <Box flexDirection="column" width="100%">
        {/* Header with count (#1446) */}
        <HeaderBar
          title="Agent Memories"
          count={agents.length}
          loading={loading && agents.length > 0}
          subtitle="agents"
          color="magenta"
        />

        {/* Search bar */}
        <Box
          marginBottom={1}
          paddingX={1}
          borderStyle="single"
          borderColor={searchMode ? 'cyan' : 'gray'}
        >
          {searchMode ? (
            <Box>
              <Text color="cyan">{'/ '}</Text>
              <Text>{searchQuery}</Text>
              <Text color="cyan">|</Text>
            </Box>
          ) : (
            <Text dimColor>Press / to search memories, Enter for details</Text>
          )}
        </Box>

        {/* Agent memory table */}
        <Panel title="Agents">
          {agents.length === 0 ? (
            <Text dimColor>No agent memories found</Text>
          ) : (
            <Box flexDirection="column">
              {/* Header row */}
              <Box paddingX={1}>
                <Box width={20}>
                  <Text bold dimColor>AGENT</Text>
                </Box>
                <Box width={15}>
                  <Text bold dimColor>EXPERIENCES</Text>
                </Box>
                <Box width={12}>
                  <Text bold dimColor>LEARNINGS</Text>
                </Box>
                <Box flexGrow={1}>
                  <Text bold dimColor>LAST UPDATED</Text>
                </Box>
              </Box>

              {/* Agent rows */}
              {agents.map((agent, idx) => (
                <AgentMemoryRow
                  key={agent.agent}
                  agent={agent}
                  selected={idx === selectedIndex}
                />
              ))}
            </Box>
          )}
        </Panel>
      </Box>
    </ViewWrapper>
  );
}

interface AgentMemoryRowProps {
  agent: AgentMemorySummary;
  selected: boolean;
}

function AgentMemoryRow({ agent, selected }: AgentMemoryRowProps): React.ReactElement {
  return (
    <Box paddingX={1}>
      <Box width={20}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {selected ? '▸ ' : '  '}
          {truncate(agent.agent, 16)}
        </Text>
      </Box>
      <Box width={15}>
        <Text color={agent.experience_count > 0 ? 'green' : 'gray'}>
          {String(agent.experience_count)}
        </Text>
      </Box>
      <Box width={12}>
        <Text color={agent.learning_count > 0 ? 'yellow' : 'gray'}>
          {String(agent.learning_count)}
        </Text>
      </Box>
      <Box flexGrow={1}>
        <Text dimColor>{agent.last_updated ? formatTime(agent.last_updated) : '-'}</Text>
      </Box>
    </Box>
  );
}

interface MemoryDetailViewProps {
  memory: AgentMemory;
  activeTab: 'experiences' | 'learnings';
}

function MemoryDetailView({ memory, activeTab }: MemoryDetailViewProps): React.ReactElement {
  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="magenta">Memory: {memory.agent}</Text>
      </Box>

      {/* Tabs */}
      <Box marginBottom={1}>
        <Box marginRight={2}>
          <Text
            color={activeTab === 'experiences' ? 'cyan' : 'gray'}
            bold={activeTab === 'experiences'}
            underline={activeTab === 'experiences'}
          >
            1: Experiences ({memory.experience_count})
          </Text>
        </Box>
        <Box>
          <Text
            color={activeTab === 'learnings' ? 'yellow' : 'gray'}
            bold={activeTab === 'learnings'}
            underline={activeTab === 'learnings'}
          >
            2: Learnings ({memory.learning_count})
          </Text>
        </Box>
      </Box>

      {/* Content */}
      <Panel title={activeTab === 'experiences' ? 'Experiences' : 'Learnings'}>
        {activeTab === 'experiences' ? (
          memory.experiences.length === 0 ? (
            <Text dimColor>No experiences recorded</Text>
          ) : (
            <Box flexDirection="column">
              {memory.experiences.slice(0, DISPLAY_LIMITS.EXPERIENCES).map((exp, idx) => (
                <Box key={exp.id || idx} marginBottom={1} flexDirection="column">
                  <Box>
                    <Text color="cyan">[{formatTime(exp.timestamp)}]</Text>
                    <Text> </Text>
                    <Text color={exp.outcome === 'success' ? 'green' : 'red'}>
                      {exp.outcome}
                    </Text>
                    {exp.category && (
                      <Text dimColor> ({exp.category})</Text>
                    )}
                  </Box>
                  <Text wrap="wrap">{truncate(exp.message, TRUNCATION.MESSAGE)}</Text>
                </Box>
              ))}
              {memory.experiences.length > DISPLAY_LIMITS.EXPERIENCES && (
                <Text dimColor>... and {memory.experiences.length - DISPLAY_LIMITS.EXPERIENCES} more</Text>
              )}
            </Box>
          )
        ) : (
          memory.learnings.length === 0 ? (
            <Text dimColor>No learnings recorded</Text>
          ) : (
            <Box flexDirection="column">
              {memory.learnings.map((learning, idx) => (
                <Box key={learning.topic || idx} marginBottom={1} flexDirection="column">
                  <Text bold color="yellow">{learning.topic}</Text>
                  <Text wrap="wrap">{truncate(learning.content, TRUNCATION.PREVIEW)}</Text>
                </Box>
              ))}
            </Box>
          )
        )}
      </Panel>

      {/* Footer */}
      <Box marginTop={1}>
        <Text dimColor>[1/2] switch tabs | [Esc/q] back to list</Text>
      </Box>
    </Box>
  );
}

interface SearchResultsViewProps {
  query: string;
  results: MemorySearchResult[];
}

function SearchResultsView({ query, results }: SearchResultsViewProps): React.ReactElement {
  return (
    <Box flexDirection="column" padding={1}>
      {/* Header */}
      <Box marginBottom={1}>
        <Text bold color="cyan">Search Results</Text>
        <Text dimColor> for &quot;{query}&quot; ({results.length} found)</Text>
      </Box>

      {/* Results */}
      <Panel title="Results">
        {results.length === 0 ? (
          <Text dimColor>No results found</Text>
        ) : (
          <Box flexDirection="column">
            {results.slice(0, DISPLAY_LIMITS.SEARCH_RESULTS).map((result, idx) => (
              <Box key={idx} marginBottom={1} flexDirection="column">
                <Box>
                  <Text color="cyan">{result.agent}</Text>
                  <Text dimColor> ({result.type})</Text>
                  {result.topic && <Text color="yellow"> [{result.topic}]</Text>}
                </Box>
                <Text wrap="wrap">{truncate(result.content, 80)}</Text>
              </Box>
            ))}
            {results.length > DISPLAY_LIMITS.SEARCH_RESULTS && (
              <Text dimColor>... and {results.length - DISPLAY_LIMITS.SEARCH_RESULTS} more</Text>
            )}
          </Box>
        )}
      </Panel>

      {/* Footer */}
      <Box marginTop={1}>
        <Text dimColor>[Esc/q] back to list</Text>
      </Box>
    </Box>
  );
}

/**
 * Format timestamp to readable time
 */
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

export default MemoryView;
