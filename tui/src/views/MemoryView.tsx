/**
 * MemoryView - View and manage agent memories
 * Issue #1231 - Add additional TUI views
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { HeaderBar } from '../components/HeaderBar';
import { ViewWrapper } from '../components/ViewWrapper';
import { useFocus } from '../navigation/FocusContext';
import { getMemoryList, getMemory, searchMemory, clearMemory } from '../services/bc';
import type { AgentMemorySummary, AgentMemory, MemorySearchResult } from '../types';

interface MemoryViewProps {
  disableInput?: boolean;
}

type ViewMode = 'list' | 'detail' | 'search';

/**
 * MemoryView - Display and manage agent memories
 */
export function MemoryView({
  disableInput = false,
}: MemoryViewProps): React.ReactElement {
  // Data state
  const [agents, setAgents] = useState<AgentMemorySummary[]>([]);
  const [selectedMemory, setSelectedMemory] = useState<AgentMemory | null>(null);
  const [searchResults, setSearchResults] = useState<MemorySearchResult[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // UI state
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchMode, setSearchMode] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);
  const [detailTab, setDetailTab] = useState<'experiences' | 'learnings'>('experiences');
  const { setFocus } = useFocus();

  // Manage focus for nested views
  useEffect(() => {
    if (viewMode === 'detail' || searchMode) {
      setFocus('view');
    } else {
      setFocus('main');
    }
  }, [viewMode, searchMode, setFocus]);

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
        setViewMode('detail');
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
      setViewMode('search');
    } catch {
      setError('Search failed');
    }
  }, []);

  // Handle clear confirmation
  const handleClear = useCallback(async () => {
    const agentToDelete = agents[selectedIndex] as AgentMemorySummary | undefined;
    if (agentToDelete === undefined) return;
    try {
      await clearMemory(agentToDelete.agent);
      setConfirmClear(false);
      await fetchMemoryList();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to clear memory');
      setConfirmClear(false);
    }
  }, [agents, selectedIndex, fetchMemoryList]);

  // Valid index for current list
  const validIndex = Math.min(selectedIndex, Math.max(0, agents.length - 1));
  const currentAgent = agents[validIndex] as AgentMemorySummary | undefined;

  // Keyboard handling
  useInput(
    (input, key) => {
      // Confirm clear mode
      if (confirmClear) {
        if (input === 'y' || input === 'Y') {
          void handleClear();
        } else {
          setConfirmClear(false);
        }
        return;
      }

      // Detail view mode
      if (viewMode === 'detail') {
        if (key.escape || input === 'q') {
          setViewMode('list');
          setSelectedMemory(null);
        } else if (input === '1') {
          setDetailTab('experiences');
        } else if (input === '2') {
          setDetailTab('learnings');
        }
        return;
      }

      // Search results view
      if (viewMode === 'search') {
        if (key.escape || input === 'q') {
          setViewMode('list');
          setSearchResults([]);
          setSearchQuery('');
        }
        return;
      }

      // Search input mode
      if (searchMode) {
        if (key.return) {
          void performSearch(searchQuery);
          setSearchMode(false);
        } else if (key.escape) {
          setSearchQuery('');
          setSearchMode(false);
        } else if (key.backspace || key.delete) {
          setSearchQuery((q) => q.slice(0, -1));
        } else if (input && !key.ctrl && !key.meta && !key.tab) {
          setSearchQuery((q) => q + input);
        }
        return;
      }

      // List navigation mode
      if (input === '/') {
        setSearchMode(true);
      } else if (key.upArrow || input === 'k') {
        if (agents.length > 0) {
          setSelectedIndex(Math.max(0, validIndex - 1));
        }
      } else if (key.downArrow || input === 'j') {
        if (agents.length > 0) {
          setSelectedIndex(Math.min(agents.length - 1, validIndex + 1));
        }
      } else if (input === 'g') {
        setSelectedIndex(0);
      } else if (input === 'G') {
        if (agents.length > 0) {
          setSelectedIndex(agents.length - 1);
        }
      } else if (key.return && currentAgent !== undefined) {
        void fetchMemoryDetail(currentAgent.agent);
      } else if (input === 'c' && currentAgent !== undefined) {
        setConfirmClear(true);
      } else if (input === 'R' || (key.ctrl && input === 'r')) {
        void fetchMemoryList();
      }
    },
    { isActive: !disableInput }
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
                  selected={idx === validIndex}
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
          {selected ? '> ' : '  '}
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
              {memory.experiences.slice(0, 10).map((exp, idx) => (
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
                  <Text wrap="wrap">{truncate(exp.message, 70)}</Text>
                </Box>
              ))}
              {memory.experiences.length > 10 && (
                <Text dimColor>... and {memory.experiences.length - 10} more</Text>
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
                  <Text wrap="wrap">{truncate(learning.content, 100)}</Text>
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
            {results.slice(0, 15).map((result, idx) => (
              <Box key={idx} marginBottom={1} flexDirection="column">
                <Box>
                  <Text color="cyan">{result.agent}</Text>
                  <Text dimColor> ({result.type})</Text>
                  {result.topic && <Text color="yellow"> [{result.topic}]</Text>}
                </Box>
                <Text wrap="wrap">{truncate(result.content, 80)}</Text>
              </Box>
            ))}
            {results.length > 15 && (
              <Text dimColor>... and {results.length - 15} more</Text>
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

/**
 * Truncate string to max length
 */
function truncate(str: string, maxLen: number): string {
  if (str.length <= maxLen) return str;
  return str.slice(0, maxLen - 1) + '...';
}

export default MemoryView;
