/**
 * MemoryView - View and manage agent memories
 * Issue #1839: Memory editor view with 3 tabs
 * - Learnings: Agent knowledge base
 * - Experiences: Recorded agent actions with outcomes
 * - Role Prompt: Agent's role prompt text
 *
 * Uses useListNavigation for consolidated keyboard patterns (#1729)
 */

import React, { useState, useEffect, useCallback, useMemo, useReducer } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { LoadingIndicator } from '../components/LoadingIndicator';
import { HeaderBar } from '../components/HeaderBar';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { useDisableInput, useListNavigation } from '../hooks';
import { truncate, formatRelativeTime } from '../utils';
import { DISPLAY_LIMITS, TRUNCATION } from '../constants';
import type { AgentMemorySummary, AgentMemory as AgentMemoryDetail } from '../types';
import { getMemoryList, getMemory, searchMemory, clearMemory } from '../services/bc';

// View mode types
type ViewMode = 'list' | 'detail' | 'search';
type DetailTab = 'experiences' | 'learnings' | 'prompt';

// UI state
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

// Search result shape (local to avoid unused import)
interface SearchResult {
  agent: string;
  type: 'experience' | 'learning';
  content: string;
  topic?: string;
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface MemoryViewProps {}

/**
 * MemoryView - Display and manage agent memories
 */
export function MemoryView(_props: MemoryViewProps = {}): React.ReactElement {
  const { isDisabled: disableInput } = useDisableInput();
  const [ui, dispatch] = useReducer(uiReducer, initialUIState);
  const [agents, setAgents] = useState<AgentMemorySummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedMemory, setSelectedMemory] = useState<AgentMemoryDetail | null>(null);
  const [searchResults, setSearchResults] = useState<SearchResult[]>([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const { setFocus } = useFocus();
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();

  // Focus and breadcrumb management
  useEffect(() => {
    if (ui.viewMode === 'detail' && selectedMemory) {
      setFocus('view');
      setBreadcrumbs([{ label: selectedMemory.agent }]);
    } else if (ui.viewMode === 'search') {
      setFocus('view');
      setBreadcrumbs([{ label: 'Search' }]);
    } else if (ui.searchMode) {
      setFocus('input');
      clearBreadcrumbs();
    } else {
      setFocus('main');
      clearBreadcrumbs();
    }
  }, [ui.viewMode, ui.searchMode, selectedMemory, setFocus, setBreadcrumbs, clearBreadcrumbs]);

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

  // Fetch agent memory details
  const fetchMemoryDetails = useCallback(async (agentName: string) => {
    try {
      const memory = await getMemory(agentName);
      setSelectedMemory(memory);
      dispatch({ type: 'SET_VIEW_MODE', mode: 'detail' });
    } catch {
      setError('Failed to fetch memory details');
    }
  }, []);

  // Execute search
  const executeSearch = useCallback(async (query: string) => {
    if (query.length === 0) return;
    setSearchLoading(true);
    try {
      const results = await searchMemory(query);
      setSearchResults(results);
      dispatch({ type: 'SET_VIEW_MODE', mode: 'search' });
    } catch {
      setError('Search failed');
    } finally {
      setSearchLoading(false);
    }
  }, []);

  // Custom key handlers
  const customKeys = useMemo(
    () => ({
      '/': () => { dispatch({ type: 'TOGGLE_SEARCH_MODE', enabled: true }); },
      'c': () => { dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: true }); },
      'R': () => { void fetchMemoryList(); },
    }),
    [fetchMemoryList]
  );

  // List navigation
  const { selectedIndex, selectedItem: currentAgent } = useListNavigation({
    items: agents,
    onSelect: (agent) => { void fetchMemoryDetails(agent.agent); },
    disabled: disableInput || ui.viewMode !== 'list' || ui.searchMode || ui.confirmClear,
    customKeys,
  });

  // Handle clear memory
  const handleClear = useCallback(async () => {
    if (!currentAgent) return;
    try {
      await clearMemory(currentAgent.agent);
      dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
      await fetchMemoryList();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear memory');
      dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
    }
  }, [currentAgent, fetchMemoryList]);

  // Keyboard handling for modal/detail/search states
  useInput(
    (input, key) => {
      // Confirm clear mode
      if (ui.confirmClear) {
        if (input === 'y' || input === 'Y') {
          void handleClear();
        } else {
          dispatch({ type: 'TOGGLE_CONFIRM_CLEAR', enabled: false });
        }
        return;
      }

      // Detail view mode
      if (ui.viewMode === 'detail') {
        if (key.escape || input === 'q') {
          dispatch({ type: 'EXIT_DETAIL' });
          setSelectedMemory(null);
          return;
        }
        if (input === '1') {
          dispatch({ type: 'SET_DETAIL_TAB', tab: 'experiences' });
          return;
        }
        if (input === '2') {
          dispatch({ type: 'SET_DETAIL_TAB', tab: 'learnings' });
          return;
        }
        if (input === '3') {
          dispatch({ type: 'SET_DETAIL_TAB', tab: 'prompt' });
          return;
        }
        return;
      }

      // Search results view
      if (ui.viewMode === 'search') {
        if (key.escape || input === 'q') {
          dispatch({ type: 'EXIT_SEARCH' });
          setSearchResults([]);
        }
        return;
      }

      // Search input mode
      if (ui.searchMode) {
        if (key.return) {
          dispatch({ type: 'TOGGLE_SEARCH_MODE', enabled: false });
          void executeSearch(ui.searchQuery);
        } else if (key.escape) {
          dispatch({ type: 'TOGGLE_SEARCH_MODE', enabled: false });
          dispatch({ type: 'SET_SEARCH_QUERY', query: '' });
        } else if (key.backspace || key.delete) {
          dispatch({ type: 'BACKSPACE_SEARCH' });
        } else if (input && !key.ctrl && !key.meta && !key.tab) {
          dispatch({ type: 'APPEND_SEARCH_CHAR', char: input });
        }
      }
    },
    { isActive: ui.confirmClear || ui.viewMode !== 'list' || ui.searchMode }
  );

  // Loading state
  if (loading && agents.length === 0) {
    return <LoadingIndicator message="Loading agent memories..." />;
  }

  // Error state
  if (error && agents.length === 0) {
    return (
      <Box flexDirection="column" padding={1}>
        <Text color="red">Error: {error}</Text>
        <Text dimColor>Press R to retry</Text>
      </Box>
    );
  }

  // Clear confirmation modal
  if (ui.confirmClear && currentAgent) {
    return (
      <Box flexDirection="column" padding={1}>
        <Panel title="Confirm Clear" borderColor="red">
          <Box flexDirection="column">
            <Text color="red">Clear all memories for &quot;{currentAgent.agent}&quot;?</Text>
            <Text dimColor>
              This will delete {String(currentAgent.experience_count)} experiences and {String(currentAgent.learning_count)} learnings.
            </Text>
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

  // Search results view
  if (ui.viewMode === 'search') {
    return (
      <Box flexDirection="column" width="100%" overflow="hidden">
        <HeaderBar
          title="Memory Search"
          count={searchResults.length}
          loading={searchLoading}
          color="magenta"
        />
        <Box marginBottom={1} paddingX={1}>
          <Text dimColor>Query: </Text>
          <Text color="magenta">{ui.searchQuery}</Text>
        </Box>
        <Box flexDirection="column" marginBottom={1}>
          {searchResults.length === 0 ? (
            <Box paddingX={1}>
              <Text dimColor>No results found for &quot;{ui.searchQuery}&quot;</Text>
            </Box>
          ) : (
            searchResults.slice(0, DISPLAY_LIMITS.SEARCH_RESULTS).map((result, idx) => (
              <Box key={`${result.agent}-${result.type}-${String(idx)}`} paddingX={1}>
                <Box width={12}>
                  <Text color="cyan">{truncate(result.agent, 10)}</Text>
                </Box>
                <Box width={12}>
                  <Text color={result.type === 'experience' ? 'green' : 'yellow'}>
                    {result.type}
                  </Text>
                </Box>
                {result.topic && (
                  <Box width={15}>
                    <Text dimColor>{truncate(result.topic, 13)}</Text>
                  </Box>
                )}
                <Box flexGrow={1}>
                  <Text>{truncate(result.content, TRUNCATION.MESSAGE)}</Text>
                </Box>
              </Box>
            ))
          )}
          {searchResults.length > DISPLAY_LIMITS.SEARCH_RESULTS && (
            <Box paddingX={1} marginTop={1}>
              <Text dimColor>
                ...and {String(searchResults.length - DISPLAY_LIMITS.SEARCH_RESULTS)} more results
              </Text>
            </Box>
          )}
        </Box>
        <Box>
          <Text dimColor wrap="truncate">Esc/q: back to list</Text>
        </Box>
      </Box>
    );
  }

  // Detail view
  if (ui.viewMode === 'detail' && selectedMemory) {
    return (
      <Box flexDirection="column" width="100%" overflow="hidden">
        <HeaderBar
          title={`Memory: ${selectedMemory.agent}`}
          color="magenta"
        />
        {/* Tab bar */}
        <Box marginBottom={1}>
          <TabButton label="Experiences" shortcut="1" active={ui.detailTab === 'experiences'} count={selectedMemory.experience_count} />
          <Text> </Text>
          <TabButton label="Learnings" shortcut="2" active={ui.detailTab === 'learnings'} count={selectedMemory.learning_count} />
          <Text> </Text>
          <TabButton label="Role Prompt" shortcut="3" active={ui.detailTab === 'prompt'} />
        </Box>

        {/* Tab content */}
        <Box flexDirection="column" flexGrow={1} overflow="hidden">
          {ui.detailTab === 'experiences' && (
            <ExperiencesTab experiences={selectedMemory.experiences} />
          )}
          {ui.detailTab === 'learnings' && (
            <LearningsTab learnings={selectedMemory.learnings} />
          )}
          {ui.detailTab === 'prompt' && (
            <RolePromptTab agent={selectedMemory.agent} />
          )}
        </Box>

        <Box>
          <Text dimColor wrap="truncate">
            1/2/3: switch tabs | Esc/q: back to list
          </Text>
        </Box>
      </Box>
    );
  }

  // Main list view
  return (
    <Box flexDirection="column" width="100%" overflow="hidden">
      <HeaderBar
        title="Memory"
        count={agents.length}
        loading={loading}
        color="magenta"
      />

      {/* Search bar */}
      <Box
        marginBottom={1}
        paddingX={1}
        borderStyle="single"
        borderColor={ui.searchMode ? 'magenta' : 'gray'}
      >
        {ui.searchMode ? (
          <Box>
            <Text color="magenta">{'/ '}</Text>
            <Text>{ui.searchQuery}</Text>
            <Text color="magenta">▌</Text>
          </Box>
        ) : (
          <Text dimColor>Press / to search memories, j/k to navigate, Enter for details</Text>
        )}
      </Box>

      {/* Agent memory table */}
      <Box flexDirection="column" marginBottom={1}>
        {/* Header row */}
        <Box paddingX={1}>
          <Box width={18}>
            <Text bold dimColor>AGENT</Text>
          </Box>
          <Box width={14}>
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
        {agents.length === 0 ? (
          <Box paddingX={1} marginTop={1}>
            <Text dimColor>No agent memories found.</Text>
          </Box>
        ) : (
          agents.map((agent, idx) => (
            <AgentMemoryRow
              key={agent.agent}
              agent={agent}
              selected={idx === selectedIndex}
            />
          ))
        )}
      </Box>

      {/* Error display */}
      {error && (
        <Box marginBottom={1} paddingX={1}>
          <Text color="red">Error: {error}</Text>
        </Box>
      )}

      {/* Footer */}
      <Box>
        <Text dimColor wrap="truncate">
          {ui.searchMode
            ? 'Type query, Enter to search, Esc to cancel'
            : 'j/k: navigate | g/G: top/bottom | /: search | Enter: details | c: clear | R: refresh | Esc: back'}
        </Text>
      </Box>
    </Box>
  );
}

// --- Sub-components ---

interface TabButtonProps {
  label: string;
  shortcut: string;
  active: boolean;
  count?: number;
}

function TabButton({ label, shortcut, active, count }: TabButtonProps): React.ReactElement {
  const countStr = count !== undefined ? ` (${String(count)})` : '';
  return (
    <Box>
      <Text color={active ? 'magenta' : undefined} bold={active} inverse={active}>
        {` ${shortcut}:${label}${countStr} `}
      </Text>
    </Box>
  );
}

interface AgentMemoryRowProps {
  agent: AgentMemorySummary;
  selected: boolean;
}

function AgentMemoryRow({ agent, selected }: AgentMemoryRowProps): React.ReactElement {
  return (
    <Box paddingX={1}>
      <Box width={18}>
        <Text color={selected ? 'magenta' : undefined} bold={selected}>
          {selected ? '▸ ' : '  '}
          {truncate(agent.agent, 14)}
        </Text>
      </Box>
      <Box width={14}>
        <Text color={agent.experience_count > 0 ? 'green' : undefined} dimColor={agent.experience_count === 0}>
          {String(agent.experience_count)}
        </Text>
      </Box>
      <Box width={12}>
        <Text color={agent.learning_count > 0 ? 'yellow' : undefined} dimColor={agent.learning_count === 0}>
          {String(agent.learning_count)}
        </Text>
      </Box>
      <Box flexGrow={1}>
        <Text dimColor>
          {agent.last_updated ? formatRelativeTime(agent.last_updated) : '-'}
        </Text>
      </Box>
    </Box>
  );
}

interface ExperiencesTabProps {
  experiences: AgentMemoryDetail['experiences'];
}

function ExperiencesTab({ experiences }: ExperiencesTabProps): React.ReactElement {
  if (experiences.length === 0) {
    return (
      <Box paddingX={1}>
        <Text dimColor>No experiences recorded.</Text>
      </Box>
    );
  }

  const displayed = experiences.slice(0, DISPLAY_LIMITS.EXPERIENCES);
  const remaining = experiences.length - DISPLAY_LIMITS.EXPERIENCES;

  return (
    <Box flexDirection="column">
      {displayed.map((exp, idx) => (
        <Box key={exp.id || String(idx)} paddingX={1} marginBottom={idx < displayed.length - 1 ? 0 : undefined}>
          <Box width={14}>
            <Text dimColor>{formatTime(exp.timestamp)}</Text>
          </Box>
          <Box width={10}>
            <Text color={exp.outcome === 'success' ? 'green' : 'red'}>
              {exp.outcome}
            </Text>
          </Box>
          {exp.category && (
            <Box width={12}>
              <Text dimColor>[{truncate(exp.category, 9)}]</Text>
            </Box>
          )}
          <Box flexGrow={1}>
            <Text>{truncate(exp.message, TRUNCATION.MESSAGE)}</Text>
          </Box>
        </Box>
      ))}
      {remaining > 0 && (
        <Box paddingX={1} marginTop={1}>
          <Text dimColor>...and {String(remaining)} more experiences</Text>
        </Box>
      )}
    </Box>
  );
}

interface LearningsTabProps {
  learnings: AgentMemoryDetail['learnings'];
}

function LearningsTab({ learnings }: LearningsTabProps): React.ReactElement {
  if (learnings.length === 0) {
    return (
      <Box paddingX={1}>
        <Text dimColor>No learnings recorded.</Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column">
      {learnings.map((learning, idx) => (
        <Box key={`${learning.topic}-${String(idx)}`} paddingX={1} flexDirection="column" marginBottom={1}>
          <Text color="yellow" bold>{learning.topic}</Text>
          <Box marginLeft={2}>
            <Text>{truncate(learning.content, TRUNCATION.PREVIEW)}</Text>
          </Box>
        </Box>
      ))}
    </Box>
  );
}

interface RolePromptTabProps {
  agent: string;
}

function RolePromptTab({ agent }: RolePromptTabProps): React.ReactElement {
  const [prompt, setPrompt] = useState<string | null>(null);
  const [tabLoading, setTabLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function fetchPrompt() {
      setTabLoading(true);
      try {
        const memory = await getMemory(agent);
        if (!cancelled) {
          // AgentMemory from `memory show` doesn't include role_prompt directly.
          // When go-eng adds role_prompt to the response, update here.
          // For now, we show a placeholder.
          setPrompt(memory ? null : null);
        }
      } catch {
        // silently fail - prompt not available
      } finally {
        if (!cancelled) setTabLoading(false);
      }
    }
    void fetchPrompt();
    return () => { cancelled = true; };
  }, [agent]);

  if (tabLoading) {
    return <LoadingIndicator message="Loading role prompt..." />;
  }

  if (prompt === null) {
    return (
      <Box paddingX={1}>
        <Text dimColor>
          Role prompt not available. Check agent status or role definition files in .bc/roles/.
        </Text>
      </Box>
    );
  }

  return (
    <Box flexDirection="column" paddingX={1}>
      <Panel title="Role Prompt" borderColor="magenta">
        <Text wrap="wrap">{prompt}</Text>
      </Panel>
    </Box>
  );
}

export default MemoryView;
