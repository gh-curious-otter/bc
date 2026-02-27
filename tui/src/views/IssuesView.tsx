/**
 * IssuesView - Display and manage GitHub issues
 * Issue #1754 - Add Issues View with full GitHub issue management
 */

import React, { useState, useMemo, useCallback, useEffect } from 'react';
import { Box, Text, useInput } from 'ink';
import { Panel } from '../components/Panel';
import { HeaderBar } from '../components/HeaderBar';
import { ViewWrapper } from '../components/ViewWrapper';
import { useFocus } from '../navigation/FocusContext';
import { useNavigation } from '../navigation/NavigationContext';
import { useIssues, useListNavigation } from '../hooks';
import { truncate } from '../utils';
import { DISPLAY_LIMITS, TRUNCATION } from '../constants';
import type { GHIssue } from '../services/bc';

// Issue type labels with color mapping
const LABEL_COLORS: Record<string, string> = {
  bug: 'red',
  enhancement: 'green',
  feature: 'cyan',
  'P0-critical': 'red',
  'P1-high': 'yellow',
  'P2-medium': 'blue',
  'P3-low': 'gray',
  tui: 'magenta',
  go: 'blue',
  epic: 'cyan',
  task: 'white',
};

/**
 * Get color for a label
 */
function getLabelColor(name: string): string {
  return LABEL_COLORS[name] ?? 'gray';
}

/**
 * Format date relative to now
 */
function formatRelativeDate(dateStr: string): string {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return 'today';
    if (diffDays === 1) return 'yesterday';
    if (diffDays < 7) return `${String(diffDays)}d ago`;
    if (diffDays < 30) return `${String(Math.floor(diffDays / 7))}w ago`;
    return `${String(Math.floor(diffDays / 30))}mo ago`;
  } catch {
    return dateStr;
  }
}

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface IssuesViewProps {}

export function IssuesView(_props: IssuesViewProps = {}): React.ReactElement {
  const { data: issues, loading, error, refresh, counts } = useIssues();
  const { setFocus } = useFocus();
  const { setBreadcrumbs, clearBreadcrumbs } = useNavigation();

  const [showDetail, setShowDetail] = useState(false);
  const [labelFilter, setLabelFilter] = useState<string | null>(null);
  const [stateFilter, setStateFilter] = useState<'open' | 'closed' | 'all'>('open');

  // Memoize issueList to prevent dependency changes on each render
  const issueList = useMemo(() => issues ?? [], [issues]);

  // Filter issues by label if filter is set
  const filteredIssues = useMemo(() => {
    let filtered = issueList;
    if (labelFilter) {
      filtered = filtered.filter(issue =>
        issue.labels.some(l => l.name === labelFilter)
      );
    }
    return filtered;
  }, [issueList, labelFilter]);

  // Get unique labels for filter menu
  const uniqueLabels = useMemo(() => {
    const labels = new Set<string>();
    for (const issue of issueList) {
      for (const label of issue.labels) {
        labels.add(label.name);
      }
    }
    return Array.from(labels).sort();
  }, [issueList]);

  // Handle selecting an issue
  const handleSelect = useCallback((_issue: GHIssue) => {
    setShowDetail(true);
  }, []);

  // Custom key handlers
  const customKeys = useMemo(() => ({
    'r': () => { void refresh(); },
    'f': () => {
      // Cycle through label filters
      const currentIdx = labelFilter ? uniqueLabels.indexOf(labelFilter) : -1;
      if (currentIdx === uniqueLabels.length - 1) {
        setLabelFilter(null);
      } else {
        setLabelFilter(uniqueLabels[currentIdx + 1] ?? null);
      }
    },
    's': () => {
      // Cycle through state filters
      if (stateFilter === 'open') setStateFilter('closed');
      else if (stateFilter === 'closed') setStateFilter('all');
      else setStateFilter('open');
    },
  }), [refresh, labelFilter, uniqueLabels, stateFilter]);

  // List navigation
  const {
    selectedIndex,
    selectedItem: selectedIssue,
  } = useListNavigation({
    items: filteredIssues,
    onSelect: handleSelect,
    customKeys,
    isActive: !showDetail,
  });

  // Manage focus and breadcrumbs
  useEffect(() => {
    if (showDetail && selectedIssue) {
      setFocus('view');
      setBreadcrumbs([{ label: `#${String(selectedIssue.number)}` }]);
    } else {
      setFocus('main');
      clearBreadcrumbs();
    }
  }, [showDetail, selectedIssue, setFocus, setBreadcrumbs, clearBreadcrumbs]);

  // Handle detail view input
  useInput(
    (input, key) => {
      if (key.escape || input === 'q' || key.return) {
        setShowDetail(false);
      }
    },
    { isActive: showDetail }
  );

  // Detail view
  if (showDetail && selectedIssue) {
    return (
      <Box flexDirection="column" padding={1}>
        <Panel title={`Issue #${String(selectedIssue.number)}`} borderColor="cyan">
          <Box flexDirection="column">
            {/* Title */}
            <Box marginBottom={1}>
              <Text bold color="cyan">{selectedIssue.title}</Text>
            </Box>

            {/* State and meta */}
            <Box marginBottom={1}>
              <Text color={selectedIssue.state === 'OPEN' ? 'green' : 'red'}>
                [{selectedIssue.state}]
              </Text>
              <Text dimColor> opened {formatRelativeDate(selectedIssue.createdAt)}</Text>
              {selectedIssue.author && (
                <Text dimColor> by {selectedIssue.author.login}</Text>
              )}
            </Box>

            {/* Labels */}
            {selectedIssue.labels.length > 0 && (
              <Box marginBottom={1}>
                <Text dimColor>Labels: </Text>
                {selectedIssue.labels.map((label, idx) => (
                  <Text key={label.name} color={getLabelColor(label.name)}>
                    {label.name}{idx < selectedIssue.labels.length - 1 ? ', ' : ''}
                  </Text>
                ))}
              </Box>
            )}

            {/* Assignees */}
            {selectedIssue.assignees.length > 0 && (
              <Box marginBottom={1}>
                <Text dimColor>Assignees: </Text>
                <Text color="green">
                  {selectedIssue.assignees.map(a => a.login).join(', ')}
                </Text>
              </Box>
            )}

            {/* Body */}
            {selectedIssue.body && (
              <Box marginTop={1} flexDirection="column">
                <Text dimColor>Description:</Text>
                <Box
                  borderStyle="single"
                  borderColor="gray"
                  padding={1}
                  marginTop={0}
                >
                  <Text wrap="wrap">
                    {selectedIssue.body.slice(0, TRUNCATION.ISSUE_BODY)}
                    {selectedIssue.body.length > TRUNCATION.ISSUE_BODY ? '...' : ''}
                  </Text>
                </Box>
              </Box>
            )}

            {/* Comments */}
            {selectedIssue.comments && selectedIssue.comments.length > 0 && (
              <Box marginTop={1} flexDirection="column">
                <Text dimColor>Comments ({selectedIssue.comments.length}):</Text>
                {selectedIssue.comments.slice(0, DISPLAY_LIMITS.ISSUE_COMMENTS).map((comment, idx) => (
                  <Box key={idx} marginTop={1} flexDirection="column">
                    <Box>
                      <Text color="cyan">{comment.author.login}</Text>
                      <Text dimColor> ({formatRelativeDate(comment.createdAt)})</Text>
                    </Box>
                    <Text wrap="wrap">{truncate(comment.body, TRUNCATION.PROMPT_PREVIEW)}</Text>
                  </Box>
                ))}
                {selectedIssue.comments.length > DISPLAY_LIMITS.ISSUE_COMMENTS && (
                  <Text dimColor>... +{selectedIssue.comments.length - DISPLAY_LIMITS.ISSUE_COMMENTS} more comments</Text>
                )}
              </Box>
            )}
          </Box>
        </Panel>

        <Box marginTop={1}>
          <Text dimColor>[Enter/Esc/q] back to list</Text>
        </Box>
      </Box>
    );
  }

  // Build hints
  const hints = [
    { key: 'j/k', label: 'navigate' },
    { key: 'g/G', label: 'top/bottom' },
    { key: 'Enter', label: 'details' },
    { key: 'f', label: labelFilter ? `filter:${truncate(labelFilter, 8)}` : 'filter' },
    { key: 's', label: stateFilter },
    { key: 'r', label: 'refresh' },
    { key: 'q/ESC', label: 'back' },
  ];

  return (
    <ViewWrapper
      loading={loading && filteredIssues.length === 0}
      loadingMessage="Loading issues..."
      error={error}
      onRetry={() => { void refresh(); }}
      hints={hints}
    >
      <Box flexDirection="column" width="100%">
        {/* Header */}
        <HeaderBar
          title="Issues"
          count={filteredIssues.length}
          loading={loading && filteredIssues.length > 0}
          subtitle={labelFilter ? `[${labelFilter}]` : undefined}
          color="cyan"
        />

        {/* Filter indicators */}
        {(labelFilter !== null || stateFilter !== 'open') && (
          <Box marginBottom={1} paddingX={1}>
            {labelFilter && (
              <Text color={getLabelColor(labelFilter)}>
                [Label: {labelFilter}]
              </Text>
            )}
            {stateFilter !== 'open' && (
              <Text color="yellow"> [State: {stateFilter}]</Text>
            )}
          </Box>
        )}

        {/* Stats bar */}
        <Box marginBottom={1} paddingX={1}>
          <Text dimColor>Open: </Text>
          <Text color="green">{counts.open}</Text>
          <Text dimColor> · Closed: </Text>
          <Text color="red">{counts.closed}</Text>
          <Text dimColor> · Total: </Text>
          <Text>{counts.total}</Text>
        </Box>

        {/* Issue list */}
        <Panel title="Issues">
          <Box flexDirection="column">
            {filteredIssues.length === 0 ? (
              <Box padding={1} flexDirection="column">
                <Text dimColor>No issues found</Text>
                <Text dimColor>Create an issue with: bc issue create --title {'"'}...{'"'}</Text>
              </Box>
            ) : (
              <>
                {/* Header row */}
                <Box paddingX={1}>
                  <Box width={8}>
                    <Text bold dimColor>#</Text>
                  </Box>
                  <Box width={50}>
                    <Text bold dimColor>TITLE</Text>
                  </Box>
                  <Box width={12}>
                    <Text bold dimColor>STATE</Text>
                  </Box>
                  <Box width={15}>
                    <Text bold dimColor>LABELS</Text>
                  </Box>
                  <Box flexGrow={1}>
                    <Text bold dimColor>UPDATED</Text>
                  </Box>
                </Box>

                {/* Issue rows */}
                {filteredIssues.map((issue, idx) => (
                  <IssueRow
                    key={issue.number}
                    issue={issue}
                    selected={idx === selectedIndex}
                  />
                ))}
              </>
            )}
          </Box>
        </Panel>
      </Box>
    </ViewWrapper>
  );
}

interface IssueRowProps {
  issue: GHIssue;
  selected: boolean;
}

function IssueRow({ issue, selected }: IssueRowProps): React.ReactElement {
  const primaryLabel = issue.labels[0]?.name ?? '';

  return (
    <Box paddingX={1}>
      <Box width={8}>
        <Text color={selected ? 'cyan' : undefined} bold={selected}>
          {selected ? '▸ ' : '  '}#{issue.number}
        </Text>
      </Box>
      <Box width={50}>
        <Text
          color={selected ? 'cyan' : undefined}
          bold={selected}
          wrap="truncate"
        >
          {truncate(issue.title, 48)}
        </Text>
      </Box>
      <Box width={12}>
        <Text color={issue.state === 'OPEN' ? 'green' : 'red'}>
          {issue.state}
        </Text>
      </Box>
      <Box width={15}>
        <Text color={getLabelColor(primaryLabel)}>
          {truncate(primaryLabel, 13)}
        </Text>
      </Box>
      <Box flexGrow={1}>
        <Text dimColor>
          {formatRelativeDate(issue.updatedAt ?? issue.createdAt)}
        </Text>
      </Box>
    </Box>
  );
}

export default IssuesView;
